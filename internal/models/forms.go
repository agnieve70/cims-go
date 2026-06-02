package models

type FieldType string

const (
	FieldText     FieldType = "text"
	FieldDate     FieldType = "date"
	FieldMoney    FieldType = "money"
	FieldNumber   FieldType = "number"
	FieldBool     FieldType = "bool"
	FieldTextarea FieldType = "textarea"
	FieldSelect   FieldType = "select"
	FieldCombo    FieldType = "combo"
	FieldHidden   FieldType = "hidden"
	FieldReadonly FieldType = "readonly"
)

type Field struct {
	Key      string
	Label    string
	Column   string
	Type     FieldType
	Required bool
	Source   string
}

type LineColumn struct {
	Key      string
	Label    string
	Type     FieldType
	Source   string
	ReadOnly bool
	Hidden   bool
}

type LineGroup struct {
	Key     string
	Label   string
	Columns []LineColumn
}

type FormDefinition struct {
	Kind       string
	Title      string
	Table      string
	RouteBase  string
	Fields     []Field
	LineGroups []LineGroup
	PartyType  string
}

func MasterForms() []FormDefinition {
	return []FormDefinition{
		{
			Kind: "stock-categories", Title: "Stock Categories", Table: "stock_categories", RouteBase: "/masters/stock-categories",
			Fields: []Field{
				{Key: "name", Label: "Name", Column: "name", Type: FieldText, Required: true},
				{Key: "group_name", Label: "Group", Column: "group_name", Type: FieldCombo, Required: true, Source: "stock_category_groups"},
				{Key: "aps_monitor", Label: "APS Monitor", Column: "aps_monitor", Type: FieldBool},
			},
		},
		{
			Kind: "branches", Title: "Branches", Table: "branches", RouteBase: "/masters/branches",
			Fields: []Field{
				{Key: "code", Label: "Code", Column: "code", Type: FieldText, Required: true},
				{Key: "name", Label: "Name", Column: "name", Type: FieldText, Required: true},
				{Key: "incharge", Label: "Incharge", Column: "incharge", Type: FieldText},
				{Key: "aps", Label: "APS", Column: "aps", Type: FieldText},
				{Key: "farm_customer", Label: "Farm Customer", Column: "farm_customer", Type: FieldBool},
				{Key: "remarks", Label: "Remarks", Column: "remarks", Type: FieldTextarea},
			},
		},
		{
			Kind: "suppliers", Title: "Suppliers", Table: "suppliers", RouteBase: "/masters/suppliers",
			Fields: partyFields("Supplier Code", false),
		},
		{
			Kind: "customers", Title: "Customers", Table: "customers", RouteBase: "/masters/customers",
			Fields: append(partyFields("Client Code", true),
				Field{Key: "credit_term", Label: "Credit Term", Column: "credit_term", Type: FieldText},
				Field{Key: "credit_limit", Label: "Credit Limit", Column: "credit_limit", Type: FieldMoney},
				Field{Key: "aps", Label: "APS", Column: "aps", Type: FieldText},
				Field{Key: "farm_customer", Label: "Farm Customer", Column: "farm_customer", Type: FieldBool},
			),
		},
		{
			Kind: "expense-chart", Title: "Expenses Chart", Table: "expense_charts", RouteBase: "/masters/expense-chart",
			Fields: []Field{
				{Key: "code", Label: "Code", Column: "code", Type: FieldText, Required: true},
				{Key: "name", Label: "Name", Column: "name", Type: FieldText, Required: true},
				{Key: "description", Label: "Description", Column: "description", Type: FieldTextarea},
				{Key: "exclude_daily_sales", Label: "Not Included in Daily Sales and Collection Report", Column: "exclude_daily_sales", Type: FieldBool},
				{Key: "daily_sales_only", Label: "For Daily Sales and Collection Report Only", Column: "daily_sales_only", Type: FieldBool},
			},
		},
		{
			Kind: "stocks", Title: "Stocks", Table: "stocks", RouteBase: "/masters/stocks",
			Fields: []Field{
				{Key: "code", Label: "Code", Column: "code", Type: FieldText, Required: true},
				{Key: "name", Label: "Name", Column: "name", Type: FieldText, Required: true},
				{Key: "category_group", Label: "Category Group", Column: "category_group", Type: FieldSelect, Source: "stock_category_groups"},
				{Key: "unit", Label: "Unit", Column: "unit", Type: FieldText},
				{Key: "description", Label: "Description", Column: "description", Type: FieldTextarea},
				{Key: "latest_cost", Label: "Latest Cost", Column: "latest_cost", Type: FieldMoney},
				{Key: "min_inventory", Label: "Min. Inventory", Column: "min_inventory", Type: FieldNumber},
			},
		},
		{
			Kind: "other-income-chart", Title: "Other Income Chart", Table: "other_income_charts", RouteBase: "/masters/other-income-chart",
			Fields: []Field{
				{Key: "code", Label: "Code", Column: "code", Type: FieldText, Required: true},
				{Key: "name", Label: "Name", Column: "name", Type: FieldText, Required: true},
				{Key: "description", Label: "Description", Column: "description", Type: FieldTextarea},
			},
		},
	}
}

