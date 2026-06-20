package http

import (
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"cims-go/internal/models"
)

const defaultReportMonth = int(time.January)

type purchaseReportData struct {
	Generated     bool
	ReportType    string
	Coverage      string
	Year          int
	Month         int
	FromDate      string
	ToDate        string
	PaperSize     string
	PaperClass    string
	Title         string
	RangeLabel    string
	CurrentPage   int
	TotalPages    int
	Suppliers     []string
	Groups        []purchaseReportSupplierGroup
	SummaryRows   []purchaseReportSummaryRow
	TotalGross    string
	TotalNet      string
	TotalGrossRaw int64
	TotalNetRaw   int64
}

type purchaseReportSupplierGroup struct {
	Supplier   string
	Rows       []purchaseReportLine
	GrossTotal string
	NetTotal   string
}

type purchaseReportLine struct {
	EntryID    string
	Date       string
	ORCINumber string
	Gross      string
	Net        string
}

type purchaseReportSummaryRow struct {
	Supplier string
	Gross    string
	Net      string
}

type purchaseByDRNumberReportData struct {
	Generated      bool
	Coverage       string
	Year           int
	Month          int
	FromDate       string
	ToDate         string
	PaperSize      string
	PaperClass     string
	Title          string
	RangeLabel     string
	CurrentPage    int
	TotalPages     int
	Groups         []purchaseByDRNumberGroup
	TotalQuantity  string
	TotalAmount    string
	TotalQtyRaw    int64
	TotalAmountRaw int64
}

type purchaseByDRNumberGroup struct {
	Reference string
	Date      string
	Rows      []purchaseByDRNumberLine
	TotalQty  string
	TotalAmt  string
	QtyRaw    int64
	AmtRaw    int64
}

type purchaseByDRNumberLine struct {
	Supplier string
	Code     string
	Stock    string
	Quantity string
	Cost     string
	Amount   string
	QtyRaw   int64
	AmtRaw   int64
}

type purchaseByStockCodeReportData struct {
	Generated      bool
	Coverage       string
	Year           int
	Month          int
	FromDate       string
	ToDate         string
	PaperSize      string
	PaperClass     string
	Title          string
	RangeLabel     string
	CurrentPage    int
	TotalPages     int
	Groups         []purchaseByStockCodeGroup
	TotalQuantity  string
	TotalAmount    string
	TotalQtyRaw    int64
	TotalAmountRaw int64
}

type purchaseByStockCodeGroup struct {
	StockCode string
	StockName string
	Rows      []purchaseByStockCodeLine
	TotalQty  string
	TotalAmt  string
	QtyRaw    int64
	AmtRaw    int64
}

type purchaseByStockCodeLine struct {
	Reference string
	Date      string
	Supplier  string
	Quantity  string
	Cost      string
	Amount    string
	QtyRaw    int64
	AmtRaw    int64
}

type purchaseBySupplierReportData struct {
	Generated      bool
	Coverage       string
	Year           int
	Month          int
	FromDate       string
	ToDate         string
	PaperSize      string
	PaperClass     string
	Title          string
	RangeLabel     string
	CurrentPage    int
	TotalPages     int
	Groups         []purchaseBySupplierGroup
	TotalQuantity  string
	TotalAmount    string
	TotalQtyRaw    int64
	TotalAmountRaw int64
}

type purchaseBySupplierGroup struct {
	Supplier    string
	StockGroups []purchaseBySupplierStockGroup
	TotalQty    string
	TotalAmt    string
	QtyRaw      int64
	AmtRaw      int64
}

type purchaseBySupplierStockGroup struct {
	StockCode string
	StockName string
	Rows      []purchaseBySupplierLine
	TotalQty  string
	TotalAmt  string
	QtyRaw    int64
	AmtRaw    int64
}

type purchaseBySupplierLine struct {
	Reference string
	Date      string
	Code      string
	Stock     string
	Quantity  string
	Cost      string
	Amount    string
	QtyRaw    int64
	AmtRaw    int64
}

type salesReportData struct {
	Generated     bool
	ReportType    string
	Coverage      string
	Year          int
	Month         int
	FromDate      string
	ToDate        string
	PaperSize     string
	PaperClass    string
	Title         string
	RangeLabel    string
	CurrentPage   int
	TotalPages    int
	Customers     []string
	Groups        []salesReportCustomerGroup
	SummaryRows   []salesReportSummaryRow
	TotalGross    string
	TotalNet      string
	TotalGrossRaw int64
	TotalNetRaw   int64
}

type salesReportCustomerGroup struct {
	Customer   string
	Rows       []salesReportLine
	GrossTotal string
	NetTotal   string
}

type salesReportLine struct {
	EntryID    string
	Date       string
	ORCINumber string
	Gross      string
	Net        string
}

type salesReportSummaryRow struct {
	Customer string
	Gross    string
	Net      string
}

type salesByORCIDRNumberReportData struct {
	Generated      bool
	Coverage       string
	Year           int
	Month          int
	FromDate       string
	ToDate         string
	PaperSize      string
	PaperClass     string
	Title          string
	RangeLabel     string
	CurrentPage    int
	TotalPages     int
	Groups         []salesByORCIDRNumberGroup
	TotalQuantity  string
	TotalAmount    string
	TotalQtyRaw    int64
	TotalAmountRaw int64
}

type salesByORCIDRNumberGroup struct {
	Reference string
	Date      string
	Rows      []salesByORCIDRNumberLine
	TotalQty  string
	TotalAmt  string
	QtyRaw    int64
	AmtRaw    int64
}

type salesByORCIDRNumberLine struct {
	Customer string
	Code     string
	Stock    string
	Quantity string
	Price    string
	Amount   string
	QtyRaw   int64
	AmtRaw   int64
}

type salesMarkupByTransactionReportData struct {
	Generated          bool
	Coverage           string
	Year               int
	Month              int
	FromDate           string
	ToDate             string
	PaperSize          string
	PaperClass         string
	Title              string
	RangeLabel         string
	CurrentPage        int
	TotalPages         int
	Rows               []salesMarkupByTransactionLine
	Pages              []salesMarkupByTransactionPage
	TotalMarkup        string
	TotalMarkupRaw     int64
	TotalCapitalRaw    int64
	TotalMarkupPercent string
}

type salesMarkupByTransactionLine struct {
	SalesDate     string
	EntryID       string
	SalesType     string
	ReceiptNo     string
	ItemGroup     string
	Markup        string
	MarkupPercent string
	MarkupRaw     int64
	CapitalRaw    int64
}

type salesMarkupByTransactionPage struct {
	Number int
	Rows   []salesMarkupByTransactionLine
	Last   bool
}

type salesSummaryByItemReportData struct {
	Generated      bool
	Coverage       string
	Year           int
	Month          int
	FromDate       string
	ToDate         string
	PaperSize      string
	PaperClass     string
	Title          string
	RangeLabel     string
	CurrentPage    int
	TotalPages     int
	Categories     []salesSummaryByItemCategoryGroup
	TotalQuantity  string
	TotalAmount    string
	TotalQtyRaw    int64
	TotalAmountRaw int64
}

type salesSummaryByItemCategoryGroup struct {
	Category string
	Rows     []salesSummaryByItemLine
	TotalQty string
	TotalAmt string
	QtyRaw   int64
	AmtRaw   int64
}

type salesSummaryByItemLine struct {
	StockCode string
	StockName string
	Quantity  string
	Amount    string
	QtyRaw    int64
	AmtRaw    int64
}

type salesByCustomerReportData struct {
	Generated      bool
	Coverage       string
	Year           int
	Month          int
	FromDate       string
	ToDate         string
	PaperSize      string
	PaperClass     string
	Title          string
	RangeLabel     string
	CurrentPage    int
	TotalPages     int
	Categories     []salesByCustomerCategoryGroup
	TotalQuantity  string
	TotalAmount    string
	TotalQtyRaw    int64
	TotalAmountRaw int64
}

type salesByCustomerSummaryByItemReportData struct {
	Generated      bool
	Coverage       string
	Year           int
	Month          int
	FromDate       string
	ToDate         string
	PaperSize      string
	PaperClass     string
	Title          string
	RangeLabel     string
	CurrentPage    int
	TotalPages     int
	Categories     []salesByCustomerSummaryByItemCategoryGroup
	TotalQuantity  string
	TotalAmount    string
	TotalQtyRaw    int64
	TotalAmountRaw int64
}

type salesByCustomerSummaryByItemCategoryGroup struct {
	Category  string
	Customers []salesByCustomerSummaryByItemCustomerGroup
	TotalQty  string
	TotalAmt  string
	QtyRaw    int64
	AmtRaw    int64
}

type salesByCustomerSummaryByItemCustomerGroup struct {
	Customer string
	Rows     []salesByCustomerSummaryByItemLine
	TotalQty string
	TotalAmt string
	QtyRaw   int64
	AmtRaw   int64
}

type salesByCustomerSummaryByItemLine struct {
	Code     string
	Stock    string
	Quantity string
	Price    string
	Amount   string
	QtyRaw   int64
	PriceRaw int64
	AmtRaw   int64
}

type salesByCustomerCategoryGroup struct {
	Category  string
	Customers []salesByCustomerGroup
	TotalQty  string
	TotalAmt  string
	QtyRaw    int64
	AmtRaw    int64
}

type salesByCustomerGroup struct {
	Customer string
	Rows     []salesByCustomerLine
	TotalQty string
	TotalAmt string
	QtyRaw   int64
	AmtRaw   int64
}

type salesByCustomerLine struct {
	Reference string
	Date      string
	Code      string
	Stock     string
	Quantity  string
	Price     string
	Amount    string
	QtyRaw    int64
	AmtRaw    int64
}

type salesByStockNameReportData struct {
	Generated      bool
	Coverage       string
	Year           int
	Month          int
	FromDate       string
	ToDate         string
	PaperSize      string
	PaperClass     string
	Title          string
	RangeLabel     string
	CurrentPage    int
	TotalPages     int
	Categories     []salesByStockNameCategoryGroup
	TotalQuantity  string
	TotalAmount    string
	TotalQtyRaw    int64
	TotalAmountRaw int64
}

type salesByStockNameCategoryGroup struct {
	Category string
	Stocks   []salesByStockNameGroup
	TotalQty string
	TotalAmt string
	QtyRaw   int64
	AmtRaw   int64
}

type salesByStockNameGroup struct {
	StockCode string
	StockName string
	Rows      []salesByStockNameLine
	TotalQty  string
	TotalAmt  string
	QtyRaw    int64
	AmtRaw    int64
}

type salesByStockNameLine struct {
	Reference string
	Date      string
	Customer  string
	Quantity  string
	Price     string
	Amount    string
	QtyRaw    int64
	AmtRaw    int64
}

type apLedgerReportData struct {
	Generated   bool
	ReportType  string
	Coverage    string
	Year        int
	Month       int
	FromDate    string
	ToDate      string
	PaperSize   string
	PaperClass  string
	Title       string
	RangeLabel  string
	AsOfLabel   string
	CurrentPage int
	TotalPages  int
	Suppliers   []string
	Groups      []apLedgerSupplierGroup
	SummaryRows []apLedgerSummaryRow
	AgingRows   []apLedgerAgingRow
	TotalDebit  string
	TotalCredit string
	TotalNet    string
	AgingTotals apLedgerAgingTotals
}

type apLedgerSupplierGroup struct {
	SupplierCode   string
	SupplierName   string
	Representative string
	Rows           []apLedgerLine
}

type apLedgerLine struct {
	Date      string
	Reference string
	Debit     string
	Credit    string
	Balance   string
}

type apLedgerSummaryRow struct {
	Code           string
	Company        string
	Representative string
	Balance        string
	BalanceRaw     int64
}

type apLedgerAgingRow struct {
	Company     string
	LastPayment string
	Bucket0     string
	Bucket31    string
	Bucket61    string
	Bucket90    string
	Balance     string
	Bucket0Raw  int64
	Bucket31Raw int64
	Bucket61Raw int64
	Bucket90Raw int64
	BalanceRaw  int64
}

type apLedgerAgingTotals struct {
	Bucket0  string
	Bucket31 string
	Bucket61 string
	Bucket90 string
	Balance  string
}

type arLedgerReportData struct {
	Generated   bool
	ReportType  string
	Coverage    string
	Year        int
	Month       int
	FromDate    string
	ToDate      string
	PaperSize   string
	PaperClass  string
	Title       string
	RangeLabel  string
	AsOfLabel   string
	CurrentPage int
	TotalPages  int
	Customers   []string
	Groups      []arLedgerCustomerGroup
	SummaryRows []arLedgerSummaryRow
	AgingRows   []arLedgerAgingRow
	TotalDebit  string
	TotalCredit string
	TotalNet    string
	AgingTotals arLedgerAgingTotals
}

type arLedgerCustomerGroup struct {
	CustomerCode   string
	CustomerName   string
	CreditTerm     string
	CreditLimit    string
	CurrentBalance string
	Rows           []arLedgerLine
}

type arLedgerLine struct {
	Date      string
	Reference string
	Debit     string
	Credit    string
	Balance   string
}

type arLedgerSummaryRow struct {
	Company    string
	Balance    string
	BalanceRaw int64
}

type arLedgerAgingRow struct {
	Company             string
	LastPayment         string
	Bucket0             string
	Bucket31            string
	Bucket61            string
	Bucket90            string
	Balance             string
	OutstandingCheck    string
	TotalBalance        string
	Bucket0Raw          int64
	Bucket31Raw         int64
	Bucket61Raw         int64
	Bucket90Raw         int64
	BalanceRaw          int64
	OutstandingCheckRaw int64
	TotalBalanceRaw     int64
}

type arLedgerAgingTotals struct {
	Bucket0          string
	Bucket31         string
	Bucket61         string
	Bucket90         string
	Balance          string
	OutstandingCheck string
	TotalBalance     string
}

type incomingCheckReportData struct {
	Generated     bool
	ReportType    string
	CutoffDate    string
	PaperSize     string
	PaperClass    string
	Title         string
	CutoffLabel   string
	CurrentPage   int
	TotalPages    int
	Payees        []string
	Groups        []incomingCheckPayeeGroup
	SummaryRows   []incomingCheckSummaryRow
	GrandTotal    string
	GrandTotalRaw int64
}

type incomingCheckPayeeGroup struct {
	Payee    string
	Months   []incomingCheckMonthGroup
	Total    string
	TotalRaw int64
}

type incomingCheckMonthGroup struct {
	Month    string
	Rows     []incomingCheckLine
	Total    string
	TotalRaw int64
}

type incomingCheckLine struct {
	Reference string
	Date      string
	Number    string
	BankName  string
	Amount    string
}

type incomingCheckSummaryRow struct {
	RecordNumber int
	Payee        string
	Total        string
	TotalRaw     int64
}

type outgoingCheckReportData struct {
	Generated     bool
	ReportType    string
	CutoffDate    string
	PaperSize     string
	PaperClass    string
	Title         string
	CutoffLabel   string
	CurrentPage   int
	TotalPages    int
	Payees        []string
	Groups        []outgoingCheckPayeeGroup
	SummaryRows   []outgoingCheckSummaryRow
	GrandTotal    string
	GrandTotalRaw int64
}

type outgoingCheckPayeeGroup struct {
	Payee    string
	Months   []outgoingCheckMonthGroup
	Total    string
	TotalRaw int64
}

type outgoingCheckMonthGroup struct {
	Month    string
	Rows     []outgoingCheckLine
	Total    string
	TotalRaw int64
}

type outgoingCheckLine struct {
	Reference string
	Date      string
	Number    string
	BankName  string
	Amount    string
}

type outgoingCheckSummaryRow struct {
	RecordNumber int
	Payee        string
	Total        string
	TotalRaw     int64
}

type expenseReportData struct {
	Generated     bool
	ReportType    string
	Coverage      string
	Year          int
	Month         int
	FromDate      string
	ToDate        string
	PaperSize     string
	PaperClass    string
	Title         string
	RangeLabel    string
	CurrentPage   int
	TotalPages    int
	Categories    []expenseReportCategory
	Pages         []expenseReportPage
	SummaryRows   []expenseReportSummaryRow
	GrandTotal    string
	GrandTotalRaw int64
}

type expenseReportCategory struct {
	ID   string
	Code string
	Name string
}

type expenseReportPage struct {
	Number     int
	Categories []expenseReportCategory
	Rows       []expenseReportDetailRow
}

type expenseReportDetailRow struct {
	Date  string
	Cells []expenseReportAmountCell
	Total expenseReportAmountCell
}

type expenseReportAmountCell struct {
	Cash     string
	Check    string
	Total    string
	CashRaw  int64
	CheckRaw int64
	TotalRaw int64
}

type expenseReportSummaryRow struct {
	Code     string
	Name     string
	Total    string
	TotalRaw int64
}

type incomeStatementReportData struct {
	Generated          bool
	Coverage           string
	Year               int
	Month              int
	FromDate           string
	ToDate             string
	PaperSize          string
	PaperClass         string
	Title              string
	RangeLabel         string
	CurrentPage        int
	TotalPages         int
	CashSales          string
	ChargeSales        string
	TotalSales         string
	SalesReturn        string
	NetSales           string
	BeginningInventory string
	Purchases          []incomeStatementLine
	TotalPurchases     string
	Withdrawals        []incomeStatementLine
	TotalWithdrawals   string
	NetPurchases       string
	GoodsAvailable     string
	EndingInventory    string
	TotalCostOfSales   string
	GrossProfit        string
	OperatingExpenses  []incomeStatementLine
	TotalExpenses      string
	IncomeBeforeOther  string
	OtherIncome        []incomeStatementLine
	TotalOtherIncome   string
	NetIncome          string

	CashSalesRaw          int64
	ChargeSalesRaw        int64
	TotalSalesRaw         int64
	SalesReturnRaw        int64
	NetSalesRaw           int64
	BeginningInventoryRaw int64
	TotalPurchasesRaw     int64
	TotalWithdrawalsRaw   int64
	NetPurchasesRaw       int64
	GoodsAvailableRaw     int64
	EndingInventoryRaw    int64
	TotalCostOfSalesRaw   int64
	GrossProfitRaw        int64
	TotalExpensesRaw      int64
	IncomeBeforeOtherRaw  int64
	TotalOtherIncomeRaw   int64
	NetIncomeRaw          int64
}

type incomeStatementLine struct {
	Label  string
	Amount string
	Raw    int64
}

type incentiveReportData struct {
	Generated      bool
	Coverage       string
	Year           int
	Month          int
	FromDate       string
	ToDate         string
	PaperSize      string
	PaperClass     string
	Title          string
	RangeLabel     string
	CurrentPage    int
	TotalPages     int
	Rows           []incentiveReportLine
	TotalQty       string
	TotalVIP       string
	TotalAPS       string
	TotalTakals    string
	TotalFarm      string
	GrandTotal     string
	TotalQtyRaw    int64
	TotalVIPRaw    int64
	TotalAPSRaw    int64
	TotalTakalsRaw int64
	TotalFarmRaw   int64
	GrandTotalRaw  int64
}

type incentiveReportLine struct {
	AgriPost  string
	Qty       string
	VIP       string
	APS       string
	Takals    string
	Farm      string
	QtyRaw    int64
	VIPRaw    int64
	APSRaw    int64
	TakalsRaw int64
	FarmRaw   int64
}

type dailySalesCollectionReportData struct {
	Generated         bool
	ReportDate        string
	PaperSize         string
	PaperClass        string
	Title             string
	DateLabel         string
	CurrentPage       int
	TotalPages        int
	Sections          []dailySalesCollectionSection
	CashSales         string
	ChargeSales       string
	CashReceipts      string
	Disbursements     string
	CheckDeposits     string
	TotalCashRemit    string
	TotalRemit        string
	CashSalesRaw      int64
	ChargeSalesRaw    int64
	CashReceiptsRaw   int64
	DisbursementsRaw  int64
	CheckDepositsRaw  int64
	TotalCashRemitRaw int64
	TotalRemitRaw     int64
}

type dailySalesCollectionSection struct {
	Key      string
	Title    string
	Rows     []dailySalesCollectionLine
	Total    string
	TotalRaw int64
}

type dailySalesCollectionLine struct {
	Name      string
	Reference string
	Amount    string
	Raw       int64
}

type stockSalesTransferReportData struct {
	Generated        bool
	Coverage         string
	Year             int
	Month            int
	FromDate         string
	ToDate           string
	PaperSize        string
	PaperClass       string
	Title            string
	RangeLabel       string
	CurrentPage      int
	TotalPages       int
	Categories       []stockSalesTransferCategory
	TotalSales       string
	TotalTransfer    string
	GrandTotal       string
	TotalSalesRaw    int64
	TotalTransferRaw int64
	GrandTotalRaw    int64
}

type stockSalesTransferCategory struct {
	Name             string
	Rows             []stockSalesTransferLine
	TotalSales       string
	TotalTransfer    string
	GrandTotal       string
	TotalSalesRaw    int64
	TotalTransferRaw int64
	GrandTotalRaw    int64
}

type stockSalesTransferLine struct {
	StockCode   string
	StockName   string
	SalesQty    string
	TransferQty string
	TotalQty    string
	SalesRaw    int64
	TransferRaw int64
	TotalRaw    int64
}

type stockSalesTransferAmountReportData struct {
	Generated               bool
	Coverage                string
	Year                    int
	Month                   int
	FromDate                string
	ToDate                  string
	PaperSize               string
	PaperClass              string
	Title                   string
	RangeLabel              string
	CurrentPage             int
	TotalPages              int
	Rows                    []stockSalesTransferAmountLine
	TotalCashSales          string
	TotalChargeSales        string
	TotalSales              string
	TotalTransfer           string
	GrandTotal              string
	TotalSalesMarkup        string
	TotalSalesMarkupPercent string
	TotalTransferMarkup     string
	TotalTransferMarkupPct  string
	TotalCashSalesRaw       int64
	TotalChargeSalesRaw     int64
	TotalSalesRaw           int64
	TotalTransferRaw        int64
	GrandTotalRaw           int64
	TotalSalesMarkupRaw     int64
	TotalTransferMarkupRaw  int64
}

type stockSalesTransferAmountLine struct {
	Category              string
	CashSales             string
	ChargeSales           string
	TotalSales            string
	Transfer              string
	Total                 string
	SalesMarkup           string
	SalesMarkupPercent    string
	TransferMarkup        string
	TransferMarkupPercent string
	CashSalesRaw          int64
	ChargeSalesRaw        int64
	TotalSalesRaw         int64
	TransferRaw           int64
	TotalRaw              int64
	SalesMarkupRaw        int64
	TransferMarkupRaw     int64
}

type stockTransferSummaryReportData struct {
	Generated      bool
	Coverage       string
	Year           int
	Month          int
	FromDate       string
	ToDate         string
	PaperSize      string
	PaperClass     string
	Title          string
	RangeLabel     string
	CurrentPage    int
	TotalPages     int
	Categories     []stockTransferSummaryCategory
	TotalQuantity  string
	TotalAmount    string
	TotalQtyRaw    int64
	TotalAmountRaw int64
}

type stockTransferSummaryCategory struct {
	Name          string
	Branches      []stockTransferSummaryBranch
	TotalQuantity string
	TotalAmount   string
	TotalQtyRaw   int64
	TotalAmtRaw   int64
}

type stockTransferSummaryBranch struct {
	Name          string
	Rows          []stockTransferSummaryLine
	TotalQuantity string
	TotalAmount   string
	TotalQtyRaw   int64
	TotalAmtRaw   int64
}

type stockTransferSummaryLine struct {
	Reference string
	Date      string
	Code      string
	StockName string
	Quantity  string
	Amount    string
	QtyRaw    int64
	AmtRaw    int64
}

type stockTransferSummaryByItemReportData struct {
	Generated      bool
	Coverage       string
	Year           int
	Month          int
	FromDate       string
	ToDate         string
	PaperSize      string
	PaperClass     string
	Title          string
	RangeLabel     string
	CurrentPage    int
	TotalPages     int
	Categories     []stockTransferSummaryByItemCategory
	TotalQuantity  string
	TotalAmount    string
	TotalQtyRaw    int64
	TotalAmountRaw int64
}

type stockTransferSummaryByItemCategory struct {
	Name          string
	Rows          []stockTransferSummaryByItemLine
	TotalQuantity string
	TotalAmount   string
	TotalQtyRaw   int64
	TotalAmtRaw   int64
}

type stockTransferSummaryByItemLine struct {
	Code      string
	StockName string
	Quantity  string
	Amount    string
	QtyRaw    int64
	AmtRaw    int64
}

type stockTransferByStockNameReportData struct {
	Generated      bool
	Coverage       string
	Year           int
	Month          int
	FromDate       string
	ToDate         string
	PaperSize      string
	PaperClass     string
	Title          string
	RangeLabel     string
	CurrentPage    int
	TotalPages     int
	Categories     []stockTransferByStockNameCategory
	TotalQuantity  string
	TotalAmount    string
	TotalQtyRaw    int64
	TotalAmountRaw int64
}

type stockTransferByStockNameCategory struct {
	Name          string
	Stocks        []stockTransferByStockNameStock
	TotalQuantity string
	TotalAmount   string
	TotalQtyRaw   int64
	TotalAmtRaw   int64
}

type stockTransferByStockNameStock struct {
	StockCode     string
	StockName     string
	Branches      []stockTransferByStockNameBranch
	TotalQuantity string
	TotalAmount   string
	TotalQtyRaw   int64
	TotalAmtRaw   int64
}

type stockTransferByStockNameBranch struct {
	Name          string
	Rows          []stockTransferByStockNameLine
	TotalQuantity string
	TotalAmount   string
	TotalQtyRaw   int64
	TotalAmtRaw   int64
}

type stockTransferByStockNameLine struct {
	Reference  string
	Date       string
	TransferID string
	Branch     string
	Quantity   string
	Amount     string
	QtyRaw     int64
	AmtRaw     int64
}

type stockTransferByBranchReportData struct {
	Generated      bool
	Coverage       string
	Year           int
	Month          int
	FromDate       string
	ToDate         string
	PaperSize      string
	PaperClass     string
	Title          string
	RangeLabel     string
	CurrentPage    int
	TotalPages     int
	Branches       []stockTransferByBranchGroup
	TotalQuantity  string
	TotalAmount    string
	TotalQtyRaw    int64
	TotalAmountRaw int64
}

type stockTransferByBranchGroup struct {
	Name          string
	Categories    []stockTransferByBranchCategory
	TotalQuantity string
	TotalAmount   string
	TotalQtyRaw   int64
	TotalAmtRaw   int64
}

type stockTransferByBranchCategory struct {
	Name          string
	Rows          []stockTransferByBranchLine
	TotalQuantity string
	TotalAmount   string
	TotalQtyRaw   int64
	TotalAmtRaw   int64
}

type stockTransferByBranchLine struct {
	Reference string
	Date      string
	Code      string
	StockName string
	Quantity  string
	Amount    string
	QtyRaw    int64
	AmtRaw    int64
}

type stockTransferByEntryIDReportData struct {
	Generated      bool
	Coverage       string
	Year           int
	Month          int
	FromDate       string
	ToDate         string
	PaperSize      string
	PaperClass     string
	Title          string
	RangeLabel     string
	CurrentPage    int
	TotalPages     int
	Groups         []stockTransferByEntryIDGroup
	TotalQuantity  string
	TotalAmount    string
	TotalQtyRaw    int64
	TotalAmountRaw int64
}

type stockTransferByEntryIDGroup struct {
	EntryID       string
	Reference     string
	TransferID    string
	Date          string
	Branch        string
	Rows          []stockTransferByEntryIDLine
	TotalQuantity string
	TotalAmount   string
	TotalQtyRaw   int64
	TotalAmtRaw   int64
}

type stockTransferByEntryIDLine struct {
	Code     string
	Stock    string
	Branch   string
	Quantity string
	Amount   string
	QtyRaw   int64
	AmtRaw   int64
}

type stockTransferSummaryByEntryIDReportData struct {
	Generated      bool
	Coverage       string
	Year           int
	Month          int
	FromDate       string
	ToDate         string
	PaperSize      string
	PaperClass     string
	Title          string
	RangeLabel     string
	CurrentPage    int
	TotalPages     int
	Branches       []stockTransferSummaryByEntryIDBranch
	TotalQuantity  string
	TotalAmount    string
	NetTotal       string
	TotalQtyRaw    int64
	TotalAmountRaw int64
	NetTotalRaw    int64
}

type stockTransferSummaryByEntryIDBranch struct {
	Name          string
	Rows          []stockTransferSummaryByEntryIDLine
	TotalQuantity string
	TotalAmount   string
	NetTotal      string
	TotalQtyRaw   int64
	TotalAmtRaw   int64
	NetTotalRaw   int64
}

type stockTransferSummaryByEntryIDLine struct {
	EntryID       string
	EntryDate     string
	Remarks       string
	TotalQuantity string
	TotalAmount   string
	NetTotal      string
	TotalQtyRaw   int64
	TotalAmtRaw   int64
	NetTotalRaw   int64
}

type stockTransferMarkupByTransactionReportData struct {
	Generated          bool
	Coverage           string
	Year               int
	Month              int
	FromDate           string
	ToDate             string
	PaperSize          string
	PaperClass         string
	Title              string
	RangeLabel         string
	CurrentPage        int
	TotalPages         int
	Rows               []stockTransferMarkupByTransactionLine
	Pages              []stockTransferMarkupByTransactionPage
	TotalMarkup        string
	TotalMarkupRaw     int64
	TotalCapitalRaw    int64
	TotalMarkupPercent string
}

type stockTransferMarkupByTransactionLine struct {
	TransferDate  string
	EntryID       string
	TransferTo    string
	ReceiptNo     string
	ItemGroup     string
	Markup        string
	MarkupPercent string
	MarkupRaw     int64
	CapitalRaw    int64
	Negative      bool
}

type stockTransferMarkupByTransactionPage struct {
	Number int
	Rows   []stockTransferMarkupByTransactionLine
	Last   bool
}

