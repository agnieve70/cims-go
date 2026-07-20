package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"cims-go/internal/config"
	appdb "cims-go/internal/db"
	"cims-go/internal/models"
	"cims-go/internal/repositories"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type legacyData struct {
	branches          []legacyBranch
	categories        []legacyCategory
	customers         []legacyCustomer
	clientAliases     []legacyClientAlias
	suppliers         []legacySupplier
	stocks            []legacyStock
	expenseCharts     []legacyExpenseChart
	otherIncomeCharts []legacyOtherIncomeChart

	checksIn       map[int]*legacyChecksIn
	otherIncome    map[int]*legacyOtherIncomeDoc
	purchases      map[int]*legacyPurchase
	sales          map[int]*legacySale
	rebates        map[int]*legacyCreditDoc
	stockIn        map[int]*legacyStockDoc
	stockOut       map[int]*legacyStockDoc
	stockTransfers map[int]*legacyStockTransfer
	apCredits      map[int]*legacyCreditDoc
	apDebits       map[int]*legacyDebitDoc
	arCredits      map[int]*legacyCreditDoc
	arDebits       map[int]*legacyDebitDoc
	expenses       map[int]*legacyExpenseDoc
}

type legacyBranch struct {
	Code     string
	Name     string
	Incharge string
	Remarks  string
	APS      string
	Farm     bool
}

type legacyCategory struct {
	Name      string
	GroupName string
	ShowInAPS bool
}

type legacyCustomer struct {
	Code        string
	Company     string
	LastName    string
	FirstName   string
	MiddleName  string
	PhoneNumber string
	Address     string
	CreditLimit string
	CreditTerm  string
	APS         string
	Farm        bool
}

type legacyClientAlias struct {
	Code        string
	Company     string
	LastName    string
	FirstName   string
	MiddleName  string
	PhoneNumber string
	Address     string
	CreditLimit string
	CreditTerm  string
	APS         string
	Farm        bool
}

type legacySupplier struct {
	Code        string
	Company     string
	LastName    string
	FirstName   string
	MiddleName  string
	PhoneNumber string
	Address     string
}

type legacyStock struct {
	Code        string
	Name        string
	Category    string
	Unit        string
	LatestCost  string
	MinimumInv  string
	Description string
}

type legacyExpenseChart struct {
	Code         string
	Name         string
	Description  string
	ExcludeSales bool
	DailyOnly    bool
}

type legacyOtherIncomeChart struct {
	Code        string
	Name        string
	Description string
}

type legacyCheck struct {
	Number   string
	Date     string
	BankName string
	Amount   string
	Nature   string
}

type legacyAdjustment struct {
	Name     string
	Price    string
	Quantity string
	Amount   string
}

type legacyStockLine struct {
	StockCode string
	StockName string
	Quantity  string
	UnitCost  string
	Amount    string
	Capital   string
	Discount  string
	OtherDisc string
	Markup    string
	MarkupPct string
}

type legacyMoneyLine struct {
	Code      string
	Name      string
	Reference string
	Cash      string
	Check     string
	Total     string
}

type legacyChecksIn struct {
	EntryID   int
	EntryDate string
	Remarks   string
	Total     string
	Checks    []legacyCheck
}

type legacyOtherIncomeDoc struct {
	EntryID   int
	EntryDate string
	Remarks   string
	Branch    string
	Total     string
	Lines     []legacyMoneyLine
}

type legacyPurchase struct {
	EntryID      int
	EntryDate    string
	ORNumber     string
	CINumber     string
	Cash         bool
	PurchaseDate string
	Supplier     string
	GrossTotal   string
	NetTotal     string
	TotalDisc    string
	TotalAdd     string
	CashAmount   string
	CheckAmount  string
	TotalQty     string
	Remarks      string
	Details      []legacyStockLine
	Discounts    []legacyAdjustment
	Additionals  []legacyAdjustment
	Checks       []legacyCheck
}

type legacySale struct {
	EntryID        int
	EntryDate      string
	ORNumber       string
	CINumber       string
	Cash           bool
	SalesDate      string
	Customer       string
	GrossTotal     string
	NetTotal       string
	TotalDisc      string
	ManualDiscount string
	TotalAdd       string
	CashAmount     string
	CheckAmount    string
	TotalQty       string
	TotalNetAmount string
	Remarks        string
	Details        []legacyStockLine
	Discounts      []legacyAdjustment
	Additionals    []legacyAdjustment
	Checks         []legacyCheck
}

type legacyStockDoc struct {
	EntryID   int
	EntryDate string
	Remarks   string
	Total     string
	TotalQty  string
	Details   []legacyStockLine
}

type legacyStockTransfer struct {
	EntryID      int
	EntryDate    string
	TransferID   string
	Remarks      string
	TransferDate string
	GrossTotal   string
	NetTotal     string
	TotalDisc    string
	TotalAdd     string
	Transaction  string
	Cash         bool
	Customer     string
	Supplier     string
	Branch       string
	Bodega       string
	TotalQty     string
	Details      []legacyStockLine
	Discounts    []legacyAdjustment
	Additionals  []legacyAdjustment
}

type legacyCreditDoc struct {
	EntryID        int
	EntryDate      string
	Reference      string
	Company        string
	Amount         string
	CurrentBalance string
	CashAmount     string
	CheckAmount    string
	Remarks        string
	Checks         []legacyCheck
}

type legacyDebitDoc struct {
	EntryID   int
	EntryDate string
	Company   string
	Amount    string
	Remarks   string
}

type legacyExpenseDoc struct {
	EntryID   int
	EntryDate string
	Remarks   string
	Reference string
	Total     string
	Lines     []legacyMoneyLine
	Checks    []legacyCheck
}

type importState struct {
	ctx               context.Context
	tx                pgx.Tx
	user              models.User
	branchByName      map[string]int64
	branchByCode      map[string]int64
	customerByName    map[string]int64
	supplierByName    map[string]int64
	stockByCode       map[string]int64
	expenseByCode     map[string]int64
	expenseByName     map[string]int64
	otherIncomeByCode map[string]int64
	otherIncomeByName map[string]int64
	categoryKeys      map[string]struct{}
	usedCustomerCodes map[string]struct{}
	usedSupplierCodes map[string]struct{}
}

type storedPayload struct {
	Kind      string                   `json:"kind"`
	Values    map[string]string        `json:"values"`
	LineInput []repositories.LineInput `json:"line_input"`
	EncoderID int64                    `json:"encoder_id"`
}

type importedLine struct {
	group  string
	row    map[string]string
	stock  *int64
	code   *int64
	qty    string
	cost   string
	price  string
	cash   string
	check  string
	amount string
}

type importedDocument struct {
	kind         string
	entryDate    time.Time
	documentDate time.Time
	branchID     *int64
	partyType    *string
	partyID      *int64
	reference    string
	cash         bool
	remarks      string
	total        string
	less         string
	add          string
	net          string
	balance      string
	values       map[string]string
	lineInput    []repositories.LineInput
	lines        []importedLine
	inventory    []legacyInventoryEffect
	balanceDelta *legacyBalanceEffect
}

type legacyInventoryEffect struct {
	BranchID *int64
	StockID  int64
	QtyDelta string
	UnitCost string
}

type legacyBalanceEffect struct {
	PartyType string
	PartyID   int64
	Amount    string
}

func main() {
	var dumpPath string
	var schema string

	flag.StringVar(&dumpPath, "dump", "", "path to the legacy SQL dump")
	flag.StringVar(&schema, "schema", "cims", "legacy MySQL schema name to import")
	flag.Parse()

	if strings.TrimSpace(dumpPath) == "" {
		log.Fatal("dump path is required")
	}

	ctx := context.Background()
	cfg, err := config.Load()
	must(err)
	must(appdb.Migrate(cfg.DatabaseURL, "db/migrations"))

	data, err := parseLegacyDump(dumpPath, schema)
	must(err)

	pool, err := appdb.OpenPool(ctx, cfg.DatabaseURL, cfg.DBMaxConns, cfg.DBMinConns)
	must(err)
	defer pool.Close()

	store := repositories.NewPostgresStore(pool)
	must(store.EnsureAdmin(ctx, cfg.AdminUsername, cfg.AdminPassword))
	user, err := store.GetUserByUsername(ctx, cfg.AdminUsername)
	must(err)

	tx, err := pool.Begin(ctx)
	must(err)
	defer tx.Rollback(ctx)

	must(truncateBusinessData(ctx, tx))
	state := &importState{
		ctx:               ctx,
		tx:                tx,
		user:              user,
		branchByName:      map[string]int64{},
		branchByCode:      map[string]int64{},
		customerByName:    map[string]int64{},
		supplierByName:    map[string]int64{},
		stockByCode:       map[string]int64{},
		expenseByCode:     map[string]int64{},
		expenseByName:     map[string]int64{},
		otherIncomeByCode: map[string]int64{},
		otherIncomeByName: map[string]int64{},
		categoryKeys:      map[string]struct{}{},
		usedCustomerCodes: map[string]struct{}{},
		usedSupplierCodes: map[string]struct{}{},
	}

	must(state.importMasters(data))
	must(state.importTransactions(data))
	must(state.recomputePartyBalances())
	must(state.assignActiveBranch())

	must(tx.Commit(ctx))
	must(printSummary(ctx, pool))
}

func parseLegacyDump(path, schema string) (*legacyData, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open dump: %w", err)
	}
	defer file.Close()

	data := &legacyData{
		checksIn:       map[int]*legacyChecksIn{},
		otherIncome:    map[int]*legacyOtherIncomeDoc{},
		purchases:      map[int]*legacyPurchase{},
		sales:          map[int]*legacySale{},
		rebates:        map[int]*legacyCreditDoc{},
		stockIn:        map[int]*legacyStockDoc{},
		stockOut:       map[int]*legacyStockDoc{},
		stockTransfers: map[int]*legacyStockTransfer{},
		apCredits:      map[int]*legacyCreditDoc{},
		apDebits:       map[int]*legacyDebitDoc{},
		arCredits:      map[int]*legacyCreditDoc{},
		arDebits:       map[int]*legacyDebitDoc{},
		expenses:       map[int]*legacyExpenseDoc{},
	}

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024*1024), 32*1024*1024)

	useStmt := "USE " + schema + ";"
	inSchema := false
	schemaFound := false
	var builder strings.Builder
	currentTable := ""

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if !inSchema {
			if trimmed == useStmt {
				inSchema = true
				schemaFound = true
			}
			continue
		}

		if strings.HasPrefix(trimmed, "USE ") {
			if currentTable != "" {
				if err := applyInsert(data, currentTable, builder.String()); err != nil {
					return nil, err
				}
				builder.Reset()
				currentTable = ""
			}
			inSchema = trimmed == useStmt
			if inSchema {
				schemaFound = true
			}
			continue
		}

		if currentTable != "" {
			builder.WriteString(line)
			builder.WriteByte('\n')
			if strings.HasSuffix(trimmed, ";") {
				if err := applyInsert(data, currentTable, builder.String()); err != nil {
					return nil, err
				}
				builder.Reset()
				currentTable = ""
			}
			continue
		}

		if !strings.HasPrefix(trimmed, "INSERT INTO `") {
			continue
		}
		table, ok := parseInsertTable(trimmed)
		if !ok || !wantedTable(table) {
			continue
		}
		currentTable = table
		builder.WriteString(line)
		builder.WriteByte('\n')
		if strings.HasSuffix(trimmed, ";") {
			if err := applyInsert(data, currentTable, builder.String()); err != nil {
				return nil, err
			}
			builder.Reset()
			currentTable = ""
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan dump: %w", err)
	}
	if !schemaFound {
		return nil, fmt.Errorf("legacy schema %q was not found in the dump; no data was imported", schema)
	}
	return data, nil
}

