# Pan-Wallet API

Status: backend core sedang dibangun. Endpoint publik belum dibuka.

## Product money flow

Produk tidak pernah mengubah tabel saldo langsung.

1. Product membuat hold untuk `wallet_user_id`, `product`, `reference_id`, `amount`, dan `idempotency_key`.
2. Setelah order sukses, product capture hold. Capture mengurangi saldo dan membuat satu ledger immutable.
3. Setelah order gagal atau dibatalkan, product release hold. Release tidak mengubah saldo.
4. Retry harus memakai idempotency key sama.

## Product authentication

Setiap product memakai API key berbeda. Backend hanya menyimpan SHA-256 hash key. API key tidak pernah diberikan ke browser dan tidak memberi akses Supabase.

## Money invariants

- Nominal adalah integer IDR, bukan float.
- Saldo tidak pernah negatif.
- Hold aktif mengurangi saldo tersedia, bukan balance tersimpan.
- Capture hanya boleh sekali.
- Semua perubahan balance memiliki `wallet_ledger` dengan `product`, `reference_id`, dan unique `idempotency_key`.
- Webhook PayPan menjadi satu-satunya kredit top-up QRIS.
