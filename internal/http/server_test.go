package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"cims-go/internal/auth"
	"cims-go/internal/models"
	"cims-go/internal/repositories"
)

type fakeStore struct {
	user models.User
	dr   repositories.DRSelection
}

func (s fakeStore) EnsureAdmin(context.Context, string, string) error { return nil }

func (s fakeStore) GetUserByUsername(context.Context, string) (models.User, error) {
	return s.user, nil
}

func (s fakeStore) GetUserByID(context.Context, int64) (models.User, error) {
	return s.user, nil
}

func (s fakeStore) ListMaster(context.Context, models.FormDefinition) ([]models.Record, error) {
	return []models.Record{{"id": "1", "code": "BR-01", "name": "Main", "encoder": "Admin"}}, nil
}

func (s fakeStore) GetMaster(context.Context, models.FormDefinition, int64) (models.Record, error) {
	return models.Record{"id": "1", "code": "BR-01", "name": "Main"}, nil
}

func (s fakeStore) SaveMaster(context.Context, models.FormDefinition, int64, map[string]string, models.User) (int64, error) {
	return 1, nil
}

func (s fakeStore) ListDocuments(context.Context, string) ([]models.DocumentListItem, error) {
	return nil, nil
}

func (s fakeStore) LoadDRSelection(context.Context, int64) (repositories.DRSelection, error) {
	return s.dr, nil
}

func (s fakeStore) SaveDocument(context.Context, models.FormDefinition, repositories.DocumentInput) (int64, error) {
	return 1, nil
}

func (s fakeStore) Options(context.Context, string) ([]models.Option, error) {
	return []models.Option{{Value: "1", Label: "Main"}}, nil
}

func TestDashboardRequiresLogin(t *testing.T) {
	store := fakeStore{user: models.User{ID: 1, Username: "admin", DisplayName: "Admin", Role: models.RoleAdmin}}
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

func TestMasterListRendersForLoggedInUser(t *testing.T) {
	store := fakeStore{user: models.User{ID: 1, Username: "admin", DisplayName: "Admin", Role: models.RoleAdmin}}
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
}

func TestSalesFormLoadsSelectedDRRows(t *testing.T) {
	store := fakeStore{
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
}
