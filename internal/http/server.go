package http

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"cims-go/internal/auth"
	"cims-go/internal/models"
	"cims-go/internal/repositories"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Store interface {
	EnsureAdmin(ctx context.Context, username, password string) error
	GetUserByUsername(ctx context.Context, username string) (models.User, error)
	GetUserByID(ctx context.Context, id int64) (models.User, error)
	ListMaster(ctx context.Context, form models.FormDefinition, search string, year int) ([]models.Record, error)
	GetMaster(ctx context.Context, form models.FormDefinition, id int64) (models.Record, error)
	SaveMaster(ctx context.Context, form models.FormDefinition, id int64, values map[string]string, user models.User) (int64, error)
	DeleteMaster(ctx context.Context, form models.FormDefinition, id int64, user models.User) error
	ListDocuments(ctx context.Context, kind string, search string, year int) ([]models.DocumentListItem, error)
	GetDocument(ctx context.Context, form models.FormDefinition, id int64) (models.Record, map[string][]models.Record, error)
	LoadDRSelection(ctx context.Context, id int64) (repositories.DRSelection, error)
	SaveDocument(ctx context.Context, form models.FormDefinition, id int64, input repositories.DocumentInput) (int64, error)
	PurchaseReportRows(ctx context.Context, from, to time.Time) ([]models.PurchaseReportRow, error)
	PurchaseByDRNumberReportRows(ctx context.Context, from, to time.Time) ([]models.PurchaseByDRNumberReportRow, error)
	PurchaseByStockCodeReportRows(ctx context.Context, from, to time.Time) ([]models.PurchaseByStockCodeReportRow, error)
	PurchaseBySupplierReportRows(ctx context.Context, from, to time.Time) ([]models.PurchaseBySupplierReportRow, error)
	SalesReportRows(ctx context.Context, from, to time.Time) ([]models.SalesReportRow, error)
	SalesByORCIDRNumberReportRows(ctx context.Context, from, to time.Time) ([]models.SalesByORCIDRNumberReportRow, error)
	SalesMarkupByTransactionReportRows(ctx context.Context, from, to time.Time) ([]models.SalesMarkupByTransactionReportRow, error)
	SalesByCustomerReportRows(ctx context.Context, from, to time.Time) ([]models.SalesByCustomerReportRow, error)
	SalesByStockNameReportRows(ctx context.Context, from, to time.Time) ([]models.SalesByStockNameReportRow, error)
	APLedgerReportRows(ctx context.Context, from, to time.Time) ([]models.APLedgerReportRow, error)
	ARLedgerReportRows(ctx context.Context, from, to time.Time) ([]models.ARLedgerReportRow, error)
	IncomingCheckReportRows(ctx context.Context, cutoff time.Time) ([]models.IncomingCheckReportRow, error)
	OutgoingCheckReportRows(ctx context.Context, cutoff time.Time) ([]models.OutgoingCheckReportRow, error)
	ExpenseReportRows(ctx context.Context, from, to time.Time) ([]models.ExpenseReportRow, error)
	IncomeStatementRows(ctx context.Context, from, to time.Time) ([]models.IncomeStatementRow, error)
	IncentiveReportRows(ctx context.Context, from, to time.Time) ([]models.IncentiveReportRow, error)
	DailySalesCollectionReportRows(ctx context.Context, reportDate time.Time) ([]models.DailySalesCollectionReportRow, error)
	StockSalesTransferReportRows(ctx context.Context, from, to time.Time) ([]models.StockSalesTransferReportRow, error)
	StockSalesTransferAmountReportRows(ctx context.Context, from, to time.Time) ([]models.StockSalesTransferAmountReportRow, error)
	StockTransferSummaryReportRows(ctx context.Context, from, to time.Time) ([]models.StockTransferSummaryReportRow, error)
	StockTransferByStockNameReportRows(ctx context.Context, from, to time.Time) ([]models.StockTransferByStockNameReportRow, error)
	StockTransferByBranchReportRows(ctx context.Context, from, to time.Time) ([]models.StockTransferByBranchReportRow, error)
	StockTransferByEntryIDReportRows(ctx context.Context, from, to time.Time) ([]models.StockTransferByEntryIDReportRow, error)
	StockTransferSummaryByItemReportRows(ctx context.Context, from, to time.Time) ([]models.StockTransferSummaryByItemReportRow, error)
	StockTransferMarkupByTransactionReportRows(ctx context.Context, from, to time.Time) ([]models.StockTransferMarkupByTransactionReportRow, error)
	StockLedgerReportRows(ctx context.Context, to time.Time) ([]models.StockLedgerReportRow, error)
	StockAgingReportRows(ctx context.Context, cutoff time.Time) ([]models.StockAgingReportRow, error)
	StockReorderPointReportRows(ctx context.Context, cutoff time.Time) ([]models.StockReorderPointReportRow, error)
	StockSummaryReportRows(ctx context.Context, cutoff time.Time) ([]models.StockSummaryReportRow, error)
	Options(ctx context.Context, source string) ([]models.Option, error)
}

