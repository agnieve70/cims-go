package repositories

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	appdb "cims-go/internal/db"
	"cims-go/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestValueForMasterFieldDefaultsBlankCustomerCreditLimit(t *testing.T) {
	form := models.FormDefinition{Kind: "customers"}
	field := models.Field{Key: "credit_limit", Type: models.FieldMoney}

	got := valueForMasterField(form, field, map[string]string{"credit_limit": ""})
	if got != "0.000" {
		t.Fatalf("blank customer credit_limit = %#v, want 0.000", got)
	}
}

func TestParseFixedPreservesThreeDecimals(t *testing.T) {
	for input, want := range map[string]int64{
		"1": 1000, "1.2": 1200, "1.234": 1234, "1,250.875": 1250875, "-0.125": -125,
	} {
		if got := parseFixed(input); got != want {
			t.Errorf("parseFixed(%q) = %d, want %d", input, got, want)
		}
	}
	if got := centsToNumeric(1250875); got != "1250.875" {
		t.Fatalf("numeric formatting = %q, want 1250.875", got)
	}
}

func TestStockTransactionRequiresStockOutLineReferences(t *testing.T) {
	input := totalsInput{net: 1000}
	groups := []LineInput{{Group: "details", Rows: []map[string]string{{
		"stock_id": "1", "qty": "1", "unit_cost": "1",
	}}}}
	err := validateDocumentInput("stock-transactions", map[string]string{
		"dr_document_id": "7", "transaction": "0 - Stock Transfer", "branch_location": "2",
	}, groups, input)
	if err == nil || !strings.Contains(err.Error(), "come from the selected Stock Out File") {
		t.Fatalf("validation error = %v, want Stock Out File line requirement", err)
	}
}

func TestValueForMasterFieldLeavesOtherBlankMoneyNullable(t *testing.T) {
	form := models.FormDefinition{Kind: "stocks"}
	field := models.Field{Key: "latest_cost", Type: models.FieldMoney}

	got := valueForMasterField(form, field, map[string]string{"latest_cost": ""})
	if got != nil {
		t.Fatalf("blank stock latest_cost = %#v, want nil", got)
	}
}

