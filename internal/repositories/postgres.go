package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"cims-go/internal/auth"
	"cims-go/internal/models"
	"cims-go/internal/services"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	pool *pgxpool.Pool
}

type LineInput struct {
	Group string
	Rows  []map[string]string
}

type DocumentInput struct {
	Kind      string
	Values    map[string]string
	LineInput []LineInput
	User      models.User
}

type DRSelection struct {
	Values models.Record
	Rows   []models.Record
}

type drLineState struct {
	StockID      int64
	RemainingQty int64
}

type drValidationState struct {
	PartyID int64
	Lines   map[int64]drLineState
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

func (s *PostgresStore) EnsureAdmin(ctx context.Context, username, password string) error {
	var exists bool
	if err := s.pool.QueryRow(ctx, `select exists(select 1 from users where username=$1)`, username).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return nil
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `
		insert into users (username, password_hash, display_name, role)
		values ($1, $2, $3, 'admin')`, username, hash, "Administrator")
	return err
}

func (s *PostgresStore) GetUserByUsername(ctx context.Context, username string) (models.User, error) {
	return s.getUser(ctx, `where username=$1`, username)
}

func (s *PostgresStore) GetUserByID(ctx context.Context, id int64) (models.User, error) {
	return s.getUser(ctx, `where id=$1`, id)
}

func (s *PostgresStore) getUser(ctx context.Context, where string, args ...any) (models.User, error) {
	var user models.User
	var activeBranchID *int64
	err := s.pool.QueryRow(ctx, `
		select id, username, password_hash, display_name, role, active_branch_id
		from users `+where, args...).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.DisplayName, &user.Role, &activeBranchID)
	if err != nil {
		return models.User{}, err
	}
	if activeBranchID != nil {
		user.ActiveBranchID = *activeBranchID
	}
	return user, nil
}

