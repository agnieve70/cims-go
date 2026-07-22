package http

import (
	"testing"
	"time"

	"cims-go/internal/models"
)

func TestARLedgerDeductsEncodedARCredit(t *testing.T) {
	from := time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, time.July, 31, 0, 0, 0, 0, time.UTC)
	report := arLedgerReportData{ReportType: "detailed"}

	report.build([]models.ARLedgerReportRow{
		{
			CustomerID:   "1",
			CustomerCode: "CUS-1",
			CustomerName: "Customer A",
			EntryID:      "SALE-1",
			EntryDate:    "07/10/2026",
			Reference:    "CI-1",
			Kind:         "sales",
			DeltaCents:   100_000,
		},
		{
			CustomerID:   "1",
			CustomerCode: "CUS-1",
			CustomerName: "Customer A",
			EntryID:      "CREDIT-1",
			EntryDate:    "07/15/2026",
			Reference:    "OR-1",
			Kind:         "ar-credit",
			DeltaCents:   -30_000,
		},
	}, from, to)

	if report.TotalDebit != "1,000.00" {
		t.Fatalf("total debit = %q, want 1,000.00", report.TotalDebit)
	}
	if report.TotalCredit != "300.00" {
		t.Fatalf("total credit = %q, want 300.00", report.TotalCredit)
	}
	if report.TotalNet != "700.00" {
		t.Fatalf("net AR balance = %q, want 700.00", report.TotalNet)
	}
	if len(report.Groups) != 1 || len(report.Groups[0].Rows) != 2 {
		t.Fatalf("ledger groups = %#v, want one customer with two entries", report.Groups)
	}
	credit := report.Groups[0].Rows[1]
	if credit.Reference != "OR-1" || credit.Credit != "300.00" || credit.Balance != "700.00" {
		t.Fatalf("AR credit row = %#v, want a 300.00 deduction and 700.00 running balance", credit)
	}
	if len(report.SummaryRows) != 1 || report.SummaryRows[0].Balance != "700.00" {
		t.Fatalf("summary rows = %#v, want customer balance 700.00", report.SummaryRows)
	}
}