func wantedTable(table string) bool {
	switch table {
	case "c_branch", "branches", "categories", "c_clients", "clientsdata", "supplierdata", "stocks",
		"expenseschartdata", "otherincome",
		"checksindata", "checksindatadetails",
		"otherincomedata", "otherincomedatadetails",
		"purchasedata", "purchasedatadetails", "purchasedatadiscounts", "purchasedataadditionals", "purchasedatachecks",
		"salesdata", "salesdatadetails", "salesdatadiscounts", "salesdataadditionals", "salesdatachecks",
		"rebatesdata", "rebatesdatachecks",
		"stockindata", "stockindatadetails",
		"stockoutdata", "stockoutdatadetails",
		"stocktransferdata", "stocktransferdatadetails", "stocktransferdatadiscounts", "stocktransferdataadditionals",
		"apcreditdata", "apcreditdatachecks", "apdebitdata",
		"arcreditdata", "arcreditdatachecks", "ardebitdata",
		"expensesdata", "expensesdatadetails", "outgoingchecks":
		return true
	default:
		return false
	}
}

func parseInsertTable(line string) (string, bool) {
	start := strings.Index(line, "`")
	if start == -1 {
		return "", false
	}
	end := strings.Index(line[start+1:], "`")
	if end == -1 {
		return "", false
	}
	return line[start+1 : start+1+end], true
}