type stockLedgerReportData struct {
	Generated   bool
	Coverage    string
	Year        int
	Month       int
	FromDate    string
	ToDate      string
	PaperSize   string
	PaperClass  string
	Title       string
	RangeLabel  string
	CurrentPage int
	TotalPages  int
	Categories  []stockLedgerCategory
}

type stockLedgerCategory struct {
	Name   string
	Stocks []stockLedgerStockGroup
}

type stockLedgerStockGroup struct {
	StockCode string
	StockName string
	Rows      []stockLedgerLine
}

type stockLedgerLine struct {
	Reference string
	Date      string
	Company   string
	Debit     string
	Credit    string
	Balance   string
}

type stockAgingReportData struct {
	Generated    bool
	CutoffDate   string
	PaperSize    string
	PaperClass   string
	Title        string
	CutoffLabel  string
	CurrentPage  int
	TotalPages   int
	BucketLabels []string
	Categories   []stockAgingCategory
	Totals       stockAgingTotals
}

type stockReorderPointReportData struct {
	Generated   bool
	CutoffDate  string
	PaperSize   string
	PaperClass  string
	Title       string
	CutoffLabel string
	CurrentPage int
	TotalPages  int
	Categories  []stockReorderPointCategory
}

type stockSummaryReportData struct {
	Generated      bool
	CutoffDate     string
	PaperSize      string
	PaperClass     string
	Title          string
	CutoffLabel    string
	CurrentPage    int
	TotalPages     int
	Categories     []stockSummaryCategory
	GrandSOH       string
	GrandAmount    string
	GrandSOHRaw    int64
	GrandAmountRaw int64
}

type stockAgingCategory struct {
	Name   string
	Rows   []stockAgingLine
	Totals stockAgingTotals
}

type stockAgingLine struct {
	StockCode string
	StockName string
	Buckets   []string
	Raw       []int64
}

type stockAgingTotals struct {
	Buckets []string
	Raw     []int64
}

type stockReorderPointCategory struct {
	Name string
	Rows []stockReorderPointLine
}

type stockReorderPointLine struct {
	StockCode    string
	StockName    string
	SOH          string
	MinInventory string
	Deficit      string
	DeficitRaw   int64
}

type stockSummaryCategory struct {
	Name        string
	Rows        []stockSummaryLine
	TotalSOH    string
	Amount      string
	TotalSOHRaw int64
	AmountRaw   int64
}

type stockSummaryLine struct {
	StockCode   string
	StockName   string
	SOH         string
	UnitCost    string
	Amount      string
	SOHRaw      int64
	UnitCostRaw int64
	AmountRaw   int64
}

type dailyDueCheckReportData struct {
	Generated     bool
	CutoffDate    string
	PaperSize     string
	PaperClass    string
	Title         string
	CutoffLabel   string
	CurrentPage   int
	TotalPages    int
	Groups        []dailyDueCheckDateGroup
	GrandTotal    string
	GrandTotalRaw int64
}

type dailyDueCheckDateGroup struct {
	CheckDate string
	DateKey   string
	Rows      []dailyDueCheckLine
	Total     string
	TotalRaw  int64
}

type dailyDueCheckLine struct {
	ClientName  string
	CheckNumber string
	BankName    string
	Amount      string
	Raw         int64
}

type incomingCheckCalendarReportData struct {
	Generated     bool
	Year          int
	Month         int
	MonthName     string
	MonthTotal    string
	MonthTotalRaw int64
	PrevYear      int
	PrevMonth     int
	NextYear      int
	NextMonth     int
	Title         string
	Weekdays      []incomingCheckCalendarWeekday
	Days          []incomingCheckCalendarDay
}

type incomingCheckCalendarWeekday struct {
	Name    string
	Weekend string
}

type incomingCheckCalendarDay struct {
	Blank    bool
	Day      int
	DateKey  string
	DateText string
	Weekend  string
	Total    string
	TotalRaw int64
	Rows     []incomingCheckCalendarLine
}

type incomingCheckCalendarLine struct {
	Payee     string
	Reference string
	Number    string
	BankName  string
	Amount    string
	Raw       int64
}

func (a *App) purchasesSummaryReport(w http.ResponseWriter, r *http.Request) {
	report := a.defaultPurchaseReportData(r)
	if r.URL.Query().Get("run") != "1" {
		a.render(w, r, "purchases_summary_report.gohtml", viewData{Title: "Purchases Summary", PurchaseReport: report})
		return
	}

	report.ReportType = normalizedReportType(r.URL.Query().Get("report_type"))
	report.Coverage = normalizedCoverage(r.URL.Query().Get("coverage"))
	report.PaperSize = normalizedPaperSize(r.URL.Query().Get("paper_size"))
	report.PaperClass = "report-paper-size-" + report.PaperSize
	report.Year = listYear(r, a.now)
	report.Month = boundedInt(r.URL.Query().Get("month"), 1, 12, report.Month)
	report.FromDate = strings.TrimSpace(r.URL.Query().Get("from_date"))
	if report.Coverage == "range" {
		report.ToDate = strings.TrimSpace(r.URL.Query().Get("range_to_date"))
	} else {
		report.ToDate = strings.TrimSpace(r.URL.Query().Get("to_date"))
	}
	from, to := reportDateRange(report, a.now)
	report.FromDate = from.Format("2006-01-02")
	report.ToDate = to.Format("2006-01-02")
	report.RangeLabel = purchaseRangeLabel(from, to)
	report.Generated = true

	rows, err := a.store.PurchaseReportRows(r.Context(), from, to)
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	report.build(rows)
	a.render(w, r, "purchases_summary_report.gohtml", viewData{Title: "Purchases Summary", PurchaseReport: report})
}

func (a *App) purchasesByDRNumberReport(w http.ResponseWriter, r *http.Request) {
	report := a.defaultPurchaseByDRNumberReportData(r)
	if r.URL.Query().Get("run") != "1" {
		a.render(w, r, "purchases_by_dr_number_report.gohtml", viewData{Title: "Purchases by DR Number", PurchaseByDRReport: report})
		return
	}

	report.Coverage = normalizedCoverage(r.URL.Query().Get("coverage"))
	report.PaperSize = normalizedPaperSize(r.URL.Query().Get("paper_size"))
	if report.PaperSize == "letter-landscape" {
		report.PaperSize = "letter"
	}
	report.PaperClass = "report-paper-size-" + report.PaperSize
	report.Year = listYear(r, a.now)
	report.Month = boundedInt(r.URL.Query().Get("month"), 1, 12, report.Month)
	report.FromDate = strings.TrimSpace(r.URL.Query().Get("from_date"))
	if report.Coverage == "range" {
		report.ToDate = strings.TrimSpace(r.URL.Query().Get("range_to_date"))
	} else {
		report.ToDate = strings.TrimSpace(r.URL.Query().Get("to_date"))
	}
	from, to := purchaseByDRDateRange(report, a.now)
	report.FromDate = from.Format("2006-01-02")
	report.ToDate = to.Format("2006-01-02")
	report.RangeLabel = purchaseRangeLabel(from, to)
	report.Generated = true

	rows, err := a.store.PurchaseByDRNumberReportRows(r.Context(), from, to)
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	report.build(rows)
	a.render(w, r, "purchases_by_dr_number_report.gohtml", viewData{Title: "Purchases by DR Number", PurchaseByDRReport: report})
}

func (a *App) purchasesByStockCodeReport(w http.ResponseWriter, r *http.Request) {
	report := a.defaultPurchaseByStockCodeReportData(r)
	if r.URL.Query().Get("run") != "1" {
		a.render(w, r, "purchases_by_stock_code_report.gohtml", viewData{Title: "Purchases by Stock Code", PurchaseByStockReport: report})
		return
	}

	report.Coverage = normalizedCoverage(r.URL.Query().Get("coverage"))
	report.PaperSize = normalizedPaperSize(r.URL.Query().Get("paper_size"))
	if report.PaperSize == "letter-landscape" {
		report.PaperSize = "letter"
	}
	report.PaperClass = "report-paper-size-" + report.PaperSize
	report.Year = listYear(r, a.now)
	report.Month = boundedInt(r.URL.Query().Get("month"), 1, 12, report.Month)
	report.FromDate = strings.TrimSpace(r.URL.Query().Get("from_date"))
	if report.Coverage == "range" {
		report.ToDate = strings.TrimSpace(r.URL.Query().Get("range_to_date"))
	} else {
		report.ToDate = strings.TrimSpace(r.URL.Query().Get("to_date"))
	}
	from, to := purchaseByStockCodeDateRange(report, a.now)
	report.FromDate = from.Format("2006-01-02")
	report.ToDate = to.Format("2006-01-02")
	report.RangeLabel = purchaseRangeLabel(from, to)
	report.Generated = true

	rows, err := a.store.PurchaseByStockCodeReportRows(r.Context(), from, to)
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	report.build(rows)
	a.render(w, r, "purchases_by_stock_code_report.gohtml", viewData{Title: "Purchases by Stock Code", PurchaseByStockReport: report})
}

func (a *App) purchasesBySupplierReport(w http.ResponseWriter, r *http.Request) {
	report := a.defaultPurchaseBySupplierReportData(r)
	if r.URL.Query().Get("run") != "1" {
		a.render(w, r, "purchases_by_supplier_report.gohtml", viewData{Title: "Purchases by Supplier", PurchaseBySupplierReport: report})
		return
	}

	report.Coverage = normalizedCoverage(r.URL.Query().Get("coverage"))
	report.PaperSize = normalizedPaperSize(r.URL.Query().Get("paper_size"))
	if report.PaperSize == "letter-landscape" {
		report.PaperSize = "letter"
	}
	report.PaperClass = "report-paper-size-" + report.PaperSize
	report.Year = listYear(r, a.now)
	report.Month = boundedInt(r.URL.Query().Get("month"), 1, 12, report.Month)
	report.FromDate = strings.TrimSpace(r.URL.Query().Get("from_date"))
	if report.Coverage == "range" {
		report.ToDate = strings.TrimSpace(r.URL.Query().Get("range_to_date"))
	} else {
		report.ToDate = strings.TrimSpace(r.URL.Query().Get("to_date"))
	}
	from, to := purchaseBySupplierDateRange(report, a.now)
	report.FromDate = from.Format("2006-01-02")
	report.ToDate = to.Format("2006-01-02")
	report.RangeLabel = purchaseRangeLabel(from, to)
	report.Generated = true

	rows, err := a.store.PurchaseBySupplierReportRows(r.Context(), from, to)
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	report.build(rows)
	a.render(w, r, "purchases_by_supplier_report.gohtml", viewData{Title: "Purchases by Supplier", PurchaseBySupplierReport: report})
}

func (a *App) salesSummaryReport(w http.ResponseWriter, r *http.Request) {
	report := a.defaultSalesReportData(r)
	if r.URL.Query().Get("run") != "1" {
		a.render(w, r, "sales_summary_report.gohtml", viewData{Title: "Sales Summary", SalesReport: report})
		return
	}

	report.ReportType = normalizedReportType(r.URL.Query().Get("report_type"))
	report.Coverage = normalizedCoverage(r.URL.Query().Get("coverage"))
	report.PaperSize = normalizedPaperSize(r.URL.Query().Get("paper_size"))
	report.PaperClass = "report-paper-size-" + report.PaperSize
	report.Year = listYear(r, a.now)
	report.Month = boundedInt(r.URL.Query().Get("month"), 1, 12, report.Month)
	report.FromDate = strings.TrimSpace(r.URL.Query().Get("from_date"))
	if report.Coverage == "range" {
		report.ToDate = strings.TrimSpace(r.URL.Query().Get("range_to_date"))
	} else {
		report.ToDate = strings.TrimSpace(r.URL.Query().Get("to_date"))
	}
	from, to := salesReportDateRange(report, a.now)
	report.FromDate = from.Format("2006-01-02")
	report.ToDate = to.Format("2006-01-02")
	report.RangeLabel = salesRangeLabel(from, to)
	report.Generated = true

	rows, err := a.store.SalesReportRows(r.Context(), from, to)
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	report.build(rows)
	a.render(w, r, "sales_summary_report.gohtml", viewData{Title: "Sales Summary", SalesReport: report})
}

func (a *App) salesByORCIDRNumberReport(w http.ResponseWriter, r *http.Request) {
	report := a.defaultSalesByORCIDRNumberReportData(r)
	if r.URL.Query().Get("run") != "1" {
		a.render(w, r, "sales_by_or_ci_dr_number_report.gohtml", viewData{Title: "Sales by OR/CI/DR Number", SalesByORCIDRReport: report})
		return
	}

	report.Coverage = normalizedCoverage(r.URL.Query().Get("coverage"))
	report.PaperSize = normalizedPaperSize(r.URL.Query().Get("paper_size"))
	if report.PaperSize == "letter-landscape" {
		report.PaperSize = "letter"
	}
	report.PaperClass = "report-paper-size-" + report.PaperSize
	report.Year = listYear(r, a.now)
	report.Month = boundedInt(r.URL.Query().Get("month"), 1, 12, report.Month)
	report.FromDate = strings.TrimSpace(r.URL.Query().Get("from_date"))
	if report.Coverage == "range" {
		report.ToDate = strings.TrimSpace(r.URL.Query().Get("range_to_date"))
	} else {
		report.ToDate = strings.TrimSpace(r.URL.Query().Get("to_date"))
	}
	from, to := salesByORCIDRDateRange(report, a.now)
	report.FromDate = from.Format("2006-01-02")
	report.ToDate = to.Format("2006-01-02")
	report.RangeLabel = salesRangeLabel(from, to)
	report.Generated = true

	rows, err := a.store.SalesByORCIDRNumberReportRows(r.Context(), from, to)
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	report.build(rows)
	a.render(w, r, "sales_by_or_ci_dr_number_report.gohtml", viewData{Title: "Sales by OR/CI/DR Number", SalesByORCIDRReport: report})
}

func (a *App) salesMarkupByTransactionReport(w http.ResponseWriter, r *http.Request) {
	report := a.defaultSalesMarkupByTransactionReportData(r)
	if r.URL.Query().Get("run") != "1" {
		a.render(w, r, "sales_markup_by_transaction_report.gohtml", viewData{Title: "Sales Markup by Transaction", SalesMarkupReport: report})
		return
	}

	report.Coverage = normalizedCoverage(r.URL.Query().Get("coverage"))
	report.PaperSize = normalizedPaperSize(r.URL.Query().Get("paper_size"))
	if report.PaperSize == "letter-landscape" {
		report.PaperSize = "letter"
	}
	report.PaperClass = "report-paper-size-" + report.PaperSize
	report.Year = listYear(r, a.now)
	report.Month = boundedInt(r.URL.Query().Get("month"), 1, 12, report.Month)
	report.FromDate = strings.TrimSpace(r.URL.Query().Get("from_date"))
	if report.Coverage == "range" {
		report.ToDate = strings.TrimSpace(r.URL.Query().Get("range_to_date"))
	} else {
		report.ToDate = strings.TrimSpace(r.URL.Query().Get("to_date"))
	}
	from, to := salesMarkupByTransactionDateRange(report, a.now)
	report.FromDate = from.Format("2006-01-02")
	report.ToDate = to.Format("2006-01-02")
	report.RangeLabel = salesRangeLabel(from, to)
	report.Generated = true

	rows, err := a.store.SalesMarkupByTransactionReportRows(r.Context(), from, to)
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	report.build(rows)
	a.render(w, r, "sales_markup_by_transaction_report.gohtml", viewData{Title: "Sales Markup by Transaction", SalesMarkupReport: report})
}

func (a *App) salesSummaryByItemReport(w http.ResponseWriter, r *http.Request) {
	report := a.defaultSalesSummaryByItemReportData(r)
	if r.URL.Query().Get("run") != "1" {
		a.render(w, r, "sales_summary_by_item_report.gohtml", viewData{Title: "Sales Summary By Item", SalesSummaryByItemReport: report})
		return
	}

	report.Coverage = normalizedCoverage(r.URL.Query().Get("coverage"))
	report.PaperSize = normalizedPaperSize(r.URL.Query().Get("paper_size"))
	if report.PaperSize == "letter-landscape" {
		report.PaperSize = "letter"
	}
	report.PaperClass = "report-paper-size-" + report.PaperSize
	report.Year = listYear(r, a.now)
	report.Month = boundedInt(r.URL.Query().Get("month"), 1, 12, report.Month)
	report.FromDate = strings.TrimSpace(r.URL.Query().Get("from_date"))
	if report.Coverage == "range" {
		report.ToDate = strings.TrimSpace(r.URL.Query().Get("range_to_date"))
	} else {
		report.ToDate = strings.TrimSpace(r.URL.Query().Get("to_date"))
	}
	from, to := salesSummaryByItemDateRange(report, a.now)
	report.FromDate = from.Format("2006-01-02")
	report.ToDate = to.Format("2006-01-02")
	report.RangeLabel = salesRangeLabel(from, to)
	report.Generated = true

	rows, err := a.store.SalesByStockNameReportRows(r.Context(), from, to)
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	report.build(rows)
	a.render(w, r, "sales_summary_by_item_report.gohtml", viewData{Title: "Sales Summary By Item", SalesSummaryByItemReport: report})
}

func (a *App) salesByCustomerReport(w http.ResponseWriter, r *http.Request) {
	report := a.defaultSalesByCustomerReportData(r)
	if r.URL.Query().Get("run") != "1" {
		a.render(w, r, "sales_by_customer_report.gohtml", viewData{Title: "Sales by Customer", SalesByCustomerReport: report})
		return
	}

	report.Coverage = normalizedCoverage(r.URL.Query().Get("coverage"))
	report.PaperSize = normalizedPaperSize(r.URL.Query().Get("paper_size"))
	if report.PaperSize == "letter-landscape" {
		report.PaperSize = "letter"
	}
	report.PaperClass = "report-paper-size-" + report.PaperSize
	report.Year = listYear(r, a.now)
	report.Month = boundedInt(r.URL.Query().Get("month"), 1, 12, report.Month)
	report.FromDate = strings.TrimSpace(r.URL.Query().Get("from_date"))
	if report.Coverage == "range" {
		report.ToDate = strings.TrimSpace(r.URL.Query().Get("range_to_date"))
	} else {
		report.ToDate = strings.TrimSpace(r.URL.Query().Get("to_date"))
	}
	from, to := salesByCustomerDateRange(report, a.now)
	report.FromDate = from.Format("2006-01-02")
	report.ToDate = to.Format("2006-01-02")
	report.RangeLabel = salesRangeLabel(from, to)
	report.Generated = true

	rows, err := a.store.SalesByCustomerReportRows(r.Context(), from, to)
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	report.build(rows)
	a.render(w, r, "sales_by_customer_report.gohtml", viewData{Title: "Sales by Customer", SalesByCustomerReport: report})
}

func (a *App) salesByCustomerSummaryByItemReport(w http.ResponseWriter, r *http.Request) {
	report := a.defaultSalesByCustomerSummaryByItemReportData(r)
	if r.URL.Query().Get("run") != "1" {
		a.render(w, r, "sales_by_customer_summary_by_item_report.gohtml", viewData{Title: "Sales by Customer (Summary By Item)", SalesByCustomerSummaryByItemReport: report})
		return
	}

	report.Coverage = normalizedCoverage(r.URL.Query().Get("coverage"))
	report.PaperSize = normalizedPaperSize(r.URL.Query().Get("paper_size"))
	if report.PaperSize == "letter-landscape" {
		report.PaperSize = "letter"
	}
	report.PaperClass = "report-paper-size-" + report.PaperSize
	report.Year = listYear(r, a.now)
	report.Month = boundedInt(r.URL.Query().Get("month"), 1, 12, report.Month)
	report.FromDate = strings.TrimSpace(r.URL.Query().Get("from_date"))
	if report.Coverage == "range" {
		report.ToDate = strings.TrimSpace(r.URL.Query().Get("range_to_date"))
	} else {
		report.ToDate = strings.TrimSpace(r.URL.Query().Get("to_date"))
	}
	from, to := salesByCustomerSummaryByItemDateRange(report, a.now)
	report.FromDate = from.Format("2006-01-02")
	report.ToDate = to.Format("2006-01-02")
	report.RangeLabel = salesRangeLabel(from, to)
	report.Generated = true

	rows, err := a.store.SalesByCustomerReportRows(r.Context(), from, to)
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	report.build(rows)
	a.render(w, r, "sales_by_customer_summary_by_item_report.gohtml", viewData{Title: "Sales by Customer (Summary By Item)", SalesByCustomerSummaryByItemReport: report})
}

func (a *App) salesByStockNameReport(w http.ResponseWriter, r *http.Request) {
	report := a.defaultSalesByStockNameReportData(r)
	if r.URL.Query().Get("run") != "1" {
		a.render(w, r, "sales_by_stock_name_report.gohtml", viewData{Title: "Sales by Stock Name", SalesByStockNameReport: report})
		return
	}

	report.Coverage = normalizedCoverage(r.URL.Query().Get("coverage"))
	report.PaperSize = normalizedPaperSize(r.URL.Query().Get("paper_size"))
	if report.PaperSize == "letter-landscape" {
		report.PaperSize = "letter"
	}
	report.PaperClass = "report-paper-size-" + report.PaperSize
	report.Year = listYear(r, a.now)
	report.Month = boundedInt(r.URL.Query().Get("month"), 1, 12, report.Month)
	report.FromDate = strings.TrimSpace(r.URL.Query().Get("from_date"))
	if report.Coverage == "range" {
		report.ToDate = strings.TrimSpace(r.URL.Query().Get("range_to_date"))
	} else {
		report.ToDate = strings.TrimSpace(r.URL.Query().Get("to_date"))
	}
	from, to := salesByStockNameDateRange(report, a.now)
	report.FromDate = from.Format("2006-01-02")
	report.ToDate = to.Format("2006-01-02")
	report.RangeLabel = salesRangeLabel(from, to)
	report.Generated = true

	rows, err := a.store.SalesByStockNameReportRows(r.Context(), from, to)
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	report.build(rows)
	a.render(w, r, "sales_by_stock_name_report.gohtml", viewData{Title: "Sales by Stock Name", SalesByStockNameReport: report})
}

func (a *App) apLedgerReport(w http.ResponseWriter, r *http.Request) {
	report := a.defaultAPLedgerReportData(r)
	if r.URL.Query().Get("run") != "1" {
		a.render(w, r, "ap_ledger_report.gohtml", viewData{Title: "AP Ledger", APLedgerReport: report})
		return
	}

	report.ReportType = normalizedLedgerReportType(r.URL.Query().Get("report_type"))
	report.Coverage = normalizedCoverage(r.URL.Query().Get("coverage"))
	report.PaperSize = normalizedPaperSize(r.URL.Query().Get("paper_size"))
	report.PaperClass = "report-paper-size-" + report.PaperSize
	report.Year = listYear(r, a.now)
	report.Month = boundedInt(r.URL.Query().Get("month"), 1, 12, report.Month)
	report.FromDate = strings.TrimSpace(r.URL.Query().Get("from_date"))
	if report.Coverage == "range" {
		report.ToDate = strings.TrimSpace(r.URL.Query().Get("range_to_date"))
	} else {
		report.ToDate = strings.TrimSpace(r.URL.Query().Get("to_date"))
	}
	from, to := apLedgerReportDateRange(report, a.now)
	report.FromDate = from.Format("2006-01-02")
	report.ToDate = to.Format("2006-01-02")
	report.RangeLabel = apLedgerRangeLabel(from, to)
	if report.ReportType == "summary" {
		report.AsOfLabel = "Summary as of: " + to.Format("January 02, 2006")
	} else {
		report.AsOfLabel = "Report as of: " + to.Format("January 02, 2006")
	}
	report.Generated = true

	rows, err := a.store.APLedgerReportRows(r.Context(), from, to)
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	report.build(rows, from, to)
	a.render(w, r, "ap_ledger_report.gohtml", viewData{Title: "AP Ledger", APLedgerReport: report})
}

func (a *App) arLedgerReport(w http.ResponseWriter, r *http.Request) {
	report := a.defaultARLedgerReportData(r)
	if r.URL.Query().Get("run") != "1" {
		a.render(w, r, "ar_ledger_report.gohtml", viewData{Title: "AR Ledger", ARLedgerReport: report})
		return
	}

	report.ReportType = normalizedLedgerReportType(r.URL.Query().Get("report_type"))
	report.Coverage = normalizedCoverage(r.URL.Query().Get("coverage"))
	report.PaperSize = normalizedPaperSize(r.URL.Query().Get("paper_size"))
	report.PaperClass = "report-paper-size-" + report.PaperSize
	report.Year = listYear(r, a.now)
	report.Month = boundedInt(r.URL.Query().Get("month"), 1, 12, report.Month)
	report.FromDate = strings.TrimSpace(r.URL.Query().Get("from_date"))
	if report.Coverage == "range" {
		report.ToDate = strings.TrimSpace(r.URL.Query().Get("range_to_date"))
	} else {
		report.ToDate = strings.TrimSpace(r.URL.Query().Get("to_date"))
	}
	from, to := arLedgerReportDateRange(report, a.now)
	report.FromDate = from.Format("2006-01-02")
	report.ToDate = to.Format("2006-01-02")
	report.RangeLabel = arLedgerRangeLabel(from, to)
	if report.ReportType == "summary" {
		report.AsOfLabel = "Summary as of: " + to.Format("January 02, 2006")
	} else {
		report.AsOfLabel = "Report as of: " + to.Format("January 02, 2006")
	}
	report.Generated = true

	rows, err := a.store.ARLedgerReportRows(r.Context(), from, to)
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	report.build(rows, from, to)
	a.render(w, r, "ar_ledger_report.gohtml", viewData{Title: "AR Ledger", ARLedgerReport: report})
}

func (a *App) incomingCheckListReport(w http.ResponseWriter, r *http.Request) {
	report := a.defaultIncomingCheckReportData(r)
	if r.URL.Query().Get("run") != "1" {
		a.render(w, r, "incoming_check_list_report.gohtml", viewData{Title: "Incoming Check List", IncomingCheckReport: report})
		return
	}

	report.ReportType = normalizedIncomingCheckReportType(r.URL.Query().Get("report_type"))
	report.PaperSize = normalizedPaperSize(r.URL.Query().Get("paper_size"))
	report.PaperClass = "report-paper-size-" + report.PaperSize
	cutoff := parseReportDate(strings.TrimSpace(r.URL.Query().Get("cutoff_date")), a.now())
	report.CutoffDate = cutoff.Format("2006-01-02")
	report.CutoffLabel = "Check Date Cut-Off: " + cutoff.Format("02-Jan-2006")
	report.Generated = true

	rows, err := a.store.IncomingCheckReportRows(r.Context(), cutoff)
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	report.build(rows, cutoff)
	a.render(w, r, "incoming_check_list_report.gohtml", viewData{Title: "Incoming Check List", IncomingCheckReport: report})
}

func (a *App) outgoingCheckListReport(w http.ResponseWriter, r *http.Request) {
	report := a.defaultOutgoingCheckReportData(r)
	if r.URL.Query().Get("run") != "1" {
		a.render(w, r, "outgoing_check_list_report.gohtml", viewData{Title: "Outgoing Check List", OutgoingCheckReport: report})
		return
	}

	report.ReportType = normalizedIncomingCheckReportType(r.URL.Query().Get("report_type"))
	report.PaperSize = normalizedPaperSize(r.URL.Query().Get("paper_size"))
	report.PaperClass = "report-paper-size-" + report.PaperSize
	cutoff := parseReportDate(strings.TrimSpace(r.URL.Query().Get("cutoff_date")), a.now())
	report.CutoffDate = cutoff.Format("2006-01-02")
	report.CutoffLabel = "Check Date Cut-Off: " + cutoff.Format("02-Jan-2006")
	report.Generated = true

	rows, err := a.store.OutgoingCheckReportRows(r.Context(), cutoff)
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	report.build(rows, cutoff)
	a.render(w, r, "outgoing_check_list_report.gohtml", viewData{Title: "Outgoing Check List", OutgoingCheckReport: report})
}

func (a *App) expensesSummaryReport(w http.ResponseWriter, r *http.Request) {
	report := a.defaultExpenseReportData(r)
	if r.URL.Query().Get("run") != "1" {
		a.render(w, r, "expenses_summary_report.gohtml", viewData{Title: "Expenses Summary", ExpenseReport: report})
		return
	}

	report.ReportType = normalizedReportType(r.URL.Query().Get("report_type"))
	report.Coverage = normalizedCoverage(r.URL.Query().Get("coverage"))
	report.PaperSize = normalizedPaperSize(r.URL.Query().Get("paper_size"))
	if report.ReportType == "detailed" && report.PaperSize == "letter" {
		report.PaperSize = "letter-landscape"
	}
	report.PaperClass = "report-paper-size-" + report.PaperSize
	report.Year = listYear(r, a.now)
	report.Month = boundedInt(r.URL.Query().Get("month"), 1, 12, report.Month)
	report.FromDate = strings.TrimSpace(r.URL.Query().Get("from_date"))
	if report.Coverage == "range" {
		report.ToDate = strings.TrimSpace(r.URL.Query().Get("range_to_date"))
	} else {
		report.ToDate = strings.TrimSpace(r.URL.Query().Get("to_date"))
	}
	from, to := expenseReportDateRange(report, a.now)
	report.FromDate = from.Format("2006-01-02")
	report.ToDate = to.Format("2006-01-02")
	report.RangeLabel = expenseRangeLabel(from, to)
	report.Generated = true

	rows, err := a.store.ExpenseReportRows(r.Context(), from, to)
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	report.build(rows, to)
	a.render(w, r, "expenses_summary_report.gohtml", viewData{Title: "Expenses Summary", ExpenseReport: report})
}

