# Legacy Daily Sales Database Notes

Source inspected:

- `/Users/gabril/Downloads/DailyBackupAuto 20260613 1802.sql`

The dump is a MySQL backup, about 2 GB, with several databases. The Daily Sales and Collection report logic is in the `cims` database.

## Main Finding

The legacy app did not appear to render Daily Sales directly from one normal Crystal `.rpt` data source. It used a staging/output table:

```sql
CREATE TABLE `dailysalesreport` (
  `pk` int(11) NOT NULL AUTO_INCREMENT,
  `header0` varchar(256) DEFAULT NULL,
  `detail0` varchar(256) DEFAULT NULL,
  `detail1` varchar(128) DEFAULT NULL,
  `credit` double(15,3) DEFAULT NULL,
  `debit` double(15,3) DEFAULT NULL,
  `total0` double(15,3) DEFAULT NULL,
  `total1` double(15,3) DEFAULT NULL,
  `IsBold` tinyint(1) NOT NULL DEFAULT '0',
  `BottomLine` int(11) DEFAULT NULL,
  `compname` varchar(64) DEFAULT NULL,
  PRIMARY KEY (`pk`),
  KEY `compname` (`compname`)
);
```

This matches the strings recovered from `CIMSBodega.exe`:

```sql
select * from dailysalesreport where compname=' ';
```

So the VB6 app likely built printable rows for the current workstation (`compname`), then Crystal displayed those rows.

Example staged rows:

```sql
(' CASH SALES', '', '', 0, 0, 0, 0, 1, 1, 'ADMIN-PC')
('', 'CASH/SAROMINES', '17570', 4370, 0, 0, 0, 0, 1, 'ADMIN-PC')
('', 'TOTAL CASH SALES', '', 0, 4370, 0, 0, 1, 0, 'ADMIN-PC')
(' CHARGE SALES', '', '', 0, 0, 0, 0, 1, 1, 'ADMIN-PC')
(' CASH RECEIPTS', '', '', 0, 0, 0, 0, 1, 1, 'ADMIN-PC')
(' DISBURSEMENTS', '', '', 0, 0, 0, 0, 1, 1, 'ADMIN-PC')
(' CHECK DEPOSITS', '', '', 0, 0, 0, 0, 1, 1, 'ADMIN-PC')
('', 'TOTAL CASH REMITTANCE', '', 0, 4370, 0, 0, 1, 1, 'ADMIN-PC')
('', 'TOTAL REMITTANCE', '', 0, 4370, 0, 0, 1, 2, 'ADMIN-PC')
```

Interpretation:

- `header0` is the section label.
- `detail0` is the customer/category/total label.
- `detail1` is the reference/check/customer text.
- `credit` is used for normal line amount in the amount column.
- `debit` is used for total lines in the totals column.
- `total0`/`total1` are available extra total columns, likely for split cash/check totals.
- `IsBold` marks section and total rows.
- `BottomLine` appears to control underline/border style.
- `compname` isolates one user's temporary report output.

## Daily Summary Table

The dump also has a summary table:

```sql
CREATE TABLE `salesreportdata` (
  `RowDateEncoded` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `EntryID` int(11) NOT NULL AUTO_INCREMENT,
  `CashSales` double(15,3) NOT NULL DEFAULT '0.000',
  `CheckSales` double(15,3) NOT NULL DEFAULT '0.000',
  `ChargeSales` double(15,3) NOT NULL DEFAULT '0.000',
  `CashCollection` double(15,3) NOT NULL DEFAULT '0.000',
  `CheckCollection` double(15,3) NOT NULL DEFAULT '0.000',
  `Expenses` double(15,3) NOT NULL DEFAULT '0.000',
  `Remittance` double(15,3) NOT NULL DEFAULT '0.000',
  `Branch` varchar(50) DEFAULT NULL,
  `SalesDate` date DEFAULT NULL,
  `Remarks` varchar(250) DEFAULT NULL,
  `Posted` tinyint(1) NOT NULL DEFAULT '0',
  `Encoder` varchar(256) DEFAULT NULL,
  `lastupdate` varchar(256) DEFAULT NULL,
  `updatedby` varchar(256) DEFAULT NULL,
  PRIMARY KEY (`EntryID`)
);
```