func applyInsert(data *legacyData, table, stmt string) error {
	columns, rows, err := parseInsertStatement(stmt)
	if err != nil {
		return fmt.Errorf("parse %s insert: %w", table, err)
	}
	for _, values := range rows {
		row := mapRow(columns, values)
		switch table {
		case "c_branch":
			data.branches = append(data.branches, legacyBranch{
				Code: row["branchcode"],
				Name: row["branchname"],
			})
		case "branches":
			data.branches = append(data.branches, legacyBranch{
				Code:     row["BranchCode"],
				Name:     row["BranchName"],
				Incharge: row["Incharge"],
				Remarks:  row["Remarks"],
				APS:      row["APS"],
				Farm:     parseBoolString(row["Farm"]),
			})
		case "categories":
			data.categories = append(data.categories, legacyCategory{
				Name:      row["Category"],
				GroupName: row["CatGroup"],
				ShowInAPS: parseBoolString(row["ShowInAPS"]),
			})
		case "c_clients":
			data.customers = append(data.customers, legacyCustomer{
				Code:        row["idnum"],
				Company:     joinName(row["lname"], row["fname"], row["mname"]),
				LastName:    row["lname"],
				FirstName:   row["fname"],
				MiddleName:  row["mname"],
				PhoneNumber: row["contactno"],
				Address:     choose(row["recentaddress"], row["address"]),
				CreditLimit: chooseNumeric(row["creditlimit"], row["creditlimit1"]),
				CreditTerm:  "",
				APS:         "",
				Farm:        false,
			})
		case "clientsdata":
			data.clientAliases = append(data.clientAliases, legacyClientAlias{
				Code:        row["ClientsCode"],
				Company:     choose(row["Company"], joinName(row["LastName"], row["FirstName"], row["MiddleName"])),
				LastName:    row["LastName"],
				FirstName:   row["FirstName"],
				MiddleName:  row["MiddleName"],
				PhoneNumber: row["PhoneNumber"],
				Address:     row["Address"],
				CreditLimit: row["CreditLimit"],
				CreditTerm:  row["CreditTerm"],
				APS:         row["APS"],
				Farm:        parseBoolString(row["Farm"]),
			})
		case "supplierdata":
			data.suppliers = append(data.suppliers, legacySupplier{
				Code:        row["SuppCode"],
				Company:     choose(row["Company"], joinName(row["LastName"], row["FirstName"], row["MiddleName"])),
				LastName:    row["LastName"],
				FirstName:   row["FirstName"],
				MiddleName:  row["MiddleName"],
				PhoneNumber: row["PhoneNumber"],
				Address:     row["Address"],
			})
		case "stocks":
			data.stocks = append(data.stocks, legacyStock{
				Code:        row["StockCode"],
				Name:        row["StockName"],
				Category:    row["Category"],
				Unit:        row["Unit"],
				LatestCost:  row["LatestCost"],
				MinimumInv:  row["MinimumInv"],
				Description: row["Description"],
			})
		case "expenseschartdata":
			data.expenseCharts = append(data.expenseCharts, legacyExpenseChart{
				Code:         row["ExpCode"],
				Name:         row["ExpName"],
				Description:  row["ExpDesc"],
				ExcludeSales: parseBoolString(row["notinsales"]),
				DailyOnly:    parseBoolString(row["fordscronly"]),
			})
		case "otherincome":
			data.otherIncomeCharts = append(data.otherIncomeCharts, legacyOtherIncomeChart{
				Code:        row["OICode"],
				Name:        row["OIName"],
				Description: row["Description"],
			})
		case "checksindata":
			entryID := parseInt(row["EntryID"])
			data.checksIn[entryID] = &legacyChecksIn{
				EntryID:   entryID,
				EntryDate: row["EntryDate"],
				Remarks:   row["Remarks"],
				Total:     row["TotalAmount"],
			}
		case "checksindatadetails":
			entryID := parseInt(row["EntryID"])
			doc := ensureChecksIn(data, entryID)
			doc.Checks = append(doc.Checks, legacyCheck{
				Number:   row["CheckNumber"],
				Date:     row["CheckDate"],
				BankName: row["BankName"],
				Amount:   row["Amount"],
				Nature:   row["Nature"],
			})
		case "otherincomedata":
			entryID := parseInt(row["EntryID"])
			data.otherIncome[entryID] = &legacyOtherIncomeDoc{
				EntryID:   entryID,
				EntryDate: row["TranDate"],
				Remarks:   row["Remarks"],
				Branch:    row["Branch"],
				Total:     row["TotalAmount"],
			}
		case "otherincomedatadetails":
			entryID := parseInt(row["EntryID"])
			doc := ensureOtherIncome(data, entryID)
			doc.Lines = append(doc.Lines, legacyMoneyLine{
				Code:      row["OICode"],
				Name:      row["OIName"],
				Reference: row["Reference"],
				Cash:      row["Cash"],
				Check:     row["Check"],
				Total:     row["TotalAmount"],
			})
		case "purchasedata":
			entryID := parseInt(row["EntryID"])
			data.purchases[entryID] = &legacyPurchase{
				EntryID:      entryID,
				EntryDate:    row["EntryDate"],
				ORNumber:     row["ORNumber"],
				CINumber:     row["CINumber"],
				Cash:         parseBoolString(row["Cash"]),
				PurchaseDate: row["PurchaseDate"],
				Supplier:     row["Supplier"],
				GrossTotal:   row["GrossTotal"],
				NetTotal:     row["NetTotal"],
				TotalDisc:    row["TotalDiscount"],
				TotalAdd:     row["TotalAdditional"],
				CashAmount:   row["CashAmount"],
				CheckAmount:  row["CheckAmount"],
				TotalQty:     row["TotalQty"],
			}
		case "purchasedatadetails":
			doc := ensurePurchase(data, parseInt(row["EntryID"]))
			doc.Details = append(doc.Details, legacyStockLine{
				StockCode: row["StockCode"],
				StockName: row["StockName"],
				Quantity:  row["Quantity"],
				UnitCost:  row["UnitCost"],
				Amount:    row["Amount"],
			})
		case "purchasedatadiscounts":
			doc := ensurePurchase(data, parseInt(row["EntryID"]))
			doc.Discounts = append(doc.Discounts, legacyAdjustment{
				Name:     row["DiscName"],
				Price:    row["Price"],
				Quantity: row["Quantity"],
				Amount:   row["DiscAmount"],
			})
		case "purchasedataadditionals":
			doc := ensurePurchase(data, parseInt(row["EntryID"]))
			doc.Additionals = append(doc.Additionals, legacyAdjustment{
				Name:     row["AddName"],
				Price:    row["Price"],
				Quantity: row["Quantity"],
				Amount:   row["AddAmount"],
			})
		case "purchasedatachecks":
			doc := ensurePurchase(data, parseInt(row["EntryID"]))
			doc.Checks = append(doc.Checks, legacyCheck{
				Number:   row["CheckNumber"],
				Date:     row["CheckDate"],
				BankName: row["BankName"],
				Amount:   row["Amount"],
			})
		case "salesdata":
			entryID := parseInt(row["EntryID"])
			data.sales[entryID] = &legacySale{
				EntryID:        entryID,
				EntryDate:      row["EntryDate"],
				ORNumber:       row["ORNumber"],
				CINumber:       row["CINumber"],
				Cash:           parseBoolString(row["Cash"]),
				SalesDate:      row["SalesDate"],
				Customer:       row["Customer"],
				GrossTotal:     row["GrossTotal"],
				NetTotal:       row["NetTotal"],
				TotalDisc:      row["TotalDiscount"],
				ManualDiscount: row["ManualDiscount"],
				TotalAdd:       row["TotalAdditional"],
				CashAmount:     row["CashAmount"],
				CheckAmount:    row["CheckAmount"],
				TotalQty:       row["TotalQty"],
				TotalNetAmount: row["TotalNetAmount"],
			}
		case "salesdatadetails":
			doc := ensureSale(data, parseInt(row["EntryID"]))
			doc.Details = append(doc.Details, legacyStockLine{
				StockCode: row["StockCode"],
				StockName: row["StockName"],
				Quantity:  row["Quantity"],
				UnitCost:  row["UnitCost"],
				Amount:    row["NetAmount"],
				Capital:   row["unitcapital"],
				Discount:  row["OD"],
				OtherDisc: row["LaborDisc"],
				Markup:    row["markupamnt"],
				MarkupPct: row["markupprcnt"],
			})
		case "salesdatadiscounts":
			doc := ensureSale(data, parseInt(row["EntryID"]))
			doc.Discounts = append(doc.Discounts, legacyAdjustment{
				Name:     row["DiscName"],
				Price:    row["Price"],
				Quantity: row["Quantity"],
				Amount:   row["DiscAmount"],
			})
		case "salesdataadditionals":
			doc := ensureSale(data, parseInt(row["EntryID"]))
			doc.Additionals = append(doc.Additionals, legacyAdjustment{
				Name:     row["AddName"],
				Price:    row["Price"],
				Quantity: row["Quantity"],
				Amount:   row["AddAmount"],
			})
		case "salesdatachecks":
			doc := ensureSale(data, parseInt(row["EntryID"]))
			doc.Checks = append(doc.Checks, legacyCheck{
				Number:   row["CheckNumber"],
				Date:     row["CheckDate"],
				BankName: row["BankName"],
				Amount:   row["Amount"],
			})
		case "rebatesdata":
			entryID := parseInt(row["EntryID"])
			data.rebates[entryID] = &legacyCreditDoc{
				EntryID:        entryID,
				EntryDate:      row["TranDate"],
				Reference:      row["Reference"],
				Company:        row["Company"],
				Amount:         row["Amount"],
				CurrentBalance: row["CurrentBalance"],
				CashAmount:     row["CashAmount"],
				CheckAmount:    row["CheckAmount"],
			}
		case "rebatesdatachecks":
			doc := ensureCreditDoc(data.rebates, parseInt(row["EntryID"]))
			doc.Checks = append(doc.Checks, legacyCheck{
				Number:   row["CheckNumber"],
				Date:     row["CheckDate"],
				BankName: row["BankName"],
				Amount:   row["Amount"],
			})
		case "stockindata":
			entryID := parseInt(row["EntryID"])
			data.stockIn[entryID] = &legacyStockDoc{
				EntryID:   entryID,
				EntryDate: row["EntryDate"],
				Remarks:   row["Remarks"],
				Total:     row["TotalAmount"],
				TotalQty:  row["TotalQty"],
			}
		case "stockindatadetails":
			doc := ensureStockDoc(data.stockIn, parseInt(row["EntryID"]))
			doc.Details = append(doc.Details, legacyStockLine{
				StockCode: row["StockCode"],
				StockName: row["StockName"],
				Quantity:  row["Quantity"],
				UnitCost:  row["UnitCost"],
				Amount:    row["Amount"],
			})
		case "stockoutdata":
			entryID := parseInt(row["EntryID"])
			data.stockOut[entryID] = &legacyStockDoc{
				EntryID:   entryID,
				EntryDate: row["EntryDate"],
				Remarks:   row["Remarks"],
				Total:     row["TotalAmount"],
				TotalQty:  row["TotalQty"],
			}
		case "stockoutdatadetails":
			doc := ensureStockDoc(data.stockOut, parseInt(row["EntryID"]))
			doc.Details = append(doc.Details, legacyStockLine{
				StockCode: row["StockCode"],
				StockName: row["StockName"],
				Quantity:  row["Quantity"],
				UnitCost:  row["UnitCost"],
				Amount:    row["Amount"],
			})
		case "stocktransferdata":
			entryID := parseInt(row["EntryID"])
			data.stockTransfers[entryID] = &legacyStockTransfer{
				EntryID:      entryID,
				EntryDate:    row["EntryDate"],
				TransferID:   row["TransferID"],
				Remarks:      row["Remarks"],
				TransferDate: row["TransferDate"],
				GrossTotal:   row["GrossTotal"],
				NetTotal:     row["NetTotal"],
				TotalDisc:    row["TotalDiscount"],
				TotalAdd:     row["TotalAdditional"],
				Transaction:  row["TransactionType"],
				Cash:         parseBoolString(row["Cash"]),
				Customer:     row["Customer"],
				Supplier:     row["Supplier"],
				Branch:       row["Branch"],
				Bodega:       row["Bodega"],
				TotalQty:     row["TotalQty"],
			}
		case "stocktransferdatadetails":
			doc := ensureStockTransfer(data, parseInt(row["EntryID"]))
			doc.Details = append(doc.Details, legacyStockLine{
				StockCode: row["StockCode"],
				StockName: row["StockName"],
				Quantity:  row["Quantity"],
				UnitCost:  row["UnitCost"],
				Amount:    row["Amount"],
				Capital:   row["unitcapital"],
				Markup:    row["markupamnt"],
				MarkupPct: row["markupprcnt"],
			})
		case "stocktransferdatadiscounts":
			doc := ensureStockTransfer(data, parseInt(row["EntryID"]))
			doc.Discounts = append(doc.Discounts, legacyAdjustment{
				Name:     row["DiscName"],
				Price:    row["Price"],
				Quantity: row["Quantity"],
				Amount:   row["DiscAmount"],
			})
		case "stocktransferdataadditionals":
			doc := ensureStockTransfer(data, parseInt(row["EntryID"]))
			doc.Additionals = append(doc.Additionals, legacyAdjustment{
				Name:     row["AddName"],
				Price:    row["Price"],
				Quantity: row["Quantity"],
				Amount:   row["AddAmount"],
			})
		case "apcreditdata":
			entryID := parseInt(row["EntryID"])
			data.apCredits[entryID] = &legacyCreditDoc{
				EntryID:        entryID,
				EntryDate:      row["TranDate"],
				Reference:      row["Reference"],
				Company:        row["Company"],
				Amount:         row["Amount"],
				CurrentBalance: row["CurrentBalance"],
				CashAmount:     row["CashAmount"],
				CheckAmount:    row["CheckAmount"],
			}
		case "apcreditdatachecks":
			doc := ensureCreditDoc(data.apCredits, parseInt(row["EntryID"]))
			doc.Checks = append(doc.Checks, legacyCheck{
				Number:   row["CheckNumber"],
				Date:     row["CheckDate"],
				BankName: row["BankName"],
				Amount:   row["Amount"],
			})
		case "apdebitdata":
			entryID := parseInt(row["EntryID"])
			data.apDebits[entryID] = &legacyDebitDoc{
				EntryID:   entryID,
				EntryDate: row["EntryDate"],
				Company:   row["Company"],
				Amount:    row["Amount"],
				Remarks:   row["Remarks"],
			}
		case "arcreditdata":
			entryID := parseInt(row["EntryID"])
			data.arCredits[entryID] = &legacyCreditDoc{
				EntryID:        entryID,
				EntryDate:      row["TranDate"],
				Reference:      row["Reference"],
				Company:        row["Company"],
				Amount:         row["Amount"],
				CurrentBalance: row["CurrentBalance"],
				CashAmount:     row["CashAmount"],
				CheckAmount:    row["CheckAmount"],
			}
		case "arcreditdatachecks":
			doc := ensureCreditDoc(data.arCredits, parseInt(row["EntryID"]))
			doc.Checks = append(doc.Checks, legacyCheck{
				Number:   row["CheckNumber"],
				Date:     row["CheckDate"],
				BankName: row["BankName"],
				Amount:   row["Amount"],
			})
		case "ardebitdata":
			entryID := parseInt(row["EntryID"])
			data.arDebits[entryID] = &legacyDebitDoc{
				EntryID:   entryID,
				EntryDate: row["EntryDate"],
				Company:   row["Company"],
				Amount:    row["Amount"],
				Remarks:   row["Remarks"],
			}
		case "expensesdata":
			entryID := parseInt(row["EntryID"])
			data.expenses[entryID] = &legacyExpenseDoc{
				EntryID:   entryID,
				EntryDate: choose(row["TranDate"], row["EntryDate"]),
				Remarks:   row["Remarks"],
				Reference: row["Reference"],
				Total:     row["TotalAmount"],
			}
		case "expensesdatadetails":
			doc := ensureExpense(data, parseInt(row["EntryID"]))
			doc.Lines = append(doc.Lines, legacyMoneyLine{
				Code:      row["ExpCode"],
				Name:      row["ExpName"],
				Reference: choose(row["Reference"], row["Particulars"]),
				Cash:      row["Cash"],
				Check:     row["Check"],
				Total:     choose(row["TotalAmount"], row["Amount"]),
			})
		case "outgoingchecks":
			if parseInt(row["SourceIndex"]) != 1 {
				break
			}
			doc := ensureExpense(data, parseInt(row["EntryID"]))
			doc.Checks = append(doc.Checks, legacyCheck{
				Number:   row["CheckNumber"],
				Date:     row["CheckDate"],
				BankName: row["BankName"],
				Amount:   row["Amount"],
				Nature:   "1 - Outgoing Check",
			})
		}
	}
	return nil
}

func ensureChecksIn(data *legacyData, entryID int) *legacyChecksIn {
	if doc, ok := data.checksIn[entryID]; ok {
		return doc
	}
	doc := &legacyChecksIn{EntryID: entryID}
	data.checksIn[entryID] = doc
	return doc
}

func ensureOtherIncome(data *legacyData, entryID int) *legacyOtherIncomeDoc {
	if doc, ok := data.otherIncome[entryID]; ok {
		return doc
	}
	doc := &legacyOtherIncomeDoc{EntryID: entryID}
	data.otherIncome[entryID] = doc
	return doc
}

func ensurePurchase(data *legacyData, entryID int) *legacyPurchase {
	if doc, ok := data.purchases[entryID]; ok {
		return doc
	}
	doc := &legacyPurchase{EntryID: entryID}
	data.purchases[entryID] = doc
	return doc
}

func ensureSale(data *legacyData, entryID int) *legacySale {
	if doc, ok := data.sales[entryID]; ok {
		return doc
	}
	doc := &legacySale{EntryID: entryID}
	data.sales[entryID] = doc
	return doc
}

func ensureStockDoc(docs map[int]*legacyStockDoc, entryID int) *legacyStockDoc {
	if doc, ok := docs[entryID]; ok {
		return doc
	}
	doc := &legacyStockDoc{EntryID: entryID}
	docs[entryID] = doc
	return doc
}

func ensureStockTransfer(data *legacyData, entryID int) *legacyStockTransfer {
	if doc, ok := data.stockTransfers[entryID]; ok {
		return doc
	}
	doc := &legacyStockTransfer{EntryID: entryID}
	data.stockTransfers[entryID] = doc
	return doc
}

func ensureCreditDoc(docs map[int]*legacyCreditDoc, entryID int) *legacyCreditDoc {
	if doc, ok := docs[entryID]; ok {
		return doc
	}
	doc := &legacyCreditDoc{EntryID: entryID}
	docs[entryID] = doc
	return doc
}

func ensureExpense(data *legacyData, entryID int) *legacyExpenseDoc {
	if doc, ok := data.expenses[entryID]; ok {
		return doc
	}
	doc := &legacyExpenseDoc{EntryID: entryID}
	data.expenses[entryID] = doc
	return doc
}

