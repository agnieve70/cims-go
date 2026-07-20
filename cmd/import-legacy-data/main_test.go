package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseLegacyDumpRejectsMissingSchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "backup.sql")
	if err := os.WriteFile(path, []byte("USE another_schema;\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := parseLegacyDump(path, "cims")
	if err == nil || !strings.Contains(err.Error(), `schema "cims" was not found`) {
		t.Fatalf("parseLegacyDump error = %v, want missing-schema safeguard", err)
	}
}

func TestParseLegacyDumpLinksExpenseOutgoingChecks(t *testing.T) {
	path := filepath.Join(t.TempDir(), "backup.sql")
	dump := "USE cims;\n" +
		"INSERT INTO `expensesdata` (`EntryID`,`TranDate`,`Remarks`,`Reference`,`TotalAmount`) VALUES (79,'2005-09-15','Rent','DRAW',8715.000);\n" +
		"INSERT INTO `outgoingchecks` (`EntryID`,`TranDate`,`CheckNumber`,`CheckDate`,`BankName`,`Amount`,`Reference`,`Company`,`SourceIndex`) VALUES (79,'2005-09-15','1395','2005-09-15','RCBC',8715.000,'DRAW','Sorongon Bodega',1);\n"
	if err := os.WriteFile(path, []byte(dump), 0o600); err != nil {
		t.Fatal(err)
	}
	data, err := parseLegacyDump(path, "cims")
	if err != nil {
		t.Fatal(err)
	}
	doc := data.expenses[79]
	if doc == nil || len(doc.Checks) != 1 {
		t.Fatalf("expense checks = %#v, want one linked outgoing check", doc)
	}
	if doc.Checks[0].Number != "1395" || doc.Checks[0].Nature != "1 - Outgoing Check" {
		t.Fatalf("linked check = %#v", doc.Checks[0])
	}
}
