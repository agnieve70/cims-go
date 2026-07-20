package http

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"cims-go/internal/auth"
	"cims-go/internal/models"
	"cims-go/internal/repositories"
)

type fakeStore struct {
	user                  models.User
	dr                    repositories.DRSelection
	documentValues        models.Record
	documentLines         map[string][]models.Record
	lastMasterSearch      string
	lastMasterYear        int
	lastMasterLimit       int
	lastMasterOffset      int
	masterRecords         []models.Record
	lastDocKind           string
	lastDocSearch         string
	lastDocYear           int
	deletedDocKind        string
	deletedDocID          int64
	deletedMasterKind     string
	deletedMasterID       int64
	saveMasterCalls       int
	lastSavedMasterKind   string
	lastSavedMasterID     int64
	lastSavedMasterValues map[string]string
	saveDocumentCalls     int
	lastSavedDocKind      string
	lastSavedDocID        int64
	lastSavedDocInput     repositories.DocumentInput
	drErr                 error
	userByIDCalls         int
	emptyIncentiveRows    bool
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

func (s *fakeStore) SaveMaster(_ context.Context, form models.FormDefinition, id int64, values map[string]string, _ models.User) (int64, error) {
	s.saveMasterCalls++
	s.lastSavedMasterKind = form.Kind
	s.lastSavedMasterID = id
	s.lastSavedMasterValues = values
	return 1, nil
}

func (s *fakeStore) DeleteMaster(_ context.Context, form models.FormDefinition, id int64, _ models.User) error {
	s.deletedMasterKind = form.Kind
	s.deletedMasterID = id
	return nil
}

func (s *fakeStore) ListDocuments(_ context.Context, kind string, search string, year int) ([]models.DocumentListItem, error) {
	s.lastDocKind = kind
	s.lastDocSearch = search
	s.lastDocYear = year
	return []models.DocumentListItem{{
		ID:          1,
		EntryID:     "ENT-1",
		EntryDate:   time.Date(2026, time.January, 5, 0, 0, 0, 0, time.UTC),
		Party:       "Supplier A",
		Reference:   "REF-1",
		DRRef:       "DR-1",
		Status:      "Open",
		Branch:      "Main",
		Net:         "123.45",
		Encoder:     "Admin",
		Transaction: "0 - Stock Transfer",
		Transactee:  "HERNANS BANSALAN",
		Remarks:     "SO#275457/YFE-407",
		LastUpdate:  "2026-03-14 02:10 PM",
		UpdatedBy:   "Ian",
		TotalQty:    "10.00",
		GrossTotal:  "11,750.00",
		TotalLess:   "0.00",
		TotalAdd:    "0.00",
		NetTotal:    "11,750.00",
	}}, nil
}

func (s *fakeStore) GetDocument(context.Context, models.FormDefinition, int64) (models.Record, map[string][]models.Record, error) {
	if s.documentValues != nil {
		return s.documentValues, s.documentLines, nil
	}
	return models.Record{"id": "ENT-197", "record_id": "197", "entry_date": "2026-04-21"}, nil, nil
}

func (s *fakeStore) LoadDRSelection(context.Context, int64) (repositories.DRSelection, error) {
	if s.drErr != nil {
		return repositories.DRSelection{}, s.drErr
	}
	return s.dr, nil
}

func (s *fakeStore) SaveDocument(_ context.Context, form models.FormDefinition, id int64, input repositories.DocumentInput) (int64, error) {
	s.saveDocumentCalls++
	s.lastSavedDocKind = form.Kind
	s.lastSavedDocID = id
	s.lastSavedDocInput = input
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
		Type:       "Cash",
		GrossCents: 12345,
		NetCents:   12000,
	}}, nil
}

func (s *fakeStore) PurchaseByDRNumberReportRows(context.Context, time.Time, time.Time) ([]models.PurchaseByDRNumberReportRow, error) {
	return []models.PurchaseByDRNumberReportRow{
		{Reference: "CI #47138", PurchaseDate: "01/05/2026", Type: "Cash", Supplier: "SOUTH SEA DESIGNS,INC.", StockCode: "HPSPP 25I", StockName: "(HPS PREM. 25KLS.) PRE-STARTER PELLET PREMIUM", Quantity: 20, UnitCostCents: 140200, AmountCents: 2804000},
		{Reference: "CI #47138", PurchaseDate: "01/05/2026", Type: "Cash", Supplier: "SOUTH SEA DESIGNS,INC.", StockCode: "HGPP", StockName: "HOG GROWER PELLET PREM.", Quantity: 200, UnitCostCents: 186600, AmountCents: 37320000},
		{Reference: "DR 044099", PurchaseDate: "01/07/2026", Type: "Charge", Supplier: "Supplier B", StockCode: "PPC", StockName: "PIG PROTEIN CONCENTRATE", Quantity: 100, UnitCostCents: 231000, AmountCents: 23100000},
	}, nil
}

func (s *fakeStore) PurchaseByStockCodeReportRows(context.Context, time.Time, time.Time) ([]models.PurchaseByStockCodeReportRow, error) {
	return []models.PurchaseByStockCodeReportRow{
		{Reference: "TA #138120", PurchaseDate: "01/13/2026", Type: "Cash", Supplier: "NESTY", StockCode: "NESTY 7KNDS", StockName: "NESTY 7 KINDS", Quantity: 139, UnitCostCents: 73266, AmountCents: 10183974},
		{Reference: "TA #138145", PurchaseDate: "01/19/2026", Type: "Charge", Supplier: "NESTY", StockCode: "NESTY 7KNDS", StockName: "NESTY 7 KINDS", Quantity: 74, UnitCostCents: 67784, AmountCents: 5016016},
		{Reference: "CI #47138", PurchaseDate: "01/05/2026", Type: "Cash", Supplier: "SOUTH SEA DESIGNS,INC.", StockCode: "HGPP", StockName: "HOG GROWER PELLET PREM.", Quantity: 200, UnitCostCents: 186600, AmountCents: 37320000},
	}, nil
}

func (s *fakeStore) PurchaseBySupplierReportRows(context.Context, time.Time, time.Time) ([]models.PurchaseBySupplierReportRow, error) {
	return []models.PurchaseBySupplierReportRow{
		{Reference: "SI #3055", PurchaseDate: "01/16/2026", Type: "Charge", Supplier: "DG AGRIVET", StockCode: "PIGRO VTL", StockName: "PIGROLAC HOG GROWER PELLET VITAL", Quantity: 150, UnitCostCents: 168300, AmountCents: 25245000},
		{Reference: "SI #3273", PurchaseDate: "01/24/2026", Type: "Cash", Supplier: "DG AGRIVET", StockCode: "PIGRO VTL", StockName: "PIGROLAC HOG GROWER PELLET VITAL", Quantity: 50, UnitCostCents: 168300, AmountCents: 8415000},
		{Reference: "TA #138120", PurchaseDate: "01/13/2026", Type: "Cash", Supplier: "NESTY", StockCode: "NESTY 7KNDS", StockName: "NESTY 7 KINDS", Quantity: 139, UnitCostCents: 73266, AmountCents: 10183974},
	}, nil
}

func (s *fakeStore) SalesReportRows(context.Context, time.Time, time.Time) ([]models.SalesReportRow, error) {
	return []models.SalesReportRow{{
		Customer:   "Customer A",
		EntryID:    "ENT-2",
		EntryDate:  "05/30/2026",
		ORCINumber: "CI-1",
		Type:       "Cash",
		GrossCents: 22345,
		NetCents:   22000,
	}}, nil
}

func (s *fakeStore) SalesByORCIDRNumberReportRows(context.Context, time.Time, time.Time) ([]models.SalesByORCIDRNumberReportRow, error) {
	return []models.SalesByORCIDRNumberReportRow{
		{Reference: "AAA CASH", SalesDate: "01/06/2026", Type: "Cash", Customer: "Cash Customer", StockCode: "CASH ITEM", StockName: "CASH ONLY ITEM", Quantity: 1, PriceCents: 10000, AmountCents: 10000},
		{Reference: "CI 011497", SalesDate: "01/07/2026", Type: "Cash", Customer: "CASH/MATARANAS", StockCode: "INT 1000", StockName: "INTEGRA 1000", Quantity: 1, PriceCents: 200200, AmountCents: 200200},
		{Reference: "CI 011497", SalesDate: "01/07/2026", Type: "Cash", Customer: "CASH/MATARANAS", StockCode: "INT 2000", StockName: "INTEGRA 2000", Quantity: 1, PriceCents: 191100, AmountCents: 191100},
		{Reference: "CHG 005245", SalesDate: "01/08/2026", Type: "Charge", Customer: "HYZIE SARI SARI STORE", StockCode: "BOW WOW", StockName: "BOW WOW ADULT", Quantity: 2, PriceCents: 120000, AmountCents: 240000},
	}, nil
}

func (s *fakeStore) SalesMarkupByTransactionReportRows(context.Context, time.Time, time.Time) ([]models.SalesMarkupByTransactionReportRow, error) {
	return []models.SalesMarkupByTransactionReportRow{
		{SalesDate: "01/02/2026", EntryID: "92196", SalesType: "Cash", ReceiptNo: "CI 011428", ItemGroup: "HOGS", MarkupCents: 9999, CapitalCents: 198000, AmountCents: 100000},
		{SalesDate: "01/02/2026", EntryID: "92197", SalesType: "Cash", ReceiptNo: "CI 011429", ItemGroup: "GRAINS", MarkupCents: 3000, CapitalCents: 82418, AmountCents: 150000},
		{SalesDate: "01/02/2026", EntryID: "92207", SalesType: "Charge", ReceiptNo: "CHG 005150", ItemGroup: "POULTRY SOLUTION", MarkupCents: 2610000, CapitalCents: 37716763, AmountCents: 50000000},
	}, nil
}

func (s *fakeStore) SalesByCustomerReportRows(context.Context, time.Time, time.Time) ([]models.SalesByCustomerReportRow, error) {
	return []models.SalesByCustomerReportRow{
		{Category: "CHICKEN LINES/NESTY", Customer: "4A MINI MART", Reference: "CI 011429", SalesDate: "01/02/2026", Type: "Cash", StockCode: "NESTY ST", StockName: "NESTY STAG MAINTENANCE", Quantity: 1, PriceCents: 82500, AmountCents: 82500},
		{Category: "CHICKEN LINES/NESTY", Customer: "AYA/SP GMD STORE", Reference: "CI 011477", SalesDate: "01/06/2026", Type: "Cash", StockCode: "NESTY 7K", StockName: "NESTY 7 KINDS", Quantity: 3, PriceCents: 89500, AmountCents: 268500},
		{Category: "DOG FOOD", Customer: "Cash Customer", Reference: "CHG 005245", SalesDate: "01/08/2026", Type: "Charge", StockCode: "BOW WOW", StockName: "BOW WOW ADULT", Quantity: 2, PriceCents: 120000, AmountCents: 240000},
	}, nil
}

func (s *fakeStore) SalesByStockNameReportRows(context.Context, time.Time, time.Time) ([]models.SalesByStockNameReportRow, error) {
	return []models.SalesByStockNameReportRow{
		{Category: "CHICKEN LINES/B-MEG", Customer: "CASH/MATARANAS", Reference: "CI 011497", SalesDate: "01/07/2026", Type: "Cash", StockCode: "INT 1000", StockName: "INTEGRA 1000", Quantity: 1, PriceCents: 200200, AmountCents: 200200},
		{Category: "CHICKEN LINES/B-MEG", Customer: "CATHALEYA AGRI-POULTRY SUPPLY", Reference: "CI 011499", SalesDate: "01/07/2026", Type: "Charge", StockCode: "INT 1000", StockName: "INTEGRA 1000", Quantity: 1, PriceCents: 199000, AmountCents: 199000},
		{Category: "CHICKEN LINES/B-MEG", Customer: "4A MINI MART", Reference: "CI 011429", SalesDate: "01/02/2026", Type: "Cash", StockCode: "INT 2000", StockName: "INTEGRA 2000", Quantity: 1, PriceCents: 191100, AmountCents: 191100},
		{Category: "DOG FOOD", Customer: "HYZIE SARI SARI STORE", Reference: "CHG 005245", SalesDate: "01/08/2026", Type: "Charge", StockCode: "BOW WOW", StockName: "BOW WOW ADULT", Quantity: 2, PriceCents: 120000, AmountCents: 240000},
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
		{Section: "markup_category", Label: "HOGS", NetSalesCents: 1000000, SalesMarkupCents: 80000, NetTransferCents: 500000, TransferMarkupCents: 35000},
		{Section: "markup_transfer_branch", Label: "HOGS", Branch: "HERNANS MALALAG", NetTransferCents: 300000, TransferMarkupCents: 21000},
		{Section: "markup_transfer_branch", Label: "HOGS", Branch: "HERNANS MALITA", NetTransferCents: 200000, TransferMarkupCents: 14000},
	}, nil
}

func (s *fakeStore) IncentiveReportRows(context.Context, time.Time, time.Time) ([]models.IncentiveReportRow, error) {
	if s.emptyIncentiveRows {
		return nil, nil
	}
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
		{Section: "cash_receipts", Name: "Customer B", Reference: "AR Credit", AmountCents: 50000, CheckAmountCents: 30000},
		{Section: "disbursements", Name: "Fuel", Reference: "EXP-1", AmountCents: 25000},
		{Section: "check_deposits", Name: "Customer C", Reference: "CHK-1", CheckAmountCents: 70000},
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
		{TransferDate: "01/02/2026", EntryID: "33137", TransferTo: "HS STA.MARIA", ReceiptNo: "ST 23343", ItemGroup: "HOGS", MarkupCents: 442420, CapitalCents: 6744200, AmountCents: 8000000},
		{TransferDate: "01/02/2026", EntryID: "33137", TransferTo: "HS STA.MARIA", ReceiptNo: "ST 23343", ItemGroup: "SALTO", MarkupCents: -48790, CapitalCents: 171900, AmountCents: 1200000},
		{TransferDate: "01/02/2026", EntryID: "33138", TransferTo: "HERNANS DIGOS", ReceiptNo: "ST 23344", ItemGroup: "GALLIMAX", MarkupCents: 187485, CapitalCents: 2670536, AmountCents: 4500000},
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
		{Category: "RICE", StockCode: "RICE-01", StockName: "Premium Rice", Bucket2: 8, Bucket5: 999},
	}, nil
}

