-- +goose Up
update stocks st
set latest_cost = latest.unit_cost,
    updated_at = now()
from (
  select distinct on (dl.stock_id)
    dl.stock_id,
    dl.unit_cost
  from document_lines dl
  join documents d on d.id = dl.document_id
  where d.kind = 'purchases'
    and dl.stock_id is not null
  order by dl.stock_id, d.document_date desc, d.entry_date desc, d.id desc, dl.line_no desc
) latest
where st.id = latest.stock_id;

-- +goose Down
-- latest_cost is derived from purchase history; previous values cannot be restored safely.
