-- +goose Up
alter table documents
  add column if not exists document_date date;

update documents
set document_date = case
  when kind = 'purchases' then coalesce(nullif(payload->'values'->>'purchase_date', '')::date, timezone('Asia/Manila', entry_date)::date)
  when kind in ('sales', 'dr') then coalesce(nullif(payload->'values'->>'sales_date', '')::date, timezone('Asia/Manila', entry_date)::date)
  when kind = 'stock-transactions' then coalesce(nullif(payload->'values'->>'transfer_date', '')::date, timezone('Asia/Manila', entry_date)::date)
  else timezone('Asia/Manila', entry_date)::date
end
where document_date is null;

alter table documents
  alter column document_date set not null;

create index if not exists branches_updated_at_idx on branches (updated_at);
create index if not exists customers_updated_at_idx on customers (updated_at);
create index if not exists expense_charts_updated_at_idx on expense_charts (updated_at);
create index if not exists other_income_charts_updated_at_idx on other_income_charts (updated_at);
create index if not exists stock_categories_updated_at_idx on stock_categories (updated_at);
create index if not exists stocks_updated_at_idx on stocks (updated_at);
create index if not exists suppliers_updated_at_idx on suppliers (updated_at);

create index if not exists documents_kind_entry_date_id_idx on documents (kind, entry_date desc, id desc);
create index if not exists documents_kind_document_date_id_idx on documents (kind, document_date desc, id desc);
create index if not exists documents_kind_id_idx on documents (kind, id desc);
create index if not exists document_lines_document_group_idx on document_lines (document_id, group_key);
create index if not exists document_lines_group_document_idx on document_lines (group_key, document_id);
create index if not exists stock_ledger_document_idx on stock_ledger (document_id);
create index if not exists balance_ledger_document_idx on balance_ledger (document_id);

-- +goose Down
drop index if exists balance_ledger_document_idx;
drop index if exists stock_ledger_document_idx;
drop index if exists document_lines_group_document_idx;
drop index if exists document_lines_document_group_idx;
drop index if exists documents_kind_id_idx;
drop index if exists documents_kind_document_date_id_idx;
drop index if exists documents_kind_entry_date_id_idx;
drop index if exists suppliers_updated_at_idx;
drop index if exists stocks_updated_at_idx;
drop index if exists stock_categories_updated_at_idx;
drop index if exists other_income_charts_updated_at_idx;
drop index if exists expense_charts_updated_at_idx;
drop index if exists customers_updated_at_idx;
drop index if exists branches_updated_at_idx;
alter table if exists documents
  drop column if exists document_date;