func partyFields(codeLabel string, customer bool) []Field {
	fields := []Field{
		{Key: "code", Label: codeLabel, Column: "code", Type: FieldText, Required: true},
		{Key: "company", Label: "Company", Column: "company", Type: FieldText},
		{Key: "lastname", Label: "Lastname", Column: "lastname", Type: FieldText},
		{Key: "firstname", Label: "Firstname", Column: "firstname", Type: FieldText},
		{Key: "middlename", Label: "Middle Name", Column: "middlename", Type: FieldText},
		{Key: "phone_number", Label: "Phone Number", Column: "phone_number", Type: FieldText},
		{Key: "address", Label: "Address", Column: "address", Type: FieldTextarea},
		{Key: "balance", Label: "Balance", Column: "balance", Type: FieldMoney},
	}
	return fields
}

func TransactionForms() []FormDefinition {
	return []FormDefinition{
		{
			Kind: "stock-in", Title: "Stock In File", RouteBase: "/transactions/stock-in",
			Fields:     baseTransactionFields(false, ""),
			LineGroups: []LineGroup{stockLines("details", "Details")},
		},
		{
			Kind: "stock-out", Title: "Stock Out File", RouteBase: "/transactions/stock-out",
			Fields:     baseTransactionFields(false, ""),
			LineGroups: []LineGroup{stockLines("details", "Details")},
		},
		{
			Kind: "checks-in", Title: "Checks In", RouteBase: "/transactions/checks-in",
			Fields:     baseTransactionFields(false, ""),
			LineGroups: []LineGroup{checkLines("checks", "Details")},
		},
		{
			Kind: "other-income", Title: "Other Income File", RouteBase: "/transactions/other-income",
			Fields:     append(baseTransactionFields(true, ""), Field{Key: "branch_id", Label: "Branch", Column: "branch_id", Type: FieldSelect, Source: "branches"}),
			LineGroups: []LineGroup{moneyLines("details", "Details", "other_income_charts")},
		},
		{
			Kind: "purchases", Title: "Purchases File", RouteBase: "/transactions/purchases", PartyType: "supplier",
			Fields: append(baseTransactionFields(false, "suppliers"),
				Field{Key: "cash", Label: "Cash", Column: "cash", Type: FieldBool},
				Field{Key: "or_ci_number", Label: "OR/CI Number", Column: "or_ci_number", Type: FieldText},
				Field{Key: "purchase_date", Label: "Purchase Date", Column: "purchase_date", Type: FieldDate},
			),
			LineGroups: []LineGroup{stockLines("details", "Details"), adjustmentLines("discounts", "Discounts"), adjustmentLines("additionals", "Additionals"), checkLines("payments", "Mode of Payment")},
		},
		{
			Kind: "dr", Title: "DR File", RouteBase: "/transactions/dr", PartyType: "customer",
			Fields: []Field{
				{Key: "party_id", Label: "Customer", Column: "party_id", Type: FieldSelect, Source: "customers", Required: true},
				{Key: "entry_date", Label: "Entry Date", Column: "entry_date", Type: FieldDate, Required: true},
				{Key: "reference", Label: "DR Number", Column: "reference", Type: FieldText},
				{Key: "sales_date", Label: "DR Date", Column: "sales_date", Type: FieldDate},
				{Key: "remarks", Label: "Remarks", Column: "remarks", Type: FieldTextarea},
			},
			LineGroups: []LineGroup{drLines("details", "Details")},
		},
		{
			Kind: "sales", Title: "Sales File", RouteBase: "/transactions/sales", PartyType: "customer",
			Fields: append([]Field{{Key: "dr_document_id", Label: "DR File", Column: "dr_document_id", Type: FieldSelect, Source: "dr_documents", Required: true}}, append(baseTransactionFields(false, "customers"),
				Field{Key: "cash", Label: "Cash", Column: "cash", Type: FieldBool},
				Field{Key: "or_ci_number", Label: "OR/CI Number", Column: "or_ci_number", Type: FieldText},
				Field{Key: "sales_date", Label: "Sales Date", Column: "sales_date", Type: FieldDate},
			)...),
			LineGroups: []LineGroup{drBackedSalesLines("details", "Details"), adjustmentLines("deductions", "Other Deductions"), adjustmentLines("additionals", "Other Additionals"), checkLines("payments", "Mode of Payment")},
		},
		{
			Kind: "stock-transactions", Title: "Stock Transactions File", RouteBase: "/transactions/stock-transactions",
			Fields: append([]Field{{Key: "dr_document_id", Label: "DR File", Column: "dr_document_id", Type: FieldSelect, Source: "dr_documents", Required: true}},
				append(baseTransactionFields(false, ""), Field{Key: "transfer_date", Label: "Transfer Date", Column: "transfer_date", Type: FieldDate}, Field{Key: "transfer_id", Label: "Transfer ID", Column: "transfer_id", Type: FieldText}, Field{Key: "transaction", Label: "Transaction", Column: "transaction", Type: FieldText}, Field{Key: "branch_location", Label: "Branch/Location", Column: "branch_location", Type: FieldSelect, Source: "branches"})...),
			LineGroups: []LineGroup{drBackedSalesLines("details", "Details"), adjustmentLines("discounts", "Discounts"), adjustmentLines("additionals", "Additionals")},
		},
		{Kind: "ap-credit", Title: "AP Credit File", RouteBase: "/transactions/ap-credit", PartyType: "supplier", Fields: creditFields("suppliers"), LineGroups: []LineGroup{checkLines("checks", "Checks")}},
		{Kind: "ap-debit", Title: "AP Debit File", RouteBase: "/transactions/ap-debit", PartyType: "supplier", Fields: debitFields("suppliers")},
		{Kind: "ar-credit", Title: "AR Credit File", RouteBase: "/transactions/ar-credit", PartyType: "customer", Fields: creditFields("customers"), LineGroups: []LineGroup{checkLines("checks", "Checks")}},
		{Kind: "ar-debit", Title: "AR Debit File", RouteBase: "/transactions/ar-debit", PartyType: "customer", Fields: debitFields("customers")},
		{Kind: "rebates", Title: "Rebates File", RouteBase: "/transactions/rebates", PartyType: "customer", Fields: creditFields("customers"), LineGroups: []LineGroup{checkLines("checks", "Checks")}},
		{Kind: "expenses", Title: "Expenses File", RouteBase: "/transactions/expenses", Fields: baseTransactionFields(false, ""), LineGroups: []LineGroup{moneyLines("details", "Details", "expense_charts")}},
	}
}

