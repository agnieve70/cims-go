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

type storedDocumentPayload struct {
	Kind      string            `json:"kind"`
	Values    map[string]string `json:"values"`
	LineInput []LineInput       `json:"line_input"`
	EncoderID int64             `json:"encoder_id"`
}

type drLineState struct {
	StockID      int64
	RemainingQty int64
}

type drValidationState struct {
	PartyID int64
	Lines   map[int64]drLineState
}

var manilaTime = time.FixedZone("Asia/Manila", 8*60*60)

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

func yearDateRange(year int) (time.Time, time.Time) {
	start := time.Date(year, time.January, 1, 0, 0, 0, 0, manilaTime)
	return start, start.AddDate(1, 0, 0)
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

func (s *PostgresStore) ListMaster(ctx context.Context, form models.FormDefinition, search string, year int, limit int, offset int) ([]models.Record, error) {
	if limit <= 0 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}
	cols := []string{"m.id::text as id"}
	for _, field := range form.Fields {
		if field.Type == models.FieldBool {
			cols = append(cols, fmt.Sprintf("case when coalesce(m.%s, false) then 'Yes' else 'No' end as %s", field.Column, field.Key))
			continue
		}
		cols = append(cols, fmt.Sprintf("coalesce(m.%s::text, '') as %s", field.Column, field.Key))
	}
	cols = append(cols, "coalesce(u.display_name, '') as encoder", "to_char(m.updated_at at time zone 'Asia/Manila', 'MM/DD/YYYY, HH12:MI AM') as last_update", "coalesce(uu.display_name, '') as updated_by")
	search = strings.TrimSpace(search)
	args := []any{}
	clauses := []string{}
	if year != 0 {
		start, end := yearDateRange(year)
		args = append(args, start, end)
		clauses = append(clauses, fmt.Sprintf("m.updated_at >= $%d and m.updated_at < $%d", len(args)-1, len(args)))
	}
	if search != "" {
		matchCols := []string{}
		for _, field := range form.Fields {
			matchCols = append(matchCols, fmt.Sprintf("coalesce(m.%s::text, '')", field.Column))
		}
		matchCols = append(matchCols, "coalesce(u.display_name, '')", "coalesce(uu.display_name, '')")
		args = append(args, "%"+search+"%")
		clauses = append(clauses, fmt.Sprintf("(%s) ilike $%d", strings.Join(matchCols, " || ' ' || "), len(args)))
	}
	where := ""
	if len(clauses) > 0 {
		where = "where " + strings.Join(clauses, " and ")
	}
	args = append(args, limit, offset)
	limitArg := len(args) - 1
	offsetArg := len(args)
	sql := fmt.Sprintf(`
		select %s
		from %s m
		left join users u on u.id = m.encoder_user_id
		left join users uu on uu.id = m.last_update_by_user_id
		%s
		order by m.id desc
		limit $%d offset $%d`, strings.Join(cols, ", "), form.Table, where, limitArg, offsetArg)
	rows, err := s.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRecords(rows)
}

