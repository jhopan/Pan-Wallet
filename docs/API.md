# PanWallet API

Pusat saldo pelanggan untuk semua produk Jhopan: AgenPulsa, Tembak Paket, AutoBuy, Xray Premium, dan bot Telegram/WhatsApp.

Produk **tidak pernah** mengubah saldo langsung. Semua perubahan saldo lewat API ini.

Base URL: `http://127.0.0.1:8080` (lokal) atau domain produksi.
Format: JSON. Error selalu `{"error": "pesan"}`.

## 0. Daftar endpoint

### API produk (butuh API key)

| Method | Endpoint | Fungsi |
|---|---|---|
| POST | `/v1/holds` | kunci saldo untuk satu order |
| POST | `/v1/holds/{id}/capture` | order sukses, potong saldo |
| POST | `/v1/holds/{id}/release` | order gagal, lepas kunci |
| GET | `/v1/wallets/{wallet_user_id}` | cek saldo (versi produk) |

### API publik (tanpa auth, rate limit 30/menit)

| Method | Endpoint | Fungsi |
|---|---|---|
| GET | `/v1/public/{id}` | cek saldo pakai ID pelanggan |
| POST | `/v1/public/{id}/topup` | buat invoice top-up |
| GET | `/v1/public/{id}/identities` | channel yang tertaut |
| POST | `/v1/identity/resolve` | bot tanya "ini siapa?" |
| POST | `/v1/identity/link/start` | minta kode ikat channel baru |
| POST | `/v1/identity/link/confirm` | konfirmasi kode dari Telegram |

### Webhook

| Method | Endpoint | Fungsi |
|---|---|---|
| POST | `/webhook/paypan` | terima `order.paid` dari PayPan (HMAC) |

### Halaman web

| Method | Endpoint | Fungsi |
|---|---|---|
| GET | `/masuk` | form masuk pelanggan (email) |
| POST | `/masuk` | kirim email, terbitkan kode |
| POST | `/masuk/cek` | cek tautan, tampilkan saldo |
| GET | `/pay/{id}` | redirect ke halaman bayar PayPan |
| GET | `/api/topup/{id}` | status invoice (JSON) |
| GET | `/healthz` | cek service hidup |
| GET | `/` | 404 (disamarkan) |

### Panel admin (butuh sesi login)

| Method | Endpoint | Fungsi |
|---|---|---|
| GET | `/admin/login` | form login |
| POST | `/admin/login` | proses login |
| GET | `/admin/logout` | keluar |
| GET | `/admin` | dashboard saldo & invoice |
| GET | `/admin/wallets` | daftar wallet pelanggan |
| POST | `/admin/wallets` | buat wallet baru |
| POST | `/admin/wallets/{id}/status` | blokir / aktifkan wallet |
| GET | `/admin/ledger` | riwayat mutasi |
| GET | `/admin/keys` | daftar API key produk |
| POST | `/admin/keys` | buat / rotasi key |
| POST | `/admin/keys/{product}/revoke` | nonaktifkan key |
| GET | `/admin/topup` | daftar invoice top-up |
| POST | `/admin/topup` | buat invoice top-up |
| POST | `/admin/topup/{id}/paid` | tandai lunas manual |
| POST | `/admin/topup/{id}/sync` | cek status ke PayPan |
| GET | `/admin/config` | halaman konfigurasi |
| POST | `/admin/config` | simpan konfigurasi PayPan |
| POST | `/admin/wallets/adjust` | tambah/kurangi saldo manual + audit |
| GET | `/admin/audit` | jejak tindakan admin |
| GET | `/admin/laporan` | laporan pendapatan |
| GET | `/admin/backup` | daftar backup |
| POST | `/admin/backup` | buat backup sekarang |

---

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

## 7. ID Pelanggan (dipakai semua bot)

Setiap wallet punya **ID pendek 8 karakter** yang dipakai pelanggan di channel mana pun:

```
JZXJFADE
WN4H3E5T
```

- Tanpa huruf ambigu (`0/O`, `1/I/L`) supaya tidak salah ketik di chat.
- Unik, huruf besar, tapi terima input huruf kecil juga.
- **Satu ID untuk semua channel**: Telegram, WhatsApp, web, bot apa pun.

Bot tidak perlu menyimpan UUID internal. Cukup simpan `public_id` milik pelanggan.

### `GET /v1/public/{id}` — cek saldo tanpa auth

```bash
curl -s https://wallet.domain/v1/public/JZXJFADE
```

```json
{
  "public_id": "JZXJFADE",
  "balance": 150000,
  "held": 0,
  "available": 150000,
  "currency": "IDR",
  "status": "active"
}
```

