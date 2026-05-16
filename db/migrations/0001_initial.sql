-- +goose Up
create extension if not exists pgcrypto;

create table users (
  id bigserial primary key,
  username text not null unique,
  password_hash text not null,
  display_name text not null,
  role text not null check (role in ('admin', 'encoder', 'viewer')),
  active_branch_id bigint,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table branches (
  id bigserial primary key,
  code text not null unique,
  name text not null,
  incharge text,
  aps text,
  farm_customer boolean not null default false,
  remarks text,
  encoder_user_id bigint references users(id),
  last_update_by_user_id bigint references users(id),
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

alter table users add constraint users_active_branch_fk foreign key (active_branch_id) references branches(id);

create table stock_categories (
  id bigserial primary key,
  name text not null,
  group_name text not null,
  encoder_user_id bigint references users(id),
  last_update_by_user_id bigint references users(id),
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  unique (name, group_name)
);

create table suppliers (
  id bigserial primary key,
  code text not null unique,
  company text,
  lastname text,
  firstname text,
  middlename text,
  phone_number text,
  address text,
  balance numeric(14,2) not null default 0,
  encoder_user_id bigint references users(id),
  last_update_by_user_id bigint references users(id),
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table customers (
  id bigserial primary key,
  code text not null unique,
  company text,
  lastname text,
  firstname text,
  middlename text,
  phone_number text,
  address text,
  balance numeric(14,2) not null default 0,
  credit_term text,
  credit_limit numeric(14,2) not null default 0,
  aps text,
  farm_customer boolean not null default false,
  encoder_user_id bigint references users(id),
  last_update_by_user_id bigint references users(id),
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table expense_charts (
  id bigserial primary key,
  code text not null unique,
  name text not null,
  description text,
  exclude_daily_sales boolean not null default false,
  daily_sales_only boolean not null default false,
  encoder_user_id bigint references users(id),
  last_update_by_user_id bigint references users(id),
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table other_income_charts (
  id bigserial primary key,
  code text not null unique,
  name text not null,
  description text,
  encoder_user_id bigint references users(id),
  last_update_by_user_id bigint references users(id),
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table stocks (
  id bigserial primary key,
  code text not null unique,
  name text not null,
  category_group text,
  unit text,
  description text,
  latest_cost numeric(14,2) not null default 0,
  min_inventory numeric(14,2) not null default 0,
  encoder_user_id bigint references users(id),
  last_update_by_user_id bigint references users(id),
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table documents (
  id bigserial primary key,
  entry_id text not null unique default ('ENT-' || to_char(now(), 'YYYYMMDD') || '-' || lpad(nextval('documents_id_seq')::text, 6, '0')),
  kind text not null,
  entry_date timestamptz not null,
  branch_id bigint references branches(id),
  party_type text check (party_type in ('supplier', 'customer')),
  party_id bigint,
  reference text,
  cash boolean not null default false,
  remarks text,
  total numeric(14,2) not null default 0,
  less_amount numeric(14,2) not null default 0,
  add_amount numeric(14,2) not null default 0,
  net numeric(14,2) not null default 0,
  balance numeric(14,2) not null default 0,
  payload jsonb not null default '{}',
  encoder_user_id bigint references users(id),
  last_update_by_user_id bigint references users(id),
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create index documents_kind_date_idx on documents (kind, entry_date desc);
create index documents_branch_idx on documents (branch_id);
create index documents_party_idx on documents (party_type, party_id);

create table document_lines (
  id bigserial primary key,
  document_id bigint not null references documents(id) on delete cascade,
  group_key text not null,
  line_no integer not null,
  stock_id bigint references stocks(id),
  code_id bigint,
  qty numeric(14,2) not null default 0,
  unit_cost numeric(14,2) not null default 0,
  price numeric(14,2) not null default 0,
  cash_amount numeric(14,2) not null default 0,
  check_amount numeric(14,2) not null default 0,
  amount numeric(14,2) not null default 0,
  payload jsonb not null default '{}'
);

create index document_lines_document_idx on document_lines (document_id);
create index document_lines_stock_idx on document_lines (stock_id);

create table stock_ledger (
  id bigserial primary key,
  document_id bigint not null references documents(id) on delete cascade,
  branch_id bigint references branches(id),
  stock_id bigint not null references stocks(id),
  qty_delta numeric(14,2) not null,
  unit_cost numeric(14,2) not null default 0,
  created_at timestamptz not null default now()
);

create index stock_ledger_stock_branch_idx on stock_ledger (stock_id, branch_id);

create table balance_ledger (
  id bigserial primary key,
  document_id bigint not null references documents(id) on delete cascade,
  party_type text not null check (party_type in ('supplier', 'customer')),
  party_id bigint not null,
  amount_delta numeric(14,2) not null,
  created_at timestamptz not null default now()
);

create index balance_ledger_party_idx on balance_ledger (party_type, party_id);

create view stock_on_hand as
select branch_id, stock_id, sum(qty_delta) as qty_on_hand
from stock_ledger
group by branch_id, stock_id;

-- +goose Down
drop view if exists stock_on_hand;
drop table if exists balance_ledger;
drop table if exists stock_ledger;
drop table if exists document_lines;
drop table if exists documents;
drop table if exists stocks;
drop table if exists other_income_charts;
drop table if exists expense_charts;
drop table if exists customers;
drop table if exists suppliers;
drop table if exists stock_categories;
alter table if exists users drop constraint if exists users_active_branch_fk;
drop table if exists branches;
drop table if exists users;