func (a *App) incomeStatementReport(w http.ResponseWriter, r *http.Request) {
	report := a.defaultIncomeStatementReportData(r)
	if r.URL.Query().Get("run") != "1" {
		a.render(w, r, "income_statement_report.gohtml", viewData{Title: "Income Statement", IncomeStatementReport: report})
		return
	}

	report.Coverage = normalizedCoverage(r.URL.Query().Get("coverage"))
	report.PaperSize = normalizedPaperSize(r.URL.Query().Get("paper_size"))
	report.PaperClass = "report-paper-size-" + report.PaperSize
	report.Year = listYear(r, a.now)
	report.Month = boundedInt(r.URL.Query().Get("month"), 1, 12, report.Month)
	report.FromDate = strings.TrimSpace(r.URL.Query().Get("from_date"))
	if report.Coverage == "range" {
		report.ToDate = strings.TrimSpace(r.URL.Query().Get("range_to_date"))
	} else {
		report.ToDate = strings.TrimSpace(r.URL.Query().Get("to_date"))
	}
	from, to := incomeStatementDateRange(report, a.now)
	report.FromDate = from.Format("2006-01-02")
	report.ToDate = to.Format("2006-01-02")
	report.RangeLabel = incomeStatementRangeLabel(from, to)
	report.Generated = true

	rows, err := a.store.IncomeStatementRows(r.Context(), from, to)
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	report.build(rows)
	a.render(w, r, "income_statement_report.gohtml", viewData{Title: "Income Statement", IncomeStatementReport: report})
}

func (a *App) incentiveReport(w http.ResponseWriter, r *http.Request) {
	report := a.defaultIncentiveReportData(r)
	if r.URL.Query().Get("run") != "1" {
		a.render(w, r, "incentive_report.gohtml", viewData{Title: "Incentive Report", IncentiveReport: report})
		return
	}

	report.Coverage = normalizedCoverage(r.URL.Query().Get("coverage"))
	report.PaperSize = normalizedPaperSize(r.URL.Query().Get("paper_size"))
	report.PaperClass = "report-paper-size-" + report.PaperSize
	report.Year = listYear(r, a.now)
	report.Month = boundedInt(r.URL.Query().Get("month"), 1, 12, report.Month)
	report.FromDate = strings.TrimSpace(r.URL.Query().Get("from_date"))
	if report.Coverage == "range" {
		report.ToDate = strings.TrimSpace(r.URL.Query().Get("range_to_date"))
	} else {
		report.ToDate = strings.TrimSpace(r.URL.Query().Get("to_date"))
	}
	from, to := incentiveReportDateRange(report, a.now)
	report.FromDate = from.Format("2006-01-02")
	report.ToDate = to.Format("2006-01-02")
	report.RangeLabel = incentiveRangeLabel(from, to)
	report.Generated = true

	rows, err := a.store.IncentiveReportRows(r.Context(), from, to)
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	report.build(rows)
	a.render(w, r, "incentive_report.gohtml", viewData{Title: "Incentive Report", IncentiveReport: report})
}

func (a *App) dailySalesCollectionReport(w http.ResponseWriter, r *http.Request) {
	report := a.defaultDailySalesCollectionReportData()
	if r.URL.Query().Get("run") != "1" {
		a.render(w, r, "daily_sales_collection_report.gohtml", viewData{Title: "Daily Sales & Collection", DailySalesReport: report})
		return
	}

	report.PaperSize = normalizedPaperSize(r.URL.Query().Get("paper_size"))
	report.PaperClass = "report-paper-size-" + report.PaperSize
	reportDate := parseReportDate(strings.TrimSpace(r.URL.Query().Get("report_date")), a.now())
	report.ReportDate = reportDate.Format("2006-01-02")
	report.DateLabel = "Report Date: " + reportDate.Format("01/02/2006")
	report.Generated = true

	rows, err := a.store.DailySalesCollectionReportRows(r.Context(), reportDate)
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	report.build(rows)
	a.render(w, r, "daily_sales_collection_report.gohtml", viewData{Title: "Daily Sales & Collection", DailySalesReport: report})
}

func (a *App) dailyDueCheckReport(w http.ResponseWriter, r *http.Request) {
	report := a.defaultDailyDueCheckReportData()
	if r.URL.Query().Get("run") != "1" {
		a.render(w, r, "daily_due_check_report.gohtml", viewData{Title: "Daily Due Check", DailyDueCheckReport: report})
		return
	}

	report.PaperSize = normalizedPaperSize(r.URL.Query().Get("paper_size"))
	report.PaperClass = "report-paper-size-" + report.PaperSize
	cutoff := parseReportDate(strings.TrimSpace(r.URL.Query().Get("cutoff_date")), a.now())
	report.CutoffDate = cutoff.Format("2006-01-02")
	report.CutoffLabel = "Cut-Off Date: " + cutoff.Format("01/02/2006")
	report.Generated = true

	rows, err := a.store.IncomingCheckReportRows(r.Context(), cutoff)
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	report.build(dailyDueCheckRowsFromIncoming(rows), cutoff)
	a.render(w, r, "daily_due_check_report.gohtml", viewData{Title: "Daily Due Check", DailyDueCheckReport: report})
}

func (a *App) stockSalesTransferReport(w http.ResponseWriter, r *http.Request) {
	report := a.defaultStockSalesTransferReportData(r)
	if r.URL.Query().Get("run") != "1" {
		a.render(w, r, "stock_sales_transfer_report.gohtml", viewData{Title: "Stock Sales & Transfer", StockSalesTransfer: report})
		return
	}

	report.Coverage = normalizedCoverage(r.URL.Query().Get("coverage"))
	report.PaperSize = normalizedPaperSize(r.URL.Query().Get("paper_size"))
	if report.PaperSize == "letter" {
		report.PaperSize = "letter-landscape"
	}
	report.PaperClass = "report-paper-size-" + report.PaperSize
	report.Year = listYear(r, a.now)
	report.Month = boundedInt(r.URL.Query().Get("month"), 1, 12, report.Month)
	report.FromDate = strings.TrimSpace(r.URL.Query().Get("from_date"))
	if report.Coverage == "range" {
		report.ToDate = strings.TrimSpace(r.URL.Query().Get("range_to_date"))
	} else {
		report.ToDate = strings.TrimSpace(r.URL.Query().Get("to_date"))
	}
	from, to := stockSalesTransferReportDateRange(report, a.now)
	report.FromDate = from.Format("2006-01-02")
	report.ToDate = to.Format("2006-01-02")
	report.RangeLabel = stockSalesTransferRangeLabel(from, to)
	report.Generated = true

	rows, err := a.store.StockSalesTransferReportRows(r.Context(), from, to)
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	report.build(rows)
	a.render(w, r, "stock_sales_transfer_report.gohtml", viewData{Title: "Stock Sales & Transfer", StockSalesTransfer: report})
}

func (a *App) stockSalesTransferAmountReport(w http.ResponseWriter, r *http.Request) {
	report := a.defaultStockSalesTransferAmountReportData(r)
	if r.URL.Query().Get("run") != "1" {
		a.render(w, r, "stock_sales_transfer_amount_report.gohtml", viewData{Title: "Stk. Sales & Transfer Amount", StockSalesTransferAmt: report})
		return
	}

	report.Coverage = normalizedCoverage(r.URL.Query().Get("coverage"))
	report.PaperSize = normalizedPaperSize(r.URL.Query().Get("paper_size"))
	if report.PaperSize == "letter" {
		report.PaperSize = "letter-landscape"
	}
	report.PaperClass = "report-paper-size-" + report.PaperSize
	report.Year = listYear(r, a.now)
	report.Month = boundedInt(r.URL.Query().Get("month"), 1, 12, report.Month)
	report.FromDate = strings.TrimSpace(r.URL.Query().Get("from_date"))
	if report.Coverage == "range" {
		report.ToDate = strings.TrimSpace(r.URL.Query().Get("range_to_date"))
	} else {
		report.ToDate = strings.TrimSpace(r.URL.Query().Get("to_date"))
	}
	from, to := stockSalesTransferAmountReportDateRange(report, a.now)
	report.FromDate = from.Format("2006-01-02")
	report.ToDate = to.Format("2006-01-02")
	report.RangeLabel = stockSalesTransferRangeLabel(from, to)
	report.Generated = true

	rows, err := a.store.StockSalesTransferAmountReportRows(r.Context(), from, to)
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	report.build(rows)
	a.render(w, r, "stock_sales_transfer_amount_report.gohtml", viewData{Title: "Stk. Sales & Transfer Amount", StockSalesTransferAmt: report})
}

func (a *App) stockTransferSummaryReport(w http.ResponseWriter, r *http.Request) {
	report := a.defaultStockTransferSummaryReportData(r)
	if r.URL.Query().Get("run") != "1" {
		a.render(w, r, "stock_transfer_summary_report.gohtml", viewData{Title: "Stock Transfers Summary", StockTransferSummaryReport: report})
		return
	}

	report.Coverage = normalizedCoverage(r.URL.Query().Get("coverage"))
	report.PaperSize = normalizedPaperSize(r.URL.Query().Get("paper_size"))
	if report.PaperSize == "letter-landscape" {
		report.PaperSize = "letter"
	}
	report.PaperClass = "report-paper-size-" + report.PaperSize
	report.Year = listYear(r, a.now)
	report.Month = boundedInt(r.URL.Query().Get("month"), 1, 12, report.Month)
	report.FromDate = strings.TrimSpace(r.URL.Query().Get("from_date"))
	if report.Coverage == "range" {
		report.ToDate = strings.TrimSpace(r.URL.Query().Get("range_to_date"))
	} else {
		report.ToDate = strings.TrimSpace(r.URL.Query().Get("to_date"))
	}
	from, to := stockTransferSummaryDateRange(report, a.now)
	report.FromDate = from.Format("2006-01-02")
	report.ToDate = to.Format("2006-01-02")
	report.RangeLabel = stockTransferSummaryRangeLabel(from, to)
	report.Generated = true

	rows, err := a.store.StockTransferSummaryReportRows(r.Context(), from, to)
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	report.build(rows)
	a.render(w, r, "stock_transfer_summary_report.gohtml", viewData{Title: "Stock Transfers Summary", StockTransferSummaryReport: report})
}

func (a *App) stockTransferByStockNameReport(w http.ResponseWriter, r *http.Request) {
	report := a.defaultStockTransferByStockNameReportData(r)
	if r.URL.Query().Get("run") != "1" {
		a.render(w, r, "stock_transfer_by_stock_name_report.gohtml", viewData{Title: "Stock Transfers by Stock Name", StockTransferByStockNameReport: report})
		return
	}

	report.Coverage = normalizedCoverage(r.URL.Query().Get("coverage"))
	report.PaperSize = normalizedPaperSize(r.URL.Query().Get("paper_size"))
	if report.PaperSize == "letter-landscape" {
		report.PaperSize = "letter"
	}
	report.PaperClass = "report-paper-size-" + report.PaperSize
	report.Year = listYear(r, a.now)
	report.Month = boundedInt(r.URL.Query().Get("month"), 1, 12, report.Month)
	report.FromDate = strings.TrimSpace(r.URL.Query().Get("from_date"))
	if report.Coverage == "range" {
		report.ToDate = strings.TrimSpace(r.URL.Query().Get("range_to_date"))
	} else {
		report.ToDate = strings.TrimSpace(r.URL.Query().Get("to_date"))
	}
	from, to := stockTransferByStockNameDateRange(report, a.now)
	report.FromDate = from.Format("2006-01-02")
	report.ToDate = to.Format("2006-01-02")
	report.RangeLabel = stockTransferSummaryRangeLabel(from, to)
	report.Generated = true

	rows, err := a.store.StockTransferByStockNameReportRows(r.Context(), from, to)
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	report.build(rows)
	a.render(w, r, "stock_transfer_by_stock_name_report.gohtml", viewData{Title: "Stock Transfers by Stock Name", StockTransferByStockNameReport: report})
}

func (a *App) stockTransferByBranchReport(w http.ResponseWriter, r *http.Request) {
	report := a.defaultStockTransferByBranchReportData(r)
	if r.URL.Query().Get("run") != "1" {
		a.render(w, r, "stock_transfer_by_branch_report.gohtml", viewData{Title: "Stock Transfers by Branch", StockTransferByBranchReport: report})
		return
	}

	report.Coverage = normalizedCoverage(r.URL.Query().Get("coverage"))
	report.PaperSize = normalizedPaperSize(r.URL.Query().Get("paper_size"))
	if report.PaperSize == "letter-landscape" {
		report.PaperSize = "letter"
	}
	report.PaperClass = "report-paper-size-" + report.PaperSize
	report.Year = listYear(r, a.now)
	report.Month = boundedInt(r.URL.Query().Get("month"), 1, 12, report.Month)
	report.FromDate = strings.TrimSpace(r.URL.Query().Get("from_date"))
	if report.Coverage == "range" {
		report.ToDate = strings.TrimSpace(r.URL.Query().Get("range_to_date"))
	} else {
		report.ToDate = strings.TrimSpace(r.URL.Query().Get("to_date"))
	}
	from, to := stockTransferByBranchDateRange(report, a.now)
	report.FromDate = from.Format("2006-01-02")
	report.ToDate = to.Format("2006-01-02")
	report.RangeLabel = stockTransferSummaryRangeLabel(from, to)
	report.Generated = true

	rows, err := a.store.StockTransferByBranchReportRows(r.Context(), from, to)
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	report.build(rows)
	a.render(w, r, "stock_transfer_by_branch_report.gohtml", viewData{Title: "Stock Transfers by Branch", StockTransferByBranchReport: report})
}

func (a *App) stockTransferByEntryIDReport(w http.ResponseWriter, r *http.Request) {
	report := a.defaultStockTransferByEntryIDReportData(r)
	if r.URL.Query().Get("run") != "1" {
		a.render(w, r, "stock_transfer_by_entry_id_report.gohtml", viewData{Title: "Stock Transfers by Entry ID", StockTransferByEntryIDReport: report})
		return
	}

	report.Coverage = normalizedCoverage(r.URL.Query().Get("coverage"))
	report.PaperSize = normalizedPaperSize(r.URL.Query().Get("paper_size"))
	if report.PaperSize == "letter-landscape" {
		report.PaperSize = "letter"
	}
	report.PaperClass = "report-paper-size-" + report.PaperSize
	report.Year = listYear(r, a.now)
	report.Month = boundedInt(r.URL.Query().Get("month"), 1, 12, report.Month)
	report.FromDate = strings.TrimSpace(r.URL.Query().Get("from_date"))
	if report.Coverage == "range" {
		report.ToDate = strings.TrimSpace(r.URL.Query().Get("range_to_date"))
	} else {
		report.ToDate = strings.TrimSpace(r.URL.Query().Get("to_date"))
	}
	from, to := stockTransferByEntryIDDateRange(report, a.now)
	report.FromDate = from.Format("2006-01-02")
	report.ToDate = to.Format("2006-01-02")
	report.RangeLabel = stockTransferSummaryRangeLabel(from, to)
	report.Generated = true

	rows, err := a.store.StockTransferByEntryIDReportRows(r.Context(), from, to)
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	report.build(rows)
	a.render(w, r, "stock_transfer_by_entry_id_report.gohtml", viewData{Title: "Stock Transfers by Entry ID", StockTransferByEntryIDReport: report})
}

func (a *App) stockTransferSummaryByEntryIDReport(w http.ResponseWriter, r *http.Request) {
	report := a.defaultStockTransferSummaryByEntryIDReportData(r)
	if r.URL.Query().Get("run") != "1" {
		a.render(w, r, "stock_transfer_summary_by_entry_id_report.gohtml", viewData{Title: "Stock Transfers Summary By Entry ID", StockTransferSummaryByEntryID: report})
		return
	}

	report.Coverage = normalizedCoverage(r.URL.Query().Get("coverage"))
	report.PaperSize = normalizedPaperSize(r.URL.Query().Get("paper_size"))
	if report.PaperSize == "letter-landscape" {
		report.PaperSize = "letter"
	}
	report.PaperClass = "report-paper-size-" + report.PaperSize
	report.Year = listYear(r, a.now)
	report.Month = boundedInt(r.URL.Query().Get("month"), 1, 12, report.Month)
	report.FromDate = strings.TrimSpace(r.URL.Query().Get("from_date"))
	if report.Coverage == "range" {
		report.ToDate = strings.TrimSpace(r.URL.Query().Get("range_to_date"))
	} else {
		report.ToDate = strings.TrimSpace(r.URL.Query().Get("to_date"))
	}
	from, to := stockTransferSummaryByEntryIDDateRange(report, a.now)
	report.FromDate = from.Format("2006-01-02")
	report.ToDate = to.Format("2006-01-02")
	report.RangeLabel = stockTransferSummaryRangeLabel(from, to)
	report.Generated = true

	rows, err := a.store.StockTransferByEntryIDReportRows(r.Context(), from, to)
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	report.build(rows)
	a.render(w, r, "stock_transfer_summary_by_entry_id_report.gohtml", viewData{Title: "Stock Transfers Summary By Entry ID", StockTransferSummaryByEntryID: report})
}

func (a *App) stockTransferSummaryByItemReport(w http.ResponseWriter, r *http.Request) {
	report := a.defaultStockTransferSummaryByItemReportData(r)
	if r.URL.Query().Get("run") != "1" {
		a.render(w, r, "stock_transfer_summary_by_item_report.gohtml", viewData{Title: "Stock Transfers Summary By Item", StockTransferSummaryByItemReport: report})
		return
	}

	report.Coverage = normalizedCoverage(r.URL.Query().Get("coverage"))
	report.PaperSize = normalizedPaperSize(r.URL.Query().Get("paper_size"))
	if report.PaperSize == "letter-landscape" {
		report.PaperSize = "letter"
	}
	report.PaperClass = "report-paper-size-" + report.PaperSize
	report.Year = listYear(r, a.now)
	report.Month = boundedInt(r.URL.Query().Get("month"), 1, 12, report.Month)
	report.FromDate = strings.TrimSpace(r.URL.Query().Get("from_date"))
	if report.Coverage == "range" {
		report.ToDate = strings.TrimSpace(r.URL.Query().Get("range_to_date"))
	} else {
		report.ToDate = strings.TrimSpace(r.URL.Query().Get("to_date"))
	}
	from, to := stockTransferSummaryByItemDateRange(report, a.now)
	report.FromDate = from.Format("2006-01-02")
	report.ToDate = to.Format("2006-01-02")
	report.RangeLabel = stockTransferSummaryRangeLabel(from, to)
	report.Generated = true

	rows, err := a.store.StockTransferSummaryByItemReportRows(r.Context(), from, to)
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	report.build(rows)
	a.render(w, r, "stock_transfer_summary_by_item_report.gohtml", viewData{Title: "Stock Transfers Summary By Item", StockTransferSummaryByItemReport: report})
}

func (a *App) stockTransferMarkupByTransactionReport(w http.ResponseWriter, r *http.Request) {
	report := a.defaultStockTransferMarkupByTransactionReportData(r)
	if r.URL.Query().Get("run") != "1" {
		a.render(w, r, "stock_transfer_markup_by_transaction_report.gohtml", viewData{Title: "Transfer Markup by Transaction", StockTransferMarkupReport: report})
		return
	}

	report.Coverage = normalizedCoverage(r.URL.Query().Get("coverage"))
	report.PaperSize = normalizedPaperSize(r.URL.Query().Get("paper_size"))
	if report.PaperSize == "letter-landscape" {
		report.PaperSize = "letter"
	}
	report.PaperClass = "report-paper-size-" + report.PaperSize
	report.Year = listYear(r, a.now)
	report.Month = boundedInt(r.URL.Query().Get("month"), 1, 12, report.Month)
	report.FromDate = strings.TrimSpace(r.URL.Query().Get("from_date"))
	if report.Coverage == "range" {
		report.ToDate = strings.TrimSpace(r.URL.Query().Get("range_to_date"))
	} else {
		report.ToDate = strings.TrimSpace(r.URL.Query().Get("to_date"))
	}
	from, to := stockTransferMarkupByTransactionDateRange(report, a.now)
	report.FromDate = from.Format("2006-01-02")
	report.ToDate = to.Format("2006-01-02")
	report.RangeLabel = stockTransferSummaryRangeLabel(from, to)
	report.Generated = true

	rows, err := a.store.StockTransferMarkupByTransactionReportRows(r.Context(), from, to)
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	report.build(rows)
	a.render(w, r, "stock_transfer_markup_by_transaction_report.gohtml", viewData{Title: "Transfer Markup by Transaction", StockTransferMarkupReport: report})
}

func (a *App) stockLedgerReport(w http.ResponseWriter, r *http.Request) {
	report := a.defaultStockLedgerReportData(r)
	if r.URL.Query().Get("run") != "1" {
		a.render(w, r, "stock_ledger_report.gohtml", viewData{Title: "Stock Ledger", StockLedgerReport: report})
		return
	}

	report.Coverage = normalizedCoverage(r.URL.Query().Get("coverage"))
	report.PaperSize = normalizedPaperSize(r.URL.Query().Get("paper_size"))
	if report.PaperSize == "letter-landscape" {
		report.PaperSize = "letter"
	}
	report.PaperClass = "report-paper-size-" + report.PaperSize
	report.Year = listYear(r, a.now)
	report.Month = boundedInt(r.URL.Query().Get("month"), 1, 12, report.Month)
	report.FromDate = strings.TrimSpace(r.URL.Query().Get("from_date"))
	if report.Coverage == "range" {
		report.ToDate = strings.TrimSpace(r.URL.Query().Get("range_to_date"))
	} else {
		report.ToDate = strings.TrimSpace(r.URL.Query().Get("to_date"))
	}
	from, to := stockLedgerReportDateRange(report, a.now)
	report.FromDate = from.Format("2006-01-02")
	report.ToDate = to.Format("2006-01-02")
	report.RangeLabel = stockLedgerRangeLabel(from, to)
	report.Generated = true

	rows, err := a.store.StockLedgerReportRows(r.Context(), to)
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	report.build(rows, from, to)
	a.render(w, r, "stock_ledger_report.gohtml", viewData{Title: "Stock Ledger", StockLedgerReport: report})
}

func (a *App) stockAgingReport(w http.ResponseWriter, r *http.Request) {
	report := a.defaultStockAgingReportData()
	if r.URL.Query().Get("run") != "1" {
		a.render(w, r, "stock_aging_report.gohtml", viewData{Title: "Stock Aging", StockAgingReport: report})
		return
	}

	report.PaperSize = normalizedPaperSize(r.URL.Query().Get("paper_size"))
	if report.PaperSize == "letter" {
		report.PaperSize = "letter-landscape"
	}
	report.PaperClass = "report-paper-size-" + report.PaperSize
	cutoff := parseReportDate(strings.TrimSpace(r.URL.Query().Get("cutoff_date")), a.now())
	report.CutoffDate = cutoff.Format("2006-01-02")
	report.CutoffLabel = "As Of " + cutoff.Format("1/02/2006")
	report.BucketLabels = stockAgingBucketLabels(cutoff)
	report.Generated = true

	rows, err := a.store.StockAgingReportRows(r.Context(), cutoff)
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	report.build(rows)
	a.render(w, r, "stock_aging_report.gohtml", viewData{Title: "Stock Aging", StockAgingReport: report})
}

func (a *App) stockReorderPointReport(w http.ResponseWriter, r *http.Request) {
	report := a.defaultStockReorderPointReportData()
	if r.URL.Query().Get("run") != "1" {
		a.render(w, r, "stock_reorder_point_report.gohtml", viewData{Title: "Stock Reorder Point", StockReorderReport: report})
		return
	}

	report.PaperSize = normalizedPaperSize(r.URL.Query().Get("paper_size"))
	if report.PaperSize == "letter-landscape" {
		report.PaperSize = "letter"
	}
	report.PaperClass = "report-paper-size-" + report.PaperSize
	cutoff := parseReportDate(strings.TrimSpace(r.URL.Query().Get("cutoff_date")), a.now())
	report.CutoffDate = cutoff.Format("2006-01-02")
	report.CutoffLabel = "Stock Summary As Of: " + cutoff.Format("January 02, 2006")
	report.Generated = true

	rows, err := a.store.StockReorderPointReportRows(r.Context(), cutoff)
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	report.build(rows)
	a.render(w, r, "stock_reorder_point_report.gohtml", viewData{Title: "Stock Reorder Point", StockReorderReport: report})
}

func (a *App) stockSummaryReport(w http.ResponseWriter, r *http.Request) {
	report := a.defaultStockSummaryReportData()
	if r.URL.Query().Get("run") != "1" {
		a.render(w, r, "stock_summary_report.gohtml", viewData{Title: "Stock Summary", StockSummaryReport: report})
		return
	}

	report.PaperSize = normalizedPaperSize(r.URL.Query().Get("paper_size"))
	if report.PaperSize == "letter-landscape" {
		report.PaperSize = "letter"
	}
	report.PaperClass = "report-paper-size-" + report.PaperSize
	cutoff := parseReportDate(strings.TrimSpace(r.URL.Query().Get("cutoff_date")), a.now())
	report.CutoffDate = cutoff.Format("2006-01-02")
	report.CutoffLabel = "Stock Summary As Of: " + cutoff.Format("January 02, 2006")
	report.Generated = true

	rows, err := a.store.StockSummaryReportRows(r.Context(), cutoff)
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	report.build(rows)
	a.render(w, r, "stock_summary_report.gohtml", viewData{Title: "Stock Summary", StockSummaryReport: report})
}

func (a *App) incomingCheckCalendarReport(w http.ResponseWriter, r *http.Request) {
	report := a.defaultIncomingCheckCalendarReportData(r)
	report.Year = boundedInt(r.URL.Query().Get("year"), 1900, 2200, report.Year)
	report.Month = boundedInt(r.URL.Query().Get("month"), 1, 12, report.Month)
	report.setCalendarNavigation()
	report.Generated = true

	monthStart := time.Date(report.Year, time.Month(report.Month), 1, 0, 0, 0, 0, a.now().Location())
	monthEnd := monthStart.AddDate(0, 1, -1)
	rows, err := a.store.IncomingCheckReportRows(r.Context(), monthEnd)
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	report.build(rows, monthStart)
	a.render(w, r, "incoming_check_calendar_report.gohtml", viewData{Title: "Incoming Check Calendar", IncomingCheckCalendar: report})
}

func (a *App) defaultPurchaseReportData(r *http.Request) purchaseReportData {
	now := a.now()
	return purchaseReportData{
		ReportType:  "detailed",
		Coverage:    "month",
		Year:        listYear(r, a.now),
		Month:       defaultReportMonth,
		FromDate:    time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02"),
		ToDate:      now.Format("2006-01-02"),
		PaperSize:   "letter",
		PaperClass:  "report-paper-size-letter",
		Title:       "CIMS",
		CurrentPage: 1,
		TotalPages:  1,
	}
}

func (a *App) defaultPurchaseByDRNumberReportData(r *http.Request) purchaseByDRNumberReportData {
	now := a.now()
	return purchaseByDRNumberReportData{
		Coverage:    "month",
		Year:        listYear(r, a.now),
		Month:       defaultReportMonth,
		FromDate:    time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02"),
		ToDate:      now.Format("2006-01-02"),
		PaperSize:   "letter",
		PaperClass:  "report-paper-size-letter",
		Title:       "CIMS",
		CurrentPage: 1,
		TotalPages:  1,
	}
}

func (a *App) defaultPurchaseByStockCodeReportData(r *http.Request) purchaseByStockCodeReportData {
	now := a.now()
	return purchaseByStockCodeReportData{
		Coverage:    "month",
		Year:        listYear(r, a.now),
		Month:       defaultReportMonth,
		FromDate:    time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02"),
		ToDate:      now.Format("2006-01-02"),
		PaperSize:   "letter",
		PaperClass:  "report-paper-size-letter",
		Title:       "CIMS",
		CurrentPage: 1,
		TotalPages:  1,
	}
}

func (a *App) defaultPurchaseBySupplierReportData(r *http.Request) purchaseBySupplierReportData {
	now := a.now()
	return purchaseBySupplierReportData{
		Coverage:    "month",
		Year:        listYear(r, a.now),
		Month:       defaultReportMonth,
		FromDate:    time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02"),
		ToDate:      now.Format("2006-01-02"),
		PaperSize:   "letter",
		PaperClass:  "report-paper-size-letter",
		Title:       "CIMS",
		CurrentPage: 1,
		TotalPages:  1,
	}
}

func (a *App) defaultSalesReportData(r *http.Request) salesReportData {
	now := a.now()
	return salesReportData{
		ReportType:  "detailed",
		Coverage:    "month",
		Year:        listYear(r, a.now),
		Month:       defaultReportMonth,
		FromDate:    time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02"),
		ToDate:      now.Format("2006-01-02"),
		PaperSize:   "letter",
		PaperClass:  "report-paper-size-letter",
		Title:       "CIMS",
		CurrentPage: 1,
		TotalPages:  1,
	}
}

func (a *App) defaultSalesByORCIDRNumberReportData(r *http.Request) salesByORCIDRNumberReportData {
	now := a.now()
	return salesByORCIDRNumberReportData{
		Coverage:    "month",
		Year:        listYear(r, a.now),
		Month:       defaultReportMonth,
		FromDate:    time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02"),
		ToDate:      now.Format("2006-01-02"),
		PaperSize:   "letter",
		PaperClass:  "report-paper-size-letter",
		Title:       "CIMS",
		CurrentPage: 1,
		TotalPages:  1,
	}
}

func (a *App) defaultSalesMarkupByTransactionReportData(r *http.Request) salesMarkupByTransactionReportData {
	now := a.now()
	return salesMarkupByTransactionReportData{
		Coverage:    "month",
		Year:        listYear(r, a.now),
		Month:       defaultReportMonth,
		FromDate:    time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02"),
		ToDate:      now.Format("2006-01-02"),
		PaperSize:   "letter",
		PaperClass:  "report-paper-size-letter",
		Title:       "CIMS",
		CurrentPage: 1,
		TotalPages:  1,
	}
}

