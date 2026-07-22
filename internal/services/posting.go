package services

type DocumentKind string

const DecimalScale int64 = 1000

const (
	DocumentDR            DocumentKind = "dr"
	DocumentPurchase      DocumentKind = "purchases"
	DocumentSale          DocumentKind = "sales"
	DocumentStockIn       DocumentKind = "stock_in"
	DocumentStockOut      DocumentKind = "stock_out"
	DocumentStockTransfer DocumentKind = "stock_transactions"
	DocumentAPCredit      DocumentKind = "ap_credit"
	DocumentAPDebit       DocumentKind = "ap_debit"
	DocumentARCredit      DocumentKind = "ar_credit"
	DocumentARDebit       DocumentKind = "ar_debit"
	DocumentRebate        DocumentKind = "rebates"
	DocumentExpense       DocumentKind = "expenses"
	DocumentOtherIncome   DocumentKind = "other_income"
	DocumentChecksIn      DocumentKind = "checks_in"
)

type TransferMode string

const (
	TransferStock       TransferMode = "stock_transfer"
	TransferSalesReturn TransferMode = "sales_return"
	TransferStockReturn TransferMode = "stock_return"
)

type PartyType string

const (
	PartyNone     PartyType = ""
	PartySupplier PartyType = "supplier"
	PartyCustomer PartyType = "customer"
)

type StockLine struct {
	StockID   int64
	StockCode string
	StockName string
	Qty       int64
	UnitCost  int64
	Capital   int64
}

type SalesLine struct {
	StockID       int64
	Code          string
	StockName     string
	Qty           int64
	UnitCost      int64
	Capital       int64
	Discount      int64
	OtherDiscount int64
}

type AdjustmentLine struct {
	Particulars string
	Qty         int64
	Price       int64
}

type Payment struct {
	CashAmount  int64
	CheckAmount int64
}

type PurchaseDocument struct {
	Cash        bool
	Lines       []StockLine
	Discounts   []AdjustmentLine
	Additionals []AdjustmentLine
	Payments    []Payment
}

type SalesDocument struct {
	Cash        bool
	Lines       []SalesLine
	Deductions  []AdjustmentLine
	Additionals []AdjustmentLine
	Payments    []Payment
}

type PurchaseTotals struct {
	TotalQty int64
	Total    int64
	Less     int64
	Add      int64
	Net      int64
	Paid     int64
	Balance  int64
}

type SalesTotals struct {
	TotalQty        int64
	TotalNetAmount  int64
	TotalCapital    int64
	TotalMarkup     int64
	AverageMarkupBP int64
	Less            int64
	Add             int64
	Net             int64
	Paid            int64
	Balance         int64
}

func ComputePurchase(doc PurchaseDocument) PurchaseTotals {
	var total PurchaseTotals
	for _, line := range doc.Lines {
		total.TotalQty += line.Qty
		total.Total += scaledProduct(line.Qty, line.UnitCost)
	}
	total.Less = adjustmentTotal(doc.Discounts)
	total.Add = adjustmentTotal(doc.Additionals)
	total.Net = total.Total - total.Less + total.Add
	total.Paid = paymentTotal(doc.Payments)
	if !doc.Cash {
		total.Balance = total.Net - total.Paid
	}
	return total
}

func ComputeSales(doc SalesDocument) SalesTotals {
	var total SalesTotals
	var totalAmount int64
	for _, line := range doc.Lines {
		gross := scaledProduct(line.Qty, line.UnitCost)
		net := gross - line.Discount - line.OtherDiscount
		capital := scaledProduct(line.Qty, line.Capital)
		total.TotalQty += line.Qty
		totalAmount += gross
		total.TotalNetAmount += net
		total.TotalCapital += capital
	}
	total.TotalMarkup = total.TotalNetAmount - total.TotalCapital
	if totalAmount > 0 {
		total.AverageMarkupBP = (total.TotalMarkup*10000 + totalAmount/2) / totalAmount
	}
	total.Less = adjustmentTotal(doc.Deductions)
	total.Add = adjustmentTotal(doc.Additionals)
	total.Net = total.TotalNetAmount - total.Less + total.Add
	total.Paid = paymentTotal(doc.Payments)
	if !doc.Cash {
		total.Balance = total.Net - total.Paid
	}
	return total
}

