# Pan-Wallet


**Dokumentasi:** [API](docs/API.md) · [Panduan integrasi bot lain](docs/INTEGRASI.md)

**Bot Telegram:** [pan-wallet-telebot](https://github.com/jhopan/pan-wallet-telebot)

Wallet pusat untuk saldo Jhopan lintas produk: AgenPulsa, AutoBuy, Tembak Paket XL, Xray Premium, dan produk berikutnya. Produk menjadi client API; hanya Pan-Wallet yang mengelola saldo, QRIS PayPan, hold, capture, release, refund, dan ledger.

## Status

Berjalan. SQLite (default) atau PostgreSQL. Gateway QRIS, panel kendali, dan API produk sudah aktif.

## Design

- SQLite (default, satu file) atau PostgreSQL/Supabase.
- Wallet API: satu-satunya penulis wallet ledger.
- Gateway QRIS bawaan: payload statis jadi dinamis, nominal unik per invoice.
- Product client: membuat hold lalu capture/release berdasarkan hasil order.
- Identity utama: `wallet_user`; Telegram, WhatsApp, email terhubung melalui `identity_links`.

## Konfigurasi infrastructure

Database dipilih sebelum service berjalan. Panel tidak menyimpan URL database atau credential.

- `PAN_WALLET_DB_DRIVER`: `sqlite` (default) atau `postgres`.
- `PAN_WALLET_DB_PATH`: path file SQLite, default `data/pan-wallet.db`.
- `PAN_WALLET_DATABASE_URL`: URL PostgreSQL, hanya bila driver `postgres`.
- `PAN_WALLET_ADDR`: bind HTTP, default `:8080`.
- `PAN_WALLET_ADMIN_USER` / `PAN_WALLET_ADMIN_PASS`: login panel (password minimal 8 karakter).
- Salin `.env.example` menjadi `.env`, isi nilainya.

## Jalan lokal

Prasyarat: Go 1.26. Tanpa install database apa pun (SQLite).

```bash
cp .env.example .env
go build -o pan-wallet.exe ./cmd/pan-wallet
go run scripts/seed-demo.go     # buat wallet demo + product key
powershell -ExecutionPolicy Bypass -File scripts/start-local.ps1
```

Buka `http://127.0.0.1:8080` lalu masuk panel di `/admin/login`.
`GET /healthz` untuk cek service.

Untuk PostgreSQL: set `PAN_WALLET_DB_DRIVER=postgres` dan `PAN_WALLET_DATABASE_URL`.

## API produk

```bash
curl -X POST http://127.0.0.1:8080/v1/holds \
  -H "Authorization: Bearer <api_key>" \
  -H "Idempotency-Key: hold:AP-1001" \
  -H "Content-Type: application/json" \
  -d '{"wallet_user_id":"<uuid>","reference_id":"AP-1001","amount":25000,"expires_at":"2030-01-01T00:00:00Z"}'
```

Lihat `docs/API.md` untuk kontrak lengkap.

## Local checks

```bash
go test ./...
go build ./cmd/pan-wallet
```

## Layout

```text
cmd/pan-wallet/   entrypoint service
internal/sqlite/  driver SQLite + skema
internal/database/ driver PostgreSQL + migrasi
internal/httpapi/ API, panel kendali, checkout
internal/qris/    payload QRIS statis -> dinamis
internal/topup/   alokasi kode unik invoice
internal/wallet/  aturan hold/capture/release
docs/             kontrak API
```

## Notifikasi

Panel → **Konfigurasi → Notifikasi** bisa mengirim kabar saat ada top-up masuk.

Dua jalur, keduanya HTTP biasa jadi gampang dicampur dengan sistem lain:

1. **Telegram** — isi bot token + chat ID admin (boleh beberapa, pisah koma).
2. **Webhook** — isi URL, Pan-Wallet akan POST JSON:

```json
{"event":"topup.paid","public_id":"65ZMPUNP","amount":50001,"balance":50001}
```

Tombol **Kirim pesan uji** tersedia untuk memastikan konfigurasi benar.

## Jalan di produksi dengan cloudflared

Tidak perlu membuka port ke internet. Cukup tunnel:

```bash
# 1. isi .env
PAN_WALLET_HOST=127.0.0.1
PAN_WALLET_PORT=8080
PAN_WALLET_PUBLIC_URL=https://wallet.domain-kamu.com
PAN_WALLET_TRUSTED_PROXIES=127.0.0.1,::1

# 2. jalankan Pan-Wallet
./pan-wallet

# 3. jalankan tunnel
cloudflared tunnel --url http://127.0.0.1:8080
```

Dengan konfigurasi tetap (`~/.cloudflared/config.yml`):

```yaml
tunnel: <TUNNEL-ID>
credentials-file: /path/<TUNNEL-ID>.json
ingress:
  - hostname: wallet.domain-kamu.com
    service: http://127.0.0.1:8080
  - service: http_status:404
```

```bash
cloudflared tunnel run <NAMA-TUNNEL>
```

### Kenapa PAN_WALLET_TRUSTED_PROXIES penting

Tanpa daftar ini, Pan-Wallet tidak mempercayai header `X-Forwarded-For`
dari siapa pun. Itu **aman**, tapi semua pengunjung terlihat berasal dari
IP tunnel yang sama — rate limit jadi cepat habis untuk semua orang.

Diisi `127.0.0.1,::1` (alamat cloudflared), maka IP asli pengunjung dipakai
untuk rate limit. Header palsu dari internet tetap diabaikan.