func (a *App) defaultSalesSummaryByItemReportData(r *http.Request) salesSummaryByItemReportData {
	now := a.now()
	return salesSummaryByItemReportData{
		Coverage:    "month",
		Year:        listYear(r, a.now),
		Month:       defaultReportMonth,
		FromDate:    time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02"),
		ToDate:      now.Format("2006-01-02"),
		PaperSize:   "letter",
		PaperClass:  "report-paper-size-letter",
		Title:       "CIMS",
		CurrentPage: 1,
		TotalPages:  1,
	}
}

func (a *App) defaultSalesByCustomerReportData(r *http.Request) salesByCustomerReportData {
	now := a.now()
	return salesByCustomerReportData{
		Coverage:    "month",
		Year:        listYear(r, a.now),
		Month:       defaultReportMonth,
		FromDate:    time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02"),
		ToDate:      now.Format("2006-01-02"),
		PaperSize:   "letter",
		PaperClass:  "report-paper-size-letter",
		Title:       "CIMS",
		CurrentPage: 1,
		TotalPages:  1,
	}
}

func (a *App) defaultSalesByCustomerSummaryByItemReportData(r *http.Request) salesByCustomerSummaryByItemReportData {
	now := a.now()
	return salesByCustomerSummaryByItemReportData{
		Coverage:    "month",
		Year:        listYear(r, a.now),
		Month:       defaultReportMonth,
		FromDate:    time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02"),
		ToDate:      now.Format("2006-01-02"),
		PaperSize:   "letter",
		PaperClass:  "report-paper-size-letter",
		Title:       "CIMS",
		CurrentPage: 1,
		TotalPages:  1,
	}
}

func (a *App) defaultSalesByStockNameReportData(r *http.Request) salesByStockNameReportData {
	now := a.now()
	return salesByStockNameReportData{
		Coverage:    "month",
		Year:        listYear(r, a.now),
		Month:       defaultReportMonth,
		FromDate:    time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02"),
		ToDate:      now.Format("2006-01-02"),
		PaperSize:   "letter",
		PaperClass:  "report-paper-size-letter",
		Title:       "CIMS",
		CurrentPage: 1,
		TotalPages:  1,
	}
}

func (a *App) defaultAPLedgerReportData(r *http.Request) apLedgerReportData {
	now := a.now()
	return apLedgerReportData{
		ReportType:  "detailed",
		Coverage:    "month",
		Year:        listYear(r, a.now),
		Month:       defaultReportMonth,
		FromDate:    time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02"),
		ToDate:      now.Format("2006-01-02"),
		PaperSize:   "letter",
		PaperClass:  "report-paper-size-letter",
		Title:       "CIMS",
		CurrentPage: 1,
		TotalPages:  1,
	}
}

func (a *App) defaultARLedgerReportData(r *http.Request) arLedgerReportData {
	now := a.now()
	return arLedgerReportData{
		ReportType:  "detailed",
		Coverage:    "month",
		Year:        listYear(r, a.now),
		Month:       defaultReportMonth,
		FromDate:    time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02"),
		ToDate:      now.Format("2006-01-02"),
		PaperSize:   "letter",
		PaperClass:  "report-paper-size-letter",
		Title:       "CIMS",
		CurrentPage: 1,
		TotalPages:  1,
	}
}

func (a *App) defaultIncomingCheckReportData(r *http.Request) incomingCheckReportData {
	now := a.now()
	return incomingCheckReportData{
		ReportType:  "detailed",
		CutoffDate:  now.Format("2006-01-02"),
		PaperSize:   "letter",
		PaperClass:  "report-paper-size-letter",
		Title:       "CIMS",
		CutoffLabel: "Check Date Cut-Off: " + now.Format("02-Jan-2006"),
		CurrentPage: 1,
		TotalPages:  1,
	}
}

func (a *App) defaultOutgoingCheckReportData(r *http.Request) outgoingCheckReportData {
	now := a.now()
	return outgoingCheckReportData{
		ReportType:  "detailed",
		CutoffDate:  now.Format("2006-01-02"),
		PaperSize:   "letter",
		PaperClass:  "report-paper-size-letter",
		Title:       "CIMS",
		CutoffLabel: "Check Date Cut-Off: " + now.Format("02-Jan-2006"),
		CurrentPage: 1,
		TotalPages:  1,
	}
}

func (a *App) defaultExpenseReportData(r *http.Request) expenseReportData {
	now := a.now()
	return expenseReportData{
		ReportType:  "detailed",
		Coverage:    "month",
		Year:        listYear(r, a.now),
		Month:       defaultReportMonth,
		FromDate:    time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02"),
		ToDate:      now.Format("2006-01-02"),
		PaperSize:   "letter-landscape",
		PaperClass:  "report-paper-size-letter-landscape",
		Title:       "CIMS",
		CurrentPage: 1,
		TotalPages:  1,
	}
}

func (a *App) defaultIncomeStatementReportData(r *http.Request) incomeStatementReportData {
	now := a.now()
	return incomeStatementReportData{
		Coverage:    "month",
		Year:        listYear(r, a.now),
		Month:       defaultReportMonth,
		FromDate:    time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02"),
		ToDate:      now.Format("2006-01-02"),
		PaperSize:   "letter",
		PaperClass:  "report-paper-size-letter",
		Title:       "CIMS",
		CurrentPage: 1,
		TotalPages:  1,
	}
}

func (a *App) defaultIncentiveReportData(r *http.Request) incentiveReportData {
	now := a.now()
	return incentiveReportData{
		Coverage:    "month",
		Year:        listYear(r, a.now),
		Month:       defaultReportMonth,
		FromDate:    time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02"),
		ToDate:      now.Format("2006-01-02"),
		PaperSize:   "letter",
		PaperClass:  "report-paper-size-letter",
		Title:       "CIMS",
		CurrentPage: 1,
		TotalPages:  1,
	}
}

func (a *App) defaultDailySalesCollectionReportData() dailySalesCollectionReportData {
	now := a.now()
	return dailySalesCollectionReportData{
		ReportDate:  now.Format("2006-01-02"),
		PaperSize:   "letter",
		PaperClass:  "report-paper-size-letter",
		Title:       "SORONGON RICE & CORN MILL",
		DateLabel:   "Report Date: " + now.Format("01/02/2006"),
		CurrentPage: 1,
		TotalPages:  1,
	}
}

func (a *App) defaultDailyDueCheckReportData() dailyDueCheckReportData {
	now := a.now()
	return dailyDueCheckReportData{
		CutoffDate:  now.Format("2006-01-02"),
		PaperSize:   "letter",
		PaperClass:  "report-paper-size-letter",
		Title:       "CIMS",
		CutoffLabel: "Cut-Off Date: " + now.Format("01/02/2006"),
		CurrentPage: 1,
		TotalPages:  1,
	}
}

func (a *App) defaultStockSalesTransferReportData(r *http.Request) stockSalesTransferReportData {
	now := a.now()
	return stockSalesTransferReportData{
		Coverage:    "month",
		Year:        listYear(r, a.now),
		Month:       defaultReportMonth,
		FromDate:    time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02"),
		ToDate:      now.Format("2006-01-02"),
		PaperSize:   "letter-landscape",
		PaperClass:  "report-paper-size-letter-landscape",
		Title:       "CIMS",
		CurrentPage: 1,
		TotalPages:  1,
	}
}

func (a *App) defaultStockSalesTransferAmountReportData(r *http.Request) stockSalesTransferAmountReportData {
	now := a.now()
	return stockSalesTransferAmountReportData{
		Coverage:    "month",
		Year:        listYear(r, a.now),
		Month:       defaultReportMonth,
		FromDate:    time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02"),
		ToDate:      now.Format("2006-01-02"),
		PaperSize:   "letter-landscape",
		PaperClass:  "report-paper-size-letter-landscape",
		Title:       "SORONGON AGRIVET",
		CurrentPage: 1,
		TotalPages:  1,
	}
}

func (a *App) defaultStockTransferSummaryReportData(r *http.Request) stockTransferSummaryReportData {
	now := a.now()
	return stockTransferSummaryReportData{
		Coverage:    "month",
		Year:        listYear(r, a.now),
		Month:       defaultReportMonth,
		FromDate:    time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02"),
		ToDate:      now.Format("2006-01-02"),
		PaperSize:   "letter",
		PaperClass:  "report-paper-size-letter",
		Title:       "SORONGON AGRIVET",
		CurrentPage: 1,
		TotalPages:  1,
	}
}

func (a *App) defaultStockTransferByStockNameReportData(r *http.Request) stockTransferByStockNameReportData {
	now := a.now()
	return stockTransferByStockNameReportData{
		Coverage:    "month",
		Year:        listYear(r, a.now),
		Month:       defaultReportMonth,
		FromDate:    time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02"),
		ToDate:      now.Format("2006-01-02"),
		PaperSize:   "letter",
		PaperClass:  "report-paper-size-letter",
		Title:       "SORONGON AGRIVET",
		CurrentPage: 1,
		TotalPages:  1,
	}
}

func (a *App) defaultStockTransferByBranchReportData(r *http.Request) stockTransferByBranchReportData {
	now := a.now()
	return stockTransferByBranchReportData{
		Coverage:    "month",
		Year:        listYear(r, a.now),
		Month:       defaultReportMonth,
		FromDate:    time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02"),
		ToDate:      now.Format("2006-01-02"),
		PaperSize:   "letter",
		PaperClass:  "report-paper-size-letter",
		Title:       "SORONGON AGRIVET",
		CurrentPage: 1,
		TotalPages:  1,
	}
}

func (a *App) defaultStockTransferByEntryIDReportData(r *http.Request) stockTransferByEntryIDReportData {
	now := a.now()
	return stockTransferByEntryIDReportData{
		Coverage:    "month",
		Year:        listYear(r, a.now),
		Month:       defaultReportMonth,
		FromDate:    time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02"),
		ToDate:      now.Format("2006-01-02"),
		PaperSize:   "letter",
		PaperClass:  "report-paper-size-letter",
		Title:       "SORONGON AGRIVET",
		CurrentPage: 1,
		TotalPages:  1,
	}
}

func (a *App) defaultStockTransferSummaryByEntryIDReportData(r *http.Request) stockTransferSummaryByEntryIDReportData {
	now := a.now()
	return stockTransferSummaryByEntryIDReportData{
		Coverage:    "month",
		Year:        listYear(r, a.now),
		Month:       defaultReportMonth,
		FromDate:    time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02"),
		ToDate:      now.Format("2006-01-02"),
		PaperSize:   "letter",
		PaperClass:  "report-paper-size-letter",
		Title:       "SORONGON AGRIVET",
		CurrentPage: 1,
		TotalPages:  1,
	}
}

func (a *App) defaultStockTransferSummaryByItemReportData(r *http.Request) stockTransferSummaryByItemReportData {
	now := a.now()
	return stockTransferSummaryByItemReportData{
		Coverage:    "month",
		Year:        listYear(r, a.now),
		Month:       defaultReportMonth,
		FromDate:    time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02"),
		ToDate:      now.Format("2006-01-02"),
		PaperSize:   "letter",
		PaperClass:  "report-paper-size-letter",
		Title:       "SORONGON AGRIVET",
		CurrentPage: 1,
		TotalPages:  1,
	}
}

func (a *App) defaultStockTransferMarkupByTransactionReportData(r *http.Request) stockTransferMarkupByTransactionReportData {
	now := a.now()
	return stockTransferMarkupByTransactionReportData{
		Coverage:    "month",
		Year:        listYear(r, a.now),
		Month:       defaultReportMonth,
		FromDate:    time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02"),
		ToDate:      now.Format("2006-01-02"),
		PaperSize:   "letter",
		PaperClass:  "report-paper-size-letter",
		Title:       "SORONGON AGRIVET",
		CurrentPage: 1,
		TotalPages:  1,
	}
}

func (a *App) defaultStockLedgerReportData(r *http.Request) stockLedgerReportData {
	now := a.now()
	return stockLedgerReportData{
		Coverage:    "month",
		Year:        listYear(r, a.now),
		Month:       defaultReportMonth,
		FromDate:    time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02"),
		ToDate:      now.Format("2006-01-02"),
		PaperSize:   "letter",
		PaperClass:  "report-paper-size-letter",
		Title:       "CIMS",
		CurrentPage: 1,
		TotalPages:  1,
	}
}

func (a *App) defaultStockAgingReportData() stockAgingReportData {
	now := a.now()
	return stockAgingReportData{
		CutoffDate:   now.Format("2006-01-02"),
		PaperSize:    "letter-landscape",
		PaperClass:   "report-paper-size-letter-landscape",
		Title:        "SORONGON AGRIVET",
		CutoffLabel:  "As Of " + now.Format("1/02/2006"),
		CurrentPage:  1,
		TotalPages:   1,
		BucketLabels: stockAgingBucketLabels(now),
	}
}

func (a *App) defaultStockReorderPointReportData() stockReorderPointReportData {
	now := a.now()
	return stockReorderPointReportData{
		CutoffDate:  now.Format("2006-01-02"),
		PaperSize:   "letter",
		PaperClass:  "report-paper-size-letter",
		Title:       "CIMS",
		CutoffLabel: "Stock Summary As Of: " + now.Format("January 02, 2006"),
		CurrentPage: 1,
		TotalPages:  1,
	}
}

func (a *App) defaultStockSummaryReportData() stockSummaryReportData {
	now := a.now()
	return stockSummaryReportData{
		CutoffDate:  now.Format("2006-01-02"),
		PaperSize:   "letter",
		PaperClass:  "report-paper-size-letter",
		Title:       "CIMS",
		CutoffLabel: "Stock Summary As Of: " + now.Format("January 02, 2006"),
		CurrentPage: 1,
		TotalPages:  1,
	}
}

func (a *App) defaultIncomingCheckCalendarReportData(r *http.Request) incomingCheckCalendarReportData {
	now := a.now()
	report := incomingCheckCalendarReportData{
		Year:      listYear(r, a.now),
		Month:     defaultReportMonth,
		MonthName: now.Month().String(),
		Title:     "CIMS",
		Weekdays: []incomingCheckCalendarWeekday{
			{Name: "SUNDAY", Weekend: "sunday"},
			{Name: "MONDAY"},
			{Name: "TUESDAY"},
			{Name: "WEDNESDAY"},
			{Name: "THURSDAY"},
			{Name: "FRIDAY"},
			{Name: "SATURDAY", Weekend: "saturday"},
		},
	}
	report.setCalendarNavigation()
	return report
}

func normalizedReportType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "summary":
		return "summary"
	default:
		return "detailed"
	}
}

func normalizedIncomingCheckReportType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "summary", "summary-postdated":
		return "summary-postdated"
	default:
		return "detailed"
	}
}

func normalizedLedgerReportType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "summary", "aging":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "detailed"
	}
}

func normalizedPaperSize(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "a4", "legal", "letter-landscape":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "letter"
	}
}

func normalizedCoverage(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "range", "to-date":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "month"
	}
}

func boundedInt(value string, min, max, fallback int) int {
	n, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || n < min || n > max {
		return fallback
	}
	return n
}

func reportDateRange(report purchaseReportData, now func() time.Time) (time.Time, time.Time) {
	loc := now().Location()
	switch report.Coverage {
	case "range":
		from := parseReportDate(report.FromDate, time.Date(report.Year, time.Month(report.Month), 1, 0, 0, 0, 0, loc))
		to := parseReportDate(report.ToDate, from)
		if to.Before(from) {
			from, to = to, from
		}
		return from, to
	case "to-date":
		to := parseReportDate(report.ToDate, now())
		from := time.Date(to.Year(), time.January, 1, 0, 0, 0, 0, loc)
		return from, to
	default:
		from := time.Date(report.Year, time.Month(report.Month), 1, 0, 0, 0, 0, loc)
		to := from.AddDate(0, 1, -1)
		return from, to
	}
}

func purchaseByDRDateRange(report purchaseByDRNumberReportData, now func() time.Time) (time.Time, time.Time) {
	loc := now().Location()
	switch report.Coverage {
	case "range":
		from := parseReportDate(report.FromDate, time.Date(report.Year, time.Month(report.Month), 1, 0, 0, 0, 0, loc))
		to := parseReportDate(report.ToDate, from)
		if to.Before(from) {
			from, to = to, from
		}
		return from, to
	case "to-date":
		to := parseReportDate(report.ToDate, now())
		from := time.Date(to.Year(), time.January, 1, 0, 0, 0, 0, loc)
		return from, to
	default:
		from := time.Date(report.Year, time.Month(report.Month), 1, 0, 0, 0, 0, loc)
		to := from.AddDate(0, 1, -1)
		return from, to
	}
}

func purchaseByStockCodeDateRange(report purchaseByStockCodeReportData, now func() time.Time) (time.Time, time.Time) {
	loc := now().Location()
	switch report.Coverage {
	case "range":
		from := parseReportDate(report.FromDate, time.Date(report.Year, time.Month(report.Month), 1, 0, 0, 0, 0, loc))
		to := parseReportDate(report.ToDate, from)
		if to.Before(from) {
			from, to = to, from
		}
		return from, to
	case "to-date":
		to := parseReportDate(report.ToDate, now())
		from := time.Date(to.Year(), time.January, 1, 0, 0, 0, 0, loc)
		return from, to
	default:
		from := time.Date(report.Year, time.Month(report.Month), 1, 0, 0, 0, 0, loc)
		to := from.AddDate(0, 1, -1)
		return from, to
	}
}

func purchaseBySupplierDateRange(report purchaseBySupplierReportData, now func() time.Time) (time.Time, time.Time) {
	loc := now().Location()
	switch report.Coverage {
	case "range":
		from := parseReportDate(report.FromDate, time.Date(report.Year, time.Month(report.Month), 1, 0, 0, 0, 0, loc))
		to := parseReportDate(report.ToDate, from)
		if to.Before(from) {
			from, to = to, from
		}
		return from, to
	case "to-date":
		to := parseReportDate(report.ToDate, now())
		from := time.Date(to.Year(), time.January, 1, 0, 0, 0, 0, loc)
		return from, to
	default:
		from := time.Date(report.Year, time.Month(report.Month), 1, 0, 0, 0, 0, loc)
		to := from.AddDate(0, 1, -1)
		return from, to
	}
}

func salesReportDateRange(report salesReportData, now func() time.Time) (time.Time, time.Time) {
	loc := now().Location()
	switch report.Coverage {
	case "range":
		from := parseReportDate(report.FromDate, time.Date(report.Year, time.Month(report.Month), 1, 0, 0, 0, 0, loc))
		to := parseReportDate(report.ToDate, from)
		if to.Before(from) {
			from, to = to, from
		}
		return from, to
	case "to-date":
		to := parseReportDate(report.ToDate, now())
		from := time.Date(to.Year(), time.January, 1, 0, 0, 0, 0, loc)
		return from, to
	default:
		from := time.Date(report.Year, time.Month(report.Month), 1, 0, 0, 0, 0, loc)
		to := from.AddDate(0, 1, -1)
		return from, to
	}
}

func salesByCustomerDateRange(report salesByCustomerReportData, now func() time.Time) (time.Time, time.Time) {
	loc := now().Location()
	switch report.Coverage {
	case "range":
		from := parseReportDate(report.FromDate, time.Date(report.Year, time.Month(report.Month), 1, 0, 0, 0, 0, loc))
		to := parseReportDate(report.ToDate, from)
		if to.Before(from) {
			from, to = to, from
		}
		return from, to
	case "to-date":
		to := parseReportDate(report.ToDate, now())
		from := time.Date(to.Year(), time.January, 1, 0, 0, 0, 0, loc)
		return from, to
	default:
		from := time.Date(report.Year, time.Month(report.Month), 1, 0, 0, 0, 0, loc)
		to := from.AddDate(0, 1, -1)
		return from, to
	}
}

func salesByCustomerSummaryByItemDateRange(report salesByCustomerSummaryByItemReportData, now func() time.Time) (time.Time, time.Time) {
	loc := now().Location()
	switch report.Coverage {
	case "range":
		from := parseReportDate(report.FromDate, time.Date(report.Year, time.Month(report.Month), 1, 0, 0, 0, 0, loc))
		to := parseReportDate(report.ToDate, from)
		if to.Before(from) {
			from, to = to, from
		}
		return from, to
	case "to-date":
		to := parseReportDate(report.ToDate, now())
		from := time.Date(to.Year(), time.January, 1, 0, 0, 0, 0, loc)
		return from, to
	default:
		from := time.Date(report.Year, time.Month(report.Month), 1, 0, 0, 0, 0, loc)
		to := from.AddDate(0, 1, -1)
		return from, to
	}
}

func salesByORCIDRDateRange(report salesByORCIDRNumberReportData, now func() time.Time) (time.Time, time.Time) {
	loc := now().Location()
	switch report.Coverage {
	case "range":
		from := parseReportDate(report.FromDate, time.Date(report.Year, time.Month(report.Month), 1, 0, 0, 0, 0, loc))
		to := parseReportDate(report.ToDate, from)
		if to.Before(from) {
			from, to = to, from
		}
		return from, to
	case "to-date":
		to := parseReportDate(report.ToDate, now())
		from := time.Date(to.Year(), time.January, 1, 0, 0, 0, 0, loc)
		return from, to
	default:
		from := time.Date(report.Year, time.Month(report.Month), 1, 0, 0, 0, 0, loc)
		to := from.AddDate(0, 1, -1)
		return from, to
	}
}

func salesMarkupByTransactionDateRange(report salesMarkupByTransactionReportData, now func() time.Time) (time.Time, time.Time) {
	loc := now().Location()
	switch report.Coverage {
	case "range":
		from := parseReportDate(report.FromDate, time.Date(report.Year, time.Month(report.Month), 1, 0, 0, 0, 0, loc))
		to := parseReportDate(report.ToDate, from)
		if to.Before(from) {
			from, to = to, from
		}
		return from, to
	case "to-date":
		to := parseReportDate(report.ToDate, now())
		from := time.Date(to.Year(), time.January, 1, 0, 0, 0, 0, loc)
		return from, to
	default:
		from := time.Date(report.Year, time.Month(report.Month), 1, 0, 0, 0, 0, loc)
		to := from.AddDate(0, 1, -1)
		return from, to
	}
}

func salesSummaryByItemDateRange(report salesSummaryByItemReportData, now func() time.Time) (time.Time, time.Time) {
	loc := now().Location()
	switch report.Coverage {
	case "range":
		from := parseReportDate(report.FromDate, time.Date(report.Year, time.Month(report.Month), 1, 0, 0, 0, 0, loc))
		to := parseReportDate(report.ToDate, from)
		if to.Before(from) {
			from, to = to, from
		}
		return from, to
	case "to-date":
		to := parseReportDate(report.ToDate, now())
		from := time.Date(to.Year(), time.January, 1, 0, 0, 0, 0, loc)
		return from, to
	default:
		from := time.Date(report.Year, time.Month(report.Month), 1, 0, 0, 0, 0, loc)
		to := from.AddDate(0, 1, -1)
		return from, to
	}
}

func salesByStockNameDateRange(report salesByStockNameReportData, now func() time.Time) (time.Time, time.Time) {
	loc := now().Location()
	switch report.Coverage {
	case "range":
		from := parseReportDate(report.FromDate, time.Date(report.Year, time.Month(report.Month), 1, 0, 0, 0, 0, loc))
		to := parseReportDate(report.ToDate, from)
		if to.Before(from) {
			from, to = to, from
		}
		return from, to
	case "to-date":
		to := parseReportDate(report.ToDate, now())
		from := time.Date(to.Year(), time.January, 1, 0, 0, 0, 0, loc)
		return from, to
	default:
		from := time.Date(report.Year, time.Month(report.Month), 1, 0, 0, 0, 0, loc)
		to := from.AddDate(0, 1, -1)
		return from, to
	}
}

func apLedgerReportDateRange(report apLedgerReportData, now func() time.Time) (time.Time, time.Time) {
	loc := now().Location()
	switch report.Coverage {
	case "range":
		from := parseReportDate(report.FromDate, time.Date(report.Year, time.Month(report.Month), 1, 0, 0, 0, 0, loc))
		to := parseReportDate(report.ToDate, from)
		if to.Before(from) {
			from, to = to, from
		}
		return from, to
	case "to-date":
		to := parseReportDate(report.ToDate, now())
		from := time.Date(to.Year(), time.January, 1, 0, 0, 0, 0, loc)
		return from, to
	default:
		from := time.Date(report.Year, time.Month(report.Month), 1, 0, 0, 0, 0, loc)
		to := from.AddDate(0, 1, -1)
		return from, to
	}
}

func arLedgerReportDateRange(report arLedgerReportData, now func() time.Time) (time.Time, time.Time) {
	loc := now().Location()
	switch report.Coverage {
	case "range":
		from := parseReportDate(report.FromDate, time.Date(report.Year, time.Month(report.Month), 1, 0, 0, 0, 0, loc))
		to := parseReportDate(report.ToDate, from)
		if to.Before(from) {
			from, to = to, from
		}
		return from, to
	case "to-date":
		to := parseReportDate(report.ToDate, now())
		from := time.Date(to.Year(), time.January, 1, 0, 0, 0, 0, loc)
		return from, to
	default:
		from := time.Date(report.Year, time.Month(report.Month), 1, 0, 0, 0, 0, loc)
		to := from.AddDate(0, 1, -1)
		return from, to
	}
}

func expenseReportDateRange(report expenseReportData, now func() time.Time) (time.Time, time.Time) {
	loc := now().Location()
	switch report.Coverage {
	case "range":
		from := parseReportDate(report.FromDate, time.Date(report.Year, time.Month(report.Month), 1, 0, 0, 0, 0, loc))
		to := parseReportDate(report.ToDate, from)
		if to.Before(from) {
			from, to = to, from
		}
		return from, to
	case "to-date":
		to := parseReportDate(report.ToDate, now())
		from := time.Date(to.Year(), time.January, 1, 0, 0, 0, 0, loc)
		return from, to
	default:
		from := time.Date(report.Year, time.Month(report.Month), 1, 0, 0, 0, 0, loc)
		to := from.AddDate(0, 1, -1)
		return from, to
	}
}

func incomeStatementDateRange(report incomeStatementReportData, now func() time.Time) (time.Time, time.Time) {
	loc := now().Location()
	switch report.Coverage {
	case "range":
		from := parseReportDate(report.FromDate, time.Date(report.Year, time.Month(report.Month), 1, 0, 0, 0, 0, loc))
		to := parseReportDate(report.ToDate, from)
		if to.Before(from) {
			from, to = to, from
		}
		return from, to
	case "to-date":
		to := parseReportDate(report.ToDate, now())
		from := time.Date(to.Year(), time.January, 1, 0, 0, 0, 0, loc)
		return from, to
	default:
		from := time.Date(report.Year, time.Month(report.Month), 1, 0, 0, 0, 0, loc)
		to := from.AddDate(0, 1, -1)
		return from, to
	}
}

func incentiveReportDateRange(report incentiveReportData, now func() time.Time) (time.Time, time.Time) {
	loc := now().Location()
	switch report.Coverage {
	case "range":
		from := parseReportDate(report.FromDate, time.Date(report.Year, time.Month(report.Month), 1, 0, 0, 0, 0, loc))
		to := parseReportDate(report.ToDate, from)
		if to.Before(from) {
			from, to = to, from
		}
		return from, to
	case "to-date":
		to := parseReportDate(report.ToDate, now())
		from := time.Date(to.Year(), time.January, 1, 0, 0, 0, 0, loc)
		return from, to
	default:
		from := time.Date(report.Year, time.Month(report.Month), 1, 0, 0, 0, 0, loc)
		to := from.AddDate(0, 1, -1)
		return from, to
	}
}

func stockSalesTransferReportDateRange(report stockSalesTransferReportData, now func() time.Time) (time.Time, time.Time) {
	loc := now().Location()
	switch report.Coverage {
	case "range":
		from := parseReportDate(report.FromDate, time.Date(report.Year, time.Month(report.Month), 1, 0, 0, 0, 0, loc))
		to := parseReportDate(report.ToDate, from)
		if to.Before(from) {
			from, to = to, from
		}
		return from, to
	case "to-date":
		to := parseReportDate(report.ToDate, now())
		from := time.Date(to.Year(), time.January, 1, 0, 0, 0, 0, loc)
		return from, to
	default:
		from := time.Date(report.Year, time.Month(report.Month), 1, 0, 0, 0, 0, loc)
		to := from.AddDate(0, 1, -1)
		return from, to
	}
}

func stockSalesTransferAmountReportDateRange(report stockSalesTransferAmountReportData, now func() time.Time) (time.Time, time.Time) {
	loc := now().Location()
	switch report.Coverage {
	case "range":
		from := parseReportDate(report.FromDate, time.Date(report.Year, time.Month(report.Month), 1, 0, 0, 0, 0, loc))
		to := parseReportDate(report.ToDate, from)
		if to.Before(from) {
			from, to = to, from
		}
		return from, to
	case "to-date":
		to := parseReportDate(report.ToDate, now())
		from := time.Date(to.Year(), time.January, 1, 0, 0, 0, 0, loc)
		return from, to
	default:
		from := time.Date(report.Year, time.Month(report.Month), 1, 0, 0, 0, 0, loc)
		to := from.AddDate(0, 1, -1)
		return from, to
	}
}

func stockTransferSummaryDateRange(report stockTransferSummaryReportData, now func() time.Time) (time.Time, time.Time) {
	loc := now().Location()
	switch report.Coverage {
	case "range":
		from := parseReportDate(report.FromDate, time.Date(report.Year, time.Month(report.Month), 1, 0, 0, 0, 0, loc))
		to := parseReportDate(report.ToDate, from)
		if to.Before(from) {
			from, to = to, from
		}
		return from, to
	case "to-date":
		to := parseReportDate(report.ToDate, now())
		from := time.Date(to.Year(), time.January, 1, 0, 0, 0, 0, loc)
		return from, to
	default:
		from := time.Date(report.Year, time.Month(report.Month), 1, 0, 0, 0, 0, loc)
		to := from.AddDate(0, 1, -1)
		return from, to
	}
}