type App struct {
	store          Store
	auth           *auth.Manager
	templates      *template.Template
	now            func() time.Time
	requestLogging bool
	optionsMu      sync.RWMutex
	optionsCache   map[string]cachedOptions
}

type cachedOptions struct {
	options []models.Option
	expires time.Time
}

const optionCacheTTL = 30 * time.Second

func NewApp(store Store, authManager *auth.Manager) (*App, error) {
	tmpl, err := parseTemplates()
	if err != nil {
		return nil, err
	}
	return &App{store: store, auth: authManager, templates: tmpl, now: time.Now, requestLogging: true, optionsCache: map[string]cachedOptions{}}, nil
}

func (a *App) SetRequestLogging(enabled bool) {
	a.requestLogging = enabled
}

func parseTemplates() (*template.Template, error) {
	funcs := template.FuncMap{
		"fieldValue": fieldValue,
		"optionCode": optionCode,
		"optionName": optionName,
		"lineName":   lineInputName,
		"dictLine": func(group models.LineGroup, options map[string][]models.Option) viewData {
			return viewData{LineGroup: group, Options: options}
		},
		"dictRow": func(group models.LineGroup, options map[string][]models.Option, row models.Record) viewData {
			return viewData{LineGroup: group, Options: options, Record: row}
		},
		"eq": func(a, b any) bool {
			return fmt.Sprint(a) == fmt.Sprint(b)
		},
		"add": func(a, b int) int {
			return a + b
		},
		"div": func(a, b int) int {
			if b == 0 {
				return 0
			}
			return a / b
		},
		"truthy": func(value string) bool {
			switch strings.TrimSpace(strings.ToLower(value)) {
			case "true", "1", "on", "yes":
				return true
			default:
				return false
			}
		},
	}
	for _, base := range []string{".", "../.."} {
		tmpl, err := template.New("").Funcs(funcs).ParseGlob(filepath.Join(base, "templates", "*.gohtml"))
		if err == nil {
			return tmpl, nil
		}
	}
	return nil, fmt.Errorf("parse templates from templates/*.gohtml")
}