func TestPostgresStoreMasterAndPurchaseDocumentCRUD(t *testing.T) {
	databaseURL := os.Getenv("CIMS_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set CIMS_TEST_DATABASE_URL to run database-backed repository CRUD coverage")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	migrationsDir := filepath.Join("..", "..", "db", "migrations")
	if err := appdb.Migrate(databaseURL, migrationsDir); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open postgres pool: %v", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping postgres: %v", err)
	}

	store := NewPostgresStore(pool)
	suffix := fmt.Sprintf("IT%d", time.Now().UnixNano())

	var userID int64
	if err := pool.QueryRow(ctx, `
		insert into users (username, password_hash, display_name, role)
		values ($1, 'test-hash', 'Integration Test User', 'admin')
		returning id`, "it-"+suffix).Scan(&userID); err != nil {
		t.Fatalf("insert test user: %v", err)
	}
	user := models.User{ID: userID, Username: "it-" + suffix, DisplayName: "Integration Test User", Role: models.RoleAdmin}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `delete from users where id=$1`, userID)
	})

	branchForm := masterFormByKind(t, "branches")
	supplierForm := masterFormByKind(t, "suppliers")
	customerForm := masterFormByKind(t, "customers")
	stockForm := masterFormByKind(t, "stocks")
	purchaseForm := transactionFormByKind(t, "purchases")
	stockOutForm := transactionFormByKind(t, "dr")
	salesForm := transactionFormByKind(t, "sales")
	stockTransactionForm := transactionFormByKind(t, "stock-transactions")

	branchID, err := store.SaveMaster(ctx, branchForm, 0, map[string]string{
		"code":     "BR-" + suffix,
		"name":     "Integration Branch " + suffix,
		"incharge": "Tester",
	}, user)
	if err != nil {
		t.Fatalf("create branch: %v", err)
	}
	t.Cleanup(func() {
		_ = store.DeleteMaster(context.Background(), branchForm, branchID, user)
	})
	user.ActiveBranchID = branchID

	updatedBranchID, err := store.SaveMaster(ctx, branchForm, branchID, map[string]string{
		"code":     "BR-" + suffix,
		"name":     "Integration Branch Updated " + suffix,
		"incharge": "Tester",
	}, user)
	if err != nil {
		t.Fatalf("update branch: %v", err)
	}
	if updatedBranchID != branchID {
		t.Fatalf("updated branch id = %d, want %d", updatedBranchID, branchID)
	}
	branch, err := store.GetMaster(ctx, branchForm, branchID)
	if err != nil {
		t.Fatalf("get updated branch: %v", err)
	}
	if branch["name"] != "Integration Branch Updated "+suffix {
		t.Fatalf("updated branch name = %q", branch["name"])
	}

	supplierID, err := store.SaveMaster(ctx, supplierForm, 0, map[string]string{
		"code":      "SUP-" + suffix,
		"company":   "Integration Supplier " + suffix,
		"lastname":  "Supplier",
		"firstname": "Integration",
	}, user)
	if err != nil {
		t.Fatalf("create supplier: %v", err)
	}
	t.Cleanup(func() {
		_ = store.DeleteMaster(context.Background(), supplierForm, supplierID, user)
	})

	customerID, err := store.SaveMaster(ctx, customerForm, 0, map[string]string{
		"code":      "CUS-" + suffix,
		"company":   "Integration Customer " + suffix,
		"lastname":  "Customer",
		"firstname": "Integration",
	}, user)
	if err != nil {
		t.Fatalf("create customer: %v", err)
	}
	t.Cleanup(func() {
		_ = store.DeleteMaster(context.Background(), customerForm, customerID, user)
	})

	stockID, err := store.SaveMaster(ctx, stockForm, 0, map[string]string{
		"code":          "STK-" + suffix,
		"name":          "Integration Stock " + suffix,
		"unit":          "BAG",
		"latest_cost":   "0",
		"min_inventory": "0",
	}, user)
	if err != nil {
		t.Fatalf("create stock: %v", err)
	}
	t.Cleanup(func() {
		_ = store.DeleteMaster(context.Background(), stockForm, stockID, user)
	})

	input := DocumentInput{
		Kind: "purchases",
		User: user,
		Values: map[string]string{
			"entry_date":    "2026-06-24",
			"purchase_date": "2026-06-24",
			"branch_id":     fmt.Sprint(branchID),
			"party_id":      fmt.Sprint(supplierID),
			"reference":     "PO-" + suffix,
			"or_ci_number":  "DR-" + suffix,
			"cash":          "false",
			"remarks":       "created by repository integration test",
		},
		LineInput: []LineInput{{
			Group: "details",
			Rows: []map[string]string{{
				"stock_id":  fmt.Sprint(stockID),
				"qty":       "3",
				"unit_cost": "12.50",
				"amount":    "37.50",
			}},
		}},
	}
	documentID, err := store.SaveDocument(ctx, purchaseForm, 0, input)
	if err != nil {
		t.Fatalf("create purchase document: %v", err)
	}
	t.Cleanup(func() {
		_ = store.DeleteDocument(context.Background(), purchaseForm, documentID, user)
	})

	assertNumericText(t, pool, `select coalesce(sum(qty_delta), 0)::text from stock_ledger where document_id=$1 and stock_id=$2`, "3.00", documentID, stockID)
	assertNumericText(t, pool, `select coalesce(balance, 0)::text from suppliers where id=$1`, "37.50", supplierID)

	input.Values["remarks"] = "updated by repository integration test"
	input.LineInput[0].Rows[0]["qty"] = "4"
	input.LineInput[0].Rows[0]["amount"] = "50.00"
	if _, err := store.SaveDocument(ctx, purchaseForm, documentID, input); err != nil {
		t.Fatalf("update purchase document: %v", err)
	}

	values, groups, err := store.GetDocument(ctx, purchaseForm, documentID)
	if err != nil {
		t.Fatalf("get updated purchase document: %v", err)
	}
	if values["remarks"] != "updated by repository integration test" {
		t.Fatalf("updated purchase remarks = %q", values["remarks"])
	}
	if len(groups["details"]) != 1 || groups["details"][0]["qty"] != "4" {
		t.Fatalf("updated purchase details = %#v", groups["details"])
	}
	assertNumericText(t, pool, `select coalesce(sum(qty_delta), 0)::text from stock_ledger where document_id=$1 and stock_id=$2`, "4.00", documentID, stockID)
	assertNumericText(t, pool, `select coalesce(balance, 0)::text from suppliers where id=$1`, "50.00", supplierID)

	if err := store.DeleteDocument(ctx, purchaseForm, documentID, user); err != nil {
		t.Fatalf("delete purchase document: %v", err)
	}
	t.Cleanup(func() {})
	if _, _, err := store.GetDocument(ctx, purchaseForm, documentID); err == nil {
		t.Fatalf("deleted purchase document was still readable")
	} else if err != pgx.ErrNoRows {
		t.Fatalf("get deleted purchase document err = %v, want pgx.ErrNoRows", err)
	}
	assertNumericText(t, pool, `select coalesce(sum(qty_delta), 0)::text from stock_ledger where document_id=$1 and stock_id=$2`, "0", documentID, stockID)
	assertNumericText(t, pool, `select coalesce(balance, 0)::text from suppliers where id=$1`, "0.00", supplierID)

	stockOutInput := DocumentInput{
		Kind: "dr",
		User: user,
		Values: map[string]string{
			"entry_date": "2026-06-25",
			"sales_date": "2026-06-25",
			"branch_id":  fmt.Sprint(branchID),
			"party_id":   fmt.Sprint(customerID),
			"reference":  "SO-" + suffix,
			"remarks":    "stock out for sales integration test",
		},
		LineInput: []LineInput{{
			Group: "details",
			Rows: []map[string]string{{
				"stock_id": fmt.Sprint(stockID),
				"qty":      "5",
			}},
		}},
	}
	stockOutID, err := store.SaveDocument(ctx, stockOutForm, 0, stockOutInput)
	if err != nil {
		t.Fatalf("create stock out document: %v", err)
	}
	t.Cleanup(func() {
		_ = store.DeleteDocument(context.Background(), stockOutForm, stockOutID, user)
	})

	selection, err := store.LoadDRSelection(ctx, stockOutID)
	if err != nil {
		t.Fatalf("load stock out selection: %v", err)
	}
	if selection.Values["dr_document_id"] != fmt.Sprint(stockOutID) || selection.Values["party_id"] != fmt.Sprint(customerID) {
		t.Fatalf("stock out selection values = %#v", selection.Values)
	}
	if len(selection.Rows) != 1 || selection.Rows[0]["stock_id"] != fmt.Sprint(stockID) || selection.Rows[0]["qty"] != "5" {
		t.Fatalf("stock out selection rows = %#v", selection.Rows)
	}

	salesRow := map[string]string{}
	for key, value := range selection.Rows[0] {
		salesRow[key] = value
	}
	salesRow["unit_cost"] = "12.50"
	salesRow["capital"] = "12.50"
	salesInput := DocumentInput{
		Kind: "sales",
		User: user,
		Values: map[string]string{
			"entry_date":     "2026-06-25",
			"sales_date":     "2026-06-25",
			"branch_id":      fmt.Sprint(branchID),
			"party_id":       fmt.Sprint(customerID),
			"dr_document_id": fmt.Sprint(stockOutID),
			"or_ci_number":   "CI-" + suffix,
			"cash":           "false",
			"remarks":        "sales integration test from stock out",
		},
		LineInput: []LineInput{{Group: "details", Rows: []map[string]string{salesRow}}},
	}
	salesID, err := store.SaveDocument(ctx, salesForm, 0, salesInput)
	if err != nil {
		t.Fatalf("create sales document from stock out: %v", err)
	}
	t.Cleanup(func() {
		_ = store.DeleteDocument(context.Background(), salesForm, salesID, user)
	})

	assertNumericText(t, pool, `select coalesce(sum(consumed_qty), 0)::text from dr_consumptions where consumer_document_id=$1 and dr_document_id=$2`, "5.00", salesID, stockOutID)
	assertNumericText(t, pool, `select coalesce(sum(qty_delta), 0)::text from stock_ledger where document_id=$1 and stock_id=$2`, "-5.00", salesID, stockID)
	assertNumericText(t, pool, `select coalesce(balance, 0)::text from customers where id=$1`, "62.50", customerID)

	if err := store.DeleteDocument(ctx, salesForm, salesID, user); err != nil {
		t.Fatalf("delete sales document: %v", err)
	}
	assertNumericText(t, pool, `select coalesce(sum(consumed_qty), 0)::text from dr_consumptions where consumer_document_id=$1 and dr_document_id=$2`, "0", salesID, stockOutID)
	assertNumericText(t, pool, `select coalesce(sum(qty_delta), 0)::text from stock_ledger where document_id=$1 and stock_id=$2`, "0", salesID, stockID)
	assertNumericText(t, pool, `select coalesce(balance, 0)::text from customers where id=$1`, "0.00", customerID)

	transferSelection, err := store.LoadDRSelection(ctx, stockOutID)
	if err != nil {
		t.Fatalf("reload stock out selection for stock transaction: %v", err)
	}
	transferRow := map[string]string{}
	for key, value := range transferSelection.Rows[0] {
		transferRow[key] = value
	}
	transferRow["unit_cost"] = "12.50"
	transferRow["capital"] = "12.50"
	transferRow["amount"] = "62.50"
	transferInput := DocumentInput{
		Kind: "stock-transactions",
		User: user,
		Values: map[string]string{
			"entry_date":      "2026-06-25",
			"transfer_date":   "2026-06-25",
			"branch_id":       fmt.Sprint(branchID),
			"branch_location": fmt.Sprint(branchID),
			"dr_document_id":  fmt.Sprint(stockOutID),
			"transfer_id":     "ST-" + suffix,
			"transaction":     "0 - Stock Transfer",
			"remarks":         "stock transaction integration test from stock out",
		},
		LineInput: []LineInput{{Group: "details", Rows: []map[string]string{transferRow}}},
	}
	transferID, err := store.SaveDocument(ctx, stockTransactionForm, 0, transferInput)
	if err != nil {
		t.Fatalf("create stock transaction document from stock out: %v", err)
	}
	t.Cleanup(func() {
		_ = store.DeleteDocument(context.Background(), stockTransactionForm, transferID, user)
	})
	assertNumericText(t, pool, `select coalesce(sum(consumed_qty), 0)::text from dr_consumptions where consumer_document_id=$1 and dr_document_id=$2`, "5.00", transferID, stockOutID)
	assertNumericText(t, pool, `select coalesce(sum(qty_delta), 0)::text from stock_ledger where document_id=$1 and stock_id=$2`, "-5.00", transferID, stockID)

	if err := store.DeleteDocument(ctx, stockTransactionForm, transferID, user); err != nil {
		t.Fatalf("delete stock transaction document: %v", err)
	}
	assertNumericText(t, pool, `select coalesce(sum(consumed_qty), 0)::text from dr_consumptions where consumer_document_id=$1 and dr_document_id=$2`, "0", transferID, stockOutID)
	assertNumericText(t, pool, `select coalesce(sum(qty_delta), 0)::text from stock_ledger where document_id=$1 and stock_id=$2`, "0", transferID, stockID)

	if err := store.DeleteDocument(ctx, stockOutForm, stockOutID, user); err != nil {
		t.Fatalf("delete stock out document: %v", err)
	}
	if _, _, err := store.GetDocument(ctx, stockOutForm, stockOutID); err == nil {
		t.Fatalf("deleted stock out document was still readable")
	} else if err != pgx.ErrNoRows {
		t.Fatalf("get deleted stock out document err = %v, want pgx.ErrNoRows", err)
	}

	if err := store.DeleteMaster(ctx, stockForm, stockID, user); err != nil {
		t.Fatalf("delete stock: %v", err)
	}
	if _, err := store.GetMaster(ctx, stockForm, stockID); err == nil {
		t.Fatalf("deleted stock was still readable")
	} else if err != pgx.ErrNoRows {
		t.Fatalf("get deleted stock err = %v, want pgx.ErrNoRows", err)
	}
}