func (s *fakeStore) StockReorderPointReportRows(context.Context, time.Time) ([]models.StockReorderPointReportRow, error) {
	return []models.StockReorderPointReportRow{
		{Category: "DOG FOOD", StockCode: "BOW", StockName: "BOW WOW", SOH: 0, MinInventory: 10, Deficit: 10},
		{Category: "DOG FOOD", StockCode: "PEDIGREE ADUL", StockName: "PEDIGREE ADULT", SOH: 4, MinInventory: 10, Deficit: 6},
		{Category: "PILMICO HOGS", StockCode: "CLASSIC FI", StockName: "CLASSIC FINEX, 50kg.", SOH: 3, MinInventory: 8, Deficit: 5},
		{Category: "RICE", StockCode: "RICE-SOH", StockName: "Rice With Stock", SOH: 7},
		{Category: "RICE", StockCode: "RICE-ZERO", StockName: "Zero Stock"},
	}, nil
}

func (s *fakeStore) StockSummaryReportRows(context.Context, time.Time) ([]models.StockSummaryReportRow, error) {
	return []models.StockSummaryReportRow{
		{Category: "CHICKEN LINES/NESTY", StockCode: "NESTY 7KNDS", StockName: "NESTY 7 KINDS", HasStock: true, SOH: 230, UnitCostCents: 70234, AmountCents: 16153926},
		{Category: "CHICKEN LINES/NESTY", StockCode: "NESTY COND.", StockName: "NESTY CONDITIONER", HasStock: true, SOH: 140, UnitCostCents: 78000, AmountCents: 10920000},
		{Category: "CHICKEN LINES/NESTY", StockCode: "NESTY ZERO", StockName: "NESTY ZERO", HasStock: true, SOH: 0, UnitCostCents: 12345, AmountCents: 0},
		{Category: "DOG FOOD", StockCode: "BOW", StockName: "BOW WOW", HasStock: true, SOH: 3, UnitCostCents: 100000, AmountCents: 300000},
		{Category: "EMPTY MASTER CATEGORY"},
	}, nil
}

func (s *fakeStore) StockLedgerReportRows(context.Context, time.Time) ([]models.StockLedgerReportRow, error) {
	return []models.StockLedgerReportRow{
		{StockID: "1", Category: "PILMICO HOGS", StockCode: "CLASSIC FI", StockName: "CLASSIC FINEX, 50kg.", EntryDate: "12/31/2025", Reference: "PO-OLD", Company: "Supplier A", Kind: "purchases", QtyDelta: 25},
		{StockID: "1", Category: "PILMICO HOGS", StockCode: "CLASSIC FI", StockName: "CLASSIC FINEX, 50kg.", EntryDate: "03/14/2026", Reference: "PO-1", Company: "Supplier A", Kind: "purchases", QtyDelta: 155},
		{StockID: "1", Category: "PILMICO HOGS", StockCode: "CLASSIC FI", StockName: "CLASSIC FINEX, 50kg.", EntryDate: "03/15/2026", Reference: "SALE-1", Company: "Customer A", Kind: "sales", QtyDelta: -20},
		{StockID: "1", Category: "PILMICO HOGS", StockCode: "CLASSIC FI", StockName: "CLASSIC FINEX, 50kg.", EntryDate: "03/16/2026", Reference: "TR-1", Company: "Branch A", Kind: "stock-transactions", QtyDelta: -15},
		{StockID: "2", Category: "PILMICO HOGS", StockCode: "776", StockName: "CLASSIC GROWEX"},
		{StockID: "3", Category: "EMPTY CATEGORY", StockCode: "EMPTY", StockName: "EMPTY STOCK"},
		{StockID: "category:MASTER ONLY", Category: "MASTER ONLY"},
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
	case "dr_documents":
		return []models.Option{{Value: "7", Label: "SO-0007 - Customer A"}}, nil
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
	if body := rec.Body.String(); !strings.Contains(body, ".purchase-supplier-browse") || !strings.Contains(body, "height: 30px;") || !strings.Contains(body, "min-height: 30px;") {
		t.Fatalf("app css missing legacy Browse button height")
	}
	if body := rec.Body.String(); !strings.Contains(body, ".purchase-legacy-form") || !strings.Contains(body, `"cash . ."`) {
		t.Fatalf("app css missing purchase cash left-column placement")
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

func TestAllMasterFormsCreateUpdateAndDeleteRouteToStore(t *testing.T) {
	for _, form := range models.MasterForms() {
		t.Run(form.Kind, func(t *testing.T) {
			store := &fakeStore{user: models.User{ID: 1, Username: "admin", DisplayName: "Admin", Role: models.RoleAdmin}}
			manager := auth.NewManager(store, "12345678901234567890123456789012", "1234567890123456")
			app, err := NewApp(store, manager)
			if err != nil {
				t.Fatal(err)
			}

			createReq := formPost(form.RouteBase+"/", valuesForFields(form.Fields))
			createReq = createReq.WithContext(auth.WithUser(createReq.Context(), store.user))
			createRec := httptest.NewRecorder()
			app.Routes().ServeHTTP(createRec, createReq)
			if createRec.Code != http.StatusSeeOther {
				t.Fatalf("create status = %d, want %d; body=%s", createRec.Code, http.StatusSeeOther, createRec.Body.String())
			}
			if store.lastSavedMasterKind != form.Kind || store.lastSavedMasterID != 0 {
				t.Fatalf("create saved %s/%d, want %s/0", store.lastSavedMasterKind, store.lastSavedMasterID, form.Kind)
			}

			updateReq := formPost(form.RouteBase+"/42", valuesForFields(form.Fields))
			updateReq = updateReq.WithContext(auth.WithUser(updateReq.Context(), store.user))
			updateRec := httptest.NewRecorder()
			app.Routes().ServeHTTP(updateRec, updateReq)
			if updateRec.Code != http.StatusSeeOther {
				t.Fatalf("update status = %d, want %d; body=%s", updateRec.Code, http.StatusSeeOther, updateRec.Body.String())
			}
			if store.lastSavedMasterKind != form.Kind || store.lastSavedMasterID != 42 {
				t.Fatalf("update saved %s/%d, want %s/42", store.lastSavedMasterKind, store.lastSavedMasterID, form.Kind)
			}

			deleteReq := httptest.NewRequest(http.MethodPost, form.RouteBase+"/42/delete", nil)
			deleteReq = deleteReq.WithContext(auth.WithUser(deleteReq.Context(), store.user))
			deleteRec := httptest.NewRecorder()
			app.Routes().ServeHTTP(deleteRec, deleteReq)
			if deleteRec.Code != http.StatusSeeOther {
				t.Fatalf("delete status = %d, want %d", deleteRec.Code, http.StatusSeeOther)
			}
			if store.deletedMasterKind != form.Kind || store.deletedMasterID != 42 {
				t.Fatalf("deleted master = %s/%d, want %s/42", store.deletedMasterKind, store.deletedMasterID, form.Kind)
			}
		})
	}
}

func TestAllTransactionFormsCreateUpdateAndDeleteRouteToStore(t *testing.T) {
	for _, form := range models.TransactionForms() {
		t.Run(form.Kind, func(t *testing.T) {
			store := &fakeStore{user: models.User{ID: 1, Username: "admin", DisplayName: "Admin", Role: models.RoleAdmin}}
			manager := auth.NewManager(store, "12345678901234567890123456789012", "1234567890123456")
			app, err := NewApp(store, manager)
			if err != nil {
				t.Fatal(err)
			}

			createReq := formPost(form.RouteBase+"/", valuesForFields(form.Fields))
			createReq = createReq.WithContext(auth.WithUser(createReq.Context(), store.user))
			createRec := httptest.NewRecorder()
			app.Routes().ServeHTTP(createRec, createReq)
			if createRec.Code != http.StatusSeeOther {
				t.Fatalf("create status = %d, want %d; body=%s", createRec.Code, http.StatusSeeOther, createRec.Body.String())
			}
			if store.lastSavedDocKind != form.Kind || store.lastSavedDocID != 0 || store.lastSavedDocInput.Kind != form.Kind {
				t.Fatalf("create saved doc %s/%d input %s, want %s/0", store.lastSavedDocKind, store.lastSavedDocID, store.lastSavedDocInput.Kind, form.Kind)
			}

			updateReq := formPost(form.RouteBase+"/42", valuesForFields(form.Fields))
			updateReq = updateReq.WithContext(auth.WithUser(updateReq.Context(), store.user))
			updateRec := httptest.NewRecorder()
			app.Routes().ServeHTTP(updateRec, updateReq)
			if updateRec.Code != http.StatusSeeOther {
				t.Fatalf("update status = %d, want %d; body=%s", updateRec.Code, http.StatusSeeOther, updateRec.Body.String())
			}
			if store.lastSavedDocKind != form.Kind || store.lastSavedDocID != 42 || store.lastSavedDocInput.Kind != form.Kind {
				t.Fatalf("update saved doc %s/%d input %s, want %s/42", store.lastSavedDocKind, store.lastSavedDocID, store.lastSavedDocInput.Kind, form.Kind)
			}

			deleteReq := httptest.NewRequest(http.MethodPost, form.RouteBase+"/42/delete", nil)
			deleteReq = deleteReq.WithContext(auth.WithUser(deleteReq.Context(), store.user))
			deleteRec := httptest.NewRecorder()
			app.Routes().ServeHTTP(deleteRec, deleteReq)
			if deleteRec.Code != http.StatusSeeOther {
				t.Fatalf("delete status = %d, want %d", deleteRec.Code, http.StatusSeeOther)
			}
			if store.deletedDocKind != form.Kind || store.deletedDocID != 42 {
				t.Fatalf("deleted doc = %s/%d, want %s/42", store.deletedDocKind, store.deletedDocID, form.Kind)
			}
		})
	}
}

func formPost(path string, values url.Values) *http.Request {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return req
}

func valuesForFields(fields []models.Field) url.Values {
	values := url.Values{}
	for _, field := range fields {
		switch field.Type {
		case models.FieldBool:
			values.Set(field.Key, "on")
		case models.FieldDate:
			values.Set(field.Key, "2026-01-02")
		case models.FieldMoney:
			values.Set(field.Key, "1.00")
		case models.FieldNumber:
			values.Set(field.Key, "1")
		case models.FieldSelect:
			values.Set(field.Key, "1")
		default:
			values.Set(field.Key, "Test")
		}
	}
	return values
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
	if !strings.Contains(body, `data-field="stock_code">ST-01 - Test Stock</td>`) || strings.Contains(body, `contenteditable="true" data-field="stock_code">ST-01 - Test Stock</td>`) {
		t.Fatalf("body should render SO stock code as read-only")
	}
	if strings.Contains(body, `contenteditable="true" data-field="qty">4</td>`) {
		t.Fatalf("body should render SO quantity as read-only")
	}
	if !strings.Contains(body, `data-sales-edit-stock-out`) {
		t.Fatalf("body missing sales stock out edit trigger")
	}
	if !strings.Contains(body, `data-sales-stock-out-editor-frame`) {
		t.Fatalf("body missing embedded stock out editor frame")
	}
	for _, want := range []string{
		`<div class="sales-summary-body">`,
		`class="line-section modal-line-section purchase-line-section sales-line-payments is-hidden" data-sales-panel="payments"`,
		`<section class="sales-payment-amount">`,
		`data-sales-check-total`,
		`data-sales-balance`,
		`<section class="sales-totals">`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("body missing sales summary layout %q", want)
		}
	}
	for _, want := range []string{
		`<span>SO Number:</span>`,
		`SO Number List`,
		`No matching SO number found.`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("body missing sales SO wording %q", want)
		}
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

func TestAllEditFormsIncludeDeleteButtonAndForm(t *testing.T) {
	store := &fakeStore{user: models.User{ID: 1, Username: "admin", DisplayName: "Admin", Role: models.RoleAdmin}}
	manager := auth.NewManager(store, "12345678901234567890123456789012", "1234567890123456")
	app, err := NewApp(store, manager)
	if err != nil {
		t.Fatal(err)
	}

	for _, form := range models.MasterForms() {
		t.Run("master_"+form.Kind, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, form.RouteBase+"/1/edit", nil)
			req = req.WithContext(auth.WithUser(req.Context(), store.user))
			rec := httptest.NewRecorder()

			app.Routes().ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
			}
			body := rec.Body.String()
			if !strings.Contains(body, `aria-label="Delete current record" form="master-delete-form"`) {
				t.Fatalf("body missing delete button")
			}
			if !strings.Contains(body, `action="`+form.RouteBase+`/1/delete"`) {
				t.Fatalf("body missing delete form action")
			}
		})
	}

	for _, form := range models.TransactionForms() {
		t.Run("transaction_"+form.Kind, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, form.RouteBase+"/197/edit", nil)
			req = req.WithContext(auth.WithUser(req.Context(), store.user))
			rec := httptest.NewRecorder()

			app.Routes().ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
			}
			body := rec.Body.String()
			if !strings.Contains(body, `aria-label="Delete current record" form="master-delete-form"`) {
				t.Fatalf("body missing delete button")
			}
			if !strings.Contains(body, `action="`+form.RouteBase+`/197/delete"`) {
				t.Fatalf("body missing delete form action")
			}
		})
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
	if strings.Contains(body, `contenteditable="true" data-field="stock_code">ST-01 - Test Stock</td>`) || strings.Contains(body, `contenteditable="true" data-field="qty">4</td>`) {
		t.Fatalf("body should keep saved SO detail row read-only")
	}
	if !strings.Contains(body, `name="cash"`) || !strings.Contains(body, `class="sales-field sales-cash"`) {
		t.Fatalf("body missing sales cash checkbox")
	}
}