func stockTransferByStockNameDateRange(report stockTransferByStockNameReportData, now func() time.Time) (time.Time, time.Time) {
	loc := now().Location()
	switch report.Coverage {
	case "range":
		from := parseReportDate(report.FromDate, time.Date(report.Year, time.Month(report.Month), 1, 0, 0, 0, 0, loc))
		to := parseReportDate(report.ToDate, from)
		if to.Before(from) {
			from, to = to, from
		}
		return from, to
	case "to-date":
		to := parseReportDate(report.ToDate, now())
		from := time.Date(to.Year(), time.January, 1, 0, 0, 0, 0, loc)
		return from, to
	default:
		from := time.Date(report.Year, time.Month(report.Month), 1, 0, 0, 0, 0, loc)
		to := from.AddDate(0, 1, -1)
		return from, to
	}
}

func stockTransferByBranchDateRange(report stockTransferByBranchReportData, now func() time.Time) (time.Time, time.Time) {
	loc := now().Location()
	switch report.Coverage {
	case "range":
		from := parseReportDate(report.FromDate, time.Date(report.Year, time.Month(report.Month), 1, 0, 0, 0, 0, loc))
		to := parseReportDate(report.ToDate, from)
		if to.Before(from) {
			from, to = to, from
		}
		return from, to
	case "to-date":
		to := parseReportDate(report.ToDate, now())
		from := time.Date(to.Year(), time.January, 1, 0, 0, 0, 0, loc)
		return from, to
	default:
		from := time.Date(report.Year, time.Month(report.Month), 1, 0, 0, 0, 0, loc)
		to := from.AddDate(0, 1, -1)
		return from, to
	}
}

func stockTransferByEntryIDDateRange(report stockTransferByEntryIDReportData, now func() time.Time) (time.Time, time.Time) {
	loc := now().Location()
	switch report.Coverage {
	case "range":
		from := parseReportDate(report.FromDate, time.Date(report.Year, time.Month(report.Month), 1, 0, 0, 0, 0, loc))
		to := parseReportDate(report.ToDate, from)
		if to.Before(from) {
			from, to = to, from
		}
		return from, to
	case "to-date":
		to := parseReportDate(report.ToDate, now())
		from := time.Date(to.Year(), time.January, 1, 0, 0, 0, 0, loc)
		return from, to
	default:
		from := time.Date(report.Year, time.Month(report.Month), 1, 0, 0, 0, 0, loc)
		to := from.AddDate(0, 1, -1)
		return from, to
	}
}

func stockTransferSummaryByEntryIDDateRange(report stockTransferSummaryByEntryIDReportData, now func() time.Time) (time.Time, time.Time) {
	loc := now().Location()
	switch report.Coverage {
	case "range":
		from := parseReportDate(report.FromDate, time.Date(report.Year, time.Month(report.Month), 1, 0, 0, 0, 0, loc))
		to := parseReportDate(report.ToDate, from)
		if to.Before(from) {
			from, to = to, from
		}
		return from, to
	case "to-date":
		to := parseReportDate(report.ToDate, now())
		from := time.Date(to.Year(), time.January, 1, 0, 0, 0, 0, loc)
		return from, to
	default:
		from := time.Date(report.Year, time.Month(report.Month), 1, 0, 0, 0, 0, loc)
		to := from.AddDate(0, 1, -1)
		return from, to
	}
}

func stockTransferSummaryByItemDateRange(report stockTransferSummaryByItemReportData, now func() time.Time) (time.Time, time.Time) {
	loc := now().Location()
	switch report.Coverage {
	case "range":
		from := parseReportDate(report.FromDate, time.Date(report.Year, time.Month(report.Month), 1, 0, 0, 0, 0, loc))
		to := parseReportDate(report.ToDate, from)
		if to.Before(from) {
			from, to = to, from
		}
		return from, to
	case "to-date":
		to := parseReportDate(report.ToDate, now())
		from := time.Date(to.Year(), time.January, 1, 0, 0, 0, 0, loc)
		return from, to
	default:
		from := time.Date(report.Year, time.Month(report.Month), 1, 0, 0, 0, 0, loc)
		to := from.AddDate(0, 1, -1)
		return from, to
	}
}

func stockTransferMarkupByTransactionDateRange(report stockTransferMarkupByTransactionReportData, now func() time.Time) (time.Time, time.Time) {
	loc := now().Location()
	switch report.Coverage {
	case "range":
		from := parseReportDate(report.FromDate, time.Date(report.Year, time.Month(report.Month), 1, 0, 0, 0, 0, loc))
		to := parseReportDate(report.ToDate, from)
		if to.Before(from) {
			from, to = to, from
		}
		return from, to
	case "to-date":
		to := parseReportDate(report.ToDate, now())
		from := time.Date(to.Year(), time.January, 1, 0, 0, 0, 0, loc)
		return from, to
	default:
		from := time.Date(report.Year, time.Month(report.Month), 1, 0, 0, 0, 0, loc)
		to := from.AddDate(0, 1, -1)
		return from, to
	}
}

func stockLedgerReportDateRange(report stockLedgerReportData, now func() time.Time) (time.Time, time.Time) {
	loc := now().Location()
	switch report.Coverage {
	case "range":
		from := parseReportDate(report.FromDate, time.Date(report.Year, time.Month(report.Month), 1, 0, 0, 0, 0, loc))
		to := parseReportDate(report.ToDate, from)
		if to.Before(from) {
			from, to = to, from
		}
		return from, to
	case "to-date":
		to := parseReportDate(report.ToDate, now())
		from := time.Date(to.Year(), time.January, 1, 0, 0, 0, 0, loc)
		return from, to
	default:
		from := time.Date(report.Year, time.Month(report.Month), 1, 0, 0, 0, 0, loc)
		to := from.AddDate(0, 1, -1)
		return from, to
	}
}

func parseReportDate(value string, fallback time.Time) time.Time {
	if t, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(value), fallback.Location()); err == nil {
		return t
	}
	return time.Date(fallback.Year(), fallback.Month(), fallback.Day(), 0, 0, 0, 0, fallback.Location())
}

func purchaseRangeLabel(from, to time.Time) string {
	return "Purchases From: " + from.Format("January 02, 2006") + " To: " + to.Format("January 02, 2006")
}

func salesRangeLabel(from, to time.Time) string {
	return "Sales From: " + from.Format("January 02, 2006") + " To: " + to.Format("January 02, 2006")
}

func apLedgerRangeLabel(from, to time.Time) string {
	return "Ledger From: " + from.Format("January 02, 2006") + " To: " + to.Format("January 02, 2006")
}

func arLedgerRangeLabel(from, to time.Time) string {
	return "Ledger From: " + from.Format("January 02, 2006") + " To: " + to.Format("January 02, 2006")
}

func expenseRangeLabel(from, to time.Time) string {
	return "Purchases From: " + from.Format("January 02, 2006") + " To: " + to.Format("January 02, 2006")
}

func incomeStatementRangeLabel(from, to time.Time) string {
	return "From: " + from.Format("January 02, 2006") + " To: " + to.Format("January 02, 2006")
}

func incentiveRangeLabel(from, to time.Time) string {
	return "Sales From: " + from.Format("January 02, 2006") + " To: " + to.Format("January 02, 2006")
}

func stockSalesTransferRangeLabel(from, to time.Time) string {
	return "Sales and Transfers From: " + from.Format("January 02, 2006") + " To: " + to.Format("January 02, 2006")
}

func stockTransferSummaryRangeLabel(from, to time.Time) string {
	return "Sales From: " + from.Format("January 02, 2006") + " To: " + to.Format("January 02, 2006")
}

func stockLedgerRangeLabel(from, to time.Time) string {
	return "Period Covered : " + from.Format("1/2/2006") + " ~ " + to.Format("1/2/2006")
}

func stockAgingBucketLabels(cutoff time.Time) []string {
	labels := make([]string, 0, 6)
	for bucket := 0; bucket < 5; bucket++ {
		end := cutoff.AddDate(0, 0, -30*bucket)
		if bucket > 0 {
			end = end.AddDate(0, 0, -1)
		}
		start := cutoff.AddDate(0, 0, -30*(bucket+1))
		labels = append(labels, start.Format("01/02/2006")+" ~ "+end.Format("01/02/2006"))
	}
	labels = append(labels, "Before "+cutoff.AddDate(0, 0, -150).Format("01/02/2006"))
	return labels
}

func (report *purchaseReportData) build(rows []models.PurchaseReportRow) {
	groupBySupplier := map[string][]models.PurchaseReportRow{}
	for _, row := range rows {
		supplier := strings.TrimSpace(row.Supplier)
		if supplier == "" {
			supplier = "No Supplier"
		}
		groupBySupplier[supplier] = append(groupBySupplier[supplier], row)
		report.TotalGrossRaw += row.GrossCents
		report.TotalNetRaw += row.NetCents
	}
	report.TotalGross = moneyString(report.TotalGrossRaw)
	report.TotalNet = moneyString(report.TotalNetRaw)

	suppliers := make([]string, 0, len(groupBySupplier))
	for supplier := range groupBySupplier {
		suppliers = append(suppliers, supplier)
	}
	sort.Strings(suppliers)
	report.Suppliers = suppliers

	for _, supplier := range suppliers {
		var gross int64
		var net int64
		group := purchaseReportSupplierGroup{Supplier: supplier}
		for _, row := range groupBySupplier[supplier] {
			gross += row.GrossCents
			net += row.NetCents
			group.Rows = append(group.Rows, purchaseReportLine{
				EntryID:    row.EntryID,
				Date:       row.EntryDate,
				ORCINumber: row.ORCINumber,
				Gross:      moneyString(row.GrossCents),
				Net:        moneyString(row.NetCents),
			})
		}
		group.GrossTotal = moneyString(gross)
		group.NetTotal = moneyString(net)
		report.Groups = append(report.Groups, group)
		report.SummaryRows = append(report.SummaryRows, purchaseReportSummaryRow{
			Supplier: supplier,
			Gross:    moneyString(gross),
			Net:      moneyString(net),
		})
	}
	if report.ReportType == "detailed" && len(report.Groups) > 0 {
		report.TotalPages = len(report.Groups)
	}
}

func (report *purchaseByDRNumberReportData) build(rows []models.PurchaseByDRNumberReportRow) {
	groupsByReference := map[string]*purchaseByDRNumberGroup{}
	for _, row := range rows {
		reference := strings.TrimSpace(row.Reference)
		if reference == "" {
			reference = "No Reference"
		}
		group := groupsByReference[reference]
		if group == nil {
			group = &purchaseByDRNumberGroup{Reference: reference, Date: row.PurchaseDate}
			groupsByReference[reference] = group
		}
		if group.Date == "" {
			group.Date = row.PurchaseDate
		}
		supplier := strings.TrimSpace(row.Supplier)
		if supplier == "" {
			supplier = "No Supplier"
		}
		stockCode := strings.TrimSpace(row.StockCode)
		if stockCode == "" {
			stockCode = "N/A"
		}
		stockName := strings.TrimSpace(row.StockName)
		if stockName == "" {
			stockName = "No Stock"
		}
		group.Rows = append(group.Rows, purchaseByDRNumberLine{
			Supplier: supplier,
			Code:     stockCode,
			Stock:    stockName,
			Quantity: qtyDecimalString(row.Quantity),
			Cost:     moneyString(row.UnitCostCents),
			Amount:   moneyString(row.AmountCents),
			QtyRaw:   row.Quantity,
			AmtRaw:   row.AmountCents,
		})
		group.QtyRaw += row.Quantity
		group.AmtRaw += row.AmountCents
		report.TotalQtyRaw += row.Quantity
		report.TotalAmountRaw += row.AmountCents
	}

	references := make([]string, 0, len(groupsByReference))
	for reference := range groupsByReference {
		references = append(references, reference)
	}
	sort.Slice(references, func(i, j int) bool {
		return strings.ToLower(references[i]) < strings.ToLower(references[j])
	})
	for _, reference := range references {
		group := groupsByReference[reference]
		sort.SliceStable(group.Rows, func(i, j int) bool {
			left := strings.ToLower(group.Rows[i].Supplier + "|" + group.Rows[i].Code + "|" + group.Rows[i].Stock)
			right := strings.ToLower(group.Rows[j].Supplier + "|" + group.Rows[j].Code + "|" + group.Rows[j].Stock)
			return left < right
		})
		group.TotalQty = qtyDecimalString(group.QtyRaw)
		group.TotalAmt = moneyString(group.AmtRaw)
		report.Groups = append(report.Groups, *group)
	}
	report.TotalQuantity = qtyDecimalString(report.TotalQtyRaw)
	report.TotalAmount = moneyString(report.TotalAmountRaw)
	if len(report.Groups) > 0 {
		report.TotalPages = len(report.Groups)
	}
}

func (report *purchaseByStockCodeReportData) build(rows []models.PurchaseByStockCodeReportRow) {
	groupsByCode := map[string]*purchaseByStockCodeGroup{}
	for _, row := range rows {
		stockCode := strings.TrimSpace(row.StockCode)
		if stockCode == "" {
			stockCode = "N/A"
		}
		stockName := strings.TrimSpace(row.StockName)
		if stockName == "" {
			stockName = "No Stock"
		}
		groupKey := strings.ToLower(stockCode + "|" + stockName)
		group := groupsByCode[groupKey]
		if group == nil {
			group = &purchaseByStockCodeGroup{StockCode: stockCode, StockName: stockName}
			groupsByCode[groupKey] = group
		}
		reference := strings.TrimSpace(row.Reference)
		if reference == "" {
			reference = "No Reference"
		}
		supplier := strings.TrimSpace(row.Supplier)
		if supplier == "" {
			supplier = "No Supplier"
		}
		group.Rows = append(group.Rows, purchaseByStockCodeLine{
			Reference: reference,
			Date:      row.PurchaseDate,
			Supplier:  supplier,
			Quantity:  qtyDecimalString(row.Quantity),
			Cost:      moneyString(row.UnitCostCents),
			Amount:    moneyString(row.AmountCents),
			QtyRaw:    row.Quantity,
			AmtRaw:    row.AmountCents,
		})
		group.QtyRaw += row.Quantity
		group.AmtRaw += row.AmountCents
		report.TotalQtyRaw += row.Quantity
		report.TotalAmountRaw += row.AmountCents
	}

	keys := make([]string, 0, len(groupsByCode))
	for key := range groupsByCode {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		group := groupsByCode[key]
		sort.SliceStable(group.Rows, func(i, j int) bool {
			left := strings.ToLower(group.Rows[i].Reference + "|" + group.Rows[i].Date + "|" + group.Rows[i].Supplier)
			right := strings.ToLower(group.Rows[j].Reference + "|" + group.Rows[j].Date + "|" + group.Rows[j].Supplier)
			return left < right
		})
		group.TotalQty = qtyDecimalString(group.QtyRaw)
		group.TotalAmt = moneyString(group.AmtRaw)
		report.Groups = append(report.Groups, *group)
	}
	report.TotalQuantity = qtyDecimalString(report.TotalQtyRaw)
	report.TotalAmount = moneyString(report.TotalAmountRaw)
	if len(report.Groups) > 0 {
		report.TotalPages = len(report.Groups)
	}
}

func (report *purchaseBySupplierReportData) build(rows []models.PurchaseBySupplierReportRow) {
	groupsBySupplier := map[string]*purchaseBySupplierGroup{}
	for _, row := range rows {
		supplier := strings.TrimSpace(row.Supplier)
		if supplier == "" {
			supplier = "No Supplier"
		}
		group := groupsBySupplier[supplier]
		if group == nil {
			group = &purchaseBySupplierGroup{Supplier: supplier}
			groupsBySupplier[supplier] = group
		}
		stockCode := strings.TrimSpace(row.StockCode)
		if stockCode == "" {
			stockCode = "N/A"
		}
		stockName := strings.TrimSpace(row.StockName)
		if stockName == "" {
			stockName = "No Stock"
		}
		stockGroupIndex := -1
		for index := range group.StockGroups {
			if strings.EqualFold(group.StockGroups[index].StockCode, stockCode) && strings.EqualFold(group.StockGroups[index].StockName, stockName) {
				stockGroupIndex = index
				break
			}
		}
		if stockGroupIndex == -1 {
			group.StockGroups = append(group.StockGroups, purchaseBySupplierStockGroup{StockCode: stockCode, StockName: stockName})
			stockGroupIndex = len(group.StockGroups) - 1
		}
		reference := strings.TrimSpace(row.Reference)
		if reference == "" {
			reference = "No Reference"
		}
		line := purchaseBySupplierLine{
			Reference: reference,
			Date:      row.PurchaseDate,
			Code:      stockCode,
			Stock:     stockName,
			Quantity:  qtyDecimalString(row.Quantity),
			Cost:      moneyString(row.UnitCostCents),
			Amount:    moneyString(row.AmountCents),
			QtyRaw:    row.Quantity,
			AmtRaw:    row.AmountCents,
		}
		stockGroup := &group.StockGroups[stockGroupIndex]
		stockGroup.Rows = append(stockGroup.Rows, line)
		stockGroup.QtyRaw += row.Quantity
		stockGroup.AmtRaw += row.AmountCents
		group.QtyRaw += row.Quantity
		group.AmtRaw += row.AmountCents
		report.TotalQtyRaw += row.Quantity
		report.TotalAmountRaw += row.AmountCents
	}

	suppliers := make([]string, 0, len(groupsBySupplier))
	for supplier := range groupsBySupplier {
		suppliers = append(suppliers, supplier)
	}
	sort.Slice(suppliers, func(i, j int) bool {
		return strings.ToLower(suppliers[i]) < strings.ToLower(suppliers[j])
	})
	for _, supplier := range suppliers {
		group := groupsBySupplier[supplier]
		sort.SliceStable(group.StockGroups, func(i, j int) bool {
			left := strings.ToLower(group.StockGroups[i].StockCode + "|" + group.StockGroups[i].StockName)
			right := strings.ToLower(group.StockGroups[j].StockCode + "|" + group.StockGroups[j].StockName)
			return left < right
		})
		for index := range group.StockGroups {
			stockGroup := &group.StockGroups[index]
			sort.SliceStable(stockGroup.Rows, func(i, j int) bool {
				left := strings.ToLower(stockGroup.Rows[i].Date + "|" + stockGroup.Rows[i].Reference)
				right := strings.ToLower(stockGroup.Rows[j].Date + "|" + stockGroup.Rows[j].Reference)
				return left < right
			})
			stockGroup.TotalQty = qtyDecimalString(stockGroup.QtyRaw)
			stockGroup.TotalAmt = moneyString(stockGroup.AmtRaw)
		}
		group.TotalQty = qtyDecimalString(group.QtyRaw)
		group.TotalAmt = moneyString(group.AmtRaw)
		report.Groups = append(report.Groups, *group)
	}
	report.TotalQuantity = qtyDecimalString(report.TotalQtyRaw)
	report.TotalAmount = moneyString(report.TotalAmountRaw)
	if len(report.Groups) > 0 {
		report.TotalPages = len(report.Groups)
	}
}

func (report *expenseReportData) build(rows []models.ExpenseReportRow, fallbackDate time.Time) {
	type categoryState struct {
		category expenseReportCategory
		total    int64
	}

	categoriesByKey := map[string]*categoryState{}
	amountsByDate := map[string]map[string]expenseReportAmountCell{}
	dateValues := map[string]time.Time{}

	for _, row := range rows {
		key := strings.TrimSpace(row.CategoryID)
		if key == "" {
			key = strings.TrimSpace(row.CategoryCode) + "|" + strings.TrimSpace(row.CategoryName)
		}
		if key == "|" {
			key = "uncategorized"
		}
		state := categoriesByKey[key]
		if state == nil {
			code := strings.TrimSpace(row.CategoryCode)
			name := strings.TrimSpace(row.CategoryName)
			if name == "" {
				name = "Uncategorized"
			}
			state = &categoryState{category: expenseReportCategory{ID: key, Code: code, Name: name}}
			categoriesByKey[key] = state
		}
		date := strings.TrimSpace(row.EntryDate)
		if date == "" {
			date = fallbackDate.Format("01/02/2006")
		}
		dateValues[date] = parseReportDate(rowDateForParse(date), fallbackDate)
		if amountsByDate[date] == nil {
			amountsByDate[date] = map[string]expenseReportAmountCell{}
		}
		cell := amountsByDate[date][key]
		cell.CashRaw += row.CashCents
		cell.CheckRaw += row.CheckCents
		cell.TotalRaw += row.TotalCents
		amountsByDate[date][key] = cell
		state.total += row.TotalCents
		report.GrandTotalRaw += row.TotalCents
	}
	report.GrandTotal = moneyString(report.GrandTotalRaw)

	categoryKeys := make([]string, 0, len(categoriesByKey))
	for key := range categoriesByKey {
		categoryKeys = append(categoryKeys, key)
	}
	sort.Slice(categoryKeys, func(i, j int) bool {
		left := categoriesByKey[categoryKeys[i]].category
		right := categoriesByKey[categoryKeys[j]].category
		if left.Code != right.Code {
			return strings.ToLower(left.Code) < strings.ToLower(right.Code)
		}
		return strings.ToLower(left.Name) < strings.ToLower(right.Name)
	})
	for _, key := range categoryKeys {
		state := categoriesByKey[key]
		report.Categories = append(report.Categories, state.category)
		report.SummaryRows = append(report.SummaryRows, expenseReportSummaryRow{
			Code:     state.category.Code,
			Name:     state.category.Name,
			Total:    moneyString(state.total),
			TotalRaw: state.total,
		})
	}

	dates := make([]string, 0, len(dateValues))
	for date := range dateValues {
		dates = append(dates, date)
	}
	sort.Slice(dates, func(i, j int) bool {
		return dateValues[dates[i]].Before(dateValues[dates[j]])
	})

	const categoriesPerPage = 8
	if len(report.Categories) == 0 {
		return
	}
	for start := 0; start < len(report.Categories); start += categoriesPerPage {
		end := start + categoriesPerPage
		if end > len(report.Categories) {
			end = len(report.Categories)
		}
		page := expenseReportPage{Number: len(report.Pages) + 1, Categories: report.Categories[start:end]}
		for _, date := range dates {
			detailRow := expenseReportDetailRow{Date: date}
			for _, category := range page.Categories {
				cell := amountsByDate[date][category.ID]
				cell.Cash = moneyString(cell.CashRaw)
				cell.Check = moneyString(cell.CheckRaw)
				cell.Total = moneyString(cell.TotalRaw)
				detailRow.Cells = append(detailRow.Cells, cell)
				detailRow.Total.CashRaw += cell.CashRaw
				detailRow.Total.CheckRaw += cell.CheckRaw
				detailRow.Total.TotalRaw += cell.TotalRaw
			}
			detailRow.Total.Cash = moneyString(detailRow.Total.CashRaw)
			detailRow.Total.Check = moneyString(detailRow.Total.CheckRaw)
			detailRow.Total.Total = moneyString(detailRow.Total.TotalRaw)
			page.Rows = append(page.Rows, detailRow)
		}
		report.Pages = append(report.Pages, page)
	}
	if report.ReportType == "detailed" && len(report.Pages) > 0 {
		report.TotalPages = len(report.Pages)
	}
}

func (report *salesReportData) build(rows []models.SalesReportRow) {
	groupByCustomer := map[string][]models.SalesReportRow{}
	for _, row := range rows {
		customer := strings.TrimSpace(row.Customer)
		if customer == "" {
			customer = "No Customer"
		}
		groupByCustomer[customer] = append(groupByCustomer[customer], row)
		report.TotalGrossRaw += row.GrossCents
		report.TotalNetRaw += row.NetCents
	}
	report.TotalGross = moneyString(report.TotalGrossRaw)
	report.TotalNet = moneyString(report.TotalNetRaw)

	customers := make([]string, 0, len(groupByCustomer))
	for customer := range groupByCustomer {
		customers = append(customers, customer)
	}
	sort.Strings(customers)
	report.Customers = customers

	for _, customer := range customers {
		var gross int64
		var net int64
		group := salesReportCustomerGroup{Customer: customer}
		for _, row := range groupByCustomer[customer] {
			gross += row.GrossCents
			net += row.NetCents
			group.Rows = append(group.Rows, salesReportLine{
				EntryID:    row.EntryID,
				Date:       row.EntryDate,
				ORCINumber: row.ORCINumber,
				Gross:      moneyString(row.GrossCents),
				Net:        moneyString(row.NetCents),
			})
		}
		group.GrossTotal = moneyString(gross)
		group.NetTotal = moneyString(net)
		report.Groups = append(report.Groups, group)
		report.SummaryRows = append(report.SummaryRows, salesReportSummaryRow{
			Customer: customer,
			Gross:    moneyString(gross),
			Net:      moneyString(net),
		})
	}
	if report.ReportType == "detailed" && len(report.Groups) > 0 {
		report.TotalPages = len(report.Groups)
	}
}

func (report *salesByORCIDRNumberReportData) build(rows []models.SalesByORCIDRNumberReportRow) {
	groupsByReference := map[string]*salesByORCIDRNumberGroup{}
	for _, row := range rows {
		reference := strings.TrimSpace(row.Reference)
		if reference == "" {
			reference = "No Reference"
		}
		group := groupsByReference[reference]
		if group == nil {
			group = &salesByORCIDRNumberGroup{Reference: reference, Date: row.SalesDate}
			groupsByReference[reference] = group
		}
		if group.Date == "" {
			group.Date = row.SalesDate
		}
		customer := strings.TrimSpace(row.Customer)
		if customer == "" {
			customer = "No Customer"
		}
		stockCode := strings.TrimSpace(row.StockCode)
		if stockCode == "" {
			stockCode = "N/A"
		}
		stockName := strings.TrimSpace(row.StockName)
		if stockName == "" {
			stockName = "No Stock"
		}
		group.Rows = append(group.Rows, salesByORCIDRNumberLine{
			Customer: customer,
			Code:     stockCode,
			Stock:    stockName,
			Quantity: qtyDecimalString(row.Quantity),
			Price:    moneyString(row.PriceCents),
			Amount:   moneyString(row.AmountCents),
			QtyRaw:   row.Quantity,
			AmtRaw:   row.AmountCents,
		})
		group.QtyRaw += row.Quantity
		group.AmtRaw += row.AmountCents
		report.TotalQtyRaw += row.Quantity
		report.TotalAmountRaw += row.AmountCents
	}

	references := make([]string, 0, len(groupsByReference))
	for reference := range groupsByReference {
		references = append(references, reference)
	}
	sort.Slice(references, func(i, j int) bool {
		return strings.ToLower(references[i]) < strings.ToLower(references[j])
	})
	for _, reference := range references {
		group := groupsByReference[reference]
		sort.SliceStable(group.Rows, func(i, j int) bool {
			left := strings.ToLower(group.Rows[i].Customer + "|" + group.Rows[i].Code + "|" + group.Rows[i].Stock)
			right := strings.ToLower(group.Rows[j].Customer + "|" + group.Rows[j].Code + "|" + group.Rows[j].Stock)
			return left < right
		})
		group.TotalQty = qtyDecimalString(group.QtyRaw)
		group.TotalAmt = moneyString(group.AmtRaw)
		report.Groups = append(report.Groups, *group)
	}
	report.TotalQuantity = qtyDecimalString(report.TotalQtyRaw)
	report.TotalAmount = moneyString(report.TotalAmountRaw)
	if len(report.Groups) > 0 {
		report.TotalPages = len(report.Groups)
	}
}

func (report *salesMarkupByTransactionReportData) build(rows []models.SalesMarkupByTransactionReportRow) {
	for _, row := range rows {
		entryID := strings.TrimSpace(row.EntryID)
		if entryID == "" {
			entryID = "No Entry ID"
		}
		salesType := strings.TrimSpace(row.SalesType)
		if salesType == "" {
			salesType = "Charge"
		}
		receiptNo := strings.TrimSpace(row.ReceiptNo)
		if receiptNo == "" {
			receiptNo = "No Receipt"
		}
		itemGroup := strings.TrimSpace(row.ItemGroup)
		if itemGroup == "" {
			itemGroup = "Uncategorized"
		}
		report.Rows = append(report.Rows, salesMarkupByTransactionLine{
			SalesDate:     row.SalesDate,
			EntryID:       entryID,
			SalesType:     salesType,
			ReceiptNo:     receiptNo,
			ItemGroup:     itemGroup,
			Markup:        moneyString(row.MarkupCents),
			MarkupPercent: percentString(row.MarkupCents, row.CapitalCents),
			MarkupRaw:     row.MarkupCents,
			CapitalRaw:    row.CapitalCents,
		})
		report.TotalMarkupRaw += row.MarkupCents
		report.TotalCapitalRaw += row.CapitalCents
	}
	report.TotalMarkup = moneyString(report.TotalMarkupRaw)
	report.TotalMarkupPercent = percentString(report.TotalMarkupRaw, report.TotalCapitalRaw)
	if len(report.Rows) > 0 {
		for start := 0; start < len(report.Rows); start += 40 {
			end := start + 40
			if end > len(report.Rows) {
				end = len(report.Rows)
			}
			report.Pages = append(report.Pages, salesMarkupByTransactionPage{
				Number: len(report.Pages) + 1,
				Rows:   report.Rows[start:end],
				Last:   end == len(report.Rows),
			})
		}
		report.TotalPages = len(report.Pages)
	}
}

