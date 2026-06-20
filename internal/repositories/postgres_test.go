package repositories

import (
	"testing"

	"cims-go/internal/models"
)

func TestValueForMasterFieldDefaultsBlankCustomerCreditLimit(t *testing.T) {
	form := models.FormDefinition{Kind: "customers"}
	field := models.Field{Key: "credit_limit", Type: models.FieldMoney}

	got := valueForMasterField(form, field, map[string]string{"credit_limit": ""})
	if got != "0.00" {
		t.Fatalf("blank customer credit_limit = %#v, want 0.00", got)
	}
}

func TestValueForMasterFieldLeavesOtherBlankMoneyNullable(t *testing.T) {
	form := models.FormDefinition{Kind: "stocks"}
	field := models.Field{Key: "latest_cost", Type: models.FieldMoney}

	got := valueForMasterField(form, field, map[string]string{"latest_cost": ""})
	if got != nil {
		t.Fatalf("blank stock latest_cost = %#v, want nil", got)
	}
}
