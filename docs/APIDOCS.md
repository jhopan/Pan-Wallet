# PanWallet API — Dokumentasi Ringkas

Dokumen ini untuk pembaca mesin (AI) dan developer yang mau cepat integrasi.
Untuk penjelasan panjang, lihat `API.md` (referensi lengkap) dan
`INTEGRASI.md` (panduan step-by-step bot).

---

## 1. Ringkasan dalam 30 detik

```
PanWallet = satu dompet saldo untuk semua produk Jhopan.
```

| Fakta | Nilai |
|---|---|
| Base URL produksi | `https://panwallet.jhopan.my.id` |
| Base URL LAN (server) | `http://192.168.10.21:8090` |
| Format uang | integer rupiah (tanpa desimal). `25000` = Rp25.000 |
| Mata uang | IDR |
| Minimum top-up | `25000` |
| ID pelanggan | `public_id` — 8 karakter, mis. `65ZMPUNP` |
| Saldo bisa diuangkan? | **TIDAK**. Hanya untuk transaksi di layanan Jhopan |
| Auth API produk | `Authorization: Bearer pw_live_...` |
| Rate limit endpoint publik | 30 request/menit per IP |
| Rate limit login admin | 8 percobaan/menit per IP |

### Aturan emas

1. **Jangan pernah menyimpan saldo di bot.** Bot hanya menyimpan `public_id`.
2. **Jangan pernah potong saldo langsung.** Pakai `hold` → `capture`/`release`.
3. **Nominal selalu integer.** Tidak ada `25000.00`.
4. **Selalu kirim `Idempotency-Key`** saat membuat hold.
5. **Pakai `available`, bukan `balance`.** `available` sudah dikurangi saldo yang ditahan.

---

## 2. Peta endpoint (ringkas)

### API produk — butuh `Authorization: Bearer <API_KEY>`

| Method | Path | Fungsi |
|---|---|---|
| POST | `/v1/holds` | kunci saldo (hold) sebelum transaksi |
| POST | `/v1/holds/{hold_id}/capture` | potong permanen (order sukses) |
| POST | `/v1/holds/{hold_id}/release` | lepas kunci (order gagal) |
| GET | `/v1/wallets/{wallet_user_id}` | detail saldo |

### API publik — tanpa auth

| Method | Path | Fungsi |
|---|---|---|
| POST | `/v1/identity/resolve` | tukar id channel → `public_id` |
| GET | `/v1/public/{public_id}` | cek saldo pelanggan |
| POST | `/v1/public/{public_id}/topup` | buat invoice top-up QRIS |
| GET | `/v1/public/{public_id}/identities` | daftar channel tertaut |
| POST | `/v1/identity/link/start` | minta kode ikat channel |
| POST | `/v1/identity/link/confirm` | konfirmasi kode ikat |

### Webhook

| Method | Path | Pengirim | Auth |
|---|---|---|---|
| POST | `/webhook/paypan` | PayPan (gateway QRIS) | HMAC `X-Paypan-Signature` |

### Halaman web

| Method | Path | Fungsi |
|---|---|---|
| GET | `/healthz` | health check (JSON) |
| GET | `/pay/{invoice_id}` | halaman pembayaran QRIS |
| GET | `/api/topup/{invoice_id}` | status top-up (JSON) |
| GET | `/` | **selalu 404** (sengaja, jangan bocorkan keberadaan) |
| GET | `/masuk` | portal pelanggan (masuk via email) |
| POST | `/masuk/cek` | cek kode OTP portal |

### Panel admin — butuh sesi login

| Method | Path | Fungsi |
|---|---|---|
| GET | `/admin` | dashboard |
| GET | `/admin/wallets` | daftar pelanggan |
| GET | `/admin/ledger` | mutasi saldo |
| GET | `/admin/keys` | API key produk |
| GET | `/admin/laporan` | laporan pendapatan |
| GET | `/admin/audit` | jejak tindakan admin |
| GET | `/admin/export` | ekspor CSV |
| GET | `/admin/backup` | daftar backup |
| GET | `/admin/config` | konfigurasi PayPan & notifikasi |
| GET | `/admin/topup` | daftar invoice top-up |

---

