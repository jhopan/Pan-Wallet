create table if not exists product_api_keys (
  id uuid primary key default gen_random_uuid(),
  product text not null unique,
  token_hash bytea not null unique,
  active boolean not null default true,
  created_at timestamptz not null default now(),
  rotated_at timestamptz not null default now()
);

create table if not exists wallet_admin_audit (
  id uuid primary key default gen_random_uuid(),
  actor text not null default 'admin',
  action text not null,
  wallet_user_id uuid references wallet_users(id),
  reference_id text not null,
  details jsonb not null default '{}'::jsonb,
  created_at timestamptz not null default now()
);

create index if not exists wallet_admin_audit_user_created_idx
  on wallet_admin_audit(wallet_user_id, created_at desc);

create table if not exists identity_link_codes (
  code text primary key,
  provider text not null,
  external_id text not null,
  expires_at timestamptz not null,
  used_at timestamptz,
  created_at timestamptz not null default now()
);
