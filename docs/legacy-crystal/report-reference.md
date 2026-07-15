# Legacy Crystal Report Reference

Source bundle inspected:

- `/Users/gabril/Downloads/cims_downloaded_crystal_pdf/SamCSS`
- `/Users/gabril/Downloads/cims_downloaded_crystal_pdf/*.pdf`

## Extraction Result

The `SamCSS` folder is a deployed Windows application bundle, not a source checkout. It contains compiled PE executables, DLLs, OCX controls, configuration files, and Crystal Reports `.rpt` files.

No VB6/project source files were found:

- `.vbp`
- `.frm`
- `.bas`
- `.cls`
- `.vb`
- `.cs`
- `.sql`

The compiled `.exe/.dll/.ocx` files expose only limited strings such as form names, Crystal runtime references, report path fields, and MySQL/ADO control names. They do not contain usable source code.

## Reusable Assets

The useful legacy report sources are:

- Crystal report definitions: `/Users/gabril/Downloads/cims_downloaded_crystal_pdf/SamCSS/APSMon/Reports/*.rpt`
- Exported report PDFs: `/Users/gabril/Downloads/cims_downloaded_crystal_pdf/*.pdf`

The `.rpt` files are OLE compound documents with `Contents`, `QESession`, `ReportInfo`, and summary streams. Direct OLE extraction did not expose report SQL/formula text, so deeper report internals likely require Crystal Reports Designer/runtime on Windows.

The PDFs are the best available source for reproducing report behavior in this Go app: titles, filters, grouping, columns, totals, and pagination style are extractable.

## Crystal Reports Found

| Crystal file | Likely Go report |
| --- | --- |
| `StockLedger.rpt` | `/reports/stock-ledger` |
| `StockAging.rpt` | `/reports/stock-aging` |
| `StockSummary.rpt` | `/reports/stock-summary` |
| `StockSummaryROP.rpt` | `/reports/stock-reorder-point` |
| `StockSalesGPCustomer.rpt` | `/reports/sales-by-customer` |
| `StockSalesGPCustomerSummaryByItem.rpt` | `/reports/sales-by-customer-summary-by-item` |
| `StockSalesGPReference.rpt` | `/reports/sales-by-or-ci-dr-number` |
| `StockSalesGPStockCode.rpt` | `/reports/sales-by-stock-name` or stock-code variant |
| `StockSalesSummaryItem.rpt` | `/reports/sales-summary-by-item` |
| `StockTransfer.rpt` | `/reports/transfers-summary` |
| `StockTransferByBranch.rpt` | `/reports/transfers-by-branch` |
| `StockTransferEntryID.rpt` | `/reports/transfers-by-entry-id` |
| `StockTransferStockCode.rpt` | `/reports/transfers-by-stock-name` |
| `StockTransferSummaryEntryID.rpt` | `/reports/transfers-summary-by-entry-id` |
| `StockTransferSummaryItem.rpt` | `/reports/transfers-summary-by-item` |
| `APSCMAging.rpt`, `APSCMAging2.rpt` | APS-specific aging reference |
| `APSCMDetailed.rpt` | APS-specific detailed report |
| `APSCMFeeds.rpt` | APS-specific feeds report |
| `APSPerBarangay.rpt` | APS-specific barangay report |

## PDF Reference Mapping

| PDF | Report behavior captured |
| --- | --- |
| `purchases.pdf` | Detailed purchases grouped by supplier, entry rows, gross/net totals |
| `purchasessummary.pdf` | Purchases summary by supplier |
| `stock purchase-by reference.pdf` | Purchases grouped by reference number |
| `stock purchase-by-stock-code.pdf` | Purchases grouped by stock code |
| `stock purchase-by supplier.pdf` | Purchases grouped by supplier and stock |
| `sales.pdf` | Detailed sales grouped by customer |
| `salessummary.pdf` | Sales summary by customer |
| `stock sales-by-customer.pdf` | Stock sales grouped by category then customer |
| `stock sales-summary-by-item.pdf` | Sales by customer summary by item |
| `stock sales-by-reference-number.pdf` | Sales grouped by receipt/reference |
| `stock sales-stock code.pdf` | Sales grouped by stock code |
| `stock sales-by-item.pdf` | Sales summary by item |
| `stock sales-by-transaction.pdf` | Sales markup by transaction |
| `stocktransfer -summary.pdf` | Stock transfer grouped by category and branch |
| `stocktransfer -by-branch.pdf` | Stock transfer grouped by branch then category |
| `stocktransfer -by-entry-id.pdf` | Stock transfer grouped by entry ID |
| `stocktransfer -by-stock-name.pdf` | Stock transfer grouped by category, stock, branch |
| `stocktransfer -summary-by-item.pdf` | Stock transfer summary by item |
| `stocktransfer -by-transation.pdf` | Transfer markup by transaction |
| `stocksalestransferssummary.pdf` | Combined stock sales and transfer quantity summary |
| `stocksalestransfersamountsummary.pdf` | Combined stock sales and transfer amount/markup summary |
| `stock summary-stock ledger.pdf` | Stock ledger by category and stock |
| `stock summary-stock aging.pdf` | Stock aging buckets by category and stock |
| `stock summary-stock summary.pdf` | Stock summary as of cutoff date |
| `stock summary-stock reorder-point.pdf` | Stock reorder point with SOH/min/deficit |
| `apledger.pdf` | AP detailed ledger |
| `apledgersummary.pdf` | AP summary |
| `apaging.pdf` | AP aging sample, although title text says Accounts Receivable Aging |
| `arledger.pdf` | AR detailed ledger |
| `arledgersummary.pdf` | AR summary |
| `araging.pdf` | AR aging |
| `incomingchecklist.pdf` | Incoming check list grouped by payee/month |
| `incomingchecklistpostdated.pdf` | Incoming postdated check summary |
| `outgoingchecklist.pdf` | Outgoing check list grouped by payee/month |
| `outgoingchecklistpostdated.pdf` | Outgoing postdated check summary |
| `dailysalesreport.pdf` | Daily sales and collection |
| `dailyduecheck.pdf` | Daily due check |
| `expensesdetailed.pdf` | Detailed expenses cross-tab by date/category |
| `expensessummary.pdf` | Expense summary by expense chart |
| `incomestatement.pdf` | Income statement |

## Existing Go Implementation Points

Report routes are registered in:

- `internal/http/server.go`

Report request/default handling is in:

- `internal/http/reports.go`

Report SQL/data logic is in:

- `internal/repositories/postgres.go`

Report HTML templates are in:

- `templates/*_report.gohtml`

## Practical Migration Approach

1. Use PDFs as the report specification: title, date wording, grouping order, columns, subtotal labels, and grand-total behavior.
2. Use current Go SQL in `internal/repositories/postgres.go` as the implementation base.
3. Compare each Go report output against its matching PDF sample.
4. Only use Crystal `.rpt` files for names/inventory unless a Windows Crystal Reports tool is available to export formulas/SQL.
5. If exact Crystal internals are needed, open the `.rpt` files in Crystal Reports Designer on Windows and export report definition/formulas/SQL from there.
