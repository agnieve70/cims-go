package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"cims-go/internal/auth"
	"cims-go/internal/models"
)

func TestTransactionDetailTablesShareEscapeDeleteBehavior(t *testing.T) {
	store := &fakeStore{user: models.User{ID: 1, Username: "admin", DisplayName: "Admin", Role: models.RoleAdmin}}
	manager := auth.NewManager(store, "12345678901234567890123456789012", "1234567890123456")
	app, err := NewApp(store, manager)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/transactions/purchases/197/edit", nil)
	req = req.WithContext(auth.WithUser(req.Context(), store.user))
	rec := httptest.NewRecorder()
	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	for _, want := range []string{
		`return !!getEditableGridRowSelector(row);`,
		`row.classList.contains("purchase-adjustment-row")`,
		`hydratePurchaseAdjustments();`,
		`row.classList.contains("purchase-payment-row")`,
		`hydratePurchasePaymentRows();`,
		`row.classList.contains("checks-in-detail-row")`,
		`var focusRow = rows[index - 1] || rows[index + 1] || row;`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("body missing shared transaction detail delete behavior %q", want)
		}
	}
}

func TestAllTransactionListsShowEditAndDeleteActions(t *testing.T) {
	for _, form := range models.TransactionForms() {
		t.Run(form.Kind, func(t *testing.T) {
			store := &fakeStore{user: models.User{ID: 1, Username: "admin", DisplayName: "Admin", Role: models.RoleAdmin}}
			manager := auth.NewManager(store, "12345678901234567890123456789012", "1234567890123456")
			app, err := NewApp(store, manager)
			if err != nil {
				t.Fatal(err)
			}

			req := httptest.NewRequest(http.MethodGet, form.RouteBase+"/", nil)
			req = req.WithContext(auth.WithUser(req.Context(), store.user))
			rec := httptest.NewRecorder()
			app.Routes().ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
			}
			body := rec.Body.String()
			for _, want := range []string{
				`class="transaction-actions-column">Actions</th>`,
				`href="` + form.RouteBase + `/1/edit"`,
				`action="` + form.RouteBase + `/1/delete"`,
				`aria-label="Edit ENT-1"`,
				`aria-label="Delete ENT-1"`,
			} {
				if !strings.Contains(body, want) {
					t.Fatalf("body missing transaction action %q", want)
				}
			}
		})
	}
}
