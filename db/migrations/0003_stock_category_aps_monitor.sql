-- +goose Up
alter table stock_categories
  add column if not exists aps_monitor boolean not null default false;

-- +goose Down
alter table if exists stock_categories
  drop column if exists aps_monitor;