func baseTransactionFields(branch bool, partySource string) []Field {
	fields := []Field{
		{Key: "entry_date", Label: "Entry Date", Column: "entry_date", Type: FieldDate, Required: true},
		{Key: "remarks", Label: "Remarks", Column: "remarks", Type: FieldTextarea},
	}
	if partySource != "" {
		fields = append([]Field{{Key: "party_id", Label: "Company", Column: "party_id", Type: FieldSelect, Source: partySource, Required: true}}, fields...)
	}
	return fields
}

func creditFields(partySource string) []Field {
	return []Field{
		{Key: "entry_date", Label: "Date", Column: "entry_date", Type: FieldDate, Required: true},
		{Key: "reference", Label: "Reference", Column: "reference", Type: FieldText},
		{Key: "party_id", Label: "Company", Column: "party_id", Type: FieldSelect, Source: partySource, Required: true},
		{Key: "cash_amount", Label: "Cash Amount", Column: "cash_amount", Type: FieldMoney},
		{Key: "remarks", Label: "Remarks", Column: "remarks", Type: FieldTextarea},
	}
}

func debitFields(partySource string) []Field {
	return []Field{
		{Key: "entry_date", Label: "Entry Date", Column: "entry_date", Type: FieldDate, Required: true},
		{Key: "party_id", Label: "Company", Column: "party_id", Type: FieldSelect, Source: partySource, Required: true},
		{Key: "amount", Label: "Amount", Column: "amount", Type: FieldMoney, Required: true},
		{Key: "remarks", Label: "Remarks", Column: "remarks", Type: FieldTextarea},
	}
}