func TestSalesEditRefreshesRowsFromSelectedSO(t *testing.T) {
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
				"stock_label": "OLD - Removed Stock",
				"qty":         "4",
				"unit_cost":   "12.50",
			}},
		},
		dr: repositories.DRSelection{
			Values: models.Record{"dr_document_id": "7", "party_id": "3"},
			Rows: []models.Record{{
				"dr_line_id":  "21",
				"stock_id":    "8",
				"stock_label": "NEW - Added Stock",
				"qty":         "6",
				"unit_cost":   "15.75",
				"capital":     "15.75",
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
	if !strings.Contains(body, `NEW - Added Stock`) || !strings.Contains(body, `name="line_details_qty" value="6" readonly`) {
		t.Fatalf("body missing refreshed SO detail row")
	}
	if strings.Contains(body, `OLD - Removed Stock`) {
		t.Fatalf("body still contains stale sales detail row")
	}
}

func TestSalesEditKeepsSavedRowsWhenSelectedSOHasNoRemainingQuantity(t *testing.T) {
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
		drErr: errors.New("selected DR has no remaining quantity"),
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
	if strings.Contains(body, "selected DR has no remaining quantity") {
		t.Fatalf("body should not show no remaining quantity alert")
	}
	if !strings.Contains(body, `ST-01 - Test Stock`) || !strings.Contains(body, `name="line_details_qty" value="4" readonly`) {
		t.Fatalf("body should keep saved sales detail row")
	}
}

func TestAPCreditFormMatchesPaymentReferenceLayout(t *testing.T) {
	store := &fakeStore{
		user: models.User{ID: 1, Username: "admin", DisplayName: "Admin", Role: models.RoleAdmin},
		documentValues: models.Record{
			"id":          "4",
			"record_id":   "4",
			"entry_date":  "2026-04-21",
			"reference":   "CV004",
			"party_id":    "1",
			"cash_amount": "12.50",
		},
		documentLines: map[string][]models.Record{
			"checks": {{
				"number":    "0388",
				"date":      "2026-04-22",
				"bank_name": "RCBC",
				"amount":    "281000.00",
				"nature":    "Internal",
			}},
		},
	}
	manager := auth.NewManager(store, "12345678901234567890123456789012", "1234567890123456")
	app, err := NewApp(store, manager)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/transactions/ap-credit/4/edit", nil)
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	for _, want := range []string{
		`<strong>AP Credit File</strong>`,
		`<section class="credit-legacy-form ap-credit-legacy-form">`,
		`<button type="button" disabled>Browse...</button>`,
		`<input type="hidden" name="remarks"`,
		`class="credit-payment-label">PAYMENT</div>`,
		`<th class="credit-hidden-check-nature">Nature</th>`,
		`data-credit-check-total`,
		`data-credit-payment-total`,
		`0388`,
		`RCBC`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("body missing %q", want)
		}
	}
}

func TestAPDebitFormMatchesReferenceLayout(t *testing.T) {
	store := &fakeStore{
		user: models.User{ID: 1, Username: "admin", DisplayName: "Admin", Role: models.RoleAdmin},
		documentValues: models.Record{
			"id":         "55",
			"record_id":  "55",
			"entry_date": "2026-04-21",
			"party_id":   "1",
			"amount":     "799.89",
			"remarks":    "Test debit",
		},
	}
	manager := auth.NewManager(store, "12345678901234567890123456789012", "1234567890123456")
	app, err := NewApp(store, manager)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/transactions/ap-debit/55/edit", nil)
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	for _, want := range []string{
		`<strong>AP Debit File</strong>`,
		`class="debit-legacy-form ap-debit-legacy-form"`,
		`<span>EntryID:</span>`,
		`<button type="button" disabled>Browse...</button>`,
		`<span>Remarks:</span><textarea name="remarks"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("body missing %q", want)
		}
	}
}

func TestStockTransactionFormIncludesStockOutPicker(t *testing.T) {
	store := &fakeStore{user: models.User{ID: 1, Username: "admin", DisplayName: "Admin", Role: models.RoleAdmin}}
	manager := auth.NewManager(store, "12345678901234567890123456789012", "1234567890123456")
	app, err := NewApp(store, manager)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/transactions/stock-transactions/new", nil)
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	for _, want := range []string{
		`<span>SO Number:</span>`,
		`Select SO Number`,
		`SO Number List`,
		`Type SO number or customer name`,
		`No matching SO number found.`,
		`Select an SO number first.`,
		`data-transfer-dr-select`,
		`data-transfer-dr-browse`,
		`data-transfer-dr-picker-search`,
		`data-transfer-dr-picker-results`,
		`data-transfer-edit-stock-out`,
		`data-transfer-stock-out-editor-frame`,
		`SO-0007 - Customer A`,
		`class="sales-summary-body transfer-summary-body"`,
		`Discounts/Additionals/Summary`,
		`<section class="sales-totals"><label>Total:<input readonly data-sales-total></label><label>Less:<input readonly data-sales-less-a></label><label>Add:<input readonly data-sales-add></label><label>Net:<input readonly data-sales-net></label></section>`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("body missing %s", want)
		}
	}
}

func TestStockTransactionEditRefreshesRowsFromSelectedSO(t *testing.T) {
	store := &fakeStore{
		user: models.User{ID: 1, Username: "admin", DisplayName: "Admin", Role: models.RoleAdmin},
		documentValues: models.Record{
			"id":                "ST-197",
			"record_id":         "197",
			"entry_date":        "2026-04-21",
			"dr_document_id":    "7",
			"dr_document_label": "SO-0007 - Customer A",
			"branch_location":   "1",
		},
		documentLines: map[string][]models.Record{
			"details": {{
				"dr_line_id":  "11",
				"stock_id":    "5",
				"stock_label": "OLD - Removed Stock",
				"qty":         "4",
				"unit_cost":   "12.50",
			}},
		},
		dr: repositories.DRSelection{
			Values: models.Record{"dr_document_id": "7", "party_id": "3"},
			Rows: []models.Record{{
				"dr_line_id":  "21",
				"stock_id":    "8",
				"stock_label": "NEW - Added Stock",
				"qty":         "6",
				"unit_cost":   "15.75",
				"capital":     "15.75",
			}},
		},
	}
	manager := auth.NewManager(store, "12345678901234567890123456789012", "1234567890123456")
	app, err := NewApp(store, manager)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/transactions/stock-transactions/197/edit", nil)
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `NEW - Added Stock`) || !strings.Contains(body, `name="line_details_qty" value="6" readonly`) {
		t.Fatalf("body missing refreshed SO detail row")
	}
	if !strings.Contains(body, `class="sales-detail-row" data-sales-detail-row`) {
		t.Fatalf("stock transaction detail row missing shared Esc-delete marker")
	}
	if strings.Contains(body, `OLD - Removed Stock`) {
		t.Fatalf("body still contains stale stock transaction detail row")
	}
}

func TestStockTransactionEditKeepsSavedRowsWhenSelectedSOHasNoRemainingQuantity(t *testing.T) {
	store := &fakeStore{
		user: models.User{ID: 1, Username: "admin", DisplayName: "Admin", Role: models.RoleAdmin},
		documentValues: models.Record{
			"id":                "ST-197",
			"record_id":         "197",
			"entry_date":        "2026-04-21",
			"dr_document_id":    "7",
			"dr_document_label": "SO-0007 - Customer A",
			"branch_location":   "1",
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
		drErr: errors.New("selected DR has no remaining quantity"),
	}
	manager := auth.NewManager(store, "12345678901234567890123456789012", "1234567890123456")
	app, err := NewApp(store, manager)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/transactions/stock-transactions/197/edit", nil)
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if strings.Contains(body, "selected DR has no remaining quantity") {
		t.Fatalf("body should not show no remaining quantity alert")
	}
	if !strings.Contains(body, `ST-01 - Test Stock`) || !strings.Contains(body, `name="line_details_qty" value="4" readonly`) {
		t.Fatalf("body should keep saved stock transaction detail row")
	}
}

func TestMergeLinkedStockOutDetailRowsPreservesSalesEditableValues(t *testing.T) {
	refreshed := []models.Record{{
		"dr_line_id":  "21",
		"stock_id":    "8",
		"stock_label": "NEW - Added Stock",
		"qty":         "9",
		"unit_cost":   "15.75",
		"capital":     "15.75",
	}}
	existing := []models.Record{{
		"dr_line_id":     "11",
		"stock_id":       "8",
		"stock_label":    "NEW - Added Stock",
		"qty":            "6",
		"unit_cost":      "22.25",
		"discount":       "1.50",
		"other_discount": "0.75",
		"capital":        "22.25",
		"markup":         "15.00",
		"markup_pct":     "10.00",
	}}

	got := mergeLinkedStockOutDetailRows(refreshed, existing, "sales")

	if got[0]["qty"] != "9" {
		t.Fatalf("qty = %q, want refreshed Stock Out qty", got[0]["qty"])
	}
	for field, want := range map[string]string{
		"unit_cost":      "22.25",
		"discount":       "1.50",
		"other_discount": "0.75",
	} {
		if got[0][field] != want {
			t.Fatalf("%s = %q, want %q", field, got[0][field], want)
		}
	}
	for field, want := range map[string]string{
		"capital":    "15.75",
		"markup":     "",
		"markup_pct": "",
	} {
		if got[0][field] != want {
			t.Fatalf("%s = %q, want refreshed/calculated value %q", field, got[0][field], want)
		}
	}
}

func TestMergeLinkedStockOutDetailRowsPreservesStockTransactionEditableValues(t *testing.T) {
	refreshed := []models.Record{{
		"dr_line_id":     "21",
		"stock_id":       "8",
		"stock_label":    "NEW - Added Stock",
		"qty":            "9",
		"unit_cost":      "15.75",
		"capital":        "15.75",
		"discount":       "2.00",
		"other_discount": "1.00",
	}}
	existing := []models.Record{{
		"dr_line_id":     "11",
		"stock_id":       "8",
		"stock_label":    "NEW - Added Stock",
		"qty":            "6",
		"unit_cost":      "22.25",
		"discount":       "1.50",
		"other_discount": "0.75",
		"capital":        "22.25",
		"markup":         "15.00",
		"markup_pct":     "10.00",
	}}

	got := mergeLinkedStockOutDetailRows(refreshed, existing, "stock-transactions")

	if got[0]["qty"] != "9" {
		t.Fatalf("qty = %q, want refreshed Stock Out qty", got[0]["qty"])
	}
	for field, want := range map[string]string{
		"unit_cost": "22.25",
	} {
		if got[0][field] != want {
			t.Fatalf("%s = %q, want %q", field, got[0][field], want)
		}
	}
	for field, want := range map[string]string{
		"capital":    "15.75",
		"markup":     "",
		"markup_pct": "",
	} {
		if got[0][field] != want {
			t.Fatalf("%s = %q, want refreshed/calculated value %q", field, got[0][field], want)
		}
	}
	if got[0]["discount"] != "2.00" || got[0]["other_discount"] != "1.00" {
		t.Fatalf("stock transaction should not copy sales discount fields: got discount=%q other=%q", got[0]["discount"], got[0]["other_discount"])
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
	if !strings.Contains(body, `name="latest_cost" type="number" step="0.001" value="" readonly tabindex="-1" aria-readonly="true"`) {
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
	if !strings.Contains(body, `class="legacy-strip-icon-button"`) || !strings.Contains(body, `class="legacy-strip-refresh">Refresh</button>`) {
		t.Fatalf("body missing legacy list search or refresh controls")
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
	if !strings.Contains(body, `class="legacy-strip-icon-button"`) || !strings.Contains(body, `class="legacy-strip-refresh">Refresh</button>`) {
		t.Fatalf("body missing legacy backdrop search or refresh controls")
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

func TestPurchaseListUsesLegacyColumns(t *testing.T) {
	store := &fakeStore{user: models.User{ID: 1, Username: "admin", DisplayName: "Admin", Role: models.RoleAdmin}}
	manager := auth.NewManager(store, "12345678901234567890123456789012", "1234567890123456")
	app, err := NewApp(store, manager)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/transactions/purchases/?year=2026&q=", nil)
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	for _, want := range []string{
		"<th>Purchase ...</th>",
		"<th>Reference</th>",
		"<th>Supplier</th>",
		"<th>Total Qty</th>",
		"<th>Gross Total</th>",
		"<th>Total Deducti...</th>",
		"<th>Total Additio...</th>",
		"<th>Net Total</th>",
		"<th>Last Update</th>",
		"<th>Updated By</th>",
		`class="legacy-strip-icon-button"`,
		`class="legacy-strip-refresh">Refresh</button>`,
		"10.00",
		"11,750.00",
		"2026-03-14 02:10 PM",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("purchase list missing %q", want)
		}
	}
	if strings.Contains(body, "<th>Company</th>") || strings.Contains(body, "<th>DR Ref</th>") || strings.Contains(body, "<th>Status</th>") {
		t.Fatalf("purchase list still uses generic transaction columns")
	}
}

func TestSalesListUsesLegacyColumns(t *testing.T) {
	store := &fakeStore{user: models.User{ID: 1, Username: "admin", DisplayName: "Admin", Role: models.RoleAdmin}}
	manager := auth.NewManager(store, "12345678901234567890123456789012", "1234567890123456")
	app, err := NewApp(store, manager)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/transactions/sales/?year=2026&q=", nil)
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	for _, want := range []string{
		"<th>Sales Date</th>",
		"<th>Reference</th>",
		"<th>Client</th>",
		"<th>Total Qty</th>",
		"<th>Gross Total</th>",
		"<th>Total Deducti...</th>",
		"<th>Total Additio...</th>",
		"<th>Net Total</th>",
		"<th>Last Update</th>",
		"<th>Updated By</th>",
		`class="legacy-strip-icon-button"`,
		`class="legacy-strip-refresh">Refresh</button>`,
		"10.00",
		"11,750.00",
		"2026-03-14 02:10 PM",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("sales list missing %q", want)
		}
	}
	if strings.Contains(body, "<th>Company</th>") || strings.Contains(body, "<th>DR Ref</th>") || strings.Contains(body, "<th>Status</th>") {
		t.Fatalf("sales list still uses generic transaction columns")
	}
}

func TestStockTransactionListUsesLegacyColumns(t *testing.T) {
	store := &fakeStore{user: models.User{ID: 1, Username: "admin", DisplayName: "Admin", Role: models.RoleAdmin}}
	manager := auth.NewManager(store, "12345678901234567890123456789012", "1234567890123456")
	app, err := NewApp(store, manager)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/transactions/stock-transactions/?year=2026&q=", nil)
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	for _, want := range []string{
		"<th>Transacti...</th>",
		"<th>Transaction</th>",
		"<th>Transactee</th>",
		"<th>Net Total</th>",
		"<th>Last Update</th>",
		"<th>Updated By</th>",
		`class="legacy-strip-icon-button"`,
		`class="legacy-strip-refresh">Refresh</button>`,
		"0 - Stock Transfer",
		"HERNANS BANSALAN",
		"SO#275457/YFE-407",
		"2026-03-14 02:10 PM",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("stock transaction list missing %q", want)
		}
	}
	if strings.Contains(body, "<th>Company</th>") || strings.Contains(body, "<th>DR Ref</th>") {
		t.Fatalf("stock transaction list still uses generic transaction columns")
	}
}

func TestOutgoingCheckListReportRendersDetailedAndSummary(t *testing.T) {
	store := &fakeStore{user: models.User{ID: 1, Username: "admin", DisplayName: "Admin", Role: models.RoleAdmin}}
	manager := auth.NewManager(store, "12345678901234567890123456789012", "1234567890123456")
	app, err := NewApp(store, manager)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/reports/outgoing-check-list", nil)
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("options status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Specify Cut-off Date") || strings.Contains(body, ">Cut-off:</span>") {
		t.Fatalf("outgoing check options should match cut-off date reference dialog")
	}

	req = httptest.NewRequest(http.MethodGet, "/reports/outgoing-check-list?run=1&report_type=detailed&cutoff_date=2026-05-31", nil)
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec = httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("detailed status = %d, want %d", rec.Code, http.StatusOK)
	}
	body = rec.Body.String()
	if !strings.Contains(body, "OUTGOING CHECK LIST") || !strings.Contains(body, "Supplier A") || strings.Contains(body, "Supplier B") {
		t.Fatalf("detailed body did not render only cutoff-included outgoing checks")
	}
	if !strings.Contains(body, "SORONGON RICE &amp; CORN MILL") {
		t.Fatalf("detailed body missing outgoing check company title")
	}
	if !strings.Contains(body, `data-report-tree-toggle`) || !strings.Contains(body, `report-tree-branch collapsed`) {
		t.Fatalf("detailed outgoing check preview should show expandable payee markers")
	}
	for _, column := range []string{"Reference", "Date", "Number", "Bank Name", "Amount"} {
		if !strings.Contains(body, column) {
			t.Fatalf("body missing outgoing check detailed column %q", column)
		}
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
	if !strings.Contains(body, "SORONGON RICE &amp; CORN MILL") {
		t.Fatalf("summary body missing outgoing check company title")
	}
	if !strings.Contains(body, "Check Date Cut-Off: 31-May-2026") || !strings.Contains(body, "G R A N D&nbsp;&nbsp; T O T A L:") {
		t.Fatalf("summary body missing outgoing check reference labels")
	}
}

func TestIncomingCheckListSummaryMatchesReferenceLabels(t *testing.T) {
	store := &fakeStore{user: models.User{ID: 1, Username: "admin", DisplayName: "Admin", Role: models.RoleAdmin}}
	manager := auth.NewManager(store, "12345678901234567890123456789012", "1234567890123456")
	app, err := NewApp(store, manager)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/reports/incoming-check-list", nil)
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("options status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Specify Cut-off Date") || strings.Contains(body, ">Cut-off:</span>") {
		t.Fatalf("incoming check options should match cut-off date reference dialog")
	}

	req = httptest.NewRequest(http.MethodGet, "/reports/incoming-check-list?run=1&report_type=detailed&cutoff_date=2026-05-30", nil)
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec = httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("detailed status = %d, want %d", rec.Code, http.StatusOK)
	}
	body = rec.Body.String()
	if !strings.Contains(body, "INCOMING CHECK LIST") || strings.Contains(body, "SORONGON RICE &amp; CORN MILL") {
		t.Fatalf("detailed body should show only incoming check list title")
	}
	if !strings.Contains(body, `data-report-tree-toggle`) || !strings.Contains(body, `class="report-tree-branch collapsed"`) {
		t.Fatalf("detailed incoming check preview should show expandable payee markers")
	}
	for _, column := range []string{"Reference", "Date", "Number", "Bank Name", "Amount"} {
		if !strings.Contains(body, column) {
			t.Fatalf("body missing incoming check detailed column %q", column)
		}
	}

	req = httptest.NewRequest(http.MethodGet, "/reports/incoming-check-list?run=1&report_type=summary-postdated&cutoff_date=2026-05-29", nil)
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec = httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body = rec.Body.String()
	if !strings.Contains(body, "INCOMING CHECK ACCOUNT SUMMARY (Postdated)") || !strings.Contains(body, "Customer A") {
		t.Fatalf("body missing incoming check summary title or payee")
	}
	if !strings.Contains(body, "SORONGON RICE &amp; CORN MILL") {
		t.Fatalf("body missing incoming check company title")
	}
	if !strings.Contains(body, "Check Date Cut-Off: 29-May-2026") || !strings.Contains(body, "G R A N D&nbsp;&nbsp; T O T A L:") {
		t.Fatalf("summary body missing incoming check reference labels")
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
	if !strings.Contains(body, "SORONGON AGRIVET") {
		t.Fatalf("body missing expenses summary company title")
	}
	if !strings.Contains(body, "ADVANCES TO EMPLOYEES") {
		t.Fatalf("body missing expense category row")
	}
	if !strings.Contains(body, `class="expenses-summary-grand-total"`) || !strings.Contains(body, "Grand Total") {
		t.Fatalf("body missing standalone expenses summary grand total")
	}
	if strings.Contains(body, "<tfoot><tr><td colspan=\"2\">Grand Total</td>") {
		t.Fatalf("expenses summary grand total should not render as a bordered table footer")
	}
}

func TestExpensesDetailedPreviewMatchesReference(t *testing.T) {
	store := &fakeStore{user: models.User{ID: 1, Username: "admin", DisplayName: "Admin", Role: models.RoleAdmin}}
	manager := auth.NewManager(store, "12345678901234567890123456789012", "1234567890123456")
	app, err := NewApp(store, manager)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/reports/expenses-summary?run=1&report_type=detailed&coverage=month&month=5&year=2026", nil)
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "DETAILED EXPENSES") || !strings.Contains(body, "Purchases From: Mayo 01, 2026 To: Mayo 31, 2026") {
		t.Fatalf("body missing detailed expenses title or coverage")
	}
	if strings.Contains(body, ">Page 1</button>") || strings.Contains(body, ">Page 2</button>") {
		t.Fatalf("detailed expenses preview should not render page nodes")
	}
	if strings.Contains(body, `class="report-tree-node"`) || strings.Contains(body, "No expense records</span>") {
		t.Fatalf("detailed expenses preview should remain blank like the reference")
	}
}

func TestAPLedgerAgingMatchesReferenceHeaders(t *testing.T) {
	store := &fakeStore{user: models.User{ID: 1, Username: "admin", DisplayName: "Admin", Role: models.RoleAdmin}}
	manager := auth.NewManager(store, "12345678901234567890123456789012", "1234567890123456")
	app, err := NewApp(store, manager)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/reports/ap-ledger?run=1&report_type=detailed&coverage=month&month=1&year=2026", nil)
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Accounts Payable Ledger") || !strings.Contains(body, "Supplier Code:") || !strings.Contains(body, "Supplier Name:") {
		t.Fatalf("body missing AP detailed ledger title or supplier profile")
	}
	if strings.Contains(body, "Purchases From:") || strings.Contains(body, "Report as of:") {
		t.Fatalf("AP detailed ledger should not render a date range/as-of line")
	}
	for _, column := range []string{"Date", "Reference", "Debit", "Credit", "Balance"} {
		if !strings.Contains(body, column) {
			t.Fatalf("body missing AP detailed column %q", column)
		}
	}

	req = httptest.NewRequest(http.MethodGet, "/reports/ap-ledger?run=1&report_type=summary&coverage=month&month=1&year=2026", nil)
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec = httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("summary status = %d, want %d", rec.Code, http.StatusOK)
	}
	body = rec.Body.String()
	if !strings.Contains(body, "Accounts Payable Summary") || !strings.Contains(body, "Summary as of: Enero 31, 2026") {
		t.Fatalf("body missing AP summary title or as-of label")
	}
	for _, column := range []string{"Code", "Company", "Representative", "Balance"} {
		if !strings.Contains(body, column) {
			t.Fatalf("body missing AP summary column %q", column)
		}
	}

	req = httptest.NewRequest(http.MethodGet, "/reports/ap-ledger?run=1&report_type=aging&coverage=month&month=1&year=2026", nil)
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec = httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("aging status = %d, want %d", rec.Code, http.StatusOK)
	}
	body = rec.Body.String()
	if !strings.Contains(body, "Accounts Receivable Aging") || !strings.Contains(body, "Report as of: Enero 31, 2026") {
		t.Fatalf("body missing AP aging title or as-of label")
	}
	if strings.Contains(body, `class="report-tree-node"`) || strings.Contains(body, "No AP ledger records") {
		t.Fatalf("AP aging preview should be blank like the reference screenshot")
	}
	if !strings.Contains(body, "SORONGON AGRIVET") {
		t.Fatalf("body missing AP aging company title")
	}
	if !strings.Contains(body, "Last Payment") || !strings.Contains(body, "0 to30 Days") || strings.Contains(body, "0 to 30 Days") {
		t.Fatalf("body missing AP aging reference bucket headers")
	}
}

func TestARLedgerAgingMatchesReferenceHeaders(t *testing.T) {
	store := &fakeStore{user: models.User{ID: 1, Username: "admin", DisplayName: "Admin", Role: models.RoleAdmin}}
	manager := auth.NewManager(store, "12345678901234567890123456789012", "1234567890123456")
	app, err := NewApp(store, manager)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/reports/ar-ledger?run=1&report_type=detailed&coverage=month&month=1&year=2026", nil)
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Accounts Receivable Ledger") || !strings.Contains(body, "Customer Name:") || !strings.Contains(body, "Credit Limit:") {
		t.Fatalf("body missing AR detailed ledger title or customer profile")
	}
	if strings.Contains(body, "Sales From:") || strings.Contains(body, "Report as of:") {
		t.Fatalf("AR detailed ledger should not render a date range/as-of line")
	}
	for _, column := range []string{"Date", "Reference", "Debit", "Credit", "Balance"} {
		if !strings.Contains(body, column) {
			t.Fatalf("body missing AR detailed column %q", column)
		}
	}

	req = httptest.NewRequest(http.MethodGet, "/reports/ar-ledger?run=1&report_type=summary&coverage=month&month=1&year=2026", nil)
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec = httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("summary status = %d, want %d", rec.Code, http.StatusOK)
	}
	body = rec.Body.String()
	if !strings.Contains(body, "Accounts Receivable Summary") || !strings.Contains(body, "Summary as of: Enero 31, 2026") {
		t.Fatalf("body missing AR summary title or as-of label")
	}
	for _, column := range []string{"Company", "Balance"} {
		if !strings.Contains(body, column) {
			t.Fatalf("body missing AR summary column %q", column)
		}
	}

	req = httptest.NewRequest(http.MethodGet, "/reports/ar-ledger?run=1&report_type=aging&coverage=month&month=1&year=2026", nil)
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec = httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("aging status = %d, want %d", rec.Code, http.StatusOK)
	}
	body = rec.Body.String()
	if !strings.Contains(body, "Accounts Receivable Aging") || !strings.Contains(body, "Report as of: Enero 31, 2026") {
		t.Fatalf("body missing AR aging title or as-of label")
	}
	if strings.Contains(body, `class="report-tree-node"`) || strings.Contains(body, "No AR ledger records") {
		t.Fatalf("AR aging preview should be blank like the reference screenshot")
	}
	if !strings.Contains(body, "SORONGON AGRIVET") {
		t.Fatalf("body missing AR aging company title")
	}
	if !strings.Contains(body, "Last Payment") || !strings.Contains(body, "0 to30 Days") || strings.Contains(body, "0 to 30 Days") {
		t.Fatalf("body missing AR aging reference bucket headers")
	}
	if !strings.Contains(body, "Outstanding Check") || !strings.Contains(body, "Total Balance") {
		t.Fatalf("body missing AR aging balance columns")
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
	if !strings.Contains(body, "Income Statement") || !strings.Contains(body, "From: Mayo 01, 2026 To: Mayo 31, 2026") {
		t.Fatalf("body missing income statement title or coverage")
	}
	if !strings.Contains(body, "SORONGON AGRIVET") {
		t.Fatalf("body missing income statement company title")
	}
	if !strings.Contains(body, "NET INCOME") {
		t.Fatalf("body missing net income row")
	}
	if !strings.Contains(body, "ADVANCES TO EMPLOYEES") {
		t.Fatalf("body missing operating expense row")
	}
	for _, want := range []string{"Sales Markup", "Transfer Markup", "Total Markup", "HOGS", "HERNANS MALALAG", "CATEGORY TOTAL", "GRAND TOTAL"} {
		if !strings.Contains(body, want) {
			t.Fatalf("body missing income statement legacy markup content %q", want)
		}
	}
	if !strings.Contains(body, `data-income-statement-page="2"`) || !strings.Contains(body, `data-income-statement-page="3"`) {
		t.Fatalf("body missing generated income statement markup pages")
	}
	for _, previewNode := range []string{">Sales</button>", ">Cost of Sales</button>", ">Operating Expenses</button>", ">Other Income</button>", ">Net Income</button>"} {
		if strings.Contains(body, previewNode) {
			t.Fatalf("income statement preview should not render node %q", previewNode)
		}
	}
	if strings.Contains(body, `class="report-tree-node"`) || strings.Contains(body, `class="report-tree-empty"`) {
		t.Fatalf("income statement preview pane should remain blank like the reference")
	}
}

func TestIncomeStatementAccountingPagesFillRemainingSpace(t *testing.T) {
	report := incomeStatementReportData{PaperSize: "letter"}
	rows := []models.IncomeStatementRow{
		{Section: "cash_sales", Label: "Cash Sales", AmountCents: 10000},
		{Section: "charge_sales", Label: "Charge Sales", AmountCents: 20000},
		{Section: "beginning_inventory", Label: "Stock Inventory, Beginning", AmountCents: 50000},
		{Section: "ending_inventory", Label: "Stock Inventory, End", AmountCents: 10000},
	}
	for idx := 0; idx < 50; idx++ {
		rows = append(rows, models.IncomeStatementRow{
			Section:     "purchases",
			Label:       fmt.Sprintf("Supplier %02d", idx+1),
			AmountCents: 1000,
		})
	}

	report.build(rows)

	if len(report.AccountingPages) < 2 {
		t.Fatalf("AccountingPages = %d, want at least 2", len(report.AccountingPages))
	}
	foundCostOfSalesOnFirstPage := false
	for _, section := range report.AccountingPages[0].Sections {
		if strings.HasPrefix(section.Title, "COST OF SALES") && len(section.Rows) > 0 {
			foundCostOfSalesOnFirstPage = true
		}
	}
	if !foundCostOfSalesOnFirstPage {
		t.Fatalf("first page did not use remaining height for cost of sales rows: %#v", report.AccountingPages[0].Sections)
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
	if !strings.Contains(body, "INCENTIVE REPORT") || !strings.Contains(body, "Sales From: Mayo 01, 2026 To: Mayo 31, 2026") {
		t.Fatalf("body missing incentive report title or coverage")
	}
	if !strings.Contains(body, "SORONGON AGRIVET") {
		t.Fatalf("body missing incentive report company title")
	}
	if !strings.Contains(body, "Agri Post") || !strings.Contains(body, "TAKALS") || !strings.Contains(body, "FARM") {
		t.Fatalf("body missing incentive report columns")
	}
	if !strings.Contains(body, "APS") || !strings.Contains(body, "10") {
		t.Fatalf("body missing incentive report row")
	}
	if strings.Contains(body, ">APS</button>") {
		t.Fatalf("incentive preview should not render agri post nodes")
	}
	if strings.Contains(body, `class="report-tree-node"`) || strings.Contains(body, `class="report-tree-empty"`) {
		t.Fatalf("incentive preview pane should remain blank like the reference")
	}
}

func TestIncentiveReportEmptyRowsMatchesReferenceShape(t *testing.T) {
	store := &fakeStore{user: models.User{ID: 1, Username: "admin", DisplayName: "Admin", Role: models.RoleAdmin}, emptyIncentiveRows: true}
	manager := auth.NewManager(store, "12345678901234567890123456789012", "1234567890123456")
	app, err := NewApp(store, manager)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/reports/incentive?run=1&coverage=month&month=12&year=2025", nil)
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `class="incentive-report-empty-row"`) {
		t.Fatalf("body missing blank incentive row for empty result")
	}
	if strings.Contains(body, "No incentive records") {
		t.Fatalf("body should not show empty-message text in incentive reference layout")
	}
	if !strings.Contains(body, "Total :") || !strings.Contains(body, "Group Total :") || !strings.Contains(body, "Grand Total :") {
		t.Fatalf("body missing incentive total rows")
	}
	if strings.Contains(body, `<td class="num">0</td>`) || strings.Contains(body, `<td colspan="4" class="num">0</td>`) {
		t.Fatalf("empty incentive reference layout should leave total cells blank")
	}
}

func TestDailySalesCollectionReportRenders(t *testing.T) {
	store := &fakeStore{user: models.User{ID: 1, Username: "admin", DisplayName: "Admin", Role: models.RoleAdmin}}
	manager := auth.NewManager(store, "12345678901234567890123456789012", "1234567890123456")
	app, err := NewApp(store, manager)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/reports/daily-sales-collection", nil)
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("options status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Select Date of Report") || strings.Contains(body, ">Date:</span>") {
		t.Fatalf("daily sales options should match date-only reference dialog")
	}

	req = httptest.NewRequest(http.MethodGet, "/reports/daily-sales-collection?run=1&report_date=2026-03-14", nil)
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec = httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body = rec.Body.String()
	if !strings.Contains(body, "DAILY SALES AND COLLECTION REPORT") {
		t.Fatalf("body missing daily sales report title")
	}
	if !strings.Contains(body, "Report Date: 03/14/2026") || !strings.Contains(body, "CHG 005965") {
		t.Fatalf("body missing selected date or charge sales row")
	}
	if !strings.Contains(body, "CASH SALES") || !strings.Contains(body, "CHECK DEPOSITS") {
		t.Fatalf("body missing daily sales section headers")
	}
	if !strings.Contains(body, "TOTAL CASH RECEIPTS") || !strings.Contains(body, "500.00") {
		t.Fatalf("body missing cash receipts section total")
	}
	wantCashReceiptsTotal := `<td colspan="3">TOTAL CASH RECEIPTS</td>
                <td class="num daily-sales-total-amount">500.00</td>
                <td class="num">300.00</td>`
	if !strings.Contains(body, wantCashReceiptsTotal) {
		t.Fatalf("body missing cash receipts amount and check totals")
	}
	if !strings.Contains(body, "TOTAL CHECK DEPOSITS") || !strings.Contains(body, "700.00") {
		t.Fatalf("body missing check deposits section total")
	}
	for _, previewNode := range []string{">CASH SALES</button>", ">CHARGE SALES</button>", ">CASH RECEIPTS</button>", ">DISBURSEMENTS</button>", ">CHECK DEPOSITS</button>"} {
		if strings.Contains(body, previewNode) {
			t.Fatalf("daily sales preview should not render section node %q", previewNode)
		}
	}
	if !strings.Contains(body, "TOTAL CASH REMITTANCE") || !strings.Contains(body, "1,450.00") {
		t.Fatalf("body missing cash remittance total")
	}
	if !strings.Contains(body, "TOTAL REMITTANCE") || !strings.Contains(body, "2,150.00") {
		t.Fatalf("body missing remittance total")
	}
}

func TestDailySalesCollectionReportPaginatesLongReports(t *testing.T) {
	report := dailySalesCollectionReportData{PaperSize: "letter"}
	rows := make([]models.DailySalesCollectionReportRow, 0, 50)
	for idx := 0; idx < 50; idx++ {
		rows = append(rows, models.DailySalesCollectionReportRow{
			Section:     "cash_sales",
			Name:        fmt.Sprintf("Customer %02d", idx+1),
			Reference:   fmt.Sprintf("OR %06d", idx+1),
			AmountCents: 10000,
			SortKey:     fmt.Sprintf("%02d", idx+1),
		})
	}

	report.build(rows)

	if report.TotalPages < 2 {
		t.Fatalf("TotalPages = %d, want at least 2", report.TotalPages)
	}
	if len(report.Pages) != report.TotalPages {
		t.Fatalf("len(Pages) = %d, want %d", len(report.Pages), report.TotalPages)
	}
	if !report.Pages[len(report.Pages)-1].ShowSummary {
		t.Fatalf("last page should show remittance summary")
	}
	for idx, page := range report.Pages[:len(report.Pages)-1] {
		if page.ShowSummary {
			t.Fatalf("page %d should not show remittance summary", idx+1)
		}
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

func TestDailyDueCheckOptionsMatchReferencePrompt(t *testing.T) {
	store := &fakeStore{user: models.User{ID: 1, Username: "admin", DisplayName: "Admin", Role: models.RoleAdmin}}
	manager := auth.NewManager(store, "12345678901234567890123456789012", "1234567890123456")
	app, err := NewApp(store, manager)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/reports/daily-due-check", nil)
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Specify report cut-off date:") {
		t.Fatalf("body missing daily due check cutoff prompt")
	}
	if strings.Contains(body, ">Cut-off:</span>") {
		t.Fatalf("daily due check options should not render extra Cut-off label")
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
	if strings.Contains(body, "incoming-check-calendar-titlebar") {
		t.Fatalf("body should not render extra incoming check calendar titlebar")
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
	if strings.Contains(body, "STOCK SALES AND TRANSFER REPORT") {
		t.Fatalf("stock sales transfer report should not render a generic report title")
	}
	if !strings.Contains(body, "CHICKEN LINES/B-MEG") || !strings.Contains(body, "INTEGRA 2000") {
		t.Fatalf("body missing category or stock row")
	}
	if !strings.Contains(body, "<td>Category:</td>") || !strings.Contains(body, "<td colspan=\"2\">Total :</td>") {
		t.Fatalf("body missing stock sales transfer category or total row")
	}
	if !strings.Contains(body, "164") || !strings.Contains(body, "1,538") || !strings.Contains(body, "1,702") {
		t.Fatalf("body missing sales, transfer, or total quantities")
	}
	if strings.Contains(body, "stock-transfer-grand-total") {
		t.Fatalf("stock sales transfer report should not render a separate grand total table")
	}

	req = httptest.NewRequest(http.MethodGet, "/reports/stock-sales-transfer-amount?run=1&coverage=month&month=5&year=2026", nil)
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec = httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("amount status = %d, want %d", rec.Code, http.StatusOK)
	}
	body = rec.Body.String()
	if !strings.Contains(body, "SORONGON AGRIVET") || !strings.Contains(body, "STOCK SALES AND TRANSFER AMOUNT SUMMARY") || !strings.Contains(body, "AQUA") {
		t.Fatalf("body missing stock sales transfer amount report")
	}
	if !strings.Contains(body, "Sales From: Mayo 01, 2026 To: Mayo 31, 2026") {
		t.Fatalf("body missing Filipino month range label")
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
	if !strings.Contains(body, "STOCK TRANSFER") || !strings.Contains(body, "Sales From: Enero 01, 2026 To: Enero 31, 2026") {
		t.Fatalf("body missing stock transfer summary title or coverage")
	}
	if !strings.Contains(body, "HERNANS BANSALAN") || !strings.Contains(body, "TAHOP WHITE BAGGER") {
		t.Fatalf("body missing branch or stock rows")
	}
	if !strings.Contains(body, "Total :") || !strings.Contains(body, "36,090.00") {
		t.Fatalf("body missing summary totals")
	}
	if !strings.Contains(body, "stock-transfer-summary-grand-total") || !strings.Contains(body, "Grand Total :") {
		t.Fatalf("body missing stock transfer summary grand total")
	}
	if !strings.Contains(body, "stock-transfer-summary-branch-block") {
		t.Fatalf("body missing stock transfer summary branch print blocks")
	}
	if !strings.Contains(body, `<span class="report-tree-expander" aria-hidden="true"></span>BY PRODUCT</button>`) {
		t.Fatalf("body missing stock transfer summary expandable category preview node")
	}
	if !strings.Contains(body, `report-tree-child`) || !strings.Contains(body, `data-report-scroll-target="stock-transfer-summary-branch-`) || !strings.Contains(body, `data-report-highlight-target="stock-transfer-summary-branch-`) || !strings.Contains(body, `data-report-highlight-label-only`) {
		t.Fatalf("body missing stock transfer summary branch preview highlight nodes")
	}
}

func TestStockTransferSummaryReportBuildPaginatesLargeCategory(t *testing.T) {
	rows := make([]models.StockTransferSummaryReportRow, 0, 18)
	for i := 0; i < 18; i++ {
		rows = append(rows, models.StockTransferSummaryReportRow{
			Category:     "FEEDS/B-MEG",
			Branch:       "BRANCH " + strconv.Itoa(i+1),
			Reference:    "TR" + strconv.Itoa(i+1),
			TransferDate: "06/27/2026",
			StockCode:    "ST" + strconv.Itoa(i+1),
			StockName:    "FEED " + strconv.Itoa(i+1),
			Quantity:     100,
			AmountCents:  10000,
		})
	}

	var report stockTransferSummaryReportData
	report.build(rows)

	if len(report.Pages) < 2 {
		t.Fatalf("pages = %d, want large category split across multiple pages", len(report.Pages))
	}
	if report.TotalPages != len(report.Pages) {
		t.Fatalf("total pages = %d, want %d", report.TotalPages, len(report.Pages))
	}
	lastPage := report.Pages[len(report.Pages)-1]
	if !lastPage.ShowGrandTotal || lastPage.GrandQuantity != report.TotalQuantity || lastPage.GrandAmount != report.TotalAmount {
		t.Fatalf("last page missing grand total")
	}
	for _, category := range report.Categories {
		for _, branch := range category.Branches {
			if branch.PageNumber < 1 || branch.PageNumber > report.TotalPages || branch.TargetID == "" {
				t.Fatalf("branch %q has invalid page/target: page=%d target=%q", branch.Name, branch.PageNumber, branch.TargetID)
			}
		}
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
	if !strings.Contains(body, "STOCK TRANSFER BY STOCK NAME") || !strings.Contains(body, "Sales From: Enero 01, 2026 To: Enero 31, 2026") {
		t.Fatalf("body missing stock transfer by stock name title or coverage")
	}
	if !strings.Contains(body, "HAMMERED YELLOW CORN") || !strings.Contains(body, "ST 23362 SO_27") {
		t.Fatalf("body missing stock or transfer rows")
	}
	if strings.Contains(body, `report-tree-child`) {
		t.Fatalf("body includes stock transfer by stock name preview subitems")
	}
	if !strings.Contains(body, `data-report-tree-toggle aria-expanded="false"><span class="report-tree-expander" aria-hidden="true"></span>BY PRODUCT</button>`) {
		t.Fatalf("body missing stock transfer by stock name expandable category preview node")
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
	if !strings.Contains(body, "STOCK TRANSFER BY BRANCH") || !strings.Contains(body, "Sales From: Enero 01, 2026 To: Enero 31, 2026") {
		t.Fatalf("body missing stock transfer by branch title or coverage")
	}
	if !strings.Contains(body, "HERNANS BANSALAN") || !strings.Contains(body, "Category:") || !strings.Contains(body, "CHICKEN LINES/B-MEG") {
		t.Fatalf("body missing branch or category sections")
	}
	if strings.Contains(body, `report-tree-child`) || strings.Contains(body, `data-report-scroll-target="transfer-branch-category-`) {
		t.Fatalf("body includes stock transfer by branch preview subitems")
	}
	if !strings.Contains(body, `data-report-tree-toggle aria-expanded="false"><span class="report-tree-expander" aria-hidden="true"></span>HERNANS BANSALAN</button>`) {
		t.Fatalf("body missing stock transfer by branch expandable branch preview node")
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
	if !strings.Contains(body, "STOCK TRANSFER BY ENTRY ID") || !strings.Contains(body, "Sales From: Enero 01, 2026 To: Enero 31, 2026") {
		t.Fatalf("body missing stock transfer by entry id title or coverage")
	}
	if !strings.Contains(body, "Entry ID:") || !strings.Contains(body, "33,155") {
		t.Fatalf("body missing entry id")
	}
	if strings.Contains(body, "Transfer ID:") || strings.Contains(body, "<th>Transfer ID</th>") || strings.Contains(body, "<th>Branch</th>") {
		t.Fatalf("body includes stock transfer by entry id fields not shown in screenshot")
	}
	if !strings.Contains(body, "Entry Date:") || !strings.Contains(body, "<th>Branch Name</th>") || !strings.Contains(body, "<th>Code</th>") || !strings.Contains(body, "<th>StockName</th>") {
		t.Fatalf("body missing stock transfer by entry id screenshot columns")
	}
	if strings.Contains(body, "stock-transfer-summary-grand-total") || strings.Contains(body, "Grand Total:") {
		t.Fatalf("body includes stock transfer by entry id grand total")
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
	if !strings.Contains(body, "STOCK TRANSFER SUMMARY BY ENTRY ID") || !strings.Contains(body, "Sales From: Enero 01, 2026 To: Enero 31, 2026") {
		t.Fatalf("body missing stock transfer summary by entry id title or coverage")
	}
	if !strings.Contains(body, "HERNANS BANSALAN") || !strings.Contains(body, "33,155") || !strings.Contains(body, "ST 23362 SO_27") {
		t.Fatalf("body missing branch, entry id, or remarks")
	}
	if !strings.Contains(body, `<span class="report-tree-expander" aria-hidden="true"></span>HERNANS BANSALAN</button>`) {
		t.Fatalf("body missing stock transfer summary by entry id branch preview node")
	}
	if strings.Contains(body, `report-tree-child`) || strings.Contains(body, `data-report-scroll-target="transfer-entry-summary-row-`) {
		t.Fatalf("stock transfer summary by entry id preview should not render entry child nodes")
	}
	if strings.Contains(body, "stock-transfer-entry-summary-grand-total") || strings.Contains(body, "Grand Total:") {
		t.Fatalf("body includes stock transfer summary by entry id grand total")
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
	if !strings.Contains(body, "STOCK TRANSFER SUMMARY BY ITEM") || !strings.Contains(body, "Sales From: Enero 01, 2026 To: Enero 31, 2026") {
		t.Fatalf("body missing stock transfer summary by item title or coverage")
	}
	if !strings.Contains(body, "CORN") || !strings.Contains(body, "HAMMERED YELLOW CORN") || !strings.Contains(body, "TAHOP WHITE BAGGER") {
		t.Fatalf("body missing category or stock rows")
	}
	if strings.Contains(body, `report-tree-child`) || strings.Contains(body, `data-report-scroll-target="stock-transfer-summary-item-row-`) {
		t.Fatalf("body includes stock transfer summary by item preview subitems")
	}
	if !strings.Contains(body, `data-report-tree-toggle aria-expanded="false"><span class="report-tree-expander" aria-hidden="true"></span>CORN</button>`) {
		t.Fatalf("body missing stock transfer summary by item expandable category preview node")
	}
	if !strings.Contains(body, "Code") || !strings.Contains(body, "StockName") || !strings.Contains(body, "Amount") {
		t.Fatalf("body missing stock transfer summary by item columns")
	}
	if strings.Contains(body, "stock-transfer-summary-item-grand-total") || strings.Contains(body, "Grand Total:") {
		t.Fatalf("body includes stock transfer summary by item grand total")
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
	if !strings.Contains(body, "SALES MARKUP BY TRANSACTION") || !strings.Contains(body, "Sales From: Enero 01, 2026 To: Enero 31, 2026") {
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
	if strings.Contains(body, `class="report-tree-node"`) || strings.Contains(body, "No markup records") {
		t.Fatalf("transfer markup by transaction preview should be empty like the reference screenshot")
	}
}

func TestStockAgingReportRenders(t *testing.T) {
	store := &fakeStore{user: models.User{ID: 1, Username: "admin", DisplayName: "Admin", Role: models.RoleAdmin}}
	manager := auth.NewManager(store, "12345678901234567890123456789012", "1234567890123456")
	app, err := NewApp(store, manager)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/reports/stock-aging?run=1&paper_size=a4-landscape&cutoff_date=2026-03-14", nil)
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "SORONGON AGRIVET") || !strings.Contains(body, "STOCK AGING") || !strings.Contains(body, "As Of 3/14/2026") {
		t.Fatalf("body missing stock aging title or cutoff")
	}
	if !strings.Contains(body, "report-paper-size-a4-landscape") || !strings.Contains(body, `value="legal-landscape"`) {
		t.Fatalf("body missing stock aging landscape paper sizes")
	}
	if !strings.Contains(body, "PILMICO HOGS") || !strings.Contains(body, "CLASSIC FINEX") {
		t.Fatalf("body missing stock aging category or row")
	}
	if strings.Contains(body, "report-tree-section\">BY PRODUCT") {
		t.Fatalf("body includes hard-coded stock aging preview category")
	}
	if !strings.Contains(body, "02/12/2026 ~ 03/14/2026(30)") || !strings.Contains(body, "01/13/2026 ~ 02/11/2026(60)") || !strings.Contains(body, "10/15/2025 ~ 11/13/2025(150)") {
		t.Fatalf("body missing stock aging bucket labels")
	}
	if strings.Contains(body, "(150+)") || strings.Contains(body, "999") {
		t.Fatalf("body includes stock aging values outside the 30 to 150 day buckets")
	}
	if strings.Contains(body, "G R A N D") {
		t.Fatalf("body includes standalone stock aging grand total")
	}
	if !strings.Contains(body, "165") || !strings.Contains(body, "477") || !strings.Contains(body, "8") {
		t.Fatalf("body missing stock aging totals")
	}

	req = httptest.NewRequest(http.MethodGet, "/reports/stock-aging", nil)
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec = httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("options status = %d, want %d", rec.Code, http.StatusOK)
	}
	body = rec.Body.String()
	if !strings.Contains(body, "Specify Cut-off Date") || strings.Contains(body, ">Cut-off:</span>") {
		t.Fatalf("stock aging options should match cut-off date reference dialog")
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
	if !strings.Contains(body, "SORONGON AGRIVET") || !strings.Contains(body, "STOCK REORDER POINT") || !strings.Contains(body, "Stock Summary As Of: Marso 14, 2026") {
		t.Fatalf("body missing stock reorder point title or cutoff")
	}
	if !strings.Contains(body, "DOG FOOD") || !strings.Contains(body, "PEDIGREE ADULT") {
		t.Fatalf("body missing stock reorder point category or row")
	}
	if !strings.Contains(body, "BOW WOW") || !strings.Contains(body, "0.00") {
		t.Fatalf("body should include zero-SOH reorder point rows with positive deficit")
	}
	if strings.Contains(body, "Rice With Stock") || strings.Contains(body, "Zero Stock") {
		t.Fatalf("body should filter stock reorder point rows by positive deficit")
	}
	if !strings.Contains(body, "SOH") || !strings.Contains(body, "Min. Inv.") || !strings.Contains(body, "Deficit") {
		t.Fatalf("body missing stock reorder point columns")
	}
	if !strings.Contains(body, "0.00") || !strings.Contains(body, "4.00") || !strings.Contains(body, "10.00") || !strings.Contains(body, "6.00") {
		t.Fatalf("body missing stock reorder point quantities")
	}
	if !strings.Contains(body, `<span class="report-tree-expander" aria-hidden="true"></span>DOG FOOD</button>`) {
		t.Fatalf("body missing stock reorder category preview node")
	}
	if strings.Contains(body, `report-tree-child`) || strings.Contains(body, `data-report-scroll-target="stock-reorder-row-`) {
		t.Fatalf("stock reorder preview should not render stock child nodes")
	}

	req = httptest.NewRequest(http.MethodGet, "/reports/stock-reorder-point", nil)
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec = httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("options status = %d, want %d", rec.Code, http.StatusOK)
	}
	body = rec.Body.String()
	if !strings.Contains(body, "Specify Cut-off Date") || strings.Contains(body, ">Cut-off:</span>") {
		t.Fatalf("stock reorder point options should match cut-off date reference dialog")
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
	if !strings.Contains(body, "SORONGON AGRIVET") || !strings.Contains(body, "STOCK SUMMARY") || !strings.Contains(body, "Stock Summary As Of: Marso 14, 2026") {
		t.Fatalf("body missing stock summary title or cutoff")
	}
	if !strings.Contains(body, "CHICKEN LINES/NESTY") || !strings.Contains(body, "NESTY 7 KINDS") {
		t.Fatalf("body missing stock summary category or row")
	}
	if !strings.Contains(body, `data-report-always-full-tree`) {
		t.Fatalf("stock summary preview should keep the full category tree visible")
	}
	if !strings.Contains(body, `EMPTY MASTER CATEGORY</button>`) || !strings.Contains(body, "No stocks in this category.") {
		t.Fatalf("body missing empty stock summary master category")
	}
	if strings.Contains(body, `data-report-noop aria-expanded="false"><span class="report-tree-expander" aria-hidden="true"></span>EMPTY MASTER CATEGORY</button>`) {
		t.Fatalf("empty stock summary category should still navigate to its report page")
	}
	if strings.Contains(body, `<td>NESTY ZERO</td>`) {
		t.Fatalf("body should not include zero-SOH stock summary row")
	}
	if !strings.Contains(body, "SOH") || !strings.Contains(body, "Unit Cost") || !strings.Contains(body, "Amount") {
		t.Fatalf("body missing stock summary columns")
	}
	if !strings.Contains(body, "230.00") || !strings.Contains(body, "702.34") || !strings.Contains(body, "161,539.26") {
		t.Fatalf("body missing stock summary quantities or amounts")
	}
	if strings.Contains(body, "stock-summary-grand-total") || strings.Contains(body, "Grand Totals:") {
		t.Fatalf("stock summary should not render standalone grand totals")
	}
	if !strings.Contains(body, `CHICKEN LINES/NESTY</button>`) || !strings.Contains(body, `data-report-tree-toggle`) {
		t.Fatalf("body missing stock summary category preview node")
	}
	if !strings.Contains(body, `report-tree-child`) || !strings.Contains(body, `data-report-scroll-target="stock-summary-row-`) {
		t.Fatalf("stock summary preview should render stock child nodes")
	}
	if !strings.Contains(body, `data-report-highlight-target="stock-summary-row-`) {
		t.Fatalf("stock summary stock child nodes should highlight matching rows")
	}
	if strings.Contains(body, `data-report-filter-target="stock-summary-row-`) {
		t.Fatalf("stock summary stock child nodes should not filter out sibling rows")
	}
	if !strings.Contains(body, `>NESTY ZERO</button>`) {
		t.Fatalf("stock summary preview should render stock child nodes even when the printed row is filtered")
	}
	if !strings.Contains(body, `data-report-noop>NESTY ZERO</button>`) {
		t.Fatalf("zero-SOH stock preview child should be non-navigating")
	}
	if strings.Contains(body, `data-report-scroll-target="stock-summary-row-3-3">NESTY ZERO</button>`) {
		t.Fatalf("zero-SOH stock preview child should not target a hidden printed row")
	}

	req = httptest.NewRequest(http.MethodGet, "/reports/stock-summary", nil)
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec = httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("options status = %d, want %d", rec.Code, http.StatusOK)
	}
	body = rec.Body.String()
	if !strings.Contains(body, "Specify Cut-off Date") || strings.Contains(body, ">Cut-off:</span>") {
		t.Fatalf("stock summary options should match cut-off date reference dialog")
	}
}

func TestPurchasesSummaryDetailedMatchesReferenceColumns(t *testing.T) {
	store := &fakeStore{user: models.User{ID: 1, Username: "admin", DisplayName: "Admin", Role: models.RoleAdmin}}
	manager := auth.NewManager(store, "12345678901234567890123456789012", "1234567890123456")
	app, err := NewApp(store, manager)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/reports/purchases-summary?run=1&report_type=detailed&coverage=month&month=1&year=2026", nil)
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "DETAILED PURCHASES") || !strings.Contains(body, "Purchases From: Enero 01, 2026 To: Enero 31, 2026") {
		t.Fatalf("body missing detailed purchases title or coverage")
	}
	for _, column := range []string{"Entry ID", "Date", "OR/CI Number", "Gross Amount", "Net Amount"} {
		if !strings.Contains(body, column) {
			t.Fatalf("body missing purchase summary column %q", column)
		}
	}
	if strings.Contains(body, "<th>Type</th>") || strings.Contains(body, "report-col-type") {
		t.Fatalf("purchase summary detailed report should not render a Type column")
	}

	req = httptest.NewRequest(http.MethodGet, "/reports/purchases-summary?run=1&report_type=summary&coverage=month&month=1&year=2026", nil)
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec = httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("summary status = %d, want %d", rec.Code, http.StatusOK)
	}
	body = rec.Body.String()
	if !strings.Contains(body, "SORONGON AGRIVET") || !strings.Contains(body, "PURCHASES SUMMARY") || !strings.Contains(body, "Purchases From: Enero 01, 2026 To: Enero 31, 2026") {
		t.Fatalf("body missing purchases summary title or coverage")
	}
	if !strings.Contains(body, "Supplier:") || !strings.Contains(body, "Gross Amount") || !strings.Contains(body, "Net Amount") {
		t.Fatalf("body missing purchases summary columns")
	}
	if !strings.Contains(body, "Supplier A") || !strings.Contains(body, "Total Purchases Made:") || strings.Contains(body, "<th>Type</th>") {
		t.Fatalf("body missing purchases summary rows or includes type column")
	}
}

func TestSalesSummaryDetailedMatchesReferenceColumns(t *testing.T) {
	store := &fakeStore{user: models.User{ID: 1, Username: "admin", DisplayName: "Admin", Role: models.RoleAdmin}}
	manager := auth.NewManager(store, "12345678901234567890123456789012", "1234567890123456")
	app, err := NewApp(store, manager)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/reports/sales-summary?run=1&report_type=detailed&coverage=month&month=1&year=2026", nil)
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "DETAILED SALES") || !strings.Contains(body, "Sales From: Enero 01, 2026 To: Enero 31, 2026") {
		t.Fatalf("body missing detailed sales title or coverage")
	}
	for _, column := range []string{"Entry ID", "Date", "OR/CI Number", "Gross Amount", "Net Amount"} {
		if !strings.Contains(body, column) {
			t.Fatalf("body missing sales summary column %q", column)
		}
	}
	if strings.Contains(body, "<th>Type</th>") || strings.Contains(body, "report-col-type") {
		t.Fatalf("sales summary detailed report should not render a Type column")
	}

	req = httptest.NewRequest(http.MethodGet, "/reports/sales-summary?run=1&report_type=summary&coverage=month&month=1&year=2026", nil)
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec = httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("summary status = %d, want %d", rec.Code, http.StatusOK)
	}
	body = rec.Body.String()
	if !strings.Contains(body, "SORONGON AGRIVET") || !strings.Contains(body, "SALES SUMMARY") || !strings.Contains(body, "Sales From: Enero 01, 2026 To: Enero 31, 2026") {
		t.Fatalf("body missing sales summary title or coverage")
	}
	if !strings.Contains(body, "<th>Customer</th>") || !strings.Contains(body, "Gross Amount") || !strings.Contains(body, "Net Amount") {
		t.Fatalf("body missing sales summary columns")
	}
	if !strings.Contains(body, "Customer A") || !strings.Contains(body, "Total Sales Made:") || strings.Contains(body, "<th>Type</th>") {
		t.Fatalf("body missing sales summary rows or includes type column")
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
	if !strings.Contains(body, "SORONGON AGRIVET") || !strings.Contains(body, "STOCK PURCHASES BY REFERENCE NUMBER") || !strings.Contains(body, "Purchases From: Enero 01, 2026 To: Enero 31, 2026") {
		t.Fatalf("body missing purchases by DR title or coverage")
	}
	if !strings.Contains(body, "CI #47138") || !strings.Contains(body, "SOUTH SEA DESIGNS,INC.") || !strings.Contains(body, "HOG GROWER PELLET PREM.") {
		t.Fatalf("body missing reference group or rows")
	}
	if !strings.Contains(body, "Reference: <strong>CI #47138</strong>") || !strings.Contains(body, "Sales Date:") {
		t.Fatalf("body missing purchases by DR reference metadata")
	}
	if strings.Contains(body, "<th>Type</th>") || strings.Contains(body, ">Cash</td>") || strings.Contains(body, ">Charge</td>") {
		t.Fatalf("purchases by DR report should not render a Type column")
	}
	if !strings.Contains(body, "Quantity") || !strings.Contains(body, "Cost") || !strings.Contains(body, "Amount") {
		t.Fatalf("body missing purchases by DR columns")
	}
	if !strings.Contains(body, "220.00") || !strings.Contains(body, "1,866.00") || !strings.Contains(body, "401,240.00") {
		t.Fatalf("body missing purchases by DR totals or amounts")
	}
	if strings.Contains(body, `data-report-scroll-target="purchase-dr-row-1-1"`) || strings.Contains(body, `purchase-dr-grand-total`) {
		t.Fatalf("purchases by DR report should render flat reference preview and no grand total block")
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
	if !strings.Contains(body, "STOCK SALES BY REFERENCE NUMBER") || !strings.Contains(body, "Sales From: Enero 01, 2026 To: Enero 31, 2026") {
		t.Fatalf("body missing sales by OR/CI/DR title or coverage")
	}
	if !strings.Contains(body, "SORONGON AGRIVET") {
		t.Fatalf("body missing sales by OR/CI/DR store title")
	}
	if !strings.Contains(body, "CI 011497") || !strings.Contains(body, "CASH/MATARANAS") || !strings.Contains(body, "INTEGRA 2000") {
		t.Fatalf("body missing reference group or rows")
	}
	chargeIndex := strings.Index(body, `data-report-tree-page="1">CHG 005245`)
	cashIndex := strings.Index(body, `data-report-tree-page="2">AAA CASH`)
	if chargeIndex < 0 || cashIndex < 0 || chargeIndex > cashIndex {
		t.Fatalf("sales by OR/CI/DR preview should list cash-unchecked sales before cash-checked sales")
	}
	if strings.Contains(body, "report-tree-child") || strings.Contains(body, "data-report-scroll-target=\"sales-or-row-") {
		t.Fatalf("body includes sales by OR/CI/DR preview subitems")
	}
	if strings.Contains(body, "<th>Type</th>") || strings.Contains(body, ">Cash</td>") || strings.Contains(body, ">Charge</td>") {
		t.Fatalf("body includes sales by OR/CI/DR cash charge type")
	}
	if !strings.Contains(body, "Qty") || !strings.Contains(body, "Price") || !strings.Contains(body, "Amount") {
		t.Fatalf("body missing sales by OR/CI/DR columns")
	}
	if strings.Contains(body, "sales-or-grand-total") || strings.Contains(body, "Grand Total:") {
		t.Fatalf("body includes sales by OR/CI/DR grand total")
	}
	if !strings.Contains(body, "2.00") || !strings.Contains(body, "3,913.00") || !strings.Contains(body, "2,002.00") {
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
	if !strings.Contains(body, "SALES MARKUP BY TRANSACTION") || !strings.Contains(body, "Sales From: Enero 01, 2026 To: Enero 31, 2026") {
		t.Fatalf("body missing sales markup title or coverage")
	}
	if !strings.Contains(body, "CI 011428") || !strings.Contains(body, "CHG 005150") || !strings.Contains(body, "POULTRY SOLUTION") {
		t.Fatalf("body missing sales markup rows")
	}
	if !strings.Contains(body, "Sales Date") || !strings.Contains(body, "Receipt No.") || !strings.Contains(body, "Markup %") {
		t.Fatalf("body missing sales markup columns")
	}
	if !strings.Contains(body, "99.99") || !strings.Contains(body, "10.00") || !strings.Contains(body, "26,100.00") {
		t.Fatalf("body missing sales markup values")
	}
	if strings.Contains(body, `class="report-tree-node"`) || strings.Contains(body, "No markup records") {
		t.Fatalf("sales markup by transaction preview should be empty like the reference screenshot")
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
	if !strings.Contains(body, "STOCK SALES SUMMARY BY ITEM") || !strings.Contains(body, "Sales From: Enero 01, 2026 To: Enero 31, 2026") {
		t.Fatalf("body missing sales summary by item title or coverage")
	}
	if !strings.Contains(body, "SORONGON AGRIVET") {
		t.Fatalf("body missing sales summary by item store title")
	}
	if !strings.Contains(body, "CHICKEN LINES/B-MEG") || !strings.Contains(body, "INT 1000") || !strings.Contains(body, "INTEGRA 1000") {
		t.Fatalf("body missing category or item rows")
	}
	if strings.Contains(body, "Cash Sales") || strings.Contains(body, "Charge Sales") {
		t.Fatalf("body includes sales summary by item cash charge columns")
	}
	if !strings.Contains(body, "Stock Code") || !strings.Contains(body, "Stock Name") || !strings.Contains(body, "Quantity") || !strings.Contains(body, "Amount") {
		t.Fatalf("body missing sales summary by item columns")
	}
	if strings.Contains(body, "sales-summary-item-grand-total") || strings.Contains(body, "Grand Total:") {
		t.Fatalf("body includes sales summary by item grand total")
	}
	if !strings.Contains(body, `<span class="report-tree-expander" aria-hidden="true"></span>CHICKEN LINES/B-MEG</button>`) {
		t.Fatalf("body missing sales summary by item category preview node")
	}
	if !strings.Contains(body, `data-report-highlight-category="sales-summary-item-category-1"`) || !strings.Contains(body, `data-report-category-name`) || !strings.Contains(body, `report-category-name-highlight`) {
		t.Fatalf("sales summary by item category preview should highlight the category label")
	}
	if !strings.Contains(body, `report-tree-child`) || !strings.Contains(body, `data-report-scroll-target="sales-summary-item-row-`) || !strings.Contains(body, `data-report-highlight-target="sales-summary-item-row-`) {
		t.Fatalf("sales summary by item preview should render stock child nodes with scroll and highlight targets")
	}
	if !strings.Contains(body, `>INTEGRA 1000</button>`) || !strings.Contains(body, `>INTEGRA 2000</button>`) {
		t.Fatalf("sales summary by item preview should render sold stock names under categories")
	}
	if !strings.Contains(body, "3.00") || !strings.Contains(body, "5,903.00") || !strings.Contains(body, "3,992.00") {
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
	if !strings.Contains(body, "STOCK SALES BY CUSTOMER") || !strings.Contains(body, "Sales From: Enero 01, 2026 To: Enero 31, 2026") {
		t.Fatalf("body missing sales by customer title or coverage")
	}
	if !strings.Contains(body, "SORONGON AGRIVET") {
		t.Fatalf("body missing sales by customer store title")
	}
	if !strings.Contains(body, "CHICKEN LINES/NESTY") || !strings.Contains(body, "4A MINI MART") || !strings.Contains(body, "NESTY STAG MAINTENANCE") {
		t.Fatalf("body missing category, customer, or stock rows")
	}
	if !strings.Contains(body, `data-report-scroll-target="sales-customer-category-1"`) || !strings.Contains(body, `>CHICKEN LINES/NESTY</button>`) {
		t.Fatalf("body missing sales by customer category preview node")
	}
	if !strings.Contains(body, `data-report-scroll-target="sales-customer-1-1"`) || !strings.Contains(body, `>4A MINI MART</button>`) || !strings.Contains(body, `>AYA/SP GMD STORE</button>`) {
		t.Fatalf("body missing sales by customer customer preview nodes")
	}
	if !strings.Contains(body, `class="report-preview-tree" data-report-always-full-tree`) {
		t.Fatalf("sales by customer preview should keep the full category and customer tree visible")
	}
	if !strings.Contains(body, `data-report-highlight-target="sales-customer-1-1"`) || !strings.Contains(body, `data-report-highlight-label-only`) || !strings.Contains(body, `data-report-filter-groups="sales-customer-filter-1-1"`) {
		t.Fatalf("body missing sales by customer customer highlight target")
	}
	if !strings.Contains(body, `data-report-highlight-category="sales-customer-category-1"`) || !strings.Contains(body, `report-category-name-highlight`) {
		t.Fatalf("sales by customer category preview should highlight the category label")
	}
	if strings.Contains(body, `data-report-filter-target="sales-customer-filter-1-1"`) {
		t.Fatalf("sales by customer customer preview should not hide other customers")
	}
	if !strings.Contains(body, `data-report-category-name`) || !strings.Contains(body, `data-report-stock-name`) || !strings.Contains(body, `report-stock-name-highlight`) {
		t.Fatalf("sales by customer should wire category and customer-name highlighting")
	}
	if strings.Contains(body, "<th>Type</th>") || strings.Contains(body, ">Cash</td>") || strings.Contains(body, ">Charge</td>") {
		t.Fatalf("body includes sales by customer cash charge type")
	}
	if !strings.Contains(body, "Reference") || !strings.Contains(body, "StockName") || !strings.Contains(body, "Price") {
		t.Fatalf("body missing sales by customer columns")
	}
	if strings.Contains(body, "sales-customer-grand-total") || strings.Contains(body, "Grand Total:") {
		t.Fatalf("body includes sales by customer grand total")
	}
	if !strings.Contains(body, "4.00") || !strings.Contains(body, "3,510.00") || !strings.Contains(body, "825.00") {
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
	if !strings.Contains(body, "STOCK SALES BY CUSTOMER - SUMMARY BY ITEM") || !strings.Contains(body, "Sales From: Enero 01, 2026 To: Enero 31, 2026") {
		t.Fatalf("body missing sales by customer summary by item title or coverage")
	}
	if !strings.Contains(body, "SORONGON AGRIVET") {
		t.Fatalf("body missing sales by customer summary by item store title")
	}
	if !strings.Contains(body, "CHICKEN LINES/NESTY") || !strings.Contains(body, "4A MINI MART") || !strings.Contains(body, "NESTY STAG MAINTENANCE") {
		t.Fatalf("body missing category, customer, or item rows")
	}
	if !strings.Contains(body, `data-report-highlight-category="sales-customer-summary-category-1"`) || !strings.Contains(body, `data-report-category-name`) || !strings.Contains(body, `report-category-name-highlight`) {
		t.Fatalf("sales by customer summary by item category preview should highlight the category label")
	}
	if !strings.Contains(body, `data-report-scroll-target="sales-customer-summary-1-1"`) || !strings.Contains(body, `data-report-highlight-label-only`) || !strings.Contains(body, `data-report-stock-name`) || !strings.Contains(body, `>4A MINI MART</button>`) {
		t.Fatalf("sales by customer summary by item preview should render customer child nodes")
	}
	if !strings.Contains(body, `data-report-scroll-target="sales-customer-summary-row-1-1-1"`) || !strings.Contains(body, `data-report-highlight-target="sales-customer-summary-row-`) || !strings.Contains(body, `>NESTY STAG MAINTENANCE</button>`) {
		t.Fatalf("sales by customer summary by item preview should render stock child nodes")
	}
	if strings.Contains(body, `>EMPTY MASTER CATEGORY</button>`) || strings.Contains(body, `>No sales records</span>`) {
		t.Fatalf("sales by customer summary by item preview should only render categories with sales data")
	}
	if strings.Contains(body, "Cash Sales") || strings.Contains(body, "Charge Sales") {
		t.Fatalf("body includes sales by customer summary by item cash charge columns")
	}
	if !strings.Contains(body, "Code") || !strings.Contains(body, "StockName") || !strings.Contains(body, "Quantity") || !strings.Contains(body, "Price") || !strings.Contains(body, "Amount") {
		t.Fatalf("body missing sales by customer summary by item columns")
	}
	if strings.Contains(body, "sales-customer-summary-grand-total") || strings.Contains(body, "Grand Total:") {
		t.Fatalf("body includes sales by customer summary by item grand total")
	}
	if !strings.Contains(body, "4.00") || !strings.Contains(body, "3,510.00") || !strings.Contains(body, "825.00") || !strings.Contains(body, "2,685.00") {
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
	if !strings.Contains(body, "STOCK SALES BY STOCK CODE") || !strings.Contains(body, "Sales From: Enero 01, 2026 To: Enero 31, 2026") {
		t.Fatalf("body missing sales by stock name title or coverage")
	}
	if !strings.Contains(body, "SORONGON AGRIVET") {
		t.Fatalf("body missing sales by stock name store title")
	}
	if !strings.Contains(body, "CHICKEN LINES/B-MEG") || !strings.Contains(body, "Stock Code: <strong>INT 1000</strong>") || !strings.Contains(body, `Stock Name: <strong data-report-stock-name>INTEGRA 1000</strong>`) {
		t.Fatalf("body missing category or stock group header")
	}
	if !strings.Contains(body, `data-report-always-full-tree`) || !strings.Contains(body, `data-report-highlight-target="sales-stock-1-1"`) || !strings.Contains(body, `data-report-highlight-label-only`) {
		t.Fatalf("body missing sales by stock name stock highlight behavior")
	}
	if strings.Contains(body, `data-report-filter-target="sales-stock-filter-`) {
		t.Fatalf("body includes sales by stock name stock tree filter target")
	}
	if !strings.Contains(body, `data-report-filter-groups="sales-stock-filter-1-1"`) || !strings.Contains(body, `data-report-stock-name`) {
		t.Fatalf("body missing sales by stock name stock grouping or highlight target")
	}
	if !strings.Contains(body, "CI 011497") || !strings.Contains(body, "CASH/MATARANAS") || !strings.Contains(body, "CATHALEYA AGRI-POULTRY SUPPLY") {
		t.Fatalf("body missing sales rows")
	}
	if strings.Contains(body, "<th>Type</th>") || strings.Contains(body, ">Cash</td>") || strings.Contains(body, ">Charge</td>") {
		t.Fatalf("body includes sales by stock name cash charge type")
	}
	if !strings.Contains(body, "Reference") || !strings.Contains(body, "Customer:") || !strings.Contains(body, "Price") {
		t.Fatalf("body missing sales by stock name columns")
	}
	if strings.Contains(body, "sales-stock-grand-total") || strings.Contains(body, "Grand Total:") {
		t.Fatalf("body includes sales by stock name grand total")
	}
	if !strings.Contains(body, "2,002.00") || !strings.Contains(body, "1,990.00") || !strings.Contains(body, "1,911.00") {
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
	if !strings.Contains(body, "STOCK PURCHASES BY STOCK CODE") || !strings.Contains(body, "Purchases From: Enero 01, 2026 To: Enero 31, 2026") {
		t.Fatalf("body missing purchases by stock code title or coverage")
	}
	if !strings.Contains(body, "Stock Code: <strong>NESTY 7KNDS</strong>") || !strings.Contains(body, "Stock Name: <strong>NESTY 7 KINDS</strong>") {
		t.Fatalf("body missing stock code group header")
	}
	if !strings.Contains(body, "TA #138120") || !strings.Contains(body, "01/13/2026") || !strings.Contains(body, "NESTY") {
		t.Fatalf("body missing purchase rows")
	}
	if !strings.Contains(body, `<span class="report-tree-expander" aria-hidden="true"></span>NESTY</button>`) || !strings.Contains(body, `<span class="report-tree-expander" aria-hidden="true"></span>SOUTH SEA DESIGNS,INC.</button>`) {
		t.Fatalf("body missing supplier preview tree")
	}
	if !strings.Contains(body, `>NESTY 7KNDS</button>`) || !strings.Contains(body, `>HGPP</button>`) || !strings.Contains(body, `data-report-scroll-target="purchase-stock-group-`) || !strings.Contains(body, `data-report-filter-target="purchase-stock-filter-`) {
		t.Fatalf("body missing stock code preview tree children")
	}
	if !strings.Contains(body, `data-report-filter-groups="purchase-stock-filter-`) || !strings.Contains(body, `data-report-filter-only`) || !strings.Contains(body, `data-report-filter-only data-report-filter-groups="purchase-stock-filter-`) {
		t.Fatalf("body missing stock code preview filter targets")
	}
	if strings.Contains(body, "<th>Type</th>") || strings.Contains(body, ">Cash</td>") || strings.Contains(body, ">Charge</td>") {
		t.Fatalf("body includes purchases by stock code cash charge type")
	}
	if !strings.Contains(body, "Reference") || !strings.Contains(body, "Date") || !strings.Contains(body, "Supplier") {
		t.Fatalf("body missing purchases by stock code columns")
	}
	if strings.Contains(body, "purchase-stock-grand-total") || strings.Contains(body, "Grand Total:") {
		t.Fatalf("body includes purchases by stock code grand total")
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
	if !strings.Contains(body, "STOCK PURCHASES BY SUPPLIER") || !strings.Contains(body, "Purchases From: Enero 01, 2026 To: Enero 31, 2026") {
		t.Fatalf("body missing purchases by supplier title or coverage")
	}
	if !strings.Contains(body, "Supplier: <strong>DG AGRIVET</strong>") || !strings.Contains(body, "PIGROLAC HOG GROWER PELLET VITAL") {
		t.Fatalf("body missing supplier group or stock rows")
	}
	if !strings.Contains(body, `data-report-scroll-target="purchase-supplier-row-1-1-1"`) || !strings.Contains(body, `data-report-highlight-target="purchase-supplier-stock-1-1"`) || !strings.Contains(body, `>PIGROLAC HOG GROWER PELLET VITAL</button>`) {
		t.Fatalf("body missing purchases by supplier stock preview tree")
	}
	if strings.Contains(body, `data-report-filter-target="purchase-supplier-stock-filter-1-1"`) {
		t.Fatalf("purchases by supplier stock preview should not hide other stock groups")
	}
	if strings.Contains(body, `>SI #3055</button>`) || strings.Contains(body, `>SI #3273</button>`) {
		t.Fatalf("body includes purchase reference in purchases by supplier preview tree")
	}
	if !strings.Contains(body, "Type") || !strings.Contains(body, ">Cash</td>") || !strings.Contains(body, ">Charge</td>") {
		t.Fatalf("body missing purchases by supplier cash charge type")
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
	if !strings.Contains(body, "SORONGON AGRIVET") || !strings.Contains(body, "STOCK LEDGER") || !strings.Contains(body, "Period Covered : 3/1/2026 ~ 3/31/2026") {
		t.Fatalf("body missing stock ledger title or coverage")
	}
	if !strings.Contains(body, `class="stock-ledger-category">Category Name: <strong data-report-category-name>PILMICO HOGS</strong>`) || !strings.Contains(body, "CLASSIC FINEX") {
		t.Fatalf("body missing stock ledger category or stock rows")
	}
	if !strings.Contains(body, `<span class="report-tree-expander" aria-hidden="true"></span>PILMICO HOGS</button>`) {
		t.Fatalf("body missing stock ledger category preview node")
	}
	if !strings.Contains(body, `class="report-preview-tree" data-report-always-full-tree`) {
		t.Fatalf("stock ledger preview should keep the full category and stock tree visible")
	}
	if !strings.Contains(body, `report-tree-child`) || !strings.Contains(body, `data-report-scroll-target="stock-ledger-stock-`) || !strings.Contains(body, `data-report-highlight-target="stock-ledger-stock-`) {
		t.Fatalf("stock ledger preview should render stock child nodes")
	}
	if !strings.Contains(body, `>CLASSIC FINEX, 50kg.</button>`) || !strings.Contains(body, `>EMPTY STOCK</button>`) || !strings.Contains(body, `>No Stock</button>`) {
		t.Fatalf("stock ledger preview should render stock names under categories")
	}
	if !strings.Contains(body, `data-report-category-name`) || !strings.Contains(body, `report-category-name-highlight`) {
		t.Fatalf("stock ledger should wire category-name highlighting for preview parent nodes")
	}
	if !strings.Contains(body, `data-report-stock-name`) || !strings.Contains(body, `report-stock-name-highlight`) {
		t.Fatalf("stock ledger should wire stock-name highlighting for preview child nodes")
	}
	if !strings.Contains(body, "CLASSIC GROWEX") || !strings.Contains(body, "EMPTY CATEGORY") || !strings.Contains(body, "EMPTY STOCK") {
		t.Fatalf("body should render stock ledger stocks and categories with zero balances")
	}
	if !strings.Contains(body, `class="stock-ledger-category">Category Name: <strong data-report-category-name>MASTER ONLY</strong>`) || !strings.Contains(body, `data-report-stock-name>No Stock</strong>`) {
		t.Fatalf("body should render stock ledger master categories without stocks")
	}
	if !strings.Contains(body, "<td>Forwarded</td><td>02/28/2026</td><td>Forwarded Balance</td><td class=\"num\"></td><td class=\"num\"></td><td class=\"num\">0.00</td>") {
		t.Fatalf("stock ledger zero-balance stocks should render a forwarded balance row")
	}
	if !strings.Contains(body, "Forwarded Balance") || !strings.Contains(body, "25.00") || !strings.Contains(body, "180.00") {
		t.Fatalf("body missing stock ledger forwarded or running balance values")
	}
	if !strings.Contains(body, "<td>PO-1</td><td>03/14/2026</td><td>Supplier A</td><td class=\"num\">155.00</td><td class=\"num\"></td>") {
		t.Fatalf("stock ledger purchase should render as debit")
	}
	if !strings.Contains(body, "<td>SALE-1</td><td>03/15/2026</td><td>Customer A</td><td class=\"num\"></td><td class=\"num\">20.00</td>") {
		t.Fatalf("stock ledger sale should render as credit")
	}
	if !strings.Contains(body, "<td>TR-1</td><td>03/16/2026</td><td>Branch A</td><td class=\"num\"></td><td class=\"num\">15.00</td>") {
		t.Fatalf("stock ledger stock transaction should render as credit")
	}
	purchaseIndex := strings.Index(body, "ZZ-PURCHASE")
	saleIndex := strings.Index(body, "AA-SALE")
	if purchaseIndex == -1 || saleIndex == -1 || purchaseIndex > saleIndex {
		t.Fatalf("stock ledger did not preserve same-day entry timestamp order")
	}
}
