package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"cims-go/internal/auth"
	"cims-go/internal/models"
	"cims-go/internal/repositories"
)

type fakeStore struct {
	user             models.User
	dr               repositories.DRSelection
	documentValues   models.Record
	documentLines    map[string][]models.Record
	lastMasterSearch string
	lastMasterYear   int
	lastMasterLimit  int
	lastMasterOffset int
	masterRecords    []models.Record
	lastDocKind      string
	lastDocSearch    string
	lastDocYear      int
	deletedDocKind   string
	deletedDocID     int64
	saveMasterCalls  int
	userByIDCalls    int
}

func (s *fakeStore) EnsureAdmin(context.Context, string, string) error { return nil }

func (s *fakeStore) GetUserByUsername(context.Context, string) (models.User, error) {
	return s.user, nil
}

func (s *fakeStore) GetUserByID(context.Context, int64) (models.User, error) {
	s.userByIDCalls++
	return s.user, nil
}

func (s *fakeStore) ListMaster(_ context.Context, _ models.FormDefinition, search string, year int, limit int, offset int) ([]models.Record, error) {
	s.lastMasterSearch = search
	s.lastMasterYear = year
	s.lastMasterLimit = limit
	s.lastMasterOffset = offset
	if s.masterRecords != nil {
		end := offset + limit
		if end > len(s.masterRecords) {
			end = len(s.masterRecords)
		}
		if offset > len(s.masterRecords) {
			offset = len(s.masterRecords)
		}
		return s.masterRecords[offset:end], nil
	}
	return []models.Record{{"id": "1", "code": "BR-01", "name": "Main", "encoder": "Admin"}}, nil
}

func (s *fakeStore) GetMaster(context.Context, models.FormDefinition, int64) (models.Record, error) {
	return models.Record{
		"id":             "1",
		"code":           "ST-01",
		"name":           "Test Stock",
		"category_group": "Feeds",
		"unit":           "bag",
		"min_inventory":  "10",
	}, nil
}

func (s *fakeStore) SaveMaster(context.Context, models.FormDefinition, int64, map[string]string, models.User) (int64, error) {
	s.saveMasterCalls++
	return 1, nil
}

func (s *fakeStore) DeleteMaster(context.Context, models.FormDefinition, int64, models.User) error {
	return nil
}

func (s *fakeStore) ListDocuments(_ context.Context, kind string, search string, year int) ([]models.DocumentListItem, error) {
	s.lastDocKind = kind
	s.lastDocSearch = search
	s.lastDocYear = year
	return []models.DocumentListItem{{
		ID:        1,
		EntryID:   "ENT-1",
		EntryDate: time.Date(2026, time.January, 5, 0, 0, 0, 0, time.UTC),
		Party:     "Supplier A",
		Reference: "REF-1",
		DRRef:     "DR-1",
		Status:    "Open",
		Branch:    "Main",
		Net:       "123.45",
		Encoder:   "Admin",
	}}, nil
}

func (s *fakeStore) GetDocument(context.Context, models.FormDefinition, int64) (models.Record, map[string][]models.Record, error) {
	if s.documentValues != nil {
		return s.documentValues, s.documentLines, nil
	}
	return models.Record{"id": "ENT-197", "record_id": "197", "entry_date": "2026-04-21"}, nil, nil
}

func (s *fakeStore) LoadDRSelection(context.Context, int64) (repositories.DRSelection, error) {
	return s.dr, nil
}

func (s *fakeStore) SaveDocument(context.Context, models.FormDefinition, int64, repositories.DocumentInput) (int64, error) {
	return 1, nil
}

func (s *fakeStore) DeleteDocument(_ context.Context, form models.FormDefinition, id int64, _ models.User) error {
	s.deletedDocKind = form.Kind
	s.deletedDocID = id
	return nil
}

func (s *fakeStore) PurchaseReportRows(context.Context, time.Time, time.Time) ([]models.PurchaseReportRow, error) {
	return []models.PurchaseReportRow{{
		Supplier:   "Supplier A",
		EntryID:    "ENT-1",
		EntryDate:  "05/30/2026",
		ORCINumber: "SI-1",
		GrossCents: 12345,
		NetCents:   12000,
	}}, nil
}

func (s *fakeStore) PurchaseByDRNumberReportRows(context.Context, time.Time, time.Time) ([]models.PurchaseByDRNumberReportRow, error) {
	return []models.PurchaseByDRNumberReportRow{
		{Reference: "CI #47138", PurchaseDate: "01/05/2026", Supplier: "SOUTH SEA DESIGNS,INC.", StockCode: "HPSPP 25I", StockName: "(HPS PREM. 25KLS.) PRE-STARTER PELLET PREMIUM", Quantity: 20, UnitCostCents: 140200, AmountCents: 2804000},
		{Reference: "CI #47138", PurchaseDate: "01/05/2026", Supplier: "SOUTH SEA DESIGNS,INC.", StockCode: "HGPP", StockName: "HOG GROWER PELLET PREM.", Quantity: 200, UnitCostCents: 186600, AmountCents: 37320000},
		{Reference: "DR 044099", PurchaseDate: "01/07/2026", Supplier: "Supplier B", StockCode: "PPC", StockName: "PIG PROTEIN CONCENTRATE", Quantity: 100, UnitCostCents: 231000, AmountCents: 23100000},
	}, nil
}

func (s *fakeStore) PurchaseByStockCodeReportRows(context.Context, time.Time, time.Time) ([]models.PurchaseByStockCodeReportRow, error) {
	return []models.PurchaseByStockCodeReportRow{
		{Reference: "TA #138120", PurchaseDate: "01/13/2026", Supplier: "NESTY", StockCode: "NESTY 7KNDS", StockName: "NESTY 7 KINDS", Quantity: 139, UnitCostCents: 73266, AmountCents: 10183974},
		{Reference: "TA #138145", PurchaseDate: "01/19/2026", Supplier: "NESTY", StockCode: "NESTY 7KNDS", StockName: "NESTY 7 KINDS", Quantity: 74, UnitCostCents: 67784, AmountCents: 5016016},
		{Reference: "CI #47138", PurchaseDate: "01/05/2026", Supplier: "SOUTH SEA DESIGNS,INC.", StockCode: "HGPP", StockName: "HOG GROWER PELLET PREM.", Quantity: 200, UnitCostCents: 186600, AmountCents: 37320000},
	}, nil
}

func (s *fakeStore) PurchaseBySupplierReportRows(context.Context, time.Time, time.Time) ([]models.PurchaseBySupplierReportRow, error) {
	return []models.PurchaseBySupplierReportRow{
		{Reference: "SI #3055", PurchaseDate: "01/16/2026", Supplier: "DG AGRIVET", StockCode: "PIGRO VTL", StockName: "PIGROLAC HOG GROWER PELLET VITAL", Quantity: 150, UnitCostCents: 168300, AmountCents: 25245000},
		{Reference: "SI #3273", PurchaseDate: "01/24/2026", Supplier: "DG AGRIVET", StockCode: "PIGRO VTL", StockName: "PIGROLAC HOG GROWER PELLET VITAL", Quantity: 50, UnitCostCents: 168300, AmountCents: 8415000},
		{Reference: "TA #138120", PurchaseDate: "01/13/2026", Supplier: "NESTY", StockCode: "NESTY 7KNDS", StockName: "NESTY 7 KINDS", Quantity: 139, UnitCostCents: 73266, AmountCents: 10183974},
	}, nil
}

func (s *fakeStore) SalesReportRows(context.Context, time.Time, time.Time) ([]models.SalesReportRow, error) {
	return []models.SalesReportRow{{
		Customer:   "Customer A",
		EntryID:    "ENT-2",
		EntryDate:  "05/30/2026",
		ORCINumber: "CI-1",
		GrossCents: 22345,
		NetCents:   22000,
	}}, nil
}

func (s *fakeStore) SalesByORCIDRNumberReportRows(context.Context, time.Time, time.Time) ([]models.SalesByORCIDRNumberReportRow, error) {
	return []models.SalesByORCIDRNumberReportRow{
		{Reference: "CI 011497", SalesDate: "01/07/2026", Customer: "CASH/MATARANAS", StockCode: "INT 1000", StockName: "INTEGRA 1000", Quantity: 1, PriceCents: 200200, AmountCents: 200200},
		{Reference: "CI 011497", SalesDate: "01/07/2026", Customer: "CASH/MATARANAS", StockCode: "INT 2000", StockName: "INTEGRA 2000", Quantity: 1, PriceCents: 191100, AmountCents: 191100},
		{Reference: "CHG 005245", SalesDate: "01/08/2026", Customer: "HYZIE SARI SARI STORE", StockCode: "BOW WOW", StockName: "BOW WOW ADULT", Quantity: 2, PriceCents: 120000, AmountCents: 240000},
	}, nil
}

func (s *fakeStore) SalesMarkupByTransactionReportRows(context.Context, time.Time, time.Time) ([]models.SalesMarkupByTransactionReportRow, error) {
	return []models.SalesMarkupByTransactionReportRow{
		{SalesDate: "01/02/2026", EntryID: "92196", SalesType: "Cash", ReceiptNo: "CI 011428", ItemGroup: "HOGS", MarkupCents: 9999, CapitalCents: 198000},
		{SalesDate: "01/02/2026", EntryID: "92197", SalesType: "Cash", ReceiptNo: "CI 011429", ItemGroup: "GRAINS", MarkupCents: 3000, CapitalCents: 82418},
		{SalesDate: "01/02/2026", EntryID: "92207", SalesType: "Charge", ReceiptNo: "CHG 005150", ItemGroup: "POULTRY SOLUTION", MarkupCents: 2610000, CapitalCents: 37716763},
	}, nil
}

