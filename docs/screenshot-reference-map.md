# Screenshot Reference Map

Reference folder: `/Users/gabril/Downloads/cims ss`

## Files

| Reference screenshots | App route |
| --- | --- |
| `Files/Purchase File/*.PNG` | `/transactions/purchases/` |
| `Files/Sales File/*.PNG` | `/transactions/sales/` |
| `Files/Stock Transfer/*.PNG` | `/transactions/stock-transactions/` |
| `Files/Purchase File/ap credit file.PNG` | `/transactions/ap-credit/` |
| `Files/Purchase File/ap debit file.PNG` | `/transactions/ap-debit/` |

## Menubar Reports

| Reference screenshots | App route |
| --- | --- |
| `MENUBAR REPORTS/(menu 1) Stock Summary/stock aging/*.PNG` | `/reports/stock-aging` |
| `MENUBAR REPORTS/(menu 1) Stock Summary/stock ledger/*.PNG` | `/reports/stock-ledger` |
| `MENUBAR REPORTS/(menu 1) Stock Summary/stock reorder point/*.PNG` | `/reports/stock-reorder-point` |
| `MENUBAR REPORTS/(menu 1) Stock Summary/stock summary/*.PNG` | `/reports/stock-summary` |
| `MENUBAR REPORTS/(menu 2) Stock Purchases/by DR Number/*.PNG` | `/reports/purchases-by-dr-number` |
| `MENUBAR REPORTS/(menu 2) Stock Purchases/by Stock Code/*.PNG` | `/reports/purchases-by-stock-code` |
| `MENUBAR REPORTS/(menu 2) Stock Purchases/by Supplier/*.PNG` | `/reports/purchases-by-supplier` |
| `MENUBAR REPORTS/(menu 3) Stock Sales/1 By Customer/*.PNG` | `/reports/sales-by-customer` |
| `MENUBAR REPORTS/(menu 3) Stock Sales/2 By Stock Name/*.PNG` | `/reports/sales-by-stock-name` |
| `MENUBAR REPORTS/(menu 3) Stock Sales/3 By OR-CI-DR Number/*.PNG` | `/reports/sales-by-or-ci-dr-number` |
| `MENUBAR REPORTS/(menu 3) Stock Sales/4 Summary By Item/*.PNG` | `/reports/sales-summary-by-item` |
| `MENUBAR REPORTS/(menu 3) Stock Sales/5 Sales Markup by Transaction/*.PNG` | `/reports/sales-markup-by-transaction` |
| `MENUBAR REPORTS/(menu 3) Stock Sales/6 By Customer (Summary By Item)/*.PNG` | `/reports/sales-by-customer-summary-by-item` |
| `MENUBAR REPORTS/(menu 4) Stock Transfers/1 Summary/*.PNG` | `/reports/transfers-summary` |
| `MENUBAR REPORTS/(menu 4) Stock Transfers/2 By Stock Name/*.PNG` | `/reports/transfers-by-stock-name` |
| `MENUBAR REPORTS/(menu 4) Stock Transfers/3 By Entry ID/*.PNG` | `/reports/transfers-by-entry-id` |
| `MENUBAR REPORTS/(menu 4) Stock Transfers/4 Summary By Entry ID/*.PNG` | `/reports/transfers-summary-by-entry-id` |
| `MENUBAR REPORTS/(menu 4) Stock Transfers/5 Summary By Item/*.PNG` | `/reports/transfers-summary-by-item` |
| `MENUBAR REPORTS/(menu 4) Stock Transfers/6 Transfer Markup By Transaction/*.PNG` | `/reports/transfers-markup-by-transaction` |
| `MENUBAR REPORTS/(menu 4) Stock Transfers/7 By Branch/*.PNG` | `/reports/transfers-by-branch` |

## Sidebar Reports

| Reference screenshots | App route |
| --- | --- |
| `Reports/purchases summary/*.PNG` | `/reports/purchases-summary` |
| `Reports/sales summary/*.PNG` | `/reports/sales-summary` |
| `Reports/ap ledger/*.PNG` | `/reports/ap-ledger` |
| `Reports/ar ledger/*.PNG` | `/reports/ar-ledger` |
| `Reports/incoming checklist/*.PNG` | `/reports/incoming-check-list` |
| `Reports/outgoing check list/*.PNG` | `/reports/outgoing-check-list` |
| `Reports/expenses summary/*.PNG` | `/reports/expenses-summary` |
| `Reports/income statement/*.PNG` | `/reports/income-statement` |
| `Reports/incentive report/*.PNG` | `/reports/incentive` |

## Special Reports

| Reference screenshots | App route |
| --- | --- |
| `Special Reports/1 Daily Sales & Collection/*.PNG` | `/reports/daily-sales-collection` |
| `Special Reports/2 Incoming Check Calendar/*.PNG` | `/reports/incoming-check-calendar` |
| `Special Reports/3 Daily Due Check/*.PNG` | `/reports/daily-due-check` |
| `Special Reports/4 Stock Sales & Transfer/*.PNG` | `/reports/stock-sales-transfer` |
| `Special Reports/5 Stk. Sales & Transfer Amount/*.PNG` | `/reports/stock-sales-transfer-amount` |

## Current Verification Notes