func (report *salesSummaryByItemReportData) build(rows []models.SalesByStockNameReportRow) {
	categories := map[string]*salesSummaryByItemCategoryGroup{}
	rowsByStock := map[string]map[string]*salesSummaryByItemLine{}
	for _, row := range rows {
		category := strings.TrimSpace(row.Category)
		if category == "" {
			category = "Uncategorized"
		}
		categoryGroup := categories[category]
		if categoryGroup == nil {
			categoryGroup = &salesSummaryByItemCategoryGroup{Category: category}
			categories[category] = categoryGroup
			rowsByStock[category] = map[string]*salesSummaryByItemLine{}
		}
		stockCode := strings.TrimSpace(row.StockCode)
		if stockCode == "" {
			stockCode = "N/A"
		}
		stockName := strings.TrimSpace(row.StockName)
		if stockName == "" {
			stockName = "No Stock"
		}
		stockKey := strings.ToLower(stockCode + "|" + stockName)
		line := rowsByStock[category][stockKey]
		if line == nil {
			line = &salesSummaryByItemLine{StockCode: stockCode, StockName: stockName}
			rowsByStock[category][stockKey] = line
		}
		line.QtyRaw += row.Quantity
		line.AmtRaw += row.AmountCents
		categoryGroup.QtyRaw += row.Quantity
		categoryGroup.AmtRaw += row.AmountCents
		report.TotalQtyRaw += row.Quantity
		report.TotalAmountRaw += row.AmountCents
	}

	categoryNames := make([]string, 0, len(categories))
	for category := range categories {
		categoryNames = append(categoryNames, category)
	}
	sort.Slice(categoryNames, func(i, j int) bool {
		return strings.ToLower(categoryNames[i]) < strings.ToLower(categoryNames[j])
	})
	for _, category := range categoryNames {
		categoryGroup := categories[category]
		stockKeys := make([]string, 0, len(rowsByStock[category]))
		for key := range rowsByStock[category] {
			stockKeys = append(stockKeys, key)
		}
		sort.Slice(stockKeys, func(i, j int) bool {
			return stockKeys[i] < stockKeys[j]
		})
		for _, key := range stockKeys {
			line := rowsByStock[category][key]
			line.Quantity = qtyDecimalString(line.QtyRaw)
			line.Amount = moneyString(line.AmtRaw)
			categoryGroup.Rows = append(categoryGroup.Rows, *line)
		}
		categoryGroup.TotalQty = qtyDecimalString(categoryGroup.QtyRaw)
		categoryGroup.TotalAmt = moneyString(categoryGroup.AmtRaw)
		report.Categories = append(report.Categories, *categoryGroup)
	}
	report.TotalQuantity = qtyDecimalString(report.TotalQtyRaw)
	report.TotalAmount = moneyString(report.TotalAmountRaw)
	if len(report.Categories) > 0 {
		report.TotalPages = len(report.Categories)
	}
}

func (report *salesByCustomerReportData) build(rows []models.SalesByCustomerReportRow) {
	categories := map[string]*salesByCustomerCategoryGroup{}
	customerGroups := map[string]map[string]*salesByCustomerGroup{}
	for _, row := range rows {
		category := strings.TrimSpace(row.Category)
		if category == "" {
			category = "Uncategorized"
		}
		customer := strings.TrimSpace(row.Customer)
		if customer == "" {
			customer = "No Customer"
		}
		categoryGroup := categories[category]
		if categoryGroup == nil {
			categoryGroup = &salesByCustomerCategoryGroup{Category: category}
			categories[category] = categoryGroup
			customerGroups[category] = map[string]*salesByCustomerGroup{}
		}
		customerGroup := customerGroups[category][customer]
		if customerGroup == nil {
			customerGroup = &salesByCustomerGroup{Customer: customer}
			customerGroups[category][customer] = customerGroup
		}
		stockCode := strings.TrimSpace(row.StockCode)
		if stockCode == "" {
			stockCode = "N/A"
		}
		stockName := strings.TrimSpace(row.StockName)
		if stockName == "" {
			stockName = "No Stock"
		}
		reference := strings.TrimSpace(row.Reference)
		if reference == "" {
			reference = "No Reference"
		}
		customerGroup.Rows = append(customerGroup.Rows, salesByCustomerLine{
			Reference: reference,
			Date:      row.SalesDate,
			Code:      stockCode,
			Stock:     stockName,
			Quantity:  qtyDecimalString(row.Quantity),
			Price:     moneyString(row.PriceCents),
			Amount:    moneyString(row.AmountCents),
			QtyRaw:    row.Quantity,
			AmtRaw:    row.AmountCents,
		})
		customerGroup.QtyRaw += row.Quantity
		customerGroup.AmtRaw += row.AmountCents
		categoryGroup.QtyRaw += row.Quantity
		categoryGroup.AmtRaw += row.AmountCents
		report.TotalQtyRaw += row.Quantity
		report.TotalAmountRaw += row.AmountCents
	}

	categoryNames := make([]string, 0, len(categories))
	for category := range categories {
		categoryNames = append(categoryNames, category)
	}
	sort.Slice(categoryNames, func(i, j int) bool {
		return strings.ToLower(categoryNames[i]) < strings.ToLower(categoryNames[j])
	})
	for _, category := range categoryNames {
		categoryGroup := categories[category]
		customerNames := make([]string, 0, len(customerGroups[category]))
		for customer := range customerGroups[category] {
			customerNames = append(customerNames, customer)
		}
		sort.Slice(customerNames, func(i, j int) bool {
			return strings.ToLower(customerNames[i]) < strings.ToLower(customerNames[j])
		})
		for _, customer := range customerNames {
			customerGroup := customerGroups[category][customer]
			sort.SliceStable(customerGroup.Rows, func(i, j int) bool {
				left := strings.ToLower(customerGroup.Rows[i].Date + "|" + customerGroup.Rows[i].Reference + "|" + customerGroup.Rows[i].Code + "|" + customerGroup.Rows[i].Stock)
				right := strings.ToLower(customerGroup.Rows[j].Date + "|" + customerGroup.Rows[j].Reference + "|" + customerGroup.Rows[j].Code + "|" + customerGroup.Rows[j].Stock)
				return left < right
			})
			customerGroup.TotalQty = qtyDecimalString(customerGroup.QtyRaw)
			customerGroup.TotalAmt = moneyString(customerGroup.AmtRaw)
			categoryGroup.Customers = append(categoryGroup.Customers, *customerGroup)
		}
		categoryGroup.TotalQty = qtyDecimalString(categoryGroup.QtyRaw)
		categoryGroup.TotalAmt = moneyString(categoryGroup.AmtRaw)
		report.Categories = append(report.Categories, *categoryGroup)
	}
	report.TotalQuantity = qtyDecimalString(report.TotalQtyRaw)
	report.TotalAmount = moneyString(report.TotalAmountRaw)
	if len(report.Categories) > 0 {
		report.TotalPages = len(report.Categories)
	}
}

func (report *salesByCustomerSummaryByItemReportData) build(rows []models.SalesByCustomerReportRow) {
	categories := map[string]*salesByCustomerSummaryByItemCategoryGroup{}
	customerGroups := map[string]map[string]*salesByCustomerSummaryByItemCustomerGroup{}
	rowsByItem := map[string]map[string]map[string]*salesByCustomerSummaryByItemLine{}
	for _, row := range rows {
		category := strings.TrimSpace(row.Category)
		if category == "" {
			category = "Uncategorized"
		}
		customer := strings.TrimSpace(row.Customer)
		if customer == "" {
			customer = "No Customer"
		}
		categoryGroup := categories[category]
		if categoryGroup == nil {
			categoryGroup = &salesByCustomerSummaryByItemCategoryGroup{Category: category}
			categories[category] = categoryGroup
			customerGroups[category] = map[string]*salesByCustomerSummaryByItemCustomerGroup{}
			rowsByItem[category] = map[string]map[string]*salesByCustomerSummaryByItemLine{}
		}
		customerGroup := customerGroups[category][customer]
		if customerGroup == nil {
			customerGroup = &salesByCustomerSummaryByItemCustomerGroup{Customer: customer}
			customerGroups[category][customer] = customerGroup
			rowsByItem[category][customer] = map[string]*salesByCustomerSummaryByItemLine{}
		}
		stockCode := strings.TrimSpace(row.StockCode)
		if stockCode == "" {
			stockCode = "N/A"
		}
		stockName := strings.TrimSpace(row.StockName)
		if stockName == "" {
			stockName = "No Stock"
		}
		itemKey := strings.ToLower(stockCode + "|" + stockName + "|" + strconv.FormatInt(row.PriceCents, 10))
		line := rowsByItem[category][customer][itemKey]
		if line == nil {
			line = &salesByCustomerSummaryByItemLine{Code: stockCode, Stock: stockName, PriceRaw: row.PriceCents}
			rowsByItem[category][customer][itemKey] = line
		}
		line.QtyRaw += row.Quantity
		line.AmtRaw += row.AmountCents
		customerGroup.QtyRaw += row.Quantity
		customerGroup.AmtRaw += row.AmountCents
		categoryGroup.QtyRaw += row.Quantity
		categoryGroup.AmtRaw += row.AmountCents
		report.TotalQtyRaw += row.Quantity
		report.TotalAmountRaw += row.AmountCents
	}

	categoryNames := make([]string, 0, len(categories))
	for category := range categories {
		categoryNames = append(categoryNames, category)
	}
	sort.Slice(categoryNames, func(i, j int) bool {
		return strings.ToLower(categoryNames[i]) < strings.ToLower(categoryNames[j])
	})
	for _, category := range categoryNames {
		categoryGroup := categories[category]
		customerNames := make([]string, 0, len(customerGroups[category]))
		for customer := range customerGroups[category] {
			customerNames = append(customerNames, customer)
		}
		sort.Slice(customerNames, func(i, j int) bool {
			return strings.ToLower(customerNames[i]) < strings.ToLower(customerNames[j])
		})
		for _, customer := range customerNames {
			customerGroup := customerGroups[category][customer]
			itemKeys := make([]string, 0, len(rowsByItem[category][customer]))
			for key := range rowsByItem[category][customer] {
				itemKeys = append(itemKeys, key)
			}
			sort.Slice(itemKeys, func(i, j int) bool {
				return itemKeys[i] < itemKeys[j]
			})
			for _, key := range itemKeys {
				line := rowsByItem[category][customer][key]
				line.Quantity = qtyDecimalString(line.QtyRaw)
				line.Price = moneyString(line.PriceRaw)
				line.Amount = moneyString(line.AmtRaw)
				customerGroup.Rows = append(customerGroup.Rows, *line)
			}
			customerGroup.TotalQty = qtyDecimalString(customerGroup.QtyRaw)
			customerGroup.TotalAmt = moneyString(customerGroup.AmtRaw)
			categoryGroup.Customers = append(categoryGroup.Customers, *customerGroup)
		}
		categoryGroup.TotalQty = qtyDecimalString(categoryGroup.QtyRaw)
		categoryGroup.TotalAmt = moneyString(categoryGroup.AmtRaw)
		report.Categories = append(report.Categories, *categoryGroup)
	}
	report.TotalQuantity = qtyDecimalString(report.TotalQtyRaw)
	report.TotalAmount = moneyString(report.TotalAmountRaw)
	if len(report.Categories) > 0 {
		report.TotalPages = len(report.Categories)
	}
}

func (report *salesByStockNameReportData) build(rows []models.SalesByStockNameReportRow) {
	categories := map[string]*salesByStockNameCategoryGroup{}
	stockGroups := map[string]map[string]*salesByStockNameGroup{}
	for _, row := range rows {
		category := strings.TrimSpace(row.Category)
		if category == "" {
			category = "Uncategorized"
		}
		stockCode := strings.TrimSpace(row.StockCode)
		if stockCode == "" {
			stockCode = "N/A"
		}
		stockName := strings.TrimSpace(row.StockName)
		if stockName == "" {
			stockName = "No Stock"
		}
		categoryGroup := categories[category]
		if categoryGroup == nil {
			categoryGroup = &salesByStockNameCategoryGroup{Category: category}
			categories[category] = categoryGroup
			stockGroups[category] = map[string]*salesByStockNameGroup{}
		}
		stockKey := strings.ToLower(stockName + "|" + stockCode)
		stockGroup := stockGroups[category][stockKey]
		if stockGroup == nil {
			stockGroup = &salesByStockNameGroup{StockCode: stockCode, StockName: stockName}
			stockGroups[category][stockKey] = stockGroup
		}
		reference := strings.TrimSpace(row.Reference)
		if reference == "" {
			reference = "No Reference"
		}
		customer := strings.TrimSpace(row.Customer)
		if customer == "" {
			customer = "No Customer"
		}
		stockGroup.Rows = append(stockGroup.Rows, salesByStockNameLine{
			Reference: reference,
			Date:      row.SalesDate,
			Customer:  customer,
			Quantity:  qtyDecimalString(row.Quantity),
			Price:     moneyString(row.PriceCents),
			Amount:    moneyString(row.AmountCents),
			QtyRaw:    row.Quantity,
			AmtRaw:    row.AmountCents,
		})
		stockGroup.QtyRaw += row.Quantity
		stockGroup.AmtRaw += row.AmountCents
		categoryGroup.QtyRaw += row.Quantity
		categoryGroup.AmtRaw += row.AmountCents
		report.TotalQtyRaw += row.Quantity
		report.TotalAmountRaw += row.AmountCents
	}

	categoryNames := make([]string, 0, len(categories))
	for category := range categories {
		categoryNames = append(categoryNames, category)
	}
	sort.Slice(categoryNames, func(i, j int) bool {
		return strings.ToLower(categoryNames[i]) < strings.ToLower(categoryNames[j])
	})
	for _, category := range categoryNames {
		categoryGroup := categories[category]
		stockKeys := make([]string, 0, len(stockGroups[category]))
		for stockKey := range stockGroups[category] {
			stockKeys = append(stockKeys, stockKey)
		}
		sort.Strings(stockKeys)
		for _, stockKey := range stockKeys {
			stockGroup := stockGroups[category][stockKey]
			sort.SliceStable(stockGroup.Rows, func(i, j int) bool {
				left := strings.ToLower(stockGroup.Rows[i].Date + "|" + stockGroup.Rows[i].Reference + "|" + stockGroup.Rows[i].Customer)
				right := strings.ToLower(stockGroup.Rows[j].Date + "|" + stockGroup.Rows[j].Reference + "|" + stockGroup.Rows[j].Customer)
				return left < right
			})
			stockGroup.TotalQty = qtyDecimalString(stockGroup.QtyRaw)
			stockGroup.TotalAmt = moneyString(stockGroup.AmtRaw)
			categoryGroup.Stocks = append(categoryGroup.Stocks, *stockGroup)
		}
		categoryGroup.TotalQty = qtyDecimalString(categoryGroup.QtyRaw)
		categoryGroup.TotalAmt = moneyString(categoryGroup.AmtRaw)
		report.Categories = append(report.Categories, *categoryGroup)
	}
	report.TotalQuantity = qtyDecimalString(report.TotalQtyRaw)
	report.TotalAmount = moneyString(report.TotalAmountRaw)
	if len(report.Categories) > 0 {
		report.TotalPages = len(report.Categories)
	}
}

func (report *apLedgerReportData) build(rows []models.APLedgerReportRow, from, to time.Time) {
	type supplierState struct {
		code           string
		name           string
		representative string
		rows           []models.APLedgerReportRow
	}

	states := map[string]*supplierState{}
	for _, row := range rows {
		key := row.SupplierID
		if key == "" {
			key = row.SupplierName
		}
		state := states[key]
		if state == nil {
			state = &supplierState{code: row.SupplierCode, name: row.SupplierName, representative: row.Representative}
			if state.name == "" {
				state.name = "No Supplier"
			}
			if state.representative == "" {
				state.representative = "NA"
			}
			states[key] = state
		}
		state.rows = append(state.rows, row)
	}

	keys := make([]string, 0, len(states))
	for key := range states {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		return strings.ToLower(states[keys[i]].name) < strings.ToLower(states[keys[j]].name)
	})

	var totalDebit, totalCredit, totalBalance int64
	var aging0, aging31, aging61, aging90 int64
	for _, key := range keys {
		state := states[key]
		report.Suppliers = append(report.Suppliers, state.name)
		var forwarded, balance, lastPositive int64
		var lastPayment time.Time
		group := apLedgerSupplierGroup{SupplierCode: state.code, SupplierName: state.name, Representative: state.representative}
		for _, row := range state.rows {
			rowDate := parseReportDate(rowDateForParse(row.EntryDate), to)
			if rowDate.Before(from) {
				forwarded += row.DeltaCents
				continue
			}
			if row.DeltaCents > 0 {
				totalCredit += row.DeltaCents
				lastPositive = row.DeltaCents
			} else if row.DeltaCents < 0 {
				totalDebit += -row.DeltaCents
				lastPayment = rowDate
			}
		}
		balance = forwarded
		if forwarded != 0 {
			group.Rows = append(group.Rows, apLedgerLine{Date: from.AddDate(0, 0, -1).Format("01/02/2006"), Reference: "Forwarded", Balance: moneyString(balance)})
		}
		for _, row := range state.rows {
			rowDate := parseReportDate(rowDateForParse(row.EntryDate), to)
			if rowDate.Before(from) || rowDate.After(to) {
				continue
			}
			balance += row.DeltaCents
			line := apLedgerLine{Date: row.EntryDate, Reference: apLedgerReference(row), Balance: moneyString(balance)}
			if row.DeltaCents < 0 {
				line.Debit = moneyString(-row.DeltaCents)
			} else {
				line.Credit = moneyString(row.DeltaCents)
			}
			group.Rows = append(group.Rows, line)
		}
		if len(group.Rows) == 0 {
			group.Rows = append(group.Rows, apLedgerLine{Date: from.AddDate(0, 0, -1).Format("01/02/2006"), Reference: "Forwarded", Balance: moneyString(balance)})
		}
		if report.ReportType == "detailed" {
			report.Groups = append(report.Groups, group)
		}

		asOfBalance := int64(0)
		var balanceAgeDate time.Time
		for _, row := range state.rows {
			rowDate := parseReportDate(rowDateForParse(row.EntryDate), to)
			if rowDate.After(to) {
				continue
			}
			asOfBalance += row.DeltaCents
			if row.DeltaCents > 0 {
				balanceAgeDate = rowDate
				lastPositive = row.DeltaCents
			}
			if row.DeltaCents < 0 {
				lastPayment = rowDate
			}
		}
		if asOfBalance != 0 {
			totalBalance += asOfBalance
			report.SummaryRows = append(report.SummaryRows, apLedgerSummaryRow{
				Code:           state.code,
				Company:        state.name,
				Representative: state.representative,
				Balance:        moneyString(asOfBalance),
				BalanceRaw:     asOfBalance,
			})
			if asOfBalance > 0 {
				agingRow := apLedgerAgingRow{
					Company:     state.name,
					LastPayment: apLedgerDateString(lastPayment),
					Balance:     moneyString(asOfBalance),
					BalanceRaw:  asOfBalance,
				}
				if balanceAgeDate.IsZero() {
					balanceAgeDate = to
				}
				if lastPositive <= 0 || lastPositive > asOfBalance {
					lastPositive = asOfBalance
				}
				ageDays := int(to.Sub(balanceAgeDate).Hours() / 24)
				switch {
				case ageDays <= 30:
					agingRow.Bucket0Raw = asOfBalance
					aging0 += asOfBalance
				case ageDays <= 60:
					agingRow.Bucket31Raw = asOfBalance
					aging31 += asOfBalance
				case ageDays <= 90:
					agingRow.Bucket61Raw = asOfBalance
					aging61 += asOfBalance
				default:
					agingRow.Bucket90Raw = asOfBalance
					aging90 += asOfBalance
				}
				agingRow.Bucket0 = moneyString(agingRow.Bucket0Raw)
				agingRow.Bucket31 = moneyString(agingRow.Bucket31Raw)
				agingRow.Bucket61 = moneyString(agingRow.Bucket61Raw)
				agingRow.Bucket90 = moneyString(agingRow.Bucket90Raw)
				report.AgingRows = append(report.AgingRows, agingRow)
			}
		}
	}
	report.TotalDebit = moneyString(totalDebit)
	report.TotalCredit = moneyString(totalCredit)
	report.TotalNet = moneyString(totalBalance)
	report.AgingTotals = apLedgerAgingTotals{
		Bucket0:  moneyString(aging0),
		Bucket31: moneyString(aging31),
		Bucket61: moneyString(aging61),
		Bucket90: moneyString(aging90),
		Balance:  moneyString(totalBalance),
	}
	if report.ReportType == "detailed" && len(report.Groups) > 0 {
		report.TotalPages = len(report.Groups)
	}
}

func (report *arLedgerReportData) build(rows []models.ARLedgerReportRow, from, to time.Time) {
	type customerState struct {
		code        string
		name        string
		creditTerm  string
		creditLimit int64
		rows        []models.ARLedgerReportRow
	}

	states := map[string]*customerState{}
	for _, row := range rows {
		key := row.CustomerID
		if key == "" {
			key = row.CustomerName
		}
		state := states[key]
		if state == nil {
			state = &customerState{code: row.CustomerCode, name: row.CustomerName, creditTerm: row.CreditTerm, creditLimit: row.CreditLimit}
			if state.name == "" {
				state.name = "No Customer"
			}
			states[key] = state
		}
		state.rows = append(state.rows, row)
	}

	keys := make([]string, 0, len(states))
	for key := range states {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		return strings.ToLower(states[keys[i]].name) < strings.ToLower(states[keys[j]].name)
	})

	var totalDebit, totalCredit, totalBalance int64
	var aging0, aging31, aging61, aging90 int64
	for _, key := range keys {
		state := states[key]
		report.Customers = append(report.Customers, state.name)
		var forwarded, balance int64
		var lastPayment time.Time
		group := arLedgerCustomerGroup{
			CustomerCode: state.code,
			CustomerName: state.name,
			CreditTerm:   state.creditTerm,
			CreditLimit:  moneyString(state.creditLimit),
		}
		for _, row := range state.rows {
			rowDate := parseReportDate(rowDateForParse(row.EntryDate), to)
			if rowDate.Before(from) {
				forwarded += row.DeltaCents
				continue
			}
			if row.DeltaCents > 0 {
				totalDebit += row.DeltaCents
			} else if row.DeltaCents < 0 {
				totalCredit += -row.DeltaCents
				lastPayment = rowDate
			}
		}
		balance = forwarded
		if forwarded != 0 {
			group.Rows = append(group.Rows, arLedgerLine{Date: from.AddDate(0, 0, -1).Format("01/02/2006"), Reference: "Forwarded", Balance: moneyString(balance)})
		}
		for _, row := range state.rows {
			rowDate := parseReportDate(rowDateForParse(row.EntryDate), to)
			if rowDate.Before(from) || rowDate.After(to) {
				continue
			}
			balance += row.DeltaCents
			line := arLedgerLine{Date: row.EntryDate, Reference: arLedgerReference(row), Balance: moneyString(balance)}
			if row.DeltaCents > 0 {
				line.Debit = moneyString(row.DeltaCents)
			} else {
				line.Credit = moneyString(-row.DeltaCents)
			}
			group.Rows = append(group.Rows, line)
		}
		if len(group.Rows) == 0 {
			group.Rows = append(group.Rows, arLedgerLine{Date: from.AddDate(0, 0, -1).Format("01/02/2006"), Reference: "Forwarded", Balance: moneyString(balance)})
		}

		asOfBalance := int64(0)
		var balanceAgeDate time.Time
		for _, row := range state.rows {
			rowDate := parseReportDate(rowDateForParse(row.EntryDate), to)
			if rowDate.After(to) {
				continue
			}
			asOfBalance += row.DeltaCents
			if row.DeltaCents > 0 {
				balanceAgeDate = rowDate
			}
			if row.DeltaCents < 0 {
				lastPayment = rowDate
			}
		}
		group.CurrentBalance = moneyString(asOfBalance)
		if report.ReportType == "detailed" {
			report.Groups = append(report.Groups, group)
		}
		if asOfBalance != 0 {
			totalBalance += asOfBalance
			report.SummaryRows = append(report.SummaryRows, arLedgerSummaryRow{
				Company:    state.name,
				Balance:    moneyString(asOfBalance),
				BalanceRaw: asOfBalance,
			})
			if asOfBalance > 0 {
				agingRow := arLedgerAgingRow{
					Company:      state.name,
					LastPayment:  apLedgerDateString(lastPayment),
					Balance:      moneyString(asOfBalance),
					TotalBalance: moneyString(asOfBalance),
					BalanceRaw:   asOfBalance,
				}
				if balanceAgeDate.IsZero() {
					balanceAgeDate = to
				}
				ageDays := int(to.Sub(balanceAgeDate).Hours() / 24)
				switch {
				case ageDays <= 30:
					agingRow.Bucket0Raw = asOfBalance
					aging0 += asOfBalance
				case ageDays <= 60:
					agingRow.Bucket31Raw = asOfBalance
					aging31 += asOfBalance
				case ageDays <= 90:
					agingRow.Bucket61Raw = asOfBalance
					aging61 += asOfBalance
				default:
					agingRow.Bucket90Raw = asOfBalance
					aging90 += asOfBalance
				}
				agingRow.Bucket0 = moneyString(agingRow.Bucket0Raw)
				agingRow.Bucket31 = moneyString(agingRow.Bucket31Raw)
				agingRow.Bucket61 = moneyString(agingRow.Bucket61Raw)
				agingRow.Bucket90 = moneyString(agingRow.Bucket90Raw)
				agingRow.OutstandingCheck = moneyString(agingRow.OutstandingCheckRaw)
				agingRow.TotalBalanceRaw = agingRow.BalanceRaw + agingRow.OutstandingCheckRaw
				agingRow.TotalBalance = moneyString(agingRow.TotalBalanceRaw)
				report.AgingRows = append(report.AgingRows, agingRow)
			}
		}
	}
	report.TotalDebit = moneyString(totalDebit)
	report.TotalCredit = moneyString(totalCredit)
	report.TotalNet = moneyString(totalBalance)
	report.AgingTotals = arLedgerAgingTotals{
		Bucket0:          moneyString(aging0),
		Bucket31:         moneyString(aging31),
		Bucket61:         moneyString(aging61),
		Bucket90:         moneyString(aging90),
		Balance:          moneyString(totalBalance),
		OutstandingCheck: moneyString(0),
		TotalBalance:     moneyString(totalBalance),
	}
	if report.ReportType == "detailed" && len(report.Groups) > 0 {
		report.TotalPages = len(report.Groups)
	}
}

func (report *incomingCheckReportData) build(rows []models.IncomingCheckReportRow, cutoff time.Time) {
	type datedRow struct {
		row       models.IncomingCheckReportRow
		checkDate time.Time
	}

	groupByPayee := map[string][]datedRow{}
	summaryByPayee := map[string]int64{}
	for _, row := range rows {
		payee := strings.TrimSpace(row.Payee)
		if payee == "" {
			payee = "No Payee"
		}
		row.Payee = payee
		checkDate := parseReportDate(rowDateForParse(row.CheckDate), cutoff)
		if report.ReportType == "summary-postdated" {
			if checkDate.After(cutoff) {
				summaryByPayee[payee] += row.AmountCents
				report.GrandTotalRaw += row.AmountCents
			}
			continue
		}
		if checkDate.After(cutoff) {
			continue
		}
		groupByPayee[payee] = append(groupByPayee[payee], datedRow{row: row, checkDate: checkDate})
		report.GrandTotalRaw += row.AmountCents
	}
	report.GrandTotal = moneyString(report.GrandTotalRaw)

	if report.ReportType == "summary-postdated" {
		payees := make([]string, 0, len(summaryByPayee))
		for payee := range summaryByPayee {
			payees = append(payees, payee)
		}
		sort.Strings(payees)
		report.Payees = payees
		for index, payee := range payees {
			total := summaryByPayee[payee]
			report.SummaryRows = append(report.SummaryRows, incomingCheckSummaryRow{
				RecordNumber: index + 1,
				Payee:        payee,
				Total:        moneyString(total),
				TotalRaw:     total,
			})
		}
		return
	}

	payees := make([]string, 0, len(groupByPayee))
	for payee := range groupByPayee {
		payees = append(payees, payee)
	}
	sort.Strings(payees)
	report.Payees = payees

	for _, payee := range payees {
		rows := groupByPayee[payee]
		sort.Slice(rows, func(i, j int) bool {
			if rows[i].checkDate.Equal(rows[j].checkDate) {
				return strings.ToLower(rows[i].row.Number) < strings.ToLower(rows[j].row.Number)
			}
			return rows[i].checkDate.Before(rows[j].checkDate)
		})

		group := incomingCheckPayeeGroup{Payee: payee}
		monthIndex := map[string]int{}
		for _, item := range rows {
			month := incomingCheckMonthLabel(item.checkDate)
			idx, exists := monthIndex[month]
			if !exists {
				group.Months = append(group.Months, incomingCheckMonthGroup{Month: month})
				idx = len(group.Months) - 1
				monthIndex[month] = idx
			}
			line := incomingCheckLine{
				Reference: item.row.Reference,
				Date:      item.row.CheckDate,
				Number:    item.row.Number,
				BankName:  item.row.BankName,
				Amount:    moneyString(item.row.AmountCents),
			}
			group.Months[idx].Rows = append(group.Months[idx].Rows, line)
			group.Months[idx].TotalRaw += item.row.AmountCents
			group.TotalRaw += item.row.AmountCents
		}
		for idx := range group.Months {
			group.Months[idx].Total = moneyString(group.Months[idx].TotalRaw)
		}
		group.Total = moneyString(group.TotalRaw)
		report.Groups = append(report.Groups, group)
	}
	if len(report.Groups) > 0 {
		report.TotalPages = len(report.Groups)
	}
}

