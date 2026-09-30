create table if not exists wallet_hold_transitions (
  id uuid primary key default gen_random_uuid(),
  hold_id uuid not null references wallet_holds(id),
  action text not null check (action in ('capture', 'release')),
  idempotency_key text not null unique,
  created_at timestamptz not null default now(),
  unique (hold_id, action)
);

alter table wallet_users enable row level security;
alter table identity_links enable row level security;
alter table wallets enable row level security;
alter table wallet_ledger enable row level security;
alter table wallet_holds enable row level security;
alter table wallet_invoices enable row level security;
alter table product_api_keys enable row level security;
alter table wallet_admin_audit enable row level security;
alter table wallet_hold_transitions enable row level security;