func (a *App) Routes() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	if a.requestLogging {
		r.Use(middleware.Logger)
	}
	r.Use(middleware.Recoverer)
	r.Use(middleware.Compress(5, "text/html", "text/css", "text/javascript", "application/javascript", "image/svg+xml"))

	r.Handle("/static/*", staticHandler())

	r.Group(func(dynamic chi.Router) {
		dynamic.Use(a.auth.LoadUser)
		dynamic.Get("/login", a.loginForm)
		dynamic.Post("/login", a.loginPost)
		dynamic.Post("/logout", a.logout)

		dynamic.Group(func(protected chi.Router) {
			protected.Use(a.auth.RequireLogin)
			protected.Get("/", func(w http.ResponseWriter, r *http.Request) {
				http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
			})
			protected.Get("/dashboard", a.dashboard)
			protected.Get("/line-row/{group}", a.lineRow)
			protected.Get("/reports/purchases-summary", a.purchasesSummaryReport)
			protected.Get("/reports/purchases-by-dr-number", a.purchasesByDRNumberReport)
			protected.Get("/reports/purchases-by-stock-code", a.purchasesByStockCodeReport)
			protected.Get("/reports/purchases-by-supplier", a.purchasesBySupplierReport)
			protected.Get("/reports/sales-summary", a.salesSummaryReport)
			protected.Get("/reports/sales-by-or-ci-dr-number", a.salesByORCIDRNumberReport)
			protected.Get("/reports/sales-markup-by-transaction", a.salesMarkupByTransactionReport)
			protected.Get("/reports/sales-summary-by-item", a.salesSummaryByItemReport)
			protected.Get("/reports/sales-by-customer", a.salesByCustomerReport)
			protected.Get("/reports/sales-by-customer-summary-by-item", a.salesByCustomerSummaryByItemReport)
			protected.Get("/reports/sales-by-stock-name", a.salesByStockNameReport)
			protected.Get("/reports/ap-ledger", a.apLedgerReport)
			protected.Get("/reports/ar-ledger", a.arLedgerReport)
			protected.Get("/reports/incoming-check-list", a.incomingCheckListReport)
			protected.Get("/reports/outgoing-check-list", a.outgoingCheckListReport)
			protected.Get("/reports/expenses-summary", a.expensesSummaryReport)
			protected.Get("/reports/income-statement", a.incomeStatementReport)
			protected.Get("/reports/incentive", a.incentiveReport)
			protected.Get("/reports/daily-sales-collection", a.dailySalesCollectionReport)
			protected.Get("/reports/incoming-check-calendar", a.incomingCheckCalendarReport)
			protected.Get("/reports/daily-due-check", a.dailyDueCheckReport)
			protected.Get("/reports/stock-sales-transfer", a.stockSalesTransferReport)
			protected.Get("/reports/stock-sales-transfer-amount", a.stockSalesTransferAmountReport)
			protected.Get("/reports/transfers-summary", a.stockTransferSummaryReport)
			protected.Get("/reports/transfers-by-stock-name", a.stockTransferByStockNameReport)
			protected.Get("/reports/transfers-by-entry-id", a.stockTransferByEntryIDReport)
			protected.Get("/reports/transfers-summary-by-entry-id", a.stockTransferSummaryByEntryIDReport)
			protected.Get("/reports/transfers-by-branch", a.stockTransferByBranchReport)
			protected.Get("/reports/transfers-summary-by-item", a.stockTransferSummaryByItemReport)
			protected.Get("/reports/transfers-markup-by-transaction", a.stockTransferMarkupByTransactionReport)
			protected.Get("/reports/stock-ledger", a.stockLedgerReport)
			protected.Get("/reports/stock-aging", a.stockAgingReport)
			protected.Get("/reports/stock-reorder-point", a.stockReorderPointReport)
			protected.Get("/reports/stock-summary", a.stockSummaryReport)

			protected.Route("/masters/{kind}", func(cr chi.Router) {
				cr.Get("/", a.masterList)
				cr.Get("/new", a.masterForm)
				cr.Post("/", a.masterCreate)
				cr.Get("/{id}/edit", a.masterEdit)
				cr.Post("/{id}", a.masterUpdate)
				cr.Post("/{id}/delete", a.masterDelete)
			})

			protected.Route("/transactions/{kind}", func(cr chi.Router) {
				cr.Get("/", a.transactionList)
				cr.Get("/new", a.transactionForm)
				cr.Post("/", a.transactionCreate)
				cr.Get("/{id}/edit", a.transactionEdit)
				cr.Post("/{id}", a.transactionUpdate)
			})
		})
	})

	return r
}

func staticHandler() http.Handler {
	staticDir := "static"
	for _, candidate := range []string{"static", filepath.Join("..", "..", "static")} {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			staticDir = candidate
			break
		}
	}
	files := http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir)))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Has("v") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			w.Header().Set("Cache-Control", "public, max-age=604800")
		}
		files.ServeHTTP(w, r)
	})
}

func (a *App) loginForm(w http.ResponseWriter, r *http.Request) {
	a.render(w, r, "login.gohtml", viewData{Title: "Login"})
}

func (a *App) loginPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	_, err := a.auth.Login(r.Context(), w, r.FormValue("username"), r.FormValue("password"))
	if err != nil {
		a.render(w, r, "login.gohtml", viewData{Title: "Login", Error: err.Error()})
		return
	}
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

