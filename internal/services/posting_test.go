package services

import "testing"

func TestPurchaseTotalsApplyDiscountsAdditionalsAndCashBalance(t *testing.T) {
	doc := PurchaseDocument{
		Cash: false,
		Lines: []StockLine{
			{Qty: 2, UnitCost: 12500},
			{Qty: 1, UnitCost: 10000},
		},
		Discounts:   []AdjustmentLine{{Qty: 1, Price: 5000}},
		Additionals: []AdjustmentLine{{Qty: 2, Price: 1000}},
		Payments:    []Payment{{CashAmount: 12000}, {CheckAmount: 8000}},
	}

	total := ComputePurchase(doc)

	if total.TotalQty != 3 {
		t.Fatalf("TotalQty = %d, want 3", total.TotalQty)
	}
	if total.Total != 35000 {
		t.Fatalf("Total = %d, want 35000", total.Total)
	}
	if total.Less != 5000 {
		t.Fatalf("Less = %d, want 5000", total.Less)
	}
	if total.Add != 2000 {
		t.Fatalf("Add = %d, want 2000", total.Add)
	}
	if total.Net != 32000 {
		t.Fatalf("Net = %d, want 32000", total.Net)
	}
	if total.Paid != 20000 {
		t.Fatalf("Paid = %d, want 20000", total.Paid)
	}
	if total.Balance != 12000 {
		t.Fatalf("Balance = %d, want 12000", total.Balance)
	}
}

func TestCashPurchaseHasZeroBalance(t *testing.T) {
	doc := PurchaseDocument{
		Cash:  true,
		Lines: []StockLine{{Qty: 4, UnitCost: 2500}},
	}

	total := ComputePurchase(doc)

	if total.Net != 10000 {
		t.Fatalf("Net = %d, want 10000", total.Net)
	}
	if total.Balance != 0 {
		t.Fatalf("Balance = %d, want 0 for cash purchase", total.Balance)
	}
}

func TestSalesTotalsMarkupAndBalance(t *testing.T) {
	doc := SalesDocument{
		Cash: false,
		Lines: []SalesLine{
			{Qty: 2, UnitCost: 15000, Capital: 10000, Discount: 500, OtherDiscount: 250},
			{Qty: 1, UnitCost: 20000, Capital: 12000},
		},
		Deductions:  []AdjustmentLine{{Qty: 1, Price: 1000}},
		Additionals: []AdjustmentLine{{Qty: 1, Price: 500}},
		Payments:    []Payment{{CashAmount: 10000}, {CheckAmount: 5000}},
	}

	total := ComputeSales(doc)

	if total.TotalQty != 3 {
		t.Fatalf("TotalQty = %d, want 3", total.TotalQty)
	}
	if total.TotalNetAmount != 49250 {
		t.Fatalf("TotalNetAmount = %d, want 49250", total.TotalNetAmount)
	}
	if total.TotalCapital != 32000 {
		t.Fatalf("TotalCapital = %d, want 32000", total.TotalCapital)
	}
	if total.TotalMarkup != 17250 {
		t.Fatalf("TotalMarkup = %d, want 17250", total.TotalMarkup)
	}
	if total.AverageMarkupBP != 5391 {
		t.Fatalf("AverageMarkupBP = %d, want 5391", total.AverageMarkupBP)
	}
	if total.Net != 48750 {
		t.Fatalf("Net = %d, want 48750", total.Net)
	}
	if total.Balance != 33750 {
		t.Fatalf("Balance = %d, want 33750", total.Balance)
	}
}

func TestPostingEffectsForInventoryAndBalances(t *testing.T) {
	purchase := PostingRequest{
		Kind:     DocumentPurchase,
		BranchID: 7,
		PartyID:  33,
		Lines:    []StockLine{{StockID: 11, Qty: 5, UnitCost: 1200}},
		Net:      6000,
		Balance:  4000,
	}

	purchaseEffects := BuildPostingEffects(purchase)

	if len(purchaseEffects.Inventory) != 1 {
		t.Fatalf("purchase inventory effects = %d, want 1", len(purchaseEffects.Inventory))
	}
	if purchaseEffects.Inventory[0].QtyDelta != 5 {
		t.Fatalf("purchase QtyDelta = %d, want 5", purchaseEffects.Inventory[0].QtyDelta)
	}
	if purchaseEffects.Balance.PartyType != PartySupplier || purchaseEffects.Balance.AmountDelta != 4000 {
		t.Fatalf("purchase balance = %#v, want supplier +4000", purchaseEffects.Balance)
	}

	sale := PostingRequest{
		Kind:     DocumentSale,
		BranchID: 7,
		PartyID:  44,
		Lines:    []StockLine{{StockID: 11, Qty: 2, UnitCost: 1600}},
		Net:      3200,
		Balance:  1200,
	}

	saleEffects := BuildPostingEffects(sale)

	if saleEffects.Inventory[0].QtyDelta != -2 {
		t.Fatalf("sale QtyDelta = %d, want -2", saleEffects.Inventory[0].QtyDelta)
	}
	if saleEffects.Balance.PartyType != PartyCustomer || saleEffects.Balance.AmountDelta != 1200 {
		t.Fatalf("sale balance = %#v, want customer +1200", saleEffects.Balance)
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
	if apCredit.Balance.PartyType != PartySupplier || apCredit.Balance.AmountDelta != -2000 {
		t.Fatalf("AP credit balance = %#v, want supplier -2000", apCredit.Balance)
	}
}
