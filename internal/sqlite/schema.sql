-- Skema PanWallet untuk SQLite (mode lokal / single-server).
-- Padanan PostgreSQL, tanpa uuid/jsonb/timestamptz: id TEXT (hex acak),
-- waktu INTEGER unix, metadata JSON disimpan sebagai TEXT.

-- public_id: ID pendek 8 huruf besar yang dipakai pelanggan di semua channel
-- (Telegram, WhatsApp, web). Unik dan mudah dibaca/disalin.
create table if not exists wallet_users (
  id text primary key,
  public_id text not null unique,
  status text not null default 'active' check (status in ('active','blocked')),
  created_at integer not null
);

create table if not exists identity_links (
  id text primary key,
  wallet_user_id text not null references wallet_users(id) on delete cascade,
  provider text not null check (provider in ('telegram','whatsapp','email','web')),
  external_id text not null,
  verified_at integer not null,
  created_at integer not null,
  unique (provider, external_id)
);

-- Kode sementara untuk mengikat channel baru (email/WA) ke wallet Telegram.
-- Pelanggan memasukkan kode ini di bot Telegram untuk konfirmasi.
create table if not exists identity_link_codes (
  code text primary key,
  provider text not null check (provider in ('telegram','whatsapp','email','web')),
  external_id text not null,
  expires_at integer not null,
  used_at integer,
  created_at integer not null
);

create table if not exists wallets (
  id text primary key,
  wallet_user_id text not null unique references wallet_users(id) on delete cascade,
  balance integer not null default 0 check (balance >= 0),
  currency text not null default 'IDR',
  updated_at integer not null
);

create table if not exists wallet_ledger (
  id text primary key,
  wallet_id text not null references wallets(id),
  amount integer not null check (amount <> 0),
  balance_after integer not null check (balance_after >= 0),
  entry_type text not null check (entry_type in ('topup','hold_capture','refund','adjustment')),
  product text not null,
  reference_id text not null,
  idempotency_key text not null unique,
  created_at integer not null
);

create table if not exists wallet_holds (
  id text primary key,
  wallet_id text not null references wallets(id),
  amount integer not null check (amount > 0),
  product text not null,
  reference_id text not null,
  status text not null default 'active' check (status in ('active','captured','released','expired')),
  expires_at integer not null,
  idempotency_key text not null unique,
  created_at integer not null,
  unique (product, reference_id)
);

create table if not exists wallet_hold_transitions (
  id text primary key,
  hold_id text not null references wallet_holds(id),
  action text not null check (action in ('capture','release')),
  idempotency_key text not null unique,
  created_at integer not null,
  unique (hold_id, action)
);

create table if not exists wallet_invoices (
  id text primary key,
  wallet_id text not null references wallets(id),
  amount integer not null check (amount > 0),
  product text not null,
  paypan_invoice_id text unique,
  status text not null default 'pending' check (status in ('pending','paid','expired','cancelled','refunded')),
  expires_at integer not null,
  created_at integer not null
);

create table if not exists product_api_keys (
  id text primary key,
  product text not null unique,
  token_hash blob not null unique,
  active integer not null default 1,
  created_at integer not null,
  rotated_at integer not null
);

create table if not exists wallet_admin_audit (
  id text primary key,
  actor text not null default 'admin',
  action text not null,
  wallet_user_id text references wallet_users(id),
  reference_id text not null,
  details text not null default '{}',
  created_at integer not null
);

create table if not exists app_settings (
  key text primary key,
  value text not null default '',
  updated_at integer not null
);

-- Invoice top-up: id lokal + referensi invoice PayPan (gateway QRIS eksternal).
create table if not exists topup_invoices (
  id text primary key,
  wallet_user_id text not null references wallet_users(id),
  paypan_invoice_id text unique,
  price integer not null check (price >= 1000),
  code integer not null default 0,
  total integer not null check (total > 0),
  status text not null default 'pending' check (status in ('pending','paid','expired','cancelled','refunded')),
  source text not null default '',
  expires_at integer not null,
  paid_at integer,
  created_at integer not null
);

create index if not exists topup_invoices_paypan_idx on topup_invoices(paypan_invoice_id);

create table if not exists payment_notifications (
  id text primary key,
  amount integer,
  source text not null default '',
  raw text not null default '',
  matched_invoice text,
  reason text not null default '',
  received_at integer not null
);

create index if not exists wallet_ledger_wallet_created_idx on wallet_ledger(wallet_id, created_at desc);
create index if not exists wallet_holds_wallet_status_idx on wallet_holds(wallet_id, status);
create index if not exists identity_links_wallet_user_idx on identity_links(wallet_user_id);
create index if not exists topup_invoices_wallet_idx on topup_invoices(wallet_user_id, created_at desc);
