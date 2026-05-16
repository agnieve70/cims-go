package http

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
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
	ListMaster(ctx context.Context, form models.FormDefinition) ([]models.Record, error)
	GetMaster(ctx context.Context, form models.FormDefinition, id int64) (models.Record, error)
	SaveMaster(ctx context.Context, form models.FormDefinition, id int64, values map[string]string, user models.User) (int64, error)
	ListDocuments(ctx context.Context, kind string) ([]models.DocumentListItem, error)
	LoadDRSelection(ctx context.Context, id int64) (repositories.DRSelection, error)
	SaveDocument(ctx context.Context, form models.FormDefinition, input repositories.DocumentInput) (int64, error)
	Options(ctx context.Context, source string) ([]models.Option, error)
}

type App struct {
	store     Store
	auth      *auth.Manager
	templates *template.Template
	now       func() time.Time
}

func NewApp(store Store, authManager *auth.Manager) (*App, error) {
	tmpl, err := parseTemplates()
	if err != nil {
		return nil, err
	}
	return &App{store: store, auth: authManager, templates: tmpl, now: time.Now}, nil
}

func parseTemplates() (*template.Template, error) {
	funcs := template.FuncMap{
		"fieldValue": fieldValue,
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
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(a.auth.LoadUser)

	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	r.Get("/login", a.loginForm)
	r.Post("/login", a.loginPost)
	r.Post("/logout", a.logout)

	r.Group(func(protected chi.Router) {
		protected.Use(a.auth.RequireLogin)
		protected.Get("/", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
		})
		protected.Get("/dashboard", a.dashboard)
		protected.Get("/line-row/{group}", a.lineRow)

		protected.Route("/masters/{kind}", func(cr chi.Router) {
			cr.Get("/", a.masterList)
			cr.Get("/new", a.masterForm)
			cr.Post("/", a.masterCreate)
			cr.Get("/{id}/edit", a.masterEdit)
			cr.Post("/{id}", a.masterUpdate)
		})

		protected.Route("/transactions/{kind}", func(cr chi.Router) {
			cr.Get("/", a.transactionList)
			cr.Get("/new", a.transactionForm)
			cr.Post("/", a.transactionCreate)
		})
	})

	return r
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
	records, err := a.store.ListMaster(r.Context(), form)
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	a.render(w, r, "list.gohtml", viewData{Title: form.Title, Form: form, Records: records, IsMaster: true})
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
	if _, err := a.store.SaveMaster(r.Context(), form, id, values, user); err != nil {
		a.renderFormError(w, r, form, values, nil, err)
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
	items, err := a.store.ListDocuments(r.Context(), form.Kind)
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	a.render(w, r, "transaction_list.gohtml", viewData{Title: form.Title, Form: form, Documents: items})
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

func (a *App) transactionCreate(w http.ResponseWriter, r *http.Request) {
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
	if _, err := a.store.SaveDocument(r.Context(), form, repositories.DocumentInput{
		Kind:      form.Kind,
		Values:    values,
		LineInput: lines,
		User:      user,
	}); err != nil {
		a.renderFormError(w, r, form, values, lineRowsFromInput(lines), err)
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
	data := viewData{
		Title:    form.Title,
		Form:     form,
		Record:   record,
		Action:   action,
		Options:  a.optionsForForm(r.Context(), form),
		LineRows: lineRows,
	}
	a.addFormBackdrop(r.Context(), &data)
	a.render(w, r, "form.gohtml", data)
}

func (a *App) renderFormError(w http.ResponseWriter, r *http.Request, form models.FormDefinition, values models.Record, lineRows map[string][]models.Record, err error) {
	w.WriteHeader(http.StatusBadRequest)
	data := viewData{
		Title:    form.Title,
		Form:     form,
		Record:   values,
		Action:   r.URL.Path,
		Error:    err.Error(),
		Options:  a.optionsForForm(r.Context(), form),
		LineRows: lineRows,
	}
	a.addFormBackdrop(r.Context(), &data)
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

func (a *App) addFormBackdrop(ctx context.Context, data *viewData) {
	if data.Form.Table != "" {
		records, err := a.store.ListMaster(ctx, data.Form)
		if err == nil {
			data.Records = records
		}
		return
	}
	docs, err := a.store.ListDocuments(ctx, data.Form.Kind)
	if err == nil {
		data.Documents = docs
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
	options, err := a.store.Options(ctx, source)
	if err != nil {
		return nil
	}
	return options
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

type viewData struct {
	Title            string
	Error            string
	User             models.User
	MasterForms      []models.FormDefinition
	TransactionForms []models.FormDefinition
	Form             models.FormDefinition
	Records          []models.Record
	Documents        []models.DocumentListItem
	Record           models.Record
	Action           string
	IsMaster         bool
	Options          map[string][]models.Option
	LineGroup        models.LineGroup
	LineRows         map[string][]models.Record
}

func (a *App) render(w http.ResponseWriter, r *http.Request, name string, data viewData) {
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
