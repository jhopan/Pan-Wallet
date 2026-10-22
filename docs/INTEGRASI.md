# Integrasi Pan-Wallet untuk Bot Lain

Panduan untuk bot toko (AgenPulsa, Xray, WhatsApp, dsb) yang ingin memakai
saldo Pan-Wallet. Semua contoh memakai `curl`, jadi bisa dipakai dari bahasa
apa pun, atau ditempel ke script.

---

## Aturan main

1. **Bot toko tidak memegang saldo.** Saldo hanya ada di Pan-Wallet.
2. **Bot toko tidak punya fitur top-up.** Kalau saldo kurang, arahkan pelanggan
   ke bot saldo (mis. `@PanWalletBot`).
3. **Identitas pelanggan = `public_id`** (8 karakter, mis. `65ZMPUNP`).
   Simpan ini di database bot toko. Jangan pakai Telegram ID sebagai kunci saldo.
4. **Semua nominal integer rupiah.** `25000`, bukan `25000.00`.
5. **Token hanya di `.env`.** Jangan tulis di source atau commit.

---

## Langkah 1 — Dapatkan API key produk

1. Buka panel: `https://wallet.domain-kamu/admin` → login admin.
2. Masuk menu **API Key Produk**.
3. Isi nama produk (mis. `agenpulsa`), klik **Buat key**.
4. **Token tampil sekali** — salin sekarang, bentuknya `pw_live_xxxxxxxx...`.
5. Simpan di `.env` bot toko:

```bash
PANWALLET_URL=https://wallet.domain-kamu
PANWALLET_TOKEN=pw_live_xxxxxxxxxxxxxxxxxxxxxxxx
```

Token ini **hanya berlaku untuk produk yang disebutkan saat dibuat**. Token
`agenpulsa` tidak bisa dipakai produk lain.

---

## Langkah 2 — Daftarkan pelanggan (sekali saja)

Saat pelanggan pertama kali memakai bot toko, tukar identitas channel mereka
menjadi `public_id`:

```bash
curl -s -X POST "$PANWALLET_URL/v1/identity/resolve" \
  -H "Content-Type: application/json" \
  -d '{"provider":"telegram","external_id":"123456789"}'
```

Balasan:

```json
{
  "public_id": "65ZMPUNP",
  "is_new": true,
  "balance": 0,
  "available": 0
}
```

`is_new: true` berarti wallet baru dibuat. Simpan `public_id` di database bot
toko, lalu jangan panggil lagi untuk pelanggan yang sama.

`provider` yang didukung: `telegram`, `whatsapp`, `email`.

---

## Langkah 3 — Cek saldo

**Cara A — lewat public_id (tanpa token, aman untuk bot):**

```bash
curl -s "$PANWALLET_URL/v1/public/65ZMPUNP"
```

```json
{
  "public_id": "65ZMPUNP",
  "balance": 150001,
  "held": 0,
  "available": 150001,
  "status": "active"
}
```

**Cara B — lewat wallet_user_id (butuh token produk):**

```bash
curl -s "$PANWALLET_URL/v1/wallets/<wallet_user_id>" \
  -H "Authorization: Bearer $PANWALLET_TOKEN"
```

Pakai `available`, bukan `balance`. `available` = saldo yang benar-benar bisa
dipakai (sudah dikurangi saldo yang sedang ditahan).

---

## Langkah 4 — Potong saldo saat transaksi

Ini tiga langkah: **hold → capture** kalau sukses, **hold → release** kalau gagal.

Kenapa tidak potong langsung? Supaya kalau order gagal di tengah jalan, saldo
pelanggan tidak hangus.

### 4a. Kunci saldo (hold)

```bash
curl -s -X POST "$PANWALLET_URL/v1/holds" \
  -H "Authorization: Bearer $PANWALLET_TOKEN" \
  -H "Idempotency-Key: hold:INV-2026-001" \
  -H "Content-Type: application/json" \
  -d '{
    "wallet_user_id": "<wallet_user_id>",
    "reference_id": "INV-2026-001",
    "amount": 25000,
    "expires_at": "2030-01-01T00:00:00Z"
  }'
```

Balasan `201`:

```json
{"id": "<hold_id>", "status": "active", "amount": 25000}
```

Kalau saldo tidak cukup, balasan `422` dengan pesan `saldo tidak cukup`.
Di titik ini bot toko **harus berhenti** dan mengarahkan pelanggan ke bot saldo.

### 4b. Order sukses → potong permanen

```bash
curl -s -X POST "$PANWALLET_URL/v1/holds/<hold_id>/capture" \
  -H "Authorization: Bearer $PANWALLET_TOKEN" \
  -H "Idempotency-Key: capture:INV-2026-001"
```

### 4c. Order gagal → kembalikan saldo

```bash
curl -s -X POST "$PANWALLET_URL/v1/holds/<hold_id>/release" \
  -H "Authorization: Bearer $PANWALLET_TOKEN" \
  -H "Idempotency-Key: release:INV-2026-001"
```

---

## Idempotency-Key: penting

`Idempotency-Key` mencegah saldo terpotong dua kali kalau request diulang
(misal timeout lalu bot retry).

- Kunci **sama** + isi sama → aman, hasil sama, tidak dobel.
- Kunci **sama** + isi beda → ditolak (mencegah bug).
- Kunci **beda** → dianggap transaksi baru.