func (s *fakeStore) SalesByCustomerReportRows(context.Context, time.Time, time.Time) ([]models.SalesByCustomerReportRow, error) {
	return []models.SalesByCustomerReportRow{
		{Category: "CHICKEN LINES/NESTY", Customer: "4A MINI MART", Reference: "CI 011429", SalesDate: "01/02/2026", StockCode: "NESTY ST", StockName: "NESTY STAG MAINTENANCE", Quantity: 1, PriceCents: 82500, AmountCents: 82500},
		{Category: "CHICKEN LINES/NESTY", Customer: "AYA/SP GMD STORE", Reference: "CI 011477", SalesDate: "01/06/2026", StockCode: "NESTY 7K", StockName: "NESTY 7 KINDS", Quantity: 3, PriceCents: 89500, AmountCents: 268500},
		{Category: "DOG FOOD", Customer: "Cash Customer", Reference: "CHG 005245", SalesDate: "01/08/2026", StockCode: "BOW WOW", StockName: "BOW WOW ADULT", Quantity: 2, PriceCents: 120000, AmountCents: 240000},
	}, nil
}

func (s *fakeStore) SalesByStockNameReportRows(context.Context, time.Time, time.Time) ([]models.SalesByStockNameReportRow, error) {
	return []models.SalesByStockNameReportRow{
		{Category: "CHICKEN LINES/B-MEG", Customer: "CASH/MATARANAS", Reference: "CI 011497", SalesDate: "01/07/2026", StockCode: "INT 1000", StockName: "INTEGRA 1000", Quantity: 1, PriceCents: 200200, AmountCents: 200200},
		{Category: "CHICKEN LINES/B-MEG", Customer: "CATHALEYA AGRI-POULTRY SUPPLY", Reference: "CI 011499", SalesDate: "01/07/2026", StockCode: "INT 1000", StockName: "INTEGRA 1000", Quantity: 1, PriceCents: 199000, AmountCents: 199000},
		{Category: "CHICKEN LINES/B-MEG", Customer: "4A MINI MART", Reference: "CI 011429", SalesDate: "01/02/2026", StockCode: "INT 2000", StockName: "INTEGRA 2000", Quantity: 1, PriceCents: 191100, AmountCents: 191100},
		{Category: "DOG FOOD", Customer: "HYZIE SARI SARI STORE", Reference: "CHG 005245", SalesDate: "01/08/2026", StockCode: "BOW WOW", StockName: "BOW WOW ADULT", Quantity: 2, PriceCents: 120000, AmountCents: 240000},
	}, nil
}

func (s *fakeStore) APLedgerReportRows(context.Context, time.Time, time.Time) ([]models.APLedgerReportRow, error) {
	return []models.APLedgerReportRow{{
		SupplierID:     "1",
		SupplierCode:   "SUP-1",
		SupplierName:   "Supplier A",
		Representative: "Rep A",
		EntryID:        "ENT-3",
		EntryDate:      "05/30/2026",
		Reference:      "SI-2",
		Kind:           "purchases",
		DeltaCents:     15000,
	}}, nil
}

func (s *fakeStore) ARLedgerReportRows(context.Context, time.Time, time.Time) ([]models.ARLedgerReportRow, error) {
	return []models.ARLedgerReportRow{{
		CustomerID:   "1",
		CustomerCode: "CUS-1",
		CustomerName: "Customer A",
		CreditTerm:   "30 days",
		CreditLimit:  500000,
		EntryID:      "ENT-4",
		EntryDate:    "05/30/2026",
		Reference:    "CI-2",
		Kind:         "sales",
		DeltaCents:   15000,
	}}, nil
}

func (s *fakeStore) IncomingCheckReportRows(context.Context, time.Time) ([]models.IncomingCheckReportRow, error) {
	return []models.IncomingCheckReportRow{{
		Payee:       "Customer A",
		Reference:   "AR Credit",
		CheckDate:   "2026-05-30",
		Number:      "CHK-1",
		BankName:    "Bank A",
		AmountCents: 12345,
	}}, nil
}

func (s *fakeStore) OutgoingCheckReportRows(context.Context, time.Time) ([]models.OutgoingCheckReportRow, error) {
	return []models.OutgoingCheckReportRow{{
		Payee:       "Supplier A",
		Reference:   "AP Credit",
		CheckDate:   "05/30/2026",
		Number:      "CHK-2",
		BankName:    "Bank B",
		AmountCents: 22345,
	}, {
		Payee:       "Supplier B",
		Reference:   "AP Credit",
		CheckDate:   "06/01/2026",
		Number:      "CHK-3",
		BankName:    "Bank C",
		AmountCents: 30000,
	}}, nil
}

func (s *fakeStore) ExpenseReportRows(context.Context, time.Time, time.Time) ([]models.ExpenseReportRow, error) {
	return []models.ExpenseReportRow{{
		CategoryID:   "1",
		CategoryCode: "011",
		CategoryName: "ADVANCES TO EMPLOYEES",
		EntryDate:    "05/30/2026",
		CashCents:    10000,
		CheckCents:   2345,
		TotalCents:   12345,
	}}, nil
}

func (s *fakeStore) IncomeStatementRows(context.Context, time.Time, time.Time) ([]models.IncomeStatementRow, error) {
	return []models.IncomeStatementRow{
		{Section: "cash_sales", Label: "Cash Sales", AmountCents: 120000},
		{Section: "charge_sales", Label: "Charge Sales", AmountCents: 340000},
		{Section: "beginning_inventory", Label: "Stock Inventory, Beginning", AmountCents: 500000},
		{Section: "purchases", Label: "Supplier A", AmountCents: 220000},
		{Section: "ending_inventory", Label: "Stock Inventory, End", AmountCents: 180000},
		{Section: "operating_expenses", Label: "ADVANCES TO EMPLOYEES", AmountCents: 45000},
		{Section: "other_income", Label: "Other Income", AmountCents: 25000},
	}, nil
}

func (s *fakeStore) IncentiveReportRows(context.Context, time.Time, time.Time) ([]models.IncentiveReportRow, error) {
	return []models.IncentiveReportRow{{
		AgriPost: "APS",
		Qty:      10,
		VIP:      2,
		APS:      3,
		Takals:   4,
		Farm:     1,
	}}, nil
}

func (s *fakeStore) DailySalesCollectionReportRows(context.Context, time.Time) ([]models.DailySalesCollectionReportRow, error) {
	return []models.DailySalesCollectionReportRow{
		{Section: "cash_sales", Name: "Cash Customer", Reference: "CSH 0001", AmountCents: 120000},
		{Section: "charge_sales", Name: "Customer A", Reference: "CHG 005965", AmountCents: 9562300},
		{Section: "cash_receipts", Name: "Customer B", Reference: "AR Credit", AmountCents: 50000},
		{Section: "disbursements", Name: "Fuel", Reference: "EXP-1", AmountCents: 25000},
		{Section: "check_deposits", Name: "Customer C", Reference: "CHK-1", AmountCents: 70000},
	}, nil
}

func (s *fakeStore) StockSalesTransferReportRows(context.Context, time.Time, time.Time) ([]models.StockSalesTransferReportRow, error) {
	return []models.StockSalesTransferReportRow{
		{Category: "CHICKEN LINES/B-MEG", StockCode: "INT 2000", StockName: "INTEGRA 2000", SalesQty: 80, TransferQty: 821},
		{Category: "CHICKEN LINES/B-MEG", StockCode: "INT 3000", StockName: "INTEGRA 3000", SalesQty: 84, TransferQty: 717},
	}, nil
}

func (s *fakeStore) StockSalesTransferAmountReportRows(context.Context, time.Time, time.Time) ([]models.StockSalesTransferAmountReportRow, error) {
	return []models.StockSalesTransferAmountReportRow{
		{Category: "AQUA", CashSalesCents: 100000, ChargeSalesCents: 250000, TransferCents: 90000, SalesMarkupCents: 35000, TransferMarkupCents: 9000},
		{Category: "BASIC", CashSalesCents: 0, ChargeSalesCents: 50000, TransferCents: 10000, SalesMarkupCents: 5000, TransferMarkupCents: 1000},
	}, nil
}

func (s *fakeStore) StockTransferSummaryReportRows(context.Context, time.Time, time.Time) ([]models.StockTransferSummaryReportRow, error) {
	return []models.StockTransferSummaryReportRow{
		{Category: "BY PRODUCT", Branch: "HERNANS BANSALAN", Reference: "33,155", TransferDate: "01/05/2026", StockCode: "HYC", StockName: "HAMMERED YELLOW CORN", Quantity: 5, AmountCents: 570000},
		{Category: "BY PRODUCT", Branch: "HERNANS BANSALAN", Reference: "33,173", TransferDate: "01/07/2026", StockCode: "TAHOP Y", StockName: "TAHOP YELLOW LOCAL", Quantity: 5, AmountCents: 595000},
		{Category: "BY PRODUCT", Branch: "HERNANS DIGOS", Reference: "33,161", TransferDate: "01/05/2026", StockCode: "THM WHT", StockName: "TAHOP WHITE BAGGER", Quantity: 20, AmountCents: 2444000},
	}, nil
}

func (s *fakeStore) StockTransferByStockNameReportRows(context.Context, time.Time, time.Time) ([]models.StockTransferByStockNameReportRow, error) {
	return []models.StockTransferByStockNameReportRow{
		{Category: "BY PRODUCT", Branch: "HERNANS BANSALAN", Reference: "33,155", TransferID: "ST 23362 SO_27", TransferDate: "01/05/2026", StockCode: "HYC", StockName: "HAMMERED YELLOW CORN", Quantity: 5, AmountCents: 570000},
		{Category: "BY PRODUCT", Branch: "HERNANS BANSALAN", Reference: "33,240", TransferID: "ST 23448 SO_27", TransferDate: "01/19/2026", StockCode: "HYC", StockName: "HAMMERED YELLOW CORN", Quantity: 5, AmountCents: 555000},
		{Category: "BY PRODUCT", Branch: "HERNANS DIGOS", Reference: "33,161", TransferID: "ST 23368 SO_27", TransferDate: "01/05/2026", StockCode: "HYC", StockName: "HAMMERED YELLOW CORN", Quantity: 10, AmountCents: 1130000},
	}, nil
}