func parseInsertStatement(stmt string) ([]string, [][]string, error) {
	open := strings.Index(stmt, "(")
	close := strings.Index(stmt, ") VALUES")
	if open == -1 || close == -1 || close <= open {
		return nil, nil, errors.New("malformed insert statement")
	}
	columns := splitColumns(stmt[open+1 : close])
	valuesIdx := strings.Index(stmt, "VALUES")
	if valuesIdx == -1 {
		return nil, nil, errors.New("missing VALUES")
	}
	rows, err := parseValuesSection(stmt[valuesIdx+len("VALUES"):])
	if err != nil {
		return nil, nil, err
	}
	return columns, rows, nil
}

func splitColumns(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		part = strings.Trim(part, "`")
		out = append(out, part)
	}
	return out
}

func parseValuesSection(raw string) ([][]string, error) {
	var rows [][]string
	i := 0
	for i < len(raw) {
		switch raw[i] {
		case ' ', '\t', '\n', '\r', ',':
			i++
			continue
		case ';':
			return rows, nil
		case '(':
			row, next, err := parseTuple(raw, i)
			if err != nil {
				return nil, err
			}
			rows = append(rows, row)
			i = next
		default:
			return nil, fmt.Errorf("unexpected character %q", raw[i])
		}
	}
	return rows, nil
}

func parseTuple(raw string, start int) ([]string, int, error) {
	var row []string
	i := start + 1
	for {
		value, next, done, err := parseValue(raw, i)
		if err != nil {
			return nil, 0, err
		}
		row = append(row, value)
		i = next
		if done {
			return row, i, nil
		}
	}
}

func parseValue(raw string, start int) (string, int, bool, error) {
	i := start
	for i < len(raw) && (raw[i] == ' ' || raw[i] == '\n' || raw[i] == '\r' || raw[i] == '\t') {
		i++
	}
	if i >= len(raw) {
		return "", 0, false, errors.New("unexpected end of value")
	}

	if raw[i] == '\'' {
		var b strings.Builder
		i++
		for i < len(raw) {
			switch raw[i] {
			case '\\':
				if i+1 >= len(raw) {
					return "", 0, false, errors.New("unfinished escape")
				}
				b.WriteByte(raw[i+1])
				i += 2
			case '\'':
				i++
				goto doneString
			default:
				b.WriteByte(raw[i])
				i++
			}
		}
		return "", 0, false, errors.New("unterminated string")
	doneString:
		for i < len(raw) && (raw[i] == ' ' || raw[i] == '\n' || raw[i] == '\r' || raw[i] == '\t') {
			i++
		}
		if i >= len(raw) {
			return b.String(), i, true, nil
		}
		switch raw[i] {
		case ',':
			return b.String(), i + 1, false, nil
		case ')':
			return b.String(), i + 1, true, nil
		default:
			return "", 0, false, fmt.Errorf("unexpected delimiter %q after string", raw[i])
		}
	}

	begin := i
	for i < len(raw) && raw[i] != ',' && raw[i] != ')' {
		i++
	}
	value := strings.TrimSpace(raw[begin:i])
	if strings.EqualFold(value, "NULL") {
		value = ""
	}
	if i >= len(raw) {
		return value, i, true, nil
	}
	if raw[i] == ',' {
		return value, i + 1, false, nil
	}
	return value, i + 1, true, nil
}

func mapRow(columns, values []string) map[string]string {
	row := make(map[string]string, len(columns))
	for i, col := range columns {
		if i < len(values) {
			row[col] = values[i]
		} else {
			row[col] = ""
		}
	}
	return row
}

func (s *importState) importMasters(data *legacyData) error {
	extraBranches := map[string]struct{}{}
	for _, doc := range data.otherIncome {
		if name := strings.TrimSpace(doc.Branch); name != "" {
			extraBranches[name] = struct{}{}
		}
	}
	for _, doc := range data.stockTransfers {
		if name := strings.TrimSpace(doc.Branch); name != "" {
			extraBranches[name] = struct{}{}
		}
		if name := strings.TrimSpace(doc.Bodega); name != "" {
			extraBranches[name] = struct{}{}
		}
	}

	for _, branch := range data.branches {
		if _, err := s.addBranch(branch.Code, branch.Name, branch.Incharge, branch.APS, branch.Remarks, branch.Farm); err != nil {
			return err
		}
	}
	for name := range extraBranches {
		if _, err := s.addBranch("", name, "", "", "", false); err != nil {
			return err
		}
	}

	for _, category := range data.categories {
		if err := s.addCategory(category.Name, category.GroupName, category.ShowInAPS); err != nil {
			return err
		}
	}
	for _, stock := range data.stocks {
		if err := s.addCategory(stock.Category, guessCategoryGroup(stock.Category), false); err != nil {
			return err
		}
	}

	for _, chart := range data.expenseCharts {
		if _, err := s.addExpenseChart(chart); err != nil {
			return err
		}
	}
	for _, chart := range data.otherIncomeCharts {
		if _, err := s.addOtherIncomeChart(chart); err != nil {
			return err
		}
	}

	for _, supplier := range data.suppliers {
		if _, err := s.addSupplier(supplier.Code, supplier.Company, supplier.LastName, supplier.FirstName, supplier.MiddleName, supplier.PhoneNumber, supplier.Address); err != nil {
			return err
		}
	}

	for _, customer := range data.customers {
		if _, err := s.addCustomer(customer.Code, customer.Company, customer.LastName, customer.FirstName, customer.MiddleName, customer.PhoneNumber, customer.Address, customer.CreditTerm, customer.CreditLimit, customer.APS, customer.Farm); err != nil {
			return err
		}
	}
	for _, customer := range data.clientAliases {
		if _, err := s.addCustomer(customer.Code, customer.Company, customer.LastName, customer.FirstName, customer.MiddleName, customer.PhoneNumber, customer.Address, customer.CreditTerm, customer.CreditLimit, customer.APS, customer.Farm); err != nil {
			return err
		}
	}

	for _, stock := range data.stocks {
		if _, err := s.addStock(stock); err != nil {
			return err
		}
	}

	return nil
}

func (s *importState) importTransactions(data *legacyData) error {
	if err := s.importChecksIn(data.checksIn); err != nil {
		return err
	}
	if err := s.importOtherIncome(data.otherIncome); err != nil {
		return err
	}
	if err := s.importPurchases(data.purchases); err != nil {
		return err
	}
	if err := s.importStockInOut("stock-in", data.stockIn, 1); err != nil {
		return err
	}
	if err := s.importStockInOut("stock-out", data.stockOut, -1); err != nil {
		return err
	}
	if err := s.importSales(data.sales); err != nil {
		return err
	}
	if err := s.importStockTransfers(data.stockTransfers); err != nil {
		return err
	}
	if err := s.importCredits("ap-credit", "supplier", data.apCredits, true); err != nil {
		return err
	}
	if err := s.importDebits("ap-debit", "supplier", data.apDebits); err != nil {
		return err
	}
	if err := s.importCredits("ar-credit", "customer", data.arCredits, true); err != nil {
		return err
	}
	if err := s.importDebits("ar-debit", "customer", data.arDebits); err != nil {
		return err
	}
	if err := s.importCredits("rebates", "customer", data.rebates, false); err != nil {
		return err
	}
	if err := s.importExpenses(data.expenses); err != nil {
		return err
	}
	return nil
}

func (s *importState) addBranch(code, name, incharge, aps, remarks string, farm bool) (int64, error) {
	code = strings.TrimSpace(code)
	name = strings.TrimSpace(name)
	if name == "" {
		name = code
	}
	if name == "" {
		return 0, errors.New("branch name is required")
	}
	nameKey := normalizeKey(name)
	if id, ok := s.branchByName[nameKey]; ok {
		if code != "" {
			s.branchByCode[normalizeKey(code)] = id
		}
		return id, nil
	}

	if code == "" {
		code = s.uniqueCode("BR", len(s.branchByName)+1, s.branchByCode)
	}
	codeKey := normalizeKey(code)
	if id, ok := s.branchByCode[codeKey]; ok {
		s.branchByName[nameKey] = id
		return id, nil
	}

	var id int64
	err := s.tx.QueryRow(s.ctx, `
		insert into branches (code, name, incharge, aps, farm_customer, remarks, encoder_user_id, last_update_by_user_id)
		values ($1,$2,$3,$4,$5,$6,$7,$7)
		returning id`,
		code, name, emptyString(incharge), emptyString(aps), farm, emptyString(remarks), s.user.ID,
	).Scan(&id)
	if err != nil {
		return 0, err
	}
	s.branchByName[nameKey] = id
	s.branchByCode[codeKey] = id
	return id, nil
}

func (s *importState) addCategory(name, group string, aps bool) error {
	name = strings.TrimSpace(name)
	group = strings.TrimSpace(group)
	if name == "" {
		return nil
	}
	if group == "" {
		group = name
	}
	key := normalizeKey(name) + "|" + normalizeKey(group)
	if _, ok := s.categoryKeys[key]; ok {
		return nil
	}
	_, err := s.tx.Exec(s.ctx, `
		insert into stock_categories (name, group_name, aps_monitor, encoder_user_id, last_update_by_user_id)
		values ($1,$2,$3,$4,$4)
		on conflict (name, group_name) do nothing`,
		name, group, aps, s.user.ID,
	)
	if err != nil {
		return err
	}
	s.categoryKeys[key] = struct{}{}
	return nil
}

func (s *importState) addSupplier(code, company, last, first, middle, phone, address string) (int64, error) {
	company = choose(company, joinName(last, first, middle))
	company = strings.TrimSpace(company)
	if company == "" {
		company = "Unnamed Supplier"
	}
	nameKey := normalizeKey(company)
	if id, ok := s.supplierByName[nameKey]; ok {
		return id, nil
	}
	code = s.ensurePartyCode("SUP", code, company, s.usedSupplierCodes)
	var id int64
	err := s.tx.QueryRow(s.ctx, `
		insert into suppliers (code, company, lastname, firstname, middlename, phone_number, address, balance, encoder_user_id, last_update_by_user_id)
		values ($1,$2,$3,$4,$5,$6,$7,0,$8,$8)
		returning id`,
		code, company, emptyString(last), emptyString(first), emptyString(middle), emptyString(phone), emptyString(address), s.user.ID,
	).Scan(&id)
	if err != nil {
		return 0, err
	}
	s.supplierByName[nameKey] = id
	return id, nil
}