## 3. Format respons

### Sukses

```json
{
  "public_id": "65ZMPUNP",
  "balance": 150001,
  "held": 0,
  "available": 150001,
  "status": "active",
  "currency": "IDR",
  "is_new": false
}
```

| Field | Arti |
|---|---|
| `balance` | total saldo |
| `held` | saldo yang sedang ditahan (hold aktif) |
| `available` | saldo yang bisa dipakai = `balance` − `held` |
| `status` | `active` atau `blocked` |
| `is_new` | true kalau pelanggan baru dibuat |

### Error

```json
{"error": "saldo tidak cukup"}
```

| Kode | Arti | Tindakan bot |
|---|---|---|
| 200 / 201 | sukses | lanjut |
| 401 | API key salah/tidak ada | cek token, jangan retry |
| 403 | wallet diblokir | suruh pelanggan hubungi admin |
| 404 | wallet/hold tidak ditemukan | cek ID |
| 422 | saldo tidak cukup | arahkan pelanggan top-up |
| 429 | rate limit | tunggu, jangan retry cepat |

---

## 4. Alur untuk bot toko

```
Pelanggan datang dari Telegram/WhatsApp
        │
        ▼
[1] POST /v1/identity/resolve
    {provider, external_id} → {public_id, is_new}
    Simpan public_id. Panggil sekali saja per pelanggan.
        │
        ▼
[2] GET /v1/public/{public_id}
    Cek available ≥ harga?
        │
   tidak cukup ──► "Isi saldo di @PanWalletBot"  (BERHENTI, jangan order)
        │ cukup
        ▼
[3] POST /v1/holds
    Kunci saldo → dapat hold_id
        │
        ▼
[4] Proses order (kirim pulsa, dll)
        │
   berhasil ──► [5a] POST /v1/holds/{id}/capture   (potong)
   gagal   ──► [5b] POST /v1/holds/{id}/release    (kembalikan saldo)
```

### [1] Resolve identitas

```http
POST /v1/identity/resolve
Content-Type: application/json

{"provider": "telegram", "external_id": "123456789"}
```

`provider`: `telegram` | `whatsapp` | `email`

```json
{"public_id":"65ZMPUNP","is_new":true,"balance":0,"available":0,"held":0,"status":"active","currency":"IDR"}
```

`is_new: true` = pelanggan baru dibuat. Kalau `false`, sudah pernah ada —
**jangan** simpan ulang.

### [2] Cek saldo

```http
GET /v1/public/65ZMPUNP
```

```json
{"public_id":"65ZMPUNP","balance":150001,"held":10000,"available":140001,"status":"active"}
```

### [3] Kunci saldo (hold)

```http
POST /v1/holds
Authorization: Bearer pw_live_...
Idempotency-Key: hold:INV-2026-001
Content-Type: application/json

{
  "wallet_user_id": "e469f462-5040-450f-ac2a-48987fbbbe7f",
  "reference_id": "INV-2026-001",
  "amount": 25000,
  "expires_at": "2030-01-01T00:00:00Z"
}
```

| Field | Wajib | Arti |
|---|---|---|
| `wallet_user_id` | ya | UUID internal (dari `GET /v1/wallets/...` atau respons admin) |
| `reference_id` | ya | nomor invoice/order — buat idempotency & audit |
| `amount` | ya | integer rupiah, ≥ 1 |
| `expires_at` | ya | RFC3339, masa berlaku hold |

Sukses `201`:

```json
{"id":"<hold_id>","wallet_user_id":"...","product":"agenpulsa","reference_id":"INV-2026-001","amount":25000,"status":"active","idempotency_key":"hold:INV-2026-001"}
```

Gagal `422` → `{"error":"saldo tidak cukup"}`

### [4a] Capture (order sukses)

```http
POST /v1/holds/{hold_id}/capture
Authorization: Bearer pw_live_...
Idempotency-Key: capture:INV-2026-001
```

```json
{"id":"<hold_id>","amount":25000,"status":"captured","product":"agenpulsa"}
```

### [4b] Release (order gagal)

```http
POST /v1/holds/{hold_id}/release
Authorization: Bearer pw_live_...
Idempotency-Key: release:INV-2026-001
```