func (s *fakeStore) StockTransferByBranchReportRows(context.Context, time.Time, time.Time) ([]models.StockTransferByBranchReportRow, error) {
	return []models.StockTransferByBranchReportRow{
		{Branch: "HERNANS BANSALAN", Category: "BY PRODUCT", Reference: "33,155", TransferDate: "01/05/2026", StockCode: "HYC", StockName: "HAMMERED YELLOW CORN", Quantity: 5, AmountCents: 570000},
		{Branch: "HERNANS BANSALAN", Category: "CHICKEN LINES/B-MEG", Reference: "33,155", TransferDate: "01/05/2026", StockCode: "INT 2000", StockName: "INTEGRA 2000", Quantity: 10, AmountCents: 1881000},
		{Branch: "HERNANS DIGOS", Category: "BY PRODUCT", Reference: "33,161", TransferDate: "01/05/2026", StockCode: "THM WHT", StockName: "TAHOP WHITE BAGGER", Quantity: 20, AmountCents: 2444000},
	}, nil
}

func (s *fakeStore) StockTransferByEntryIDReportRows(context.Context, time.Time, time.Time) ([]models.StockTransferByEntryIDReportRow, error) {
	return []models.StockTransferByEntryIDReportRow{
		{EntryID: "33,155", Reference: "33,155", TransferID: "ST 23362 SO_27", Remarks: "ST 23362 SO_27", TransferDate: "01/05/2026", Branch: "HERNANS BANSALAN", StockCode: "HYC", StockName: "HAMMERED YELLOW CORN", Quantity: 5, AmountCents: 570000, NetCents: 570000},
	}, nil
}

func (s *fakeStore) StockTransferMarkupByTransactionReportRows(context.Context, time.Time, time.Time) ([]models.StockTransferMarkupByTransactionReportRow, error) {
	return []models.StockTransferMarkupByTransactionReportRow{
		{TransferDate: "01/02/2026", EntryID: "33137", TransferTo: "HS STA.MARIA", ReceiptNo: "ST 23343", ItemGroup: "HOGS", MarkupCents: 442420, CapitalCents: 6744200},
		{TransferDate: "01/02/2026", EntryID: "33137", TransferTo: "HS STA.MARIA", ReceiptNo: "ST 23343", ItemGroup: "SALTO", MarkupCents: -48790, CapitalCents: 171900},
		{TransferDate: "01/02/2026", EntryID: "33138", TransferTo: "HERNANS DIGOS", ReceiptNo: "ST 23344", ItemGroup: "GALLIMAX", MarkupCents: 187485, CapitalCents: 2670536},
	}, nil
}

func (s *fakeStore) StockTransferSummaryByItemReportRows(context.Context, time.Time, time.Time) ([]models.StockTransferSummaryByItemReportRow, error) {
	return []models.StockTransferSummaryByItemReportRow{
		{Category: "CORN", StockCode: "HYC", StockName: "HAMMERED YELLOW CORN  (srcm)", Quantity: 247, AmountCents: 27358500},
		{Category: "CORN", StockCode: "TAHOP Y LCL", StockName: "TAHOP YELLOW LOCAL", Quantity: 127, AmountCents: 14597000},
		{Category: "CORN", StockCode: "THM WHT BG", StockName: "TAHOP WHITE BAGGER", Quantity: 1530, AmountCents: 187000000},
	}, nil
}

func (s *fakeStore) StockAgingReportRows(context.Context, time.Time) ([]models.StockAgingReportRow, error) {
	return []models.StockAgingReportRow{
		{Category: "PILMICO HOGS", StockCode: "CLASSIC FI", StockName: "CLASSIC FINEX, 50kg.", Bucket0: 155, Bucket1: 134},
		{Category: "PILMICO HOGS", StockCode: "776", StockName: "CLASSIC GROWEX", Bucket0: 10, Bucket1: 343},
		{Category: "RICE", StockCode: "RICE-01", StockName: "Premium Rice", Bucket2: 8},
	}, nil
}

func (s *fakeStore) StockReorderPointReportRows(context.Context, time.Time) ([]models.StockReorderPointReportRow, error) {
	return []models.StockReorderPointReportRow{
		{Category: "DOG FOOD", StockCode: "BOW", StockName: "BOW WOW", SOH: 0, MinInventory: 10, Deficit: 10},
		{Category: "DOG FOOD", StockCode: "PEDIGREE ADUL", StockName: "PEDIGREE ADULT", SOH: 4, MinInventory: 10, Deficit: 6},
		{Category: "PILMICO HOGS", StockCode: "CLASSIC FI", StockName: "CLASSIC FINEX, 50kg.", SOH: 3, MinInventory: 8, Deficit: 5},
	}, nil
}

func (s *fakeStore) StockSummaryReportRows(context.Context, time.Time) ([]models.StockSummaryReportRow, error) {
	return []models.StockSummaryReportRow{
		{Category: "CHICKEN LINES/NESTY", StockCode: "NESTY 7KNDS", StockName: "NESTY 7 KINDS", SOH: 230, UnitCostCents: 70234, AmountCents: 16153926},
		{Category: "CHICKEN LINES/NESTY", StockCode: "NESTY COND.", StockName: "NESTY CONDITIONER", SOH: 140, UnitCostCents: 78000, AmountCents: 10920000},
		{Category: "DOG FOOD", StockCode: "BOW", StockName: "BOW WOW", SOH: 3, UnitCostCents: 100000, AmountCents: 300000},
	}, nil
}

func (s *fakeStore) StockLedgerReportRows(context.Context, time.Time) ([]models.StockLedgerReportRow, error) {
	return []models.StockLedgerReportRow{
		{StockID: "1", Category: "PILMICO HOGS", StockCode: "CLASSIC FI", StockName: "CLASSIC FINEX, 50kg.", EntryDate: "12/31/2025", Reference: "PO-OLD", Company: "Supplier A", Kind: "purchases", QtyDelta: 25},
		{StockID: "1", Category: "PILMICO HOGS", StockCode: "CLASSIC FI", StockName: "CLASSIC FINEX, 50kg.", EntryDate: "03/14/2026", Reference: "PO-1", Company: "Supplier A", Kind: "purchases", QtyDelta: 155},
		{StockID: "2", Category: "PILMICO HOGS", StockCode: "776", StockName: "CLASSIC GROWEX"},
		{StockID: "3", Category: "EMPTY CATEGORY", StockCode: "EMPTY", StockName: "EMPTY STOCK"},
		{StockID: "4", Category: "ORDER TEST", StockCode: "ORDER", StockName: "ORDER STOCK", EntryDate: "03/14/2026", SortKey: "20260314090000000000-00000000000000000010-00000000000000000020", Reference: "ZZ-PURCHASE", Company: "Supplier A", Kind: "purchases", QtyDelta: 10},
		{StockID: "4", Category: "ORDER TEST", StockCode: "ORDER", StockName: "ORDER STOCK", EntryDate: "03/14/2026", SortKey: "20260314100000000000-00000000000000000011-00000000000000000021", Reference: "AA-SALE", Company: "Customer A", Kind: "sales", QtyDelta: -5},
	}, nil
}

func (s *fakeStore) Options(_ context.Context, source string) ([]models.Option, error) {
	switch source {
	case "stock_categories":
		return []models.Option{{Value: "Feeds", Label: "Feeds"}}, nil
	case "stock_category_groups":
		return []models.Option{{Value: "Legacy Group", Label: "Legacy Group"}}, nil
	default:
		return []models.Option{{Value: "1", Label: "Main"}}, nil
	}
}

func TestDashboardRequiresLogin(t *testing.T) {
	store := &fakeStore{user: models.User{ID: 1, Username: "admin", DisplayName: "Admin", Role: models.RoleAdmin}}
	app, err := NewApp(store, auth.NewManager(store, "12345678901234567890123456789012", "1234567890123456"))
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	if got := rec.Header().Get("Location"); got != "/login" {
		t.Fatalf("Location = %q, want /login", got)
	}
}