func (s *importState) addCustomer(code, company, last, first, middle, phone, address, creditTerm, creditLimit, aps string, farm bool) (int64, error) {
	company = choose(company, joinName(last, first, middle))
	company = strings.TrimSpace(company)
	if company == "" {
		company = "Unnamed Customer"
	}
	nameKey := normalizeKey(company)
	if id, ok := s.customerByName[nameKey]; ok {
		return id, nil
	}
	code = s.ensurePartyCode("CUS", code, company, s.usedCustomerCodes)
	var id int64
	err := s.tx.QueryRow(s.ctx, `
		insert into customers (code, company, lastname, firstname, middlename, phone_number, address, balance, credit_term, credit_limit, aps, farm_customer, encoder_user_id, last_update_by_user_id)
		values ($1,$2,$3,$4,$5,$6,$7,0,$8,$9,$10,$11,$12,$12)
		returning id`,
		code, company, emptyString(last), emptyString(first), emptyString(middle), emptyString(phone), emptyString(address),
		emptyString(creditTerm), numericString(creditLimit), emptyString(aps), farm, s.user.ID,
	).Scan(&id)
	if err != nil {
		return 0, err
	}
	s.customerByName[nameKey] = id
	return id, nil
}

func (s *importState) addExpenseChart(chart legacyExpenseChart) (int64, error) {
	code := strings.TrimSpace(chart.Code)
	if code == "" {
		code = s.uniqueCode("EXP", len(s.expenseByCode)+1, s.expenseByCode)
	}
	if id, ok := s.expenseByCode[normalizeKey(code)]; ok {
		return id, nil
	}
	var id int64
	err := s.tx.QueryRow(s.ctx, `
		insert into expense_charts (code, name, description, exclude_daily_sales, daily_sales_only, encoder_user_id, last_update_by_user_id)
		values ($1,$2,$3,$4,$5,$6,$6)
		returning id`,
		code, choose(chart.Name, code), emptyString(chart.Description), chart.ExcludeSales, chart.DailyOnly, s.user.ID,
	).Scan(&id)
	if err != nil {
		return 0, err
	}
	s.expenseByCode[normalizeKey(code)] = id
	s.expenseByName[normalizeKey(chart.Name)] = id
	return id, nil
}

func (s *importState) addOtherIncomeChart(chart legacyOtherIncomeChart) (int64, error) {
	code := strings.TrimSpace(chart.Code)
	if code == "" {
		code = s.uniqueCode("OIN", len(s.otherIncomeByCode)+1, s.otherIncomeByCode)
	}
	if id, ok := s.otherIncomeByCode[normalizeKey(code)]; ok {
		return id, nil
	}
	var id int64
	err := s.tx.QueryRow(s.ctx, `
		insert into other_income_charts (code, name, description, encoder_user_id, last_update_by_user_id)
		values ($1,$2,$3,$4,$4)
		returning id`,
		code, choose(chart.Name, code), emptyString(chart.Description), s.user.ID,
	).Scan(&id)
	if err != nil {
		return 0, err
	}
	s.otherIncomeByCode[normalizeKey(code)] = id
	s.otherIncomeByName[normalizeKey(chart.Name)] = id
	return id, nil
}

func (s *importState) addStock(stock legacyStock) (int64, error) {
	code := strings.TrimSpace(stock.Code)
	if code == "" {
		return 0, errors.New("stock code is required")
	}
	if id, ok := s.stockByCode[normalizeKey(code)]; ok {
		return id, nil
	}
	var id int64
	err := s.tx.QueryRow(s.ctx, `
		insert into stocks (code, name, category_group, unit, description, latest_cost, min_inventory, encoder_user_id, last_update_by_user_id)
		values ($1,$2,$3,$4,$5,$6,$7,$8,$8)
		returning id`,
		code, choose(stock.Name, code), emptyString(stock.Category), emptyString(stock.Unit), emptyString(stock.Description),
		numericString(stock.LatestCost), numericString(stock.MinimumInv), s.user.ID,
	).Scan(&id)
	if err != nil {
		return 0, err
	}
	s.stockByCode[normalizeKey(code)] = id
	return id, nil
}

func (s *importState) importChecksIn(docs map[int]*legacyChecksIn) error {
	ids := sortedKeys(docs)
	for _, id := range ids {
		doc := docs[id]
		values := map[string]string{
			"entry_date": doc.EntryDate,
			"remarks":    doc.Remarks,
		}
		lineRows := make([]map[string]string, 0, len(doc.Checks))
		for _, check := range doc.Checks {
			lineRows = append(lineRows, map[string]string{
				"number":    check.Number,
				"date":      check.Date,
				"bank_name": check.BankName,
				"amount":    check.Amount,
				"nature":    check.Nature,
			})
		}
		importDoc := importedDocument{
			kind:         "checks-in",
			entryDate:    parseDate(doc.EntryDate),
			documentDate: parseDate(doc.EntryDate),
			remarks:      doc.Remarks,
			total:        numericString(doc.Total),
			net:          numericString(doc.Total),
			values:       values,
			lineInput:    []repositories.LineInput{{Group: "checks", Rows: lineRows}},
			lines:        buildCheckLines(lineRows),
		}
		if err := s.insertDocument(importDoc); err != nil {
			return err
		}
	}
	return nil
}

func (s *importState) importOtherIncome(docs map[int]*legacyOtherIncomeDoc) error {
	ids := sortedKeys(docs)
	for _, id := range ids {
		doc := docs[id]
		branchID := s.branchID(doc.Branch)
		values := map[string]string{
			"entry_date": doc.EntryDate,
			"remarks":    doc.Remarks,
			"branch_id":  int64String(branchID),
		}
		lineRows := make([]map[string]string, 0, len(doc.Lines))
		lines := make([]importedLine, 0, len(doc.Lines))
		for _, line := range doc.Lines {
			codeID := s.otherIncomeCodeID(line.Code, line.Name)
			row := map[string]string{
				"reference": line.Reference,
				"cash":      line.Cash,
				"check":     line.Check,
				"total":     choose(line.Total, sumMoney(line.Cash, line.Check)),
			}
			lineRows = append(lineRows, rowWithCodeID(row, codeID))
			lines = append(lines, importedLine{
				group:  "details",
				row:    rowWithCodeID(row, codeID),
				code:   codeID,
				cash:   line.Cash,
				check:  line.Check,
				amount: choose(line.Total, sumMoney(line.Cash, line.Check)),
			})
		}
		importDoc := importedDocument{
			kind:         "other-income",
			entryDate:    parseDate(doc.EntryDate),
			documentDate: parseDate(doc.EntryDate),
			branchID:     branchID,
			remarks:      doc.Remarks,
			total:        numericString(doc.Total),
			net:          numericString(doc.Total),
			values:       values,
			lineInput:    []repositories.LineInput{{Group: "details", Rows: lineRows}},
			lines:        lines,
		}
		if err := s.insertDocument(importDoc); err != nil {
			return err
		}
	}
	return nil
}

func (s *importState) importPurchases(docs map[int]*legacyPurchase) error {
	ids := sortedKeys(docs)
	for _, id := range ids {
		doc := docs[id]
		partyID, err := s.ensureSupplier(doc.Supplier)
		if err != nil {
			return err
		}
		values := map[string]string{
			"party_id":      strconv.FormatInt(partyID, 10),
			"entry_date":    doc.EntryDate,
			"remarks":       doc.Remarks,
			"cash":          boolString(doc.Cash),
			"or_ci_number":  joinReference(doc.ORNumber, doc.CINumber),
			"purchase_date": doc.PurchaseDate,
			"legacy_entry":  strconv.Itoa(doc.EntryID),
		}
		lineGroups, lines, inventory := s.purchaseLines(doc)
		checkRows, checkLines := paymentRows(doc.CashAmount, doc.CheckAmount, doc.Checks)
		if len(checkRows) > 0 {
			lineGroups = append(lineGroups, repositories.LineInput{Group: "payments", Rows: checkRows})
			lines = append(lines, checkLines...)
		}
		if len(doc.Discounts) > 0 {
			rows, discountLines := adjustmentRows("discounts", doc.Discounts)
			lineGroups = append(lineGroups, repositories.LineInput{Group: "discounts", Rows: rows})
			lines = append(lines, discountLines...)
		}
		if len(doc.Additionals) > 0 {
			rows, addLines := adjustmentRows("additionals", doc.Additionals)
			lineGroups = append(lineGroups, repositories.LineInput{Group: "additionals", Rows: rows})
			lines = append(lines, addLines...)
		}
		net := choose(doc.NetTotal, moneyAdd(doc.GrossTotal, negateMoney(doc.TotalDisc), doc.TotalAdd))
		balance := "0.00"
		if !doc.Cash {
			balance = moneySubtract(net, choose(doc.CashAmount, "0"), choose(doc.CheckAmount, "0"))
		}
		partyType := "supplier"
		importDoc := importedDocument{
			kind:         "purchases",
			entryDate:    parseDate(doc.EntryDate),
			documentDate: parseDate(choose(doc.PurchaseDate, doc.EntryDate)),
			partyType:    &partyType,
			partyID:      &partyID,
			reference:    joinReference(doc.ORNumber, doc.CINumber),
			cash:         doc.Cash,
			remarks:      doc.Remarks,
			total:        numericString(choose(doc.GrossTotal, totalAmountFromLines(doc.Details))),
			less:         numericString(sumAdjustments(doc.Discounts)),
			add:          numericString(sumAdjustments(doc.Additionals)),
			net:          numericString(net),
			balance:      numericString(balance),
			values:       values,
			lineInput:    lineGroups,
			lines:        lines,
			inventory:    inventory,
			balanceDelta: &legacyBalanceEffect{PartyType: "supplier", PartyID: partyID, Amount: numericString(balance)},
		}
		if err := s.insertDocument(importDoc); err != nil {
			return err
		}
	}
	return nil
}

