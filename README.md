# Pan-Wallet

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