func TestStaticAssetsBypassUserLookup(t *testing.T) {
	hash, err := auth.HashPassword("password")
	if err != nil {
		t.Fatal(err)
	}
	store := &fakeStore{user: models.User{ID: 1, Username: "admin", PasswordHash: hash, DisplayName: "Admin", Role: models.RoleAdmin}}
	manager := auth.NewManager(store, "12345678901234567890123456789012", "1234567890123456")
	app, err := NewApp(store, manager)
	if err != nil {
		t.Fatal(err)
	}

	loginRec := httptest.NewRecorder()
	if _, err := manager.Login(context.Background(), loginRec, "admin", "password"); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/static/app.css?v=test", nil)
	for _, cookie := range loginRec.Result().Cookies() {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if store.userByIDCalls != 0 {
		t.Fatalf("GetUserByID calls = %d, want 0", store.userByIDCalls)
	}
	if got := rec.Header().Get("Cache-Control"); !strings.Contains(got, "max-age=31536000") {
		t.Fatalf("Cache-Control = %q, want versioned static cache header", got)
	}
}

func TestDynamicHTMLIsCompressed(t *testing.T) {
	store := &fakeStore{user: models.User{ID: 1, Username: "admin", DisplayName: "Admin", Role: models.RoleAdmin}}
	app, err := NewApp(store, auth.NewManager(store, "12345678901234567890123456789012", "1234567890123456"))
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("Content-Type"); !strings.Contains(got, "text/html") {
		t.Fatalf("Content-Type = %q, want text/html", got)
	}
	if got := rec.Header().Get("Content-Encoding"); got != "gzip" {
		t.Fatalf("Content-Encoding = %q, want gzip", got)
	}
	if got := rec.Header().Get("Vary"); got != "Accept-Encoding" {
		t.Fatalf("Vary = %q, want Accept-Encoding", got)
	}
}

func TestMasterListRendersForLoggedInUser(t *testing.T) {
	store := &fakeStore{user: models.User{ID: 1, Username: "admin", DisplayName: "Admin", Role: models.RoleAdmin}}
	manager := auth.NewManager(store, "12345678901234567890123456789012", "1234567890123456")
	app, err := NewApp(store, manager)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/masters/branches/", nil)
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(rec.Body.String(), "Branches") {
		t.Fatalf("body does not contain Branches")
	}
	if !strings.Contains(rec.Body.String(), "BR-01") {
		t.Fatalf("body does not contain fake branch row")
	}
	if !strings.Contains(rec.Body.String(), `class="nav-logout-button"`) || !strings.Contains(rec.Body.String(), `action="/logout"`) {
		t.Fatalf("body does not contain topbar logout button")
	}
	if !strings.Contains(rec.Body.String(), `menu-dismissed`) || !strings.Contains(rec.Body.String(), `.dropdown-menu a[href]`) {
		t.Fatalf("body does not contain dropdown dismiss behavior")
	}
}

func TestCustomerCreateRequiresClientCode(t *testing.T) {
	store := &fakeStore{user: models.User{ID: 1, Username: "admin", DisplayName: "Admin", Role: models.RoleAdmin}}
	manager := auth.NewManager(store, "12345678901234567890123456789012", "1234567890123456")
	app, err := NewApp(store, manager)
	if err != nil {
		t.Fatal(err)
	}

	body := strings.NewReader("code=&company=Acme&lastname=Doe&firstname=Jane")
	req := httptest.NewRequest(http.MethodPost, "/masters/customers", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if store.saveMasterCalls != 0 {
		t.Fatalf("SaveMaster called %d times, want 0", store.saveMasterCalls)
	}
	if !strings.Contains(rec.Body.String(), "Client Code is required") {
		t.Fatalf("body missing client-code validation error")
	}
}

func TestSalesFormLoadsSelectedDRRows(t *testing.T) {
	store := &fakeStore{
		user: models.User{ID: 1, Username: "admin", DisplayName: "Admin", Role: models.RoleAdmin},
		dr: repositories.DRSelection{
			Values: models.Record{"dr_document_id": "7", "party_id": "3"},
			Rows: []models.Record{{
				"dr_line_id":  "11",
				"stock_id":    "5",
				"stock_label": "ST-01 - Test Stock",
				"qty":         "4",
			}},
		},
	}
	manager := auth.NewManager(store, "12345678901234567890123456789012", "1234567890123456")
	app, err := NewApp(store, manager)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/transactions/sales/new?dr_document_id=7", nil)
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `name="dr_document_id"`) {
		t.Fatalf("body missing dr_document_id selector")
	}
	if !strings.Contains(body, `ST-01 - Test Stock`) {
		t.Fatalf("body missing selected DR stock row")
	}
	if !strings.Contains(body, `name="line_details_qty" value="4" readonly`) {
		t.Fatalf("body missing readonly DR qty row")
	}
	if !strings.Contains(body, `data-sales-edit-stock-out`) {
		t.Fatalf("body missing sales stock out edit trigger")
	}
	if !strings.Contains(body, `data-sales-stock-out-editor-frame`) {
		t.Fatalf("body missing embedded stock out editor frame")
	}
}

func TestSalesEditIncludesDeleteButton(t *testing.T) {
	store := &fakeStore{user: models.User{ID: 1, Username: "admin", DisplayName: "Admin", Role: models.RoleAdmin}}
	manager := auth.NewManager(store, "12345678901234567890123456789012", "1234567890123456")
	app, err := NewApp(store, manager)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/transactions/sales/197/edit", nil)
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `aria-label="Delete current record" form="master-delete-form"`) {
		t.Fatalf("body missing sales delete button")
	}
	if !strings.Contains(body, `action="/transactions/sales/197/delete"`) {
		t.Fatalf("body missing sales delete form action")
	}
}

func TestSalesEditSelectsCurrentSOAndRows(t *testing.T) {
	store := &fakeStore{
		user: models.User{ID: 1, Username: "admin", DisplayName: "Admin", Role: models.RoleAdmin},
		documentValues: models.Record{
			"id":                "ENT-197",
			"record_id":         "197",
			"entry_date":        "2026-04-21",
			"dr_document_id":    "7",
			"dr_document_label": "SO-0007 - Customer A",
			"party_id":          "3",
		},
		documentLines: map[string][]models.Record{
			"details": {{
				"dr_line_id":  "11",
				"stock_id":    "5",
				"stock_label": "ST-01 - Test Stock",
				"qty":         "4",
				"unit_cost":   "12.50",
			}},
		},
	}
	manager := auth.NewManager(store, "12345678901234567890123456789012", "1234567890123456")
	app, err := NewApp(store, manager)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/transactions/sales/197/edit", nil)
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `<option value="7" selected>SO-0007 - Customer A</option>`) {
		t.Fatalf("body missing selected SO option")
	}
	if !strings.Contains(body, `ST-01 - Test Stock`) || !strings.Contains(body, `name="line_details_qty" value="4" readonly`) {
		t.Fatalf("body missing saved sales detail row")
	}
}

func TestSalesDeleteRedirectsToList(t *testing.T) {
	store := &fakeStore{user: models.User{ID: 1, Username: "admin", DisplayName: "Admin", Role: models.RoleAdmin}}
	manager := auth.NewManager(store, "12345678901234567890123456789012", "1234567890123456")
	app, err := NewApp(store, manager)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/transactions/sales/197/delete", nil)
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	if got := rec.Header().Get("Location"); got != "/transactions/sales/" {
		t.Fatalf("Location = %q, want sales list", got)
	}
	if store.deletedDocKind != "sales" || store.deletedDocID != 197 {
		t.Fatalf("deleted doc = %s/%d, want sales/197", store.deletedDocKind, store.deletedDocID)
	}
}

func TestPurchaseFormIncludesStockPicker(t *testing.T) {
	store := &fakeStore{user: models.User{ID: 1, Username: "admin", DisplayName: "Admin", Role: models.RoleAdmin}}
	manager := auth.NewManager(store, "12345678901234567890123456789012", "1234567890123456")
	app, err := NewApp(store, manager)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/transactions/purchases/new", nil)
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `data-purchase-party-select`) {
		t.Fatalf("body missing purchase supplier select")
	}
	if !strings.Contains(body, `data-purchase-supplier-browse`) || !strings.Contains(body, `data-purchase-supplier-picker`) {
		t.Fatalf("body missing purchase supplier browse modal")
	}
	if !strings.Contains(body, `data-purchase-stock-trigger`) {
		t.Fatalf("body missing purchase stock trigger input")
	}
	if !strings.Contains(body, `data-purchase-stock-picker`) || !strings.Contains(body, `data-purchase-stock-picker-results`) {
		t.Fatalf("body missing purchase stock picker modal")
	}
	if strings.Contains(body, `purchase-supplier-options`) {
		t.Fatalf("body still contains purchase supplier datalist")
	}
	if strings.Contains(body, `purchase-stock-code-options`) {
		t.Fatalf("body still contains purchase stock datalist")
	}
}

func TestStockFormUsesCategorySelect(t *testing.T) {
	store := &fakeStore{user: models.User{ID: 1, Username: "admin", DisplayName: "Admin", Role: models.RoleAdmin}}
	manager := auth.NewManager(store, "12345678901234567890123456789012", "1234567890123456")
	app, err := NewApp(store, manager)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/masters/stocks/new", nil)
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `<span>Category:</span>`) {
		t.Fatalf("body missing category label")
	}
	if !strings.Contains(body, `name="category_group"`) || !strings.Contains(body, `<select name="category_group">`) {
		t.Fatalf("body missing category select")
	}
	if !strings.Contains(body, `Select category`) {
		t.Fatalf("body missing category placeholder option")
	}
	if !strings.Contains(body, `value="Feeds"`) || !strings.Contains(body, `>Feeds</option>`) {
		t.Fatalf("body missing stock category option")
	}
	if !strings.Contains(body, `legacy-record-row legacy-record-row-split`) {
		t.Fatalf("body missing split row for unit and latest cost")
	}
	if !strings.Contains(body, `legacy-record-field-split`) {
		t.Fatalf("body missing split field class for unit and latest cost")
	}
	if !strings.Contains(body, `<span>SOH:</span>`) {
		t.Fatalf("body missing SOH field label")
	}
	if !strings.Contains(body, `name="latest_cost" type="number" step="0.01" value="" readonly tabindex="-1" aria-readonly="true"`) {
		t.Fatalf("body missing readonly latest cost field")
	}
	if !strings.Contains(body, `value="" readonly tabindex="-1" aria-readonly="true"`) {
		t.Fatalf("body missing readonly stock-derived field")
	}
	if !strings.Contains(body, `aria-readonly="true"`) {
		t.Fatalf("body missing readonly stock-derived field")
	}
	if !strings.Contains(body, `aria-label="First record" onclick="goStockCategoryRecord('1')" disabled`) {
		t.Fatalf("body missing disabled first-record button")
	}
	if !strings.Contains(body, `aria-label="Previous record" onclick="goStockCategoryRecord('1')" disabled`) {
		t.Fatalf("body missing disabled previous-record button")
	}
	if !strings.Contains(body, `aria-label="Next record" onclick="goStockCategoryRecord('1')" disabled`) {
		t.Fatalf("body missing disabled next-record button")
	}
	if !strings.Contains(body, `aria-label="Last record" onclick="goStockCategoryRecord('1')" disabled`) {
		t.Fatalf("body missing disabled last-record button")
	}
	if !strings.Contains(body, `aria-label="Delete current record" form="master-delete-form" disabled`) {
		t.Fatalf("body missing disabled delete button")
	}
	if !strings.Contains(body, `aria-label="Search" data-form-action="search" onclick="searchStockCategoryRecord()" disabled`) {
		t.Fatalf("body missing disabled search button")
	}
	if !strings.Contains(body, `aria-label="Open list" data-form-action="open-list" disabled`) {
		t.Fatalf("body missing disabled open-list button")
	}
	if strings.Contains(body, `Category Group:`) {
		t.Fatalf("body still contains category group label")
	}
	if strings.Contains(body, `list-category_group`) {
		t.Fatalf("body still contains category datalist")
	}
	if strings.Contains(body, `Legacy Group`) {
		t.Fatalf("body still contains stock category group options")
	}
}