func (s *importState) importSales(docs map[int]*legacySale) error {
	ids := sortedKeys(docs)
	for _, id := range ids {
		doc := docs[id]
		partyID, err := s.ensureCustomer(doc.Customer)
		if err != nil {
			return err
		}
		values := map[string]string{
			"party_id":     strconv.FormatInt(partyID, 10),
			"entry_date":   doc.EntryDate,
			"cash":         boolString(doc.Cash),
			"or_ci_number": joinReference(doc.ORNumber, doc.CINumber),
			"sales_date":   doc.SalesDate,
			"legacy_entry": strconv.Itoa(doc.EntryID),
		}
		lineGroups, lines, inventory := s.salesLines(doc)
		deductions := append([]legacyAdjustment{}, doc.Discounts...)
		if moneyToCents(doc.ManualDiscount) != 0 {
			deductions = append(deductions, legacyAdjustment{Name: "Manual Discount", Quantity: "1", Price: doc.ManualDiscount, Amount: doc.ManualDiscount})
		}
		if len(deductions) > 0 {
			rows, deductionLines := adjustmentRows("deductions", deductions)
			lineGroups = append(lineGroups, repositories.LineInput{Group: "deductions", Rows: rows})
			lines = append(lines, deductionLines...)
		}
		if len(doc.Additionals) > 0 {
			rows, addLines := adjustmentRows("additionals", doc.Additionals)
			lineGroups = append(lineGroups, repositories.LineInput{Group: "additionals", Rows: rows})
			lines = append(lines, addLines...)
		}
		checkRows, checkLines := paymentRows(doc.CashAmount, doc.CheckAmount, doc.Checks)
		if len(checkRows) > 0 {
			lineGroups = append(lineGroups, repositories.LineInput{Group: "payments", Rows: checkRows})
			lines = append(lines, checkLines...)
		}
		total := choose(doc.TotalNetAmount, doc.GrossTotal)
		net := choose(doc.NetTotal, moneyAdd(total, negateMoney(sumAdjustments(deductions)), sumAdjustments(doc.Additionals)))
		balance := "0.00"
		if !doc.Cash {
			balance = moneySubtract(net, choose(doc.CashAmount, "0"), choose(doc.CheckAmount, "0"))
		}
		partyType := "customer"
		importDoc := importedDocument{
			kind:         "sales",
			entryDate:    parseDate(doc.EntryDate),
			documentDate: parseDate(choose(doc.SalesDate, doc.EntryDate)),
			partyType:    &partyType,
			partyID:      &partyID,
			reference:    joinReference(doc.ORNumber, doc.CINumber),
			cash:         doc.Cash,
			total:        numericString(total),
			less:         numericString(sumAdjustments(deductions)),
			add:          numericString(sumAdjustments(doc.Additionals)),
			net:          numericString(net),
			balance:      numericString(balance),
			values:       values,
			lineInput:    lineGroups,
			lines:        lines,
			inventory:    inventory,
			balanceDelta: &legacyBalanceEffect{PartyType: "customer", PartyID: partyID, Amount: numericString(balance)},
		}
		if err := s.insertDocument(importDoc); err != nil {
			return err
		}
	}
	return nil
}

func (s *importState) importStockInOut(kind string, docs map[int]*legacyStockDoc, direction int) error {
	ids := sortedKeys(docs)
	for _, id := range ids {
		doc := docs[id]
		values := map[string]string{
			"entry_date":   doc.EntryDate,
			"remarks":      doc.Remarks,
			"legacy_entry": strconv.Itoa(doc.EntryID),
		}
		rows := make([]map[string]string, 0, len(doc.Details))
		lines := make([]importedLine, 0, len(doc.Details))
		inventory := make([]legacyInventoryEffect, 0, len(doc.Details))
		for _, line := range doc.Details {
			stockID := s.stockID(line.StockCode)
			if stockID == nil {
				continue
			}
			row := map[string]string{
				"qty":       line.Quantity,
				"unit_cost": line.UnitCost,
				"amount":    choose(line.Amount, multiplyMoney(line.Quantity, line.UnitCost)),
			}
			row = rowWithStockID(row, stockID)
			rows = append(rows, row)
			lines = append(lines, importedLine{
				group:  "details",
				row:    row,
				stock:  stockID,
				qty:    line.Quantity,
				cost:   line.UnitCost,
				amount: choose(line.Amount, multiplyMoney(line.Quantity, line.UnitCost)),
			})
			inventory = append(inventory, legacyInventoryEffect{
				StockID:  *stockID,
				QtyDelta: signedQuantity(line.Quantity, direction),
				UnitCost: line.UnitCost,
			})
		}
		importDoc := importedDocument{
			kind:         kind,
			entryDate:    parseDate(doc.EntryDate),
			documentDate: parseDate(doc.EntryDate),
			remarks:      doc.Remarks,
			total:        numericString(doc.Total),
			net:          numericString(doc.Total),
			values:       values,
			lineInput:    []repositories.LineInput{{Group: "details", Rows: rows}},
			lines:        lines,
			inventory:    inventory,
		}
		if err := s.insertDocument(importDoc); err != nil {
			return err
		}
	}
	return nil
}

func (s *importState) importStockTransfers(docs map[int]*legacyStockTransfer) error {
	ids := sortedKeys(docs)
	for _, id := range ids {
		doc := docs[id]
		values := map[string]string{
			"entry_date":      doc.EntryDate,
			"remarks":         doc.Remarks,
			"transfer_date":   doc.TransferDate,
			"transfer_id":     doc.TransferID,
			"transaction":     doc.Transaction,
			"branch_location": doc.Branch,
			"legacy_entry":    strconv.Itoa(doc.EntryID),
			"customer_name":   doc.Customer,
			"supplier_name":   doc.Supplier,
			"bodega_name":     doc.Bodega,
		}
		sourceBranch := s.branchID(doc.Bodega)
		transaction := strings.ToLower(strings.TrimSpace(doc.Transaction))
		direction := -1
		var partyType *string
		var partyID *int64
		if strings.HasPrefix(transaction, "1") || strings.Contains(transaction, "sales return") {
			direction = 1
			id, err := s.ensureParty("customer", doc.Customer)
			if err != nil {
				return err
			}
			party := "customer"
			partyType, partyID = &party, &id
			values["customer_id"] = strconv.FormatInt(id, 10)
		} else if strings.HasPrefix(transaction, "2") || strings.Contains(transaction, "stock return") {
			id, err := s.ensureParty("supplier", doc.Supplier)
			if err != nil {
				return err
			}
			party := "supplier"
			partyType, partyID = &party, &id
			values["supplier_id"] = strconv.FormatInt(id, 10)
		}
		rows := make([]map[string]string, 0, len(doc.Details))
		lines := make([]importedLine, 0, len(doc.Details))
		inventory := make([]legacyInventoryEffect, 0, len(doc.Details))
		for _, line := range doc.Details {
			stockID := s.stockID(line.StockCode)
			if stockID == nil {
				continue
			}
			row := map[string]string{
				"qty":         line.Quantity,
				"unit_cost":   line.UnitCost,
				"amount":      choose(line.Amount, multiplyMoney(line.Quantity, line.UnitCost)),
				"capital":     line.Capital,
				"markup":      line.Markup,
				"markup_pct":  line.MarkupPct,
				"stock_label": line.StockName,
			}
			row = rowWithStockID(row, stockID)
			rows = append(rows, row)
			lines = append(lines, importedLine{
				group:  "details",
				row:    row,
				stock:  stockID,
				qty:    line.Quantity,
				cost:   line.UnitCost,
				amount: choose(line.Amount, multiplyMoney(line.Quantity, line.UnitCost)),
			})
			inventory = append(inventory, legacyInventoryEffect{
				BranchID: sourceBranch,
				StockID:  *stockID,
				QtyDelta: signedQuantity(line.Quantity, direction),
				UnitCost: line.UnitCost,
			})
		}
		lineGroups := []repositories.LineInput{{Group: "details", Rows: rows}}
		if len(doc.Discounts) > 0 {
			rows, discountLines := adjustmentRows("discounts", doc.Discounts)
			lineGroups = append(lineGroups, repositories.LineInput{Group: "discounts", Rows: rows})
			lines = append(lines, discountLines...)
		}
		if len(doc.Additionals) > 0 {
			rows, addLines := adjustmentRows("additionals", doc.Additionals)
			lineGroups = append(lineGroups, repositories.LineInput{Group: "additionals", Rows: rows})
			lines = append(lines, addLines...)
		}
		net := numericString(choose(doc.NetTotal, moneyAdd(doc.GrossTotal, negateMoney(sumAdjustments(doc.Discounts)), sumAdjustments(doc.Additionals))))
		var balanceDelta *legacyBalanceEffect
		if partyType != nil && partyID != nil {
			balanceDelta = &legacyBalanceEffect{PartyType: *partyType, PartyID: *partyID, Amount: numericString(negateMoney(net))}
		}
		importDoc := importedDocument{
			kind:         "stock-transactions",
			entryDate:    parseDate(doc.EntryDate),
			documentDate: parseDate(choose(doc.TransferDate, doc.EntryDate)),
			branchID:     sourceBranch,
			partyType:    partyType,
			partyID:      partyID,
			reference:    doc.TransferID,
			cash:         doc.Cash,
			remarks:      doc.Remarks,
			total:        numericString(doc.GrossTotal),
			less:         numericString(sumAdjustments(doc.Discounts)),
			add:          numericString(sumAdjustments(doc.Additionals)),
			net:          net,
			values:       values,
			lineInput:    lineGroups,
			lines:        lines,
			inventory:    inventory,
			balanceDelta: balanceDelta,
		}
		if err := s.insertDocument(importDoc); err != nil {
			return err
		}
	}
	return nil
}

func (s *importState) importCredits(kind, partyType string, docs map[int]*legacyCreditDoc, postBalance bool) error {
	ids := sortedKeys(docs)
	for _, id := range ids {
		doc := docs[id]
		partyID, err := s.ensureParty(partyType, doc.Company)
		if err != nil {
			return err
		}
		values := map[string]string{
			"entry_date":   doc.EntryDate,
			"reference":    doc.Reference,
			"party_id":     strconv.FormatInt(partyID, 10),
			"cash_amount":  choose(doc.CashAmount, "0"),
			"remarks":      doc.Remarks,
			"legacy_entry": strconv.Itoa(doc.EntryID),
		}
		checkRows, checkLines := paymentRows(doc.CashAmount, doc.CheckAmount, doc.Checks)
		lineGroups := []repositories.LineInput{}
		lines := []importedLine{}
		if len(checkRows) > 0 {
			lineGroups = append(lineGroups, repositories.LineInput{Group: "checks", Rows: checkRows})
			lines = append(lines, checkLines...)
		}
		amount := choose(doc.Amount, moneyAdd(doc.CashAmount, doc.CheckAmount))
		var balanceDelta *legacyBalanceEffect
		if postBalance {
			balanceDelta = &legacyBalanceEffect{PartyType: partyType, PartyID: partyID, Amount: numericString(negateMoney(amount))}
		}
		importDoc := importedDocument{
			kind:         kind,
			entryDate:    parseDate(doc.EntryDate),
			documentDate: parseDate(doc.EntryDate),
			partyType:    &partyType,
			partyID:      &partyID,
			reference:    doc.Reference,
			remarks:      doc.Remarks,
			net:          numericString(amount),
			values:       values,
			lineInput:    lineGroups,
			lines:        lines,
			balanceDelta: balanceDelta,
		}
		if err := s.insertDocument(importDoc); err != nil {
			return err
		}
	}
	return nil
}

