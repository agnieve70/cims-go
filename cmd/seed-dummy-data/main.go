package main

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"cims-go/internal/config"
	appdb "cims-go/internal/db"
	"cims-go/internal/models"
	"cims-go/internal/repositories"

	"github.com/jackc/pgx/v5/pgxpool"
)

type stockSeed struct {
	ID         int64
	Code       string
	Name       string
	Category   string
	CostCents  int64
	PriceCents int64
}

type drLineSeed struct {
	LineID    int64
	StockID   int64
	StockCode string
	Remaining int64
}

type drDocSeed struct {
	ID           int64
	CustomerID   int64
	CustomerCode string
	Reference    string
	Lines        []drLineSeed
}

type seedState struct {
	ctx               context.Context
	pool              *pgxpool.Pool
	store             *repositories.PostgresStore
	user              models.User
	branches          map[string]int64
	suppliers         map[string]int64
	customers         map[string]int64
	expenseCharts     map[string]int64
	otherIncomeCharts map[string]int64
	stocks            map[string]stockSeed
	stockOrder        []stockSeed
	drDocs            []drDocSeed
}

func main() {
	ctx := context.Background()
	cfg, err := config.Load()
	must(err)

	must(appdb.Migrate(cfg.DatabaseURL, "db/migrations"))

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	must(err)
	defer pool.Close()

	store := repositories.NewPostgresStore(pool)
	must(truncateBusinessData(ctx, pool))
	must(store.EnsureAdmin(ctx, cfg.AdminUsername, cfg.AdminPassword))
	user, err := store.GetUserByUsername(ctx, cfg.AdminUsername)
	must(err)

	state := &seedState{
		ctx:               ctx,
		pool:              pool,
		store:             store,
		user:              user,
		branches:          map[string]int64{},
		suppliers:         map[string]int64{},
		customers:         map[string]int64{},
		expenseCharts:     map[string]int64{},
		otherIncomeCharts: map[string]int64{},
		stocks:            map[string]stockSeed{},
	}

	state.seedMasters()
	state.seedTransactions()
	state.printSummary()
}