func TestEmbeddedStockEditRendersPopupMode(t *testing.T) {
	store := &fakeStore{user: models.User{ID: 1, Username: "admin", DisplayName: "Admin", Role: models.RoleAdmin}}
	manager := auth.NewManager(store, "12345678901234567890123456789012", "1234567890123456")
	app, err := NewApp(store, manager)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/masters/stocks/1/edit?embedded=1", nil)
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `data-embedded="true"`) {
		t.Fatalf("body missing embedded form marker")
	}
	if !strings.Contains(body, `action="/masters/stocks/1?embedded=1"`) {
		t.Fatalf("body missing embedded save action")
	}
	if !strings.Contains(body, `data-embedded-close`) {
		t.Fatalf("body missing embedded close trigger")
	}
	if strings.Contains(body, `class="form-backdrop-table table-wrap"`) {
		t.Fatalf("body should not render backdrop list in embedded mode")
	}
	if strings.Contains(body, `class="navrail"`) {
		t.Fatalf("body should not render topbar in embedded mode")
	}
}

func TestEmbeddedStockSaveRedirectsBackToEdit(t *testing.T) {
	store := &fakeStore{user: models.User{ID: 1, Username: "admin", DisplayName: "Admin", Role: models.RoleAdmin}}
	manager := auth.NewManager(store, "12345678901234567890123456789012", "1234567890123456")
	app, err := NewApp(store, manager)
	if err != nil {
		t.Fatal(err)
	}

	form := strings.NewReader("code=ST-01&name=Updated+Stock&category_group=Feeds&unit=bag&description=test&min_inventory=12")
	req := httptest.NewRequest(http.MethodPost, "/masters/stocks/1?embedded=1", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	if got := rec.Header().Get("Location"); got != "/masters/stocks/1/edit?embedded=1&saved=1" {
		t.Fatalf("Location = %q, want embedded edit redirect", got)
	}
}

func TestEmbeddedTransactionSaveRedirectsBackToEdit(t *testing.T) {
	store := &fakeStore{user: models.User{ID: 1, Username: "admin", DisplayName: "Admin", Role: models.RoleAdmin}}
	manager := auth.NewManager(store, "12345678901234567890123456789012", "1234567890123456")
	app, err := NewApp(store, manager)
	if err != nil {
		t.Fatal(err)
	}

	form := strings.NewReader("party_id=1&entry_date=2026-05-26&dr_number=DR-CUST-026&dr_date=2026-05-26&remarks=test")
	req := httptest.NewRequest(http.MethodPost, "/transactions/dr/155?embedded=1", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	if got := rec.Header().Get("Location"); got != "/transactions/dr/1/edit?embedded=1&saved=1" {
		t.Fatalf("Location = %q, want embedded edit redirect", got)
	}
}

func TestMasterListPassesSearchAndYear(t *testing.T) {
	records := make([]models.Record, masterListPageSize+1)
	for i := range records {
		records[i] = models.Record{"id": "1", "code": "BR-01", "name": "Main", "encoder": "Admin"}
	}
	store := &fakeStore{
		user:          models.User{ID: 1, Username: "admin", DisplayName: "Admin", Role: models.RoleAdmin},
		masterRecords: records,
	}
	manager := auth.NewManager(store, "12345678901234567890123456789012", "1234567890123456")
	app, err := NewApp(store, manager)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/masters/stocks/?year=2025&q=test", nil)
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if store.lastMasterSearch != "test" {
		t.Fatalf("master search = %q, want test", store.lastMasterSearch)
	}
	if store.lastMasterYear != 2025 {
		t.Fatalf("master year = %d, want 2025", store.lastMasterYear)
	}
	if store.lastMasterLimit != masterListPageSize+1 || store.lastMasterOffset != 0 {
		t.Fatalf("master page = limit %d offset %d, want limit %d offset 0", store.lastMasterLimit, store.lastMasterOffset, masterListPageSize+1)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `class="data-row"`) || !strings.Contains(body, `ondblclick="window.location='`) {
		t.Fatalf("body missing master double-click row open behavior")
	}
	if strings.Contains(body, `class="data-row" tabindex="0" onclick="window.location='`) {
		t.Fatalf("body still uses single-click row open behavior")
	}
	if !strings.Contains(body, `hx-trigger="intersect once root:.content-window-body threshold:0.01"`) {
		t.Fatalf("body missing content-window lazy-load trigger")
	}
	if !strings.Contains(body, `class="content-window-body content-window-list-body"`) || !strings.Contains(body, `data-fill-empty-rows`) {
		t.Fatalf("body missing full-height list table markers")
	}
}

func TestExistingMasterEditStartsInViewModeMarkup(t *testing.T) {
	store := &fakeStore{user: models.User{ID: 1, Username: "admin", DisplayName: "Admin", Role: models.RoleAdmin}}
	manager := auth.NewManager(store, "12345678901234567890123456789012", "1234567890123456")
	app, err := NewApp(store, manager)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/masters/branches/1/edit", nil)
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `class="form-shell modal-form" data-existing-record="true"`) {
		t.Fatalf("body missing existing-record marker")
	}
}

func TestTransactionListPassesSearchAndYear(t *testing.T) {
	store := &fakeStore{user: models.User{ID: 1, Username: "admin", DisplayName: "Admin", Role: models.RoleAdmin}}
	manager := auth.NewManager(store, "12345678901234567890123456789012", "1234567890123456")
	app, err := NewApp(store, manager)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/transactions/purchases/?year=2025&q=ent", nil)
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if store.lastDocKind != "purchases" {
		t.Fatalf("doc kind = %q, want purchases", store.lastDocKind)
	}
	if store.lastDocSearch != "ent" {
		t.Fatalf("doc search = %q, want ent", store.lastDocSearch)
	}
	if store.lastDocYear != 2025 {
		t.Fatalf("doc year = %d, want 2025", store.lastDocYear)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `class="data-row"`) || !strings.Contains(body, `ondblclick="window.location='`) {
		t.Fatalf("body missing transaction double-click row open behavior")
	}
	if strings.Contains(body, `class="data-row" tabindex="0" onclick="window.location='`) {
		t.Fatalf("body still uses single-click row open behavior")
	}
}

func TestOutgoingCheckListReportRendersDetailedAndSummary(t *testing.T) {
	store := &fakeStore{user: models.User{ID: 1, Username: "admin", DisplayName: "Admin", Role: models.RoleAdmin}}
	manager := auth.NewManager(store, "12345678901234567890123456789012", "1234567890123456")
	app, err := NewApp(store, manager)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/reports/outgoing-check-list?run=1&report_type=detailed&cutoff_date=2026-05-31", nil)
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("detailed status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "OUTGOING CHECK LIST") || !strings.Contains(body, "Supplier A") || strings.Contains(body, "Supplier B") {
		t.Fatalf("detailed body did not render only cutoff-included outgoing checks")
	}

	req = httptest.NewRequest(http.MethodGet, "/reports/outgoing-check-list?run=1&report_type=summary-postdated&cutoff_date=2026-05-31", nil)
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec = httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("summary status = %d, want %d", rec.Code, http.StatusOK)
	}
	body = rec.Body.String()
	if !strings.Contains(body, "OUTGOING CHECK ACCOUNT SUMMARY (Postdated)") || !strings.Contains(body, "Supplier B") || strings.Contains(body, "Supplier A") {
		t.Fatalf("summary body did not render only postdated outgoing checks")
	}
}

func TestExpensesSummaryReportRenders(t *testing.T) {
	store := &fakeStore{user: models.User{ID: 1, Username: "admin", DisplayName: "Admin", Role: models.RoleAdmin}}
	manager := auth.NewManager(store, "12345678901234567890123456789012", "1234567890123456")
	app, err := NewApp(store, manager)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/reports/expenses-summary?run=1&report_type=summary&coverage=month&month=5&year=2026", nil)
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "EXPENSES SUMMARY") {
		t.Fatalf("body missing expenses summary title")
	}
	if !strings.Contains(body, "ADVANCES TO EMPLOYEES") {
		t.Fatalf("body missing expense category row")
	}
}

func TestReportMonthOptionDefaultsToJanuary(t *testing.T) {
	store := &fakeStore{user: models.User{ID: 1, Username: "admin", DisplayName: "Admin", Role: models.RoleAdmin}}
	manager := auth.NewManager(store, "12345678901234567890123456789012", "1234567890123456")
	app, err := NewApp(store, manager)
	if err != nil {
		t.Fatal(err)
	}
	app.now = func() time.Time {
		return time.Date(2026, time.June, 20, 0, 0, 0, 0, time.UTC)
	}

	req := httptest.NewRequest(http.MethodGet, "/reports/sales-summary", nil)
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `value="1" selected>1 - January</option>`) {
		t.Fatalf("body missing selected January month option")
	}
	if strings.Contains(body, `value="6" selected>6 - June</option>`) {
		t.Fatalf("body selected current month instead of January")
	}
}

func TestIncomeStatementReportRenders(t *testing.T) {
	store := &fakeStore{user: models.User{ID: 1, Username: "admin", DisplayName: "Admin", Role: models.RoleAdmin}}
	manager := auth.NewManager(store, "12345678901234567890123456789012", "1234567890123456")
	app, err := NewApp(store, manager)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/reports/income-statement?run=1&coverage=month&month=5&year=2026", nil)
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Income Statement") {
		t.Fatalf("body missing income statement title")
	}
	if !strings.Contains(body, "NET INCOME") {
		t.Fatalf("body missing net income row")
	}
	if !strings.Contains(body, "ADVANCES TO EMPLOYEES") {
		t.Fatalf("body missing operating expense row")
	}
}

func TestIncentiveReportRenders(t *testing.T) {
	store := &fakeStore{user: models.User{ID: 1, Username: "admin", DisplayName: "Admin", Role: models.RoleAdmin}}
	manager := auth.NewManager(store, "12345678901234567890123456789012", "1234567890123456")
	app, err := NewApp(store, manager)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/reports/incentive?run=1&coverage=month&month=5&year=2026", nil)
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "INCENTIVE REPORT") {
		t.Fatalf("body missing incentive report title")
	}
	if !strings.Contains(body, "Agri Post") || !strings.Contains(body, "TAKALS") || !strings.Contains(body, "FARM") {
		t.Fatalf("body missing incentive report columns")
	}
	if !strings.Contains(body, "APS") || !strings.Contains(body, "10") {
		t.Fatalf("body missing incentive report row")
	}
}