func (s *importState) importDebits(kind, partyType string, docs map[int]*legacyDebitDoc) error {
	ids := sortedKeys(docs)
	for _, id := range ids {
		doc := docs[id]
		partyID, err := s.ensureParty(partyType, doc.Company)
		if err != nil {
			return err
		}
		values := map[string]string{
			"entry_date":   doc.EntryDate,
			"party_id":     strconv.FormatInt(partyID, 10),
			"amount":       doc.Amount,
			"remarks":      doc.Remarks,
			"legacy_entry": strconv.Itoa(doc.EntryID),
		}
		importDoc := importedDocument{
			kind:         kind,
			entryDate:    parseDate(doc.EntryDate),
			documentDate: parseDate(doc.EntryDate),
			partyType:    &partyType,
			partyID:      &partyID,
			remarks:      doc.Remarks,
			net:          numericString(doc.Amount),
			values:       values,
			balanceDelta: &legacyBalanceEffect{PartyType: partyType, PartyID: partyID, Amount: numericString(doc.Amount)},
		}
		if err := s.insertDocument(importDoc); err != nil {
			return err
		}
	}
	return nil
}

func (s *importState) importExpenses(docs map[int]*legacyExpenseDoc) error {
	ids := sortedKeys(docs)
	for _, id := range ids {
		doc := docs[id]
		values := map[string]string{
			"entry_date":   doc.EntryDate,
			"remarks":      doc.Remarks,
			"legacy_entry": strconv.Itoa(doc.EntryID),
		}
		rows := make([]map[string]string, 0, len(doc.Lines))
		lines := make([]importedLine, 0, len(doc.Lines))
		for _, line := range doc.Lines {
			codeID := s.expenseCodeID(line.Code, line.Name)
			row := map[string]string{
				"reference": line.Reference,
				"cash":      line.Cash,
				"check":     line.Check,
				"total":     choose(line.Total, sumMoney(line.Cash, line.Check)),
			}
			row = rowWithCodeID(row, codeID)
			rows = append(rows, row)
			lines = append(lines, importedLine{
				group:  "details",
				row:    row,
				code:   codeID,
				cash:   line.Cash,
				check:  line.Check,
				amount: choose(line.Total, sumMoney(line.Cash, line.Check)),
			})
		}
		lineGroups := []repositories.LineInput{{Group: "details", Rows: rows}}
		checkRows, checkLines := paymentRows("0", "0", doc.Checks)
		if len(checkRows) > 0 {
			lineGroups = append(lineGroups, repositories.LineInput{Group: "checks", Rows: checkRows})
			lines = append(lines, checkLines...)
		}
		importDoc := importedDocument{
			kind:         "expenses",
			entryDate:    parseDate(doc.EntryDate),
			documentDate: parseDate(doc.EntryDate),
			remarks:      joinReference(doc.Remarks, doc.Reference),
			total:        numericString(doc.Total),
			net:          numericString(doc.Total),
			values:       values,
			lineInput:    lineGroups,
			lines:        lines,
		}
		if err := s.insertDocument(importDoc); err != nil {
			return err
		}
	}
	return nil
}