func adjustmentTotal(lines []AdjustmentLine) int64 {
	var total int64
	for _, line := range lines {
		total += scaledProduct(line.Qty, line.Price)
	}
	return total
}

func scaledProduct(left, right int64) int64 {
	product := left * right
	if product < 0 {
		return (product - DecimalScale/2) / DecimalScale
	}
	return (product + DecimalScale/2) / DecimalScale
}

func paymentTotal(lines []Payment) int64 {
	var total int64
	for _, line := range lines {
		total += line.CashAmount + line.CheckAmount
	}
	return total
}

type InventoryEffect struct {
	BranchID int64
	StockID  int64
	QtyDelta int64
	Cost     int64
}

type BalanceEffect struct {
	PartyType   PartyType
	PartyID     int64
	AmountDelta int64
}

type PostingRequest struct {
	Kind     DocumentKind
	BranchID int64
	PartyID  int64
	Lines    []StockLine
	Net      int64
	Balance  int64
	Paid     int64
	Amount   int64
	Transfer TransferMode
}

type PostingEffects struct {
	Inventory []InventoryEffect
	Balance   BalanceEffect
}

func BuildPostingEffects(req PostingRequest) PostingEffects {
	effects := PostingEffects{}

	switch req.Kind {
	case DocumentPurchase, DocumentStockIn:
		effects.Inventory = inventoryEffects(req.BranchID, req.Lines, 1)
	case DocumentSale, DocumentStockOut:
		effects.Inventory = inventoryEffects(req.BranchID, req.Lines, -1)
	case DocumentStockTransfer:
		direction := int64(-1)
		if req.Transfer == TransferSalesReturn {
			direction = 1
		}
		effects.Inventory = inventoryEffects(req.BranchID, req.Lines, direction)
	}

	switch req.Kind {
	case DocumentPurchase:
		effects.Balance = BalanceEffect{PartyType: PartySupplier, PartyID: req.PartyID, AmountDelta: req.Balance}
	case DocumentSale:
		effects.Balance = BalanceEffect{PartyType: PartyCustomer, PartyID: req.PartyID, AmountDelta: req.Balance}
	case DocumentAPCredit:
		effects.Balance = BalanceEffect{PartyType: PartyCustomer, PartyID: req.PartyID, AmountDelta: -req.Paid}
	case DocumentAPDebit:
		effects.Balance = BalanceEffect{PartyType: PartySupplier, PartyID: req.PartyID, AmountDelta: req.Amount}
	case DocumentARCredit:
		effects.Balance = BalanceEffect{PartyType: PartyCustomer, PartyID: req.PartyID, AmountDelta: -req.Paid}
	case DocumentARDebit:
		effects.Balance = BalanceEffect{PartyType: PartyCustomer, PartyID: req.PartyID, AmountDelta: req.Amount}
	case DocumentStockTransfer:
		switch req.Transfer {
		case TransferSalesReturn:
			effects.Balance = BalanceEffect{PartyType: PartyCustomer, PartyID: req.PartyID, AmountDelta: -req.Net}
		case TransferStockReturn:
			effects.Balance = BalanceEffect{PartyType: PartySupplier, PartyID: req.PartyID, AmountDelta: -req.Net}
		}
	}

	return effects
}

func inventoryEffects(branchID int64, lines []StockLine, direction int64) []InventoryEffect {
	effects := make([]InventoryEffect, 0, len(lines))
	for _, line := range lines {
		if line.StockID == 0 || line.Qty == 0 {
			continue
		}
		effects = append(effects, InventoryEffect{
			BranchID: branchID,
			StockID:  line.StockID,
			QtyDelta: line.Qty * direction,
			Cost:     line.UnitCost,
		})
	}
	return effects
}