func TestDailySalesCollectionReportRenders(t *testing.T) {
	store := &fakeStore{user: models.User{ID: 1, Username: "admin", DisplayName: "Admin", Role: models.RoleAdmin}}
	manager := auth.NewManager(store, "12345678901234567890123456789012", "1234567890123456")
	app, err := NewApp(store, manager)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/reports/daily-sales-collection?run=1&report_date=2026-03-14", nil)
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "DAILY SALES AND COLLECTION REPORT") {
		t.Fatalf("body missing daily sales report title")
	}
	if !strings.Contains(body, "Report Date: 03/14/2026") || !strings.Contains(body, "CHG 005965") {
		t.Fatalf("body missing selected date or charge sales row")
	}
	if !strings.Contains(body, "TOTAL CASH REMITTANCE") || !strings.Contains(body, "1,450.00") {
		t.Fatalf("body missing cash remittance total")
	}
	if !strings.Contains(body, "TOTAL REMITTANCE") || !strings.Contains(body, "2,150.00") {
		t.Fatalf("body missing remittance total")
	}
}

func TestDailyDueCheckReportRenders(t *testing.T) {
	store := &fakeStore{user: models.User{ID: 1, Username: "admin", DisplayName: "Admin", Role: models.RoleAdmin}}
	manager := auth.NewManager(store, "12345678901234567890123456789012", "1234567890123456")
	app, err := NewApp(store, manager)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/reports/daily-due-check?run=1&cutoff_date=2026-05-29", nil)
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Cut-Off Date: 05/29/2026") || !strings.Contains(body, "Check Date:") {
		t.Fatalf("body missing daily due check date labels")
	}
	if !strings.Contains(body, "Customer A") || !strings.Contains(body, "CHK-1") || !strings.Contains(body, "123.45") {
		t.Fatalf("body missing due check row")
	}
}

func TestIncomingCheckCalendarReportRenders(t *testing.T) {
	store := &fakeStore{user: models.User{ID: 1, Username: "admin", DisplayName: "Admin", Role: models.RoleAdmin}}
	manager := auth.NewManager(store, "12345678901234567890123456789012", "1234567890123456")
	app, err := NewApp(store, manager)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/reports/incoming-check-calendar?run=1&month=5&year=2026", nil)
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Incoming Check Calendar") {
		t.Fatalf("body missing incoming check calendar title")
	}
	if !strings.Contains(body, "Month Total") || !strings.Contains(body, "123.45") {
		t.Fatalf("body missing month total")
	}
	if !strings.Contains(body, "Customer A") || !strings.Contains(body, "CHK-1") || !strings.Contains(body, "Bank A") {
		t.Fatalf("body missing incoming check calendar row")
	}
	if !strings.Contains(body, `data-calendar-date="2026-05-30"`) {
		t.Fatalf("body missing selected check day")
	}
}

func TestStockSalesTransferReportRenders(t *testing.T) {
	store := &fakeStore{user: models.User{ID: 1, Username: "admin", DisplayName: "Admin", Role: models.RoleAdmin}}
	manager := auth.NewManager(store, "12345678901234567890123456789012", "1234567890123456")
	app, err := NewApp(store, manager)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/reports/stock-sales-transfer?run=1&coverage=month&month=5&year=2026", nil)
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "STOCK SALES AND TRANSFER REPORT") {
		t.Fatalf("body missing stock sales transfer report title")
	}
	if !strings.Contains(body, "CHICKEN LINES/B-MEG") || !strings.Contains(body, "INTEGRA 2000") {
		t.Fatalf("body missing category or stock row")
	}
	if !strings.Contains(body, "164") || !strings.Contains(body, "1,538") || !strings.Contains(body, "1,702") {
		t.Fatalf("body missing sales, transfer, or total quantities")
	}

	req = httptest.NewRequest(http.MethodGet, "/reports/stock-sales-transfer-amount?run=1&coverage=month&month=5&year=2026", nil)
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec = httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("amount status = %d, want %d", rec.Code, http.StatusOK)
	}
	body = rec.Body.String()
	if !strings.Contains(body, "STOCK SALES AND TRANSFER AMOUNT SUMMARY") || !strings.Contains(body, "AQUA") {
		t.Fatalf("body missing stock sales transfer amount report")
	}
	if !strings.Contains(body, "3,500.00") || !strings.Contains(body, "10.00") {
		t.Fatalf("body missing amount report markup values")
	}
}

func TestStockTransferSummaryReportRenders(t *testing.T) {
	store := &fakeStore{user: models.User{ID: 1, Username: "admin", DisplayName: "Admin", Role: models.RoleAdmin}}
	manager := auth.NewManager(store, "12345678901234567890123456789012", "1234567890123456")
	app, err := NewApp(store, manager)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/reports/transfers-summary?run=1&coverage=month&month=1&year=2026", nil)
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "STOCK TRANSFER") || !strings.Contains(body, "Sales From: January 01, 2026 To: January 31, 2026") {
		t.Fatalf("body missing stock transfer summary title or coverage")
	}
	if !strings.Contains(body, "HERNANS BANSALAN") || !strings.Contains(body, "TAHOP WHITE BAGGER") {
		t.Fatalf("body missing branch or stock rows")
	}
	if !strings.Contains(body, "Total :") || !strings.Contains(body, "36,090.00") {
		t.Fatalf("body missing summary totals")
	}
}

func TestStockTransferByStockNameReportRenders(t *testing.T) {
	store := &fakeStore{user: models.User{ID: 1, Username: "admin", DisplayName: "Admin", Role: models.RoleAdmin}}
	manager := auth.NewManager(store, "12345678901234567890123456789012", "1234567890123456")
	app, err := NewApp(store, manager)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/reports/transfers-by-stock-name?run=1&coverage=month&month=1&year=2026", nil)
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "STOCK TRANSFER BY STOCK NAME") || !strings.Contains(body, "Sales From: January 01, 2026 To: January 31, 2026") {
		t.Fatalf("body missing stock transfer by stock name title or coverage")
	}
	if !strings.Contains(body, "HAMMERED YELLOW CORN") || !strings.Contains(body, "ST 23362 SO_27") {
		t.Fatalf("body missing stock or transfer rows")
	}
	if !strings.Contains(body, "Branch Total") || !strings.Contains(body, "11,250.00") || !strings.Contains(body, "22,550.00") {
		t.Fatalf("body missing branch or grand totals")
	}
}

func TestStockTransferByBranchReportRenders(t *testing.T) {
	store := &fakeStore{user: models.User{ID: 1, Username: "admin", DisplayName: "Admin", Role: models.RoleAdmin}}
	manager := auth.NewManager(store, "12345678901234567890123456789012", "1234567890123456")
	app, err := NewApp(store, manager)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/reports/transfers-by-branch?run=1&coverage=month&month=1&year=2026", nil)
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "STOCK TRANSFER BY BRANCH") || !strings.Contains(body, "Sales From: January 01, 2026 To: January 31, 2026") {
		t.Fatalf("body missing stock transfer by branch title or coverage")
	}
	if !strings.Contains(body, "HERNANS BANSALAN") || !strings.Contains(body, "Category:") || !strings.Contains(body, "CHICKEN LINES/B-MEG") {
		t.Fatalf("body missing branch or category sections")
	}
	if !strings.Contains(body, "HAMMERED YELLOW CORN") || !strings.Contains(body, "INTEGRA 2000") || !strings.Contains(body, "stock-transfer-by-branch.csv") {
		t.Fatalf("body missing stock rows or export configuration")
	}
	if !strings.Contains(body, "24,510.00") || !strings.Contains(body, "48,950.00") {
		t.Fatalf("body missing branch or grand totals")
	}
}

func TestStockTransferByEntryIDReportRenders(t *testing.T) {
	store := &fakeStore{user: models.User{ID: 1, Username: "admin", DisplayName: "Admin", Role: models.RoleAdmin}}
	manager := auth.NewManager(store, "12345678901234567890123456789012", "1234567890123456")
	app, err := NewApp(store, manager)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/reports/transfers-by-entry-id?run=1&coverage=month&month=1&year=2026", nil)
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "STOCK TRANSFER BY ENTRY ID") || !strings.Contains(body, "Sales From: January 01, 2026 To: January 31, 2026") {
		t.Fatalf("body missing stock transfer by entry id title or coverage")
	}
	if !strings.Contains(body, "Entry ID:") || !strings.Contains(body, "33,155") || !strings.Contains(body, "ST 23362 SO_27") {
		t.Fatalf("body missing entry id or transfer id")
	}
	if !strings.Contains(body, "HAMMERED YELLOW CORN") || !strings.Contains(body, "5.00") || !strings.Contains(body, "5,700.00") {
		t.Fatalf("body missing stock row or totals")
	}
}

func TestStockTransferSummaryByEntryIDReportRenders(t *testing.T) {
	store := &fakeStore{user: models.User{ID: 1, Username: "admin", DisplayName: "Admin", Role: models.RoleAdmin}}
	manager := auth.NewManager(store, "12345678901234567890123456789012", "1234567890123456")
	app, err := NewApp(store, manager)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/reports/transfers-summary-by-entry-id?run=1&coverage=month&month=1&year=2026", nil)
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "STOCK TRANSFER SUMMARY BY ENTRY ID") || !strings.Contains(body, "Sales From: January 01, 2026 To: January 31, 2026") {
		t.Fatalf("body missing stock transfer summary by entry id title or coverage")
	}
	if !strings.Contains(body, "HERNANS BANSALAN") || !strings.Contains(body, "33,155") || !strings.Contains(body, "ST 23362 SO_27") {
		t.Fatalf("body missing branch, entry id, or remarks")
	}
	if !strings.Contains(body, "Branch Total") || !strings.Contains(body, "5.00") || !strings.Contains(body, "5,700.00") {
		t.Fatalf("body missing stock transfer summary by entry id totals")
	}
}