func (s *PostgresStore) ListMaster(ctx context.Context, form models.FormDefinition) ([]models.Record, error) {
	cols := []string{"m.id::text as id"}
	for _, field := range form.Fields {
		cols = append(cols, fmt.Sprintf("coalesce(m.%s::text, '') as %s", field.Column, field.Key))
	}
	cols = append(cols, "coalesce(u.display_name, '') as encoder")
	sql := fmt.Sprintf(`
		select %s
		from %s m
		left join users u on u.id = m.encoder_user_id
		order by m.id desc
		limit 200`, strings.Join(cols, ", "), form.Table)
	rows, err := s.pool.Query(ctx, sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRecords(rows)
}

func (s *PostgresStore) GetMaster(ctx context.Context, form models.FormDefinition, id int64) (models.Record, error) {
	cols := []string{"id::text as id"}
	for _, field := range form.Fields {
		cols = append(cols, fmt.Sprintf("coalesce(%s::text, '') as %s", field.Column, field.Key))
	}
	sql := fmt.Sprintf(`select %s from %s where id=$1`, strings.Join(cols, ", "), form.Table)
	rows, err := s.pool.Query(ctx, sql, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	records, err := scanRecords(rows)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, pgx.ErrNoRows
	}
	return records[0], nil
}

func (s *PostgresStore) SaveMaster(ctx context.Context, form models.FormDefinition, id int64, values map[string]string, user models.User) (int64, error) {
	if !user.CanWrite() {
		return 0, errors.New("write access required")
	}
	columns := make([]string, 0, len(form.Fields)+2)
	args := make([]any, 0, len(form.Fields)+3)
	for _, field := range form.Fields {
		columns = append(columns, field.Column)
		args = append(args, valueForField(field, values[field.Key]))
	}
	if id == 0 {
		columns = append(columns, "encoder_user_id", "last_update_by_user_id")
		args = append(args, user.ID, user.ID)
		placeholders := make([]string, len(columns))
		for i := range placeholders {
			placeholders[i] = "$" + strconv.Itoa(i+1)
		}
		sql := fmt.Sprintf(`insert into %s (%s) values (%s) returning id`, form.Table, strings.Join(columns, ", "), strings.Join(placeholders, ", "))
		if err := s.pool.QueryRow(ctx, sql, args...).Scan(&id); err != nil {
			return 0, err
		}
		return id, nil
	}

	set := make([]string, 0, len(columns)+1)
	for i, column := range columns {
		set = append(set, fmt.Sprintf("%s=$%d", column, i+1))
	}
	args = append(args, user.ID, id)
	set = append(set, fmt.Sprintf("last_update_by_user_id=$%d", len(args)-1), "updated_at=now()")
	sql := fmt.Sprintf(`update %s set %s where id=$%d`, form.Table, strings.Join(set, ", "), len(args))
	if _, err := s.pool.Exec(ctx, sql, args...); err != nil {
		return 0, err
	}
	return id, nil
}

func (s *PostgresStore) ListDocuments(ctx context.Context, kind string) ([]models.DocumentListItem, error) {
	rows, err := s.pool.Query(ctx, `
		select d.id, d.entry_id, d.entry_date,
		       coalesce(nullif(s.company, ''), nullif(c.company, ''), s.code, c.code, ''),
		       coalesce(b.name, ''),
		       coalesce(d.reference, ''),
		       coalesce(dr.entry_id, ''),
		       case
		         when d.kind <> 'dr' then ''
		         when coalesce((
		           select sum(dl.qty)
		           from document_lines dl
		           where dl.document_id = d.id and dl.group_key = 'details'
		         ), 0) = 0 then ''
		         when coalesce((
		           select sum(dc.consumed_qty)
		           from dr_consumptions dc
		           where dc.dr_document_id = d.id
		         ), 0) = 0 then 'Open'
		         when coalesce((
		           select sum(dc.consumed_qty)
		           from dr_consumptions dc
		           where dc.dr_document_id = d.id
		         ), 0) >= coalesce((
		           select sum(dl.qty)
		           from document_lines dl
		           where dl.document_id = d.id and dl.group_key = 'details'
		         ), 0) then 'Fully Used'
		         else 'Partial'
		       end,
		       coalesce(d.net::text, '0'), coalesce(u.display_name, '')
		from documents d
		left join branches b on b.id = d.branch_id
		left join users u on u.id = d.encoder_user_id
		left join suppliers s on d.party_type='supplier' and s.id=d.party_id
		left join customers c on d.party_type='customer' and c.id=d.party_id
		left join documents dr on dr.id = d.dr_reference_id
		where d.kind=$1
		order by d.id desc
		limit 200`, kind)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []models.DocumentListItem
	for rows.Next() {
		var item models.DocumentListItem
		if err := rows.Scan(&item.ID, &item.EntryID, &item.EntryDate, &item.Party, &item.Branch, &item.Reference, &item.DRRef, &item.Status, &item.Net, &item.Encoder); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) LoadDRSelection(ctx context.Context, id int64) (DRSelection, error) {
	selection := DRSelection{Values: models.Record{}}
	var partyID int64
	var reference string
	var entryDate string
	var remarks string
	var salesDate string
	err := s.pool.QueryRow(ctx, `
		select party_id, coalesce(reference, ''), entry_date::date::text, coalesce(remarks, ''), coalesce(payload->'values'->>'sales_date', '')
		from documents
		where id=$1 and kind='dr'`, id).Scan(&partyID, &reference, &entryDate, &remarks, &salesDate)
	if err != nil {
		return DRSelection{}, err
	}
	selection.Values["dr_document_id"] = strconv.FormatInt(id, 10)
	selection.Values["party_id"] = strconv.FormatInt(partyID, 10)
	selection.Values["reference"] = reference
	selection.Values["entry_date"] = entryDate
	selection.Values["remarks"] = remarks
	selection.Values["sales_date"] = salesDate
	rows, err := s.pool.Query(ctx, `
		select dl.id::text,
		       dl.stock_id::text,
		       coalesce(st.code || ' - ' || st.name, ''),
		       ((dl.qty - coalesce((
		         select sum(dc.consumed_qty)
		         from dr_consumptions dc
		         where dc.dr_line_id = dl.id
		       ), 0)))::bigint::text
		from document_lines dl
		left join stocks st on st.id = dl.stock_id
		where dl.document_id=$1
		  and dl.group_key='details'
		  and (dl.qty - coalesce((
		    select sum(dc.consumed_qty)
		    from dr_consumptions dc
		    where dc.dr_line_id = dl.id
		  ), 0)) > 0
		order by dl.line_no`, id)
	if err != nil {
		return DRSelection{}, err
	}
	defer rows.Close()
	for rows.Next() {
		row := models.Record{}
		var drLineID string
		var stockID string
		var stockLabel string
		var qty string
		if err := rows.Scan(&drLineID, &stockID, &stockLabel, &qty); err != nil {
			return DRSelection{}, err
		}
		row["dr_line_id"] = drLineID
		row["stock_id"] = stockID
		row["stock_label"] = stockLabel
		row["qty"] = qty
		selection.Rows = append(selection.Rows, row)
	}
	if err := rows.Err(); err != nil {
		return DRSelection{}, err
	}
	if len(selection.Rows) == 0 {
		return DRSelection{}, errors.New("selected DR has no remaining quantity")
	}
	return selection, nil
}

func (s *PostgresStore) validateDRReference(ctx context.Context, tx pgx.Tx, kind string, drReferenceID int64, values map[string]string, groups []LineInput) (drValidationState, error) {
	state := drValidationState{Lines: map[int64]drLineState{}}
	err := tx.QueryRow(ctx, `
		select party_id
		from documents
		where id=$1 and kind='dr'`, drReferenceID).Scan(&state.PartyID)
	if err != nil {
		return drValidationState{}, err
	}
	if kind == "sales" && parseInt(values["party_id"]) != 0 && parseInt(values["party_id"]) != state.PartyID {
		return drValidationState{}, errors.New("customer must match selected DR")
	}
	rows, err := tx.Query(ctx, `
		select dl.id,
		       dl.stock_id,
		       ((dl.qty - coalesce((
		         select sum(dc.consumed_qty)
		         from dr_consumptions dc
		         where dc.dr_line_id = dl.id
		       ), 0)))::bigint
		from document_lines dl
		where dl.document_id=$1
		  and dl.group_key='details'
		  and (dl.qty - coalesce((
		    select sum(dc.consumed_qty)
		    from dr_consumptions dc
		    where dc.dr_line_id = dl.id
		  ), 0)) > 0`, drReferenceID)
	if err != nil {
		return drValidationState{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var lineID, stockID, remainingQty int64
		if err := rows.Scan(&lineID, &stockID, &remainingQty); err != nil {
			return drValidationState{}, err
		}
		state.Lines[lineID] = drLineState{StockID: stockID, RemainingQty: remainingQty}
	}
	if err := rows.Err(); err != nil {
		return drValidationState{}, err
	}
	var sawDetail bool
	for _, group := range groups {
		if group.Group != "details" {
			continue
		}
		for _, row := range group.Rows {
			if rowIsBlank(row) {
				continue
			}
			sawDetail = true
			drLineID := parseInt(row["dr_line_id"])
			line, ok := state.Lines[drLineID]
			if !ok {
				return drValidationState{}, errors.New("selected DR row is no longer available")
			}
			stockID := parseInt(row["stock_id"])
			if stockID == 0 || stockID != line.StockID {
				return drValidationState{}, errors.New("stock does not match selected DR")
			}
			qty := parseInt(row["qty"])
			if qty <= 0 || qty > line.RemainingQty {
				return drValidationState{}, errors.New("quantity exceeds remaining DR quantity")
			}
		}
	}
	if !sawDetail {
		return drValidationState{}, errors.New("at least one DR line is required")
	}
	return state, nil
}

func (s *PostgresStore) insertDRConsumption(ctx context.Context, tx pgx.Tx, state drValidationState, drReferenceID, consumerDocumentID, consumerLineID int64, row map[string]string) error {
	drLineID := parseInt(row["dr_line_id"])
	if drLineID == 0 {
		return errors.New("missing DR line reference")
	}
	line, ok := state.Lines[drLineID]
	if !ok {
		return errors.New("selected DR row is no longer available")
	}
	qty := parseInt(row["qty"])
	if qty <= 0 || qty > line.RemainingQty {
		return errors.New("quantity exceeds remaining DR quantity")
	}
	_, err := tx.Exec(ctx, `
		insert into dr_consumptions (dr_document_id, dr_line_id, consumer_document_id, consumer_line_id, consumed_qty)
		values ($1, $2, $3, $4, $5)`,
		drReferenceID, drLineID, consumerDocumentID, consumerLineID, centsToNumeric(qty*100))
	return err
}

func (s *PostgresStore) SaveDocument(ctx context.Context, form models.FormDefinition, input DocumentInput) (int64, error) {
	if !input.User.CanWrite() {
		return 0, errors.New("write access required")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	entryDate := nullableDate(input.Values["entry_date"])
	if entryDate == nil {
		now := time.Now()
		entryDate = &now
	}
	branchID := parseInt(input.Values["branch_id"])
	if branchID == 0 {
		branchID = input.User.ActiveBranchID
	}
	input.Values["branch_id"] = strconv.FormatInt(branchID, 10)
	partyID := parseInt(input.Values["party_id"])
	payload, err := json.Marshal(struct {
		Kind      string            `json:"kind"`
		Values    map[string]string `json:"values"`
		LineInput []LineInput       `json:"line_input"`
		EncoderID int64             `json:"encoder_id"`
	}{
		Kind:      input.Kind,
		Values:    input.Values,
		LineInput: input.LineInput,
		EncoderID: input.User.ID,
	})
	if err != nil {
		return 0, err
	}

	totalInput := buildTotalsInput(form.Kind, input.Values, input.LineInput)
	effects := services.BuildPostingEffects(totalInput.posting)
	entryID := ""
	drReferenceID := parseInt(input.Values["dr_document_id"])
	var drState drValidationState
	if drReferenceID != 0 {
		drState, err = s.validateDRReference(ctx, tx, form.Kind, drReferenceID, input.Values, input.LineInput)
		if err != nil {
			return 0, err
		}
	}
	err = tx.QueryRow(ctx, `
		insert into documents
			(kind, entry_date, branch_id, party_type, party_id, reference, cash, remarks, total, less_amount, add_amount, net, balance, payload, dr_reference_id, encoder_user_id, last_update_by_user_id)
		values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$16)
		returning id, entry_id`,
		form.Kind, entryDate, nullInt(branchID), emptyToNil(form.PartyType), nullInt(partyID), emptyToNil(input.Values["reference"]),
		parseBool(input.Values["cash"]), input.Values["remarks"], centsToNumeric(totalInput.total), centsToNumeric(totalInput.less),
		centsToNumeric(totalInput.add), centsToNumeric(totalInput.net), centsToNumeric(totalInput.balance), payload, nullInt(drReferenceID), input.User.ID,
	).Scan(&totalInput.documentID, &entryID)
	if err != nil {
		return 0, err
	}

	for _, group := range input.LineInput {
		for idx, row := range group.Rows {
			if rowIsBlank(row) {
				continue
			}
			linePayload, err := json.Marshal(row)
			if err != nil {
				return 0, err
			}
			var lineID int64
			err = tx.QueryRow(ctx, `
				insert into document_lines (document_id, group_key, line_no, stock_id, code_id, qty, unit_cost, price, cash_amount, check_amount, amount, payload, dr_line_id)
				values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
				returning id`,
				totalInput.documentID, group.Group, idx+1, nullInt(parseInt(row["stock_id"])), nullInt(parseInt(row["code_id"])),
				parseInt(row["qty"]), centsToNumeric(parseMoney(row["unit_cost"])), centsToNumeric(parseMoney(row["price"])),
				centsToNumeric(parseMoney(row["cash"])), centsToNumeric(parseMoney(row["check"])), centsToNumeric(parseMoney(row["amount"])), linePayload, nullInt(parseInt(row["dr_line_id"]))).Scan(&lineID)
			if err != nil {
				return 0, err
			}
			if group.Group == "details" && drReferenceID != 0 {
				if err := s.insertDRConsumption(ctx, tx, drState, drReferenceID, totalInput.documentID, lineID, row); err != nil {
					return 0, err
				}
			}
		}
	}

	for _, effect := range effects.Inventory {
		_, err = tx.Exec(ctx, `
			insert into stock_ledger (document_id, branch_id, stock_id, qty_delta, unit_cost)
			values ($1,$2,$3,$4,$5)`,
			totalInput.documentID, nullInt(effect.BranchID), effect.StockID, effect.QtyDelta, centsToNumeric(effect.Cost))
		if err != nil {
			return 0, err
		}
	}
	if effects.Balance.PartyType != services.PartyNone && effects.Balance.PartyID != 0 && effects.Balance.AmountDelta != 0 {
		_, err = tx.Exec(ctx, `
			insert into balance_ledger (document_id, party_type, party_id, amount_delta)
			values ($1,$2,$3,$4)`,
			totalInput.documentID, effects.Balance.PartyType, effects.Balance.PartyID, centsToNumeric(effects.Balance.AmountDelta))
		if err != nil {
			return 0, err
		}
		table := "customers"
		if effects.Balance.PartyType == services.PartySupplier {
			table = "suppliers"
		}
		_, err = tx.Exec(ctx, fmt.Sprintf(`update %s set balance = coalesce(balance, 0) + $1 where id=$2`, table), centsToNumeric(effects.Balance.AmountDelta), effects.Balance.PartyID)
		if err != nil {
			return 0, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	_ = entryID
	return totalInput.documentID, nil
}

func (s *PostgresStore) Options(ctx context.Context, source string) ([]models.Option, error) {
	query := optionQuery(source)
	if query == "" {
		return nil, nil
	}
	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var options []models.Option
	for rows.Next() {
		var option models.Option
		if err := rows.Scan(&option.Value, &option.Label); err != nil {
			return nil, err
		}
		options = append(options, option)
	}
	return options, rows.Err()
}

func optionQuery(source string) string {
	switch source {
	case "branches":
		return `select id::text, code || ' - ' || name from branches order by code`
	case "suppliers":
		return `select id::text, coalesce(nullif(company,''), code) from suppliers order by code`
	case "customers":
		return `select id::text, coalesce(nullif(company,''), code) from customers order by code`
	case "stocks":
		return `select id::text, code || ' - ' || name from stocks order by code`
	case "dr_documents":
		return `
			select d.id::text,
			       d.entry_id || ' - ' || coalesce(nullif(c.company,''), c.code, '') || ' - ' || coalesce(d.reference, 'DR')
			from documents d
			left join customers c on c.id = d.party_id
			where d.kind='dr'
			  and exists (
			    select 1
			    from document_lines dl
			    where dl.document_id = d.id
			      and dl.group_key = 'details'
			      and (dl.qty - coalesce((
			        select sum(dc.consumed_qty)
			        from dr_consumptions dc
			        where dc.dr_line_id = dl.id
			      ), 0)) > 0
			  )
			order by d.id desc`
	case "stock_category_groups":
		return `select distinct group_name, group_name from stock_categories where coalesce(group_name, '') <> '' order by group_name`
	case "expense_charts":
		return `select id::text, code || ' - ' || name from expense_charts order by code`
	case "other_income_charts":
		return `select id::text, code || ' - ' || name from other_income_charts order by code`
	default:
		return ""
	}
}

type totalsInput struct {
	documentID int64
	total      int64
	less       int64
	add        int64
	net        int64
	balance    int64
	posting    services.PostingRequest
}

func buildTotalsInput(kind string, values map[string]string, groups []LineInput) totalsInput {
	out := totalsInput{}
	switch kind {
	case "purchases":
		doc := services.PurchaseDocument{
			Cash:        parseBool(values["cash"]),
			Lines:       stockLinesFromGroups(groups),
			Discounts:   adjustmentsFromGroup(groups, "discounts"),
			Additionals: adjustmentsFromGroup(groups, "additionals"),
			Payments:    paymentsFromGroups(groups, "payments", parseMoney(values["cash_amount"])),
		}
		total := services.ComputePurchase(doc)
		out.total, out.less, out.add, out.net, out.balance = total.Total, total.Less, total.Add, total.Net, total.Balance
		out.posting = services.PostingRequest{Kind: services.DocumentPurchase, BranchID: parseInt(values["branch_id"]), PartyID: parseInt(values["party_id"]), Lines: doc.Lines, Net: total.Net, Balance: total.Balance, Paid: total.Paid}
	case "sales":
		doc := services.SalesDocument{
			Cash:        parseBool(values["cash"]),
			Lines:       salesLinesFromGroups(groups),
			Deductions:  adjustmentsFromGroup(groups, "deductions"),
			Additionals: adjustmentsFromGroup(groups, "additionals"),
			Payments:    paymentsFromGroups(groups, "payments", parseMoney(values["cash_amount"])),
		}
		total := services.ComputeSales(doc)
		out.total, out.less, out.add, out.net, out.balance = total.TotalNetAmount, total.Less, total.Add, total.Net, total.Balance
		out.posting = services.PostingRequest{Kind: services.DocumentSale, BranchID: parseInt(values["branch_id"]), PartyID: parseInt(values["party_id"]), Lines: stockLinesFromSales(doc.Lines), Net: total.Net, Balance: total.Balance, Paid: total.Paid}
	case "stock-in":
		lines := stockLinesFromGroups(groups)
		out.total = stockLineTotal(lines)
		out.net = out.total
		out.posting = services.PostingRequest{Kind: services.DocumentStockIn, BranchID: parseInt(values["branch_id"]), Lines: lines, Net: out.net}
	case "stock-out":
		lines := stockLinesFromGroups(groups)
		out.total = stockLineTotal(lines)
		out.net = out.total
		out.posting = services.PostingRequest{Kind: services.DocumentStockOut, BranchID: parseInt(values["branch_id"]), Lines: lines, Net: out.net}
	case "stock-transactions":
		lines := stockLinesFromGroups(groups)
		out.total = stockLineTotal(lines)
		out.less = adjustmentTotal(groups, "discounts")
		out.add = adjustmentTotal(groups, "additionals")
		out.net = out.total - out.less + out.add
		out.posting = services.PostingRequest{Kind: services.DocumentStockTransfer, BranchID: parseInt(values["branch_id"]), Lines: lines, Net: out.net}
	case "ap-credit":
		paid := parseMoney(values["cash_amount"]) + paymentAmount(groups, "checks")
		out.net = paid
		out.posting = services.PostingRequest{Kind: services.DocumentAPCredit, PartyID: parseInt(values["party_id"]), Paid: paid}
	case "ar-credit", "rebates":
		paid := parseMoney(values["cash_amount"]) + paymentAmount(groups, "checks")
		out.net = paid
		out.posting = services.PostingRequest{Kind: services.DocumentARCredit, PartyID: parseInt(values["party_id"]), Paid: paid}
	case "ap-debit":
		amount := parseMoney(values["amount"])
		out.net = amount
		out.posting = services.PostingRequest{Kind: services.DocumentAPDebit, PartyID: parseInt(values["party_id"]), Amount: amount}
	case "ar-debit":
		amount := parseMoney(values["amount"])
		out.net = amount
		out.posting = services.PostingRequest{Kind: services.DocumentARDebit, PartyID: parseInt(values["party_id"]), Amount: amount}
	default:
		out.total = groupMoneyTotal(groups)
		out.net = out.total
	}
	return out
}

func scanRecords(rows pgx.Rows) ([]models.Record, error) {
	fields := rows.FieldDescriptions()
	var records []models.Record
	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return nil, err
		}
		record := models.Record{}
		for i, field := range fields {
			if values[i] == nil {
				record[field.Name] = ""
			} else {
				record[field.Name] = fmt.Sprint(values[i])
			}
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

func valueForField(field models.Field, value string) any {
	if field.Type == models.FieldBool {
		return parseBool(value)
	}
	if strings.TrimSpace(value) == "" {
		return nil
	}
	if field.Type == models.FieldMoney {
		return centsToNumeric(parseMoney(value))
	}
	return value
}

func emptyToNil(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func nullInt(value int64) any {
	if value == 0 {
		return nil
	}
	return value
}

func nullableDate(value string) *time.Time {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	t, err := time.Parse("2006-01-02", value)
	if err != nil {
		return nil
	}
	return &t
}

func parseBool(value string) bool {
	return value == "on" || value == "true" || value == "1"
}

func parseInt(value string) int64 {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	if n, err := strconv.ParseInt(value, 10, 64); err == nil {
		return n
	}
	if f, err := strconv.ParseFloat(value, 64); err == nil {
		return int64(f)
	}
	return 0
}

func parseMoney(value string) int64 {
	value = strings.ReplaceAll(strings.TrimSpace(value), ",", "")
	if value == "" {
		return 0
	}
	negative := strings.HasPrefix(value, "-")
	value = strings.TrimPrefix(value, "-")
	parts := strings.SplitN(value, ".", 3)
	whole, _ := strconv.ParseInt(parts[0], 10, 64)
	var cents int64
	if len(parts) > 1 {
		frac := parts[1]
		if len(frac) == 1 {
			frac += "0"
		}
		if len(frac) > 2 {
			frac = frac[:2]
		}
		cents, _ = strconv.ParseInt(frac, 10, 64)
	}
	total := whole*100 + cents
	if negative {
		return -total
	}
	return total
}

func centsToNumeric(cents int64) string {
	sign := ""
	if cents < 0 {
		sign = "-"
		cents = -cents
	}
	return fmt.Sprintf("%s%d.%02d", sign, cents/100, cents%100)
}

func stockLinesFromGroups(groups []LineInput) []services.StockLine {
	for _, group := range groups {
		if group.Group != "details" {
			continue
		}
		lines := make([]services.StockLine, 0, len(group.Rows))
		for _, row := range group.Rows {
			line := services.StockLine{
				StockID:  parseInt(row["stock_id"]),
				Qty:      parseInt(row["qty"]),
				UnitCost: parseMoney(row["unit_cost"]),
				Capital:  parseMoney(row["capital"]),
			}
			if line.StockID != 0 || line.Qty != 0 {
				lines = append(lines, line)
			}
		}
		return lines
	}
	return nil
}

func salesLinesFromGroups(groups []LineInput) []services.SalesLine {
	for _, group := range groups {
		if group.Group != "details" {
			continue
		}
		lines := make([]services.SalesLine, 0, len(group.Rows))
		for _, row := range group.Rows {
			line := services.SalesLine{
				StockID:       parseInt(row["stock_id"]),
				Qty:           parseInt(row["qty"]),
				UnitCost:      parseMoney(row["unit_cost"]),
				Capital:       parseMoney(row["capital"]),
				Discount:      parseMoney(row["discount"]),
				OtherDiscount: parseMoney(row["other_discount"]),
			}
			if line.StockID != 0 || line.Qty != 0 {
				lines = append(lines, line)
			}
		}
		return lines
	}
	return nil
}

func stockLinesFromSales(lines []services.SalesLine) []services.StockLine {
	out := make([]services.StockLine, 0, len(lines))
	for _, line := range lines {
		out = append(out, services.StockLine{StockID: line.StockID, Qty: line.Qty, UnitCost: line.UnitCost, Capital: line.Capital})
	}
	return out
}

func adjustmentsFromGroup(groups []LineInput, key string) []services.AdjustmentLine {
	for _, group := range groups {
		if group.Group != key {
			continue
		}
		lines := make([]services.AdjustmentLine, 0, len(group.Rows))
		for _, row := range group.Rows {
			line := services.AdjustmentLine{Particulars: row["particulars"], Qty: parseInt(row["qty"]), Price: parseMoney(row["price"])}
			if line.Particulars != "" || line.Qty != 0 || line.Price != 0 {
				lines = append(lines, line)
			}
		}
		return lines
	}
	return nil
}

func paymentsFromGroups(groups []LineInput, key string, cash int64) []services.Payment {
	payments := []services.Payment{}
	if cash != 0 {
		payments = append(payments, services.Payment{CashAmount: cash})
	}
	for _, group := range groups {
		if group.Group != key {
			continue
		}
		for _, row := range group.Rows {
			amount := parseMoney(row["amount"])
			if amount != 0 {
				payments = append(payments, services.Payment{CheckAmount: amount})
			}
		}
	}
	return payments
}

func paymentAmount(groups []LineInput, key string) int64 {
	var total int64
	for _, group := range groups {
		if group.Group != key {
			continue
		}
		for _, row := range group.Rows {
			total += parseMoney(row["amount"])
		}
	}
	return total
}

func adjustmentTotal(groups []LineInput, key string) int64 {
	return services.ComputePurchase(services.PurchaseDocument{Discounts: adjustmentsFromGroup(groups, key)}).Less
}

func stockLineTotal(lines []services.StockLine) int64 {
	var total int64
	for _, line := range lines {
		total += line.Qty * line.UnitCost
	}
	return total
}

func groupMoneyTotal(groups []LineInput) int64 {
	var total int64
	for _, group := range groups {
		for _, row := range group.Rows {
			if value := parseMoney(row["total"]); value != 0 {
				total += value
			} else {
				total += parseMoney(row["cash"]) + parseMoney(row["check"]) + parseMoney(row["amount"])
			}
		}
	}
	return total
}

func rowIsBlank(row map[string]string) bool {
	for _, value := range row {
		if strings.TrimSpace(value) != "" {
			return false
		}
	}
	return true
}
