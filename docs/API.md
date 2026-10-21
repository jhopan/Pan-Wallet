# Pan-Wallet API

Pusat saldo pelanggan untuk semua produk Jhopan: AgenPulsa, Tembak Paket, AutoBuy, Xray Premium, dan bot Telegram/WhatsApp.

Produk **tidak pernah** mengubah saldo langsung. Semua perubahan saldo lewat API ini.

Base URL: `http://127.0.0.1:8080` (lokal) atau domain produksi.
Format: JSON. Error selalu `{"error": "pesan"}`.

---

## 1. Konsep saldo

Satu pelanggan punya tiga angka:

| Angka | Arti |
|---|---|
| `balance` | Total saldo tersimpan |
| `held` | Saldo yang dikunci order berjalan |
| `available` | `balance - held` — yang bisa dipakai |

Pola **hold -> capture/release**:

```
hold     = kunci saldo untuk satu order
capture  = order sukses, saldo dipotong permanen + ledger dicatat
release  = order gagal, kunci dilepas, saldo tidak berubah
```

Kalau produk memotong saldo langsung lalu order gagal, uang pelanggan hilang. Hold mencegah itu.

---

## 2. Autentikasi produk

Setiap produk punya API key sendiri. Backend hanya menyimpan **hash SHA-256**.

```text
Authorization: Bearer pw_live_xxxxxxxxxxxxxxxx
```

- Key `agenpulsa` tidak bisa menyentuh hold milik `tembak-paket`.
- Key tidak pernah dikirim ke browser atau bot pelanggan.
- Key tidak memberi akses langsung ke database.

Buat key lewat seed script:

```bash
go run scripts/seed-demo.go
```

---

## 3. POST /v1/holds

Mengunci saldo untuk satu order.

**Header**

```text
Authorization: Bearer <api-key>
Idempotency-Key: hold:<reference-id>
Content-Type: application/json
```

**Body**

```json
{
  "wallet_user_id": "0bb67b16bd7d5ea02ef3653885d3a174",
  "reference_id": "AP-1001",
  "amount": 25000,
  "expires_at": "2030-01-01T00:00:00Z"
}
```

`product` **tidak** dikirim di body, diambil dari API key.

**Sukses 201**

```json
{
  "ID": "d354511cfc933ad67854ea26edfda0bd",
  "WalletUserID": "0bb67b16bd7d5ea02ef3653885d3a174",
  "Product": "agenpulsa",
  "ReferenceID": "AP-1001",
  "Amount": 25000,
  "Status": "active",
  "IdempotencyKey": "hold:AP-1001"
}
```

**Error**

| HTTP | Arti |
|---|---|
| `400` | Body tidak valid |
| `401` | API key salah / tidak aktif |
| `422` | Saldo tidak cukup, atau idempotency key dipakai dengan payload berbeda |

Retry dengan `Idempotency-Key` sama mengembalikan hold yang sama.

---

## 4. POST /v1/holds/{id}/capture

Order sukses. Saldo dipotong, ledger dicatat.

```text
Authorization: Bearer <api-key>
Idempotency-Key: capture:<reference-id>
```

Body: `{}`

**Sukses 200**

```json
{
  "id": "d354511cfc933ad67854ea26edfda0bd",
  "status": "captured",
  "amount": 10000,
  "product": "agenpulsa"
}
```

**Error**

| HTTP | Arti |
|---|---|
| `401` | API key salah |
| `404` | Hold tidak ada atau milik produk lain |
| `409` | Hold sudah diproses / kedaluwarsa / idempotency key konflik |
| `422` | Saldo tidak cukup |

Capture dua kali dengan key sama = aman, saldo tidak terpotong dua kali.

---

## 5. POST /v1/holds/{id}/release

Order gagal. Saldo tidak berubah.

```text
Authorization: Bearer <api-key>
Idempotency-Key: release:<reference-id>
```

**Sukses 200**

```json
{
  "id": "5220cef866c1d7462de859ad32196822",
  "status": "released",
  "amount": 5000,
  "product": "agenpulsa"
}
```

---

## 6. GET /v1/wallets/{wallet_user_id}

Cek saldo pelanggan (untuk bot menampilkan "saldo kamu: Rp...").

```text
Authorization: Bearer <api-key>
```

**Sukses 200**

```json
{
  "wallet_user_id": "0bb67b16bd7d5ea02ef3653885d3a174",
  "balance": 170002,
  "held": 20000,
  "available": 150002,
  "currency": "IDR"
}
```

---

## 7. Top-up QRIS lewat PayPan

