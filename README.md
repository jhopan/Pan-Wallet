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
