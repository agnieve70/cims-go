package repositories

import (
	"testing"

	"cims-go/internal/services"
)

func TestARCreditPostingDeductsCashAndChecks(t *testing.T) {
	totals := buildTotalsInput("ar-credit", map[string]string{
		"party_id":    "7",
		"cash_amount": "20000",
	}, []LineInput{{
		Group: "checks",
		Rows: []map[string]string{{
			"number":    "2222",
			"bank_name": "MBTC",
			"amount":    "100",
		}},
	}})

	const wantPayment = int64(20_100_000)
	if totals.net != wantPayment || totals.posting.Paid != wantPayment {
		t.Fatalf("AR Credit payment = net %d, posted %d; want cash plus checks %d", totals.net, totals.posting.Paid, wantPayment)
	}

	effects := services.BuildPostingEffects(totals.posting)
	if effects.Balance.PartyType != services.PartyCustomer || effects.Balance.PartyID != 7 || effects.Balance.AmountDelta != -wantPayment {
		t.Fatalf("AR Credit balance effect = %#v, want customer 7 deduction of %d", effects.Balance, wantPayment)
	}
}

func TestAPCreditPostingDeductsSupplierBalance(t *testing.T) {
	totals := buildTotalsInput("ap-credit", map[string]string{
		"party_id":    "7",
		"cash_amount": "200",
	}, []LineInput{{
		Group: "checks",
		Rows: []map[string]string{{
			"number": "2222", "bank_name": "MBTC", "amount": "100",
		}},
	}})

	const wantPayment = int64(300_000)
	effects := services.BuildPostingEffects(totals.posting)
	if totals.net != wantPayment || effects.Balance.PartyType != services.PartySupplier || effects.Balance.PartyID != 7 || effects.Balance.AmountDelta != -wantPayment {
		t.Fatalf("AP Credit posting = totals %#v, effect %#v; want supplier 7 deduction of %d", totals, effects.Balance, wantPayment)
	}
}