```json
{"id":"<hold_id>","amount":25000,"status":"released"}
```

### Idempotency

| Situasi | Hasil |
|---|---|
| Key sama + body sama | aman, hasil sama, tidak dobel potong |
| Key sama + body beda | `409` — bug, perbaiki |
| Key beda | transaksi baru |

Pola kunci yang disarankan: `hold:<nomor-invoice>`, `capture:<nomor-invoice>`,
`release:<nomor-invoice>`.

---

## 5. Top-up QRIS

### Buat invoice

```http
POST /v1/public/65ZMPUNP/topup
Content-Type: application/json

{"amount": 25000}
```

```json
{
  "invoice_id": "04ea9711884ed79034e77c2f04662862",
  "pay_url": "/pay/27d22bd8fececb99",
  "total": 25001,
  "status": "pending",
  "expires_at": 1793156785
}
```

| Field | Arti |
|---|---|
| `invoice_id` | id invoice internal PanWallet |
| `pay_url` | path halaman bayar (gabung dengan base URL) |
| `total` | **total unik** = nominal + kode unik PayPan |
| `expires_at` | unix deadline |

> **Penting:** `total` (25001) **lebih besar** dari `amount` (25000) karena
> PayPan menambah kode unik 1–999. Yang masuk ke saldo adalah `total` **penuh**.
> Pelanggan tidak boleh kurang 1 rupiah.

### Cek status

```http
GET /api/topup/{invoice_id}
```

```json
{"invoice_id":"...","status":"pending","amount":25000,"total":25001,"pay_url":"/pay/..."}
```

`status`: `pending` | `paid` | `expired` | `refunded`

### Halaman bayar

`GET {base}/pay/{paypan_invoice_id}` — halaman QR untuk pelanggan.

---

## 6. Webhook PayPan (masuk ke sistem)

```http
POST /webhook/paypan
Content-Type: application/json
X-Paypan-Signature: <hmac-sha256 hex>

{
  "event": "order.paid",
  "order": {
    "id": "27d22bd8fececb99",
    "price": 25000,
    "code": 1,
    "total": 25001,
    "paid_at": 1792822535
  }
}
```

Verifikasi signature:

```
HMAC_SHA256(secret, raw_body) == X-Paypan-Signature
```

Respons:

| Situation | Kode | Body |
|---|---|---|
| Order milik PanWallet, baru lunas | 200 | `{"ok":true,"invoice":"...","credited":25001}` |
| Order milik PanWallet, sudah pernah diproses | 200 | `{"ok":true,...}` (tidak dobel kredit) |
| Order milik aplikasi lain | 200 | `{"ok":true,"ignored":"<order-id>"}` |
| Signature salah | 403 | `{"error":"signature tidak valid"}` |

> **Penting:** PayPan mengirim ke **semua** webhook aktif tanpa filter per
> aplikasi. Order milik AgenPulsa ikut sampai — PanWallet mengabaikannya
> dengan 200 supaya PayPan tidak retry sia-sia.

---

## 7. Notifikasi keluar (webhook bot)

```http
POST https://bot-anda/hook/panwallet
Content-Type: application/json

{
  "event": "topup.paid",
  "public_id": "65ZMPUNP",
  "requested": 25000,
  "amount": 25001,
  "balance": 150001,
  "paid_at": 1792822535,
  "invoice_id": "04ea9711884ed790..."
}
```

| Field | Arti |
|---|---|
| `requested` | nominal yang diminta pelanggan (25000) |
| `amount` | yang benar-benar masuk saldo (25001, termasuk kode unik) |
| `balance` | saldo setelah top-up |

Event: `topup.paid`, `test`.

---

## 8. Health check

```http
GET /healthz
```

```json
{
  "status": "ok",
  "components": [
    {"name":"database","ok":true,"message":"ok"},
    {"name":"website","ok":true,"message":"ok"},
    {"name":"paypan","ok":false,"message":"belum dikonfigurasi"}
  ]
}
```

`503` kalau database atau website bermasalah. `paypan` yang false **tidak**
menggagalkan health (top-up hanya perlu saat pelanggan mau isi saldo).

---

## 9. Autentikasi API produk

