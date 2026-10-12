# Pan-Wallet

Wallet pusat untuk saldo Jhopan lintas produk: AgenPulsa, AutoBuy, Tembak Paket XL, Xray Premium, dan produk berikutnya. Produk menjadi client API; hanya Pan-Wallet yang mengelola saldo, QRIS PayPan, hold, capture, release, refund, dan ledger.

## Status

Foundation only. Belum ada database cloud, PayPan webhook, atau saldo produksi.

## Design

- Supabase PostgreSQL: sumber data wallet.
- Wallet API: satu-satunya penulis wallet ledger.
- PayPan webhook: hanya ke Wallet API.
- Product client: membuat hold lalu capture/release berdasarkan hasil order.
- Identity utama: `wallet_user` UUID; Telegram, WhatsApp, email terhubung melalui `identity_links`.

## Konfigurasi infrastructure

Database dipilih sebelum service berjalan. Dashboard tidak menyimpan URL database atau credential.

- `PAN_WALLET_DATABASE_URL`: URL PostgreSQL. Supabase sekarang, PostgreSQL terkelola lain nanti.
- `PAN_WALLET_ADDR`: bind HTTP, default `:8080`.
- Salin `.env.example` menjadi `.env`, isi URL lokal, lalu muat environment lewat service manager.

## Jalan lokal

Prasyarat: PostgreSQL lokal dan Go 1.26.

```bash
# 1. siapkan database
createdb pan_wallet            # atau pakai Supabase
cp .env.example .env           # isi PAN_WALLET_DATABASE_URL

# 2. build
go build -o pan-wallet.exe ./cmd/pan-wallet

# 3. jalankan (Windows PowerShell)
powershell -ExecutionPolicy Bypass -File scripts/start-local.ps1
```

Buka `http://127.0.0.1:8080` untuk dashboard saldo, `GET /healthz` untuk cek service.

Buat product API key (hash SHA-256 saja yang disimpan):

```bash
# ganti TOKEN dengan key pilihanmu
TOKEN=pw_live_contoh
HASH=$(printf '%s' "$TOKEN" | sha256sum | cut -d' ' -f1)
psql "$PAN_WALLET_DATABASE_URL" -c \
  "insert into product_api_keys (product, token_hash) values ('agenpulsa', decode('$HASH','hex'));"
```

## Local checks

```bash
go test ./...
go build ./cmd/pan-wallet
python -m unittest discover -s tests -p "test_*.py" -v
python -m py_compile wallet/*.py tests/*.py
```

## Layout

```text
migrations/       PostgreSQL schema for Supabase
wallet/           domain models and later API implementation
tests/            wallet rules
docs/             API and integration contracts
```