func truncateBusinessData(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
		update users set active_branch_id = null;

		delete from dr_consumptions;
		delete from stock_ledger;
		delete from balance_ledger;
		delete from document_lines;
		delete from documents;
		delete from stocks;
		delete from other_income_charts;
		delete from expense_charts;
		delete from customers;
		delete from suppliers;
		delete from stock_categories;
		delete from branches;

		alter sequence dr_consumptions_id_seq restart with 1;
		alter sequence stock_ledger_id_seq restart with 1;
		alter sequence balance_ledger_id_seq restart with 1;
		alter sequence document_lines_id_seq restart with 1;
		alter sequence documents_id_seq restart with 1;
		alter sequence stocks_id_seq restart with 1;
		alter sequence other_income_charts_id_seq restart with 1;
		alter sequence expense_charts_id_seq restart with 1;
		alter sequence customers_id_seq restart with 1;
		alter sequence suppliers_id_seq restart with 1;
		alter sequence stock_categories_id_seq restart with 1;
		alter sequence branches_id_seq restart with 1;
	`)
	return err
}

func (s *seedState) seedMasters() {
	s.seedStockCategories()
	s.seedBranches()
	s.seedSuppliers()
	s.seedCustomers()
	s.seedExpenseCharts()
	s.seedOtherIncomeCharts()
	s.seedStocks()
}

func (s *seedState) seedStockCategories() {
	categories := []map[string]string{
		{"name": "B-MEG Feeds", "group_name": "Feeds", "aps_monitor": "true"},
		{"name": "Pigrolac Feeds", "group_name": "Feeds", "aps_monitor": "true"},
		{"name": "Pilmeco Feeds", "group_name": "Feeds", "aps_monitor": "true"},
		{"name": "Poultry Feeds", "group_name": "Feeds", "aps_monitor": "true"},
		{"name": "Hog Supplements", "group_name": "Supplements", "aps_monitor": "false"},
		{"name": "Veterinary Supplies", "group_name": "Supplies", "aps_monitor": "false"},
		{"name": "Farm Supplies", "group_name": "Supplies", "aps_monitor": "false"},
		{"name": "Raw Materials", "group_name": "Materials", "aps_monitor": "false"},
		{"name": "Packaging", "group_name": "Supplies", "aps_monitor": "false"},
	}
	for _, values := range categories {
		s.saveMaster("stock-categories", values)
	}
}

func (s *seedState) seedBranches() {
	branches := []struct {
		Code     string
		Name     string
		Incharge string
		APS      string
		Farm     bool
	}{
		{"HOF", "Head Office Warehouse", "Mara Santos", "VIP", false},
		{"NWH", "North Warehouse", "Leo Cruz", "APS", false},
		{"SFD", "South Farm Depot", "Tina Flores", "Farm", true},
		{"CFO", "Central Farm Outlet", "Ramon Dela Cruz", "Takals", false},
		{"WBR", "West Bulacan Retail", "Nico Reyes", "VIP", false},
		{"EIB", "East Iloilo Branch", "Grace Tan", "APS", false},
		{"CDH", "Cebu Dealer Hub", "Joel Yap", "Takals", false},
		{"DAS", "Davao Agri Supply", "Mina Lopez", "Farm", true},
	}
	for _, branch := range branches {
		id := s.saveMaster("branches", map[string]string{
			"code":          branch.Code,
			"name":          branch.Name,
			"incharge":      branch.Incharge,
			"aps":           branch.APS,
			"farm_customer": boolString(branch.Farm),
			"remarks":       "Dummy branch for feed, hog, and poultry report testing.",
		})
		s.branches[branch.Code] = id
	}

	s.user.ActiveBranchID = s.branches["HOF"]
	_, err := s.pool.Exec(s.ctx, `update users set active_branch_id=$1 where id=$2`, s.user.ActiveBranchID, s.user.ID)
	must(err)
}

func (s *seedState) seedSuppliers() {
	suppliers := []struct {
		Code, Company, Last, First, Phone, Address string
	}{
		{"BMEG", "San Miguel Foods B-MEG", "Garcia", "Paolo", "0917-100-2001", "Mandaluyong City"},
		{"PIGRO", "Pigrolac Feedmill Corp.", "Sy", "Angela", "0917-100-2002", "Tarlac City"},
		{"PILMECO", "Pilmeco Animal Nutrition", "Ramos", "Victor", "0917-100-2003", "Iligan City"},
		{"VETPLUS", "VetPlus Animal Health", "Lim", "Carla", "0917-100-2004", "Quezon City"},
		{"FARMTECH", "FarmTech Equipment Supply", "Ong", "Edwin", "0917-100-2005", "Valenzuela City"},
		{"RICEBRAN", "Golden Rice Bran Trading", "Villanueva", "Ana", "0917-100-2006", "Nueva Ecija"},
		{"SOYPRO", "SoyPro Ingredients", "Castro", "Migs", "0917-100-2007", "Batangas"},
		{"TRUCKCO", "Agri Haul Logistics", "Dizon", "Mark", "0917-100-2008", "Pampanga"},
		{"PACKPRO", "PackPro Sacks and Twine", "Uy", "Jenn", "0917-100-2009", "Caloocan City"},
		{"MOLPLUS", "Molasses Plus Trading", "Navarro", "Seth", "0917-100-2010", "Negros Occidental"},
	}
	for _, supplier := range suppliers {
		id := s.saveMaster("suppliers", map[string]string{
			"code":         supplier.Code,
			"company":      supplier.Company,
			"lastname":     supplier.Last,
			"firstname":    supplier.First,
			"phone_number": supplier.Phone,
			"address":      supplier.Address,
			"balance":      "0.00",
		})
		s.suppliers[supplier.Code] = id
	}
}

func (s *seedState) seedCustomers() {
	customers := []struct {
		Code, Company, Last, First, Phone, Address, Term, APS string
		LimitCents                                            int64
		Farm                                                  bool
	}{
		{"JHOG", "Juan's Hog Farm", "Dela Cruz", "Juan", "0920-200-3001", "Bulacan", "30 days", "VIP", 25000000, false},
		{"LUNA", "Luna Swine Growers", "Luna", "Carlo", "0920-200-3002", "Tarlac", "30 days", "APS", 18000000, false},
		{"TAKAL", "Takals Farm Supply", "Santos", "Mika", "0920-200-3003", "Pampanga", "15 days", "Takals", 12000000, false},
		{"MABUHAY", "Mabuhay Pig Farm", "Reyes", "Nora", "0920-200-3004", "Nueva Ecija", "45 days", "Farm", 30000000, true},
		{"AGRIDEAL", "AgriCentral Dealer", "Chua", "Ben", "0920-200-3005", "Quezon", "30 days", "VIP", 22000000, false},
		{"NORTHSTAR", "North Star Poultry", "Aquino", "Liza", "0920-200-3006", "Pangasinan", "21 days", "APS", 16000000, false},
		{"DAVHOG", "Davao Hog Raisers", "Lopez", "Ernesto", "0920-200-3007", "Davao", "45 days", "Farm", 26000000, true},
		{"GOLDVAL", "Golden Valley Feedmart", "Tan", "Helen", "0920-200-3008", "Iloilo", "15 days", "Takals", 14000000, false},
		{"CEBUFEED", "Cebu Feed Center", "Yap", "Jonas", "0920-200-3009", "Cebu", "30 days", "APS", 20000000, false},
		{"SOUTHPIG", "Southline Pig Farm", "Flores", "Dina", "0920-200-3010", "Laguna", "30 days", "VIP", 21000000, false},
		{"HAPPYHOG", "Happy Hog Backyard", "Mercado", "Allan", "0920-200-3011", "Rizal", "Cash", "Takals", 7000000, false},
		{"GREENPEN", "Green Pen Farms", "Bautista", "Manny", "0920-200-3012", "Batangas", "30 days", "Farm", 24000000, true},
		{"FIESTA", "Fiesta Feed Retail", "Diaz", "Nelia", "0920-200-3013", "Cavite", "15 days", "VIP", 9000000, false},
		{"AGAPAY", "Agapay Hog Growers", "Cruz", "Rico", "0920-200-3014", "Bicol", "30 days", "APS", 11000000, false},
		{"TRESMARIAS", "Tres Marias Farm", "Soriano", "Leni", "0920-200-3015", "Zambales", "45 days", "Farm", 28000000, true},
		{"PIGPEN", "Pig Pen Cooperative", "Villanueva", "Oscar", "0920-200-3016", "La Union", "30 days", "Takals", 17500000, false},
		{"BARNONE", "Barn One Agri", "Domingo", "Ivy", "0920-200-3017", "Isabela", "30 days", "VIP", 20500000, false},
		{"FEEDMIX", "Feedmix Trading", "Mendoza", "Kiko", "0920-200-3018", "Camarines Sur", "21 days", "APS", 15000000, false},
	}
	for _, customer := range customers {
		id := s.saveMaster("customers", map[string]string{
			"code":          customer.Code,
			"company":       customer.Company,
			"lastname":      customer.Last,
			"firstname":     customer.First,
			"phone_number":  customer.Phone,
			"address":       customer.Address,
			"balance":       "0.00",
			"credit_term":   customer.Term,
			"credit_limit":  money(customer.LimitCents),
			"aps":           customer.APS,
			"farm_customer": boolString(customer.Farm),
		})
		s.customers[customer.Code] = id
	}
}

func (s *seedState) seedExpenseCharts() {
	charts := []struct {
		Code, Name, Description string
		Exclude, DailyOnly      bool
	}{
		{"FREIGHT", "Freight and Delivery", "Truck rental and delivery expenses.", false, false},
		{"FUEL", "Fuel and Lubricants", "Fuel for delivery and warehouse vehicles.", false, false},
		{"LABOR", "Warehouse Labor", "Hauling, stacking, and helper allowances.", false, false},
		{"UTIL", "Utilities", "Electricity, water, and internet.", false, false},
		{"REPAIR", "Repairs and Maintenance", "Warehouse and equipment repairs.", false, false},
		{"REBATE", "Dealer Promo Expense", "Dealer rebates and small promos.", true, false},
		{"MEAL", "Staff Meals", "Daily meal allowance.", false, true},
		{"BANK", "Bank Charges", "Deposit and transfer fees.", false, false},
	}
	for _, chart := range charts {
		id := s.saveMaster("expense-chart", map[string]string{
			"code":                chart.Code,
			"name":                chart.Name,
			"description":         chart.Description,
			"exclude_daily_sales": boolString(chart.Exclude),
			"daily_sales_only":    boolString(chart.DailyOnly),
		})
		s.expenseCharts[chart.Code] = id
	}
}

func (s *seedState) seedOtherIncomeCharts() {
	charts := []struct {
		Code, Name, Description string
	}{
		{"REBINC", "Supplier Rebate Income", "Volume rebate from feed suppliers."},
		{"DELCHG", "Delivery Charge Income", "Delivery charges billed to customers."},
		{"SACKS", "Empty Sack Sales", "Sales of empty feed sacks."},
		{"MISC", "Miscellaneous Income", "Small non-operating income."},
		{"PENALTY", "Late Payment Penalty", "Penalty for overdue accounts."},
		{"SERVICE", "Feed Mixing Service", "Custom feed mixing service income."},
	}
	for _, chart := range charts {
		id := s.saveMaster("other-income-chart", map[string]string{
			"code":        chart.Code,
			"name":        chart.Name,
			"description": chart.Description,
		})
		s.otherIncomeCharts[chart.Code] = id
	}
}

func (s *seedState) seedStocks() {
	stocks := []stockSeed{
		{Code: "BMG-HS50", Name: "B-MEG Hog Starter 50kg", Category: "B-MEG Feeds", CostCents: 172000, PriceCents: 190000},
		{Code: "BMG-HG50", Name: "B-MEG Hog Grower 50kg", Category: "B-MEG Feeds", CostCents: 163000, PriceCents: 181000},
		{Code: "BMG-HF50", Name: "B-MEG Hog Finisher 50kg", Category: "B-MEG Feeds", CostCents: 157500, PriceCents: 175000},
		{Code: "BMG-LAC50", Name: "B-MEG Lactating Sow 50kg", Category: "B-MEG Feeds", CostCents: 168500, PriceCents: 187500},
		{Code: "BMG-PIG25", Name: "B-MEG Piglet Booster 25kg", Category: "B-MEG Feeds", CostCents: 112000, PriceCents: 128000},
		{Code: "BMG-BRL50", Name: "B-MEG Broiler Ration 50kg", Category: "Poultry Feeds", CostCents: 151000, PriceCents: 169000},
		{Code: "PIG-BABY", Name: "Pigrolac Baby Pig Crumble", Category: "Pigrolac Feeds", CostCents: 185000, PriceCents: 207500},
		{Code: "PIG-START", Name: "Pigrolac Starter Pellets", Category: "Pigrolac Feeds", CostCents: 174000, PriceCents: 195000},
		{Code: "PIG-GROW", Name: "Pigrolac Grower Mash", Category: "Pigrolac Feeds", CostCents: 159000, PriceCents: 178000},
		{Code: "PIG-FIN", Name: "Pigrolac Finisher Mash", Category: "Pigrolac Feeds", CostCents: 153000, PriceCents: 171000},
		{Code: "PIG-BREED", Name: "Pigrolac Breeder Pellets", Category: "Pigrolac Feeds", CostCents: 166000, PriceCents: 185000},
		{Code: "PIG-PREMIX", Name: "Pigrolac Premix 10kg", Category: "Hog Supplements", CostCents: 92000, PriceCents: 110000},
		{Code: "PIL-START", Name: "Pilmeco Hog Starter 50kg", Category: "Pilmeco Feeds", CostCents: 171000, PriceCents: 191000},
		{Code: "PIL-GROW", Name: "Pilmeco Hog Grower 50kg", Category: "Pilmeco Feeds", CostCents: 158000, PriceCents: 177000},
		{Code: "PIL-FIN", Name: "Pilmeco Hog Finisher 50kg", Category: "Pilmeco Feeds", CostCents: 150000, PriceCents: 169000},
		{Code: "PIL-SOW", Name: "Pilmeco Gestating Sow 50kg", Category: "Pilmeco Feeds", CostCents: 164000, PriceCents: 184000},
		{Code: "PIL-BRL", Name: "Pilmeco Broiler Booster 50kg", Category: "Poultry Feeds", CostCents: 154000, PriceCents: 173000},
		{Code: "PIL-LYR", Name: "Pilmeco Layer Mash 50kg", Category: "Poultry Feeds", CostCents: 148000, PriceCents: 166000},
		{Code: "VIT-HOG", Name: "Hog Vitamin Mix 1L", Category: "Hog Supplements", CostCents: 42000, PriceCents: 56000},
		{Code: "PROBIO", Name: "Probiotic Feed Additive 1kg", Category: "Hog Supplements", CostCents: 68000, PriceCents: 85000},
		{Code: "IRONDEX", Name: "Iron Dextran 100ml", Category: "Veterinary Supplies", CostCents: 31500, PriceCents: 42500},
		{Code: "DEWORM", Name: "Swine Dewormer 100ml", Category: "Veterinary Supplies", CostCents: 38500, PriceCents: 52000},
		{Code: "DISINF", Name: "Farm Disinfectant 1L", Category: "Farm Supplies", CostCents: 28000, PriceCents: 39000},
		{Code: "NIPPLE", Name: "Hog Nipple Drinker", Category: "Farm Supplies", CostCents: 12000, PriceCents: 18500},
		{Code: "RBRAN", Name: "Premium Rice Bran 30kg", Category: "Raw Materials", CostCents: 78000, PriceCents: 94000},
		{Code: "COPRA", Name: "Copra Meal 50kg", Category: "Raw Materials", CostCents: 98000, PriceCents: 119000},
		{Code: "SOY44", Name: "Soybean Meal 44pct 50kg", Category: "Raw Materials", CostCents: 215000, PriceCents: 246000},
		{Code: "MOL", Name: "Molasses 20L", Category: "Raw Materials", CostCents: 64000, PriceCents: 82000},
		{Code: "SACK50", Name: "Feed Sack 50kg", Category: "Packaging", CostCents: 1700, PriceCents: 2500},
		{Code: "TWINE", Name: "Baling Twine Roll", Category: "Packaging", CostCents: 9000, PriceCents: 13500},
		{Code: "BMG-LOW", Name: "B-MEG Fast Seller Low SOH", Category: "B-MEG Feeds", CostCents: 160000, PriceCents: 179000},
		{Code: "PIG-LOW", Name: "Pigrolac Low Stock Monitor", Category: "Pigrolac Feeds", CostCents: 155000, PriceCents: 174000},
		{Code: "PIL-LOW", Name: "Pilmeco Low Stock Monitor", Category: "Pilmeco Feeds", CostCents: 152000, PriceCents: 171000},
		{Code: "VET-LOW", Name: "Vet Supply Low Stock", Category: "Veterinary Supplies", CostCents: 33000, PriceCents: 45500},
	}

	for idx, stock := range stocks {
		minInventory := int64(35 + (idx%6)*15)
		if stock.Code == "BMG-LOW" || stock.Code == "PIG-LOW" || stock.Code == "PIL-LOW" || stock.Code == "VET-LOW" {
			minInventory = 220
		}
		id := s.saveMaster("stocks", map[string]string{
			"code":           stock.Code,
			"name":           stock.Name,
			"category_group": stock.Category,
			"unit":           "bag",
			"description":    "Dummy inventory item for feed and agri reports.",
			"latest_cost":    money(stock.CostCents),
			"min_inventory":  strconv.FormatInt(minInventory, 10),
		})
		stock.ID = id
		s.stocks[stock.Code] = stock
		s.stockOrder = append(s.stockOrder, stock)
	}
}

func (s *seedState) seedTransactions() {
	s.seedStockIn()
	s.seedPurchases()
	s.seedStockOut()
	s.seedDRFiles()
	s.seedSales()
	s.seedStockTransfers()
	s.seedChecksIn()
	s.seedOtherIncome()
	s.seedExpenses()
	s.seedAPAdjustments()
	s.seedARAdjustments()
}

func (s *seedState) seedStockIn() {
	dates := []string{"2025-11-15", "2026-01-08", "2026-02-17", "2026-03-12", "2026-04-09", "2026-05-03", "2026-05-09", "2026-05-16", "2026-05-23", "2026-05-30"}
	branchCodes := []string{"HOF", "NWH", "SFD", "CFO", "WBR"}
	for i, date := range dates {
		lines := []map[string]string{}
		for j := 0; j < 5; j++ {
			stock := s.stockOrder[(i*3+j)%len(s.stockOrder)]
			qty := int64(35 + ((i+j)%5)*12)
			if stock.Code == "BMG-LOW" || stock.Code == "PIG-LOW" || stock.Code == "PIL-LOW" || stock.Code == "VET-LOW" {
				qty = 18
			}
			lines = append(lines, stockLine(stock, qty, stock.CostCents))
		}
		branch := branchCodes[i%len(branchCodes)]
		s.saveDocument("stock-in", map[string]string{
			"entry_date": date,
			"branch_id":  idString(s.branches[branch]),
			"reference":  fmt.Sprintf("SI-%02d", i+1),
			"remarks":    "Opening and replenishment stock-in dummy data.",
		}, []repositories.LineInput{{Group: "details", Rows: lines}})
	}
}

func (s *seedState) seedPurchases() {
	supplierCodes := []string{"BMEG", "PIGRO", "PILMECO", "VETPLUS", "RICEBRAN", "SOYPRO", "PACKPRO", "MOLPLUS"}
	branchCodes := []string{"HOF", "NWH", "SFD", "CFO", "WBR", "EIB"}
	for i := 0; i < 30; i++ {
		date := mayDate(1 + i%31)
		supplierCode := supplierCodes[i%len(supplierCodes)]
		lines := []map[string]string{}
		for j := 0; j < 3; j++ {
			stock := s.stockOrder[(i*2+j*5)%len(s.stockOrder)]
			qty := int64(20 + ((i+j)%6)*8)
			if i%9 == 0 {
				qty += 25
			}
			lines = append(lines, stockLine(stock, qty, stock.CostCents))
		}
		discounts := []map[string]string{}
		if i%4 == 0 {
			discounts = append(discounts, adjustmentLine("Volume discount", 1, 85000+int64(i%5)*10000))
		}
		additionals := []map[string]string{}
		if i%5 == 0 {
			additionals = append(additionals, adjustmentLine("Freight add-on", 1, 65000+int64(i%3)*12000))
		}
		payments := []map[string]string{}
		if i%3 != 0 {
			payments = append(payments, checkLine(fmt.Sprintf("OUT-%04d", i+1), mayDate(3+(i%27)), bankName(i), 900000+int64(i%7)*125000, "Supplier payment"))
		}
		cash := i%3 == 0
		branch := branchCodes[i%len(branchCodes)]
		s.saveDocument("purchases", map[string]string{
			"party_id":      idString(s.suppliers[supplierCode]),
			"entry_date":    date,
			"branch_id":     idString(s.branches[branch]),
			"reference":     fmt.Sprintf("DR-%s-%03d", supplierCode, i+1),
			"cash":          boolString(cash),
			"or_ci_number":  fmt.Sprintf("CI-%s-%03d", supplierCode, i+1),
			"purchase_date": date,
			"remarks":       "Dummy purchase of feeds, supplements, or farm supplies.",
		}, []repositories.LineInput{
			{Group: "details", Rows: lines},
			{Group: "discounts", Rows: discounts},
			{Group: "additionals", Rows: additionals},
			{Group: "payments", Rows: payments},
		})
	}
}

func (s *seedState) seedStockOut() {
	reasons := []string{"Sampling", "Damaged sacks", "Warehouse shrinkage", "Branch demo", "Farm use", "Expired vet supply"}
	for i := 0; i < 12; i++ {
		lines := []map[string]string{}
		for j := 0; j < 2; j++ {
			stock := s.stockOrder[(i*4+j*7)%len(s.stockOrder)]
			qty := int64(2 + (i+j)%5)
			lines = append(lines, stockLine(stock, qty, stock.CostCents))
		}
		s.saveDocument("stock-out", map[string]string{
			"entry_date": mayDate(2 + i*2%29),
			"branch_id":  idString(s.branches[[]string{"HOF", "NWH", "CFO", "EIB"}[i%4]]),
			"reference":  fmt.Sprintf("SO-%03d", i+1),
			"remarks":    reasons[i%len(reasons)],
		}, []repositories.LineInput{{Group: "details", Rows: lines}})
	}
}

func (s *seedState) seedDRFiles() {
	customerCodes := []string{"JHOG", "LUNA", "TAKAL", "MABUHAY", "AGRIDEAL", "NORTHSTAR", "DAVHOG", "GOLDVAL", "CEBUFEED", "SOUTHPIG", "HAPPYHOG", "GREENPEN", "FIESTA", "AGAPAY", "TRESMARIAS", "PIGPEN", "BARNONE", "FEEDMIX"}
	for i := 0; i < 26; i++ {
		customerCode := customerCodes[i%len(customerCodes)]
		lines := []map[string]string{}
		for j := 0; j < 3; j++ {
			stock := s.stockOrder[(i*3+j*4)%24]
			qty := int64(8 + ((i+j)%6)*5)
			lines = append(lines, drLine(stock, qty))
		}
		id := s.saveDocument("dr", map[string]string{
			"party_id":   idString(s.customers[customerCode]),
			"entry_date": mayDate(1 + i%27),
			"branch_id":  idString(s.branches["HOF"]),
			"reference":  fmt.Sprintf("DR-CUST-%03d", i+1),
			"sales_date": mayDate(1 + i%27),
			"remarks":    "Dummy delivery receipt for customer feed release.",
		}, []repositories.LineInput{{Group: "details", Rows: lines}})
		s.drDocs = append(s.drDocs, drDocSeed{
			ID:           id,
			CustomerID:   s.customers[customerCode],
			CustomerCode: customerCode,
			Reference:    fmt.Sprintf("DR-CUST-%03d", i+1),
			Lines:        s.loadDRLines(id),
		})
	}
}

func (s *seedState) seedSales() {
	branchCodes := []string{"HOF", "NWH", "SFD", "CFO", "WBR", "EIB", "CDH", "DAS"}
	for i := 0; i < 21 && i < len(s.drDocs); i++ {
		dr := &s.drDocs[i]
		rows := []map[string]string{}
		for lineIndex := range dr.Lines {
			line := &dr.Lines[lineIndex]
			if line.Remaining <= 0 {
				continue
			}
			qty := int64(3 + (i+lineIndex)%7)
			if i%6 == 0 {
				qty = line.Remaining
			}
			if qty > line.Remaining {
				qty = line.Remaining
			}
			stock := s.stockByID(line.StockID)
			rows = append(rows, salesBackedLine(stock, line.LineID, qty, i+lineIndex))
			line.Remaining -= qty
		}
		if len(rows) == 0 {
			continue
		}
		deductions := []map[string]string{}
		if i%5 == 0 {
			deductions = append(deductions, adjustmentLine("Promo discount", 1, 35000))
		}
		additionals := []map[string]string{}
		if i%7 == 0 {
			additionals = append(additionals, adjustmentLine("Delivery fee", 1, 25000))
		}
		payments := []map[string]string{}
		cash := i%4 == 0
		if !cash && i%3 != 0 {
			checkDate := mayDate(5 + (i % 25))
			if i%5 == 1 {
				checkDate = futureDate(4 + i)
			}
			payments = append(payments, checkLine(fmt.Sprintf("IN-%04d", i+1), checkDate, bankName(i+4), 650000+int64(i%8)*100000, "Customer check"))
		}
		salesDate := mayDate(4 + i%27)
		if i == 20 {
			salesDate = "2026-05-31"
		}
		s.saveDocument("sales", map[string]string{
			"dr_document_id": idString(dr.ID),
			"party_id":       idString(dr.CustomerID),
			"entry_date":     salesDate,
			"branch_id":      idString(s.branches[branchCodes[i%len(branchCodes)]]),
			"reference":      dr.Reference,
			"cash":           boolString(cash),
			"or_ci_number":   fmt.Sprintf("OR-SALE-%03d", i+1),
			"sales_date":     salesDate,
			"remarks":        "Dummy sales entry generated from DR.",
		}, []repositories.LineInput{
			{Group: "details", Rows: rows},
			{Group: "deductions", Rows: deductions},
			{Group: "additionals", Rows: additionals},
			{Group: "payments", Rows: payments},
		})
	}
}

func (s *seedState) seedStockTransfers() {
	branchCodes := []string{"NWH", "SFD", "CFO", "WBR", "EIB", "CDH", "DAS"}
	for i := 0; i < 16 && i+6 < len(s.drDocs); i++ {
		dr := &s.drDocs[i+6]
		rows := []map[string]string{}
		for lineIndex := range dr.Lines {
			line := &dr.Lines[lineIndex]
			if line.Remaining <= 0 {
				continue
			}
			qty := int64(2 + (i+lineIndex)%5)
			if i%5 == 0 {
				qty = line.Remaining
			}
			if qty > line.Remaining {
				qty = line.Remaining
			}
			stock := s.stockByID(line.StockID)
			rows = append(rows, transferBackedLine(stock, line.LineID, qty, i+lineIndex))
			line.Remaining -= qty
		}
		if len(rows) == 0 {
			continue
		}
		discounts := []map[string]string{}
		if i%4 == 0 {
			discounts = append(discounts, adjustmentLine("Transfer allowance", 1, 18000))
		}
		additionals := []map[string]string{}
		if i%6 == 0 {
			additionals = append(additionals, adjustmentLine("Handling", 1, 12000))
		}
		transferDate := mayDate(8 + i%21)
		if i == 15 {
			transferDate = "2026-05-31"
		}
		targetBranch := branchCodes[i%len(branchCodes)]
		s.saveDocument("stock-transactions", map[string]string{
			"dr_document_id":  idString(dr.ID),
			"entry_date":      transferDate,
			"branch_id":       idString(s.branches["HOF"]),
			"reference":       dr.Reference,
			"transfer_date":   transferDate,
			"transfer_id":     fmt.Sprintf("TR-%03d", i+1),
			"transaction":     "Branch transfer",
			"branch_location": idString(s.branches[targetBranch]),
			"remarks":         "Dummy stock transfer from head office to branch.",
		}, []repositories.LineInput{
			{Group: "details", Rows: rows},
			{Group: "discounts", Rows: discounts},
			{Group: "additionals", Rows: additionals},
		})
	}
}

func (s *seedState) seedChecksIn() {
	for i := 0; i < 9; i++ {
		secondDate := mayDate(2 + i*3%30)
		if i%3 == 0 {
			secondDate = futureDate(8 + i)
		}
		rows := []map[string]string{
			checkLine(fmt.Sprintf("CHK-IN-%03dA", i+1), mayDate(1+i*3%31), bankName(i), 420000+int64(i)*35000, "Post-dated customer check"),
			checkLine(fmt.Sprintf("CHK-IN-%03dB", i+1), secondDate, bankName(i+1), 265000+int64(i)*28000, "Deposit clearing"),
		}
		s.saveDocument("checks-in", map[string]string{
			"entry_date": mayDate(1 + i*3%31),
			"branch_id":  idString(s.branches["HOF"]),
			"reference":  fmt.Sprintf("CI-CHK-%03d", i+1),
			"remarks":    "Dummy incoming check batch.",
		}, []repositories.LineInput{{Group: "checks", Rows: rows}})
	}
}

func (s *seedState) seedOtherIncome() {
	codes := []string{"REBINC", "DELCHG", "SACKS", "MISC", "PENALTY", "SERVICE"}
	for i := 0; i < 12; i++ {
		rows := []map[string]string{}
		for j := 0; j < 2; j++ {
			code := codes[(i+j)%len(codes)]
			cash := int64(120000 + (i+j)%5*35000)
			check := int64(0)
			if (i+j)%3 == 0 {
				check = 90000 + int64(i%4)*25000
			}
			rows = append(rows, moneyLine(s.otherIncomeCharts[code], fmt.Sprintf("OI-%03d-%d", i+1, j+1), cash, check))
		}
		s.saveDocument("other-income", map[string]string{
			"entry_date": mayDate(3 + i*2%29),
			"branch_id":  idString(s.branches[[]string{"HOF", "NWH", "CFO", "CDH"}[i%4]]),
			"reference":  fmt.Sprintf("OI-%03d", i+1),
			"remarks":    "Dummy other income for reports.",
		}, []repositories.LineInput{{Group: "details", Rows: rows}})
	}
}

func (s *seedState) seedExpenses() {
	codes := []string{"FREIGHT", "FUEL", "LABOR", "UTIL", "REPAIR", "REBATE", "MEAL", "BANK"}
	for i := 0; i < 16; i++ {
		rows := []map[string]string{}
		for j := 0; j < 2; j++ {
			code := codes[(i+j)%len(codes)]
			cash := int64(75000 + int64((i+j)%6)*18000)
			check := int64(0)
			if (i+j)%2 == 0 {
				check = 55000 + int64(i%5)*15000
			}
			rows = append(rows, moneyLine(s.expenseCharts[code], fmt.Sprintf("EXP-%03d-%d", i+1, j+1), cash, check))
		}
		expenseDate := mayDate(2 + i*2%30)
		if i == 15 {
			expenseDate = "2026-05-31"
		}
		s.saveDocument("expenses", map[string]string{
			"entry_date": expenseDate,
			"branch_id":  idString(s.branches[[]string{"HOF", "NWH", "SFD", "CFO", "EIB"}[i%5]]),
			"reference":  fmt.Sprintf("EXP-%03d", i+1),
			"remarks":    "Dummy operating expense.",
		}, []repositories.LineInput{{Group: "details", Rows: rows}})
	}
}

func (s *seedState) seedAPAdjustments() {
	supplierCodes := []string{"BMEG", "PIGRO", "PILMECO", "VETPLUS", "RICEBRAN", "SOYPRO", "PACKPRO", "MOLPLUS"}
	for i := 0; i < 8; i++ {
		supplierID := s.suppliers[supplierCodes[i%len(supplierCodes)]]
		s.saveDocument("ap-credit", map[string]string{
			"entry_date":  mayDate(6 + i*3%24),
			"party_id":    idString(supplierID),
			"reference":   fmt.Sprintf("APC-%03d", i+1),
			"cash_amount": money(220000 + int64(i)*40000),
			"remarks":     "Dummy AP payment.",
		}, []repositories.LineInput{{Group: "checks", Rows: []map[string]string{
			checkLine(fmt.Sprintf("AP-CHK-%03d", i+1), mayDate(7+i*3%23), bankName(i+2), 180000+int64(i)*25000, "AP check payment"),
		}}})
		s.saveDocument("ap-debit", map[string]string{
			"entry_date": mayDate(8 + i*2%22),
			"party_id":   idString(supplierID),
			"amount":     money(145000 + int64(i)*30000),
			"reference":  fmt.Sprintf("APD-%03d", i+1),
			"remarks":    "Dummy AP adjustment debit.",
		}, nil)
	}
}

func (s *seedState) seedARAdjustments() {
	customerCodes := []string{"JHOG", "LUNA", "TAKAL", "MABUHAY", "AGRIDEAL", "NORTHSTAR", "DAVHOG", "GOLDVAL"}
	for i := 0; i < 8; i++ {
		customerID := s.customers[customerCodes[i%len(customerCodes)]]
		s.saveDocument("ar-credit", map[string]string{
			"entry_date":  mayDate(5 + i*3%25),
			"party_id":    idString(customerID),
			"reference":   fmt.Sprintf("ARC-%03d", i+1),
			"cash_amount": money(160000 + int64(i)*35000),
			"remarks":     "Dummy AR collection.",
		}, []repositories.LineInput{{Group: "checks", Rows: []map[string]string{
			checkLine(fmt.Sprintf("AR-CHK-%03d", i+1), arCheckDate(i), bankName(i), 210000+int64(i)*30000, "AR collection check"),
		}}})
		s.saveDocument("ar-debit", map[string]string{
			"entry_date": mayDate(7 + i*2%23),
			"party_id":   idString(customerID),
			"amount":     money(90000 + int64(i)*22000),
			"reference":  fmt.Sprintf("ARD-%03d", i+1),
			"remarks":    "Dummy AR adjustment debit.",
		}, nil)
		s.saveDocument("rebates", map[string]string{
			"entry_date":  mayDate(9 + i*2%21),
			"party_id":    idString(customerID),
			"reference":   fmt.Sprintf("REB-%03d", i+1),
			"cash_amount": money(65000 + int64(i)*15000),
			"remarks":     "Dummy customer rebate.",
		}, []repositories.LineInput{{Group: "checks", Rows: []map[string]string{
			checkLine(fmt.Sprintf("REB-CHK-%03d", i+1), mayDate(10+i*2%20), bankName(i+3), 45000+int64(i)*10000, "Rebate check"),
		}}})
	}
}

func (s *seedState) saveMaster(kind string, values map[string]string) int64 {
	form, ok := models.FindMaster(kind)
	if !ok {
		log.Fatalf("unknown master form %q", kind)
	}
	id, err := s.store.SaveMaster(s.ctx, form, 0, values, s.user)
	must(err)
	return id
}

func (s *seedState) saveDocument(kind string, values map[string]string, groups []repositories.LineInput) int64 {
	form, ok := models.FindTransaction(kind)
	if !ok {
		log.Fatalf("unknown transaction form %q", kind)
	}
	id, err := s.store.SaveDocument(s.ctx, form, 0, repositories.DocumentInput{
		Kind:      kind,
		Values:    values,
		LineInput: groups,
		User:      s.user,
	})
	must(err)
	return id
}

func (s *seedState) loadDRLines(documentID int64) []drLineSeed {
	rows, err := s.pool.Query(s.ctx, `
		select id, stock_id, round(qty)::bigint
		from document_lines
		where document_id=$1 and group_key='details'
		order by line_no`, documentID)
	must(err)
	defer rows.Close()

	var out []drLineSeed
	for rows.Next() {
		var line drLineSeed
		must(rows.Scan(&line.LineID, &line.StockID, &line.Remaining))
		line.StockCode = s.stockByID(line.StockID).Code
		out = append(out, line)
	}
	must(rows.Err())
	return out
}

func (s *seedState) stockByID(id int64) stockSeed {
	for _, stock := range s.stocks {
		if stock.ID == id {
			return stock
		}
	}
	log.Fatalf("unknown stock id %d", id)
	return stockSeed{}
}

func (s *seedState) printSummary() {
	fmt.Println("Dummy data seed complete.")
	fmt.Println("Business tables were truncated; the admin login is available.")
	fmt.Println()
	fmt.Println("Document counts:")
	rows, err := s.pool.Query(s.ctx, `
		select kind, count(*)
		from documents
		group by kind
		order by kind`)
	must(err)
	defer rows.Close()
	for rows.Next() {
		var kind string
		var count int64
		must(rows.Scan(&kind, &count))
		fmt.Printf("  %-20s %d\n", kind, count)
	}
	must(rows.Err())

	fmt.Println()
	fmt.Println("Master counts:")
	counts := []struct {
		Label string
		SQL   string
	}{
		{"branches", "select count(*) from branches"},
		{"stock_categories", "select count(*) from stock_categories"},
		{"suppliers", "select count(*) from suppliers"},
		{"customers", "select count(*) from customers"},
		{"stocks", "select count(*) from stocks"},
		{"expense_charts", "select count(*) from expense_charts"},
		{"other_income_charts", "select count(*) from other_income_charts"},
	}
	for _, item := range counts {
		var count int64
		must(s.pool.QueryRow(s.ctx, item.SQL).Scan(&count))
		fmt.Printf("  %-20s %d\n", item.Label, count)
	}
}

func stockLine(stock stockSeed, qty, unitCostCents int64) map[string]string {
	return map[string]string{
		"stock_id":  idString(stock.ID),
		"qty":       strconv.FormatInt(qty, 10),
		"unit_cost": money(unitCostCents),
		"amount":    money(qty * unitCostCents),
	}
}

func drLine(stock stockSeed, qty int64) map[string]string {
	return map[string]string{
		"stock_id": idString(stock.ID),
		"qty":      strconv.FormatInt(qty, 10),
	}
}

func salesBackedLine(stock stockSeed, drLineID, qty int64, seed int) map[string]string {
	price := stock.PriceCents + int64(seed%4)*2500
	amount := qty * price
	discount := int64(seed%3) * 1200
	otherDiscount := int64(seed%2) * 800
	net := amount - discount - otherDiscount
	capital := qty * stock.CostCents
	markup := net - capital
	markupPct := int64(0)
	if capital > 0 {
		markupPct = markup * 10000 / capital
	}
	return map[string]string{
		"dr_line_id":     idString(drLineID),
		"stock_id":       idString(stock.ID),
		"stock_label":    stock.Code + " - " + stock.Name,
		"qty":            strconv.FormatInt(qty, 10),
		"unit_cost":      money(price),
		"price":          money(price),
		"amount":         money(amount),
		"capital":        money(capital),
		"discount":       money(discount),
		"other_discount": money(otherDiscount),
		"markup":         money(markup),
		"markup_pct":     money(markupPct),
	}
}

func transferBackedLine(stock stockSeed, drLineID, qty int64, seed int) map[string]string {
	unit := stock.CostCents + int64(seed%5)*1800
	amount := qty * unit
	capital := qty * stock.CostCents
	markup := amount - capital
	markupPct := int64(0)
	if capital > 0 {
		markupPct = markup * 10000 / capital
	}
	return map[string]string{
		"dr_line_id":  idString(drLineID),
		"stock_id":    idString(stock.ID),
		"stock_label": stock.Code + " - " + stock.Name,
		"qty":         strconv.FormatInt(qty, 10),
		"unit_cost":   money(unit),
		"amount":      money(amount),
		"capital":     money(capital),
		"markup":      money(markup),
		"markup_pct":  money(markupPct),
	}
}

func adjustmentLine(particulars string, qty, priceCents int64) map[string]string {
	return map[string]string{
		"particulars": particulars,
		"qty":         strconv.FormatInt(qty, 10),
		"price":       money(priceCents),
		"amount":      money(qty * priceCents),
	}
}

func checkLine(number, date, bank string, amountCents int64, nature string) map[string]string {
	return map[string]string{
		"number":    number,
		"date":      date,
		"bank_name": bank,
		"amount":    money(amountCents),
		"nature":    nature,
	}
}

func moneyLine(codeID int64, reference string, cashCents, checkCents int64) map[string]string {
	return map[string]string{
		"code_id":   idString(codeID),
		"reference": reference,
		"cash":      money(cashCents),
		"check":     money(checkCents),
		"total":     money(cashCents + checkCents),
	}
}

func bankName(seed int) string {
	banks := []string{"BDO", "BPI", "Metrobank", "RCBC", "UnionBank", "Landbank", "PNB", "Security Bank"}
	return banks[seed%len(banks)]
}

func mayDate(day int) string {
	if day < 1 {
		day = 1
	}
	if day > 31 {
		day = 31
	}
	return fmt.Sprintf("2026-05-%02d", day)
}

func futureDate(day int) string {
	if day < 1 {
		day = 1
	}
	if day > 30 {
		day = 30
	}
	return fmt.Sprintf("2026-06-%02d", day)
}

func arCheckDate(seed int) string {
	if seed >= 4 {
		return futureDate(12 + seed)
	}
	return mayDate(6 + seed*3%24)
}

func idString(id int64) string {
	return strconv.FormatInt(id, 10)
}

func boolString(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func money(cents int64) string {
	sign := ""
	if cents < 0 {
		sign = "-"
		cents = -cents
	}
	return fmt.Sprintf("%s%d.%02d", sign, cents/100, cents%100)
}

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
