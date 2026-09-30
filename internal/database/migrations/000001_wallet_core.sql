-- Wallet core schema. Product clients never receive direct table write access.

create table if not exists wallet_users (
  id uuid primary key default gen_random_uuid(),
  status text not null default 'active' check (status in ('active', 'blocked')),
  created_at timestamptz not null default now()
);

create table if not exists identity_links (
  id uuid primary key default gen_random_uuid(),
  wallet_user_id uuid not null references wallet_users(id) on delete cascade,
  provider text not null check (provider in ('telegram', 'whatsapp', 'email', 'web')),
  external_id text not null,
  verified_at timestamptz not null default now(),
  created_at timestamptz not null default now(),
  unique (provider, external_id)
);

create table if not exists wallets (
  id uuid primary key default gen_random_uuid(),
  wallet_user_id uuid not null unique references wallet_users(id) on delete cascade,
  balance bigint not null default 0 check (balance >= 0),
  currency text not null default 'IDR',
  updated_at timestamptz not null default now()
);

create table if not exists wallet_ledger (
  id uuid primary key default gen_random_uuid(),
  wallet_id uuid not null references wallets(id),
  amount bigint not null check (amount <> 0),
  balance_after bigint not null check (balance_after >= 0),
  entry_type text not null check (entry_type in ('topup', 'hold_capture', 'refund', 'adjustment')),
  product text not null,
  reference_id text not null,
  idempotency_key text not null unique,
  created_at timestamptz not null default now()
);

create table if not exists wallet_holds (
  id uuid primary key default gen_random_uuid(),
  wallet_id uuid not null references wallets(id),
  amount bigint not null check (amount > 0),
  product text not null,
  reference_id text not null,
  status text not null default 'active' check (status in ('active', 'captured', 'released', 'expired')),
  expires_at timestamptz not null,
  idempotency_key text not null unique,
  created_at timestamptz not null default now(),
  unique (product, reference_id)
);

create table if not exists wallet_invoices (
  id uuid primary key default gen_random_uuid(),
  wallet_id uuid not null references wallets(id),
  amount bigint not null check (amount > 0),
  product text not null,
  paypan_invoice_id text unique,
  status text not null default 'pending' check (status in ('pending', 'paid', 'expired', 'cancelled')),
  expires_at timestamptz not null,
  created_at timestamptz not null default now()
);

create index if not exists wallet_ledger_wallet_created_idx on wallet_ledger(wallet_id, created_at desc);
create index if not exists wallet_holds_wallet_status_idx on wallet_holds(wallet_id, status);
create index if not exists identity_links_wallet_user_idx on identity_links(wallet_user_id);
