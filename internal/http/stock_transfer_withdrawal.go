package http

import (
	"net/http"
	"strconv"
	"strings"

	"cims-go/internal/models"

	"github.com/go-chi/chi/v5"
)

type stockTransferWithdrawalPage struct {
	Document    models.Record
	TemplateURL string
	Date        string
	DateYear    string
	From        string
	To          string
	Rows        []stockTransferWithdrawalLine
	TotalAmount string
}

type stockTransferWithdrawalLine struct {
	Qty      string
	Article  string
	UnitCost string
	Amount   string
}

func (a *App) stockTransferWithdrawalPrint(w http.ResponseWriter, r *http.Request) {
	form, ok := transactionFormByKind("stock-transactions")
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
	page := buildStockTransferWithdrawalPage(record, lineRows, a.optionsForForm(r.Context(), form))
	a.render(w, r, "stock_transfer_withdrawal_print.gohtml", viewData{
		Title:                   "Stock Transfer Withdrawal",
		StockTransferWithdrawal: page,
	})
}

func buildStockTransferWithdrawalPage(record models.Record, lineRows map[string][]models.Record, options map[string][]models.Option) stockTransferWithdrawalPage {
	page := stockTransferWithdrawalPage{
		Document:    record,
		TemplateURL: invoiceAssetURL("hs-stock-transfer-withdrawal.png"),
		From:        optionLabel(options, "branches", record["branch_id"]),
		To:          optionLabel(options, "branches", record["branch_location"]),
	}
	page.Date, page.DateYear = invoiceDateParts(record["transfer_date"], record["entry_date"])

	var totalAmount int64
	for _, row := range lineRows["details"] {
		qty := parseInvoiceInt(row["qty"])
		unitCost := parseInvoiceMoney(row["unit_cost"])
		amount := parseInvoiceMoney(row["amount"])
		if amount == 0 {
			amount = qty * unitCost
		}
		if qty == 0 && unitCost == 0 && amount == 0 && strings.TrimSpace(row["stock_label"]) == "" {
			continue
		}
		totalAmount += amount
		page.Rows = append(page.Rows, stockTransferWithdrawalLine{
			Qty:      invoiceQty(qty),
			Article:  invoiceStockDescription(row["stock_label"], row["stock_name"]),
			UnitCost: invoiceMoney(unitCost),
			Amount:   invoiceMoney(amount),
		})
	}
	page.TotalAmount = invoiceMoney(totalAmount)
	return page
}
