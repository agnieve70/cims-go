# Reverse Engineering Notes

Target inspected:

- `/Users/gabril/Downloads/cims_downloaded_crystal_pdf/SamCSS/APSMon/APSMon.exe`
- supporting OCX/DLL files in `/Users/gabril/Downloads/cims_downloaded_crystal_pdf/SamCSS/APSMon`

## What Is Recoverable

The application can be partially reverse engineered. The useful recoverable pieces are:

- Binary type and runtime dependencies
- Form/control/class names embedded by VB6
- Menu/report labels
- Crystal report file names
- Some SQL fragments, stored procedure calls, table names, and view names
- Configuration keys and database connection strings
- Custom OCX public/event names

The original source code cannot be faithfully recovered from these files on this machine. The main executable is a compiled VB6 Windows GUI binary with stripped symbols. There are no `.vbp`, `.frm`, `.bas`, or `.cls` files in the bundle.

## Binary Facts

`APSMon.exe`:

- PE32 Windows GUI executable
- Intel 80386
- VB6 runtime import: `MSVBVM60.DLL`
- Linker version: 6.0
- Timestamp: 2013-05-17
- Main sections:
  - `.text`
  - `.data`
  - `.rsrc`
- Resources: icons/cursors/version-style resources; no directly recoverable VB source

The supporting custom controls/libraries are also VB6 binaries:

- `MyRecordsetMySQL.ocx`
- `MyControls.ocx`
- `MyButton.ocx`
- `PCSGrid.ocx`
- `CompInfo.dll`
- `JSSecurityMod.dll`

## Important Runtime Dependencies

Recovered from strings/imports:

- `MSVBVM60.DLL`
- `ADODB`
- `msado28.tlb`
- `crviewer9.oca`
- Crystal Decisions ActiveX Viewer
- `MyRecordsetMySQL.ocx`
- `MyButton.ocx`
- `PCSGrid.ocx`

This confirms the legacy app is VB6 + ADO/MySQL + Crystal Reports.

## Report Runtime Pattern

Recovered symbols indicate a generic Crystal report form:

- `ReportFormTTX`
- `ReportFile`
- `Report Caption`
- `LoadReportEx`
- `RepCaption`
- `RepShowTree`
- `oParams`
- `tReportPath`
- `txtDBServer`
- `txtDatabaseName`

The app likely builds/selects a recordset, loads a `.rpt` file by name, passes parameters/record data, and displays it through Crystal ActiveX Viewer.

## Recovered Report Names And Calls

The following report menu labels, Crystal file names, and SQL/procedure fragments were recovered from `APSMon.exe` strings.

| Menu/behavior | Crystal file/string | Recovered data source clue |
| --- | --- | --- |
| Stock Summary | `StockSummary` | `call cims.Rpt_StockSummary_APS(...)` |
| Stock Sales By Customer | `StockSalesGPCustomer` | `call cims.Rpt_StockSales_APS(...)` |
| Stock Transfer | `StockTransfer` | likely report over transfer summary data |
| Stock Transfer Summary | `StockAging` appears near this menu block | `call cims.Rpt_StockAging_APS(...)` appears nearby |
| Stock Transfer Summary | `StockTransfer` / summary menu | `select * from cims.stocktransfersummary where showinaps=1 and transferdate between ...` |
| Stock Ledger | `StockLedger` | `call cims.Rpt_StockLedger_APS(...)` |
| Stock Aging | `StockAging` | `call cims.Rpt_StockAging_APS(...)` |
| Stock Reorder Point | `StockSummaryROP` | no full SQL recovered |
| Stock Sales By Customer - Summary By Item | `StockSalesGPCustomerSummaryByItem` | likely shares sales APS procedure/result set |
| Stock Sales By Stock Code | `StockSalesGPStockCode` | likely shares sales APS procedure/result set |
| Stock Sales Summary By Item | `StockSalesSummaryItem` | likely shares sales APS procedure/result set |
| Stock Transfer By Branch | `StockTransferByBranch` | likely uses transfer summary rows |
| Stock Sales By Reference Number | `StockSalesGPReference` | likely shares sales APS procedure/result set |
| Stock Transfer Summary By Item | `StockTransferSummaryItem` | likely uses transfer summary rows |
| Stock Transfer Summary By Stock Name | `StockTransferStockCode` | likely uses transfer summary rows |
| Stock Transfer By Entry ID | `StockTransferEntryID` | likely uses transfer summary rows |
| Stock Transfer Summary By Entry ID | `StockTransferSummaryEntryID` | likely uses transfer summary rows |
| APS Client Monitor Aging | `APSCMAging` | `call rpt_apscmaging('...')` |
| APS Client Monitor Aging 2 | `APSCMAging2` | `select * from vw_apsclients;` |