func TestPostgresStoreIncomeStatementRowsQuery(t *testing.T) {
	databaseURL := os.Getenv("CIMS_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set CIMS_TEST_DATABASE_URL to run database-backed income statement query coverage")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	migrationsDir := filepath.Join("..", "..", "db", "migrations")
	if err := appdb.Migrate(databaseURL, migrationsDir); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open postgres pool: %v", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping postgres: %v", err)
	}

	store := NewPostgresStore(pool)
	from := time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, time.June, 30, 0, 0, 0, 0, time.UTC)
	if _, err := store.IncomeStatementRows(ctx, from, to); err != nil {
		t.Fatalf("income statement rows: %v", err)
	}
}

func masterFormByKind(t *testing.T, kind string) models.FormDefinition {
	t.Helper()
	for _, form := range models.MasterForms() {
		if form.Kind == kind {
			return form
		}
	}
	t.Fatalf("master form %q not found", kind)
	return models.FormDefinition{}
}

func transactionFormByKind(t *testing.T, kind string) models.FormDefinition {
	t.Helper()
	for _, form := range models.TransactionForms() {
		if form.Kind == kind {
			return form
		}
	}
	t.Fatalf("transaction form %q not found", kind)
	return models.FormDefinition{}
}

func assertNumericText(t *testing.T, pool *pgxpool.Pool, query, want string, args ...any) {
	t.Helper()
	var got string
	if err := pool.QueryRow(context.Background(), query, args...).Scan(&got); err != nil {
		t.Fatalf("query numeric value: %v", err)
	}
	if got != want {
		t.Fatalf("numeric value = %q, want %q", got, want)
	}
}
