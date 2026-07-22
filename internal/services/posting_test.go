package services

import "testing"

func TestPurchaseTotalsApplyDiscountsAdditionalsAndCashBalance(t *testing.T) {
	doc := PurchaseDocument{
		Cash: false,
		Lines: []StockLine{
			{Qty: 2000, UnitCost: 125000},
			{Qty: 1000, UnitCost: 100000},
		},
		Discounts:   []AdjustmentLine{{Qty: 1000, Price: 50000}},
		Additionals: []AdjustmentLine{{Qty: 2000, Price: 10000}},
		Payments:    []Payment{{CashAmount: 120000}, {CheckAmount: 80000}},
	}

	total := ComputePurchase(doc)

	if total.TotalQty != 3000 {
		t.Fatalf("TotalQty = %d, want 3000", total.TotalQty)
	}
	if total.Total != 350000 {
		t.Fatalf("Total = %d, want 350000", total.Total)
	}
	if total.Less != 50000 {
		t.Fatalf("Less = %d, want 50000", total.Less)
	}
	if total.Add != 20000 {
		t.Fatalf("Add = %d, want 20000", total.Add)
	}
	if total.Net != 320000 {
		t.Fatalf("Net = %d, want 320000", total.Net)
	}
	if total.Paid != 200000 {
		t.Fatalf("Paid = %d, want 200000", total.Paid)
	}
	if total.Balance != 120000 {
		t.Fatalf("Balance = %d, want 120000", total.Balance)
	}
}

func TestCashPurchaseHasZeroBalance(t *testing.T) {
	doc := PurchaseDocument{
		Cash:  true,
		Lines: []StockLine{{Qty: 4000, UnitCost: 25000}},
	}

	total := ComputePurchase(doc)

	if total.Net != 100000 {
		t.Fatalf("Net = %d, want 100000", total.Net)
	}
	if total.Balance != 0 {
		t.Fatalf("Balance = %d, want 0 for cash purchase", total.Balance)
	}
}

func TestSalesTotalsMarkupAndBalance(t *testing.T) {
	doc := SalesDocument{
		Cash: false,
		Lines: []SalesLine{
			{Qty: 2000, UnitCost: 150000, Capital: 100000, Discount: 5000, OtherDiscount: 2500},
			{Qty: 1000, UnitCost: 200000, Capital: 120000},
		},
		Deductions:  []AdjustmentLine{{Qty: 1000, Price: 10000}},
		Additionals: []AdjustmentLine{{Qty: 1000, Price: 5000}},
		Payments:    []Payment{{CashAmount: 100000}, {CheckAmount: 50000}},
	}

	total := ComputeSales(doc)

	if total.TotalQty != 3000 {
		t.Fatalf("TotalQty = %d, want 3000", total.TotalQty)
	}
	if total.TotalNetAmount != 492500 {
		t.Fatalf("TotalNetAmount = %d, want 492500", total.TotalNetAmount)
	}
	if total.TotalCapital != 320000 {
		t.Fatalf("TotalCapital = %d, want 320000", total.TotalCapital)
	}
	if total.TotalMarkup != 172500 {
		t.Fatalf("TotalMarkup = %d, want 172500", total.TotalMarkup)
	}
	if total.AverageMarkupBP != 3450 {
		t.Fatalf("AverageMarkupBP = %d, want 3450", total.AverageMarkupBP)
	}
	if total.Net != 487500 {
		t.Fatalf("Net = %d, want 487500", total.Net)
	}
	if total.Balance != 337500 {
		t.Fatalf("Balance = %d, want 337500", total.Balance)
	}
}

func TestSalesMarkupPercentUsesAmountAsDenominator(t *testing.T) {
	total := ComputeSales(SalesDocument{
		Lines: []SalesLine{{Qty: 1000, UnitCost: 11500000, Capital: 10000000}},
	})

	if total.TotalMarkup != 1500000 {
		t.Fatalf("TotalMarkup = %d, want 1500000", total.TotalMarkup)
	}
	if total.AverageMarkupBP != 1304 {
		t.Fatalf("AverageMarkupBP = %d, want 1304", total.AverageMarkupBP)
	}
}

func TestCashSaleDoesNotCreateReceivableBalance(t *testing.T) {
	total := ComputeSales(SalesDocument{
		Cash:  true,
		Lines: []SalesLine{{Qty: 2000, UnitCost: 150000}},
	})

	if total.Net != 300000 {
		t.Fatalf("Net = %d, want 300000", total.Net)
	}
	if total.Balance != 0 {
		t.Fatalf("Balance = %d, want 0 for a cash sale", total.Balance)
	}
}