func TestStockTransferSummaryByItemReportRenders(t *testing.T) {
	store := &fakeStore{user: models.User{ID: 1, Username: "admin", DisplayName: "Admin", Role: models.RoleAdmin}}
	manager := auth.NewManager(store, "12345678901234567890123456789012", "1234567890123456")
	app, err := NewApp(store, manager)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/reports/transfers-summary-by-item?run=1&coverage=month&month=1&year=2026", nil)
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "STOCK TRANSFER SUMMARY BY ITEM") || !strings.Contains(body, "Sales From: January 01, 2026 To: January 31, 2026") {
		t.Fatalf("body missing stock transfer summary by item title or coverage")
	}
	if !strings.Contains(body, "CORN") || !strings.Contains(body, "HAMMERED YELLOW CORN") || !strings.Contains(body, "TAHOP WHITE BAGGER") {
		t.Fatalf("body missing category or stock rows")
	}
	if !strings.Contains(body, "Code") || !strings.Contains(body, "StockName") || !strings.Contains(body, "Amount") {
		t.Fatalf("body missing stock transfer summary by item columns")
	}
	if !strings.Contains(body, "1,904.00") || !strings.Contains(body, "2,289,555.00") {
		t.Fatalf("body missing stock transfer summary by item totals")
	}
}

func TestStockTransferMarkupByTransactionReportRenders(t *testing.T) {
	store := &fakeStore{user: models.User{ID: 1, Username: "admin", DisplayName: "Admin", Role: models.RoleAdmin}}
	manager := auth.NewManager(store, "12345678901234567890123456789012", "1234567890123456")
	app, err := NewApp(store, manager)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/reports/transfers-markup-by-transaction?run=1&coverage=month&month=1&year=2026", nil)
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "TRANSFER MARKUP BY TRANSACTION") || !strings.Contains(body, "Sales From: January 01, 2026 To: January 31, 2026") {
		t.Fatalf("body missing transfer markup title or coverage")
	}
	if !strings.Contains(body, "Transfer To") || !strings.Contains(body, "Receipt No.") || !strings.Contains(body, "Markup %") {
		t.Fatalf("body missing transfer markup columns")
	}
	if !strings.Contains(body, "HS STA.MARIA") || !strings.Contains(body, "ST 23343") || !strings.Contains(body, "GALLIMAX") {
		t.Fatalf("body missing transfer markup rows")
	}
	if !strings.Contains(body, "4,424.20") || !strings.Contains(body, "-487.90") || !strings.Contains(body, "report-negative-row") {
		t.Fatalf("body missing transfer markup values or negative row highlight")
	}
	if !strings.Contains(body, "transfer-markup-by-transaction.csv") {
		t.Fatalf("body missing export configuration")
	}
}

func TestStockAgingReportRenders(t *testing.T) {
	store := &fakeStore{user: models.User{ID: 1, Username: "admin", DisplayName: "Admin", Role: models.RoleAdmin}}
	manager := auth.NewManager(store, "12345678901234567890123456789012", "1234567890123456")
	app, err := NewApp(store, manager)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/reports/stock-aging?run=1&cutoff_date=2026-03-14", nil)
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "STOCK AGING") || !strings.Contains(body, "As Of 3/14/2026") {
		t.Fatalf("body missing stock aging title or cutoff")
	}
	if !strings.Contains(body, "PILMICO HOGS") || !strings.Contains(body, "CLASSIC FINEX") {
		t.Fatalf("body missing stock aging category or row")
	}
	if !strings.Contains(body, "02/12/2026 ~ 03/14/2026") || !strings.Contains(body, "01/13/2026 ~ 02/11/2026") {
		t.Fatalf("body missing stock aging bucket labels")
	}
	if !strings.Contains(body, "165") || !strings.Contains(body, "477") || !strings.Contains(body, "8") {
		t.Fatalf("body missing stock aging totals")
	}
}

func TestStockReorderPointReportRenders(t *testing.T) {
	store := &fakeStore{user: models.User{ID: 1, Username: "admin", DisplayName: "Admin", Role: models.RoleAdmin}}
	manager := auth.NewManager(store, "12345678901234567890123456789012", "1234567890123456")
	app, err := NewApp(store, manager)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/reports/stock-reorder-point?run=1&cutoff_date=2026-03-14", nil)
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "STOCK REORDER POINT") || !strings.Contains(body, "Stock Summary As Of: March 14, 2026") {
		t.Fatalf("body missing stock reorder point title or cutoff")
	}
	if !strings.Contains(body, "DOG FOOD") || !strings.Contains(body, "BOW WOW") {
		t.Fatalf("body missing stock reorder point category or row")
	}
	if !strings.Contains(body, "SOH") || !strings.Contains(body, "Min. Inv.") || !strings.Contains(body, "Deficit") {
		t.Fatalf("body missing stock reorder point columns")
	}
	if !strings.Contains(body, "0.00") || !strings.Contains(body, "10.00") || !strings.Contains(body, "6.00") {
		t.Fatalf("body missing stock reorder point quantities")
	}
}

func TestStockSummaryReportRenders(t *testing.T) {
	store := &fakeStore{user: models.User{ID: 1, Username: "admin", DisplayName: "Admin", Role: models.RoleAdmin}}
	manager := auth.NewManager(store, "12345678901234567890123456789012", "1234567890123456")
	app, err := NewApp(store, manager)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/reports/stock-summary?run=1&cutoff_date=2026-03-14", nil)
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "STOCK SUMMARY") || !strings.Contains(body, "Stock Summary As Of: March 14, 2026") {
		t.Fatalf("body missing stock summary title or cutoff")
	}
	if !strings.Contains(body, "CHICKEN LINES/NESTY") || !strings.Contains(body, "NESTY 7 KINDS") {
		t.Fatalf("body missing stock summary category or row")
	}
	if !strings.Contains(body, "SOH") || !strings.Contains(body, "Unit Cost") || !strings.Contains(body, "Amount") {
		t.Fatalf("body missing stock summary columns")
	}
	if !strings.Contains(body, "230.00") || !strings.Contains(body, "702.34") || !strings.Contains(body, "161,539.26") {
		t.Fatalf("body missing stock summary quantities or amounts")
	}
}

func TestPurchasesByDRNumberReportRenders(t *testing.T) {
	store := &fakeStore{user: models.User{ID: 1, Username: "admin", DisplayName: "Admin", Role: models.RoleAdmin}}
	manager := auth.NewManager(store, "12345678901234567890123456789012", "1234567890123456")
	app, err := NewApp(store, manager)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/reports/purchases-by-dr-number?run=1&coverage=month&month=1&year=2026", nil)
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "STOCK PURCHASES BY REFERENCE NUMBER") || !strings.Contains(body, "Purchases From: January 01, 2026 To: January 31, 2026") {
		t.Fatalf("body missing purchases by DR title or coverage")
	}
	if !strings.Contains(body, "CI #47138") || !strings.Contains(body, "SOUTH SEA DESIGNS,INC.") || !strings.Contains(body, "HOG GROWER PELLET PREM.") {
		t.Fatalf("body missing reference group or rows")
	}
	if !strings.Contains(body, "Quantity") || !strings.Contains(body, "Cost") || !strings.Contains(body, "Amount") {
		t.Fatalf("body missing purchases by DR columns")
	}
	if !strings.Contains(body, "220.00") || !strings.Contains(body, "1,866.00") || !strings.Contains(body, "401,240.00") {
		t.Fatalf("body missing purchases by DR totals or amounts")
	}
}

func TestSalesByORCIDRNumberReportRenders(t *testing.T) {
	store := &fakeStore{user: models.User{ID: 1, Username: "admin", DisplayName: "Admin", Role: models.RoleAdmin}}
	manager := auth.NewManager(store, "12345678901234567890123456789012", "1234567890123456")
	app, err := NewApp(store, manager)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/reports/sales-by-or-ci-dr-number?run=1&coverage=month&month=1&year=2026", nil)
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "STOCK SALES BY REFERENCE NUMBER") || !strings.Contains(body, "Sales From: January 01, 2026 To: January 31, 2026") {
		t.Fatalf("body missing sales by OR/CI/DR title or coverage")
	}
	if !strings.Contains(body, "CI 011497") || !strings.Contains(body, "CASH/MATARANAS") || !strings.Contains(body, "INTEGRA 2000") {
		t.Fatalf("body missing reference group or rows")
	}
	if !strings.Contains(body, "Quantity") || !strings.Contains(body, "Price") || !strings.Contains(body, "Amount") {
		t.Fatalf("body missing sales by OR/CI/DR columns")
	}
	if !strings.Contains(body, "2.00") || !strings.Contains(body, "3,913.00") || !strings.Contains(body, "6,313.00") {
		t.Fatalf("body missing sales by OR/CI/DR totals or amounts")
	}
}

func TestSalesMarkupByTransactionReportRenders(t *testing.T) {
	store := &fakeStore{user: models.User{ID: 1, Username: "admin", DisplayName: "Admin", Role: models.RoleAdmin}}
	manager := auth.NewManager(store, "12345678901234567890123456789012", "1234567890123456")
	app, err := NewApp(store, manager)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/reports/sales-markup-by-transaction?run=1&coverage=month&month=1&year=2026", nil)
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "SALES MARKUP BY TRANSACTION") || !strings.Contains(body, "Sales From: January 01, 2026 To: January 31, 2026") {
		t.Fatalf("body missing sales markup title or coverage")
	}
	if !strings.Contains(body, "CI 011428") || !strings.Contains(body, "CHG 005150") || !strings.Contains(body, "POULTRY SOLUTION") {
		t.Fatalf("body missing sales markup rows")
	}
	if !strings.Contains(body, "Sales Date") || !strings.Contains(body, "Receipt No.") || !strings.Contains(body, "Markup %") {
		t.Fatalf("body missing sales markup columns")
	}
	if !strings.Contains(body, "99.99") || !strings.Contains(body, "5.05") || !strings.Contains(body, "26,100.00") {
		t.Fatalf("body missing sales markup values")
	}
}