And a view:

```sql
CREATE VIEW `salesreport` AS
select
  `salesreportdata`.`EntryID` AS `EntryID`,
  `salesreportdata`.`SalesDate` AS `SalesDate`,
  `salesreportdata`.`Branch` AS `Branch`,
  (`salesreportdata`.`CheckSales` + `salesreportdata`.`CashSales`) AS `Sales`,
  `salesreportdata`.`ChargeSales` AS `Charge`,
  (`salesreportdata`.`CashCollection` + `salesreportdata`.`CheckCollection`) AS `Collection`,
  `salesreportdata`.`Expenses` AS `Expenses`,
  `salesreportdata`.`Remittance` AS `Remittance`
from `salesreportdata`;
```

Important: `salesreportdata.Remittance` in sample rows equals:

```text
CashSales + CheckSales + CashCollection + CheckCollection - Expenses
```

For the detailed Daily Sales PDF currently being reproduced, the visible legacy sample effectively uses:

```text
TOTAL CASH REMITTANCE = Cash Sales + Cash Receipts - Disbursements
TOTAL REMITTANCE = TOTAL CASH REMITTANCE + Check Deposits
```

The summary table treats check sales/check collections as part of remittance; the detailed report separates check deposits into the final remittance line.

## Source Tables Confirmed

The legacy source tables relevant to Daily Sales include:

- `salesdata`: sales documents, with `Cash`, `ORNumber`, `CINumber`, `SalesDate`, `Customer`, `NetTotal`, `CashAmount`, and `CheckAmount`.
- `arcreditdata`: accounts receivable collections, with `TranDate`, `Reference`, `Company`, `Amount`, `CashAmount`, and `CheckAmount`.
- `rebatesdata`: rebates/other collection-style rows, with `EntryDate`, `Reference`, `Company`, `CashAmount`, `CheckAmount`, and `TotalAmount`.
- `expensesdata`: expense header, with `TranDate`, `Reference`, `TotalAmount`, and `notinsales`.
- `expensesdatadetails`: expense lines, with `ExpCode`, `ExpName`, `Cash`, `Check`, `Reference`, and `TotalAmount`.
- `expenseschartdata`: expense chart, including Daily Sales exclusion flags.
- `incomingchecks`: incoming check detail rows, with `TranDate`, `CheckNumber`, `CheckDate`, `BankName`, `Amount`, `Reference`, and `Company`.

## Expense Exclusion Flags

The expense chart has the same special behavior recovered from the VB6 binary:

```sql
CREATE TABLE `expenseschartdata` (
  ...
  `notinsales` tinyint(1) NOT NULL DEFAULT '0',
  `fordscronly` tinyint(1) NOT NULL DEFAULT '0',
  ...
);
```

The legacy expense summary view excludes `fordscronly` rows:

```sql
CREATE VIEW `expensessummary` AS
select ...
from expensesdata a
left join expensesdatadetails b on a.EntryID = b.EntryID
left join expenseschartdata c on b.ExpCode = c.ExpCode
where ifnull(c.fordscronly, 0) = 0;
```

The VB6 UI label says:

- `FOR DAILY SALES AND COLLECTION REPORT ONLY.`
- `NOT INCLUDED IN DAILY SALES AND COLLECTION REPORT.`

The Go app currently maps this behavior through `expense_charts.exclude_daily_sales`.

## Comparison To Current Go Logic

Current Go logic is directionally aligned:

- Sales split into cash and charge sales.
- AR credit/rebates populate cash receipts and check amounts.
- Expense details populate disbursements.
- Expense chart exclusion is honored.
- Checks are separated into a check deposits section.
- Final remittance totals are calculated separately from charge sales.

The main structural difference is implementation style:

- Legacy: VB6 fills `dailysalesreport` as printable rows, then Crystal selects by `compname`.
- Go: SQL returns normalized report rows, and the Go template renders sections/totals directly.

This is a good modernization. It avoids temporary per-workstation report rows while preserving the report behavior.