func (s *importState) insertDocument(doc importedDocument) error {
	payload, err := json.Marshal(storedPayload{
		Kind:      doc.kind,
		Values:    doc.values,
		LineInput: doc.lineInput,
		EncoderID: s.user.ID,
	})
	if err != nil {
		return err
	}
	var docID int64
	err = s.tx.QueryRow(s.ctx, `
		insert into documents
			(kind, entry_date, document_date, branch_id, party_type, party_id, reference, cash, remarks, total, less_amount, add_amount, net, balance, payload, encoder_user_id, last_update_by_user_id)
		values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$16)
		returning id`,
		doc.kind,
		doc.entryDate,
		doc.documentDate,
		doc.branchID,
		doc.partyType,
		doc.partyID,
		nilIfEmpty(doc.reference),
		doc.cash,
		doc.remarks,
		numericString(doc.total),
		numericString(doc.less),
		numericString(doc.add),
		numericString(doc.net),
		numericString(doc.balance),
		payload,
		s.user.ID,
	).Scan(&docID)
	if err != nil {
		return err
	}

	for idx, line := range doc.lines {
		linePayload, err := json.Marshal(line.row)
		if err != nil {
			return err
		}
		if _, err := s.tx.Exec(s.ctx, `
			insert into document_lines (document_id, group_key, line_no, stock_id, code_id, qty, unit_cost, price, cash_amount, check_amount, amount, payload)
			values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
			docID, line.group, idx+1, line.stock, line.code,
			qtyString(line.qty), numericString(line.cost), numericString(line.price),
			numericString(line.cash), numericString(line.check), numericString(line.amount), linePayload,
		); err != nil {
			return err
		}
	}

	for _, effect := range doc.inventory {
		if _, err := s.tx.Exec(s.ctx, `
			insert into stock_ledger (document_id, branch_id, stock_id, qty_delta, unit_cost)
			values ($1,$2,$3,$4,$5)`,
			docID, effect.BranchID, effect.StockID, qtyString(effect.QtyDelta), numericString(effect.UnitCost),
		); err != nil {
			return err
		}
	}

	if doc.balanceDelta != nil && moneyToCents(doc.balanceDelta.Amount) != 0 {
		if _, err := s.tx.Exec(s.ctx, `
			insert into balance_ledger (document_id, party_type, party_id, amount_delta)
			values ($1,$2,$3,$4)`,
			docID, doc.balanceDelta.PartyType, doc.balanceDelta.PartyID, numericString(doc.balanceDelta.Amount),
		); err != nil {
			return err
		}
	}

	return nil
}

func (s *importState) purchaseLines(doc *legacyPurchase) ([]repositories.LineInput, []importedLine, []legacyInventoryEffect) {
	rows := make([]map[string]string, 0, len(doc.Details))
	lines := make([]importedLine, 0, len(doc.Details))
	inventory := make([]legacyInventoryEffect, 0, len(doc.Details))
	for _, line := range doc.Details {
		stockID := s.stockID(line.StockCode)
		if stockID == nil {
			continue
		}
		row := map[string]string{
			"qty":       line.Quantity,
			"unit_cost": line.UnitCost,
			"amount":    choose(line.Amount, multiplyMoney(line.Quantity, line.UnitCost)),
		}
		row = rowWithStockID(row, stockID)
		rows = append(rows, row)
		lines = append(lines, importedLine{
			group:  "details",
			row:    row,
			stock:  stockID,
			qty:    line.Quantity,
			cost:   line.UnitCost,
			amount: choose(line.Amount, multiplyMoney(line.Quantity, line.UnitCost)),
		})
		inventory = append(inventory, legacyInventoryEffect{
			StockID:  *stockID,
			QtyDelta: line.Quantity,
			UnitCost: line.UnitCost,
		})
	}
	return []repositories.LineInput{{Group: "details", Rows: rows}}, lines, inventory
}

func (s *importState) salesLines(doc *legacySale) ([]repositories.LineInput, []importedLine, []legacyInventoryEffect) {
	rows := make([]map[string]string, 0, len(doc.Details))
	lines := make([]importedLine, 0, len(doc.Details))
	inventory := make([]legacyInventoryEffect, 0, len(doc.Details))
	for _, line := range doc.Details {
		stockID := s.stockID(line.StockCode)
		if stockID == nil {
			continue
		}
		row := map[string]string{
			"qty":            line.Quantity,
			"unit_cost":      line.UnitCost,
			"amount":         choose(line.Amount, multiplyMoney(line.Quantity, line.UnitCost)),
			"capital":        line.Capital,
			"discount":       line.Discount,
			"other_discount": line.OtherDisc,
			"markup":         line.Markup,
			"markup_pct":     line.MarkupPct,
		}
		row = rowWithStockID(row, stockID)
		rows = append(rows, row)
		lines = append(lines, importedLine{
			group:  "details",
			row:    row,
			stock:  stockID,
			qty:    line.Quantity,
			cost:   line.UnitCost,
			price:  line.UnitCost,
			amount: choose(line.Amount, multiplyMoney(line.Quantity, line.UnitCost)),
		})
		inventory = append(inventory, legacyInventoryEffect{
			StockID:  *stockID,
			QtyDelta: signedQuantity(line.Quantity, -1),
			UnitCost: line.UnitCost,
		})
	}
	return []repositories.LineInput{{Group: "details", Rows: rows}}, lines, inventory
}

func adjustmentRows(group string, adjustments []legacyAdjustment) ([]map[string]string, []importedLine) {
	rows := make([]map[string]string, 0, len(adjustments))
	lines := make([]importedLine, 0, len(adjustments))
	for _, adj := range adjustments {
		amount := choose(adj.Amount, multiplyMoney(adj.Quantity, adj.Price))
		row := map[string]string{
			"particulars": adj.Name,
			"qty":         choose(adj.Quantity, "1"),
			"price":       adj.Price,
			"amount":      amount,
		}
		rows = append(rows, row)
		lines = append(lines, importedLine{
			group:  group,
			row:    row,
			qty:    choose(adj.Quantity, "1"),
			price:  adj.Price,
			amount: amount,
		})
	}
	return rows, lines
}

func paymentRows(cashAmount, checkAmount string, checks []legacyCheck) ([]map[string]string, []importedLine) {
	rows := make([]map[string]string, 0, len(checks))
	lines := make([]importedLine, 0, len(checks))
	if moneyToCents(checkAmount) != 0 && len(checks) == 0 {
		checks = append(checks, legacyCheck{Amount: checkAmount})
	}
	_ = cashAmount
	for _, check := range checks {
		row := map[string]string{
			"number":    check.Number,
			"date":      check.Date,
			"bank_name": check.BankName,
			"amount":    check.Amount,
			"nature":    check.Nature,
		}
		rows = append(rows, row)
		lines = append(lines, importedLine{
			group:  "checks",
			row:    row,
			amount: check.Amount,
		})
	}
	return rows, lines
}

func buildCheckLines(rows []map[string]string) []importedLine {
	lines := make([]importedLine, 0, len(rows))
	for _, row := range rows {
		lines = append(lines, importedLine{
			group:  "checks",
			row:    row,
			amount: row["amount"],
		})
	}
	return lines
}

func (s *importState) ensureSupplier(name string) (int64, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "Unknown Supplier"
	}
	if id, ok := s.supplierByName[normalizeKey(name)]; ok {
		return id, nil
	}
	return s.addSupplier("", name, "", "", "", "", "")
}

func (s *importState) ensureCustomer(name string) (int64, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "Unknown Customer"
	}
	if id, ok := s.customerByName[normalizeKey(name)]; ok {
		return id, nil
	}
	return s.addCustomer("", name, "", "", "", "", "", "", "0", "", false)
}

func (s *importState) ensureParty(partyType, name string) (int64, error) {
	if partyType == "supplier" {
		return s.ensureSupplier(name)
	}
	return s.ensureCustomer(name)
}

func (s *importState) branchID(name string) *int64 {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}
	if id, ok := s.branchByName[normalizeKey(name)]; ok {
		return &id
	}
	return nil
}

func (s *importState) stockID(code string) *int64 {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil
	}
	if id, ok := s.stockByCode[normalizeKey(code)]; ok {
		return &id
	}
	return nil
}

func (s *importState) expenseCodeID(code, name string) *int64 {
	if code != "" {
		if id, ok := s.expenseByCode[normalizeKey(code)]; ok {
			return &id
		}
	}
	if name != "" {
		if id, ok := s.expenseByName[normalizeKey(name)]; ok {
			return &id
		}
	}
	return nil
}

func (s *importState) otherIncomeCodeID(code, name string) *int64 {
	if code != "" {
		if id, ok := s.otherIncomeByCode[normalizeKey(code)]; ok {
			return &id
		}
	}
	if name != "" {
		if id, ok := s.otherIncomeByName[normalizeKey(name)]; ok {
			return &id
		}
	}
	return nil
}

func (s *importState) recomputePartyBalances() error {
	_, err := s.tx.Exec(s.ctx, `
		update customers c
		set balance = coalesce((
			select sum(bl.amount_delta)
			from balance_ledger bl
			where bl.party_type = 'customer'
			  and bl.party_id = c.id
		), 0);

		update suppliers s
		set balance = coalesce((
			select sum(bl.amount_delta)
			from balance_ledger bl
			where bl.party_type = 'supplier'
			  and bl.party_id = s.id
		), 0);
	`)
	return err
}

func (s *importState) assignActiveBranch() error {
	if len(s.branchByName) == 0 {
		return nil
	}
	var firstID int64
	for _, id := range s.branchByName {
		firstID = id
		break
	}
	_, err := s.tx.Exec(s.ctx, `update users set active_branch_id = coalesce(active_branch_id, $1)`, firstID)
	return err
}

func (s *importState) ensurePartyCode(prefix, rawCode, company string, used map[string]struct{}) string {
	code := strings.TrimSpace(rawCode)
	if code == "" {
		code = prefix + "-" + strings.ToUpper(compactCode(company))
	}
	base := code
	for i := 1; ; i++ {
		key := normalizeKey(code)
		if _, ok := used[key]; !ok {
			used[key] = struct{}{}
			return code
		}
		code = fmt.Sprintf("%s-%d", base, i)
	}
}

func (s *importState) uniqueCode(prefix string, index int, existing map[string]int64) string {
	for i := index; ; i++ {
		code := fmt.Sprintf("%s-%04d", prefix, i)
		if _, ok := existing[normalizeKey(code)]; !ok {
			return code
		}
	}
}

func sortedKeys[T any](items map[int]T) []int {
	keys := make([]int, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	sort.Ints(keys)
	return keys
}

func rowWithStockID(row map[string]string, stockID *int64) map[string]string {
	out := cloneRow(row)
	out["stock_id"] = int64String(stockID)
	return out
}

func rowWithCodeID(row map[string]string, codeID *int64) map[string]string {
	out := cloneRow(row)
	out["code_id"] = int64String(codeID)
	return out
}

func cloneRow(row map[string]string) map[string]string {
	out := make(map[string]string, len(row)+1)
	for key, value := range row {
		out[key] = value
	}
	return out
}

func totalAmountFromLines(lines []legacyStockLine) string {
	total := int64(0)
	for _, line := range lines {
		total += moneyToCents(choose(line.Amount, multiplyMoney(line.Quantity, line.UnitCost)))
	}
	return centsToNumeric(total)
}

func sumAdjustments(adjustments []legacyAdjustment) string {
	total := int64(0)
	for _, adj := range adjustments {
		total += moneyToCents(choose(adj.Amount, multiplyMoney(adj.Quantity, adj.Price)))
	}
	return centsToNumeric(total)
}

func sumMoney(parts ...string) string {
	total := int64(0)
	for _, part := range parts {
		total += moneyToCents(part)
	}
	return centsToNumeric(total)
}

func multiplyMoney(qty, price string) string {
	q := numericToMilli(qty)
	p := moneyToCents(price)
	return centsToNumeric((q * p) / 1000)
}

func signedQuantity(value string, direction int) string {
	milli := numericToMilli(value)
	if direction < 0 {
		milli = -milli
	}
	return milliToNumeric(milli)
}

func numericString(value string) string {
	return centsToNumeric(moneyToCents(value))
}

func qtyString(value string) string {
	milli := numericToMilli(value)
	negative := milli < 0
	if negative {
		milli = -milli
	}
	cents := (milli + 5) / 10
	if negative {
		cents = -cents
	}
	return centsToNumeric(cents)
}

func chooseNumeric(values ...string) string {
	for _, value := range values {
		if moneyToCents(value) != 0 || strings.TrimSpace(value) != "" {
			return value
		}
	}
	return "0.00"
}

func choose(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func chooseReference(a, b string) string {
	switch {
	case strings.TrimSpace(a) != "" && strings.TrimSpace(b) != "":
		return strings.TrimSpace(a) + " / " + strings.TrimSpace(b)
	case strings.TrimSpace(a) != "":
		return strings.TrimSpace(a)
	default:
		return strings.TrimSpace(b)
	}
}

func joinReference(a, b string) string {
	return chooseReference(a, b)
}

func joinName(parts ...string) string {
	words := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" || part == "-" || part == "0" {
			continue
		}
		words = append(words, part)
	}
	return strings.Join(words, " ")
}

func guessCategoryGroup(category string) string {
	category = strings.TrimSpace(category)
	if category == "" {
		return ""
	}
	if strings.Contains(category, "/") {
		parts := strings.Split(category, "/")
		return strings.TrimSpace(parts[0])
	}
	return category
}

func compactCode(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	value = strings.NewReplacer(" ", "", "/", "", "\\", "", "-", "", ",", "", ".", "", "#", "", "(", "", ")", "", "&", "").Replace(value)
	if value == "" {
		return "GEN"
	}
	if len(value) > 12 {
		value = value[:12]
	}
	return value
}

func normalizeKey(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	value = strings.Join(strings.Fields(value), " ")
	return value
}

func moneyToCents(value string) int64 {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	negative := strings.HasPrefix(value, "-")
	value = strings.TrimPrefix(value, "-")
	parts := strings.SplitN(value, ".", 3)
	whole, _ := strconv.ParseInt(parts[0], 10, 64)
	var cents int64
	if len(parts) > 1 {
		frac := parts[1]
		if len(frac) == 1 {
			frac += "0"
		}
		if len(frac) > 2 {
			frac = frac[:2]
		}
		cents, _ = strconv.ParseInt(frac, 10, 64)
	}
	total := whole*100 + cents
	if negative {
		return -total
	}
	return total
}

func parseInt(value string) int {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	n, _ := strconv.Atoi(value)
	return n
}

func numericToMilli(value string) int64 {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	negative := strings.HasPrefix(value, "-")
	value = strings.TrimPrefix(value, "-")
	parts := strings.SplitN(value, ".", 3)
	whole, _ := strconv.ParseInt(parts[0], 10, 64)
	var milli int64
	if len(parts) > 1 {
		frac := parts[1]
		for len(frac) < 3 {
			frac += "0"
		}
		if len(frac) > 3 {
			frac = frac[:3]
		}
		milli, _ = strconv.ParseInt(frac, 10, 64)
	}
	total := whole*1000 + milli
	if negative {
		return -total
	}
	return total
}

func centsToNumeric(cents int64) string {
	sign := ""
	if cents < 0 {
		sign = "-"
		cents = -cents
	}
	return fmt.Sprintf("%s%d.%02d", sign, cents/100, cents%100)
}

func milliToNumeric(milli int64) string {
	sign := ""
	if milli < 0 {
		sign = "-"
		milli = -milli
	}
	return fmt.Sprintf("%s%d.%03d", sign, milli/1000, milli%1000)
}

func moneyAdd(base string, changes ...string) string {
	total := moneyToCents(base)
	for _, change := range changes {
		total += moneyToCents(change)
	}
	return centsToNumeric(total)
}

func moneySubtract(base string, parts ...string) string {
	total := moneyToCents(base)
	for _, part := range parts {
		total -= moneyToCents(part)
	}
	return centsToNumeric(total)
}

func negateMoney(value string) string {
	return centsToNumeric(-moneyToCents(value))
}

func parseDate(value string) time.Time {
	value = strings.TrimSpace(value)
	if value == "" || value == "0000-00-00" {
		return time.Date(2000, 1, 1, 0, 0, 0, 0, time.FixedZone("Asia/Manila", 8*60*60))
	}
	t, err := time.Parse("2006-01-02", value)
	if err != nil {
		return time.Date(2000, 1, 1, 0, 0, 0, 0, time.FixedZone("Asia/Manila", 8*60*60))
	}
	return t
}

func parseBoolString(value string) bool {
	return value == "1" || strings.EqualFold(value, "true")
}

func boolString(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func emptyString(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return strings.TrimSpace(value)
}

func nilIfEmpty(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return strings.TrimSpace(value)
}

func int64String(value *int64) string {
	if value == nil {
		return ""
	}
	return strconv.FormatInt(*value, 10)
}

func truncateBusinessData(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `
		update users set active_branch_id = null;

		delete from dr_consumptions;
		delete from stock_ledger;
		delete from balance_ledger;
		delete from document_lines;
		delete from documents;
		delete from stocks;
		delete from other_income_charts;
		delete from expense_charts;
		delete from customers;
		delete from suppliers;
		delete from stock_categories;
		delete from branches;

		alter sequence dr_consumptions_id_seq restart with 1;
		alter sequence stock_ledger_id_seq restart with 1;
		alter sequence balance_ledger_id_seq restart with 1;
		alter sequence document_lines_id_seq restart with 1;
		alter sequence documents_id_seq restart with 1;
		alter sequence stocks_id_seq restart with 1;
		alter sequence other_income_charts_id_seq restart with 1;
		alter sequence expense_charts_id_seq restart with 1;
		alter sequence customers_id_seq restart with 1;
		alter sequence suppliers_id_seq restart with 1;
		alter sequence stock_categories_id_seq restart with 1;
		alter sequence branches_id_seq restart with 1;
	`)
	return err
}

func printSummary(ctx context.Context, pool *pgxpool.Pool) error {
	fmt.Println("Import complete")
	rows, err := pool.Query(ctx, `
		select kind, count(*)
		from documents
		group by kind
		order by kind`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var kind string
		var count int64
		if err := rows.Scan(&kind, &count); err != nil {
			return err
		}
		fmt.Printf("%s: %d\n", kind, count)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	var users, branches, customers, suppliers, stocks int64
	if err := pool.QueryRow(ctx, `
		select
			(select count(*) from users),
			(select count(*) from branches),
			(select count(*) from customers),
			(select count(*) from suppliers),
			(select count(*) from stocks)`).Scan(&users, &branches, &customers, &suppliers, &stocks); err != nil {
		return err
	}
	fmt.Printf("users=%d branches=%d customers=%d suppliers=%d stocks=%d\n", users, branches, customers, suppliers, stocks)
	return nil
}

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