```http
Authorization: Bearer pw_live_xxxxxxxxxxxxxxxxxxxxxxxx
```

- Token dibuat di panel → **API Key Produk** → nama produk.
- Token **tampil sekali** saat dibuat.
- Token terikat satu produk. Token `agenpulsa` tidak bisa dipakai produk lain.
- Hash SHA-256 yang disimpan, bukan token aslinya.
- Dicabut lewat `POST /admin/keys/{product}/revoke`.

---

## 10. Checklist integrasi bot

- [ ] Simpan `public_id` per pelanggan (bukan Telegram ID, bukan nomor HP)
- [ ] Panggil `resolve` sekali saja, simpan hasilnya
- [ ] Cek `available` (bukan `balance`) sebelum order
- [ ] Kalau kurang → arahkan top-up, **berhenti**
- [ ] `hold` dengan `Idempotency-Key` sebelum proses order
- [ ] `capture` kalau sukses, `release` kalau gagal
- [ ] Semua nominal integer rupiah
- [ ] Token API hanya di `.env`, tidak di source
- [ ] Handle `422` (saldo kurang) dan `429` (rate limit)
- [ ] Jangan pernahMzimpan saldo di bot

---

## 11. Contoh lengkap (curl)

```bash
BASE="https://panwallet.jhopan.my.id"
KEY="pw_live_xxxxxxxx"

# 1. daftar pelanggan sekali
RES=$(curl -s -X POST "$BASE/v1/identity/resolve" \
  -H "Content-Type: application/json" \
  -d '{"provider":"telegram","external_id":"123456789"}')
PUB=$(echo "$RES" | jq -r .public_id)

# 2. cek saldo
SALDO=$(curl -s "$BASE/v1/public/$PUB" | jq -r .available)

if [ "$SALDO" -lt 25000 ]; then
  echo "Saldo kurang. Isi dulu di @PanWalletBot"
  exit 0
fi

# 3. kunci saldo
HOLD=$(curl -s -X POST "$BASE/v1/holds" \
  -H "Authorization: Bearer $KEY" \
  -H "Idempotency-Key: hold:INV-001" \
  -H "Content-Type: application/json" \
  -d '{"wallet_user_id":"<WALLET_UUID>","reference_id":"INV-001","amount":25000,"expires_at":"2030-01-01T00:00:00Z"}')
HOLD_ID=$(echo "$HOLD" | jq -r .id)

# 4. proses order (milik bot toko)
if kirim_pulsa_sukses; then
  curl -s -X POST "$BASE/v1/holds/$HOLD_ID/capture" \
    -H "Authorization: Bearer $KEY" -H "Idempotency-Key: capture:INV-001"
else
  curl -s -X POST "$BASE/v1/holds/$HOLD_ID/release" \
    -H "Authorization: Bearer $KEY" -H "Idempotency-Key: release:INV-001"
fi
```

---

## 12. Status implementasi PayPan

| Fitur PayPan | Status di PanWallet |
|---|---|
| `POST /api/invoice` (buat invoice) | ✅ dipakai |
| `GET /api/invoice/{id}` (status) | ✅ dipakai — checkout, rekonsiliasi, panel |
| `POST /api/invoice/{id}/cancel` (batalkan) | ⚠️ client ada, belum ada tombol di panel |
| `POST /api/invoice/{id}/refund` | ⚠️ client belum ada, hanya dicatat di rekonsiliasi |
| `POST /api/notif` | ➖ tidak perlu — PayPan mendengarin pembayaran sendiri |
| `POST /api/order` | ➖ tidak dipakai — mode invoice QRIS yang dipakai |

Catatan: **refund di PayPan itu administratif** (status invoice jadi
`refunded`). PayPan tidak mengembalikan dana. Pengembalian dana ke pelanggan
adalah urusan manual admin.

---

## 13. Berkas terkait

| Berkas | Isi |
|---|---|
| `docs/API.md` | referensi endpoint lengkap, 22 bagian |
| `docs/INTEGRASI.md` | panduan integrasi bot step-by-step |
| `docs/BACKUP.md` | backup, restore, ekspor CSV |
| `docs/APIDOCS.md` | dokumen ini (ringkas, untuk AI) |
