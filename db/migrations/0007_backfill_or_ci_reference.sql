-- +goose Up
update documents
set reference = nullif(trim(payload->'values'->>'or_ci_number'), '')
where kind in ('sales', 'purchases')
  and nullif(trim(reference), '') is null
  and nullif(trim(payload->'values'->>'or_ci_number'), '') is not null;

-- +goose Down
-- The previous reference values were blank and cannot be distinguished safely after backfill.
