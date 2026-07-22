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

func TestInsufficientStockErrorIdentifiesStockAndQuantities(t *testing.T) {
	err := insufficientStockError(189, "GALLIMAX 1", "GALLIMAX 1 BOOSTER CRUMBLE, 50KG", 10000, 4000)
	want := "insufficient stock on hand for GALLIMAX 1 - GALLIMAX 1 BOOSTER CRUMBLE, 50KG (requested: 10, available: 4)"
	if err.Error() != want {
		t.Fatalf("error = %q, want %q", err, want)
	}
}

func TestStockTransactionRequiresStockOutLineReferences(t *testing.T) {
	input := totalsInput{net: 1000}
	groups := []LineInput{{Group: "details", Rows: []map[string]string{{
		"stock_id": "1", "qty": "1", "unit_cost": "1", "amount": "1",
	}}}}
	err := validateDocumentInput("stock-transactions", map[string]string{
		"dr_document_id": "7", "transaction": "0 - Stock Transfer", "branch_location": "2",
	}, groups, input)
	if err == nil || !strings.Contains(err.Error(), "come from the selected Stock Out File") {
		t.Fatalf("validation error = %v, want Stock Out File line requirement", err)
	}
}

func TestParseStockOutPartyDistinguishesOverlappingIDs(t *testing.T) {
	tests := []struct {
		value     string
		wantType  string
		wantParty int64
	}{
		{value: "customer:7", wantType: "customer", wantParty: 7},
		{value: "branch:7", wantType: "branch", wantParty: 7},
		{value: "7", wantType: "customer", wantParty: 7},
		{value: "supplier:7", wantType: "", wantParty: 0},
	}
	for _, test := range tests {
		gotType, gotParty := parseStockOutParty(test.value)
		if gotType != test.wantType || gotParty != test.wantParty {
			t.Errorf("parseStockOutParty(%q) = (%q, %d), want (%q, %d)", test.value, gotType, gotParty, test.wantType, test.wantParty)
		}
	}
}