func stockLines(key, label string) LineGroup {
	return LineGroup{Key: key, Label: label, Columns: []LineColumn{
		{Key: "stock_id", Label: "Stock", Type: FieldSelect, Source: "stocks"},
		{Key: "qty", Label: "Qty", Type: FieldNumber},
		{Key: "unit_cost", Label: "Unit Cost", Type: FieldMoney},
		{Key: "amount", Label: "Amount", Type: FieldMoney},
	}}
}

func drLines(key, label string) LineGroup {
	return LineGroup{Key: key, Label: label, Columns: []LineColumn{
		{Key: "stock_id", Label: "Stock", Type: FieldSelect, Source: "stocks"},
		{Key: "qty", Label: "Qty", Type: FieldNumber},
	}}
}

func salesLines(key, label string) LineGroup {
	group := stockLines(key, label)
	group.Columns = append(group.Columns,
		LineColumn{Key: "capital", Label: "Capital", Type: FieldMoney},
		LineColumn{Key: "discount", Label: "Discount", Type: FieldMoney},
		LineColumn{Key: "other_discount", Label: "Other Disc.", Type: FieldMoney},
		LineColumn{Key: "markup", Label: "Markup", Type: FieldMoney},
		LineColumn{Key: "markup_pct", Label: "MU %", Type: FieldNumber},
	)
	return group
}

func drBackedSalesLines(key, label string) LineGroup {
	return LineGroup{Key: key, Label: label, Columns: []LineColumn{
		{Key: "dr_line_id", Type: FieldHidden, Hidden: true},
		{Key: "stock_id", Type: FieldHidden, Hidden: true},
		{Key: "stock_label", Label: "Stock", Type: FieldReadonly, ReadOnly: true},
		{Key: "qty", Label: "Qty", Type: FieldNumber, ReadOnly: true},
		{Key: "unit_cost", Label: "Unit Cost", Type: FieldMoney},
		{Key: "amount", Label: "Amount", Type: FieldMoney},
		{Key: "capital", Label: "Capital", Type: FieldMoney},
		{Key: "discount", Label: "Discount", Type: FieldMoney},
		{Key: "other_discount", Label: "Other Disc.", Type: FieldMoney},
		{Key: "markup", Label: "Markup", Type: FieldMoney},
		{Key: "markup_pct", Label: "MU %", Type: FieldNumber},
	}}
}

func adjustmentLines(key, label string) LineGroup {
	return LineGroup{Key: key, Label: label, Columns: []LineColumn{
		{Key: "particulars", Label: "Particulars", Type: FieldText},
		{Key: "qty", Label: "Qty", Type: FieldNumber},
		{Key: "price", Label: "Price", Type: FieldMoney},
		{Key: "amount", Label: "Amount", Type: FieldMoney},
	}}
}

func checkLines(key, label string) LineGroup {
	return LineGroup{Key: key, Label: label, Columns: []LineColumn{
		{Key: "number", Label: "Number", Type: FieldText},
		{Key: "date", Label: "Date", Type: FieldDate},
		{Key: "bank_name", Label: "Bank Name", Type: FieldText},
		{Key: "amount", Label: "Amount", Type: FieldMoney},
		{Key: "nature", Label: "Nature", Type: FieldText},
	}}
}

func moneyLines(key, label, source string) LineGroup {
	return LineGroup{Key: key, Label: label, Columns: []LineColumn{
		{Key: "code_id", Label: "Code", Type: FieldSelect, Source: source},
		{Key: "reference", Label: "Reference", Type: FieldText},
		{Key: "cash", Label: "Cash", Type: FieldMoney},
		{Key: "check", Label: "Check", Type: FieldMoney},
		{Key: "total", Label: "Total", Type: FieldMoney},
	}}
}

func FindMaster(kind string) (FormDefinition, bool) {
	for _, form := range MasterForms() {
		if form.Kind == kind {
			return form, true
		}
	}
	return FormDefinition{}, false
}

func FindTransaction(kind string) (FormDefinition, bool) {
	for _, form := range TransactionForms() {
		if form.Kind == kind {
			return form, true
		}
	}
	return FormDefinition{}, false
}
