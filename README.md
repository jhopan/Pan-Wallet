# PanWallet


**Dokumentasi:** [APIDOCS](docs/APIDOCS.md) (ringkas) · [API](docs/API.md) (lengkap) · [Integrasi bot lain](docs/INTEGRASI.md) · [Backup & pemulihan](docs/BACKUP.md)

**Bot Telegram:** [panwallet-telebot](https://github.com/jhopan/panwallet-telebot)

Wallet pusat untuk saldo Jhopan lintas produk: AgenPulsa, AutoBuy, Tembak Paket XL, Xray Premium, dan produk berikutnya. Produk menjadi client API; hanya PanWallet yang mengelola saldo, QRIS PayPan, hold, capture, release, refund, dan ledger.

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

- `PANWALLET_DB_DRIVER`: `sqlite` (default) atau `postgres`.
- `PANWALLET_DB_PATH`: path file SQLite, default `data/panwallet.db`.
- `PANWALLET_DATABASE_URL`: URL PostgreSQL, hanya bila driver `postgres`.
- `PANWALLET_ADDR`: bind HTTP, default `:8080`.
- `PANWALLET_ADMIN_USER` / `PANWALLET_ADMIN_PASS`: login panel (password minimal 8 karakter).
- Salin `.env.example` menjadi `.env`, isi nilainya.

## Jalan lokal

Prasyarat: Go 1.26. Tanpa install database apa pun (SQLite).

```bash
cp .env.example .env
go build -o panwallet.exe ./cmd/panwallet
go run scripts/seed-demo.go     # buat wallet demo + product key
powershell -ExecutionPolicy Bypass -File scripts/start-local.ps1
```

Buka `http://127.0.0.1:8080` lalu masuk panel di `/admin/login`.
`GET /healthz` untuk cek service.

Untuk PostgreSQL: set `PANWALLET_DB_DRIVER=postgres` dan `PANWALLET_DATABASE_URL`.

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
go build ./cmd/panwallet
```

## Layout

```text
cmd/panwallet/   entrypoint service
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
2. **Webhook** — isi URL, PanWallet akan POST JSON:

```json
{"event":"topup.paid","public_id":"65ZMPUNP","amount":50001,"balance":50001}
```

Tombol **Kirim pesan uji** tersedia untuk memastikan konfigurasi benar.

## Jalan di produksi dengan cloudflared

Tidak perlu membuka port ke internet. Cukup tunnel:

```bash
# 1. isi .env
PANWALLET_HOST=127.0.0.1
PANWALLET_PORT=8080
PANWALLET_PUBLIC_URL=https://wallet.domain-kamu.com
PANWALLET_TRUSTED_PROXIES=127.0.0.1,::1

# 2. jalankan PanWallet
./panwallet

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

### Kenapa PANWALLET_TRUSTED_PROXIES penting

Tanpa daftar ini, PanWallet tidak mempercayai header `X-Forwarded-For`
dari siapa pun. Itu **aman**, tapi semua pengunjung terlihat berasal dari
IP tunnel yang sama — rate limit jadi cepat habis untuk semua orang.

Diisi `127.0.0.1,::1` (alamat cloudflared), maka IP asli pengunjung dipakai
untuk rate limit. Header palsu dari internet tetap diabaikan.