func (s *PostgresStore) GetMaster(ctx context.Context, form models.FormDefinition, id int64) (models.Record, error) {
	cols := []string{"m.id::text as id"}
	for _, field := range form.Fields {
		if field.Type == models.FieldBool {
			cols = append(cols, fmt.Sprintf("case when coalesce(m.%s, false) then 'true' else 'false' end as %s", field.Column, field.Key))
			continue
		}
		cols = append(cols, fmt.Sprintf("coalesce(m.%s::text, '') as %s", field.Column, field.Key))
	}
	if form.Kind == "stocks" {
		cols = append(cols, "coalesce((select coalesce(round(sum(sl.qty_delta)), 0)::bigint::text from stock_ledger sl where sl.stock_id = m.id), '0') as soh")
	}
	cols = append(cols, "coalesce(u.display_name, '') as encoder", "to_char(m.updated_at at time zone 'Asia/Manila', 'YYYY-MM-DD HH12:MI:SS AM') as last_update", "coalesce(uu.display_name, '') as updated_by")
	sql := fmt.Sprintf(`
		select %s
		from %s m
		left join users u on u.id = m.encoder_user_id
		left join users uu on uu.id = m.last_update_by_user_id
		where m.id=$1`, strings.Join(cols, ", "), form.Table)
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
		args = append(args, valueForMasterField(form, field, values))
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

func (s *PostgresStore) DeleteMaster(ctx context.Context, form models.FormDefinition, id int64, user models.User) error {
	if !user.CanWrite() {
		return errors.New("write access required")
	}
	sql := fmt.Sprintf(`delete from %s where id=$1`, form.Table)
	tag, err := s.pool.Exec(ctx, sql, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (s *PostgresStore) ListDocuments(ctx context.Context, kind string, search string, year int) ([]models.DocumentListItem, error) {
	args := []any{kind}
	clauses := []string{"d.kind=$1"}
	if year != 0 {
		start, end := yearDateRange(year)
		args = append(args, start, end)
		clauses = append(clauses, fmt.Sprintf("d.entry_date >= $%d and d.entry_date < $%d", len(args)-1, len(args)))
	}
	search = strings.TrimSpace(search)
	if search != "" {
		args = append(args, "%"+search+"%")
		clauses = append(clauses, fmt.Sprintf(`(
			concat_ws(' ',
				coalesce(d.entry_id, ''),
				coalesce(to_char(d.entry_date, 'YYYY-MM-DD'), ''),
				coalesce(nullif(s.company, ''), nullif(c.company, ''), s.code, c.code, ''),
				coalesce(b.name, ''),
				coalesce(d.reference, ''),
				coalesce(dr.entry_id, ''),
				coalesce(u.display_name, ''),
				coalesce(d.payload::text, '')
			) ilike $%d
		)`, len(args)))
	}
	query := fmt.Sprintf(`
		with listed as (
			select d.id, d.entry_id, d.entry_date, d.kind,
			       coalesce(nullif(s.company, ''), nullif(c.company, ''), s.code, c.code, '') as party,
			       coalesce(b.name, '') as branch,
			       coalesce(nullif(d.reference, ''), nullif(d.payload->'values'->>'transfer_id', ''), '') as reference,
			       coalesce(dr.entry_id, '') as dr_ref,
			       d.total,
			       d.less_amount,
			       d.add_amount,
			       d.net,
			       coalesce(u.display_name, '') as encoder,
			       coalesce(d.payload->'values'->>'transaction', '') as transaction,
			       coalesce(nullif(bl.name, ''), nullif(bl.code, ''), '') as transactee,
			       coalesce(d.remarks, '') as remarks,
			       to_char(d.updated_at at time zone 'Asia/Manila', 'YYYY-MM-DD HH12:MI AM') as last_update,
			       coalesce(uu.display_name, '') as updated_by
			from documents d
			left join branches b on b.id = d.branch_id
			left join branches bl on bl.id = nullif(d.payload->'values'->>'branch_location', '')::bigint
			left join users u on u.id = d.encoder_user_id
			left join users uu on uu.id = d.last_update_by_user_id
			left join suppliers s on d.party_type='supplier' and s.id=d.party_id
			left join customers c on d.party_type='customer' and c.id=d.party_id
			left join documents dr on dr.id = d.dr_reference_id
			where %s
			order by d.id desc
			limit 200
		),
		dr_qty as (
			select dl.document_id, sum(dl.qty) as qty
			from document_lines dl
			join listed l on l.kind = 'dr' and l.id = dl.document_id
			where dl.group_key = 'details'
			group by dl.document_id
		),
		dr_used as (
			select dc.dr_document_id, sum(dc.consumed_qty) as consumed_qty
			from dr_consumptions dc
			join listed l on l.kind = 'dr' and l.id = dc.dr_document_id
			group by dc.dr_document_id
		),
		document_totals as (
			select l.id,
			       coalesce(sum(dl.qty), 0) as total_qty
			from listed l
			left join document_lines dl on dl.document_id = l.id and dl.group_key = 'details'
			where l.kind in ('purchases', 'sales')
			group by l.id
		)
		select l.id, l.entry_id, l.entry_date,
		       l.party,
		       l.branch,
		       l.reference,
		       l.dr_ref,
		       case
		         when l.kind <> 'dr' then ''
		         when coalesce(q.qty, 0) = 0 then ''
		         when coalesce(u.consumed_qty, 0) = 0 then 'Open'
		         when coalesce(u.consumed_qty, 0) >= coalesce(q.qty, 0) then 'Fully Used'
		         else 'Partial'
		       end,
		       coalesce(l.net::text, '0'), l.encoder,
		       l.transaction, l.transactee, l.remarks, l.last_update, l.updated_by,
		       coalesce(trim(to_char(pt.total_qty, 'FM999G999G999G990D00')), ''),
		       coalesce(trim(to_char(l.total, 'FM999G999G999G990D00')), ''),
		       coalesce(trim(to_char(l.less_amount, 'FM999G999G999G990D00')), ''),
		       coalesce(trim(to_char(l.add_amount, 'FM999G999G999G990D00')), ''),
		       coalesce(trim(to_char(l.net, 'FM999G999G999G990D00')), '')
		from listed l
		left join dr_qty q on q.document_id = l.id
		left join dr_used u on u.dr_document_id = l.id
		left join document_totals pt on pt.id = l.id
		order by l.id desc`, strings.Join(clauses, " and "))
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []models.DocumentListItem
	for rows.Next() {
		var item models.DocumentListItem
		if err := rows.Scan(&item.ID, &item.EntryID, &item.EntryDate, &item.Party, &item.Branch, &item.Reference, &item.DRRef, &item.Status, &item.Net, &item.Encoder, &item.Transaction, &item.Transactee, &item.Remarks, &item.LastUpdate, &item.UpdatedBy, &item.TotalQty, &item.GrossTotal, &item.TotalLess, &item.TotalAdd, &item.NetTotal); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) PurchaseReportRows(ctx context.Context, from, to time.Time) ([]models.PurchaseReportRow, error) {
	rows, err := s.pool.Query(ctx, `
		select coalesce(nullif(s.company, ''), nullif(s.code, ''), 'No Supplier') as supplier,
		       coalesce(d.entry_id, '') as entry_id,
		       to_char(d.entry_date, 'MM/DD/YYYY') as entry_date,
		       coalesce(d.payload->'values'->>'or_ci_number', d.reference, '') as or_ci_number,
		       case when coalesce(d.cash, false) then 'Cash' else 'Charge' end as payment_type,
		       coalesce(round(coalesce(d.total, d.net, 0) * 100), 0)::bigint as gross_cents,
		       coalesce(round(coalesce(d.net, d.total, 0) * 100), 0)::bigint as net_cents
		from documents d
		left join suppliers s on d.party_type = 'supplier' and s.id = d.party_id
		where d.kind = 'purchases'
		  and d.entry_date >= $1::date
		  and d.entry_date <= $2::date
		order by supplier, d.entry_date, d.entry_id`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var reportRows []models.PurchaseReportRow
	for rows.Next() {
		var row models.PurchaseReportRow
		if err := rows.Scan(&row.Supplier, &row.EntryID, &row.EntryDate, &row.ORCINumber, &row.Type, &row.GrossCents, &row.NetCents); err != nil {
			return nil, err
		}
		reportRows = append(reportRows, row)
	}
	return reportRows, rows.Err()
}

func (s *PostgresStore) PurchaseByDRNumberReportRows(ctx context.Context, from, to time.Time) ([]models.PurchaseByDRNumberReportRow, error) {
	rows, err := s.pool.Query(ctx, `
		select coalesce(nullif(d.payload->'values'->>'or_ci_number', ''), nullif(d.reference, ''), nullif(d.entry_id, ''), 'No Reference') as reference,
		       to_char(d.document_date, 'MM/DD/YYYY') as purchase_date,
		       case when coalesce(d.cash, false) then 'Cash' else 'Charge' end as payment_type,
		       coalesce(nullif(s.company, ''), nullif(s.code, ''), 'No Supplier') as supplier,
		       coalesce(st.code, '') as stock_code,
		       coalesce(nullif(st.name, ''), nullif(st.code, ''), 'No Stock') as stock_name,
		       coalesce(round(dl.qty), 0)::bigint as quantity,
		       coalesce(round(dl.unit_cost * 100), 0)::bigint as unit_cost_cents,
		       coalesce(round((case when coalesce(dl.amount, 0) <> 0 then dl.amount else dl.qty * dl.unit_cost end) * 100), 0)::bigint as amount_cents
		from documents d
		join document_lines dl on dl.document_id = d.id and dl.group_key = 'details'
		left join stocks st on st.id = dl.stock_id
		left join suppliers s on d.party_type = 'supplier' and s.id = d.party_id
		where d.kind = 'purchases'
		  and d.document_date >= $1::date
		  and d.document_date <= $2::date
		order by reference, purchase_date, supplier, stock_code, stock_name`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var reportRows []models.PurchaseByDRNumberReportRow
	for rows.Next() {
		var row models.PurchaseByDRNumberReportRow
		if err := rows.Scan(&row.Reference, &row.PurchaseDate, &row.Type, &row.Supplier, &row.StockCode, &row.StockName, &row.Quantity, &row.UnitCostCents, &row.AmountCents); err != nil {
			return nil, err
		}
		reportRows = append(reportRows, row)
	}
	return reportRows, rows.Err()
}

func (s *PostgresStore) PurchaseByStockCodeReportRows(ctx context.Context, from, to time.Time) ([]models.PurchaseByStockCodeReportRow, error) {
	rows, err := s.pool.Query(ctx, `
		select coalesce(nullif(d.payload->'values'->>'or_ci_number', ''), nullif(d.reference, ''), nullif(d.entry_id, ''), 'No Reference') as reference,
		       to_char(d.document_date, 'MM/DD/YYYY') as purchase_date,
		       case when coalesce(d.cash, false) then 'Cash' else 'Charge' end as payment_type,
		       coalesce(nullif(s.company, ''), nullif(s.code, ''), 'No Supplier') as supplier,
		       coalesce(st.code, '') as stock_code,
		       coalesce(nullif(st.name, ''), nullif(st.code, ''), 'No Stock') as stock_name,
		       coalesce(round(dl.qty), 0)::bigint as quantity,
		       coalesce(round(dl.unit_cost * 100), 0)::bigint as unit_cost_cents,
		       coalesce(round((case when coalesce(dl.amount, 0) <> 0 then dl.amount else dl.qty * dl.unit_cost end) * 100), 0)::bigint as amount_cents
		from documents d
		join document_lines dl on dl.document_id = d.id and dl.group_key = 'details'
		left join stocks st on st.id = dl.stock_id
		left join suppliers s on d.party_type = 'supplier' and s.id = d.party_id
		where d.kind = 'purchases'
		  and d.document_date >= $1::date
		  and d.document_date <= $2::date
		order by stock_code, stock_name, purchase_date, reference, supplier`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var reportRows []models.PurchaseByStockCodeReportRow
	for rows.Next() {
		var row models.PurchaseByStockCodeReportRow
		if err := rows.Scan(&row.Reference, &row.PurchaseDate, &row.Type, &row.Supplier, &row.StockCode, &row.StockName, &row.Quantity, &row.UnitCostCents, &row.AmountCents); err != nil {
			return nil, err
		}
		reportRows = append(reportRows, row)
	}
	return reportRows, rows.Err()
}

func (s *PostgresStore) PurchaseBySupplierReportRows(ctx context.Context, from, to time.Time) ([]models.PurchaseBySupplierReportRow, error) {
	rows, err := s.pool.Query(ctx, `
		select coalesce(nullif(d.payload->'values'->>'or_ci_number', ''), nullif(d.reference, ''), nullif(d.entry_id, ''), 'No Reference') as reference,
		       to_char(d.document_date, 'MM/DD/YYYY') as purchase_date,
		       case when coalesce(d.cash, false) then 'Cash' else 'Charge' end as payment_type,
		       coalesce(nullif(s.company, ''), nullif(s.code, ''), 'No Supplier') as supplier,
		       coalesce(st.code, '') as stock_code,
		       coalesce(nullif(st.name, ''), nullif(st.code, ''), 'No Stock') as stock_name,
		       coalesce(round(dl.qty), 0)::bigint as quantity,
		       coalesce(round(dl.unit_cost * 100), 0)::bigint as unit_cost_cents,
		       coalesce(round((case when coalesce(dl.amount, 0) <> 0 then dl.amount else dl.qty * dl.unit_cost end) * 100), 0)::bigint as amount_cents
		from documents d
		join document_lines dl on dl.document_id = d.id and dl.group_key = 'details'
		left join stocks st on st.id = dl.stock_id
		left join suppliers s on d.party_type = 'supplier' and s.id = d.party_id
		where d.kind = 'purchases'
		  and d.document_date >= $1::date
		  and d.document_date <= $2::date
		order by supplier, stock_code, stock_name, purchase_date, reference`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var reportRows []models.PurchaseBySupplierReportRow
	for rows.Next() {
		var row models.PurchaseBySupplierReportRow
		if err := rows.Scan(&row.Reference, &row.PurchaseDate, &row.Type, &row.Supplier, &row.StockCode, &row.StockName, &row.Quantity, &row.UnitCostCents, &row.AmountCents); err != nil {
			return nil, err
		}
		reportRows = append(reportRows, row)
	}
	return reportRows, rows.Err()
}

func (s *PostgresStore) SalesReportRows(ctx context.Context, from, to time.Time) ([]models.SalesReportRow, error) {
	rows, err := s.pool.Query(ctx, `
		select coalesce(nullif(c.company, ''), nullif(c.code, ''), 'No Customer') as customer,
		       coalesce(d.entry_id, '') as entry_id,
		       to_char(d.document_date, 'MM/DD/YYYY') as entry_date,
		       coalesce(d.payload->'values'->>'or_ci_number', d.reference, '') as or_ci_number,
		       case when coalesce(d.cash, false) then 'Cash' else 'Charge' end as payment_type,
		       coalesce(round(coalesce(d.total, d.net, 0) * 100), 0)::bigint as gross_cents,
		       coalesce(round(coalesce(d.net, d.total, 0) * 100), 0)::bigint as net_cents
		from documents d
		left join customers c on d.party_type = 'customer' and c.id = d.party_id
		where d.kind = 'sales'
		  and d.document_date >= $1::date
		  and d.document_date <= $2::date
		order by customer, d.document_date, d.entry_id`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var reportRows []models.SalesReportRow
	for rows.Next() {
		var row models.SalesReportRow
		if err := rows.Scan(&row.Customer, &row.EntryID, &row.EntryDate, &row.ORCINumber, &row.Type, &row.GrossCents, &row.NetCents); err != nil {
			return nil, err
		}
		reportRows = append(reportRows, row)
	}
	return reportRows, rows.Err()
}

func (s *PostgresStore) SalesByORCIDRNumberReportRows(ctx context.Context, from, to time.Time) ([]models.SalesByORCIDRNumberReportRow, error) {
	rows, err := s.pool.Query(ctx, `
		select coalesce(nullif(d.payload->'values'->>'or_ci_number', ''), nullif(d.reference, ''), nullif(d.entry_id, ''), 'No Reference') as reference,
		       to_char(d.document_date, 'MM/DD/YYYY') as sales_date,
		       case when coalesce(d.cash, false) then 'Cash' else 'Charge' end as payment_type,
		       coalesce(nullif(c.company, ''), nullif(c.code, ''), 'No Customer') as customer,
		       coalesce(st.code, '') as stock_code,
		       coalesce(nullif(st.name, ''), nullif(st.code, ''), 'No Stock') as stock_name,
		       coalesce(round(dl.qty), 0)::bigint as quantity,
		       coalesce(round((case
		         when coalesce(dl.price, 0) <> 0 then dl.price
		         when coalesce(dl.qty, 0) <> 0 and coalesce(dl.amount, 0) <> 0 then dl.amount / dl.qty
		         else 0
		       end) * 100), 0)::bigint as price_cents,
		       coalesce(round((case
		         when coalesce(dl.amount, 0) <> 0 then dl.amount
		         else coalesce(dl.qty, 0) * coalesce(dl.price, 0)
		       end) * 100), 0)::bigint as amount_cents
		from documents d
		join document_lines dl on dl.document_id = d.id and dl.group_key = 'details'
		left join stocks st on st.id = dl.stock_id
		left join customers c on d.party_type = 'customer' and c.id = d.party_id
		where d.kind = 'sales'
		  and d.document_date >= $1::date
		  and d.document_date <= $2::date
		  and coalesce(dl.qty, 0) <> 0
		order by reference, sales_date, customer, stock_code, stock_name`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var reportRows []models.SalesByORCIDRNumberReportRow
	for rows.Next() {
		var row models.SalesByORCIDRNumberReportRow
		if err := rows.Scan(&row.Reference, &row.SalesDate, &row.Type, &row.Customer, &row.StockCode, &row.StockName, &row.Quantity, &row.PriceCents, &row.AmountCents); err != nil {
			return nil, err
		}
		reportRows = append(reportRows, row)
	}
	return reportRows, rows.Err()
}

func (s *PostgresStore) SalesMarkupByTransactionReportRows(ctx context.Context, from, to time.Time) ([]models.SalesMarkupByTransactionReportRow, error) {
	rows, err := s.pool.Query(ctx, `
		with sales_lines as (
			select to_char(d.document_date, 'MM/DD/YYYY') as sales_date,
			       coalesce(nullif(d.entry_id, ''), d.id::text) as entry_id,
			       case when coalesce(d.cash, false) then 'Cash' else 'Charge' end as sales_type,
			       coalesce(nullif(d.payload->'values'->>'or_ci_number', ''), nullif(d.reference, ''), nullif(d.entry_id, ''), 'No Receipt') as receipt_no,
				       coalesce(nullif(st.category_group, ''), 'Uncategorized') as item_group,
				       case
				         when coalesce(dl.amount, 0) <> 0 then dl.amount
				         when nullif(regexp_replace(coalesce(dl.payload->>'amount', ''), '[,\s]', '', 'g'), '') ~ '^-?\d+(\.\d+)?$'
				           then regexp_replace(dl.payload->>'amount', '[,\s]', '', 'g')::numeric
				         else coalesce(dl.qty, 0) * coalesce(dl.price, 0)
				       end as amount,
				       case
				         when nullif(regexp_replace(coalesce(dl.payload->>'capital', ''), '[,\s]', '', 'g'), '') ~ '^-?\d+(\.\d+)?$'
				           then regexp_replace(dl.payload->>'capital', '[,\s]', '', 'g')::numeric
			         else coalesce(dl.qty, 0) * coalesce(dl.unit_cost, 0)
			       end as capital,
			       case
			         when nullif(regexp_replace(coalesce(dl.payload->>'markup', ''), '[,\s]', '', 'g'), '') ~ '^-?\d+(\.\d+)?$'
			           then regexp_replace(dl.payload->>'markup', '[,\s]', '', 'g')::numeric
			         else (
			           case
			             when coalesce(dl.amount, 0) <> 0 then dl.amount
			             when nullif(regexp_replace(coalesce(dl.payload->>'amount', ''), '[,\s]', '', 'g'), '') ~ '^-?\d+(\.\d+)?$'
			               then regexp_replace(dl.payload->>'amount', '[,\s]', '', 'g')::numeric
			             else coalesce(dl.qty, 0) * coalesce(dl.price, 0)
			           end
			         ) - (
			           case
			             when nullif(regexp_replace(coalesce(dl.payload->>'capital', ''), '[,\s]', '', 'g'), '') ~ '^-?\d+(\.\d+)?$'
			               then regexp_replace(dl.payload->>'capital', '[,\s]', '', 'g')::numeric
			             else coalesce(dl.qty, 0) * coalesce(dl.unit_cost, 0)
			           end
			         )
			       end as markup
			from documents d
			join document_lines dl on dl.document_id = d.id and dl.group_key = 'details'
			left join stocks st on st.id = dl.stock_id
			where d.kind = 'sales'
			  and d.document_date >= $1::date
			  and d.document_date <= $2::date
			  and coalesce(dl.qty, 0) <> 0
		)
		select sales_date,
		       entry_id,
		       sales_type,
		       receipt_no,
			       item_group,
			       coalesce(round(markup * 100), 0)::bigint as markup_cents,
			       coalesce(round(capital * 100), 0)::bigint as capital_cents,
			       coalesce(round(amount * 100), 0)::bigint as amount_cents
		from sales_lines
		where coalesce(markup, 0) <> 0 or coalesce(capital, 0) <> 0
		order by sales_date, entry_id, sales_type, receipt_no, item_group`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var reportRows []models.SalesMarkupByTransactionReportRow
	for rows.Next() {
		var row models.SalesMarkupByTransactionReportRow
		if err := rows.Scan(&row.SalesDate, &row.EntryID, &row.SalesType, &row.ReceiptNo, &row.ItemGroup, &row.MarkupCents, &row.CapitalCents, &row.AmountCents); err != nil {
			return nil, err
		}
		reportRows = append(reportRows, row)
	}
	return reportRows, rows.Err()
}

func (s *PostgresStore) SalesByCustomerReportRows(ctx context.Context, from, to time.Time) ([]models.SalesByCustomerReportRow, error) {
	rows, err := s.pool.Query(ctx, `
		select coalesce(nullif(st.category_group, ''), 'Uncategorized') as category,
		       coalesce(nullif(c.company, ''), nullif(c.code, ''), 'No Customer') as customer,
		       coalesce(nullif(d.payload->'values'->>'or_ci_number', ''), nullif(d.reference, ''), nullif(d.entry_id, ''), 'No Reference') as reference,
		       to_char(d.document_date, 'MM/DD/YYYY') as sales_date,
		       case when coalesce(d.cash, false) then 'Cash' else 'Charge' end as payment_type,
		       coalesce(st.code, '') as stock_code,
		       coalesce(nullif(st.name, ''), nullif(st.code, ''), 'No Stock') as stock_name,
		       coalesce(round(dl.qty), 0)::bigint as quantity,
		       coalesce(round((case
		         when coalesce(dl.price, 0) <> 0 then dl.price
		         when coalesce(dl.qty, 0) <> 0 and coalesce(dl.amount, 0) <> 0 then dl.amount / dl.qty
		         else 0
		       end) * 100), 0)::bigint as price_cents,
		       coalesce(round((case
		         when coalesce(dl.amount, 0) <> 0 then dl.amount
		         else coalesce(dl.qty, 0) * coalesce(dl.price, 0)
		       end) * 100), 0)::bigint as amount_cents
		from documents d
		join document_lines dl on dl.document_id = d.id and dl.group_key = 'details'
		left join stocks st on st.id = dl.stock_id
		left join customers c on d.party_type = 'customer' and c.id = d.party_id
		where d.kind = 'sales'
		  and d.document_date >= $1::date
		  and d.document_date <= $2::date
		  and coalesce(dl.qty, 0) <> 0
		order by category, customer, sales_date, reference, stock_code, stock_name`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var reportRows []models.SalesByCustomerReportRow
	for rows.Next() {
		var row models.SalesByCustomerReportRow
		if err := rows.Scan(&row.Category, &row.Customer, &row.Reference, &row.SalesDate, &row.Type, &row.StockCode, &row.StockName, &row.Quantity, &row.PriceCents, &row.AmountCents); err != nil {
			return nil, err
		}
		reportRows = append(reportRows, row)
	}
	return reportRows, rows.Err()
}

func (s *PostgresStore) SalesByStockNameReportRows(ctx context.Context, from, to time.Time) ([]models.SalesByStockNameReportRow, error) {
	rows, err := s.pool.Query(ctx, `
		select coalesce(nullif(st.category_group, ''), 'Uncategorized') as category,
		       coalesce(nullif(c.company, ''), nullif(c.code, ''), 'No Customer') as customer,
		       coalesce(nullif(d.payload->'values'->>'or_ci_number', ''), nullif(d.reference, ''), nullif(d.entry_id, ''), 'No Reference') as reference,
		       to_char(d.document_date, 'MM/DD/YYYY') as sales_date,
		       case when coalesce(d.cash, false) then 'Cash' else 'Charge' end as payment_type,
		       coalesce(st.code, '') as stock_code,
		       coalesce(nullif(st.name, ''), nullif(st.code, ''), 'No Stock') as stock_name,
		       coalesce(round(dl.qty), 0)::bigint as quantity,
		       coalesce(round((case
		         when coalesce(dl.price, 0) <> 0 then dl.price
		         when coalesce(dl.qty, 0) <> 0 and coalesce(dl.amount, 0) <> 0 then dl.amount / dl.qty
		         else 0
		       end) * 100), 0)::bigint as price_cents,
		       coalesce(round((case
		         when coalesce(dl.amount, 0) <> 0 then dl.amount
		         else coalesce(dl.qty, 0) * coalesce(dl.price, 0)
		       end) * 100), 0)::bigint as amount_cents
		from documents d
		join document_lines dl on dl.document_id = d.id and dl.group_key = 'details'
		left join stocks st on st.id = dl.stock_id
		left join customers c on d.party_type = 'customer' and c.id = d.party_id
		where d.kind = 'sales'
		  and d.document_date >= $1::date
		  and d.document_date <= $2::date
		  and coalesce(dl.qty, 0) <> 0
		order by category, stock_name, stock_code, sales_date, reference, customer`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var reportRows []models.SalesByStockNameReportRow
	for rows.Next() {
		var row models.SalesByStockNameReportRow
		if err := rows.Scan(&row.Category, &row.Customer, &row.Reference, &row.SalesDate, &row.Type, &row.StockCode, &row.StockName, &row.Quantity, &row.PriceCents, &row.AmountCents); err != nil {
			return nil, err
		}
		reportRows = append(reportRows, row)
	}
	return reportRows, rows.Err()
}

func (s *PostgresStore) APLedgerReportRows(ctx context.Context, _ time.Time, to time.Time) ([]models.APLedgerReportRow, error) {
	rows, err := s.pool.Query(ctx, `
		select s.id::text as supplier_id,
		       coalesce(s.code, '') as supplier_code,
		       coalesce(nullif(s.company, ''), nullif(s.code, ''), 'No Supplier') as supplier_name,
		       coalesce(nullif(trim(concat_ws(' ', s.lastname, s.firstname, s.middlename)), ''), 'NA') as representative,
		       coalesce(d.entry_id, '') as entry_id,
		       to_char(d.entry_date, 'MM/DD/YYYY') as entry_date,
		       coalesce(nullif(d.reference, ''), nullif(d.payload->'values'->>'or_ci_number', ''), d.entry_id, '') as reference,
		       coalesce(d.kind, '') as kind,
		       coalesce(round(bl.amount_delta * 100), 0)::bigint as delta_cents
		from balance_ledger bl
		join documents d on d.id = bl.document_id
		join suppliers s on bl.party_type = 'supplier' and s.id = bl.party_id
		where bl.party_type = 'supplier'
		  and d.entry_date <= $1::date
		  and d.kind in ('purchases', 'ap-credit', 'ap-debit')
		order by supplier_name, d.entry_date, d.entry_id`, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var reportRows []models.APLedgerReportRow
	for rows.Next() {
		var row models.APLedgerReportRow
		if err := rows.Scan(&row.SupplierID, &row.SupplierCode, &row.SupplierName, &row.Representative, &row.EntryID, &row.EntryDate, &row.Reference, &row.Kind, &row.DeltaCents); err != nil {
			return nil, err
		}
		reportRows = append(reportRows, row)
	}
	return reportRows, rows.Err()
}

func (s *PostgresStore) ARLedgerReportRows(ctx context.Context, _ time.Time, to time.Time) ([]models.ARLedgerReportRow, error) {
	rows, err := s.pool.Query(ctx, `
		select c.id::text as customer_id,
		       coalesce(c.code, '') as customer_code,
		       coalesce(nullif(c.company, ''), nullif(c.code, ''), 'No Customer') as customer_name,
		       coalesce(nullif(c.credit_term, ''), '') as credit_term,
		       coalesce(round(coalesce(c.credit_limit, 0) * 100), 0)::bigint as credit_limit_cents,
		       coalesce(d.entry_id, '') as entry_id,
		       to_char(d.entry_date, 'MM/DD/YYYY') as entry_date,
		       coalesce(nullif(d.reference, ''), nullif(d.payload->'values'->>'or_ci_number', ''), d.entry_id, '') as reference,
		       coalesce(d.kind, '') as kind,
		       coalesce(round(bl.amount_delta * 100), 0)::bigint as delta_cents
		from balance_ledger bl
		join documents d on d.id = bl.document_id
		join customers c on bl.party_type = 'customer' and c.id = bl.party_id
		where bl.party_type = 'customer'
		  and d.entry_date <= $1::date
		  and d.kind in ('sales', 'ar-credit', 'ar-debit', 'rebates')
		order by customer_name, d.entry_date, d.entry_id`, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var reportRows []models.ARLedgerReportRow
	for rows.Next() {
		var row models.ARLedgerReportRow
		if err := rows.Scan(&row.CustomerID, &row.CustomerCode, &row.CustomerName, &row.CreditTerm, &row.CreditLimit, &row.EntryID, &row.EntryDate, &row.Reference, &row.Kind, &row.DeltaCents); err != nil {
			return nil, err
		}
		reportRows = append(reportRows, row)
	}
	return reportRows, rows.Err()
}

func (s *PostgresStore) IncomingCheckReportRows(ctx context.Context, _ time.Time) ([]models.IncomingCheckReportRow, error) {
	rows, err := s.pool.Query(ctx, `
		with check_lines as (
			select d.kind,
			       d.entry_id,
			       d.reference as document_reference,
			       coalesce(nullif(c.company, ''), nullif(c.code, ''), nullif(d.payload->'values'->>'payee', ''), 'No Payee') as payee,
			       coalesce(dl.payload->>'number', '') as check_number,
			       coalesce(dl.payload->>'bank_name', '') as bank_name,
			       case
			       	when coalesce(dl.payload->>'date', '') ~ '^\d{4}-\d{2}-\d{2}$' then (dl.payload->>'date')::date
			       	else d.entry_date::date
			       end as check_date,
			       case
			       	when coalesce(dl.payload->>'amount', '') ~ '^-?\d+(\.\d+)?$' then (dl.payload->>'amount')::numeric
			       	else coalesce(dl.amount, dl.check_amount, 0)
			       end as amount
			from document_lines dl
			join documents d on d.id = dl.document_id
			left join customers c on d.party_type = 'customer' and c.id = d.party_id
			where dl.group_key in ('checks', 'payments')
			  and d.kind in ('ar-credit', 'rebates', 'sales', 'checks-in')
		)
		select payee,
		       case kind
		       	when 'ar-credit' then 'AR Credit'
		       	when 'rebates' then 'Rebates'
		       	when 'sales' then coalesce(nullif(document_reference, ''), nullif(entry_id, ''), 'Sales')
		       	when 'checks-in' then 'Checks In'
		       	else coalesce(nullif(document_reference, ''), nullif(entry_id, ''), '')
		       end as reference,
		       to_char(check_date, 'MM/DD/YYYY') as check_date,
		       check_number,
		       bank_name,
		       coalesce(round(amount * 100), 0)::bigint as amount_cents
		from check_lines
		where coalesce(amount, 0) <> 0
		order by payee, check_date, check_number`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var reportRows []models.IncomingCheckReportRow
	for rows.Next() {
		var row models.IncomingCheckReportRow
		if err := rows.Scan(&row.Payee, &row.Reference, &row.CheckDate, &row.Number, &row.BankName, &row.AmountCents); err != nil {
			return nil, err
		}
		reportRows = append(reportRows, row)
	}
	return reportRows, rows.Err()
}

func (s *PostgresStore) OutgoingCheckReportRows(ctx context.Context, _ time.Time) ([]models.OutgoingCheckReportRow, error) {
	rows, err := s.pool.Query(ctx, `
		with check_lines as (
			select d.kind,
			       d.entry_id,
			       d.reference as document_reference,
			       coalesce(nullif(s.company, ''), nullif(s.code, ''), nullif(d.payload->'values'->>'payee', ''), 'No Payee') as payee,
			       coalesce(dl.payload->>'number', '') as check_number,
			       coalesce(dl.payload->>'bank_name', '') as bank_name,
			       case
			       	when coalesce(dl.payload->>'date', '') ~ '^\d{4}-\d{2}-\d{2}$' then (dl.payload->>'date')::date
			       	else d.entry_date::date
			       end as check_date,
			       case
			       	when coalesce(dl.payload->>'amount', '') ~ '^-?\d+(\.\d+)?$' then (dl.payload->>'amount')::numeric
			       	else coalesce(dl.amount, dl.check_amount, 0)
			       end as amount
			from document_lines dl
			join documents d on d.id = dl.document_id
			left join suppliers s on d.party_type = 'supplier' and s.id = d.party_id
			where dl.group_key in ('checks', 'payments')
			  and d.kind in ('ap-credit', 'purchases')
		)
		select payee,
		       case kind
		       	when 'ap-credit' then 'AP Credit'
		       	when 'purchases' then coalesce(nullif(document_reference, ''), nullif(entry_id, ''), 'Purchases')
		       	else coalesce(nullif(document_reference, ''), nullif(entry_id, ''), '')
		       end as reference,
		       to_char(check_date, 'MM/DD/YYYY') as check_date,
		       check_number,
		       bank_name,
		       coalesce(round(amount * 100), 0)::bigint as amount_cents
		from check_lines
		where coalesce(amount, 0) <> 0
		order by payee, check_date, check_number`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var reportRows []models.OutgoingCheckReportRow
	for rows.Next() {
		var row models.OutgoingCheckReportRow
		if err := rows.Scan(&row.Payee, &row.Reference, &row.CheckDate, &row.Number, &row.BankName, &row.AmountCents); err != nil {
			return nil, err
		}
		reportRows = append(reportRows, row)
	}
	return reportRows, rows.Err()
}

func (s *PostgresStore) ExpenseReportRows(ctx context.Context, from, to time.Time) ([]models.ExpenseReportRow, error) {
	rows, err := s.pool.Query(ctx, `
		with expense_lines as (
			select coalesce(ec.id::text, '') as category_id,
			       coalesce(ec.code, '') as category_code,
			       coalesce(nullif(ec.name, ''), 'Uncategorized') as category_name,
			       d.entry_date::date as entry_date,
			       coalesce(dl.cash_amount, 0) as cash_amount,
			       coalesce(dl.check_amount, 0) as check_amount,
			       case
			       	when coalesce(dl.payload->>'total', '') ~ '^-?\d+(\.\d+)?$' then (dl.payload->>'total')::numeric
			       	when coalesce(dl.amount, 0) <> 0 then dl.amount
			       	else coalesce(dl.cash_amount, 0) + coalesce(dl.check_amount, 0)
			       end as total_amount
			from documents d
			join document_lines dl on dl.document_id = d.id and dl.group_key = 'details'
			left join expense_charts ec on ec.id = dl.code_id
			where d.kind = 'expenses'
			  and d.entry_date >= $1::date
			  and d.entry_date <= $2::date
		)
		select category_id,
		       category_code,
		       category_name,
		       to_char(entry_date, 'MM/DD/YYYY') as entry_date,
		       coalesce(round(sum(cash_amount) * 100), 0)::bigint as cash_cents,
		       coalesce(round(sum(check_amount) * 100), 0)::bigint as check_cents,
		       coalesce(round(sum(total_amount) * 100), 0)::bigint as total_cents
		from expense_lines
		group by category_id, category_code, category_name, entry_date
		having coalesce(sum(total_amount), 0) <> 0
		    or coalesce(sum(cash_amount), 0) <> 0
		    or coalesce(sum(check_amount), 0) <> 0
		order by entry_date, category_code, category_name`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var reportRows []models.ExpenseReportRow
	for rows.Next() {
		var row models.ExpenseReportRow
		if err := rows.Scan(&row.CategoryID, &row.CategoryCode, &row.CategoryName, &row.EntryDate, &row.CashCents, &row.CheckCents, &row.TotalCents); err != nil {
			return nil, err
		}
		reportRows = append(reportRows, row)
	}
	return reportRows, rows.Err()
}

func (s *PostgresStore) IncomeStatementRows(ctx context.Context, from, to time.Time) ([]models.IncomeStatementRow, error) {
	rows, err := s.pool.Query(ctx, `
		with sales_rows as (
			select case when d.cash then 'cash_sales' else 'charge_sales' end as section,
			       case when d.cash then 'Cash Sales' else 'Charge Sales' end as label,
			       coalesce(d.net, d.total, 0) as amount
			from documents d
			where d.kind = 'sales'
			  and d.document_date >= $1::date
			  and d.document_date <= $2::date
		),
		inventory_rows as (
			select 'beginning_inventory' as section,
			       'Stock Inventory, Beginning' as label,
			       coalesce(sum(sl.qty_delta * sl.unit_cost), 0) as amount
			from stock_ledger sl
			join documents d on d.id = sl.document_id
			where d.entry_date < $1::date
			union all
			select 'ending_inventory' as section,
			       'Stock Inventory, End' as label,
			       coalesce(sum(sl.qty_delta * sl.unit_cost), 0) as amount
			from stock_ledger sl
			join documents d on d.id = sl.document_id
			where d.entry_date <= $2::date
		),
		purchase_rows as (
			select 'purchases' as section,
			       coalesce(nullif(s.company, ''), nullif(s.code, ''), 'No Supplier') as label,
			       coalesce(sum(dl.qty * dl.unit_cost), 0) as amount
			from documents d
			join document_lines dl on dl.document_id = d.id and dl.group_key = 'details'
			left join suppliers s on d.party_type = 'supplier' and s.id = d.party_id
			where d.kind = 'purchases'
			  and d.entry_date >= $1::date
			  and d.entry_date <= $2::date
			group by label
		),
		withdrawal_rows as (
			select 'withdrawals' as section,
			       coalesce(nullif(b.name, ''), nullif(b.code, ''), 'Stock Withdrawals') as label,
			       coalesce(sum(sl.qty_delta * sl.unit_cost), 0) as amount
			from stock_ledger sl
			join documents d on d.id = sl.document_id
			left join branches b on b.id = d.branch_id
			where d.kind in ('stock-out', 'stock-transactions')
			  and d.entry_date >= $1::date
			  and d.entry_date <= $2::date
			group by label
		),
		expense_rows as (
			select 'operating_expenses' as section,
			       coalesce(nullif(ec.name, ''), 'Uncategorized') as label,
			       coalesce(sum(case
			       	when coalesce(dl.payload->>'total', '') ~ '^-?\d+(\.\d+)?$' then (dl.payload->>'total')::numeric
			       	when coalesce(dl.amount, 0) <> 0 then dl.amount
			       	else coalesce(dl.cash_amount, 0) + coalesce(dl.check_amount, 0)
			       end), 0) as amount
			from documents d
			join document_lines dl on dl.document_id = d.id and dl.group_key = 'details'
			left join expense_charts ec on ec.id = dl.code_id
			where d.kind = 'expenses'
			  and d.entry_date >= $1::date
			  and d.entry_date <= $2::date
			group by label
		),
		other_income_rows as (
			select 'other_income' as section,
			       coalesce(nullif(oic.name, ''), 'Other Income') as label,
			       coalesce(sum(case
			       	when coalesce(dl.payload->>'total', '') ~ '^-?\d+(\.\d+)?$' then (dl.payload->>'total')::numeric
			       	when coalesce(dl.amount, 0) <> 0 then dl.amount
			       	else coalesce(dl.cash_amount, 0) + coalesce(dl.check_amount, 0)
			       end), 0) as amount
			from documents d
			join document_lines dl on dl.document_id = d.id and dl.group_key = 'details'
			left join other_income_charts oic on oic.id = dl.code_id
			where d.kind = 'other-income'
			  and d.entry_date >= $1::date
			  and d.entry_date <= $2::date
			group by label
			union all
			select 'other_income' as section,
			       'Rebates - ' || coalesce(nullif(c.company, ''), nullif(c.code, ''), 'No Customer') as label,
			       coalesce(sum(d.net), 0) as amount
			from documents d
			left join customers c on d.party_type = 'customer' and c.id = d.party_id
			where d.kind = 'rebates'
			  and d.entry_date >= $1::date
			  and d.entry_date <= $2::date
			group by label
			union all
			select 'other_income' as section,
			       'Purchases - ' || coalesce(nullif(dl.payload->>'particulars', ''), 'Discount') as label,
			       coalesce(sum(case
			       	when coalesce(dl.payload->>'amount', '') ~ '^-?\d+(\.\d+)?$' then (dl.payload->>'amount')::numeric
			       	else dl.qty * dl.price
			       end), 0) as amount
			from documents d
			join document_lines dl on dl.document_id = d.id and dl.group_key = 'discounts'
			where d.kind = 'purchases'
			  and d.entry_date >= $1::date
			  and d.entry_date <= $2::date
			group by label
		),
		sales_markup_rows as (
			select coalesce(nullif(st.category_group, ''), 'Uncategorized') as category,
			       coalesce(sum(case
			         when coalesce(dl.amount, 0) <> 0 then dl.amount
			         when nullif(regexp_replace(coalesce(dl.payload->>'amount', ''), '[,\s]', '', 'g'), '') ~ '^-?\d+(\.\d+)?$'
			           then regexp_replace(dl.payload->>'amount', '[,\s]', '', 'g')::numeric
			         else coalesce(dl.qty, 0) * coalesce(dl.price, 0)
			       end), 0) as amount,
			       coalesce(sum(case
			         when nullif(regexp_replace(coalesce(dl.payload->>'markup', ''), '[,\s]', '', 'g'), '') ~ '^-?\d+(\.\d+)?$'
			           then regexp_replace(dl.payload->>'markup', '[,\s]', '', 'g')::numeric
			         else (
			           case
			             when coalesce(dl.amount, 0) <> 0 then dl.amount
			             when nullif(regexp_replace(coalesce(dl.payload->>'amount', ''), '[,\s]', '', 'g'), '') ~ '^-?\d+(\.\d+)?$'
			               then regexp_replace(dl.payload->>'amount', '[,\s]', '', 'g')::numeric
			             else coalesce(dl.qty, 0) * coalesce(dl.price, 0)
			           end
			         ) - (
			           case
			             when nullif(regexp_replace(coalesce(dl.payload->>'capital_total', ''), '[,\s]', '', 'g'), '') ~ '^-?\d+(\.\d+)?$'
			               then regexp_replace(dl.payload->>'capital_total', '[,\s]', '', 'g')::numeric
			             when nullif(regexp_replace(coalesce(dl.payload->>'capital', ''), '[,\s]', '', 'g'), '') ~ '^-?\d+(\.\d+)?$'
			               then coalesce(dl.qty, 0) * regexp_replace(dl.payload->>'capital', '[,\s]', '', 'g')::numeric
			             else coalesce(dl.qty, 0) * coalesce(dl.unit_cost, 0)
			           end
			         )
			       end), 0) as markup
			from documents d
			join document_lines dl on dl.document_id = d.id and dl.group_key = 'details'
			left join stocks st on st.id = dl.stock_id
			where d.kind = 'sales'
			  and d.document_date >= $1::date
			  and d.document_date <= $2::date
			  and coalesce(dl.qty, 0) <> 0
			group by category
		),
		transfer_markup_rows as (
			select coalesce(nullif(st.category_group, ''), 'Uncategorized') as category,
			       coalesce(case when nullif(d.payload->'values'->>'branch_location', '') !~ '^\d+$' then nullif(d.payload->'values'->>'branch_location', '') end, nullif(b.name, ''), nullif(b.code, ''), 'No Branch') as branch,
			       coalesce(sum(case
			         when coalesce(dl.amount, 0) <> 0 then dl.amount
			         when nullif(regexp_replace(coalesce(dl.payload->>'amount', ''), '[,\s]', '', 'g'), '') ~ '^-?\d+(\.\d+)?$'
			           then regexp_replace(dl.payload->>'amount', '[,\s]', '', 'g')::numeric
			         else coalesce(dl.qty, 0) * coalesce(dl.unit_cost, 0)
			       end), 0) as amount,
			       coalesce(sum(case
			         when nullif(regexp_replace(coalesce(dl.payload->>'markup', ''), '[,\s]', '', 'g'), '') ~ '^-?\d+(\.\d+)?$'
			           then regexp_replace(dl.payload->>'markup', '[,\s]', '', 'g')::numeric
			         else (
			           case
			             when coalesce(dl.amount, 0) <> 0 then dl.amount
			             when nullif(regexp_replace(coalesce(dl.payload->>'amount', ''), '[,\s]', '', 'g'), '') ~ '^-?\d+(\.\d+)?$'
			               then regexp_replace(dl.payload->>'amount', '[,\s]', '', 'g')::numeric
			             else coalesce(dl.qty, 0) * coalesce(dl.unit_cost, 0)
			           end
			         ) - (
			           case
			             when nullif(regexp_replace(coalesce(dl.payload->>'capital_total', ''), '[,\s]', '', 'g'), '') ~ '^-?\d+(\.\d+)?$'
			               then regexp_replace(dl.payload->>'capital_total', '[,\s]', '', 'g')::numeric
			             when nullif(regexp_replace(coalesce(dl.payload->>'capital', ''), '[,\s]', '', 'g'), '') ~ '^-?\d+(\.\d+)?$'
			               then coalesce(dl.qty, 0) * regexp_replace(dl.payload->>'capital', '[,\s]', '', 'g')::numeric
			             else coalesce(dl.qty, 0) * coalesce(dl.unit_cost, 0)
			           end
			         )
			       end), 0) as markup
			from documents d
			join document_lines dl on dl.document_id = d.id and dl.group_key = 'details'
			left join stocks st on st.id = dl.stock_id
			left join branches b on b.id = coalesce(case when nullif(d.payload->'values'->>'branch_location', '') ~ '^\d+$' then nullif(d.payload->'values'->>'branch_location', '')::bigint end, d.branch_id)
			where d.kind = 'stock-transactions'
			  and coalesce(nullif(d.payload->'values'->>'transfer_date', '')::date, d.document_date) >= $1::date
			  and coalesce(nullif(d.payload->'values'->>'transfer_date', '')::date, d.document_date) <= $2::date
			  and (coalesce(dl.qty, 0) <> 0 or coalesce(dl.amount, 0) <> 0)
			group by category, branch
		),
		transfer_category_rows as (
			select category,
			       coalesce(sum(amount), 0) as amount,
			       coalesce(sum(markup), 0) as markup
			from transfer_markup_rows
			group by category
		),
		all_rows as (
			select section, label, ''::text as branch, amount, 0::numeric as net_sales, 0::numeric as sales_markup, 0::numeric as net_transfer, 0::numeric as transfer_markup
			from sales_rows
			union all select section, label, ''::text, amount, 0::numeric, 0::numeric, 0::numeric, 0::numeric from inventory_rows
			union all select section, label, ''::text, amount, 0::numeric, 0::numeric, 0::numeric, 0::numeric from purchase_rows
			union all select section, label, ''::text, amount, 0::numeric, 0::numeric, 0::numeric, 0::numeric from withdrawal_rows
			union all select section, label, ''::text, amount, 0::numeric, 0::numeric, 0::numeric, 0::numeric from expense_rows
			union all select section, label, ''::text, amount, 0::numeric, 0::numeric, 0::numeric, 0::numeric from other_income_rows
			union all
			select 'markup_category' as section,
			       coalesce(s.category, t.category) as label,
			       ''::text as branch,
			       0::numeric as amount,
			       coalesce(s.amount, 0) as net_sales,
			       coalesce(s.markup, 0) as sales_markup,
			       coalesce(t.amount, 0) as net_transfer,
			       coalesce(t.markup, 0) as transfer_markup
			from sales_markup_rows s
			full outer join transfer_category_rows t on lower(t.category) = lower(s.category)
			union all
			select 'markup_transfer_branch' as section,
			       category as label,
			       branch,
			       0::numeric as amount,
			       0::numeric as net_sales,
			       0::numeric as sales_markup,
			       amount as net_transfer,
			       markup as transfer_markup
			from transfer_markup_rows
		)
		select section,
		       label,
		       branch,
		       coalesce(round(sum(amount) * 100), 0)::bigint as amount_cents,
		       coalesce(round(sum(net_sales) * 100), 0)::bigint as net_sales_cents,
		       coalesce(round(sum(sales_markup) * 100), 0)::bigint as sales_markup_cents,
		       coalesce(round(sum(net_transfer) * 100), 0)::bigint as net_transfer_cents,
		       coalesce(round(sum(transfer_markup) * 100), 0)::bigint as transfer_markup_cents
		from all_rows
		group by section, label, branch
		having coalesce(sum(amount), 0) <> 0
		   or coalesce(sum(net_sales), 0) <> 0
		   or coalesce(sum(sales_markup), 0) <> 0
		   or coalesce(sum(net_transfer), 0) <> 0
		   or coalesce(sum(transfer_markup), 0) <> 0
		   or section in ('beginning_inventory', 'ending_inventory', 'cash_sales', 'charge_sales')
		order by case section
			when 'cash_sales' then 1
			when 'charge_sales' then 2
			when 'sales_return' then 3
			when 'beginning_inventory' then 4
			when 'purchases' then 5
			when 'withdrawals' then 6
			when 'ending_inventory' then 7
			when 'operating_expenses' then 8
			when 'other_income' then 9
			when 'markup_category' then 10
			when 'markup_transfer_branch' then 11
			else 10
		end, label, branch`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var reportRows []models.IncomeStatementRow
	for rows.Next() {
		var row models.IncomeStatementRow
		if err := rows.Scan(&row.Section, &row.Label, &row.Branch, &row.AmountCents, &row.NetSalesCents, &row.SalesMarkupCents, &row.NetTransferCents, &row.TransferMarkupCents); err != nil {
			return nil, err
		}
		reportRows = append(reportRows, row)
	}
	return reportRows, rows.Err()
}

func (s *PostgresStore) IncentiveReportRows(ctx context.Context, from, to time.Time) ([]models.IncentiveReportRow, error) {
	rows, err := s.pool.Query(ctx, `
		with sales_lines as (
			select coalesce(nullif(st.category_group, ''), 'Uncategorized') as agri_post,
			       coalesce(dl.qty, 0) as qty,
			       lower(regexp_replace(coalesce(c.aps, ''), '[^a-z0-9]+', '', 'g')) as aps_key,
			       coalesce(c.farm_customer, false) as farm_customer
			from documents d
			join document_lines dl on dl.document_id = d.id and dl.group_key = 'details'
			left join stocks st on st.id = dl.stock_id
			left join customers c on d.party_type = 'customer' and c.id = d.party_id
			where d.kind = 'sales'
			  and d.document_date >= $1::date
			  and d.document_date <= $2::date
			  and coalesce(dl.qty, 0) <> 0
			  and exists (
			  	select 1
			  	from stock_categories sc
			  	where sc.aps_monitor
			  	  and (
			  	  	lower(sc.name) = lower(coalesce(st.category_group, ''))
			  	  	or lower(sc.group_name) = lower(coalesce(st.category_group, ''))
			  	  )
			  )
		)
		select agri_post,
		       coalesce(round(sum(qty)), 0)::bigint as qty,
		       coalesce(round(sum(case when not farm_customer and aps_key = 'vip' then qty else 0 end)), 0)::bigint as vip_qty,
		       coalesce(round(sum(case when not farm_customer and aps_key = 'aps' then qty else 0 end)), 0)::bigint as aps_qty,
		       coalesce(round(sum(case when not farm_customer and aps_key = 'takals' then qty else 0 end)), 0)::bigint as takals_qty,
		       coalesce(round(sum(case when farm_customer then qty else 0 end)), 0)::bigint as farm_qty
		from sales_lines
		group by agri_post
		order by agri_post`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var reportRows []models.IncentiveReportRow
	for rows.Next() {
		var row models.IncentiveReportRow
		if err := rows.Scan(&row.AgriPost, &row.Qty, &row.VIP, &row.APS, &row.Takals, &row.Farm); err != nil {
			return nil, err
		}
		reportRows = append(reportRows, row)
	}
	return reportRows, rows.Err()
}

func (s *PostgresStore) DailySalesCollectionReportRows(ctx context.Context, reportDate time.Time) ([]models.DailySalesCollectionReportRow, error) {
	rows, err := s.pool.Query(ctx, `
		with sales_rows as (
			select case
			       	when upper(trim(coalesce(nullif(d.payload->'values'->>'or_ci_number', ''), nullif(d.reference, ''), nullif(d.entry_id, ''), ''))) ~ '^CHG([[:space:]_-]|[0-9])' then 'charge_sales'
			       	when upper(trim(coalesce(nullif(d.payload->'values'->>'or_ci_number', ''), nullif(d.reference, ''), nullif(d.entry_id, ''), ''))) ~ '^CI([[:space:]_-]|[0-9])' then 'cash_sales'
			       	when d.cash then 'cash_sales'
			       	else 'charge_sales'
			       end as section,
			       coalesce(nullif(c.company, ''), nullif(c.code, ''), 'No Customer') as name,
			       trim(concat(case when d.cash then 'CSH ' else 'CHG ' end,
			         coalesce(nullif(d.payload->'values'->>'or_ci_number', ''), nullif(d.reference, ''), nullif(d.entry_id, ''), '')
			       )) as reference,
			       coalesce(d.net, d.total, 0) as amount,
			       0::numeric as check_amount,
			       to_char(d.entry_date at time zone 'Asia/Manila', 'YYYYMMDDHH24MISSUS') || '-' || lpad(d.id::text, 20, '0') as sort_key
			from documents d
			left join customers c on d.party_type = 'customer' and c.id = d.party_id
			where d.kind = 'sales'
			  and d.document_date = $1::date
		),
		cash_receipts as (
			select case
			       	when upper(trim(coalesce(nullif(d.reference, ''), nullif(d.payload->'values'->>'or_ci_number', ''), nullif(d.entry_id, ''), ''))) ~ '^CR([[:space:]_-]|[0-9])' then 'cash_receipts'
			       	when upper(trim(coalesce(nullif(d.reference, ''), nullif(d.payload->'values'->>'or_ci_number', ''), nullif(d.entry_id, ''), ''))) ~ '^CI([[:space:]_-]|[0-9])' then 'cash_sales'
			       	when upper(trim(coalesce(nullif(d.reference, ''), nullif(d.payload->'values'->>'or_ci_number', ''), nullif(d.entry_id, ''), ''))) ~ '^CHG([[:space:]_-]|[0-9])' then 'charge_sales'
			       	else 'cash_receipts'
			       end as section,
			       coalesce(nullif(c.company, ''), nullif(c.code, ''), 'No Customer') as name,
			       coalesce(nullif(d.reference, ''), nullif(d.entry_id, ''), case when d.kind = 'rebates' then 'Rebates' else 'AR Credit' end) as reference,
			       case
			       	when coalesce(d.payload->'values'->>'cash_amount', '') ~ '^-?\d+(\.\d+)?$' then (d.payload->'values'->>'cash_amount')::numeric
			       	else 0
			       end as amount,
			       coalesce((
			       	select sum(case
			       		when coalesce(check_line.payload->>'amount', '') ~ '^-?\d+(\.\d+)?$' then (check_line.payload->>'amount')::numeric
			       		else coalesce(check_line.amount, check_line.check_amount, 0)
			       	end)
			       	from document_lines check_line
			       	where check_line.document_id = d.id
			       	  and check_line.group_key = 'checks'
			       ), 0) as check_amount,
			       to_char(d.entry_date at time zone 'Asia/Manila', 'YYYYMMDDHH24MISSUS') || '-' || lpad(d.id::text, 20, '0') as sort_key
			from documents d
			left join customers c on d.party_type = 'customer' and c.id = d.party_id
			where d.kind in ('ar-credit', 'rebates')
			  and d.entry_date::date = $1::date
		),
		disbursements as (
			select case
			       	when upper(trim(coalesce(nullif(dl.payload->>'reference', ''), nullif(d.reference, ''), nullif(d.entry_id, ''), ''))) ~ '^CV([[:space:]_-]|[0-9])' then 'disbursements'
			       	when upper(trim(coalesce(nullif(dl.payload->>'reference', ''), nullif(d.reference, ''), nullif(d.entry_id, ''), ''))) ~ '^CR([[:space:]_-]|[0-9])' then 'cash_receipts'
			       	else 'disbursements'
			       end as section,
			       coalesce(nullif(ec.name, ''), 'Uncategorized') as name,
			       coalesce(nullif(dl.payload->>'reference', ''), nullif(d.reference, ''), nullif(d.entry_id, ''), '') as reference,
			       case
			       	when coalesce(dl.amount, 0) <> 0 then dl.amount
			       	when coalesce(dl.payload->>'total', '') ~ '^-?\d+(\.\d+)?$' then (dl.payload->>'total')::numeric
			       	when coalesce(dl.cash_amount, 0) <> 0 or coalesce(dl.check_amount, 0) <> 0 then coalesce(dl.cash_amount, 0) + coalesce(dl.check_amount, 0)
			       	when coalesce(dl.payload->>'cash', '') ~ '^-?\d+(\.\d+)?$' or coalesce(dl.payload->>'check', '') ~ '^-?\d+(\.\d+)?$' then
			       		(case when coalesce(dl.payload->>'cash', '') ~ '^-?\d+(\.\d+)?$' then (dl.payload->>'cash')::numeric else 0 end) +
			       		(case when coalesce(dl.payload->>'check', '') ~ '^-?\d+(\.\d+)?$' then (dl.payload->>'check')::numeric else 0 end)
			       	else 0
			       end as amount,
			       0::numeric as check_amount,
			       to_char(d.entry_date at time zone 'Asia/Manila', 'YYYYMMDDHH24MISSUS') || '-' ||
			         lpad(d.id::text, 20, '0') || '-' || lpad(dl.line_no::text, 8, '0') as sort_key
			from documents d
			join document_lines dl on dl.document_id = d.id and dl.group_key = 'details'
			left join expense_charts ec on ec.id = dl.code_id
			where d.kind = 'expenses'
			  and d.entry_date::date = $1::date
			  and not coalesce(ec.exclude_daily_sales, false)
		),
		check_deposits as (
			select 'check_deposits' as section,
			       coalesce(nullif(c.company, ''), nullif(c.code, ''), nullif(d.payload->'values'->>'payee', ''), 'No Payee') as name,
			       coalesce(nullif(dl.payload->>'number', ''), nullif(d.reference, ''), nullif(d.entry_id, ''), '') as reference,
			       0::numeric as amount,
			       case
			       	when coalesce(dl.payload->>'amount', '') ~ '^-?\d+(\.\d+)?$' then (dl.payload->>'amount')::numeric
			       	else coalesce(dl.amount, dl.check_amount, 0)
			       end as check_amount,
			       to_char(d.entry_date at time zone 'Asia/Manila', 'YYYYMMDDHH24MISSUS') || '-' ||
			         lpad(d.id::text, 20, '0') || '-' || lpad(dl.line_no::text, 8, '0') as sort_key
			from document_lines dl
			join documents d on d.id = dl.document_id
			left join customers c on d.party_type = 'customer' and c.id = d.party_id
			where dl.group_key in ('checks', 'payments')
			  and d.kind in ('ar-credit', 'rebates', 'sales', 'checks-in')
			  and (
			  	(d.kind in ('ar-credit', 'rebates') and d.entry_date::date = $1::date)
			  	or (
			  		d.kind not in ('ar-credit', 'rebates')
			  		and case
			  			when coalesce(dl.payload->>'date', '') ~ '^\d{4}-\d{2}-\d{2}$' then (dl.payload->>'date')::date
			  			else d.entry_date::date
			  		end = $1::date
			  	)
			  )
		),
		all_rows as (
			select * from sales_rows
			union all select * from cash_receipts
			union all select * from disbursements
			union all select * from check_deposits
		)
		select section,
		       name,
		       reference,
		       coalesce(round(sum(amount) * 100), 0)::bigint as amount_cents,
		       coalesce(round(sum(check_amount) * 100), 0)::bigint as check_amount_cents,
		       min(sort_key) as sort_key
		from all_rows
		group by section, name, reference
		having coalesce(sum(amount), 0) <> 0
		    or coalesce(sum(check_amount), 0) <> 0
		order by case section
			when 'cash_sales' then 1
			when 'charge_sales' then 2
			when 'cash_receipts' then 3
			when 'disbursements' then 4
			when 'check_deposits' then 5
			else 6
		end, min(sort_key), name, reference`, reportDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var reportRows []models.DailySalesCollectionReportRow
	for rows.Next() {
		var row models.DailySalesCollectionReportRow
		if err := rows.Scan(&row.Section, &row.Name, &row.Reference, &row.AmountCents, &row.CheckAmountCents, &row.SortKey); err != nil {
			return nil, err
		}
		reportRows = append(reportRows, row)
	}
	return reportRows, rows.Err()
}

func (s *PostgresStore) StockSalesTransferReportRows(ctx context.Context, from, to time.Time) ([]models.StockSalesTransferReportRow, error) {
	rows, err := s.pool.Query(ctx, `
		with movement_lines as (
			select coalesce(nullif(st.category_group, ''), 'Uncategorized') as category,
			       coalesce(st.code, '') as stock_code,
			       coalesce(nullif(st.name, ''), nullif(st.code, ''), 'No Stock') as stock_name,
			       case when d.kind = 'sales' then coalesce(dl.qty, 0) else 0 end as sales_qty,
			       case when d.kind = 'stock-transactions' then coalesce(dl.qty, 0) else 0 end as transfer_qty
			from documents d
			join document_lines dl on dl.document_id = d.id and dl.group_key = 'details'
			left join stocks st on st.id = dl.stock_id
			where d.kind in ('sales', 'stock-transactions')
			  and case
			    when d.kind = 'sales' then d.document_date
			    when d.kind = 'stock-transactions' then d.document_date
			    else d.entry_date::date
			  end >= $1::date
			  and case
			    when d.kind = 'sales' then d.document_date
			    when d.kind = 'stock-transactions' then d.document_date
			    else d.entry_date::date
			  end <= $2::date
			  and coalesce(dl.qty, 0) <> 0
		)
		select category,
		       stock_code,
		       stock_name,
		       coalesce(round(sum(sales_qty)), 0)::bigint as sales_qty,
		       coalesce(round(sum(transfer_qty)), 0)::bigint as transfer_qty
		from movement_lines
		group by category, stock_code, stock_name
		having coalesce(sum(sales_qty), 0) <> 0
		    or coalesce(sum(transfer_qty), 0) <> 0
		order by category, stock_code, stock_name`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var reportRows []models.StockSalesTransferReportRow
	for rows.Next() {
		var row models.StockSalesTransferReportRow
		if err := rows.Scan(&row.Category, &row.StockCode, &row.StockName, &row.SalesQty, &row.TransferQty); err != nil {
			return nil, err
		}
		reportRows = append(reportRows, row)
	}
	return reportRows, rows.Err()
}

func (s *PostgresStore) StockSalesTransferAmountReportRows(ctx context.Context, from, to time.Time) ([]models.StockSalesTransferAmountReportRow, error) {
	rows, err := s.pool.Query(ctx, `
		with movement_lines as (
			select coalesce(nullif(st.category_group, ''), 'Uncategorized') as category,
			       d.kind,
			       coalesce(d.cash, false) as cash,
			       case
			         when coalesce(dl.amount, 0) <> 0 then dl.amount
			         when coalesce(dl.payload->>'amount', '') ~ '^-?\d+(\.\d+)?$' then (dl.payload->>'amount')::numeric
			         when d.kind = 'sales' then coalesce(dl.qty, 0) * coalesce(dl.price, dl.unit_cost, 0)
			         else coalesce(dl.qty, 0) * coalesce(dl.unit_cost, 0)
			       end as amount,
			       coalesce(dl.qty, 0) * coalesce(dl.unit_cost, 0) as cost
			from documents d
			join document_lines dl on dl.document_id = d.id and dl.group_key = 'details'
			left join stocks st on st.id = dl.stock_id
			where d.kind in ('sales', 'stock-transactions')
			  and case
			    when d.kind = 'sales' then d.document_date
			    when d.kind = 'stock-transactions' then d.document_date
			    else d.entry_date::date
			  end >= $1::date
			  and case
			    when d.kind = 'sales' then d.document_date
			    when d.kind = 'stock-transactions' then d.document_date
			    else d.entry_date::date
			  end <= $2::date
		)
		select category,
		       coalesce(round(sum(case when kind = 'sales' and cash then amount else 0 end) * 100), 0)::bigint,
		       coalesce(round(sum(case when kind = 'sales' and not cash then amount else 0 end) * 100), 0)::bigint,
		       coalesce(round(sum(case when kind = 'stock-transactions' then amount else 0 end) * 100), 0)::bigint,
		       coalesce(round(sum(case when kind = 'sales' then amount - cost else 0 end) * 100), 0)::bigint,
		       coalesce(round(sum(case when kind = 'stock-transactions' then amount - cost else 0 end) * 100), 0)::bigint
		from movement_lines
		group by category
		having coalesce(sum(amount), 0) <> 0
		order by category`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var reportRows []models.StockSalesTransferAmountReportRow
	for rows.Next() {
		var row models.StockSalesTransferAmountReportRow
		if err := rows.Scan(&row.Category, &row.CashSalesCents, &row.ChargeSalesCents, &row.TransferCents, &row.SalesMarkupCents, &row.TransferMarkupCents); err != nil {
			return nil, err
		}
		reportRows = append(reportRows, row)
	}
	return reportRows, rows.Err()
}

func (s *PostgresStore) StockTransferSummaryReportRows(ctx context.Context, from, to time.Time) ([]models.StockTransferSummaryReportRow, error) {
	rows, err := s.pool.Query(ctx, `
		with transfer_lines as (
			select d.*,
			       dl.stock_id,
			       dl.qty,
			       dl.unit_cost,
			       dl.amount as line_amount,
			       dl.payload as line_payload,
			       coalesce(nullif(d.payload->'values'->>'transfer_date', '')::date, d.document_date) as report_date,
			       nullif(d.payload->'values'->>'branch_location', '') as transfer_branch
			from documents d
			join document_lines dl on dl.document_id = d.id and dl.group_key = 'details'
			where d.kind = 'stock-transactions'
			  and coalesce(nullif(d.payload->'values'->>'transfer_date', '')::date, d.document_date) >= $1::date
			  and coalesce(nullif(d.payload->'values'->>'transfer_date', '')::date, d.document_date) <= $2::date
			  and (coalesce(dl.qty, 0) <> 0 or coalesce(dl.amount, 0) <> 0)
		)
		select coalesce(nullif(st.category_group, ''), 'Uncategorized') as category,
		       coalesce(case when transfer_branch !~ '^\d+$' then transfer_branch end, nullif(b.name, ''), nullif(b.code, ''), 'No Branch') as branch,
		       coalesce(nullif(d.payload->'values'->>'transfer_id', ''), nullif(d.reference, ''), nullif(d.entry_id, ''), '') as reference,
		       to_char(d.report_date, 'MM/DD/YYYY') as transfer_date,
		       coalesce(st.code, '') as stock_code,
		       coalesce(nullif(st.name, ''), nullif(st.code, ''), 'No Stock') as stock_name,
		       coalesce(round(d.qty), 0)::bigint as quantity,
		       coalesce(round((case
		         when coalesce(d.line_amount, 0) <> 0 then d.line_amount
		         when nullif(regexp_replace(coalesce(d.line_payload->>'amount', ''), '[,\s]', '', 'g'), '') ~ '^-?\d+(\.\d+)?$'
		           then regexp_replace(d.line_payload->>'amount', '[,\s]', '', 'g')::numeric
		         else coalesce(d.qty, 0) * coalesce(d.unit_cost, 0)
		       end) * 100), 0)::bigint as amount_cents
		from transfer_lines d
		left join stocks st on st.id = d.stock_id
		left join branches b on b.id = coalesce(case when d.transfer_branch ~ '^\d+$' then d.transfer_branch::bigint end, d.branch_id)
		order by category, branch, transfer_date, reference, stock_code, stock_name`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var reportRows []models.StockTransferSummaryReportRow
	for rows.Next() {
		var row models.StockTransferSummaryReportRow
		if err := rows.Scan(&row.Category, &row.Branch, &row.Reference, &row.TransferDate, &row.StockCode, &row.StockName, &row.Quantity, &row.AmountCents); err != nil {
			return nil, err
		}
		reportRows = append(reportRows, row)
	}
	return reportRows, rows.Err()
}

func (s *PostgresStore) StockTransferByStockNameReportRows(ctx context.Context, from, to time.Time) ([]models.StockTransferByStockNameReportRow, error) {
	rows, err := s.pool.Query(ctx, `
		with transfer_lines as (
			select d.*,
			       dl.stock_id,
			       dl.qty,
			       dl.unit_cost,
			       dl.amount as line_amount,
			       dl.payload as line_payload,
			       coalesce(nullif(d.payload->'values'->>'transfer_date', '')::date, d.document_date) as report_date,
			       nullif(d.payload->'values'->>'branch_location', '') as transfer_branch
			from documents d
			join document_lines dl on dl.document_id = d.id and dl.group_key = 'details'
			where d.kind = 'stock-transactions'
			  and coalesce(nullif(d.payload->'values'->>'transfer_date', '')::date, d.document_date) >= $1::date
			  and coalesce(nullif(d.payload->'values'->>'transfer_date', '')::date, d.document_date) <= $2::date
			  and (coalesce(dl.qty, 0) <> 0 or coalesce(dl.amount, 0) <> 0)
		)
		select coalesce(nullif(st.category_group, ''), 'Uncategorized') as category,
		       coalesce(case when transfer_branch !~ '^\d+$' then transfer_branch end, nullif(b.name, ''), nullif(b.code, ''), 'No Branch') as branch,
		       coalesce(nullif(d.reference, ''), nullif(d.entry_id, ''), d.id::text) as reference,
		       coalesce(nullif(d.payload->'values'->>'transfer_id', ''), nullif(d.entry_id, ''), '') as transfer_id,
		       to_char(d.report_date, 'MM/DD/YYYY') as transfer_date,
		       coalesce(st.code, '') as stock_code,
		       coalesce(nullif(st.name, ''), nullif(st.code, ''), 'No Stock') as stock_name,
		       coalesce(round(d.qty), 0)::bigint as quantity,
		       coalesce(round((case
		         when coalesce(d.line_amount, 0) <> 0 then d.line_amount
		         when nullif(regexp_replace(coalesce(d.line_payload->>'amount', ''), '[,\s]', '', 'g'), '') ~ '^-?\d+(\.\d+)?$'
		           then regexp_replace(d.line_payload->>'amount', '[,\s]', '', 'g')::numeric
		         else coalesce(d.qty, 0) * coalesce(d.unit_cost, 0)
		       end) * 100), 0)::bigint as amount_cents
		from transfer_lines d
		left join stocks st on st.id = d.stock_id
		left join branches b on b.id = coalesce(case when d.transfer_branch ~ '^\d+$' then d.transfer_branch::bigint end, d.branch_id)
		order by category, stock_name, stock_code, branch, transfer_date, reference, transfer_id`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var reportRows []models.StockTransferByStockNameReportRow
	for rows.Next() {
		var row models.StockTransferByStockNameReportRow
		if err := rows.Scan(&row.Category, &row.Branch, &row.Reference, &row.TransferID, &row.TransferDate, &row.StockCode, &row.StockName, &row.Quantity, &row.AmountCents); err != nil {
			return nil, err
		}
		reportRows = append(reportRows, row)
	}
	return reportRows, rows.Err()
}

func (s *PostgresStore) StockTransferByBranchReportRows(ctx context.Context, from, to time.Time) ([]models.StockTransferByBranchReportRow, error) {
	rows, err := s.pool.Query(ctx, `
		with transfer_lines as (
			select d.*,
			       dl.stock_id,
			       dl.qty,
			       dl.unit_cost,
			       dl.amount as line_amount,
			       dl.payload as line_payload,
			       coalesce(nullif(d.payload->'values'->>'transfer_date', '')::date, d.document_date) as report_date,
			       nullif(d.payload->'values'->>'branch_location', '') as transfer_branch
			from documents d
			join document_lines dl on dl.document_id = d.id and dl.group_key = 'details'
			where d.kind = 'stock-transactions'
			  and coalesce(nullif(d.payload->'values'->>'transfer_date', '')::date, d.document_date) >= $1::date
			  and coalesce(nullif(d.payload->'values'->>'transfer_date', '')::date, d.document_date) <= $2::date
			  and (coalesce(dl.qty, 0) <> 0 or coalesce(dl.amount, 0) <> 0)
		)
		select coalesce(case when transfer_branch !~ '^\d+$' then transfer_branch end, nullif(b.name, ''), nullif(b.code, ''), 'No Branch') as branch,
		       coalesce(nullif(st.category_group, ''), 'Uncategorized') as category,
		       coalesce(nullif(d.reference, ''), nullif(d.entry_id, ''), d.id::text) as reference,
		       to_char(d.report_date, 'MM/DD/YYYY') as transfer_date,
		       coalesce(st.code, '') as stock_code,
		       coalesce(nullif(st.name, ''), nullif(st.code, ''), 'No Stock') as stock_name,
		       coalesce(round(d.qty), 0)::bigint as quantity,
		       coalesce(round((case
		         when coalesce(d.line_amount, 0) <> 0 then d.line_amount
		         when nullif(regexp_replace(coalesce(d.line_payload->>'amount', ''), '[,\s]', '', 'g'), '') ~ '^-?\d+(\.\d+)?$'
		           then regexp_replace(d.line_payload->>'amount', '[,\s]', '', 'g')::numeric
		         else coalesce(d.qty, 0) * coalesce(d.unit_cost, 0)
		       end) * 100), 0)::bigint as amount_cents
		from transfer_lines d
		left join stocks st on st.id = d.stock_id
		left join branches b on b.id = coalesce(case when d.transfer_branch ~ '^\d+$' then d.transfer_branch::bigint end, d.branch_id)
		order by branch, category, transfer_date, reference, stock_code, stock_name`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var reportRows []models.StockTransferByBranchReportRow
	for rows.Next() {
		var row models.StockTransferByBranchReportRow
		if err := rows.Scan(&row.Branch, &row.Category, &row.Reference, &row.TransferDate, &row.StockCode, &row.StockName, &row.Quantity, &row.AmountCents); err != nil {
			return nil, err
		}
		reportRows = append(reportRows, row)
	}
	return reportRows, rows.Err()
}

func (s *PostgresStore) StockTransferByEntryIDReportRows(ctx context.Context, from, to time.Time) ([]models.StockTransferByEntryIDReportRow, error) {
	rows, err := s.pool.Query(ctx, `
		with transfer_lines as (
			select d.*,
			       dl.stock_id,
			       dl.qty,
			       dl.unit_cost,
			       dl.amount as line_amount,
			       dl.payload as line_payload,
			       coalesce(nullif(d.payload->'values'->>'transfer_date', '')::date, d.document_date) as report_date,
			       nullif(d.payload->'values'->>'branch_location', '') as transfer_branch
			from documents d
			join document_lines dl on dl.document_id = d.id and dl.group_key = 'details'
			where d.kind = 'stock-transactions'
			  and coalesce(nullif(d.payload->'values'->>'transfer_date', '')::date, d.document_date) >= $1::date
			  and coalesce(nullif(d.payload->'values'->>'transfer_date', '')::date, d.document_date) <= $2::date
			  and (coalesce(dl.qty, 0) <> 0 or coalesce(dl.amount, 0) <> 0)
		)
		select coalesce(nullif(d.entry_id, ''), d.id::text) as entry_id,
		       coalesce(nullif(d.reference, ''), nullif(d.entry_id, ''), d.id::text) as reference,
		       coalesce(nullif(d.payload->'values'->>'transfer_id', ''), nullif(d.entry_id, ''), '') as transfer_id,
		       coalesce(
		         nullif(trim(concat_ws(' ',
		           nullif(d.payload->'values'->>'transfer_id', ''),
		           nullif(d.payload->'values'->>'transaction', '')
		         )), ''),
		         nullif(d.remarks, ''),
		         nullif(d.reference, ''),
		         nullif(d.entry_id, ''),
		         d.id::text
		       ) as remarks,
		       to_char(d.report_date, 'MM/DD/YYYY') as transfer_date,
		       coalesce(case when transfer_branch !~ '^\d+$' then transfer_branch end, nullif(b.name, ''), nullif(b.code, ''), 'No Branch') as branch,
		       coalesce(st.code, '') as stock_code,
		       coalesce(nullif(st.name, ''), nullif(st.code, ''), 'No Stock') as stock_name,
		       coalesce(round(d.qty), 0)::bigint as quantity,
		       coalesce(round((case
		         when coalesce(d.line_amount, 0) <> 0 then d.line_amount
		         when nullif(regexp_replace(coalesce(d.line_payload->>'amount', ''), '[,\s]', '', 'g'), '') ~ '^-?\d+(\.\d+)?$'
		           then regexp_replace(d.line_payload->>'amount', '[,\s]', '', 'g')::numeric
		         else coalesce(d.qty, 0) * coalesce(d.unit_cost, 0)
		       end) * 100), 0)::bigint as amount_cents,
		       coalesce(round(d.net * 100), 0)::bigint as net_cents
		from transfer_lines d
		left join stocks st on st.id = d.stock_id
		left join branches b on b.id = coalesce(case when d.transfer_branch ~ '^\d+$' then d.transfer_branch::bigint end, d.branch_id)
		order by entry_id, transfer_date, branch, stock_name, stock_code`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var reportRows []models.StockTransferByEntryIDReportRow
	for rows.Next() {
		var row models.StockTransferByEntryIDReportRow
		if err := rows.Scan(&row.EntryID, &row.Reference, &row.TransferID, &row.Remarks, &row.TransferDate, &row.Branch, &row.StockCode, &row.StockName, &row.Quantity, &row.AmountCents, &row.NetCents); err != nil {
			return nil, err
		}
		reportRows = append(reportRows, row)
	}
	return reportRows, rows.Err()
}

func (s *PostgresStore) StockTransferSummaryByItemReportRows(ctx context.Context, from, to time.Time) ([]models.StockTransferSummaryByItemReportRow, error) {
	rows, err := s.pool.Query(ctx, `
		with transfer_lines as (
			select d.id,
			       dl.stock_id,
			       dl.qty,
			       dl.unit_cost,
			       dl.amount as line_amount,
			       dl.payload as line_payload
			from documents d
			join document_lines dl on dl.document_id = d.id and dl.group_key = 'details'
			where d.kind = 'stock-transactions'
			  and coalesce(nullif(d.payload->'values'->>'transfer_date', '')::date, d.document_date) >= $1::date
			  and coalesce(nullif(d.payload->'values'->>'transfer_date', '')::date, d.document_date) <= $2::date
			  and (coalesce(dl.qty, 0) <> 0 or coalesce(dl.amount, 0) <> 0)
		)
		select coalesce(nullif(st.category_group, ''), 'Uncategorized') as category,
		       coalesce(st.code, '') as stock_code,
		       coalesce(nullif(st.name, ''), nullif(st.code, ''), 'No Stock') as stock_name,
		       coalesce(round(sum(coalesce(d.qty, 0))), 0)::bigint as quantity,
		       coalesce(round(sum(case
		         when coalesce(d.line_amount, 0) <> 0 then d.line_amount
		         when nullif(regexp_replace(coalesce(d.line_payload->>'amount', ''), '[,\s]', '', 'g'), '') ~ '^-?\d+(\.\d+)?$'
		           then regexp_replace(d.line_payload->>'amount', '[,\s]', '', 'g')::numeric
		         else coalesce(d.qty, 0) * coalesce(d.unit_cost, 0)
		       end) * 100), 0)::bigint as amount_cents
		from transfer_lines d
		left join stocks st on st.id = d.stock_id
		group by 1, 2, 3
		order by 1, 3, 2`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var reportRows []models.StockTransferSummaryByItemReportRow
	for rows.Next() {
		var row models.StockTransferSummaryByItemReportRow
		if err := rows.Scan(&row.Category, &row.StockCode, &row.StockName, &row.Quantity, &row.AmountCents); err != nil {
			return nil, err
		}
		reportRows = append(reportRows, row)
	}
	return reportRows, rows.Err()
}

func (s *PostgresStore) StockTransferMarkupByTransactionReportRows(ctx context.Context, from, to time.Time) ([]models.StockTransferMarkupByTransactionReportRow, error) {
	rows, err := s.pool.Query(ctx, `
		with transfer_lines as (
			select to_char(coalesce(nullif(d.payload->'values'->>'transfer_date', '')::date, d.document_date), 'MM/DD/YYYY') as transfer_date,
			       coalesce(nullif(d.entry_id, ''), d.id::text) as entry_id,
			       coalesce(case when nullif(d.payload->'values'->>'branch_location', '') !~ '^\d+$' then nullif(d.payload->'values'->>'branch_location', '') end, nullif(b.name, ''), nullif(b.code, ''), 'No Branch') as transfer_to,
			       coalesce(nullif(d.payload->'values'->>'transfer_id', ''), nullif(d.reference, ''), nullif(d.entry_id, ''), 'No Receipt') as receipt_no,
			       coalesce(nullif(st.category_group, ''), 'Uncategorized') as item_group,
			       case
			         when coalesce(dl.amount, 0) <> 0 then dl.amount
			         when nullif(regexp_replace(coalesce(dl.payload->>'amount', ''), '[,\s]', '', 'g'), '') ~ '^-?\d+(\.\d+)?$'
			           then regexp_replace(dl.payload->>'amount', '[,\s]', '', 'g')::numeric
			         else coalesce(dl.qty, 0) * coalesce(dl.unit_cost, 0)
			       end as amount,
			       case
			         when nullif(regexp_replace(coalesce(dl.payload->>'capital_total', ''), '[,\s]', '', 'g'), '') ~ '^-?\d+(\.\d+)?$'
			           then regexp_replace(dl.payload->>'capital_total', '[,\s]', '', 'g')::numeric
			         when nullif(regexp_replace(coalesce(dl.payload->>'capital', ''), '[,\s]', '', 'g'), '') ~ '^-?\d+(\.\d+)?$'
			           then coalesce(dl.qty, 0) * regexp_replace(dl.payload->>'capital', '[,\s]', '', 'g')::numeric
			         else coalesce(dl.qty, 0) * coalesce(dl.unit_cost, 0)
			       end as capital,
			       case
			         when nullif(regexp_replace(coalesce(dl.payload->>'markup', ''), '[,\s]', '', 'g'), '') ~ '^-?\d+(\.\d+)?$'
			           then regexp_replace(dl.payload->>'markup', '[,\s]', '', 'g')::numeric
			         else (
			           case
			             when coalesce(dl.amount, 0) <> 0 then dl.amount
			             when nullif(regexp_replace(coalesce(dl.payload->>'amount', ''), '[,\s]', '', 'g'), '') ~ '^-?\d+(\.\d+)?$'
			               then regexp_replace(dl.payload->>'amount', '[,\s]', '', 'g')::numeric
			             else coalesce(dl.qty, 0) * coalesce(dl.unit_cost, 0)
			           end
			         ) - (
			           case
			             when nullif(regexp_replace(coalesce(dl.payload->>'capital_total', ''), '[,\s]', '', 'g'), '') ~ '^-?\d+(\.\d+)?$'
			               then regexp_replace(dl.payload->>'capital_total', '[,\s]', '', 'g')::numeric
			             when nullif(regexp_replace(coalesce(dl.payload->>'capital', ''), '[,\s]', '', 'g'), '') ~ '^-?\d+(\.\d+)?$'
			               then coalesce(dl.qty, 0) * regexp_replace(dl.payload->>'capital', '[,\s]', '', 'g')::numeric
			             else coalesce(dl.qty, 0) * coalesce(dl.unit_cost, 0)
			           end
			         )
			       end as markup
			from documents d
			join document_lines dl on dl.document_id = d.id and dl.group_key = 'details'
			left join stocks st on st.id = dl.stock_id
			left join branches b on b.id = coalesce(case when nullif(d.payload->'values'->>'branch_location', '') ~ '^\d+$' then nullif(d.payload->'values'->>'branch_location', '')::bigint end, d.branch_id)
			where d.kind = 'stock-transactions'
			  and coalesce(nullif(d.payload->'values'->>'transfer_date', '')::date, d.document_date) >= $1::date
			  and coalesce(nullif(d.payload->'values'->>'transfer_date', '')::date, d.document_date) <= $2::date
			  and (coalesce(dl.qty, 0) <> 0 or coalesce(dl.amount, 0) <> 0)
		)
		select transfer_date,
		       entry_id,
		       transfer_to,
		       receipt_no,
		       item_group,
		       coalesce(round(markup * 100), 0)::bigint as markup_cents,
		       coalesce(round(capital * 100), 0)::bigint as capital_cents,
		       coalesce(round(amount * 100), 0)::bigint as amount_cents
		from transfer_lines
		where coalesce(markup, 0) <> 0 or coalesce(capital, 0) <> 0
		order by transfer_date, entry_id, transfer_to, receipt_no, item_group`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var reportRows []models.StockTransferMarkupByTransactionReportRow
	for rows.Next() {
		var row models.StockTransferMarkupByTransactionReportRow
		if err := rows.Scan(&row.TransferDate, &row.EntryID, &row.TransferTo, &row.ReceiptNo, &row.ItemGroup, &row.MarkupCents, &row.CapitalCents, &row.AmountCents); err != nil {
			return nil, err
		}
		reportRows = append(reportRows, row)
	}
	return reportRows, rows.Err()
}

func (s *PostgresStore) StockLedgerReportRows(ctx context.Context, to time.Time) ([]models.StockLedgerReportRow, error) {
	rows, err := s.pool.Query(ctx, `
		with ledger as (
			select sl.stock_id,
			       case
			       	when d.kind = 'purchases' then d.document_date
			       	when d.kind = 'sales' then d.document_date
			       	when d.kind = 'stock-transactions' then d.document_date
			       	else d.entry_date::date
			       end as movement_date,
			       to_char(d.entry_date at time zone 'Asia/Manila', 'YYYYMMDDHH24MISSUS') || '-' ||
			         lpad(d.id::text, 20, '0') || '-' ||
			         lpad(sl.id::text, 20, '0') as sort_key,
			       coalesce(nullif(d.reference, ''), nullif(d.payload->'values'->>'transfer_id', ''), nullif(d.payload->'values'->>'or_ci_number', ''), nullif(d.entry_id, ''), '') as reference,
			       coalesce(nullif(
			       	case
			       		when d.party_type = 'supplier' then coalesce(s.company, s.code, '')
			       		when d.party_type = 'customer' then coalesce(c.company, c.code, '')
			       		else coalesce(b.name, b.code, '')
			       	end, ''), '') as company,
			       coalesce(d.kind, '') as kind,
			       coalesce(round(sl.qty_delta), 0)::bigint as qty_delta
			from stock_ledger sl
			join documents d on d.id = sl.document_id
			left join suppliers s on d.party_type = 'supplier' and s.id = d.party_id
			left join customers c on d.party_type = 'customer' and c.id = d.party_id
			left join branches b on b.id = coalesce(sl.branch_id, d.branch_id)
			where d.kind in ('purchases', 'sales', 'stock-transactions')
			  and case
			       	when d.kind = 'purchases' then d.document_date
			       	when d.kind = 'sales' then d.document_date
			       	when d.kind = 'stock-transactions' then d.document_date
			       	else d.entry_date::date
			      end <= $1::date
		),
		categories as (
			select distinct trim(name) as category
			from stock_categories
			where coalesce(trim(name), '') <> ''
			union
			select distinct coalesce(nullif(st.category_group, ''), 'Uncategorized') as category
			from stocks st
		),
		stock_rows as (
			select st.id::text as stock_id,
			       coalesce(nullif(st.category_group, ''), 'Uncategorized') as category,
			       coalesce(st.code, '') as stock_code,
			       coalesce(nullif(st.name, ''), nullif(st.code, ''), 'No Stock') as stock_name,
			       coalesce(to_char(l.movement_date, 'MM/DD/YYYY'), '') as entry_date,
			       coalesce(l.sort_key, '') as sort_key,
			       coalesce(l.reference, '') as reference,
			       coalesce(l.company, '') as company,
			       coalesce(l.kind, '') as kind,
			       coalesce(l.qty_delta, 0)::bigint as qty_delta
			from stocks st
			left join ledger l on l.stock_id = st.id
		)
		select coalesce(sr.stock_id, 'category:' || c.category) as stock_id,
		       c.category,
		       coalesce(sr.stock_code, '') as stock_code,
		       coalesce(sr.stock_name, 'No Stock') as stock_name,
		       coalesce(sr.entry_date, '') as entry_date,
		       coalesce(sr.sort_key, '') as sort_key,
		       coalesce(sr.reference, '') as reference,
		       coalesce(sr.company, '') as company,
		       coalesce(sr.kind, '') as kind,
		       coalesce(sr.qty_delta, 0)::bigint as qty_delta
		from categories c
		left join stock_rows sr on sr.category = c.category
		order by c.category, stock_code, stock_name, entry_date, sort_key`, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var reportRows []models.StockLedgerReportRow
	for rows.Next() {
		var row models.StockLedgerReportRow
		if err := rows.Scan(&row.StockID, &row.Category, &row.StockCode, &row.StockName, &row.EntryDate, &row.SortKey, &row.Reference, &row.Company, &row.Kind, &row.QtyDelta); err != nil {
			return nil, err
		}
		reportRows = append(reportRows, row)
	}
	return reportRows, rows.Err()
}

func (s *PostgresStore) StockAgingReportRows(ctx context.Context, cutoff time.Time) ([]models.StockAgingReportRow, error) {
	rows, err := s.pool.Query(ctx, `
		with ledger as (
			select sl.stock_id,
			       coalesce(nullif(st.category_group, ''), 'Uncategorized') as category,
			       coalesce(st.code, '') as stock_code,
			       coalesce(nullif(st.name, ''), nullif(st.code, ''), 'No Stock') as stock_name,
			       coalesce(sl.qty_delta, 0) as qty_delta,
			       case
			       	when d.kind = 'purchases' then d.document_date
			       	when d.kind = 'sales' then d.document_date
			       	when d.kind = 'stock-transactions' then d.document_date
			       	else d.entry_date::date
			       end as movement_date
			from stock_ledger sl
			join documents d on d.id = sl.document_id
			left join stocks st on st.id = sl.stock_id
			where case
			       	when d.kind = 'purchases' then d.document_date
			       	when d.kind = 'sales' then d.document_date
			       	when d.kind = 'stock-transactions' then d.document_date
			       	else d.entry_date::date
			      end <= $1::date
		),
		stock_out as (
			select stock_id, greatest(-coalesce(sum(case when qty_delta < 0 then qty_delta else 0 end), 0), 0) as out_qty
			from ledger
			group by stock_id
		),
		positive_lots as (
			select l.stock_id,
			       l.category,
			       l.stock_code,
			       l.stock_name,
			       l.movement_date,
			       l.qty_delta as lot_qty,
			       coalesce(sum(l.qty_delta) over (
			       	partition by l.stock_id
			       	order by l.movement_date, l.stock_code, l.stock_name
			       	rows between unbounded preceding and 1 preceding
			       ), 0) as prev_in_qty,
			       sum(l.qty_delta) over (
			       	partition by l.stock_id
			       	order by l.movement_date, l.stock_code, l.stock_name
			       	rows unbounded preceding
			       ) as cumulative_in_qty,
			       coalesce(o.out_qty, 0) as out_qty
			from ledger l
			left join stock_out o on o.stock_id = l.stock_id
			where l.qty_delta > 0
		),
		remaining_lots as (
			select category,
			       stock_code,
			       stock_name,
			       movement_date,
			       greatest(cumulative_in_qty - out_qty, 0) - greatest(prev_in_qty - out_qty, 0) as remaining_qty
			from positive_lots
		)
		select category,
		       stock_code,
		       stock_name,
		       coalesce(round(sum(case when movement_date >= ($1::date - interval '30 days') then remaining_qty else 0 end)), 0)::bigint as bucket0,
		       coalesce(round(sum(case when movement_date >= ($1::date - interval '60 days') and movement_date < ($1::date - interval '30 days') then remaining_qty else 0 end)), 0)::bigint as bucket1,
		       coalesce(round(sum(case when movement_date >= ($1::date - interval '90 days') and movement_date < ($1::date - interval '60 days') then remaining_qty else 0 end)), 0)::bigint as bucket2,
		       coalesce(round(sum(case when movement_date >= ($1::date - interval '120 days') and movement_date < ($1::date - interval '90 days') then remaining_qty else 0 end)), 0)::bigint as bucket3,
		       coalesce(round(sum(case when movement_date >= ($1::date - interval '150 days') and movement_date < ($1::date - interval '120 days') then remaining_qty else 0 end)), 0)::bigint as bucket4
		from remaining_lots
		group by category, stock_code, stock_name
		having coalesce(sum(case when movement_date >= ($1::date - interval '150 days') then remaining_qty else 0 end), 0) <> 0
		order by category, stock_code, stock_name`, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var reportRows []models.StockAgingReportRow
	for rows.Next() {
		var row models.StockAgingReportRow
		if err := rows.Scan(&row.Category, &row.StockCode, &row.StockName, &row.Bucket0, &row.Bucket1, &row.Bucket2, &row.Bucket3, &row.Bucket4); err != nil {
			return nil, err
		}
		reportRows = append(reportRows, row)
	}
	return reportRows, rows.Err()
}

func (s *PostgresStore) StockReorderPointReportRows(ctx context.Context, cutoff time.Time) ([]models.StockReorderPointReportRow, error) {
	rows, err := s.pool.Query(ctx, `
		with balances as (
			select sl.stock_id,
			       coalesce(round(sum(sl.qty_delta)), 0)::bigint as soh
			from stock_ledger sl
			join documents d on d.id = sl.document_id
			where case
			       	when d.kind = 'purchases' then d.document_date
			       	when d.kind = 'sales' then d.document_date
			       	when d.kind = 'stock-transactions' then d.document_date
			       	else d.entry_date::date
			      end <= $1::date
			group by sl.stock_id
		)
		select coalesce(nullif(st.category_group, ''), 'Uncategorized') as category,
		       coalesce(st.code, '') as stock_code,
		       coalesce(nullif(st.name, ''), nullif(st.code, ''), 'No Stock') as stock_name,
		       coalesce(b.soh, 0)::bigint as soh,
		       coalesce(round(st.min_inventory), 0)::bigint as min_inventory,
		       greatest(coalesce(round(st.min_inventory), 0)::bigint - coalesce(b.soh, 0)::bigint, 0)::bigint as deficit
		from stocks st
		left join balances b on b.stock_id = st.id
		where greatest(coalesce(round(st.min_inventory), 0)::bigint - coalesce(b.soh, 0)::bigint, 0)::bigint > 0
		order by category, stock_code, stock_name`, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var reportRows []models.StockReorderPointReportRow
	for rows.Next() {
		var row models.StockReorderPointReportRow
		if err := rows.Scan(&row.Category, &row.StockCode, &row.StockName, &row.SOH, &row.MinInventory, &row.Deficit); err != nil {
			return nil, err
		}
		reportRows = append(reportRows, row)
	}
	return reportRows, rows.Err()
}

func (s *PostgresStore) StockSummaryReportRows(ctx context.Context, cutoff time.Time) ([]models.StockSummaryReportRow, error) {
	rows, err := s.pool.Query(ctx, `
		with balances as (
			select sl.stock_id,
			       coalesce(round(sum(sl.qty_delta)), 0)::bigint as soh
			from stock_ledger sl
			join documents d on d.id = sl.document_id
			where case
			       	when d.kind = 'purchases' then d.document_date
			       	when d.kind = 'sales' then d.document_date
			       	when d.kind = 'stock-transactions' then d.document_date
			       	else d.entry_date::date
			      end <= $1::date
			group by sl.stock_id
		),
		categories as (
			select distinct trim(name) as category
			from stock_categories
			where coalesce(trim(name), '') <> ''
			union
			select distinct coalesce(nullif(st.category_group, ''), 'Uncategorized') as category
			from stocks st
		),
		stock_rows as (
			select coalesce(nullif(st.category_group, ''), 'Uncategorized') as category,
			       coalesce(st.code, '') as stock_code,
			       coalesce(nullif(st.name, ''), nullif(st.code, ''), 'No Stock') as stock_name,
			       true as has_stock,
			       coalesce(b.soh, 0)::bigint as soh,
			       coalesce(round(st.latest_cost * 100), 0)::bigint as unit_cost_cents,
			       coalesce(round(coalesce(b.soh, 0) * st.latest_cost * 100), 0)::bigint as amount_cents
			from stocks st
			left join balances b on b.stock_id = st.id
		)
		select c.category,
		       coalesce(sr.stock_code, '') as stock_code,
		       coalesce(sr.stock_name, 'No Stock') as stock_name,
		       coalesce(sr.has_stock, false) as has_stock,
		       coalesce(sr.soh, 0)::bigint as soh,
		       coalesce(sr.unit_cost_cents, 0)::bigint as unit_cost_cents,
		       coalesce(sr.amount_cents, 0)::bigint as amount_cents
		from categories c
		left join stock_rows sr on sr.category = c.category
		order by c.category, stock_code, stock_name`, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var reportRows []models.StockSummaryReportRow
	for rows.Next() {
		var row models.StockSummaryReportRow
		if err := rows.Scan(&row.Category, &row.StockCode, &row.StockName, &row.HasStock, &row.SOH, &row.UnitCostCents, &row.AmountCents); err != nil {
			return nil, err
		}
		reportRows = append(reportRows, row)
	}
	return reportRows, rows.Err()
}

func (s *PostgresStore) GetDocument(ctx context.Context, form models.FormDefinition, id int64) (models.Record, map[string][]models.Record, error) {
	var kind string
	var entryID string
	var entryDate string
	var branchID string
	var partyID string
	var reference string
	var remarks string
	var cash bool
	var drReferenceID string
	var encoder string
	var updatedBy string
	var payloadBytes []byte
	err := s.pool.QueryRow(ctx, `
		select kind,
		       entry_id,
		       entry_date::date::text,
		       coalesce(branch_id::text, ''),
		       coalesce(party_id::text, ''),
		       coalesce(reference, ''),
		       coalesce(remarks, ''),
		       cash,
		       coalesce(dr_reference_id::text, ''),
		       coalesce(u.display_name, ''),
		       coalesce(uu.display_name, ''),
		       payload
		from documents
		left join users u on u.id = documents.encoder_user_id
		left join users uu on uu.id = documents.last_update_by_user_id
		where documents.id=$1`, id).Scan(&kind, &entryID, &entryDate, &branchID, &partyID, &reference, &remarks, &cash, &drReferenceID, &encoder, &updatedBy, &payloadBytes)
	if err != nil {
		return nil, nil, err
	}
	if kind != form.Kind {
		return nil, nil, errors.New("document kind mismatch")
	}

	values := models.Record{}
	values["record_id"] = strconv.FormatInt(id, 10)
	payload := storedDocumentPayload{}
	if len(payloadBytes) != 0 {
		if err := json.Unmarshal(payloadBytes, &payload); err != nil {
			return nil, nil, err
		}
		for key, value := range payload.Values {
			values[key] = value
		}
	}

	if values["entry_date"] == "" {
		values["entry_date"] = entryDate
	}
	if values["id"] == "" {
		values["id"] = entryID
	}
	if values["encoder"] == "" {
		values["encoder"] = encoder
	}
	if values["updated_by"] == "" {
		values["updated_by"] = updatedBy
	}
	if values["branch_id"] == "" {
		values["branch_id"] = branchID
	}
	if values["party_id"] == "" {
		values["party_id"] = partyID
	}
	if values["remarks"] == "" {
		values["remarks"] = remarks
	}
	if cash {
		values["cash"] = "true"
	} else if values["cash"] == "" {
		values["cash"] = "false"
	}
	if drReferenceID != "" {
		values["dr_document_id"] = drReferenceID
		var drLabel string
		err := s.pool.QueryRow(ctx, `
			select coalesce(nullif(d.reference, ''), d.entry_id, 'SO') || ' - ' || coalesce(nullif(c.company,''), c.code, '')
			from documents d
			left join customers c on c.id = d.party_id
			where d.id=$1 and d.kind='dr'`, drReferenceID).Scan(&drLabel)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, err
		}
		if drLabel != "" {
			values["dr_document_label"] = drLabel
		}
	}
	if values["reference"] == "" && reference != "" {
		values["reference"] = reference
	}

	lineRows := lineRowsFromInput(payload.LineInput)
	if len(lineRows) == 0 {
		lineRows, err = s.documentLineRows(ctx, id)
		if err != nil {
			return nil, nil, err
		}
	}
	return values, lineRows, nil
}

func (s *PostgresStore) documentLineRows(ctx context.Context, documentID int64) (map[string][]models.Record, error) {
	rows, err := s.pool.Query(ctx, `
		select group_key, payload
		from document_lines
		where document_id=$1
		order by group_key, line_no`, documentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rowsByGroup := map[string][]models.Record{}
	for rows.Next() {
		var group string
		var payloadBytes []byte
		if err := rows.Scan(&group, &payloadBytes); err != nil {
			return nil, err
		}
		record := models.Record{}
		if len(payloadBytes) != 0 {
			if err := json.Unmarshal(payloadBytes, &record); err != nil {
				return nil, err
			}
		}
		rowsByGroup[group] = append(rowsByGroup[group], record)
	}
	return rowsByGroup, rows.Err()
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
		       coalesce(st.category_group, ''),
		       ((dl.qty - coalesce((
		         select sum(dc.consumed_qty)
		         from dr_consumptions dc
		         where dc.dr_line_id = dl.id
		       ), 0)))::bigint::text,
		       coalesce(latest_purchase.unit_cost::text, st.latest_cost::text, '0')
		from document_lines dl
		left join stocks st on st.id = dl.stock_id
		left join lateral (
		  select pdl.unit_cost
		  from document_lines pdl
		  join documents pd on pd.id = pdl.document_id
		  where pd.kind = 'purchases'
		    and pdl.stock_id = dl.stock_id
		  order by pd.document_date desc, pd.entry_date desc, pd.id desc, pdl.line_no desc
		  limit 1
		) latest_purchase on true
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
		var stockGroup string
		var qty string
		var latestCost string
		if err := rows.Scan(&drLineID, &stockID, &stockLabel, &stockGroup, &qty, &latestCost); err != nil {
			return DRSelection{}, err
		}
		row["dr_line_id"] = drLineID
		row["stock_id"] = stockID
		row["stock_label"] = stockLabel
		row["stock_group"] = stockGroup
		row["qty"] = qty
		row["unit_cost"] = latestCost
		row["capital"] = latestCost
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

func (s *PostgresStore) SaveDocument(ctx context.Context, form models.FormDefinition, id int64, input DocumentInput) (int64, error) {
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
	documentDate := effectiveDocumentDate(form.Kind, input.Values, *entryDate)
	branchID := parseInt(input.Values["branch_id"])
	if branchID == 0 {
		branchID = input.User.ActiveBranchID
	}
	input.Values["branch_id"] = strconv.FormatInt(branchID, 10)
	partyID := parseInt(input.Values["party_id"])
	payload, err := json.Marshal(storedDocumentPayload{
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
	drReferenceID := parseInt(input.Values["dr_document_id"])
	var drState drValidationState
	var latestCostStockIDs []int64
	if id != 0 {
		if form.Kind == "purchases" {
			latestCostStockIDs, err = s.purchaseDocumentStockIDs(ctx, tx, id)
			if err != nil {
				return 0, err
			}
		}
		if err := s.prepareDocumentUpdate(ctx, tx, form.Kind, id); err != nil {
			return 0, err
		}
	}
	if drReferenceID != 0 {
		drState, err = s.validateDRReference(ctx, tx, form.Kind, drReferenceID, input.Values, input.LineInput)
		if err != nil {
			return 0, err
		}
	}
	if id == 0 {
		err = tx.QueryRow(ctx, `
				insert into documents
					(kind, entry_date, document_date, branch_id, party_type, party_id, reference, cash, remarks, total, less_amount, add_amount, net, balance, payload, dr_reference_id, encoder_user_id, last_update_by_user_id)
				values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$17)
				returning id`,
			form.Kind, entryDate, documentDate, nullInt(branchID), emptyToNil(form.PartyType), nullInt(partyID), emptyToNil(input.Values["reference"]),
			parseBool(input.Values["cash"]), input.Values["remarks"], centsToNumeric(totalInput.total), centsToNumeric(totalInput.less),
			centsToNumeric(totalInput.add), centsToNumeric(totalInput.net), centsToNumeric(totalInput.balance), payload, nullInt(drReferenceID), input.User.ID,
		).Scan(&totalInput.documentID)
		if err != nil {
			return 0, err
		}
	} else {
		totalInput.documentID = id
		_, err = tx.Exec(ctx, `
				update documents
				set entry_date=$2,
				    document_date=$3,
				    branch_id=$4,
				    party_type=$5,
				    party_id=$6,
				    reference=$7,
				    cash=$8,
				    remarks=$9,
				    total=$10,
				    less_amount=$11,
				    add_amount=$12,
				    net=$13,
				    balance=$14,
				    payload=$15,
				    dr_reference_id=$16,
				    last_update_by_user_id=$17,
				    updated_at=now()
				where id=$1`,
			id, entryDate, documentDate, nullInt(branchID), emptyToNil(form.PartyType), nullInt(partyID), emptyToNil(input.Values["reference"]),
			parseBool(input.Values["cash"]), input.Values["remarks"], centsToNumeric(totalInput.total), centsToNumeric(totalInput.less),
			centsToNumeric(totalInput.add), centsToNumeric(totalInput.net), centsToNumeric(totalInput.balance), payload, nullInt(drReferenceID), input.User.ID,
		)
		if err != nil {
			return 0, err
		}
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
	if form.Kind == "purchases" {
		latestCostStockIDs = appendStockLineIDs(latestCostStockIDs, totalInput.posting.Lines)
		if err := s.refreshLatestPurchaseCosts(ctx, tx, latestCostStockIDs); err != nil {
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
	return totalInput.documentID, nil
}

func (s *PostgresStore) DeleteDocument(ctx context.Context, form models.FormDefinition, id int64, user models.User) error {
	if !user.CanWrite() {
		return errors.New("write access required")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var latestCostStockIDs []int64
	if form.Kind == "purchases" {
		latestCostStockIDs, err = s.purchaseDocumentStockIDs(ctx, tx, id)
		if err != nil {
			return err
		}
	}
	if err := s.prepareDocumentUpdate(ctx, tx, form.Kind, id); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `delete from documents where id=$1 and kind=$2`, id, form.Kind); err != nil {
		return err
	}
	if form.Kind == "purchases" {
		if err := s.refreshLatestPurchaseCosts(ctx, tx, latestCostStockIDs); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (s *PostgresStore) purchaseDocumentStockIDs(ctx context.Context, tx pgx.Tx, documentID int64) ([]int64, error) {
	rows, err := tx.Query(ctx, `
		select distinct dl.stock_id
		from document_lines dl
		join documents d on d.id = dl.document_id
		where d.id = $1
		  and d.kind = 'purchases'
		  and dl.stock_id is not null`, documentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stockIDs []int64
	for rows.Next() {
		var stockID int64
		if err := rows.Scan(&stockID); err != nil {
			return nil, err
		}
		stockIDs = appendUniqueInt64(stockIDs, stockID)
	}
	return stockIDs, rows.Err()
}

func (s *PostgresStore) refreshLatestPurchaseCosts(ctx context.Context, tx pgx.Tx, stockIDs []int64) error {
	stockIDs = uniquePositiveInt64s(stockIDs)
	if len(stockIDs) == 0 {
		return nil
	}
	_, err := tx.Exec(ctx, `
		update stocks st
		set latest_cost = coalesce((
		    select dl.unit_cost
		    from document_lines dl
		    join documents d on d.id = dl.document_id
		    where d.kind = 'purchases'
		      and dl.stock_id = st.id
		    order by d.document_date desc, d.entry_date desc, d.id desc, dl.line_no desc
		    limit 1
		  ), 0),
		  updated_at = now()
		where st.id = any($1::bigint[])`, stockIDs)
	return err
}

func (s *PostgresStore) prepareDocumentUpdate(ctx context.Context, tx pgx.Tx, kind string, id int64) error {
	var existingKind string
	if err := tx.QueryRow(ctx, `select kind from documents where id=$1`, id).Scan(&existingKind); err != nil {
		return err
	}
	if existingKind != kind {
		return errors.New("document kind mismatch")
	}
	type balanceAdjustment struct {
		partyType string
		partyID   int64
		amount    string
	}
	var adjustments []balanceAdjustment
	rows, err := tx.Query(ctx, `select party_type, party_id, amount_delta from balance_ledger where document_id=$1`, id)
	if err != nil {
		return err
	}
	for rows.Next() {
		var adjustment balanceAdjustment
		if err := rows.Scan(&adjustment.partyType, &adjustment.partyID, &adjustment.amount); err != nil {
			rows.Close()
			return err
		}
		adjustments = append(adjustments, adjustment)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()

	for _, adjustment := range adjustments {
		table := "customers"
		if adjustment.partyType == string(services.PartySupplier) {
			table = "suppliers"
		}
		if _, err := tx.Exec(ctx, fmt.Sprintf(`update %s set balance = coalesce(balance, 0) - $1 where id=$2`, table), adjustment.amount, adjustment.partyID); err != nil {
			return err
		}
	}

	if _, err := tx.Exec(ctx, `delete from dr_consumptions where consumer_document_id=$1`, id); err != nil {
		return err
	}
	if kind == "dr" {
		if _, err := tx.Exec(ctx, `
			update document_lines consumer_line
			set dr_line_id = null
			from document_lines dr_line
			where consumer_line.dr_line_id = dr_line.id
			  and dr_line.document_id = $1`, id); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `delete from dr_consumptions where dr_document_id=$1`, id); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(ctx, `delete from stock_ledger where document_id=$1`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `delete from balance_ledger where document_id=$1`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `delete from document_lines where document_id=$1`, id); err != nil {
		return err
	}
	return nil
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
	case "stock_groups":
		return `select id::text, coalesce(nullif(category_group,''), 'Ungrouped') from stocks order by code`
	case "dr_documents":
		return `
			select d.id::text,
			       coalesce(nullif(d.reference, ''), d.entry_id, 'SO') || ' - ' || coalesce(nullif(c.company,''), c.code, '')
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
		return `
			select value, value
			from (
				select distinct group_name as value
				from stock_categories
				where coalesce(group_name, '') <> ''
				union
				select distinct name as value
				from stock_categories
				where coalesce(name, '') <> ''
			) options
			order by value`
	case "stock_categories":
		return `
			select distinct name, name
			from stock_categories
			where coalesce(name, '') <> ''
			order by name`
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

func valueForMasterField(form models.FormDefinition, field models.Field, values map[string]string) any {
	value := values[field.Key]
	if form.Kind == "customers" && field.Key == "credit_limit" && strings.TrimSpace(value) == "" {
		value = "0"
	}
	return valueForField(field, value)
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

func effectiveDocumentDate(kind string, values map[string]string, entryDate time.Time) time.Time {
	var key string
	switch kind {
	case "purchases":
		key = "purchase_date"
	case "sales", "dr":
		key = "sales_date"
	case "stock-transactions":
		key = "transfer_date"
	}
	if key != "" {
		if date := nullableDate(values[key]); date != nil {
			return *date
		}
	}
	return entryDate
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

func appendStockLineIDs(stockIDs []int64, lines []services.StockLine) []int64 {
	for _, line := range lines {
		stockIDs = appendUniqueInt64(stockIDs, line.StockID)
	}
	return stockIDs
}

func appendUniqueInt64(values []int64, value int64) []int64 {
	if value <= 0 {
		return values
	}
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func uniquePositiveInt64s(values []int64) []int64 {
	unique := make([]int64, 0, len(values))
	for _, value := range values {
		unique = appendUniqueInt64(unique, value)
	}
	return unique
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

func lineRowsFromInput(inputs []LineInput) map[string][]models.Record {
	rowsByGroup := make(map[string][]models.Record, len(inputs))
	for _, group := range inputs {
		groupRows := make([]models.Record, 0, len(group.Rows))
		for _, row := range group.Rows {
			record := models.Record{}
			for key, value := range row {
				record[key] = value
			}
			groupRows = append(groupRows, record)
		}
		rowsByGroup[group.Group] = groupRows
	}
	return rowsByGroup
}

func rowIsBlank(row map[string]string) bool {
	for _, value := range row {
		if strings.TrimSpace(value) != "" {
			return false
		}
	}
	return true
}
