package http

import (
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"cims-go/internal/models"
	"cims-go/internal/services"

	"github.com/go-chi/chi/v5"
)

type salesInvoicePage struct {
	Document     models.Record
	Customer     models.Record
	Template     string
	TemplateKind string
	TemplateURL  string
	Date         string
	DateYear     string
	CustomerTIN  string
	Rows         []salesInvoiceLine
	TotalSales   string
	LessVAT      string
	NetOfVAT     string
	Discount     string
	AddVAT       string
	Withholding  string
	AmountDue    string
}

type salesInvoiceLine struct {
	Description string
	Qty         string
	UnitPrice   string
	Amount      string
}

func (a *App) salesInvoicePrint(w http.ResponseWriter, r *http.Request) {
	form, ok := transactionFormByKind("sales")
	if !ok {
		http.NotFound(w, r)
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id == 0 {
		http.NotFound(w, r)
		return
	}

	record, lineRows, err := a.store.GetDocument(r.Context(), form, id)
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	customer := models.Record{}
	if partyID := parseInvoiceInt(record["party_id"]); partyID != 0 {
		if customerForm, ok := masterFormByKind("customers"); ok {
			customer, err = a.store.GetMaster(r.Context(), customerForm, partyID)
			if err != nil {
				a.serverError(w, r, err)
				return
			}
		}
	}

	page := buildSalesInvoicePage(record, customer, lineRows)
	a.render(w, r, "sales_invoice_print.gohtml", viewData{
		Title:        page.Template,
		SalesInvoice: page,
	})
}

func buildSalesInvoicePage(record models.Record, customer models.Record, lineRows map[string][]models.Record) salesInvoicePage {
	page := salesInvoicePage{
		Document:     record,
		Customer:     customer,
		Template:     "Charge Invoice",
		TemplateKind: "charge",
		TemplateURL:  invoiceAssetURL("hs-charge-invoice.png"),
		CustomerTIN:  strings.TrimSpace(customer["tin"]),
	}
	page.Date, page.DateYear = invoiceDateParts(record["sales_date"], record["entry_date"])
	if truthyString(record["cash"]) {
		page.Template = "Sales Invoice"
		page.TemplateKind = "sales"
		page.TemplateURL = invoiceAssetURL("hs-sales-invoice.png")
	}

	details := lineRows["details"]
	page.Rows = make([]salesInvoiceLine, 0, len(details))
	salesDoc := services.SalesDocument{Cash: truthyString(record["cash"])}
	for _, row := range details {
		qty := parseInvoiceInt(row["qty"])
		unitPrice := parseInvoiceMoney(row["unit_cost"])
		amount := parseInvoiceMoney(row["amount"])
		if amount == 0 {
			amount = qty * unitPrice
		}
		discount := parseInvoiceMoney(row["discount"])
		otherDiscount := parseInvoiceMoney(row["other_discount"])
		capital := parseInvoiceMoney(row["capital"])
		if capital == 0 {
			capital = unitPrice
		}
		if qty != 0 || unitPrice != 0 || amount != 0 || strings.TrimSpace(row["stock_label"]) != "" {
			page.Rows = append(page.Rows, salesInvoiceLine{
				Description: invoiceStockDescription(row["stock_label"], row["stock_name"]),
				Qty:         invoiceQty(qty),
				UnitPrice:   invoiceMoney(unitPrice),
				Amount:      invoiceMoney(amount),
			})
			salesDoc.Lines = append(salesDoc.Lines, services.SalesLine{
				Qty:           qty,
				UnitCost:      unitPrice,
				Capital:       capital,
				Discount:      discount,
				OtherDiscount: otherDiscount,
			})
		}
	}
	salesDoc.Deductions = invoiceAdjustments(lineRows["deductions"])
	salesDoc.Additionals = invoiceAdjustments(lineRows["additionals"])
	salesDoc.Payments = invoicePayments(lineRows["payments"])
	totals := services.ComputeSales(salesDoc)
	page.TotalSales = invoiceMoney(totals.TotalNetAmount)
	page.Discount = invoiceMoney(totals.Less)
	page.AddVAT = invoiceMoney(totals.Add)
	page.AmountDue = invoiceMoney(totals.Net)
	page.NetOfVAT = invoiceMoney(totals.TotalNetAmount)
	page.LessVAT = invoiceMoney(0)
	page.Withholding = invoiceMoney(0)
	return page
}

func invoiceAdjustments(rows []models.Record) []services.AdjustmentLine {
	out := make([]services.AdjustmentLine, 0, len(rows))
	for _, row := range rows {
		line := services.AdjustmentLine{
			Particulars: row["particulars"],
			Qty:         parseInvoiceInt(row["qty"]),
			Price:       parseInvoiceMoney(row["price"]),
		}
		if strings.TrimSpace(line.Particulars) != "" || line.Qty != 0 || line.Price != 0 {
			out = append(out, line)
		}
	}
	return out
}

func invoicePayments(rows []models.Record) []services.Payment {
	out := make([]services.Payment, 0, len(rows))
	for _, row := range rows {
		check := parseInvoiceMoney(row["amount"])
		if check != 0 {
			out = append(out, services.Payment{CheckAmount: check})
		}
	}
	return out
}

func invoiceAssetURL(filename string) string {
	url := "/static/invoices/" + filename
	for _, base := range []string{".", "../.."} {
		path := filepath.Join(base, "static", "invoices", filename)
		info, err := os.Stat(path)
		if err == nil {
			return url + "?v=" + strconv.FormatInt(info.ModTime().UnixNano(), 10)
		}
	}
	return url
}

func invoiceDateParts(values ...string) (string, string) {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if parsed, err := time.Parse("2006-01-02", value); err == nil {
			return parsed.Format("01/02"), parsed.Format("06")
		}
		return value, ""
	}
	return "", ""
}

func invoiceStockDescription(stockLabel string, stockName string) string {
	value := strings.TrimSpace(stockLabel)
	if strings.TrimSpace(stockName) != "" {
		value = strings.TrimSpace(stockName)
	}
	if before, after, ok := strings.Cut(value, " - "); ok && strings.TrimSpace(after) != "" {
		return strings.TrimSpace(after) + " (" + strings.TrimSpace(before) + ")"
	}
	return value
}

func invoiceQty(qty int64) string {
	if qty == 0 {
		return ""
	}
	return commaInt(qty)
}

func invoiceMoney(cents int64) string {
	if cents == 0 {
		return ""
	}
	return moneyString(cents)
}

func parseInvoiceInt(value string) int64 {
	value = strings.ReplaceAll(strings.TrimSpace(value), ",", "")
	if value == "" {
		return 0
	}
	if parsed, err := strconv.ParseInt(value, 10, 64); err == nil {
		return parsed
	}
	if parsed, err := strconv.ParseFloat(value, 64); err == nil {
		return int64(math.Round(parsed))
	}
	return 0
}

func parseInvoiceMoney(value string) int64 {
	value = strings.ReplaceAll(strings.TrimSpace(value), ",", "")
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

func truthyString(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "true", "1", "on", "yes":
		return true
	default:
		return false
	}
}

func transactionFormByKind(kind string) (models.FormDefinition, bool) {
	for _, form := range models.TransactionForms() {
		if form.Kind == kind {
			return form, true
		}
	}
	return models.FormDefinition{}, false
}

func masterFormByKind(kind string) (models.FormDefinition, bool) {
	for _, form := range models.MasterForms() {
		if form.Kind == kind {
			return form, true
		}
	}
	return models.FormDefinition{}, false
}
