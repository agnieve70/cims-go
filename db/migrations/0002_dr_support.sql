-- +goose Up
alter table documents
  add column if not exists dr_reference_id bigint references documents(id);

alter table document_lines
  add column if not exists dr_line_id bigint references document_lines(id);

create index if not exists documents_dr_reference_idx on documents (dr_reference_id);
create index if not exists document_lines_dr_line_idx on document_lines (dr_line_id);

create table if not exists dr_consumptions (
  id bigserial primary key,
  dr_document_id bigint not null references documents(id) on delete cascade,
  dr_line_id bigint not null references document_lines(id) on delete cascade,
  consumer_document_id bigint not null references documents(id) on delete cascade,
  consumer_line_id bigint not null references document_lines(id) on delete cascade,
  consumed_qty numeric(14,2) not null default 0,
  created_at timestamptz not null default now()
);

create index if not exists dr_consumptions_document_idx on dr_consumptions (dr_document_id);
create index if not exists dr_consumptions_line_idx on dr_consumptions (dr_line_id);
create index if not exists dr_consumptions_consumer_idx on dr_consumptions (consumer_document_id);

-- +goose Down
drop table if exists dr_consumptions;
drop index if exists document_lines_dr_line_idx;
drop index if exists documents_dr_reference_idx;
alter table if exists document_lines drop column if exists dr_line_id;
alter table if exists documents drop column if exists dr_reference_id;
