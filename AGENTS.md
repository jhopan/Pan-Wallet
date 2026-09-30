# AGENTS.md

Pan-Wallet is the central wallet service for Jhopan products. It owns users, linked identities, wallet balances, immutable ledger entries, PayPan QRIS invoices, holds, capture/release/refund. Product repos never modify balances directly.

## Scope

In scope: this repository, its Supabase schema, Wallet API, admin/user wallet website, and payment-webhook logic.

Out of scope: business logic inside Tembak Paket, AutoBuy, AgenPulsa, or Xray Premium. Report integration requirements there; do not edit sibling repos from this workspace.

## Runtime

- Python 3.11+ for current foundation checks.
- Target database: Supabase PostgreSQL. Do not add SQLite as a production wallet backend.
- Local commands:
  - `python -m unittest discover -s tests -p "test_*.py" -v`
  - `python -m py_compile wallet/*.py tests/*.py`

## Architecture rules

- `wallet_user` UUID owns money. Telegram IDs, WhatsApp numbers, emails, and product accounts link through `identity_links`.
- Every balance change creates an immutable `wallet_ledger` row with `product`, `reference_id`, and unique `idempotency_key`.
- Products use hold → capture or release. Never debit a wallet directly from a product repo.
- PayPan webhook is the only path that credits QRIS top-ups. Duplicate webhooks must be idempotent.
- A product API token is restricted to its own `product` key and never receives Supabase service-role access.
- Keep MyXL refresh tokens outside this service unless a future shared session service is explicitly added.

## Security

- `.env` holds only infrastructure secrets: database connection, backend encryption key, service credentials, local port.
- Never put Supabase service-role keys, PayPan credentials, tokens, or real user data in source, migrations, tests, docs, or commits.
- Website/browser code never receives a service-role key.

## Git

- Do not commit generated databases, `.env`, or payment/test data.
- Do not commit or push unless the user asks.