- Sales File cash checkbox placement was checked against `Files/Sales File/1.PNG`; it should appear directly below Entry ID.
- Purchase File cash checkbox placement was checked against `Files/Purchase File/2.PNG`; it should appear directly below Entry ID in the left column.
- Sales File `Mode Of Payment` was checked against `Files/Sales File/mode-of-payment tab.PNG`; the checks table opens while the right totals column remains visible.
- Stock Transaction File `Discounts/Additionals/Summary` was checked against `Files/Stock Transfer/discounts - additional - summary.PNG`; the discounts/additionals panel opens while the right totals column remains visible.
- Sales File and Stock Transaction File intentionally use `SO Number` selection to populate detail rows from Stock Out data, per the later requirement. That supersedes the older direct-entry customer/transaction layout shown in some form screenshots.
- Stock Reorder Point was checked against `MENUBAR REPORTS/(menu 1) Stock Summary/stock reorder point/2.PNG`; zero-SOH rows with positive deficit must be included, while rows with no deficit are excluded.
- Stock Transfer Summary was checked against `MENUBAR REPORTS/(menu 4) Stock Transfers/1 Summary/2.PNG`; the preview sidebar should show expandable category markers without rendering child nodes.
- Stock Transfer By Stock Name was checked against `MENUBAR REPORTS/(menu 4) Stock Transfers/2 By Stock Name/2.PNG`; the preview sidebar should show expandable category markers without stock-code child nodes.
- Stock Transfer By Entry ID was checked against `MENUBAR REPORTS/(menu 4) Stock Transfers/3 By Entry ID/2.PNG`; the preview sidebar should be a flat Entry ID list.
- Stock Transfer Summary By Entry ID was checked against `MENUBAR REPORTS/(menu 4) Stock Transfers/4 Summary By Entry ID/2.PNG`; the preview sidebar should show expandable branch markers without entry child nodes.
- Stock Transfer Summary By Item was checked against `MENUBAR REPORTS/(menu 4) Stock Transfers/5 Summary By Item/2.PNG`; the preview sidebar should show expandable category markers without stock child nodes.
- Transfer Markup By Transaction was checked against `MENUBAR REPORTS/(menu 4) Stock Transfers/6 Transfer Markup By Transaction/2.PNG`; the paper title should read `SALES MARKUP BY TRANSACTION` and the preview pane should remain empty.
- Stock Transfer By Branch was checked against `MENUBAR REPORTS/(menu 4) Stock Transfers/7 By Branch/2.PNG`; the preview sidebar should show expandable branch markers without category child nodes.
- Sales Summary was checked against `Reports/sales summary/detailed.PNG` and `Reports/sales summary/summary.PNG`; the preview sidebar is a customer list, detailed columns are Entry ID/Date/OR-CI Number/Gross Amount/Net Amount, and summary's first header is `Customer` without a colon.
- AP Ledger was checked against `Reports/ap ledger/detailed.PNG`, `summary.PNG`, and `aging.PNG`; detailed mode should not show a range/as-of line, summary keeps supplier rows with an as-of line, and aging keeps a blank preview pane with the screenshot's `Accounts Receivable Aging` title.
- AR Ledger was checked against `Reports/ar ledger/detailed.PNG`, `summary.PNG`, and `aging.PNG`; detailed mode should not show a range/as-of line, summary is Company/Balance with an as-of line, and aging keeps a blank preview pane with Outstanding Check/Total Balance columns.
- Incoming Check List was checked against `Reports/incoming checklist/detailed.PNG` and `summary post dated.PNG`; detailed mode uses expandable payee markers and only the `INCOMING CHECK LIST` paper title, while postdated summary keeps the company title, cut-off label, and spaced grand-total footer.
- Outgoing Check List was checked against `Reports/outgoing check list/detailed.PNG` and `summary.PNG`; detailed mode uses expandable payee markers and keeps the company title, while postdated summary keeps the company title, cut-off label, and spaced grand-total footer.
- Expenses Summary was checked against `Reports/expenses summary/detailed 1.PNG`, `detailed 2.PNG`, and `summary.PNG`; detailed preview stays blank, summary preview lists expense categories, and summary `Grand Total` renders below the table instead of as a bordered footer row.
- Income Statement was checked against `Reports/income statement/income statement 1.PNG` and `income statement 2.PNG`; the preview pane stays blank, the paper title is `Income Statement`, and the report uses the From/To range line with Sales, Cost of Sales, Operating Expenses, Other Income, and Net Income sections.
- Incentive Report was checked against `Reports/incentive report/2.PNG`; the preview pane stays blank, empty results render one blank table row, and empty Total/Group Total/Grand Total cells remain blank rather than displaying zeroes.
- Daily Sales & Collection was checked against `Special Reports/1 Daily Sales & Collection/1.PNG` and `2.PNG`; the option dialog shows only the `Select Date of Report` fieldset and date picker, the preview pane stays blank, and remittance totals use cash sales/cash receipts/disbursements/check deposits.
- Route-level create/update/delete tests cover every configured master and transaction form.
- Production report repository methods are SQL-backed through `internal/repositories/postgres.go`; hardcoded report rows are confined to `internal/http/server_test.go` fake-store tests.
- Postgres integration coverage has passed for master create/update/delete, purchase document create/update/delete, Stock Out selection, Sales from Stock Out, and Stock Transaction from Stock Out:
  `CIMS_TEST_DATABASE_URL='postgres://cims:cims@localhost:5432/cims?sslmode=disable' go test ./internal/repositories -run TestPostgresStoreMasterAndPurchaseDocumentCRUD -count=1 -v`.
- Remaining report visual work should compare each route's option dialog and generated paper against the mapped reference screenshots.