func (a *App) logout(w http.ResponseWriter, r *http.Request) {
	a.auth.Logout(w)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (a *App) dashboard(w http.ResponseWriter, r *http.Request) {
	a.render(w, r, "dashboard.gohtml", viewData{
		Title:            "Dashboard",
		MasterForms:      models.MasterForms(),
		TransactionForms: models.TransactionForms(),
	})
}

func (a *App) masterList(w http.ResponseWriter, r *http.Request) {
	form, err := masterFormFromRequest(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	search := strings.TrimSpace(r.URL.Query().Get("q"))
	year := listYear(r, a.now)
	records, err := a.store.ListMaster(r.Context(), form, search, year)
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	a.render(w, r, "list.gohtml", viewData{Title: form.Title, Form: form, Records: records, IsMaster: true, Search: search, Year: year})
}

func (a *App) masterForm(w http.ResponseWriter, r *http.Request) {
	form, err := masterFormFromRequest(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	a.renderForm(w, r, form, models.Record{}, "/masters/"+form.Kind, nil)
}

func (a *App) masterEdit(w http.ResponseWriter, r *http.Request) {
	form, err := masterFormFromRequest(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	record, err := a.store.GetMaster(r.Context(), form, id)
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	a.renderForm(w, r, form, record, "/masters/"+form.Kind+"/"+strconv.FormatInt(id, 10), nil)
}

func (a *App) masterCreate(w http.ResponseWriter, r *http.Request) {
	a.saveMaster(w, r, 0)
}

func (a *App) masterUpdate(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	a.saveMaster(w, r, id)
}

func (a *App) masterDelete(w http.ResponseWriter, r *http.Request) {
	form, err := masterFormFromRequest(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	user, _ := auth.CurrentUser(r.Context())
	if err := a.store.DeleteMaster(r.Context(), form, id, user); err != nil {
		record, getErr := a.store.GetMaster(r.Context(), form, id)
		if getErr != nil {
			a.serverError(w, r, err)
			return
		}
		a.renderFormError(w, r, form, record, nil, err)
		return
	}
	a.invalidateOptions()
	http.Redirect(w, r, form.RouteBase+"/", http.StatusSeeOther)
}

func (a *App) saveMaster(w http.ResponseWriter, r *http.Request, id int64) {
	form, err := masterFormFromRequest(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	user, _ := auth.CurrentUser(r.Context())
	values, err := parseValues(r, form.Fields)
	if err != nil {
		a.renderFormError(w, r, form, values, nil, err)
		return
	}
	savedID, err := a.store.SaveMaster(r.Context(), form, id, values, user)
	if err != nil {
		a.renderFormError(w, r, form, values, nil, err)
		return
	}
	a.invalidateOptions()
	if isEmbeddedForm(r) {
		http.Redirect(w, r, withQueryParam(form.RouteBase+"/"+strconv.FormatInt(savedID, 10)+"/edit", "embedded", "1", "saved", "1"), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, form.RouteBase+"/", http.StatusSeeOther)
}

func (a *App) transactionList(w http.ResponseWriter, r *http.Request) {
	form, err := transactionFormFromRequest(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	search := strings.TrimSpace(r.URL.Query().Get("q"))
	year := listYear(r, a.now)
	items, err := a.store.ListDocuments(r.Context(), form.Kind, search, year)
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	a.render(w, r, "transaction_list.gohtml", viewData{Title: form.Title, Form: form, Documents: items, Search: search, Year: year})
}

func (a *App) transactionForm(w http.ResponseWriter, r *http.Request) {
	form, err := transactionFormFromRequest(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	values := models.Record{"entry_date": a.now().Format("2006-01-02")}
	lineRows, err := a.loadTransactionFormRows(r.Context(), form, values, r.URL.Query().Get("dr_document_id"))
	if err != nil {
		a.renderFormError(w, r, form, values, nil, err)
		return
	}
	a.renderForm(w, r, form, values, "/transactions/"+form.Kind, lineRows)
}

func (a *App) transactionEdit(w http.ResponseWriter, r *http.Request) {
	form, err := transactionFormFromRequest(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	values, lineRows, err := a.store.GetDocument(r.Context(), form, id)
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	a.renderForm(w, r, form, values, "/transactions/"+form.Kind+"/"+strconv.FormatInt(id, 10), lineRows)
}

func (a *App) transactionCreate(w http.ResponseWriter, r *http.Request) {
	a.saveTransaction(w, r, 0)
}

func (a *App) transactionUpdate(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	a.saveTransaction(w, r, id)
}

func (a *App) saveTransaction(w http.ResponseWriter, r *http.Request, id int64) {
	form, err := transactionFormFromRequest(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	user, _ := auth.CurrentUser(r.Context())
	values, err := parseValues(r, form.Fields)
	lines := parseLineInputs(r, form.LineGroups)
	if err != nil {
		a.renderFormError(w, r, form, values, lineRowsFromInput(lines), err)
		return
	}
	savedID, err := a.store.SaveDocument(r.Context(), form, id, repositories.DocumentInput{
		Kind:      form.Kind,
		Values:    values,
		LineInput: lines,
		User:      user,
	})
	if err != nil {
		a.renderFormError(w, r, form, values, lineRowsFromInput(lines), err)
		return
	}
	a.invalidateOptions()
	if isEmbeddedForm(r) {
		http.Redirect(w, r, withQueryParam(form.RouteBase+"/"+strconv.FormatInt(savedID, 10)+"/edit", "embedded", "1", "saved", "1"), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, form.RouteBase+"/", http.StatusSeeOther)
}

func (a *App) lineRow(w http.ResponseWriter, r *http.Request) {
	groupKey := chi.URLParam(r, "group")
	for _, form := range models.TransactionForms() {
		for _, group := range form.LineGroups {
			if group.Key == groupKey {
				a.render(w, r, "line_row.gohtml", viewData{LineGroup: group, Options: a.optionsFor(r.Context(), group.Columns)})
				return
			}
		}
	}
	http.NotFound(w, r)
}

func (a *App) renderForm(w http.ResponseWriter, r *http.Request, form models.FormDefinition, record models.Record, action string, lineRows map[string][]models.Record) {
	embedded := isEmbeddedForm(r)
	if embedded {
		action = withQueryParam(action, "embedded", "1")
	}
	data := viewData{
		Title:         form.Title,
		Form:          form,
		Record:        record,
		Action:        action,
		CanDelete:     record["id"] != "",
		Options:       a.optionsForForm(r.Context(), form),
		LineRows:      lineRows,
		Embedded:      embedded,
		EmbeddedSaved: embedded && r.URL.Query().Get("saved") == "1",
	}
	a.addFormBackdrop(r, &data)
	a.render(w, r, "form.gohtml", data)
}

func (a *App) renderFormError(w http.ResponseWriter, r *http.Request, form models.FormDefinition, values models.Record, lineRows map[string][]models.Record, err error) {
	w.WriteHeader(http.StatusBadRequest)
	embedded := isEmbeddedForm(r)
	action := r.URL.Path
	if embedded {
		action = withQueryParam(action, "embedded", "1")
	}
	data := viewData{
		Title:         form.Title,
		Form:          form,
		Record:        values,
		Action:        action,
		Error:         err.Error(),
		CanDelete:     values["id"] != "",
		Options:       a.optionsForForm(r.Context(), form),
		LineRows:      lineRows,
		Embedded:      embedded,
		EmbeddedSaved: embedded && r.URL.Query().Get("saved") == "1",
	}
	a.addFormBackdrop(r, &data)
	a.render(w, r, "form.gohtml", data)
}

func (a *App) loadTransactionFormRows(ctx context.Context, form models.FormDefinition, values models.Record, drIDParam string) (map[string][]models.Record, error) {
	if drIDParam == "" || (form.Kind != "sales" && form.Kind != "stock-transactions") {
		return nil, nil
	}
	drID, err := strconv.ParseInt(drIDParam, 10, 64)
	if err != nil || drID == 0 {
		return nil, errors.New("invalid DR selection")
	}
	selection, err := a.store.LoadDRSelection(ctx, drID)
	if err != nil {
		return nil, err
	}
	values["dr_document_id"] = selection.Values["dr_document_id"]
	if form.Kind == "sales" {
		values["party_id"] = selection.Values["party_id"]
	}
	return map[string][]models.Record{"details": selection.Rows}, nil
}

func (a *App) addFormBackdrop(r *http.Request, data *viewData) {
	search := strings.TrimSpace(r.URL.Query().Get("q"))
	year := listYear(r, a.now)
	data.Search = search
	data.Year = year

	if data.Form.Table != "" {
		records, err := a.store.ListMaster(r.Context(), data.Form, search, year)
		if err == nil {
			data.Records = records
			setRecordNavigation(data, records)
			if data.Record == nil {
				data.Record = models.Record{}
			}
			if data.Record["id"] == "" {
				data.Record["id"] = strconv.FormatInt(maxRecordID(records)+1, 10)
			}
			if user, ok := auth.CurrentUser(r.Context()); ok {
				if data.Record["encoder"] == "" {
					data.Record["encoder"] = user.DisplayName
				}
				if data.Record["updated_by"] == "" {
					data.Record["updated_by"] = user.DisplayName
				}
			}
			if data.Record["last_update"] == "" {
				data.Record["last_update"] = a.now().Format("2006-01-02 03:04:05 PM")
			}
		}
		return
	}
	docs, err := a.store.ListDocuments(r.Context(), data.Form.Kind, search, year)
	if err == nil {
		data.Documents = docs
	}
	if user, ok := auth.CurrentUser(r.Context()); ok {
		if data.Record == nil {
			data.Record = models.Record{}
		}
		if data.Record["updated_by"] == "" {
			data.Record["updated_by"] = user.DisplayName
		}
	}
}

func listYear(r *http.Request, now func() time.Time) int {
	raw := strings.TrimSpace(r.URL.Query().Get("year"))
	if raw == "" {
		return now().Year()
	}
	year, err := strconv.Atoi(raw)
	if err != nil || year < 2000 || year > 2100 {
		return now().Year()
	}
	return year
}

func maxRecordID(records []models.Record) int64 {
	var maxID int64
	for _, record := range records {
		id, err := strconv.ParseInt(strings.TrimSpace(record["id"]), 10, 64)
		if err == nil && id > maxID {
			maxID = id
		}
	}
	return maxID
}

func setRecordNavigation(data *viewData, records []models.Record) {
	if len(records) == 0 {
		return
	}
	ids := make([]int64, 0, len(records))
	for _, record := range records {
		id, err := strconv.ParseInt(strings.TrimSpace(record["id"]), 10, 64)
		if err == nil {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	data.FirstRecordID = strconv.FormatInt(ids[0], 10)
	data.LastRecordID = strconv.FormatInt(ids[len(ids)-1], 10)
	var currentID int64
	if data.Record != nil {
		currentID, _ = strconv.ParseInt(strings.TrimSpace(data.Record["id"]), 10, 64)
	}
	currentIndex := -1
	for i, id := range ids {
		if id == currentID {
			currentIndex = i
			break
		}
	}
	if currentIndex == -1 {
		data.PreviousRecordID = data.FirstRecordID
		data.NextRecordID = data.FirstRecordID
		return
	}
	if currentIndex > 0 {
		data.PreviousRecordID = strconv.FormatInt(ids[currentIndex-1], 10)
	} else {
		data.PreviousRecordID = data.FirstRecordID
	}
	if currentIndex < len(ids)-1 {
		data.NextRecordID = strconv.FormatInt(ids[currentIndex+1], 10)
	} else {
		data.NextRecordID = data.LastRecordID
	}
}

func (a *App) optionsForForm(ctx context.Context, form models.FormDefinition) map[string][]models.Option {
	options := map[string][]models.Option{}
	for _, field := range form.Fields {
		if field.Source != "" {
			options[field.Source] = a.loadOptions(ctx, field.Source)
		}
	}
	for _, group := range form.LineGroups {
		for _, column := range group.Columns {
			if column.Source != "" {
				options[column.Source] = a.loadOptions(ctx, column.Source)
			}
		}
	}
	if form.Kind == "sales" || form.Kind == "stock-transactions" {
		options["stock_groups"] = a.loadOptions(ctx, "stock_groups")
	}
	return options
}

func (a *App) optionsFor(ctx context.Context, columns []models.LineColumn) map[string][]models.Option {
	options := map[string][]models.Option{}
	for _, column := range columns {
		if column.Source != "" {
			options[column.Source] = a.loadOptions(ctx, column.Source)
		}
	}
	return options
}

func (a *App) loadOptions(ctx context.Context, source string) []models.Option {
	now := a.now()
	a.optionsMu.RLock()
	if cached, ok := a.optionsCache[source]; ok && now.Before(cached.expires) {
		a.optionsMu.RUnlock()
		return copyOptions(cached.options)
	}
	a.optionsMu.RUnlock()

	options, err := a.store.Options(ctx, source)
	if err != nil {
		return nil
	}
	a.optionsMu.Lock()
	a.optionsCache[source] = cachedOptions{options: copyOptions(options), expires: now.Add(optionCacheTTL)}
	a.optionsMu.Unlock()
	return options
}

func (a *App) invalidateOptions() {
	a.optionsMu.Lock()
	a.optionsCache = map[string]cachedOptions{}
	a.optionsMu.Unlock()
}

func copyOptions(options []models.Option) []models.Option {
	if len(options) == 0 {
		return nil
	}
	out := make([]models.Option, len(options))
	copy(out, options)
	return out
}

func parseValues(r *http.Request, fields []models.Field) (models.Record, error) {
	if err := r.ParseForm(); err != nil {
		return nil, err
	}
	values := models.Record{}
	for _, field := range fields {
		value := strings.TrimSpace(r.FormValue(field.Key))
		if field.Type == models.FieldBool && value == "" {
			value = "false"
		}
		if field.Required && value == "" {
			return values, fmt.Errorf("%s is required", field.Label)
		}
		values[field.Key] = value
	}
	return values, nil
}

func parseLineInputs(r *http.Request, groups []models.LineGroup) []repositories.LineInput {
	_ = r.ParseForm()
	inputs := make([]repositories.LineInput, 0, len(groups))
	for _, group := range groups {
		maxRows := 0
		for _, column := range group.Columns {
			key := lineInputName(group.Key, column.Key)
			if count := len(r.Form[key]); count > maxRows {
				maxRows = count
			}
		}
		rows := make([]map[string]string, maxRows)
		for i := range rows {
			rows[i] = map[string]string{}
		}
		for _, column := range group.Columns {
			key := lineInputName(group.Key, column.Key)
			for i, value := range r.Form[key] {
				rows[i][column.Key] = strings.TrimSpace(value)
			}
		}
		inputs = append(inputs, repositories.LineInput{Group: group.Key, Rows: rows})
	}
	return inputs
}

func lineRowsFromInput(inputs []repositories.LineInput) map[string][]models.Record {
	rowsByGroup := make(map[string][]models.Record, len(inputs))
	for _, group := range inputs {
		groupRows := make([]models.Record, 0, len(group.Rows))
		for _, row := range group.Rows {
			record := models.Record{}
			for key, value := range row {
				record[key] = value
			}
			groupRows = append(groupRows, record)
		}
		rowsByGroup[group.Group] = groupRows
	}
	return rowsByGroup
}

func lineInputName(group, column string) string {
	return "line_" + group + "_" + column
}

func masterFormFromRequest(r *http.Request) (models.FormDefinition, error) {
	if form, ok := models.FindMaster(chi.URLParam(r, "kind")); ok {
		return form, nil
	}
	return models.FormDefinition{}, errors.New("unknown master form")
}

func transactionFormFromRequest(r *http.Request) (models.FormDefinition, error) {
	if form, ok := models.FindTransaction(chi.URLParam(r, "kind")); ok {
		return form, nil
	}
	return models.FormDefinition{}, errors.New("unknown transaction form")
}

func fieldValue(record models.Record, key string) string {
	if record == nil {
		return ""
	}
	return record[key]
}

func isEmbeddedForm(r *http.Request) bool {
	return r.URL.Query().Get("embedded") == "1"
}

func withQueryParam(path string, keyValues ...string) string {
	if len(keyValues) == 0 {
		return path
	}
	separator := "?"
	if strings.Contains(path, "?") {
		separator = "&"
	}
	var builder strings.Builder
	builder.WriteString(path)
	for index := 0; index+1 < len(keyValues); index += 2 {
		builder.WriteString(separator)
		builder.WriteString(keyValues[index])
		builder.WriteString("=")
		builder.WriteString(keyValues[index+1])
		separator = "&"
	}
	return builder.String()
}

func optionCode(options map[string][]models.Option, source string, id string) string {
	label := optionLabel(options, source, id)
	parts := strings.SplitN(label, " - ", 2)
	return strings.TrimSpace(parts[0])
}

func optionName(options map[string][]models.Option, source string, id string) string {
	label := optionLabel(options, source, id)
	parts := strings.SplitN(label, " - ", 2)
	if len(parts) < 2 {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func optionLabel(options map[string][]models.Option, source string, id string) string {
	if options == nil || id == "" {
		return ""
	}
	for _, option := range options[source] {
		if option.Value == id {
			return option.Label
		}
	}
	return ""
}

type viewData struct {
	Title                              string
	Error                              string
	User                               models.User
	MasterForms                        []models.FormDefinition
	TransactionForms                   []models.FormDefinition
	Form                               models.FormDefinition
	Records                            []models.Record
	Documents                          []models.DocumentListItem
	Record                             models.Record
	Action                             string
	IsMaster                           bool
	CanDelete                          bool
	Embedded                           bool
	EmbeddedSaved                      bool
	Search                             string
	Year                               int
	FirstRecordID                      string
	PreviousRecordID                   string
	NextRecordID                       string
	LastRecordID                       string
	Options                            map[string][]models.Option
	LineGroup                          models.LineGroup
	LineRows                           map[string][]models.Record
	PurchaseReport                     purchaseReportData
	PurchaseByDRReport                 purchaseByDRNumberReportData
	PurchaseByStockReport              purchaseByStockCodeReportData
	PurchaseBySupplierReport           purchaseBySupplierReportData
	SalesReport                        salesReportData
	SalesByORCIDRReport                salesByORCIDRNumberReportData
	SalesMarkupReport                  salesMarkupByTransactionReportData
	SalesSummaryByItemReport           salesSummaryByItemReportData
	SalesByCustomerReport              salesByCustomerReportData
	SalesByCustomerSummaryByItemReport salesByCustomerSummaryByItemReportData
	SalesByStockNameReport             salesByStockNameReportData
	APLedgerReport                     apLedgerReportData
	ARLedgerReport                     arLedgerReportData
	IncomingCheckReport                incomingCheckReportData
	OutgoingCheckReport                outgoingCheckReportData
	ExpenseReport                      expenseReportData
	IncomeStatementReport              incomeStatementReportData
	IncentiveReport                    incentiveReportData
	DailySalesReport                   dailySalesCollectionReportData
	IncomingCheckCalendar              incomingCheckCalendarReportData
	DailyDueCheckReport                dailyDueCheckReportData
	StockSalesTransfer                 stockSalesTransferReportData
	StockSalesTransferAmt              stockSalesTransferAmountReportData
	StockTransferSummaryReport         stockTransferSummaryReportData
	StockTransferByStockNameReport     stockTransferByStockNameReportData
	StockTransferByBranchReport        stockTransferByBranchReportData
	StockTransferByEntryIDReport       stockTransferByEntryIDReportData
	StockTransferSummaryByEntryID      stockTransferSummaryByEntryIDReportData
	StockTransferSummaryByItemReport   stockTransferSummaryByItemReportData
	StockTransferMarkupReport          stockTransferMarkupByTransactionReportData
	StockLedgerReport                  stockLedgerReportData
	StockAgingReport                   stockAgingReportData
	StockReorderReport                 stockReorderPointReportData
	StockSummaryReport                 stockSummaryReportData
}

func (a *App) render(w http.ResponseWriter, r *http.Request, name string, data viewData) {
	if w.Header().Get("Content-Type") == "" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	}
	if user, ok := auth.CurrentUser(r.Context()); ok {
		data.User = user
	}
	if len(data.MasterForms) == 0 {
		data.MasterForms = models.MasterForms()
	}
	if len(data.TransactionForms) == 0 {
		data.TransactionForms = models.TransactionForms()
	}
	if err := a.templates.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (a *App) serverError(w http.ResponseWriter, r *http.Request, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	a.render(w, r, "error.gohtml", viewData{Title: "Error", Error: err.Error()})
}