`404` bila ID tidak ada. `status` bernilai `active` atau `blocked`.

Catatan keamanan: endpoint ini tanpa auth karena ID itu sendiri adalah rahasia
(8 karakter acak dari 31 huruf = ±39 bit). Jangan pernah menampilkan daftar ID
publik. Kalau butuh proteksi lebih, tambahkan rate limit per IP.

### `POST /v1/public/{id}/topup` — buat invoice top-up

```bash
curl -s -X POST https://wallet.domain/v1/public/WN4H3E5T/topup \
  -H "Content-Type: application/json" \
  -d '{"amount": 50000}'
```

```json
{
  "invoice_id": "8c451a1ee6af02fdf426c35f2702877b",
  "total": 50002,
  "status": "pending",
  "pay_url": "/pay/cdca10f00e000623",
  "expires_at": 1792551559
}
```

Bot tinggal mengirim `pay_url` (gabung dengan base URL PayPan) ke pelanggan.
Setelah pelanggan bayar, webhook PayPan menambah saldo otomatis.

`403` bila wallet diblokir.


---

## 8. Identitas: satu saldo untuk semua channel

**Telegram adalah identitas primer.** Telegram ID diberikan oleh server Telegram,
tidak bisa dipalsukan pelanggan — jadi aman dijadikan kunci.

Channel lain (WhatsApp, email, web) **ditaulukan** ke wallet yang sama lewat kode
6 angka yang dikonfirmasi di Telegram. Tidak perlu kirim email sama sekali.

```
Telegram ID 123456789  ──►  ID Pelanggan JKCUX8CK  ──►  Saldo
Email budi@gmail.com   ──┘
WhatsApp 628123456789  ──┘
```

### `POST /v1/identity/resolve` — bot tanya "ini siapa?"

Dipanggil bot setiap pelanggan chat.

```bash
curl -s -X POST https://wallet.domain/v1/identity/resolve \
  -H "Content-Type: application/json" \
  -d '{"provider":"telegram","external_id":"123456789"}'
```

```json
{
  "public_id": "JKCUX8CK",
  "balance": 0,
  "held": 0,
  "available": 0,
  "currency": "IDR",
  "status": "active",
  "is_new": true
}
```

- **Telegram**: kalau belum ada, wallet dibuat otomatis (`is_new: true`).
- **Email/WhatsApp/web**: kalau belum tertaut, balas `404` — pelanggan harus mengikat dulu.

### `POST /v1/identity/link/start` — minta kode ikat

Dipanggil website saat pelanggan memasukkan email.

```bash
curl -s -X POST https://wallet.domain/v1/identity/link/start \
  -H "Content-Type: application/json" \
  -d '{"provider":"email","external_id":"Budi@Gmail.com"}'
```

```json
{
  "code": "674089",
  "instruction": "Buka bot Telegram PanWallet, lalu kirim:\n\n/ikat 674089\n\nKode berlaku 10 menit.",
  "expires_in": 600
}
```

Tampilkan `instruction` ke pelanggan. Kode berlaku 10 menit.

`409` bila identitas itu sudah tertaut ke akun lain.

### `POST /v1/identity/link/confirm` — bot Telegram konfirmasi

Dipanggil bot saat pelanggan mengirim `/ikat 674089`.

```bash
curl -s -X POST https://wallet.domain/v1/identity/link/confirm \
  -H "Content-Type: application/json" \
  -d '{"code":"674089","telegram_id":"123456789"}'
```

```json
{"public_id": "JKCUX8CK", "linked": true}
```

Setelah ini, `resolve` dengan email tersebut mengembalikan wallet yang sama.

### `GET /v1/public/{id}/identities` — lihat channel tertaut

```json
{
  "public_id": "JKCUX8CK",
  "links": ["email:budi@gmail.com", "telegram:123456789"]
}
```

### Aturan keamanan

| Aturan | Alasan |
|---|---|
| Hanya Telegram yang boleh membuat wallet otomatis | Telegram ID tidak bisa dipalsukan |
| Channel lain wajib lewat kode + konfirmasi Telegram | Mencegah orang menautkan email/nomor orang lain |
| Kode sekali pakai, berlaku 10 menit | Mencegah dipakai ulang |
| Identitas yang sudah tertaut tidak bisa direbut | Mencegah pengambilalihan akun |
| Email dinormalisasi ke huruf kecil | `Budi@Gmail.com` = `budi@gmail.com` |
| Nomor WA dinormalisasi ke format 62 | `0812...` = `62812...` |

### Contoh alur website