Pan-Wallet tidak membuat QR sendiri. Dia meminta invoice ke PayPan, lalu menerima webhook saat lunas.

```
Pelanggan minta top-up Rp50.000
   -> Pan-Wallet POST PayPan /api/invoice
   -> PayPan tentukan kode unik, balas total 50.001 + pay_url
   -> Pelanggan bayar QRIS
   -> HP merchant deteksi notif, kirim ke PayPan
   -> PayPan cocokkan, POST /webhook/paypan (HMAC)
   -> Pan-Wallet verifikasi, saldo bertambah + ledger
```

**Konfigurasi** (panel -> Konfigurasi, berlaku tanpa restart):

| Field | Contoh |
|---|---|
| Base URL PayPan | `https://paypan.jhopan.my.id` |
| Token API (scope order) | `pp_xxxxxxxxxxxxxxxx` |
| Webhook secret | `pp_xxxxxxxxxxxxxxxx` |

Daftarkan URL webhook ini di admin PayPan -> Konfigurasi -> Webhook:

```text
https://<domain-pan-wallet>/webhook/paypan
```

**POST /webhook/paypan** (dipanggil PayPan, bukan produk)

```text
X-Paypan-Event: order.paid
X-Paypan-Signature: <hex HMAC-SHA256 dari raw body>
```

```json
{"event":"order.paid","order":{"id":"...","price":50000,"code":1,"total":50001,"paid_at":1790000000}}
```

Signature salah -> `403`. Webhook dobel -> saldo tetap (idempoten).

| Endpoint | Fungsi |
|---|---|
| `GET /pay/{id}` | redirect pelanggan ke halaman bayar PayPan |
| `GET /api/topup/{id}` | status invoice untuk polling |

---

## 8. Panel kendali

| URL | Fungsi |
|---|---|
| `/` | 404 (disamarkan) |
| `/admin/login` | Login panel |
| `/admin` | Dashboard: saldo, hold, ledger, invoice |
| `/admin/topup` | Buat invoice top-up, sync status, tandai lunas |
| `/admin/ledger` | Riwayat mutasi saldo per wallet |
| `/admin/keys` | Buat / rotasi / nonaktifkan API key produk |
| `/admin/config` | Konfigurasi PayPan |

---

## 9. Aturan uang yang tidak boleh dilanggar

- Nominal integer rupiah, bukan float.
- Saldo tidak pernah negatif.
- Hold mengurangi `available`, bukan `balance`.
- Capture hanya sekali per hold.
- Setiap perubahan saldo punya baris `wallet_ledger` dengan `product`, `reference_id`, dan `idempotency_key` unik.
- Webhook PayPan satu-satunya jalur kredit top-up QRIS.
- Hold kedaluwarsa tidak bisa di-capture.

---

## 10. Contoh lengkap (curl)

```bash
# 1. cek saldo
curl -s http://127.0.0.1:8080/v1/wallets/0bb67b16bd7d5ea02ef3653885d3a174 \
  -H "Authorization: Bearer pw_live_xxx"

# 2. kunci saldo
curl -s -X POST http://127.0.0.1:8080/v1/holds \
  -H "Authorization: Bearer pw_live_xxx" \
  -H "Idempotency-Key: hold:AP-1001" \
  -H "Content-Type: application/json" \
  -d '{"wallet_user_id":"0bb67b16bd7d5ea02ef3653885d3a174","reference_id":"AP-1001","amount":25000,"expires_at":"2030-01-01T00:00:00Z"}'

# 3a. order sukses -> potong saldo
curl -s -X POST http://127.0.0.1:8080/v1/holds/<HOLD_ID>/capture \
  -H "Authorization: Bearer pw_live_xxx" \
  -H "Idempotency-Key: capture:AP-1001"

# 3b. order gagal -> lepas kunci
curl -s -X POST http://127.0.0.1:8080/v1/holds/<HOLD_ID>/release \
  -H "Authorization: Bearer pw_live_xxx" \
  -H "Idempotency-Key: release:AP-1001"
```

---

## 11. Status implementasi

| Fitur | Status |
|---|---|
| `POST /v1/holds` | selesai |
| `POST /v1/holds/{id}/capture` | selesai |
| `POST /v1/holds/{id}/release` | selesai |
| `GET /v1/wallets/{id}` | selesai |
| Top-up via PayPan + webhook | selesai |
| Panel admin (dashboard, topup, config) | selesai |
| Kelola API key dari panel | selesai |
| Riwayat mutasi di panel | selesai |
| Rekonsiliasi otomatis | belum (tombol sync manual ada) |
| Identity Telegram/WhatsApp | schema ada, endpoint belum |
