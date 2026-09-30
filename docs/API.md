# Pan-Wallet API

Status: product hold endpoint tersedia. Admin, customer, top-up PayPan, dan website sedang dibangun.

## Product money flow

Produk tidak pernah mengubah tabel saldo langsung.

1. Product membuat hold untuk `wallet_user_id`, `product`, `reference_id`, `amount`, dan `idempotency_key`.
2. Setelah order sukses, product capture hold. Capture mengurangi saldo dan membuat satu ledger immutable.
3. Setelah order gagal atau dibatalkan, product release hold. Release tidak mengubah saldo.
4. Retry harus memakai idempotency key sama.

## Product authentication

Setiap product memakai API key berbeda. Backend hanya menyimpan SHA-256 hash key. API key tidak pernah diberikan ke browser dan tidak memberi akses Supabase.

## Endpoint product

### `POST /v1/holds`

Header wajib:

```text
Authorization: Bearer <product-api-key>
Idempotency-Key: hold:<product>:<reference-id>
Content-Type: application/json
```

Body:

```json
{
  "wallet_user_id": "UUID",
  "reference_id": "AP-1001",
  "amount": 25000,
  "expires_at": "2030-01-01T00:00:00Z"
}
```

`product` tidak diterima dari body. Backend mengambilnya dari product API key. Body maksimal 1 MiB. Sukses memberi `201 Created` dan hold aktif.

## Money invariants

- Nominal adalah integer IDR, bukan float.
- Saldo tidak pernah negatif.
- Hold aktif mengurangi saldo tersedia, bukan balance tersimpan.
- Capture hanya boleh sekali.
- Semua perubahan balance memiliki `wallet_ledger` dengan `product`, `reference_id`, dan unique `idempotency_key`.
- Webhook PayPan menjadi satu-satunya kredit top-up QRIS.