func TestPostingEffectsForInventoryAndBalances(t *testing.T) {
	purchase := PostingRequest{
		Kind:     DocumentPurchase,
		BranchID: 7,
		PartyID:  33,
		Lines:    []StockLine{{StockID: 11, Qty: 5000, UnitCost: 12000}},
		Net:      60000,
		Balance:  40000,
	}

	purchaseEffects := BuildPostingEffects(purchase)

	if len(purchaseEffects.Inventory) != 1 {
		t.Fatalf("purchase inventory effects = %d, want 1", len(purchaseEffects.Inventory))
	}
	if purchaseEffects.Inventory[0].QtyDelta != 5000 {
		t.Fatalf("purchase QtyDelta = %d, want 5000", purchaseEffects.Inventory[0].QtyDelta)
	}
	if purchaseEffects.Balance.PartyType != PartySupplier || purchaseEffects.Balance.AmountDelta != 40000 {
		t.Fatalf("purchase balance = %#v, want supplier +40000", purchaseEffects.Balance)
	}

	sale := PostingRequest{
		Kind:     DocumentSale,
		BranchID: 7,
		PartyID:  44,
		Lines:    []StockLine{{StockID: 11, Qty: 2000, UnitCost: 16000}},
		Net:      32000,
		Balance:  12000,
	}

	saleEffects := BuildPostingEffects(sale)

	if saleEffects.Inventory[0].QtyDelta != -2000 {
		t.Fatalf("sale QtyDelta = %d, want -2000", saleEffects.Inventory[0].QtyDelta)
	}
	if saleEffects.Balance.PartyType != PartyCustomer || saleEffects.Balance.AmountDelta != 12000 {
		t.Fatalf("sale balance = %#v, want customer +12000", saleEffects.Balance)
	}

	arCredit := BuildPostingEffects(PostingRequest{
		Kind:    DocumentARCredit,
		PartyID: 44,
		Paid:    1200,
	})
	if arCredit.Balance.PartyType != PartyCustomer || arCredit.Balance.AmountDelta != -1200 {
		t.Fatalf("AR credit balance = %#v, want customer -1200", arCredit.Balance)
	}

	apCredit := BuildPostingEffects(PostingRequest{
		Kind:    DocumentAPCredit,
		PartyID: 33,
		Paid:    2000,
	})
	if apCredit.Balance.PartyType != PartyCustomer || apCredit.Balance.AmountDelta != -2000 {
		t.Fatalf("AP credit balance = %#v, want customer -2000", apCredit.Balance)
	}
}

func TestStockTransferReturnDirections(t *testing.T) {
	line := []StockLine{{StockID: 9, Qty: 1500, UnitCost: 10000}}

	salesReturn := BuildPostingEffects(PostingRequest{
		Kind: DocumentStockTransfer, BranchID: 2, PartyID: 4, Lines: line, Net: 15000, Transfer: TransferSalesReturn,
	})
	if salesReturn.Inventory[0].QtyDelta != 1500 {
		t.Fatalf("sales return quantity = %d, want +1500", salesReturn.Inventory[0].QtyDelta)
	}
	if salesReturn.Balance.PartyType != PartyCustomer || salesReturn.Balance.AmountDelta != -15000 {
		t.Fatalf("sales return balance = %#v, want customer -15000", salesReturn.Balance)
	}

	stockReturn := BuildPostingEffects(PostingRequest{
		Kind: DocumentStockTransfer, BranchID: 2, PartyID: 5, Lines: line, Net: 15000, Transfer: TransferStockReturn,
	})
	if stockReturn.Inventory[0].QtyDelta != -1500 {
		t.Fatalf("stock return quantity = %d, want -1500", stockReturn.Inventory[0].QtyDelta)
	}
	if stockReturn.Balance.PartyType != PartySupplier || stockReturn.Balance.AmountDelta != -15000 {
		t.Fatalf("stock return balance = %#v, want supplier -15000", stockReturn.Balance)
	}

	rebate := BuildPostingEffects(PostingRequest{Kind: DocumentRebate, PartyID: 4, Paid: 5000})
	if rebate.Balance.PartyType != PartyNone || rebate.Balance.AmountDelta != 0 {
		t.Fatalf("rebate balance = %#v, want no AR effect", rebate.Balance)
	}
}
