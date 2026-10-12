-- QRIS gateway settings (payload statis merchant + opsi) dan invoice top-up.
-- Payload disimpan di sini supaya admin bisa mengubahnya lewat panel kendali
-- tanpa restart, sama seperti PayPan.

create table if not exists app_settings (
  key text primary key,
  value text not null default '',
  updated_at timestamptz not null default now()
);

create table if not exists topup_invoices (
  id uuid primary key default gen_random_uuid(),
  wallet_user_id uuid not null references wallet_users(id),
  price bigint not null check (price >= 1000),
  code integer not null check (code between 1 and 999),
  total bigint not null check (total > 0),
  status text not null default 'pending' check (status in ('pending', 'paid', 'expired', 'cancelled')),
  source text not null default '',
  expires_at timestamptz not null,
  paid_at timestamptz,
  created_at timestamptz not null default now()
);

-- Satu total hanya boleh dimiliki satu invoice aktif: ini yang membuat nominal
-- menjadi identitas pembayaran yang tidak ambigu.
create unique index if not exists topup_invoices_pending_total_idx
  on topup_invoices(total) where status = 'pending';

create index if not exists topup_invoices_wallet_idx
  on topup_invoices(wallet_user_id, created_at desc);

-- Catatan notifikasi pembayaran mentah (relay HP atau input admin), audit trail.
create table if not exists payment_notifications (
  id text primary key,
  amount bigint,
  source text not null default '',
  raw text not null default '',
  matched_invoice uuid,
  reason text not null default '',
  received_at timestamptz not null default now()
);

alter table app_settings enable row level security;
alter table topup_invoices enable row level security;
alter table payment_notifications enable row level security;