func (report *outgoingCheckReportData) build(rows []models.OutgoingCheckReportRow, cutoff time.Time) {
	type datedRow struct {
		row       models.OutgoingCheckReportRow
		checkDate time.Time
	}

	groupByPayee := map[string][]datedRow{}
	summaryByPayee := map[string]int64{}
	for _, row := range rows {
		payee := strings.TrimSpace(row.Payee)
		if payee == "" {
			payee = "No Payee"
		}
		row.Payee = payee
		checkDate := parseReportDate(rowDateForParse(row.CheckDate), cutoff)
		if report.ReportType == "summary-postdated" {
			if checkDate.After(cutoff) {
				summaryByPayee[payee] += row.AmountCents
				report.GrandTotalRaw += row.AmountCents
			}
			continue
		}
		if checkDate.After(cutoff) {
			continue
		}
		groupByPayee[payee] = append(groupByPayee[payee], datedRow{row: row, checkDate: checkDate})
		report.GrandTotalRaw += row.AmountCents
	}
	report.GrandTotal = moneyString(report.GrandTotalRaw)

	if report.ReportType == "summary-postdated" {
		payees := make([]string, 0, len(summaryByPayee))
		for payee := range summaryByPayee {
			payees = append(payees, payee)
		}
		sort.Strings(payees)
		report.Payees = payees
		for index, payee := range payees {
			total := summaryByPayee[payee]
			report.SummaryRows = append(report.SummaryRows, outgoingCheckSummaryRow{
				RecordNumber: index + 1,
				Payee:        payee,
				Total:        moneyString(total),
				TotalRaw:     total,
			})
		}
		return
	}

	payees := make([]string, 0, len(groupByPayee))
	for payee := range groupByPayee {
		payees = append(payees, payee)
	}
	sort.Strings(payees)
	report.Payees = payees

	for _, payee := range payees {
		rows := groupByPayee[payee]
		sort.Slice(rows, func(i, j int) bool {
			if rows[i].checkDate.Equal(rows[j].checkDate) {
				return strings.ToLower(rows[i].row.Number) < strings.ToLower(rows[j].row.Number)
			}
			return rows[i].checkDate.Before(rows[j].checkDate)
		})

		group := outgoingCheckPayeeGroup{Payee: payee}
		monthIndex := map[string]int{}
		for _, item := range rows {
			month := incomingCheckMonthLabel(item.checkDate)
			idx, exists := monthIndex[month]
			if !exists {
				group.Months = append(group.Months, outgoingCheckMonthGroup{Month: month})
				idx = len(group.Months) - 1
				monthIndex[month] = idx
			}
			line := outgoingCheckLine{
				Reference: item.row.Reference,
				Date:      item.row.CheckDate,
				Number:    item.row.Number,
				BankName:  item.row.BankName,
				Amount:    moneyString(item.row.AmountCents),
			}
			group.Months[idx].Rows = append(group.Months[idx].Rows, line)
			group.Months[idx].TotalRaw += item.row.AmountCents
			group.TotalRaw += item.row.AmountCents
		}
		for idx := range group.Months {
			group.Months[idx].Total = moneyString(group.Months[idx].TotalRaw)
		}
		group.Total = moneyString(group.TotalRaw)
		report.Groups = append(report.Groups, group)
	}
	if len(report.Groups) > 0 {
		report.TotalPages = len(report.Groups)
	}
}

func (report *incomeStatementReportData) build(rows []models.IncomeStatementRow) {
	addLine := func(lines *[]incomeStatementLine, label string, amount int64) {
		label = strings.TrimSpace(label)
		if label == "" {
			label = "Unclassified"
		}
		*lines = append(*lines, incomeStatementLine{Label: label, Amount: moneyString(amount), Raw: amount})
	}

	for _, row := range rows {
		switch strings.ToLower(strings.TrimSpace(row.Section)) {
		case "cash_sales":
			report.CashSalesRaw += row.AmountCents
		case "charge_sales":
			report.ChargeSalesRaw += row.AmountCents
		case "sales_return":
			report.SalesReturnRaw += row.AmountCents
		case "beginning_inventory":
			report.BeginningInventoryRaw += row.AmountCents
		case "purchases":
			report.TotalPurchasesRaw += row.AmountCents
			addLine(&report.Purchases, row.Label, row.AmountCents)
		case "withdrawals":
			report.TotalWithdrawalsRaw += row.AmountCents
			addLine(&report.Withdrawals, row.Label, row.AmountCents)
		case "ending_inventory":
			report.EndingInventoryRaw += row.AmountCents
		case "operating_expenses":
			report.TotalExpensesRaw += row.AmountCents
			addLine(&report.OperatingExpenses, row.Label, row.AmountCents)
		case "other_income":
			report.TotalOtherIncomeRaw += row.AmountCents
			addLine(&report.OtherIncome, row.Label, row.AmountCents)
		}
	}

	report.TotalSalesRaw = report.CashSalesRaw + report.ChargeSalesRaw
	report.NetSalesRaw = report.TotalSalesRaw - report.SalesReturnRaw
	report.NetPurchasesRaw = report.TotalPurchasesRaw + report.TotalWithdrawalsRaw
	report.GoodsAvailableRaw = report.BeginningInventoryRaw + report.NetPurchasesRaw
	report.TotalCostOfSalesRaw = report.GoodsAvailableRaw - report.EndingInventoryRaw
	report.GrossProfitRaw = report.NetSalesRaw - report.TotalCostOfSalesRaw
	report.IncomeBeforeOtherRaw = report.GrossProfitRaw - report.TotalExpensesRaw
	report.NetIncomeRaw = report.IncomeBeforeOtherRaw + report.TotalOtherIncomeRaw

	report.CashSales = moneyString(report.CashSalesRaw)
	report.ChargeSales = moneyString(report.ChargeSalesRaw)
	report.TotalSales = moneyString(report.TotalSalesRaw)
	report.SalesReturn = moneyString(report.SalesReturnRaw)
	report.NetSales = moneyString(report.NetSalesRaw)
	report.BeginningInventory = moneyString(report.BeginningInventoryRaw)
	report.TotalPurchases = moneyString(report.TotalPurchasesRaw)
	report.TotalWithdrawals = moneyString(report.TotalWithdrawalsRaw)
	report.NetPurchases = moneyString(report.NetPurchasesRaw)
	report.GoodsAvailable = moneyString(report.GoodsAvailableRaw)
	report.EndingInventory = moneyString(-report.EndingInventoryRaw)
	report.TotalCostOfSales = moneyString(report.TotalCostOfSalesRaw)
	report.GrossProfit = moneyString(report.GrossProfitRaw)
	report.TotalExpenses = moneyString(report.TotalExpensesRaw)
	report.IncomeBeforeOther = moneyString(report.IncomeBeforeOtherRaw)
	report.TotalOtherIncome = moneyString(report.TotalOtherIncomeRaw)
	report.NetIncome = moneyString(report.NetIncomeRaw)
}

func (report *incentiveReportData) build(rows []models.IncentiveReportRow) {
	for _, row := range rows {
		agriPost := strings.TrimSpace(row.AgriPost)
		if agriPost == "" {
			agriPost = "Uncategorized"
		}
		line := incentiveReportLine{
			AgriPost:  agriPost,
			Qty:       qtyString(row.Qty),
			VIP:       qtyString(row.VIP),
			APS:       qtyString(row.APS),
			Takals:    qtyString(row.Takals),
			Farm:      qtyString(row.Farm),
			QtyRaw:    row.Qty,
			VIPRaw:    row.VIP,
			APSRaw:    row.APS,
			TakalsRaw: row.Takals,
			FarmRaw:   row.Farm,
		}
		report.Rows = append(report.Rows, line)
		report.TotalQtyRaw += row.Qty
		report.TotalVIPRaw += row.VIP
		report.TotalAPSRaw += row.APS
		report.TotalTakalsRaw += row.Takals
		report.TotalFarmRaw += row.Farm
	}
	report.GrandTotalRaw = report.TotalVIPRaw + report.TotalAPSRaw + report.TotalTakalsRaw + report.TotalFarmRaw
	report.TotalQty = qtyString(report.TotalQtyRaw)
	report.TotalVIP = qtyString(report.TotalVIPRaw)
	report.TotalAPS = qtyString(report.TotalAPSRaw)
	report.TotalTakals = qtyString(report.TotalTakalsRaw)
	report.TotalFarm = qtyString(report.TotalFarmRaw)
	report.GrandTotal = qtyString(report.GrandTotalRaw)
}

func (report *dailySalesCollectionReportData) build(rows []models.DailySalesCollectionReportRow) {
	sections := []dailySalesCollectionSection{
		{Key: "cash_sales", Title: "CASH SALES"},
		{Key: "charge_sales", Title: "CHARGE SALES"},
		{Key: "cash_receipts", Title: "CASH RECEIPTS"},
		{Key: "disbursements", Title: "DISBURSEMENTS"},
		{Key: "check_deposits", Title: "CHECK DEPOSITS"},
	}
	sectionIndex := map[string]int{}
	for idx := range sections {
		sectionIndex[sections[idx].Key] = idx
	}

	for _, row := range rows {
		key := strings.TrimSpace(row.Section)
		idx, ok := sectionIndex[key]
		if !ok {
			continue
		}
		name := strings.TrimSpace(row.Name)
		if name == "" {
			name = "Unspecified"
		}
		sections[idx].Rows = append(sections[idx].Rows, dailySalesCollectionLine{
			Name:      name,
			Reference: strings.TrimSpace(row.Reference),
			Amount:    moneyString(row.AmountCents),
			Raw:       row.AmountCents,
		})
		sections[idx].TotalRaw += row.AmountCents
	}

	for idx := range sections {
		sort.SliceStable(sections[idx].Rows, func(i, j int) bool {
			left := strings.ToLower(sections[idx].Rows[i].Name + "|" + sections[idx].Rows[i].Reference)
			right := strings.ToLower(sections[idx].Rows[j].Name + "|" + sections[idx].Rows[j].Reference)
			return left < right
		})
		sections[idx].Total = moneyString(sections[idx].TotalRaw)
	}
	report.Sections = sections
	report.CashSalesRaw = sections[sectionIndex["cash_sales"]].TotalRaw
	report.ChargeSalesRaw = sections[sectionIndex["charge_sales"]].TotalRaw
	report.CashReceiptsRaw = sections[sectionIndex["cash_receipts"]].TotalRaw
	report.DisbursementsRaw = sections[sectionIndex["disbursements"]].TotalRaw
	report.CheckDepositsRaw = sections[sectionIndex["check_deposits"]].TotalRaw
	report.TotalCashRemitRaw = report.CashSalesRaw + report.CashReceiptsRaw - report.DisbursementsRaw
	report.TotalRemitRaw = report.TotalCashRemitRaw + report.CheckDepositsRaw
	report.CashSales = moneyString(report.CashSalesRaw)
	report.ChargeSales = moneyString(report.ChargeSalesRaw)
	report.CashReceipts = moneyString(report.CashReceiptsRaw)
	report.Disbursements = moneyString(report.DisbursementsRaw)
	report.CheckDeposits = moneyString(report.CheckDepositsRaw)
	report.TotalCashRemit = moneyString(report.TotalCashRemitRaw)
	report.TotalRemit = moneyString(report.TotalRemitRaw)
}

func (report *stockSalesTransferReportData) build(rows []models.StockSalesTransferReportRow) {
	categoryMap := map[string]*stockSalesTransferCategory{}
	for _, row := range rows {
		categoryName := strings.TrimSpace(row.Category)
		if categoryName == "" {
			categoryName = "Uncategorized"
		}
		category := categoryMap[categoryName]
		if category == nil {
			category = &stockSalesTransferCategory{Name: categoryName}
			categoryMap[categoryName] = category
		}
		stockCode := strings.TrimSpace(row.StockCode)
		if stockCode == "" {
			stockCode = "N/A"
		}
		stockName := strings.TrimSpace(row.StockName)
		if stockName == "" {
			stockName = "No Stock"
		}
		totalQty := row.SalesQty + row.TransferQty
		category.Rows = append(category.Rows, stockSalesTransferLine{
			StockCode:   stockCode,
			StockName:   stockName,
			SalesQty:    qtyString(row.SalesQty),
			TransferQty: qtyString(row.TransferQty),
			TotalQty:    qtyString(totalQty),
			SalesRaw:    row.SalesQty,
			TransferRaw: row.TransferQty,
			TotalRaw:    totalQty,
		})
		category.TotalSalesRaw += row.SalesQty
		category.TotalTransferRaw += row.TransferQty
		category.GrandTotalRaw += totalQty
		report.TotalSalesRaw += row.SalesQty
		report.TotalTransferRaw += row.TransferQty
		report.GrandTotalRaw += totalQty
	}

	categoryNames := make([]string, 0, len(categoryMap))
	for categoryName := range categoryMap {
		categoryNames = append(categoryNames, categoryName)
	}
	sort.Slice(categoryNames, func(i, j int) bool {
		return strings.ToLower(categoryNames[i]) < strings.ToLower(categoryNames[j])
	})

	for _, categoryName := range categoryNames {
		category := categoryMap[categoryName]
		sort.SliceStable(category.Rows, func(i, j int) bool {
			left := strings.ToLower(category.Rows[i].StockCode + "|" + category.Rows[i].StockName)
			right := strings.ToLower(category.Rows[j].StockCode + "|" + category.Rows[j].StockName)
			return left < right
		})
		category.TotalSales = qtyString(category.TotalSalesRaw)
		category.TotalTransfer = qtyString(category.TotalTransferRaw)
		category.GrandTotal = qtyString(category.GrandTotalRaw)
		report.Categories = append(report.Categories, *category)
	}
	report.TotalSales = qtyString(report.TotalSalesRaw)
	report.TotalTransfer = qtyString(report.TotalTransferRaw)
	report.GrandTotal = qtyString(report.GrandTotalRaw)
	if len(report.Categories) > 0 {
		report.TotalPages = len(report.Categories)
	}
}

func (report *stockSalesTransferAmountReportData) build(rows []models.StockSalesTransferAmountReportRow) {
	for _, row := range rows {
		category := strings.TrimSpace(row.Category)
		if category == "" {
			category = "Uncategorized"
		}
		totalSales := row.CashSalesCents + row.ChargeSalesCents
		total := totalSales + row.TransferCents
		line := stockSalesTransferAmountLine{
			Category:              category,
			CashSales:             moneyString(row.CashSalesCents),
			ChargeSales:           moneyString(row.ChargeSalesCents),
			TotalSales:            moneyString(totalSales),
			Transfer:              moneyString(row.TransferCents),
			Total:                 moneyString(total),
			SalesMarkup:           moneyString(row.SalesMarkupCents),
			SalesMarkupPercent:    percentString(row.SalesMarkupCents, totalSales),
			TransferMarkup:        moneyString(row.TransferMarkupCents),
			TransferMarkupPercent: percentString(row.TransferMarkupCents, row.TransferCents),
			CashSalesRaw:          row.CashSalesCents,
			ChargeSalesRaw:        row.ChargeSalesCents,
			TotalSalesRaw:         totalSales,
			TransferRaw:           row.TransferCents,
			TotalRaw:              total,
			SalesMarkupRaw:        row.SalesMarkupCents,
			TransferMarkupRaw:     row.TransferMarkupCents,
		}
		report.Rows = append(report.Rows, line)
		report.TotalCashSalesRaw += row.CashSalesCents
		report.TotalChargeSalesRaw += row.ChargeSalesCents
		report.TotalSalesRaw += totalSales
		report.TotalTransferRaw += row.TransferCents
		report.GrandTotalRaw += total
		report.TotalSalesMarkupRaw += row.SalesMarkupCents
		report.TotalTransferMarkupRaw += row.TransferMarkupCents
	}

	sort.SliceStable(report.Rows, func(i, j int) bool {
		return strings.ToLower(report.Rows[i].Category) < strings.ToLower(report.Rows[j].Category)
	})
	report.TotalCashSales = moneyString(report.TotalCashSalesRaw)
	report.TotalChargeSales = moneyString(report.TotalChargeSalesRaw)
	report.TotalSales = moneyString(report.TotalSalesRaw)
	report.TotalTransfer = moneyString(report.TotalTransferRaw)
	report.GrandTotal = moneyString(report.GrandTotalRaw)
	report.TotalSalesMarkup = moneyString(report.TotalSalesMarkupRaw)
	report.TotalTransferMarkup = moneyString(report.TotalTransferMarkupRaw)
	report.TotalSalesMarkupPercent = percentString(report.TotalSalesMarkupRaw, report.TotalSalesRaw)
	report.TotalTransferMarkupPct = percentString(report.TotalTransferMarkupRaw, report.TotalTransferRaw)
}

func (report *stockTransferSummaryReportData) build(rows []models.StockTransferSummaryReportRow) {
	categoryMap := map[string]*stockTransferSummaryCategory{}
	branchMap := map[string]map[string]*stockTransferSummaryBranch{}
	for _, row := range rows {
		categoryName := strings.TrimSpace(row.Category)
		if categoryName == "" {
			categoryName = "Uncategorized"
		}
		branchName := strings.TrimSpace(row.Branch)
		if branchName == "" {
			branchName = "No Branch"
		}
		category := categoryMap[categoryName]
		if category == nil {
			category = &stockTransferSummaryCategory{Name: categoryName}
			categoryMap[categoryName] = category
			branchMap[categoryName] = map[string]*stockTransferSummaryBranch{}
		}
		branch := branchMap[categoryName][branchName]
		if branch == nil {
			branch = &stockTransferSummaryBranch{Name: branchName}
			branchMap[categoryName][branchName] = branch
		}
		reference := strings.TrimSpace(row.Reference)
		if reference == "" {
			reference = "N/A"
		}
		code := strings.TrimSpace(row.StockCode)
		if code == "" {
			code = "N/A"
		}
		stockName := strings.TrimSpace(row.StockName)
		if stockName == "" {
			stockName = "No Stock"
		}
		branch.Rows = append(branch.Rows, stockTransferSummaryLine{
			Reference: reference,
			Date:      strings.TrimSpace(row.TransferDate),
			Code:      code,
			StockName: stockName,
			Quantity:  qtyDecimalString(row.Quantity),
			Amount:    moneyString(row.AmountCents),
			QtyRaw:    row.Quantity,
			AmtRaw:    row.AmountCents,
		})
		branch.TotalQtyRaw += row.Quantity
		branch.TotalAmtRaw += row.AmountCents
		category.TotalQtyRaw += row.Quantity
		category.TotalAmtRaw += row.AmountCents
		report.TotalQtyRaw += row.Quantity
		report.TotalAmountRaw += row.AmountCents
	}

	categoryNames := make([]string, 0, len(categoryMap))
	for categoryName := range categoryMap {
		categoryNames = append(categoryNames, categoryName)
	}
	sort.Slice(categoryNames, func(i, j int) bool {
		return strings.ToLower(categoryNames[i]) < strings.ToLower(categoryNames[j])
	})
	for _, categoryName := range categoryNames {
		category := categoryMap[categoryName]
		branchNames := make([]string, 0, len(branchMap[categoryName]))
		for branchName := range branchMap[categoryName] {
			branchNames = append(branchNames, branchName)
		}
		sort.Slice(branchNames, func(i, j int) bool {
			return strings.ToLower(branchNames[i]) < strings.ToLower(branchNames[j])
		})
		for _, branchName := range branchNames {
			branch := branchMap[categoryName][branchName]
			sort.SliceStable(branch.Rows, func(i, j int) bool {
				left := strings.ToLower(branch.Rows[i].Reference + "|" + branch.Rows[i].Date + "|" + branch.Rows[i].Code)
				right := strings.ToLower(branch.Rows[j].Reference + "|" + branch.Rows[j].Date + "|" + branch.Rows[j].Code)
				return left < right
			})
			branch.TotalQuantity = qtyDecimalString(branch.TotalQtyRaw)
			branch.TotalAmount = moneyString(branch.TotalAmtRaw)
			category.Branches = append(category.Branches, *branch)
		}
		category.TotalQuantity = qtyDecimalString(category.TotalQtyRaw)
		category.TotalAmount = moneyString(category.TotalAmtRaw)
		report.Categories = append(report.Categories, *category)
	}
	report.TotalQuantity = qtyDecimalString(report.TotalQtyRaw)
	report.TotalAmount = moneyString(report.TotalAmountRaw)
	if len(report.Categories) > 0 {
		report.TotalPages = len(report.Categories)
	}
}

func (report *stockTransferSummaryByItemReportData) build(rows []models.StockTransferSummaryByItemReportRow) {
	categoryMap := map[string]*stockTransferSummaryByItemCategory{}
	rowsByStock := map[string]map[string]*stockTransferSummaryByItemLine{}
	for _, row := range rows {
		categoryName := defaultText(row.Category, "Uncategorized")
		category := categoryMap[categoryName]
		if category == nil {
			category = &stockTransferSummaryByItemCategory{Name: categoryName}
			categoryMap[categoryName] = category
			rowsByStock[categoryName] = map[string]*stockTransferSummaryByItemLine{}
		}

		stockCode := defaultText(row.StockCode, "N/A")
		stockName := defaultText(row.StockName, "No Stock")
		stockKey := strings.ToLower(stockName + "|" + stockCode)
		line := rowsByStock[categoryName][stockKey]
		if line == nil {
			line = &stockTransferSummaryByItemLine{Code: stockCode, StockName: stockName}
			rowsByStock[categoryName][stockKey] = line
		}
		line.QtyRaw += row.Quantity
		line.AmtRaw += row.AmountCents
		category.TotalQtyRaw += row.Quantity
		category.TotalAmtRaw += row.AmountCents
		report.TotalQtyRaw += row.Quantity
		report.TotalAmountRaw += row.AmountCents
	}

	categoryNames := make([]string, 0, len(categoryMap))
	for categoryName := range categoryMap {
		categoryNames = append(categoryNames, categoryName)
	}
	sort.Slice(categoryNames, func(i, j int) bool {
		return strings.ToLower(categoryNames[i]) < strings.ToLower(categoryNames[j])
	})
	for _, categoryName := range categoryNames {
		category := categoryMap[categoryName]
		stockKeys := make([]string, 0, len(rowsByStock[categoryName]))
		for stockKey := range rowsByStock[categoryName] {
			stockKeys = append(stockKeys, stockKey)
		}
		sort.Strings(stockKeys)
		for _, stockKey := range stockKeys {
			line := rowsByStock[categoryName][stockKey]
			line.Quantity = qtyDecimalString(line.QtyRaw)
			line.Amount = moneyString(line.AmtRaw)
			category.Rows = append(category.Rows, *line)
		}
		category.TotalQuantity = qtyDecimalString(category.TotalQtyRaw)
		category.TotalAmount = moneyString(category.TotalAmtRaw)
		report.Categories = append(report.Categories, *category)
	}
	report.TotalQuantity = qtyDecimalString(report.TotalQtyRaw)
	report.TotalAmount = moneyString(report.TotalAmountRaw)
	if len(report.Categories) > 0 {
		report.TotalPages = len(report.Categories)
	}
}

func (report *stockTransferByStockNameReportData) build(rows []models.StockTransferByStockNameReportRow) {
	categoryMap := map[string]*stockTransferByStockNameCategory{}
	stockMap := map[string]map[string]*stockTransferByStockNameStock{}
	branchMap := map[string]map[string]map[string]*stockTransferByStockNameBranch{}
	for _, row := range rows {
		categoryName := strings.TrimSpace(row.Category)
		if categoryName == "" {
			categoryName = "Uncategorized"
		}
		stockCode := strings.TrimSpace(row.StockCode)
		if stockCode == "" {
			stockCode = "N/A"
		}
		stockName := strings.TrimSpace(row.StockName)
		if stockName == "" {
			stockName = "No Stock"
		}
		branchName := strings.TrimSpace(row.Branch)
		if branchName == "" {
			branchName = "No Branch"
		}

		category := categoryMap[categoryName]
		if category == nil {
			category = &stockTransferByStockNameCategory{Name: categoryName}
			categoryMap[categoryName] = category
			stockMap[categoryName] = map[string]*stockTransferByStockNameStock{}
			branchMap[categoryName] = map[string]map[string]*stockTransferByStockNameBranch{}
		}
		stockKey := strings.ToLower(stockName + "|" + stockCode)
		stock := stockMap[categoryName][stockKey]
		if stock == nil {
			stock = &stockTransferByStockNameStock{StockCode: stockCode, StockName: stockName}
			stockMap[categoryName][stockKey] = stock
			branchMap[categoryName][stockKey] = map[string]*stockTransferByStockNameBranch{}
		}
		branch := branchMap[categoryName][stockKey][branchName]
		if branch == nil {
			branch = &stockTransferByStockNameBranch{Name: branchName}
			branchMap[categoryName][stockKey][branchName] = branch
		}

		reference := strings.TrimSpace(row.Reference)
		if reference == "" {
			reference = "N/A"
		}
		transferID := strings.TrimSpace(row.TransferID)
		if transferID == "" {
			transferID = "N/A"
		}
		branch.Rows = append(branch.Rows, stockTransferByStockNameLine{
			Reference:  reference,
			Date:       strings.TrimSpace(row.TransferDate),
			TransferID: transferID,
			Branch:     branchName,
			Quantity:   qtyDecimalString(row.Quantity),
			Amount:     moneyString(row.AmountCents),
			QtyRaw:     row.Quantity,
			AmtRaw:     row.AmountCents,
		})
		branch.TotalQtyRaw += row.Quantity
		branch.TotalAmtRaw += row.AmountCents
		stock.TotalQtyRaw += row.Quantity
		stock.TotalAmtRaw += row.AmountCents
		category.TotalQtyRaw += row.Quantity
		category.TotalAmtRaw += row.AmountCents
		report.TotalQtyRaw += row.Quantity
		report.TotalAmountRaw += row.AmountCents
	}

	categoryNames := make([]string, 0, len(categoryMap))
	for categoryName := range categoryMap {
		categoryNames = append(categoryNames, categoryName)
	}
	sort.Slice(categoryNames, func(i, j int) bool {
		return strings.ToLower(categoryNames[i]) < strings.ToLower(categoryNames[j])
	})
	for _, categoryName := range categoryNames {
		category := categoryMap[categoryName]
		stockKeys := make([]string, 0, len(stockMap[categoryName]))
		for stockKey := range stockMap[categoryName] {
			stockKeys = append(stockKeys, stockKey)
		}
		sort.Strings(stockKeys)
		for _, stockKey := range stockKeys {
			stock := stockMap[categoryName][stockKey]
			branchNames := make([]string, 0, len(branchMap[categoryName][stockKey]))
			for branchName := range branchMap[categoryName][stockKey] {
				branchNames = append(branchNames, branchName)
			}
			sort.Slice(branchNames, func(i, j int) bool {
				return strings.ToLower(branchNames[i]) < strings.ToLower(branchNames[j])
			})
			for _, branchName := range branchNames {
				branch := branchMap[categoryName][stockKey][branchName]
				sort.SliceStable(branch.Rows, func(i, j int) bool {
					left := strings.ToLower(branch.Rows[i].Date + "|" + branch.Rows[i].Reference + "|" + branch.Rows[i].TransferID)
					right := strings.ToLower(branch.Rows[j].Date + "|" + branch.Rows[j].Reference + "|" + branch.Rows[j].TransferID)
					return left < right
				})
				branch.TotalQuantity = qtyDecimalString(branch.TotalQtyRaw)
				branch.TotalAmount = moneyString(branch.TotalAmtRaw)
				stock.Branches = append(stock.Branches, *branch)
			}
			stock.TotalQuantity = qtyDecimalString(stock.TotalQtyRaw)
			stock.TotalAmount = moneyString(stock.TotalAmtRaw)
			category.Stocks = append(category.Stocks, *stock)
		}
		category.TotalQuantity = qtyDecimalString(category.TotalQtyRaw)
		category.TotalAmount = moneyString(category.TotalAmtRaw)
		report.Categories = append(report.Categories, *category)
	}
	report.TotalQuantity = qtyDecimalString(report.TotalQtyRaw)
	report.TotalAmount = moneyString(report.TotalAmountRaw)
	if len(report.Categories) > 0 {
		report.TotalPages = len(report.Categories)
	}
}

func (report *stockTransferByBranchReportData) build(rows []models.StockTransferByBranchReportRow) {
	branchMap := map[string]*stockTransferByBranchGroup{}
	categoryMap := map[string]map[string]*stockTransferByBranchCategory{}
	for _, row := range rows {
		branchName := defaultText(row.Branch, "No Branch")
		categoryName := defaultText(row.Category, "Uncategorized")

		branch := branchMap[branchName]
		if branch == nil {
			branch = &stockTransferByBranchGroup{Name: branchName}
			branchMap[branchName] = branch
			categoryMap[branchName] = map[string]*stockTransferByBranchCategory{}
		}
		category := categoryMap[branchName][categoryName]
		if category == nil {
			category = &stockTransferByBranchCategory{Name: categoryName}
			categoryMap[branchName][categoryName] = category
		}

		category.Rows = append(category.Rows, stockTransferByBranchLine{
			Reference: defaultText(row.Reference, "N/A"),
			Date:      strings.TrimSpace(row.TransferDate),
			Code:      defaultText(row.StockCode, "N/A"),
			StockName: defaultText(row.StockName, "No Stock"),
			Quantity:  qtyDecimalString(row.Quantity),
			Amount:    moneyString(row.AmountCents),
			QtyRaw:    row.Quantity,
			AmtRaw:    row.AmountCents,
		})
		category.TotalQtyRaw += row.Quantity
		category.TotalAmtRaw += row.AmountCents
		branch.TotalQtyRaw += row.Quantity
		branch.TotalAmtRaw += row.AmountCents
		report.TotalQtyRaw += row.Quantity
		report.TotalAmountRaw += row.AmountCents
	}

	branchNames := make([]string, 0, len(branchMap))
	for branchName := range branchMap {
		branchNames = append(branchNames, branchName)
	}
	sort.Slice(branchNames, func(i, j int) bool {
		return strings.ToLower(branchNames[i]) < strings.ToLower(branchNames[j])
	})
	for _, branchName := range branchNames {
		branch := branchMap[branchName]
		categoryNames := make([]string, 0, len(categoryMap[branchName]))
		for categoryName := range categoryMap[branchName] {
			categoryNames = append(categoryNames, categoryName)
		}
		sort.Slice(categoryNames, func(i, j int) bool {
			return strings.ToLower(categoryNames[i]) < strings.ToLower(categoryNames[j])
		})
		for _, categoryName := range categoryNames {
			category := categoryMap[branchName][categoryName]
			sort.SliceStable(category.Rows, func(i, j int) bool {
				left := strings.ToLower(category.Rows[i].Reference + "|" + category.Rows[i].Date + "|" + category.Rows[i].Code)
				right := strings.ToLower(category.Rows[j].Reference + "|" + category.Rows[j].Date + "|" + category.Rows[j].Code)
				return left < right
			})
			category.TotalQuantity = qtyDecimalString(category.TotalQtyRaw)
			category.TotalAmount = moneyString(category.TotalAmtRaw)
			branch.Categories = append(branch.Categories, *category)
		}
		branch.TotalQuantity = qtyDecimalString(branch.TotalQtyRaw)
		branch.TotalAmount = moneyString(branch.TotalAmtRaw)
		report.Branches = append(report.Branches, *branch)
	}
	report.TotalQuantity = qtyDecimalString(report.TotalQtyRaw)
	report.TotalAmount = moneyString(report.TotalAmountRaw)
	if len(report.Branches) > 0 {
		report.TotalPages = len(report.Branches)
	}
}