## Other Recovered SQL/Schema Clues

APS/client-monitor and utility logic:

```sql
select * from vw_apsclients;
select * from EmployeeData where EmpCode='...'
select last_day('...')
select year(...)
select * from (select ...) x where (...)
select * from c_particulars where pk=...
delete from c_particulars where pk=...
insert into c_particulars (pk,partname,partqty,partcost,partamount,partorder,parttype) values ...
select calculate_interestDays('...')
select * from maints;
SELECT idnum,company from c_recepients where (company LIKE '...')
select posted, concat(company,' ',checknumber,' ',bankname,' ',format(checkamount,2)) chkdata from vw_checkdata where checkdate='...'
select posted, concat(cast(checkdate as char),' ',company,' ',checknumber,' ',bankname,' ',format(checkamount,2)) chkdata from vw_checkdata where posted=0 and checkdate<'...'
select sum(checkamount) tca from c_outgoingchecks where checkdate between ...
select * from items where name='...'
select * from items where code='...'
select aps_name from apsprofile where aps_id=...
select * from apsclients where cli_name='...'
select * from apsprofile where aps_name='...'
update apsvisits set transdate='...'
insert into apsvisits (transno,idnum,transdate,sowno,fatteningno,boarno,pigletno,feedsused,medsused,feedsused_others,medsused_others,user,lastupdate,remarks) values (...)
select last_insert_id() liid;
select * from apsclients where cli_id=...
select * from feeds order by f_id;
select * from meds order by m_id;
select * from apsvisits where idnum='...' order by transdate, pk;
delete from apsvisits where pk=...
select * from apsvisits where pk=...
select max(transno) mtn from apsvisits;
```

Security/user module logic in `JSSecurityMod.dll`:

```sql
select usr_id,usr_loginname,usr_name,usr_description,usr_password from users where usr_deleted=0 order by usr_loginname;
select * from modules order by mod_id;
update users set usr_deleted=1,usr_deleteddt=now() where usr_id=...
select ifnull(max(usr_id),0)+1 mui from users;
call spjs_getbrancheswithuser(...)
select * from users where usr_id=...
select a.* from access a left join users b on (a.usr_id=b.usr_id) where ifnull(b.usr_deleted,0)=0 and a.usr_id=...
insert into logs (usr_id,mod_id,rit_id,ref_table,ref_id,log_pc) values (...)
select * from users where usr_loginname='...'
select usr_id,usr_loginname,(usr_password=md5('...')) pwok from users where usr_loginname='...'
select * from access where usr_id=...
select * from settings where setname='overridekey';
insert into logins (usr_id,mod_id,login_pc) values (...)
select (usr_password=md5('...')) oldpwok from users where usr_id=...
update users set usr_password=md5('...') where usr_id=...
```

## Custom Control Behavior

`MyRecordsetMySQL.ocx` appears to implement the legacy data/navigation/delete behavior:

- `MyRecordSetMS`
- `Recordset`
- `AllowDelete`
- `BeforeDelete`
- `AfterDelete`
- `ClickButton`
- "Transaction file recordset"
- "Search specific record from the file"
- "Delete current displayed record."

This is relevant to the Go app because some behavior such as form navigation, search, and delete visibility may have come from this control in the original app.

`PCSGrid.ocx` exposes editable grid behavior:

- `InsertRowItem`
- `ConfirmDelete`
- `ValidateRow`
- `ValidateColumn`
- `RowDeleted`
- `ColFormula`
- `DataFormat`

## Limits

The pass above is static string/import/resource reverse engineering. It does not reconstruct:

- Original VB6 source code
- Full control event handlers
- Complete SQL assembled by string concatenation
- Crystal formulas or grouping definitions stored inside `.rpt`
- Runtime values passed into report parameters

For deeper recovery, use a Windows VM with:

- VB6-aware decompiler/disassembler for `APSMon.exe`
- Crystal Reports Designer/runtime to open `.rpt` files and export formulas/SQL/report definition
- Optional ODBC/MySQL test database to observe generated queries at runtime

## Practical Use For The Go Rewrite

For the current Go app, the recovered report procedure names and PDF samples are the most useful assets:

- Use the recovered stored procedure/view names as clues for report grouping and data-source intent.
- Use the exported PDFs as the visible report specification.
- Use the current Go repository queries/templates as the implementation surface.
- Do not depend on exact source recovery from the VB6 binaries.