func TestSalesAndStockTransferRejectZeroDetailAmounts(t *testing.T) {
	tests := []struct {
		name   string
		kind   string
		values map[string]string
	}{
		{
			name: "sales",
			kind: "sales",
			values: map[string]string{
				"dr_document_id": "7", "party_id": "3",
			},
		},
		{
			name: "stock transfer",
			kind: "stock-transactions",
			values: map[string]string{
				"dr_document_id": "7", "transaction": "0 - Stock Transfer", "branch_location": "2",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			groups := []LineInput{{Group: "details", Rows: []map[string]string{{
				"dr_line_id": "11", "stock_id": "1", "qty": "1", "unit_cost": "1", "amount": "0",
			}}}}

			err := validateDocumentInput(test.kind, test.values, groups, totalsInput{net: 1000})
			if err == nil || !strings.Contains(err.Error(), "amount greater than zero") {
				t.Fatalf("validation error = %v, want positive detail amount requirement", err)
			}
		})
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

func TestDocumentReferenceUsesORCINumberForSalesAndPurchases(t *testing.T) {
	for _, kind := range []string{"sales", "purchases"} {
		t.Run(kind, func(t *testing.T) {
			values := map[string]string{"reference": "legacy-reference", "or_ci_number": " CI-0042 "}
			if got := documentReference(kind, values); got != "CI-0042" {
				t.Fatalf("documentReference(%q) = %q, want CI-0042", kind, got)
			}
		})
	}

	if got := documentReference("dr", map[string]string{"reference": " SO-0042 ", "or_ci_number": "CI-0042"}); got != "SO-0042" {
		t.Fatalf("DR document reference = %q, want SO-0042", got)
	}
}

func TestInventoryAvailabilityAllowsMetadataOnlyUpdate(t *testing.T) {
	const previousQuantity = int64(10000)

	if !inventoryAvailableForUpdate(7000, previousQuantity, previousQuantity) {
		t.Fatal("unchanged outbound quantity should remain valid when later activity leaves stock below zero")
	}
	if inventoryAvailableForUpdate(7000, 11000, previousQuantity) {
		t.Fatal("an update must not increase outbound quantity while stock is below zero")
	}
	if !inventoryAvailableForUpdate(15000, 12000, previousQuantity) {
		t.Fatal("an update should allow an increase covered by current stock")
	}
	if inventoryAvailableForUpdate(5000, 6000, 0) {
		t.Fatal("a new document must still reject an outbound quantity above stock on hand")
	}
}

func TestPaymentBalanceAllowsMetadataOnlyUpdate(t *testing.T) {
	const previousPayment = int64(145383500)

	if !paymentWithinAccountBalance(140000000, previousPayment, previousPayment) {
		t.Fatal("an unchanged historical payment should remain valid when later activity lowered the account balance")
	}
	if !paymentWithinAccountBalance(140000000, 144383500, previousPayment) {
		t.Fatal("a reduced historical payment should remain valid")
	}
	if paymentWithinAccountBalance(140000000, 146383500, previousPayment) {
		t.Fatal("an update must not increase a payment beyond the available account balance")
	}
	if !paymentWithinAccountBalance(150000000, 146383500, previousPayment) {
		t.Fatal("an increased payment should be valid when the restored account balance covers it")
	}
	if paymentWithinAccountBalance(140000000, previousPayment, 0) {
		t.Fatal("a new payment must still reject an amount above the account balance")
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
	salesRow["amount"] = "62.50"
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
	if _, err := pool.Exec(ctx, `
		insert into stock_ledger (document_id, branch_id, stock_id, qty_delta, unit_cost)
		values ($1, $2, $3, -1000, 0)`, stockOutID, branchID, stockID); err != nil {
		t.Fatalf("create later stock deficit for metadata-only update coverage: %v", err)
	}
	wantORCI := "CI-UPDATED-" + suffix
	salesInput.Values["or_ci_number"] = wantORCI
	if _, err := store.SaveDocument(ctx, salesForm, salesID, salesInput); err != nil {
		t.Fatalf("update only sales OR/CI number while stock is below zero: %v", err)
	}
	assertNumericText(t, pool, `select coalesce(sum(qty_delta), 0)::text from stock_ledger where document_id=$1 and stock_id=$2`, "-5.00", salesID, stockID)
	var salesReference, salesPayloadORCI string
	if err := pool.QueryRow(ctx, `
		select coalesce(reference, ''), coalesce(payload->'values'->>'or_ci_number', '')
		from documents
		where id=$1`, salesID).Scan(&salesReference, &salesPayloadORCI); err != nil {
		t.Fatalf("read saved sales OR/CI number: %v", err)
	}
	if salesReference != wantORCI || salesPayloadORCI != wantORCI {
		t.Fatalf("saved sales OR/CI reference=%q payload=%q, want %q", salesReference, salesPayloadORCI, wantORCI)
	}
	salesValues, _, err := store.GetDocument(ctx, salesForm, salesID)
	if err != nil {
		t.Fatalf("get saved sales document: %v", err)
	}
	if salesValues["or_ci_number"] != wantORCI {
		t.Fatalf("reloaded sales OR/CI number = %q, want %q", salesValues["or_ci_number"], wantORCI)
	}
	dailySalesRows, err := store.DailySalesCollectionReportRows(ctx, time.Date(2026, time.June, 25, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("read daily sales report rows: %v", err)
	}
	foundDailySalesORCI := false
	for _, row := range dailySalesRows {
		if strings.Contains(row.Reference, wantORCI) {
			foundDailySalesORCI = true
			if strings.Contains(row.Reference, "ENT-") {
				t.Fatalf("daily sales reference %q includes generated Entry ID", row.Reference)
			}
		}
	}
	if !foundDailySalesORCI {
		t.Fatalf("daily sales rows do not include saved OR/CI number %q: %#v", wantORCI, dailySalesRows)
	}

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

func TestPostgresStoreARLedgerRecoversUnpostedARCreditDocument(t *testing.T) {
	databaseURL := os.Getenv("CIMS_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set CIMS_TEST_DATABASE_URL to run database-backed AR ledger recovery coverage")
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

	suffix := fmt.Sprintf("ARLEDGER%d", time.Now().UnixNano())
	var userID, customerID, documentID int64
	if err := pool.QueryRow(ctx, `
		insert into users (username, password_hash, display_name, role)
		values ($1, 'test-hash', 'AR Ledger Recovery Test', 'admin')
		returning id`, "ar-ledger-"+suffix).Scan(&userID); err != nil {
		t.Fatalf("insert test user: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `delete from users where id=$1`, userID)
	})
	if err := pool.QueryRow(ctx, `
		insert into customers (code, company)
		values ($1, $2)
		returning id`, "CUS-"+suffix, "AR Ledger Recovery "+suffix).Scan(&customerID); err != nil {
		t.Fatalf("insert test customer: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `delete from customers where id=$1`, customerID)
	})
	if err := pool.QueryRow(ctx, `
		insert into documents
			(kind, entry_date, document_date, party_type, party_id, reference, net, payload, encoder_user_id, last_update_by_user_id)
		values
			('ar-credit', '2099-12-15 15:30:00+08', '2099-12-15', 'customer', $1::bigint, $2, 0, jsonb_build_object('values', jsonb_build_object('party_id', ($1::bigint)::text, 'cash_amount', '200')), $3, $3)
		returning id`, customerID, "ARC-"+suffix, userID).Scan(&documentID); err != nil {
		t.Fatalf("insert unposted AR credit document: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `delete from documents where id=$1`, documentID)
	})
	if _, err := pool.Exec(ctx, `
		insert into document_lines (document_id, group_key, line_no, amount, payload)
		values ($1, 'checks', 1, 100, '{"amount":"100"}')`, documentID); err != nil {
		t.Fatalf("insert unposted AR credit check: %v", err)
	}

	store := NewPostgresStore(pool)
	to := time.Date(2099, time.December, 15, 0, 0, 0, 0, time.UTC)
	assertRecoveredCredit := func(wantRows int) {
		t.Helper()
		rows, err := store.ARLedgerReportRows(ctx, time.Time{}, to)
		if err != nil {
			t.Fatalf("load AR ledger rows: %v", err)
		}
		matches := 0
		for _, row := range rows {
			if row.CustomerID != fmt.Sprint(customerID) {
				continue
			}
			matches++
			if row.Kind != "ar-credit" || row.Reference != "ARC-"+suffix || row.DeltaCents != -30_000 {
				t.Fatalf("recovered AR credit = %#v, want a -300.00 credit", row)
			}
		}
		if matches != wantRows {
			t.Fatalf("recovered AR credit rows = %d, want %d", matches, wantRows)
		}
	}

	assertRecoveredCredit(1)
	if _, err := pool.Exec(ctx, `
		insert into balance_ledger (document_id, party_type, party_id, amount_delta)
		values ($1, 'customer', $2, -300)`, documentID, customerID); err != nil {
		t.Fatalf("insert normal AR credit posting: %v", err)
	}
	assertRecoveredCredit(1)
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

func TestPostgresStoreStockInAppearsInAcquisitionReports(t *testing.T) {
	databaseURL := os.Getenv("CIMS_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set CIMS_TEST_DATABASE_URL to run database-backed stock-in report coverage")
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

	store := NewPostgresStore(pool)
	suffix := fmt.Sprintf("STOCKIN%d", time.Now().UnixNano())
	reportDate := time.Date(2099, time.December, 31, 0, 0, 0, 0, time.UTC)
	baselineIncome, err := store.IncomeStatementRows(ctx, reportDate, reportDate)
	if err != nil {
		t.Fatalf("load baseline income statement: %v", err)
	}
	baselineStockIn := incomeStatementAmount(baselineIncome, "purchases", "Stock In")

	var userID int64
	if err := pool.QueryRow(ctx, `
		insert into users (username, password_hash, display_name, role)
		values ($1, 'test-hash', 'Stock In Report Test', 'admin')
		returning id`, "stock-in-report-"+suffix).Scan(&userID); err != nil {
		t.Fatalf("insert test user: %v", err)
	}
	user := models.User{ID: userID, Username: "stock-in-report-" + suffix, DisplayName: "Stock In Report Test", Role: models.RoleAdmin}
	defer func() {
		_, _ = pool.Exec(context.Background(), `delete from users where id=$1`, userID)
	}()

	branchForm := masterFormByKind(t, "branches")
	branchID, err := store.SaveMaster(ctx, branchForm, 0, map[string]string{
		"code": "BR-" + suffix, "name": "Stock In Report Branch " + suffix, "incharge": "Tester",
	}, user)
	if err != nil {
		t.Fatalf("create branch: %v", err)
	}
	user.ActiveBranchID = branchID
	defer func() { _ = store.DeleteMaster(context.Background(), branchForm, branchID, user) }()

	stockForm := masterFormByKind(t, "stocks")
	stockID, err := store.SaveMaster(ctx, stockForm, 0, map[string]string{
		"code": "STK-" + suffix, "name": "Stock In Report Item " + suffix, "unit": "BAG", "latest_cost": "0", "min_inventory": "0",
	}, user)
	if err != nil {
		t.Fatalf("create stock: %v", err)
	}
	defer func() { _ = store.DeleteMaster(context.Background(), stockForm, stockID, user) }()

	stockInForm := transactionFormByKind(t, "stock-in")
	documentID, err := store.SaveDocument(ctx, stockInForm, 0, DocumentInput{
		Kind: "stock-in",
		User: user,
		Values: map[string]string{
			"entry_date": "2099-12-31", "branch_id": fmt.Sprint(branchID), "remarks": "stock-in report regression test",
		},
		LineInput: []LineInput{{Group: "details", Rows: []map[string]string{{
			"stock_id": fmt.Sprint(stockID), "qty": "7", "unit_cost": "12.50", "amount": "87.50",
		}}}},
	})
	if err != nil {
		t.Fatalf("create stock-in document: %v", err)
	}
	defer func() { _ = store.DeleteDocument(context.Background(), stockInForm, documentID, user) }()

	var entryID string
	if err := pool.QueryRow(ctx, `select entry_id from documents where id=$1`, documentID).Scan(&entryID); err != nil {
		t.Fatalf("load stock-in entry ID: %v", err)
	}

	summaryRows, err := store.PurchaseReportRows(ctx, reportDate, reportDate)
	if err != nil {
		t.Fatalf("purchase summary rows: %v", err)
	}
	foundSummary := false
	for _, row := range summaryRows {
		if row.EntryID == entryID {
			foundSummary = row.Supplier == "Stock In" && row.Type == "Stock In" && row.NetCents == 8750
		}
	}
	if !foundSummary {
		t.Fatalf("stock-in entry %q missing or mislabeled in purchase summary: %#v", entryID, summaryRows)
	}

	byReference, err := store.PurchaseByDRNumberReportRows(ctx, reportDate, reportDate)
	if err != nil {
		t.Fatalf("purchase-by-reference rows: %v", err)
	}
	foundReference := false
	for _, row := range byReference {
		if row.Reference == entryID && row.StockCode == "STK-"+suffix {
			foundReference = row.Supplier == "Stock In" && row.Type == "Stock In" && row.Quantity == 7
		}
	}
	if !foundReference {
		t.Fatalf("stock-in entry %q missing or mislabeled in purchase-by-reference report: %#v", entryID, byReference)
	}

	byStock, err := store.PurchaseByStockCodeReportRows(ctx, reportDate, reportDate)
	if err != nil {
		t.Fatalf("purchase-by-stock rows: %v", err)
	}
	foundStock := false
	for _, row := range byStock {
		if row.Reference == entryID && row.StockCode == "STK-"+suffix {
			foundStock = row.Supplier == "Stock In" && row.Type == "Stock In" && row.Quantity == 7
		}
	}
	if !foundStock {
		t.Fatalf("stock-in entry %q missing or mislabeled in purchase-by-stock report: %#v", entryID, byStock)
	}

	bySupplier, err := store.PurchaseBySupplierReportRows(ctx, reportDate, reportDate)
	if err != nil {
		t.Fatalf("purchase-by-supplier rows: %v", err)
	}
	foundSupplier := false
	for _, row := range bySupplier {
		if row.Reference == entryID && row.StockCode == "STK-"+suffix {
			foundSupplier = row.Supplier == "Stock In" && row.Type == "Stock In" && row.Quantity == 7
		}
	}
	if !foundSupplier {
		t.Fatalf("stock-in entry %q missing or mislabeled in purchase-by-supplier report: %#v", entryID, bySupplier)
	}

	ledgerRows, err := store.StockLedgerReportRows(ctx, reportDate)
	if err != nil {
		t.Fatalf("stock ledger rows: %v", err)
	}
	foundLedger := false
	for _, row := range ledgerRows {
		if row.StockID == fmt.Sprint(stockID) && row.Kind == "stock-in" && row.Reference == entryID && row.QtyDelta == 7 {
			foundLedger = true
		}
	}
	if !foundLedger {
		t.Fatalf("stock-in entry %q missing from stock ledger", entryID)
	}

	incomeRows, err := store.IncomeStatementRows(ctx, reportDate, reportDate)
	if err != nil {
		t.Fatalf("income statement rows: %v", err)
	}
	if got := incomeStatementAmount(incomeRows, "purchases", "Stock In") - baselineStockIn; got != 8750 {
		t.Fatalf("stock-in purchases amount delta = %d, want 8750", got)
	}
}

func TestPostgresStoreStockOutAppearsInSalesActivityReports(t *testing.T) {
	databaseURL := os.Getenv("CIMS_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set CIMS_TEST_DATABASE_URL to run database-backed stock-out report coverage")
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

	store := NewPostgresStore(pool)
	suffix := fmt.Sprintf("STOCKOUT%d", time.Now().UnixNano())
	var userID int64
	if err := pool.QueryRow(ctx, `
		insert into users (username, password_hash, display_name, role)
		values ($1, 'test-hash', 'Stock Out Report Test', 'admin')
		returning id`, "stock-out-report-"+suffix).Scan(&userID); err != nil {
		t.Fatalf("insert test user: %v", err)
	}
	user := models.User{ID: userID, Username: "stock-out-report-" + suffix, DisplayName: "Stock Out Report Test", Role: models.RoleAdmin}
	defer func() { _, _ = pool.Exec(context.Background(), `delete from users where id=$1`, userID) }()

	branchForm := masterFormByKind(t, "branches")
	branchID, err := store.SaveMaster(ctx, branchForm, 0, map[string]string{
		"code": "BR-" + suffix, "name": "Stock Out Report Branch " + suffix, "incharge": "Tester",
	}, user)
	if err != nil {
		t.Fatalf("create branch: %v", err)
	}
	user.ActiveBranchID = branchID
	defer func() { _ = store.DeleteMaster(context.Background(), branchForm, branchID, user) }()

	stockForm := masterFormByKind(t, "stocks")
	stockID, err := store.SaveMaster(ctx, stockForm, 0, map[string]string{
		"code": "STK-" + suffix, "name": "Stock Out Report Item " + suffix, "unit": "BAG", "latest_cost": "0", "min_inventory": "0",
	}, user)
	if err != nil {
		t.Fatalf("create stock: %v", err)
	}
	defer func() { _ = store.DeleteMaster(context.Background(), stockForm, stockID, user) }()

	stockInForm := transactionFormByKind(t, "stock-in")
	stockInID, err := store.SaveDocument(ctx, stockInForm, 0, DocumentInput{
		Kind: "stock-in",
		User: user,
		Values: map[string]string{
			"entry_date": "2099-12-29", "branch_id": fmt.Sprint(branchID), "remarks": "stock for stock-out report test",
		},
		LineInput: []LineInput{{Group: "details", Rows: []map[string]string{{
			"stock_id": fmt.Sprint(stockID), "qty": "10", "unit_cost": "12.50", "amount": "125.00",
		}}}},
	})
	if err != nil {
		t.Fatalf("create supporting stock-in document: %v", err)
	}
	defer func() { _ = store.DeleteDocument(context.Background(), stockInForm, stockInID, user) }()

	reportDate := time.Date(2099, time.December, 30, 0, 0, 0, 0, time.UTC)
	baselineIncome, err := store.IncomeStatementRows(ctx, reportDate, reportDate)
	if err != nil {
		t.Fatalf("load baseline income statement: %v", err)
	}
	baselineStockOut := incomeStatementAmount(baselineIncome, "withdrawals", "Stock Out")

	stockOutForm := transactionFormByKind(t, "stock-out")
	stockOutID, err := store.SaveDocument(ctx, stockOutForm, 0, DocumentInput{
		Kind: "stock-out",
		User: user,
		Values: map[string]string{
			"entry_date": "2099-12-30", "branch_id": fmt.Sprint(branchID), "remarks": "stock-out report regression test",
		},
		LineInput: []LineInput{{Group: "details", Rows: []map[string]string{{
			"stock_id": fmt.Sprint(stockID), "qty": "7", "unit_cost": "12.50", "amount": "87.50",
		}}}},
	})
	if err != nil {
		t.Fatalf("create stock-out document: %v", err)
	}
	defer func() { _ = store.DeleteDocument(context.Background(), stockOutForm, stockOutID, user) }()

	var entryID string
	if err := pool.QueryRow(ctx, `select entry_id from documents where id=$1`, stockOutID).Scan(&entryID); err != nil {
		t.Fatalf("load stock-out entry ID: %v", err)
	}

	summaryRows, err := store.SalesReportRows(ctx, reportDate, reportDate)
	if err != nil {
		t.Fatalf("sales summary rows: %v", err)
	}
	foundSummary := false
	for _, row := range summaryRows {
		if row.EntryID == entryID {
			foundSummary = row.Customer == "Stock Out" && row.Type == "Stock Out" && row.NetCents == 8750
		}
	}
	if !foundSummary {
		t.Fatalf("stock-out entry %q missing or mislabeled in sales summary: %#v", entryID, summaryRows)
	}

	byReference, err := store.SalesByORCIDRNumberReportRows(ctx, reportDate, reportDate)
	if err != nil {
		t.Fatalf("sales-by-reference rows: %v", err)
	}
	foundReference := false
	for _, row := range byReference {
		if row.Reference == entryID && row.StockCode == "STK-"+suffix {
			foundReference = row.Customer == "Stock Out" && row.Type == "Stock Out" && row.Quantity == 7
		}
	}
	if !foundReference {
		t.Fatalf("stock-out entry %q missing or mislabeled in sales-by-reference report: %#v", entryID, byReference)
	}

	byCustomer, err := store.SalesByCustomerReportRows(ctx, reportDate, reportDate)
	if err != nil {
		t.Fatalf("sales-by-customer rows: %v", err)
	}
	foundCustomer := false
	for _, row := range byCustomer {
		if row.Reference == entryID && row.StockCode == "STK-"+suffix {
			foundCustomer = row.Customer == "Stock Out" && row.Type == "Stock Out" && row.Quantity == 7
		}
	}
	if !foundCustomer {
		t.Fatalf("stock-out entry %q missing or mislabeled in sales-by-customer report: %#v", entryID, byCustomer)
	}

	byStock, err := store.SalesByStockNameReportRows(ctx, reportDate, reportDate)
	if err != nil {
		t.Fatalf("sales-by-stock rows: %v", err)
	}
	foundStock := false
	for _, row := range byStock {
		if row.Reference == entryID && row.StockCode == "STK-"+suffix {
			foundStock = row.Customer == "Stock Out" && row.Type == "Stock Out" && row.Quantity == 7
		}
	}
	if !foundStock {
		t.Fatalf("stock-out entry %q missing or mislabeled in sales-by-stock report: %#v", entryID, byStock)
	}

	ledgerRows, err := store.StockLedgerReportRows(ctx, reportDate)
	if err != nil {
		t.Fatalf("stock ledger rows: %v", err)
	}
	foundLedger := false
	for _, row := range ledgerRows {
		if row.StockID == fmt.Sprint(stockID) && row.Kind == "stock-out" && row.Reference == entryID && row.QtyDelta == -7 {
			foundLedger = true
		}
	}
	if !foundLedger {
		t.Fatalf("stock-out entry %q missing from stock ledger", entryID)
	}

	incomeRows, err := store.IncomeStatementRows(ctx, reportDate, reportDate)
	if err != nil {
		t.Fatalf("income statement rows: %v", err)
	}
	if got := incomeStatementAmount(incomeRows, "withdrawals", "Stock Out") - baselineStockOut; got != -8750 {
		t.Fatalf("stock-out withdrawal amount delta = %d, want -8750", got)
	}
}

func incomeStatementAmount(rows []models.IncomeStatementRow, section, label string) int64 {
	var amount int64
	for _, row := range rows {
		if row.Section == section && row.Label == label {
			amount += row.AmountCents
		}
	}
	return amount
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
