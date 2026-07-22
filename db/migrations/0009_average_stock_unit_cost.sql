-- +goose Up
-- Repeated purchases at an unchanged price do not add another value to the average.
with ordered_costs as (
  select dl.stock_id,
         dl.unit_cost,
         lag(dl.unit_cost) over (
           partition by dl.stock_id
           order by d.document_date, d.entry_date, d.id, dl.line_no, dl.id
         ) as previous_unit_cost
  from document_lines dl
  join documents d on d.id = dl.document_id
  where d.kind in ('purchases', 'stock-in')
    and dl.group_key = 'details'
    and dl.stock_id is not null
    and dl.unit_cost > 0
),
average_costs as (
  select stock_id,
         avg(unit_cost) as average_unit_cost
  from ordered_costs
  where unit_cost is distinct from previous_unit_cost
  group by stock_id
)
update stocks st
set latest_cost = average_costs.average_unit_cost,
    updated_at = now()
from average_costs
where st.id = average_costs.stock_id;

-- +goose Down
update stocks st
set latest_cost = latest.unit_cost,
    updated_at = now()
from (
  select distinct on (dl.stock_id)
    dl.stock_id,
    dl.unit_cost
  from document_lines dl
  join documents d on d.id = dl.document_id
  where d.kind in ('purchases', 'stock-in')
    and dl.group_key = 'details'
    and dl.stock_id is not null
    and dl.unit_cost > 0
  order by dl.stock_id, d.document_date desc, d.entry_date desc, d.id desc, dl.line_no desc, dl.id desc
) latest
where st.id = latest.stock_id;