func (report *stockTransferByEntryIDReportData) build(rows []models.StockTransferByEntryIDReportRow) {
	groupsByEntryID := map[string]*stockTransferByEntryIDGroup{}
	for _, row := range rows {
		entryID := defaultText(row.EntryID, "No Entry ID")
		group := groupsByEntryID[entryID]
		if group == nil {
			group = &stockTransferByEntryIDGroup{
				EntryID:    entryID,
				Reference:  defaultText(row.Reference, "N/A"),
				TransferID: defaultText(row.TransferID, "N/A"),
				Date:       strings.TrimSpace(row.TransferDate),
				Branch:     defaultText(row.Branch, "No Branch"),
			}
			groupsByEntryID[entryID] = group
		}
		group.Rows = append(group.Rows, stockTransferByEntryIDLine{
			Code:     defaultText(row.StockCode, "N/A"),
			Stock:    defaultText(row.StockName, "No Stock"),
			Branch:   defaultText(row.Branch, "No Branch"),
			Quantity: qtyDecimalString(row.Quantity),
			Amount:   moneyString(row.AmountCents),
			QtyRaw:   row.Quantity,
			AmtRaw:   row.AmountCents,
		})
		group.TotalQtyRaw += row.Quantity
		group.TotalAmtRaw += row.AmountCents
		report.TotalQtyRaw += row.Quantity
		report.TotalAmountRaw += row.AmountCents
	}
	entryIDs := make([]string, 0, len(groupsByEntryID))
	for entryID := range groupsByEntryID {
		entryIDs = append(entryIDs, entryID)
	}
	sort.Slice(entryIDs, func(i, j int) bool { return strings.ToLower(entryIDs[i]) < strings.ToLower(entryIDs[j]) })
	for _, entryID := range entryIDs {
		group := groupsByEntryID[entryID]
		sort.SliceStable(group.Rows, func(i, j int) bool {
			return strings.ToLower(group.Rows[i].Stock+"|"+group.Rows[i].Code) < strings.ToLower(group.Rows[j].Stock+"|"+group.Rows[j].Code)
		})
		group.TotalQuantity = qtyDecimalString(group.TotalQtyRaw)
		group.TotalAmount = moneyString(group.TotalAmtRaw)
		report.Groups = append(report.Groups, *group)
	}
	report.TotalQuantity = qtyDecimalString(report.TotalQtyRaw)
	report.TotalAmount = moneyString(report.TotalAmountRaw)
	if len(report.Groups) > 0 {
		report.TotalPages = len(report.Groups)
	}
}

func (report *stockTransferSummaryByEntryIDReportData) build(rows []models.StockTransferByEntryIDReportRow) {
	branchMap := map[string]*stockTransferSummaryByEntryIDBranch{}
	lineMap := map[string]map[string]*stockTransferSummaryByEntryIDLine{}
	for _, row := range rows {
		branchName := defaultText(row.Branch, "No Branch")
		branch := branchMap[branchName]
		if branch == nil {
			branch = &stockTransferSummaryByEntryIDBranch{Name: branchName}
			branchMap[branchName] = branch
			lineMap[branchName] = map[string]*stockTransferSummaryByEntryIDLine{}
		}

		entryID := defaultText(row.EntryID, "No Entry ID")
		lineKey := strings.ToLower(entryID)
		line := lineMap[branchName][lineKey]
		if line == nil {
			line = &stockTransferSummaryByEntryIDLine{
				EntryID:   entryID,
				EntryDate: strings.TrimSpace(row.TransferDate),
				Remarks:   stockTransferSummaryByEntryIDRemarks(row),
			}
			lineMap[branchName][lineKey] = line
		}
		line.TotalQtyRaw += row.Quantity
		line.TotalAmtRaw += row.AmountCents
		line.NetTotalRaw = row.NetCents
	}

	branchNames := make([]string, 0, len(branchMap))
	for branchName := range branchMap {
		branchNames = append(branchNames, branchName)
	}
	sort.Slice(branchNames, func(i, j int) bool {
		return strings.ToLower(branchNames[i]) < strings.ToLower(branchNames[j])
	})
	for _, branchName := range branchNames {
		branch := branchMap[branchName]
		lineKeys := make([]string, 0, len(lineMap[branchName]))
		for lineKey := range lineMap[branchName] {
			lineKeys = append(lineKeys, lineKey)
		}
		sort.Slice(lineKeys, func(i, j int) bool {
			left := lineMap[branchName][lineKeys[i]]
			right := lineMap[branchName][lineKeys[j]]
			leftKey := strings.ToLower(left.EntryDate + "|" + left.EntryID)
			rightKey := strings.ToLower(right.EntryDate + "|" + right.EntryID)
			return leftKey < rightKey
		})
		for _, lineKey := range lineKeys {
			line := lineMap[branchName][lineKey]
			line.TotalQuantity = qtyDecimalString(line.TotalQtyRaw)
			line.TotalAmount = moneyString(line.TotalAmtRaw)
			line.NetTotal = moneyString(line.NetTotalRaw)
			branch.TotalQtyRaw += line.TotalQtyRaw
			branch.TotalAmtRaw += line.TotalAmtRaw
			branch.NetTotalRaw += line.NetTotalRaw
			branch.Rows = append(branch.Rows, *line)
		}
		branch.TotalQuantity = qtyDecimalString(branch.TotalQtyRaw)
		branch.TotalAmount = moneyString(branch.TotalAmtRaw)
		branch.NetTotal = moneyString(branch.NetTotalRaw)
		report.TotalQtyRaw += branch.TotalQtyRaw
		report.TotalAmountRaw += branch.TotalAmtRaw
		report.NetTotalRaw += branch.NetTotalRaw
		report.Branches = append(report.Branches, *branch)
	}
	report.TotalQuantity = qtyDecimalString(report.TotalQtyRaw)
	report.TotalAmount = moneyString(report.TotalAmountRaw)
	report.NetTotal = moneyString(report.NetTotalRaw)
	if len(report.Branches) > 0 {
		report.TotalPages = len(report.Branches)
	}
}

func stockTransferSummaryByEntryIDRemarks(row models.StockTransferByEntryIDReportRow) string {
	if remarks := strings.TrimSpace(row.Remarks); remarks != "" {
		return remarks
	}
	parts := make([]string, 0, 2)
	if transferID := strings.TrimSpace(row.TransferID); transferID != "" {
		parts = append(parts, transferID)
	}
	reference := strings.TrimSpace(row.Reference)
	if reference != "" && !strings.EqualFold(reference, strings.TrimSpace(row.EntryID)) && !strings.EqualFold(reference, strings.TrimSpace(row.TransferID)) {
		parts = append(parts, reference)
	}
	if len(parts) == 0 {
		return "N/A"
	}
	return strings.Join(parts, " ")
}

func (report *stockTransferMarkupByTransactionReportData) build(rows []models.StockTransferMarkupByTransactionReportRow) {
	for _, row := range rows {
		entryID := defaultText(row.EntryID, "No Entry ID")
		transferTo := defaultText(row.TransferTo, "No Branch")
		receiptNo := defaultText(row.ReceiptNo, "No Receipt")
		itemGroup := defaultText(row.ItemGroup, "Uncategorized")
		line := stockTransferMarkupByTransactionLine{
			TransferDate:  strings.TrimSpace(row.TransferDate),
			EntryID:       entryID,
			TransferTo:    transferTo,
			ReceiptNo:     receiptNo,
			ItemGroup:     itemGroup,
			Markup:        moneyString(row.MarkupCents),
			MarkupPercent: percentString(row.MarkupCents, row.CapitalCents),
			MarkupRaw:     row.MarkupCents,
			CapitalRaw:    row.CapitalCents,
			Negative:      row.MarkupCents < 0,
		}
		report.Rows = append(report.Rows, line)
		report.TotalMarkupRaw += row.MarkupCents
		report.TotalCapitalRaw += row.CapitalCents
	}
	report.TotalMarkup = moneyString(report.TotalMarkupRaw)
	report.TotalMarkupPercent = percentString(report.TotalMarkupRaw, report.TotalCapitalRaw)
	if len(report.Rows) > 0 {
		for start := 0; start < len(report.Rows); start += 40 {
			end := start + 40
			if end > len(report.Rows) {
				end = len(report.Rows)
			}
			report.Pages = append(report.Pages, stockTransferMarkupByTransactionPage{
				Number: len(report.Pages) + 1,
				Rows:   report.Rows[start:end],
				Last:   end == len(report.Rows),
			})
		}
		report.TotalPages = len(report.Pages)
	}
}

func defaultText(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func (report *stockLedgerReportData) build(rows []models.StockLedgerReportRow, from, to time.Time) {
	type stockState struct {
		category string
		code     string
		name     string
		rows     []models.StockLedgerReportRow
	}

	states := map[string]*stockState{}
	for _, row := range rows {
		key := strings.TrimSpace(row.StockID)
		if key == "" {
			key = strings.TrimSpace(row.StockCode) + "|" + strings.TrimSpace(row.StockName)
		}
		if key == "|" {
			key = "unknown"
		}
		state := states[key]
		if state == nil {
			category := strings.TrimSpace(row.Category)
			if category == "" {
				category = "Uncategorized"
			}
			code := strings.TrimSpace(row.StockCode)
			if code == "" {
				code = "N/A"
			}
			name := strings.TrimSpace(row.StockName)
			if name == "" {
				name = "No Stock"
			}
			state = &stockState{category: category, code: code, name: name}
			states[key] = state
		}
		if strings.TrimSpace(row.EntryDate) != "" || row.QtyDelta != 0 {
			state.rows = append(state.rows, row)
		}
	}

	categoryMap := map[string][]*stockState{}
	for _, state := range states {
		categoryMap[state.category] = append(categoryMap[state.category], state)
	}

	categoryNames := make([]string, 0, len(categoryMap))
	for categoryName := range categoryMap {
		categoryNames = append(categoryNames, categoryName)
	}
	sort.Slice(categoryNames, func(i, j int) bool {
		return strings.ToLower(categoryNames[i]) < strings.ToLower(categoryNames[j])
	})

	for _, categoryName := range categoryNames {
		stocks := categoryMap[categoryName]
		sort.SliceStable(stocks, func(i, j int) bool {
			left := strings.ToLower(stocks[i].code + "|" + stocks[i].name)
			right := strings.ToLower(stocks[j].code + "|" + stocks[j].name)
			return left < right
		})
		category := stockLedgerCategory{Name: categoryName}
		for _, state := range stocks {
			sort.SliceStable(state.rows, func(i, j int) bool {
				left := parseReportDate(rowDateForParse(state.rows[i].EntryDate), to)
				right := parseReportDate(rowDateForParse(state.rows[j].EntryDate), to)
				if left.Equal(right) {
					return strings.ToLower(state.rows[i].Reference) < strings.ToLower(state.rows[j].Reference)
				}
				return left.Before(right)
			})

			var forwarded, balance int64
			for _, row := range state.rows {
				rowDate := parseReportDate(rowDateForParse(row.EntryDate), to)
				if rowDate.Before(from) {
					forwarded += row.QtyDelta
				}
			}
			balance = forwarded
			group := stockLedgerStockGroup{
				StockCode: state.code,
				StockName: state.name,
				Rows: []stockLedgerLine{{
					Reference: "Forwarded",
					Date:      from.AddDate(0, 0, -1).Format("01/02/2006"),
					Company:   "Forwarded Balance",
					Balance:   qtyDecimalString(balance),
				}},
			}
			for _, row := range state.rows {
				rowDate := parseReportDate(rowDateForParse(row.EntryDate), to)
				if rowDate.Before(from) || rowDate.After(to) {
					continue
				}
				balance += row.QtyDelta
				line := stockLedgerLine{
					Reference: stockLedgerReference(row),
					Date:      row.EntryDate,
					Company:   strings.TrimSpace(row.Company),
					Balance:   qtyDecimalString(balance),
				}
				if line.Company == "" {
					line.Company = stockLedgerCompany(row.Kind)
				}
				if row.QtyDelta > 0 {
					line.Debit = qtyDecimalString(row.QtyDelta)
				} else if row.QtyDelta < 0 {
					line.Credit = qtyDecimalString(-row.QtyDelta)
				}
				group.Rows = append(group.Rows, line)
			}
			category.Stocks = append(category.Stocks, group)
		}
		report.Categories = append(report.Categories, category)
	}
	if len(report.Categories) > 0 {
		report.TotalPages = len(report.Categories)
	}
}

func (report *stockAgingReportData) build(rows []models.StockAgingReportRow) {
	categoryMap := map[string]*stockAgingCategory{}
	report.Totals.Raw = make([]int64, 6)
	for _, row := range rows {
		categoryName := strings.TrimSpace(row.Category)
		if categoryName == "" {
			categoryName = "Uncategorized"
		}
		category := categoryMap[categoryName]
		if category == nil {
			category = &stockAgingCategory{
				Name: categoryName,
				Totals: stockAgingTotals{
					Raw: make([]int64, 6),
				},
			}
			categoryMap[categoryName] = category
		}
		stockCode := strings.TrimSpace(row.StockCode)
		if stockCode == "" {
			stockCode = "N/A"
		}
		stockName := strings.TrimSpace(row.StockName)
		if stockName == "" {
			stockName = "No Stock"
		}
		buckets := []int64{row.Bucket0, row.Bucket1, row.Bucket2, row.Bucket3, row.Bucket4, row.Bucket5}
		line := stockAgingLine{
			StockCode: stockCode,
			StockName: stockName,
			Raw:       buckets,
		}
		for idx, value := range buckets {
			line.Buckets = append(line.Buckets, qtyString(value))
			category.Totals.Raw[idx] += value
			report.Totals.Raw[idx] += value
		}
		category.Rows = append(category.Rows, line)
	}

	categoryNames := make([]string, 0, len(categoryMap))
	for categoryName := range categoryMap {
		categoryNames = append(categoryNames, categoryName)
	}
	sort.Slice(categoryNames, func(i, j int) bool {
		return strings.ToLower(categoryNames[i]) < strings.ToLower(categoryNames[j])
	})
	for _, categoryName := range categoryNames {
		category := categoryMap[categoryName]
		sort.SliceStable(category.Rows, func(i, j int) bool {
			left := strings.ToLower(category.Rows[i].StockCode + "|" + category.Rows[i].StockName)
			right := strings.ToLower(category.Rows[j].StockCode + "|" + category.Rows[j].StockName)
			return left < right
		})
		for _, value := range category.Totals.Raw {
			category.Totals.Buckets = append(category.Totals.Buckets, qtyString(value))
		}
		report.Categories = append(report.Categories, *category)
	}
	for _, value := range report.Totals.Raw {
		report.Totals.Buckets = append(report.Totals.Buckets, qtyString(value))
	}
	if len(report.Categories) > 0 {
		report.TotalPages = len(report.Categories)
	}
}

func (report *stockReorderPointReportData) build(rows []models.StockReorderPointReportRow) {
	categoryMap := map[string]*stockReorderPointCategory{}
	for _, row := range rows {
		categoryName := strings.TrimSpace(row.Category)
		if categoryName == "" {
			categoryName = "Uncategorized"
		}
		category := categoryMap[categoryName]
		if category == nil {
			category = &stockReorderPointCategory{Name: categoryName}
			categoryMap[categoryName] = category
		}
		stockCode := strings.TrimSpace(row.StockCode)
		if stockCode == "" {
			stockCode = "N/A"
		}
		stockName := strings.TrimSpace(row.StockName)
		if stockName == "" {
			stockName = "No Stock"
		}
		deficit := row.Deficit
		if deficit < 0 {
			deficit = 0
		}
		category.Rows = append(category.Rows, stockReorderPointLine{
			StockCode:    stockCode,
			StockName:    stockName,
			SOH:          qtyDecimalString(row.SOH),
			MinInventory: qtyDecimalString(row.MinInventory),
			Deficit:      qtyDecimalString(deficit),
			DeficitRaw:   deficit,
		})
	}

	categoryNames := make([]string, 0, len(categoryMap))
	for categoryName := range categoryMap {
		categoryNames = append(categoryNames, categoryName)
	}
	sort.Slice(categoryNames, func(i, j int) bool {
		return strings.ToLower(categoryNames[i]) < strings.ToLower(categoryNames[j])
	})
	for _, categoryName := range categoryNames {
		category := categoryMap[categoryName]
		sort.SliceStable(category.Rows, func(i, j int) bool {
			left := strings.ToLower(category.Rows[i].StockCode + "|" + category.Rows[i].StockName)
			right := strings.ToLower(category.Rows[j].StockCode + "|" + category.Rows[j].StockName)
			return left < right
		})
		report.Categories = append(report.Categories, *category)
	}
	if len(report.Categories) > 0 {
		report.TotalPages = len(report.Categories)
	}
}

func (report *stockSummaryReportData) build(rows []models.StockSummaryReportRow) {
	categoryMap := map[string]*stockSummaryCategory{}
	for _, row := range rows {
		categoryName := strings.TrimSpace(row.Category)
		if categoryName == "" {
			categoryName = "Uncategorized"
		}
		category := categoryMap[categoryName]
		if category == nil {
			category = &stockSummaryCategory{Name: categoryName}
			categoryMap[categoryName] = category
		}
		stockCode := strings.TrimSpace(row.StockCode)
		if stockCode == "" {
			stockCode = "N/A"
		}
		stockName := strings.TrimSpace(row.StockName)
		if stockName == "" {
			stockName = "No Stock"
		}
		category.Rows = append(category.Rows, stockSummaryLine{
			StockCode:   stockCode,
			StockName:   stockName,
			SOH:         qtyDecimalString(row.SOH),
			UnitCost:    moneyString(row.UnitCostCents),
			Amount:      moneyString(row.AmountCents),
			SOHRaw:      row.SOH,
			UnitCostRaw: row.UnitCostCents,
			AmountRaw:   row.AmountCents,
		})
		category.TotalSOHRaw += row.SOH
		category.AmountRaw += row.AmountCents
		report.GrandSOHRaw += row.SOH
		report.GrandAmountRaw += row.AmountCents
	}

	categoryNames := make([]string, 0, len(categoryMap))
	for categoryName := range categoryMap {
		categoryNames = append(categoryNames, categoryName)
	}
	sort.Slice(categoryNames, func(i, j int) bool {
		return strings.ToLower(categoryNames[i]) < strings.ToLower(categoryNames[j])
	})
	for _, categoryName := range categoryNames {
		category := categoryMap[categoryName]
		sort.SliceStable(category.Rows, func(i, j int) bool {
			left := strings.ToLower(category.Rows[i].StockCode + "|" + category.Rows[i].StockName)
			right := strings.ToLower(category.Rows[j].StockCode + "|" + category.Rows[j].StockName)
			return left < right
		})
		category.TotalSOH = qtyDecimalString(category.TotalSOHRaw)
		category.Amount = moneyString(category.AmountRaw)
		report.Categories = append(report.Categories, *category)
	}
	report.GrandSOH = qtyDecimalString(report.GrandSOHRaw)
	report.GrandAmount = moneyString(report.GrandAmountRaw)
	if len(report.Categories) > 0 {
		report.TotalPages = len(report.Categories)
	}
}

func dailyDueCheckRowsFromIncoming(rows []models.IncomingCheckReportRow) []models.DailyDueCheckReportRow {
	dueRows := make([]models.DailyDueCheckReportRow, 0, len(rows))
	for _, row := range rows {
		dueRows = append(dueRows, models.DailyDueCheckReportRow{
			ClientName:  row.Payee,
			CheckDate:   row.CheckDate,
			CheckNumber: row.Number,
			BankName:    row.BankName,
			AmountCents: row.AmountCents,
		})
	}
	return dueRows
}

func (report *dailyDueCheckReportData) build(rows []models.DailyDueCheckReportRow, cutoff time.Time) {
	type datedRow struct {
		row       models.DailyDueCheckReportRow
		checkDate time.Time
	}

	rowsByDate := map[string][]datedRow{}
	dateValues := map[string]time.Time{}
	for _, row := range rows {
		checkDate := parseReportDate(rowDateForParse(row.CheckDate), cutoff)
		if !checkDate.After(cutoff) {
			continue
		}
		dateKey := checkDate.Format("2006-01-02")
		rowsByDate[dateKey] = append(rowsByDate[dateKey], datedRow{row: row, checkDate: checkDate})
		dateValues[dateKey] = checkDate
		report.GrandTotalRaw += row.AmountCents
	}
	report.GrandTotal = moneyString(report.GrandTotalRaw)

	dateKeys := make([]string, 0, len(rowsByDate))
	for dateKey := range rowsByDate {
		dateKeys = append(dateKeys, dateKey)
	}
	sort.Slice(dateKeys, func(i, j int) bool {
		return dateValues[dateKeys[i]].Before(dateValues[dateKeys[j]])
	})

	for _, dateKey := range dateKeys {
		groupRows := rowsByDate[dateKey]
		sort.SliceStable(groupRows, func(i, j int) bool {
			left := strings.ToLower(groupRows[i].row.ClientName + "|" + groupRows[i].row.CheckNumber)
			right := strings.ToLower(groupRows[j].row.ClientName + "|" + groupRows[j].row.CheckNumber)
			return left < right
		})

		group := dailyDueCheckDateGroup{
			CheckDate: dateValues[dateKey].Format("01/02/2006"),
			DateKey:   dateKey,
		}
		for _, item := range groupRows {
			client := strings.TrimSpace(item.row.ClientName)
			if client == "" {
				client = "No Client"
			}
			group.Rows = append(group.Rows, dailyDueCheckLine{
				ClientName:  client,
				CheckNumber: strings.TrimSpace(item.row.CheckNumber),
				BankName:    strings.TrimSpace(item.row.BankName),
				Amount:      moneyString(item.row.AmountCents),
				Raw:         item.row.AmountCents,
			})
			group.TotalRaw += item.row.AmountCents
		}
		group.Total = moneyString(group.TotalRaw)
		report.Groups = append(report.Groups, group)
	}
	if len(report.Groups) > 0 {
		report.TotalPages = len(report.Groups)
	}
}

func (report *incomingCheckCalendarReportData) setCalendarNavigation() {
	current := time.Date(report.Year, time.Month(report.Month), 1, 0, 0, 0, 0, time.Local)
	prev := current.AddDate(0, -1, 0)
	next := current.AddDate(0, 1, 0)
	report.MonthName = current.Month().String()
	report.PrevYear = prev.Year()
	report.PrevMonth = int(prev.Month())
	report.NextYear = next.Year()
	report.NextMonth = int(next.Month())
}

func (report *incomingCheckCalendarReportData) build(rows []models.IncomingCheckReportRow, monthStart time.Time) {
	monthEnd := monthStart.AddDate(0, 1, -1)
	daysByNumber := make(map[int]int, monthEnd.Day())

	for blank := 0; blank < int(monthStart.Weekday()); blank++ {
		report.Days = append(report.Days, incomingCheckCalendarDay{Blank: true})
	}

	for day := 1; day <= monthEnd.Day(); day++ {
		date := time.Date(monthStart.Year(), monthStart.Month(), day, 0, 0, 0, 0, monthStart.Location())
		calendarDay := incomingCheckCalendarDay{
			Day:      day,
			DateKey:  date.Format("2006-01-02"),
			DateText: date.Format("Jan 02"),
			Weekend:  incomingCheckWeekendClass(date.Weekday()),
			Total:    "0",
		}
		report.Days = append(report.Days, calendarDay)
		daysByNumber[day] = len(report.Days) - 1
	}

	for len(report.Days)%7 != 0 {
		report.Days = append(report.Days, incomingCheckCalendarDay{Blank: true})
	}

	for _, row := range rows {
		checkDate := parseReportDate(rowDateForParse(row.CheckDate), monthStart)
		if checkDate.Year() != monthStart.Year() || checkDate.Month() != monthStart.Month() {
			continue
		}
		dayIndex, ok := daysByNumber[checkDate.Day()]
		if !ok {
			continue
		}
		day := &report.Days[dayIndex]

		payee := strings.TrimSpace(row.Payee)
		if payee == "" {
			payee = "No Payee"
		}
		line := incomingCheckCalendarLine{
			Payee:     payee,
			Reference: strings.TrimSpace(row.Reference),
			Number:    strings.TrimSpace(row.Number),
			BankName:  strings.TrimSpace(row.BankName),
			Amount:    moneyString(row.AmountCents),
			Raw:       row.AmountCents,
		}
		day.Rows = append(day.Rows, line)
		day.TotalRaw += row.AmountCents
		report.MonthTotalRaw += row.AmountCents
	}

	for idx := range report.Days {
		if report.Days[idx].Blank {
			continue
		}
		sort.SliceStable(report.Days[idx].Rows, func(i, j int) bool {
			left := strings.ToLower(report.Days[idx].Rows[i].Payee + "|" + report.Days[idx].Rows[i].Number)
			right := strings.ToLower(report.Days[idx].Rows[j].Payee + "|" + report.Days[idx].Rows[j].Number)
			return left < right
		})
		if report.Days[idx].TotalRaw != 0 {
			report.Days[idx].Total = moneyString(report.Days[idx].TotalRaw)
		}
	}
	report.MonthTotal = moneyString(report.MonthTotalRaw)
}

func incomingCheckWeekendClass(weekday time.Weekday) string {
	switch weekday {
	case time.Sunday:
		return "sunday"
	case time.Saturday:
		return "saturday"
	default:
		return ""
	}
}

func rowDateForParse(value string) string {
	if t, err := time.Parse("01/02/2006", strings.TrimSpace(value)); err == nil {
		return t.Format("2006-01-02")
	}
	return value
}

func incomingCheckMonthLabel(value time.Time) string {
	months := []string{"Enero", "Pebrero", "Marso", "Abril", "Mayo", "Hunyo", "Hulyo", "Agosto", "Setyembre", "Oktubre", "Nobyembre", "Disyembre"}
	return months[int(value.Month())-1] + ", " + strconv.Itoa(value.Year())
}

func apLedgerDateString(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format("01/02/2006")
}

func apLedgerReference(row models.APLedgerReportRow) string {
	ref := strings.TrimSpace(row.Reference)
	if ref != "" {
		return ref
	}
	switch row.Kind {
	case "ap-credit":
		return "AP Credit"
	case "ap-debit":
		return "AP Debit"
	default:
		return row.EntryID
	}
}

func arLedgerReference(row models.ARLedgerReportRow) string {
	ref := strings.TrimSpace(row.Reference)
	if ref != "" {
		return ref
	}
	switch row.Kind {
	case "ar-credit", "rebates":
		return "AR Credit"
	case "ar-debit":
		return "AR Debit"
	default:
		return row.EntryID
	}
}

func stockLedgerReference(row models.StockLedgerReportRow) string {
	ref := strings.TrimSpace(row.Reference)
	if ref != "" {
		return ref
	}
	switch row.Kind {
	case "purchases":
		return "Purchases"
	case "sales":
		return "Sales"
	case "stock-in":
		return "Stock In"
	case "stock-out":
		return "Stock Out"
	case "stock-transactions":
		return "Stock Transfer"
	default:
		return strings.TrimSpace(row.Kind)
	}
}

func stockLedgerCompany(kind string) string {
	switch kind {
	case "purchases":
		return "Supplier"
	case "sales":
		return "Customer"
	case "stock-in":
		return "Stock In"
	case "stock-out":
		return "Stock Out"
	case "stock-transactions":
		return "Stock Transfer"
	default:
		return ""
	}
}

func moneyString(cents int64) string {
	sign := ""
	if cents < 0 {
		sign = "-"
		cents = -cents
	}
	return sign + commaInt(cents/100) + "." + fmt2(cents%100)
}

func percentString(numerator, denominator int64) string {
	if denominator == 0 {
		return "0.00"
	}
	sign := ""
	if numerator < 0 {
		sign = "-"
		numerator = -numerator
	}
	if denominator < 0 {
		denominator = -denominator
	}
	scaled := (numerator*10000 + denominator/2) / denominator
	return sign + commaInt(scaled/100) + "." + fmt2(scaled%100)
}

func qtyString(value int64) string {
	return commaInt(value)
}

func qtyDecimalString(value int64) string {
	if value < 0 {
		return "-" + commaInt(-value) + ".00"
	}
	return commaInt(value) + ".00"
}

func commaInt(value int64) string {
	raw := strconv.FormatInt(value, 10)
	if len(raw) <= 3 {
		return raw
	}
	var parts []string
	for len(raw) > 3 {
		parts = append([]string{raw[len(raw)-3:]}, parts...)
		raw = raw[:len(raw)-3]
	}
	parts = append([]string{raw}, parts...)
	return strings.Join(parts, ",")
}

func fmt2(value int64) string {
	if value < 10 {
		return "0" + strconv.FormatInt(value, 10)
	}
	return strconv.FormatInt(value, 10)
}
