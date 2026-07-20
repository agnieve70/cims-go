-- +goose Up
alter table suppliers alter column balance type numeric(18,3);
alter table customers alter column balance type numeric(18,3);
alter table customers alter column credit_limit type numeric(18,3);
alter table stocks alter column latest_cost type numeric(18,3);
alter table stocks alter column min_inventory type numeric(18,3);

alter table documents alter column total type numeric(18,3);
alter table documents alter column less_amount type numeric(18,3);
alter table documents alter column add_amount type numeric(18,3);
alter table documents alter column net type numeric(18,3);
alter table documents alter column balance type numeric(18,3);

alter table document_lines alter column qty type numeric(18,3);
alter table document_lines alter column unit_cost type numeric(18,3);
alter table document_lines alter column price type numeric(18,3);
alter table document_lines alter column cash_amount type numeric(18,3);
alter table document_lines alter column check_amount type numeric(18,3);
alter table document_lines alter column amount type numeric(18,3);

alter table stock_ledger alter column qty_delta type numeric(18,3);
alter table stock_ledger alter column unit_cost type numeric(18,3);
alter table balance_ledger alter column amount_delta type numeric(18,3);
alter table dr_consumptions alter column consumed_qty type numeric(18,3);

-- +goose Down
alter table suppliers alter column balance type numeric(14,2) using round(balance, 2);
alter table customers alter column balance type numeric(14,2) using round(balance, 2);
alter table customers alter column credit_limit type numeric(14,2) using round(credit_limit, 2);
alter table stocks alter column latest_cost type numeric(14,2) using round(latest_cost, 2);
alter table stocks alter column min_inventory type numeric(14,2) using round(min_inventory, 2);

alter table documents alter column total type numeric(14,2) using round(total, 2);
alter table documents alter column less_amount type numeric(14,2) using round(less_amount, 2);
alter table documents alter column add_amount type numeric(14,2) using round(add_amount, 2);
alter table documents alter column net type numeric(14,2) using round(net, 2);
alter table documents alter column balance type numeric(14,2) using round(balance, 2);

alter table document_lines alter column qty type numeric(14,2) using round(qty, 2);
alter table document_lines alter column unit_cost type numeric(14,2) using round(unit_cost, 2);
alter table document_lines alter column price type numeric(14,2) using round(price, 2);
alter table document_lines alter column cash_amount type numeric(14,2) using round(cash_amount, 2);
alter table document_lines alter column check_amount type numeric(14,2) using round(check_amount, 2);
alter table document_lines alter column amount type numeric(14,2) using round(amount, 2);

alter table stock_ledger alter column qty_delta type numeric(14,2) using round(qty_delta, 2);
alter table stock_ledger alter column unit_cost type numeric(14,2) using round(unit_cost, 2);
alter table balance_ledger alter column amount_delta type numeric(14,2) using round(amount_delta, 2);
alter table dr_consumptions alter column consumed_qty type numeric(14,2) using round(consumed_qty, 2);
