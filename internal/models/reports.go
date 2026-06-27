package models

type PurchaseReportRow struct {
	Supplier   string
	EntryID    string
	EntryDate  string
	ORCINumber string
	Type       string
	GrossCents int64
	NetCents   int64
}

type PurchaseByDRNumberReportRow struct {
	Reference     string
	PurchaseDate  string
	Type          string
	Supplier      string
	StockCode     string
	StockName     string
	Quantity      int64
	UnitCostCents int64
	AmountCents   int64
}

type PurchaseByStockCodeReportRow struct {
	Reference     string
	PurchaseDate  string
	Type          string
	Supplier      string
	StockCode     string
	StockName     string
	Quantity      int64
	UnitCostCents int64
	AmountCents   int64
}

type PurchaseBySupplierReportRow struct {
	Reference     string
	PurchaseDate  string
	Type          string
	Supplier      string
	StockCode     string
	StockName     string
	Quantity      int64
	UnitCostCents int64
	AmountCents   int64
}

type SalesReportRow struct {
	Customer   string
	EntryID    string
	EntryDate  string
	ORCINumber string
	Type       string
	GrossCents int64
	NetCents   int64
}

type SalesByORCIDRNumberReportRow struct {
	Reference   string
	SalesDate   string
	Type        string
	Customer    string
	StockCode   string
	StockName   string
	Quantity    int64
	PriceCents  int64
	AmountCents int64
}

type SalesMarkupByTransactionReportRow struct {
	SalesDate    string
	EntryID      string
	SalesType    string
	ReceiptNo    string
	ItemGroup    string
	MarkupCents  int64
	CapitalCents int64
}

type SalesByCustomerReportRow struct {
	Category    string
	Customer    string
	Reference   string
	SalesDate   string
	Type        string
	StockCode   string
	StockName   string
	Quantity    int64
	PriceCents  int64
	AmountCents int64
}

type SalesByStockNameReportRow struct {
	Category    string
	Customer    string
	Reference   string
	SalesDate   string
	Type        string
	StockCode   string
	StockName   string
	Quantity    int64
	PriceCents  int64
	AmountCents int64
}

type APLedgerReportRow struct {
	SupplierID     string
	SupplierCode   string
	SupplierName   string
	Representative string
	EntryID        string
	EntryDate      string
	Reference      string
	Kind           string
	DeltaCents     int64
}

type ARLedgerReportRow struct {
	CustomerID   string
	CustomerCode string
	CustomerName string
	CreditTerm   string
	CreditLimit  int64
	EntryID      string
	EntryDate    string
	Reference    string
	Kind         string
	DeltaCents   int64
}

type IncomingCheckReportRow struct {
	Payee       string
	Reference   string
	CheckDate   string
	Number      string
	BankName    string
	AmountCents int64
}

type OutgoingCheckReportRow struct {
	Payee       string
	Reference   string
	CheckDate   string
	Number      string
	BankName    string
	AmountCents int64
}

type ExpenseReportRow struct {
	CategoryID   string
	CategoryCode string
	CategoryName string
	EntryDate    string
	CashCents    int64
	CheckCents   int64
	TotalCents   int64
}

type IncomeStatementRow struct {
	Section     string
	Label       string
	AmountCents int64
}

type IncentiveReportRow struct {
	AgriPost string
	Qty      int64
	VIP      int64
	APS      int64
	Takals   int64
	Farm     int64
}

type DailySalesCollectionReportRow struct {
	Section          string
	Name             string
	Reference        string
	AmountCents      int64
	CheckAmountCents int64
	SortKey          string
}

type StockSalesTransferReportRow struct {
	Category    string
	StockCode   string
	StockName   string
	SalesQty    int64
	TransferQty int64
}

type StockSalesTransferAmountReportRow struct {
	Category            string
	CashSalesCents      int64
	ChargeSalesCents    int64
	TransferCents       int64
	SalesMarkupCents    int64
	TransferMarkupCents int64
}

type StockTransferSummaryReportRow struct {
	Category     string
	Branch       string
	Reference    string
	TransferDate string
	StockCode    string
	StockName    string
	Quantity     int64
	AmountCents  int64
}

type StockTransferByStockNameReportRow struct {
	Category     string
	Branch       string
	Reference    string
	TransferID   string
	TransferDate string
	StockCode    string
	StockName    string
	Quantity     int64
	AmountCents  int64
}

type StockTransferByBranchReportRow struct {
	Branch       string
	Category     string
	Reference    string
	TransferDate string
	StockCode    string
	StockName    string
	Quantity     int64
	AmountCents  int64
}

type StockTransferSummaryByItemReportRow struct {
	Category    string
	StockCode   string
	StockName   string
	Quantity    int64
	AmountCents int64
}

type StockTransferByEntryIDReportRow struct {
	EntryID      string
	Reference    string
	TransferID   string
	Remarks      string
	TransferDate string
	Branch       string
	StockCode    string
	StockName    string
	Quantity     int64
	AmountCents  int64
	NetCents     int64
}

type StockTransferMarkupByTransactionReportRow struct {
	TransferDate string
	EntryID      string
	TransferTo   string
	ReceiptNo    string
	ItemGroup    string
	MarkupCents  int64
	CapitalCents int64
}

type StockLedgerReportRow struct {
	StockID   string
	Category  string
	StockCode string
	StockName string
	EntryDate string
	SortKey   string
	Reference string
	Company   string
	Kind      string
	QtyDelta  int64
}

type StockAgingReportRow struct {
	Category  string
	StockCode string
	StockName string
	Bucket0   int64
	Bucket1   int64
	Bucket2   int64
	Bucket3   int64
	Bucket4   int64
	Bucket5   int64
}

type StockReorderPointReportRow struct {
	Category     string
	StockCode    string
	StockName    string
	SOH          int64
	MinInventory int64
	Deficit      int64
}

type StockSummaryReportRow struct {
	Category      string
	StockCode     string
	StockName     string
	HasStock      bool
	SOH           int64
	UnitCostCents int64
	AmountCents   int64
}

type DailyDueCheckReportRow struct {
	ClientName  string
	CheckDate   string
	CheckNumber string
	BankName    string
	AmountCents int64
}