```
1. Pelanggan buka website, masukkan email budi@gmail.com
2. Website -> POST /v1/identity/link/start {provider:"email", external_id:"budi@gmail.com"}
3. Website tampilkan: "Kirim /ikat 674089 ke bot Telegram PanWallet"
4. Pelanggan buka Telegram, kirim /ikat 674089
5. Bot -> POST /v1/identity/link/confirm {code:"674089", telegram_id:"123456789"}
6. Bot: "Berhasil! Akun terhubung."
7. Website cek ulang -> POST /v1/identity/resolve {provider:"email", ...} -> dapat public_id
8. Website tampilkan saldo
```


---

## 9. Portal pelanggan (web)

Pelanggan bisa lihat saldo di browser tanpa install apa pun.

| URL | Fungsi |
|---|---|
| `/masuk` | Form masukkan email |
| `POST /masuk` | Kirim email -> terbitkan kode ikat |
| `POST /masuk/cek` | Cek apakah sudah tertaut, tampilkan saldo |

Alur:

```
1. Buka /masuk, masukkan email
2. Kalau sudah tertaut -> saldo langsung tampil
3. Kalau belum -> tampil kode 6 angka + instruksi /ikat ke bot Telegram
4. Kirim kode ke Telegram, klik "Saya sudah kirim"
5. Saldo + riwayat tampil
```

Halaman ini tidak butuh password. Verifikasi terjadi lewat Telegram.

---

## 10. Rate limit endpoint publik

Semua endpoint publik dibatasi **30 permintaan per menit per IP**:

| Endpoint | Batas |
|---|---|
| `GET /v1/public/{id}` | 30/menit |
| `POST /v1/public/{id}/topup` | 30/menit |
| `POST /v1/identity/resolve` | 30/menit |
| `POST /v1/identity/link/start` | 30/menit |
| `POST /v1/identity/link/confirm` | 30/menit |
| `GET /masuk`, `POST /masuk` | 30/menit |

Melebihi batas -> `429` dengan header `Retry-After: 60`.

IP diambil dari `CF-Connecting-IP`, lalu `X-Forwarded-For`, lalu `RemoteAddr` —
sadar Cloudflare dan reverse proxy.

---

## 11. Rekonsiliasi otomatis

Webhook bisa gagal (jaringan, firewall, restart). Karena itu ada pemeriksaan
berkala sebagai cadangan:

- Setiap **2 menit**, PanWallet menanyakan status invoice pending yang berumur
  lebih dari **3 menit** ke PayPan.
- `paid` -> saldo dikredit (idempoten, tidak dobel).
- `expired` / `refunded` -> invoice ditandai kedaluwarsa.
- `pending` -> dibiarkan untuk putaran berikutnya.

Artinya: pelanggan tetap dapat saldo walau webhook tidak sampai.

---

## 12. Top-up QRIS lewat PayPan

PanWallet tidak membuat QR sendiri. Dia meminta invoice ke PayPan, lalu menerima webhook saat lunas.

```
Pelanggan minta top-up Rp50.000
   -> PanWallet POST PayPan /api/invoice
   -> PayPan tentukan kode unik, balas total 50.001 + pay_url
   -> Pelanggan bayar QRIS
   -> HP merchant deteksi notif, kirim ke PayPan
   -> PayPan cocokkan, POST /webhook/paypan (HMAC)
   -> PanWallet verifikasi, saldo bertambah + ledger
```

**Konfigurasi** (panel -> Konfigurasi, berlaku tanpa restart):

| Field | Contoh |
|---|---|
| Base URL PayPan | `https://paypan.jhopan.my.id` |
| Token API (scope order) | `pp_xxxxxxxxxxxxxxxx` |
| Webhook secret | `pp_xxxxxxxxxxxxxxxx` |

Daftarkan URL webhook ini di admin PayPan -> Konfigurasi -> Webhook:

```text
https://<domain-panwallet>/webhook/paypan
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

## 13. Panel kendali

| URL | Fungsi |
|---|---|
| `/` | 404 (disamarkan) |
| `/admin/login` | Login panel |
| `/admin` | Dashboard: saldo, hold, ledger, invoice |
| `/admin/wallets` | Daftar wallet, buat wallet, blokir/aktifkan |
| `/admin/ledger` | Riwayat mutasi saldo per wallet |
| `/admin/keys` | Buat / rotasi / nonaktifkan API key produk |
| `/admin/topup` | Buat invoice top-up, sync status, tandai lunas |
| `/admin/config` | Konfigurasi PayPan |

### Aksi panel yang mengubah data

| Aksi | Efek |
|---|---|
| `POST /admin/wallets` | buat wallet + ID pelanggan baru |
| `POST /admin/wallets/{id}/status` | `blocked` = tidak bisa hold/top-up (403) |
| `POST /admin/keys` | buat key baru; produk sama = rotasi, token lama mati |
| `POST /admin/keys/{product}/revoke` | key langsung tidak valid (401) |
| `POST /admin/topup` | minta invoice ke PayPan |
| `POST /admin/topup/{id}/paid` | kredit saldo manual (idempoten) |
| `POST /admin/topup/{id}/sync` | tanya status ke PayPan, kredit kalau lunas |

## 14. Aturan uang yang tidak boleh dilanggar

- Nominal integer rupiah, bukan float.
- Top-up minimal **Rp25.000**.
- Saldo **tidak dapat diuangkan** atau ditarik tunai. Hanya untuk transaksi di layanan Jhopan.
- Saldo tidak pernah negatif.
- Hold mengurangi `available`, bukan `balance`.
- Capture hanya sekali per hold.
- Setiap perubahan saldo punya baris `wallet_ledger` dengan `product`, `reference_id`, dan `idempotency_key` unik.
- Webhook PayPan satu-satunya jalur kredit top-up QRIS.
- Hold kedaluwarsa tidak bisa di-capture.

---

## 15. Contoh lengkap (curl)

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

## 16. Status implementasi

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
| Rekonsiliasi otomatis | selesai |
| ID pelanggan pendek (public_id) | selesai |
| Cek saldo & top-up lewat ID publik | selesai |
| Identitas Telegram (primer, auto-daftar) | selesai |
| Ikat email/WhatsApp lewat kode Telegram | selesai |

---

## 17. Penyesuaian saldo manual

Admin bisa menambah atau mengurangi saldo pelanggan tanpa lewat PayPan.
Dipakai untuk bonus, koreksi selisih, atau kompensasi.

```text
POST /admin/wallets/adjust
wallet_user_id=<id>
amount=<positif tambah / negatif kurangi>
reason=<alasan wajib>
```

Efek:
- Saldo berubah, tercatat di `wallet_ledger` sebagai `adjustment`.
- Masuk `wallet_admin_audit` dengan aksi `wallet.adjust`.
- Ditolak bila hasilnya saldo negatif.
- Idempoten bila `reference_id` sama.

---

## 18. Audit log

Semua tindakan admin yang mengubah data tercatat:

| Aksi | Kapan |
|---|---|
| `wallet.adjust` | penyesuaian saldo manual |
| `wallet.block` | blokir pelanggan |

Halaman: `/admin/audit` — menampilkan waktu, pelaku, aksi, wallet, referensi, detail.

Tujuannya: kalau ada selisih saldo, bisa dilacak siapa melakukan apa dan kapan.

---

## 19. Backup

SQLite disimpan sebagai satu berkas, jadi backup = salin berkas.

- Otomatis **setiap jam**, disimpan di folder `backup/` (default).
- Menyimpan **24 berkas terbaru**, sisanya dihapus otomatis.
- Backup pertama dibuat **saat service start**.
- Tombol manual: `/admin/backup` → "Buat backup sekarang".
- Ubah folder lewat env `PANWALLET_BACKUP_DIR`.

Untuk PostgreSQL, gunakan `pg_dump` bawaan — backup berkas tidak berlaku.

---

## 20. Health check

`GET /healthz` memeriksa tiga komponen:

```json
{
  "status": "ok",
  "components": [
    {"name": "database", "ok": true, "message": "ok"},
    {"name": "website", "ok": true, "message": "ok"},
    {"name": "paypan", "ok": false, "message": "belum dikonfigurasi"}
  ]
}
```

- `200` bila database dan website sehat.
- `503` bila database mati atau template gagal termuat.
- `paypan` hanya informasi, tidak memengaruhi status.

---

## 21. Graceful shutdown

Saat menerima `SIGINT`/`SIGTERM`, service:

1. Berhenti menerima koneksi baru.
2. Menyelesaikan permintaan yang sedang berjalan (maks 15 detik).
3. Menutup database dengan rapi.

Aman untuk `systemctl restart` tanpa merusak transaksi berjalan.

---

## 22. Laporan pendapatan

Halaman `/admin/laporan` menampilkan:

- Pendapatan hari ini + jumlah transaksi
- Total pendapatan (hanya invoice `paid`)
- Jumlah invoice per status: lunas / tertunda / kedaluwarsa
- Rekap harian 30 hari terakhir

Pendapatan dihitung dari `total` invoice yang **lunas**, bukan dari saldo tersimpan.