func TestSalesSummaryByItemReportRenders(t *testing.T) {
	store := &fakeStore{user: models.User{ID: 1, Username: "admin", DisplayName: "Admin", Role: models.RoleAdmin}}
	manager := auth.NewManager(store, "12345678901234567890123456789012", "1234567890123456")
	app, err := NewApp(store, manager)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/reports/sales-summary-by-item?run=1&coverage=month&month=1&year=2026", nil)
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "STOCK SALES SUMMARY BY ITEM") || !strings.Contains(body, "Sales From: January 01, 2026 To: January 31, 2026") {
		t.Fatalf("body missing sales summary by item title or coverage")
	}
	if !strings.Contains(body, "CHICKEN LINES/B-MEG") || !strings.Contains(body, "INT 1000") || !strings.Contains(body, "INTEGRA 1000") {
		t.Fatalf("body missing category or item rows")
	}
	if !strings.Contains(body, "Stock Code") || !strings.Contains(body, "Stock Name") || !strings.Contains(body, "Amount") {
		t.Fatalf("body missing sales summary by item columns")
	}
	if !strings.Contains(body, "3.00") || !strings.Contains(body, "5,903.00") || !strings.Contains(body, "8,303.00") {
		t.Fatalf("body missing sales summary by item totals")
	}
}

func TestSalesByCustomerReportRenders(t *testing.T) {
	store := &fakeStore{user: models.User{ID: 1, Username: "admin", DisplayName: "Admin", Role: models.RoleAdmin}}
	manager := auth.NewManager(store, "12345678901234567890123456789012", "1234567890123456")
	app, err := NewApp(store, manager)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/reports/sales-by-customer?run=1&coverage=month&month=1&year=2026", nil)
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "STOCK SALES BY CUSTOMER") || !strings.Contains(body, "Sales From: January 01, 2026 To: January 31, 2026") {
		t.Fatalf("body missing sales by customer title or coverage")
	}
	if !strings.Contains(body, "CHICKEN LINES/NESTY") || !strings.Contains(body, "4A MINI MART") || !strings.Contains(body, "NESTY STAG MAINTENANCE") {
		t.Fatalf("body missing category, customer, or stock rows")
	}
	if !strings.Contains(body, "Reference") || !strings.Contains(body, "StockName") || !strings.Contains(body, "Price") {
		t.Fatalf("body missing sales by customer columns")
	}
	if !strings.Contains(body, "6.00") || !strings.Contains(body, "5,910.00") {
		t.Fatalf("body missing sales by customer totals")
	}
}

func TestSalesByCustomerSummaryByItemReportRenders(t *testing.T) {
	store := &fakeStore{user: models.User{ID: 1, Username: "admin", DisplayName: "Admin", Role: models.RoleAdmin}}
	manager := auth.NewManager(store, "12345678901234567890123456789012", "1234567890123456")
	app, err := NewApp(store, manager)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/reports/sales-by-customer-summary-by-item?run=1&coverage=month&month=1&year=2026", nil)
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "STOCK SALES BY CUSTOMER - SUMMARY BY ITEM") || !strings.Contains(body, "Sales From: January 01, 2026 To: January 31, 2026") {
		t.Fatalf("body missing sales by customer summary by item title or coverage")
	}
	if !strings.Contains(body, "CHICKEN LINES/NESTY") || !strings.Contains(body, "4A MINI MART") || !strings.Contains(body, "NESTY STAG MAINTENANCE") {
		t.Fatalf("body missing category, customer, or item rows")
	}
	if !strings.Contains(body, "Code") || !strings.Contains(body, "StockName") || !strings.Contains(body, "Price") {
		t.Fatalf("body missing sales by customer summary by item columns")
	}
	if !strings.Contains(body, "3.00") || !strings.Contains(body, "2,685.00") || !strings.Contains(body, "5,910.00") {
		t.Fatalf("body missing sales by customer summary by item totals")
	}
}

func TestSalesByStockNameReportRenders(t *testing.T) {
	store := &fakeStore{user: models.User{ID: 1, Username: "admin", DisplayName: "Admin", Role: models.RoleAdmin}}
	manager := auth.NewManager(store, "12345678901234567890123456789012", "1234567890123456")
	app, err := NewApp(store, manager)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/reports/sales-by-stock-name?run=1&coverage=month&month=1&year=2026", nil)
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "STOCK SALES BY STOCK NAME") || !strings.Contains(body, "Sales From: January 01, 2026 To: January 31, 2026") {
		t.Fatalf("body missing sales by stock name title or coverage")
	}
	if !strings.Contains(body, "CHICKEN LINES/B-MEG") || !strings.Contains(body, "Stock Code: <strong>INT 1000</strong>") || !strings.Contains(body, "Stock Name: <strong>INTEGRA 1000</strong>") {
		t.Fatalf("body missing category or stock group header")
	}
	if !strings.Contains(body, "CI 011497") || !strings.Contains(body, "CASH/MATARANAS") || !strings.Contains(body, "CATHALEYA AGRI-POULTRY SUPPLY") {
		t.Fatalf("body missing sales rows")
	}
	if !strings.Contains(body, "Reference") || !strings.Contains(body, "Customer:") || !strings.Contains(body, "Price") {
		t.Fatalf("body missing sales by stock name columns")
	}
	if !strings.Contains(body, "2.00") || !strings.Contains(body, "3,992.00") || !strings.Contains(body, "8,303.00") {
		t.Fatalf("body missing sales by stock name totals")
	}
}

func TestPurchasesByStockCodeReportRenders(t *testing.T) {
	store := &fakeStore{user: models.User{ID: 1, Username: "admin", DisplayName: "Admin", Role: models.RoleAdmin}}
	manager := auth.NewManager(store, "12345678901234567890123456789012", "1234567890123456")
	app, err := NewApp(store, manager)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/reports/purchases-by-stock-code?run=1&coverage=month&month=1&year=2026", nil)
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "STOCK PURCHASES BY STOCK CODE") || !strings.Contains(body, "Purchases From: January 01, 2026 To: January 31, 2026") {
		t.Fatalf("body missing purchases by stock code title or coverage")
	}
	if !strings.Contains(body, "Stock Code: <strong>NESTY 7KNDS</strong>") || !strings.Contains(body, "Stock Name: <strong>NESTY 7 KINDS</strong>") {
		t.Fatalf("body missing stock code group header")
	}
	if !strings.Contains(body, "TA #138120") || !strings.Contains(body, "01/13/2026") || !strings.Contains(body, "NESTY") {
		t.Fatalf("body missing purchase rows")
	}
	if !strings.Contains(body, "Reference") || !strings.Contains(body, "Date") || !strings.Contains(body, "Supplier") {
		t.Fatalf("body missing purchases by stock code columns")
	}
	if !strings.Contains(body, "213.00") || !strings.Contains(body, "151,999.90") || !strings.Contains(body, "200.00") {
		t.Fatalf("body missing purchases by stock code totals or amounts")
	}
}

func TestPurchasesBySupplierReportRenders(t *testing.T) {
	store := &fakeStore{user: models.User{ID: 1, Username: "admin", DisplayName: "Admin", Role: models.RoleAdmin}}
	manager := auth.NewManager(store, "12345678901234567890123456789012", "1234567890123456")
	app, err := NewApp(store, manager)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/reports/purchases-by-supplier?run=1&coverage=month&month=1&year=2026", nil)
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "STOCK PURCHASES BY SUPPLIER") || !strings.Contains(body, "Purchases From: January 01, 2026 To: January 31, 2026") {
		t.Fatalf("body missing purchases by supplier title or coverage")
	}
	if !strings.Contains(body, "Supplier: <strong>DG AGRIVET</strong>") || !strings.Contains(body, "PIGROLAC HOG GROWER PELLET VITAL") {
		t.Fatalf("body missing supplier group or stock rows")
	}
	if !strings.Contains(body, "Reference") || !strings.Contains(body, "Date") || !strings.Contains(body, "Code") || !strings.Contains(body, "StockName") {
		t.Fatalf("body missing purchases by supplier columns")
	}
	if !strings.Contains(body, "Stock Total:") || !strings.Contains(body, "Supplier Total:") || !strings.Contains(body, "200.00") || !strings.Contains(body, "336,600.00") {
		t.Fatalf("body missing purchases by supplier totals or amounts")
	}
}

func TestStockLedgerReportRenders(t *testing.T) {
	store := &fakeStore{user: models.User{ID: 1, Username: "admin", DisplayName: "Admin", Role: models.RoleAdmin}}
	manager := auth.NewManager(store, "12345678901234567890123456789012", "1234567890123456")
	app, err := NewApp(store, manager)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/reports/stock-ledger?run=1&coverage=month&month=3&year=2026", nil)
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "STOCK LEDGER") || !strings.Contains(body, "Period Covered : 3/1/2026 ~ 3/31/2026") {
		t.Fatalf("body missing stock ledger title or coverage")
	}
	if !strings.Contains(body, "PILMICO HOGS") || !strings.Contains(body, "CLASSIC FINEX") {
		t.Fatalf("body missing stock ledger category or stock rows")
	}
	if strings.Contains(body, "CLASSIC GROWEX") || strings.Contains(body, "EMPTY CATEGORY") || strings.Contains(body, "EMPTY STOCK") {
		t.Fatalf("body rendered stock ledger rows without quantity values")
	}
	if !strings.Contains(body, "Forwarded Balance") || !strings.Contains(body, "25.00") || !strings.Contains(body, "180.00") {
		t.Fatalf("body missing stock ledger forwarded or running balance values")
	}
	purchaseIndex := strings.Index(body, "ZZ-PURCHASE")
	saleIndex := strings.Index(body, "AA-SALE")
	if purchaseIndex == -1 || saleIndex == -1 || purchaseIndex > saleIndex {
		t.Fatalf("stock ledger did not preserve same-day entry timestamp order")
	}
}
