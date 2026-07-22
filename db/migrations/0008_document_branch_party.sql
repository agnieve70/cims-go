-- +goose Up
alter table documents
  drop constraint if exists documents_party_type_check;

alter table documents
  add constraint documents_party_type_check
  check (party_type in ('supplier', 'customer', 'branch'));

-- +goose Down
update documents
set party_type = null,
    party_id = null
where party_type = 'branch';

alter table documents
  drop constraint if exists documents_party_type_check;

alter table documents
  add constraint documents_party_type_check
  check (party_type in ('supplier', 'customer'));