Format yang disarankan: `hold:<nomor-invoice>`, `capture:<nomor-invoice>`.

---

## Kode balasan

| Kode | Arti | Yang harus dilakukan bot |
|---|---|---|
| `201` | hold dibuat | lanjut proses order |
| `200` | berhasil | lanjut |
| `401` | token salah / tidak ada | cek `.env`, jangan retry |
| `403` | wallet diblokir | beri tahu pelanggan hubungi admin |
| `422` | saldo tidak cukup | arahkan ke bot saldo |
| `404` | wallet/hold tidak ada | cek ID |

---

## Kalau saldo kurang — arahkan, jangan top-up sendiri

```json
{
  "ok": false,
  "error": "saldo tidak cukup",
  "available": 5000,
  "required": 25000
}
```

Bot toko merespons:

```
Saldo kamu Rp 5.000, transaksi ini butuh Rp 25.000.

Isi saldo dulu di @PanWalletBot
```

**Jangan** membuat fitur top-up di bot toko. Semua pembayaran lewat PayPan dan
hanya diproses Pan-Wallet.

---

## Notifikasi ke bot toko (opsional)

Pan-Wallet bisa mengirim webhook setiap ada top-up lunas. Isi di panel
**Konfigurasi → Notifikasi → Webhook URL**:

```bash
POST https://bot-toko-kamu/hook/panwallet
Content-Type: application/json

{
  "event": "topup.paid",
  "public_id": "65ZMPUNP",
  "amount": 25004,
  "balance": 125004,
  "paid_at": 1790000000,
  "invoice_id": "5d73c16e..."
}
```

Field penting:

| Field | Arti |
|---|---|
| `amount` | nominal top-up (sudah termasuk kode unik) |
| `balance` | **saldo pelanggan setelah** top-up ini |
| `paid_at` | waktu lunas, unix detik (zona server) |
| `invoice_id` | id invoice internal, untuk rekonsiliasi |

Event yang dikirim: `topup.paid`, `test`.

### Kirim ke topic supergrup (mis. Testimoni)

Telegram mengirim pesan ke supergrup dengan `message_thread_id`:

```bash
curl -s -X POST "https://api.telegram.org/bot<TOKEN>/sendMessage" \
  -d "chat_id=-1001234567890" \
  -d "message_thread_id=12" \
  -d "text=Top-up berhasil Rp25.004"
```

Pan-Wallet sudah mendukung ini: panel **Konfigurasi → Notifikasi →
Topic ID supergrup**. Isi angka topic-nya (bukan nama).

**Penting:** webhook hanya dikirim **sekali per pelunasan**. Kalau PayPan
mengirim ulang webhook yang sama (retry), notifikasi tidak akan dobel —
dijaga oleh flag `FreshlyPaid` di sisi Pan-Wallet.

Kalau bot toko ingin tahu saldo pelanggan berubah, cukup panggil
`GET /v1/public/<public_id>` saat dibutuhkan — tidak perlu menyimpan saldo.

---

## Ringkasan endpoint

| Endpoint | Auth | Fungsi |
|---|---|---|
| `POST /v1/identity/resolve` | publik | telegram_id → public_id |
| `GET /v1/public/{public_id}` | publik | cek saldo pelanggan |
| `POST /v1/public/{public_id}/topup` | publik | buat invoice top-up |
| `GET /v1/wallets/{wallet_user_id}` | token produk | cek saldo detail |
| `POST /v1/holds` | token produk | kunci saldo |
| `POST /v1/holds/{id}/capture` | token produk | potong saldo |
| `POST /v1/holds/{id}/release` | token produk | lepas kunci |
| `GET /healthz` | publik | status layanan |

---

## Contoh lengkap: satu transaksi

```bash
# 1. pelanggan minta beli pulsa 25.000
PUBLIC_ID=65ZMPUNP

# 2. cek saldo
SALDO=$(curl -s "$PANWALLET_URL/v1/public/$PUBLIC_ID")
AVAILABLE=$(echo "$SALDO" | jq -r .available)

if [ "$AVAILABLE" -lt 25000 ]; then
  echo "Saldo kurang. Isi dulu di @PanWalletBot"
  exit 0
fi

# 3. kunci saldo
HOLD=$(curl -s -X POST "$PANWALLET_URL/v1/holds" \
  -H "Authorization: Bearer $PANWALLET_TOKEN" \
  -H "Idempotency-Key: hold:INV-001" \
  -H "Content-Type: application/json" \
  -d "{\"wallet_user_id\":\"$WALLET_ID\",\"reference_id\":\"INV-001\",\"amount\":25000,\"expires_at\":\"2030-01-01T00:00:00Z\"}")
HOLD_ID=$(echo "$HOLD" | jq -r .id)

# 4. kirim pulsa ke pelanggan (proses bot toko sendiri)
if kirim_pulsa_sukses; then
  curl -s -X POST "$PANWALLET_URL/v1/holds/$HOLD_ID/capture" \
    -H "Authorization: Bearer $PANWALLET_TOKEN" \
    -H "Idempotency-Key: capture:INV-001"
else
  curl -s -X POST "$PANWALLET_URL/v1/holds/$HOLD_ID/release" \
    -H "Authorization: Bearer $PANWALLET_TOKEN" \
    -H "Idempotency-Key: release:INV-001"
fi
```
