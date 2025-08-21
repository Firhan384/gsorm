package gsorm

import (
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Builder is the main ORM Builder structure
type Builder struct {
	db         *sql.DB
	table      string
	selectCols []string
	whereConds []WhereCondition
	joins      []JoinCondition
	orderBy    []OrderCondition
	groupBy    []string
	having     []WhereCondition
	limitVal   int
	offsetVal  int
	args       []interface{}
	tx         *sql.Tx
}

// WhereCondition stores safe WHERE conditions
type WhereCondition struct {
	Column   string
	Operator string
	Value    interface{}
	Logic    string // AND, OR
}

// JoinCondition stores JOIN conditions
type JoinCondition struct {
	Type      string // LEFT, RIGHT, INNER
	Table     string
	Condition string
}

// OrderCondition stores ORDER BY conditions
type OrderCondition struct {
	Column string
	Dir    string // ASC, DESC
}

var gsormOnce sync.Once
var gsormInstance *Builder

// Pool for string builders to reduce allocations
var stringBuilderPool = sync.Pool{
	New: func() interface{} {
		return &strings.Builder{}
	},
}

// getStringBuilder gets a string builder from pool
func getStringBuilder() *strings.Builder {
	sb := stringBuilderPool.Get().(*strings.Builder)
	sb.Reset()
	return sb
}

// putStringBuilder returns a string builder to pool
func putStringBuilder(sb *strings.Builder) {
	stringBuilderPool.Put(sb)
}

// Set initializes singleton instance (called only once)
func Set(db *sql.DB) *Builder {
	gsormOnce.Do(func() {
		gsormInstance = &Builder{
			db:         db,
			selectCols: []string{"*"},
			args:       make([]interface{}, 0),
		}
	})

	return gsormInstance
}

// DB returns the initialized singleton instance
func DB() *Builder {
	if gsormInstance == nil {
		panic("GSORM not initialized. Call Set() first.")
	}
	// Return clone to avoid state sharing
	return gsormInstance.Clone()
}

// Table sets the target table
func (b *Builder) Table(table string) *Builder {
	b.table = table
	return b
}

// Select sets the columns to be selected
func (b *Builder) Select(cols ...string) *Builder {
	b.selectCols = cols
	return b
}

// Where adds WHERE condition with prepared statements
func (b *Builder) Where(column string, operator string, value interface{}) *Builder {
	b.whereConds = append(b.whereConds, WhereCondition{
		Column:   column,
		Operator: operator,
		Value:    value,
		Logic:    "AND",
	})
	return b
}

// OrWhere adds WHERE condition with OR logic
func (b *Builder) OrWhere(column string, operator string, value interface{}) *Builder {
	b.whereConds = append(b.whereConds, WhereCondition{
		Column:   column,
		Operator: operator,
		Value:    value,
		Logic:    "OR",
	})
	return b
}

// WhereIn adds safe WHERE IN condition
func (b *Builder) WhereIn(column string, values []interface{}) *Builder {
	if len(values) > 0 {
		placeholders := make([]string, len(values))
		for i := range values {
			placeholders[i] = "?"
		}

		b.whereConds = append(b.whereConds, WhereCondition{
			Column:   column,
			Operator: "IN (" + strings.Join(placeholders, ",") + ")",
			Value:    values,
			Logic:    "AND",
		})
	}
	return b
}

// WhereNotNull adds WHERE column IS NOT NULL condition
func (b *Builder) WhereNotNull(column string) *Builder {
	b.whereConds = append(b.whereConds, WhereCondition{
		Column:   column,
		Operator: "IS NOT NULL",
		Value:    nil,
		Logic:    "AND",
	})
	return b
}

// WhereNull adds WHERE column IS NULL condition
func (b *Builder) WhereNull(column string) *Builder {
	b.whereConds = append(b.whereConds, WhereCondition{
		Column:   column,
		Operator: "IS NULL",
		Value:    nil,
		Logic:    "AND",
	})
	return b
}

// LeftJoin adds LEFT JOIN
func (b *Builder) LeftJoin(table, condition string) *Builder {
	b.joins = append(b.joins, JoinCondition{
		Type:      "LEFT",
		Table:     table,
		Condition: condition,
	})
	return b
}

// RightJoin adds RIGHT JOIN
func (b *Builder) RightJoin(table, condition string) *Builder {
	b.joins = append(b.joins, JoinCondition{
		Type:      "RIGHT",
		Table:     table,
		Condition: condition,
	})
	return b
}

// InnerJoin adds INNER JOIN
func (b *Builder) InnerJoin(table, condition string) *Builder {
	b.joins = append(b.joins, JoinCondition{
		Type:      "INNER",
		Table:     table,
		Condition: condition,
	})
	return b
}

// OrderBy adds ORDER BY clause
func (b *Builder) OrderBy(column, direction string) *Builder {
	// Validate direction to prevent injection
	dir := strings.ToUpper(direction)
	if dir != "ASC" && dir != "DESC" {
		dir = "ASC"
	}

	b.orderBy = append(b.orderBy, OrderCondition{
		Column: column,
		Dir:    dir,
	})
	return b
}

// GroupBy adds GROUP BY clause
func (b *Builder) GroupBy(columns ...string) *Builder {
	b.groupBy = append(b.groupBy, columns...)
	return b
}

// Having adds HAVING condition
func (b *Builder) Having(column string, operator string, value interface{}) *Builder {
	b.having = append(b.having, WhereCondition{
		Column:   column,
		Operator: operator,
		Value:    value,
		Logic:    "AND",
	})
	return b
}

// Limit sets the LIMIT clause
func (b *Builder) Limit(limit int) *Builder {
	b.limitVal = limit
	return b
}

// Offset sets the OFFSET clause for pagination
func (b *Builder) Offset(offset int) *Builder {
	b.offsetVal = offset
	return b
}

// Paginate sets up pagination
func (b *Builder) Paginate(page, perPage int) *Builder {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 10
	}

	b.limitVal = perPage
	b.offsetVal = (page - 1) * perPage
	return b
}

// buildSelectQuery builds safe SELECT query
func (b *Builder) buildSelectQuery() (string, []interface{}) {
	query := getStringBuilder()
	defer putStringBuilder(query)
	
	args := make([]interface{}, 0, 4) // Pre-allocate for common case
	
	// SELECT clause
	query.WriteString("SELECT ")
	query.WriteString(strings.Join(b.selectCols, ", "))
	query.WriteString(" FROM ")
	query.WriteString(b.table)

	// JOIN clauses
	for _, join := range b.joins {
		query.WriteString(" ")
		query.WriteString(join.Type)
		query.WriteString(" JOIN ")
		query.WriteString(join.Table)
		query.WriteString(" ON ")
		query.WriteString(join.Condition)
	}

	// WHERE clauses
	if len(b.whereConds) > 0 {
		query.WriteString(" WHERE ")
		whereClause, whereArgs := b.buildWhereClause(b.whereConds)
		query.WriteString(whereClause)
		args = append(args, whereArgs...)
	}

	// GROUP BY
	if len(b.groupBy) > 0 {
		query.WriteString(" GROUP BY ")
		query.WriteString(strings.Join(b.groupBy, ", "))
	}

	// HAVING
	if len(b.having) > 0 {
		query.WriteString(" HAVING ")
		havingClause, havingArgs := b.buildWhereClause(b.having)
		query.WriteString(havingClause)
		args = append(args, havingArgs...)
	}

	// ORDER BY
	if len(b.orderBy) > 0 {
		query.WriteString(" ORDER BY ")
		for i, order := range b.orderBy {
			if i > 0 {
				query.WriteString(", ")
			}
			query.WriteString(order.Column)
			query.WriteString(" ")
			query.WriteString(order.Dir)
		}
	}

	// LIMIT and OFFSET
	if b.limitVal > 0 {
		query.WriteString(" LIMIT ?")
		args = append(args, b.limitVal)
	}

	if b.offsetVal > 0 {
		query.WriteString(" OFFSET ?")
		args = append(args, b.offsetVal)
	}

	return query.String(), args
}

// buildWhereClause builds safe WHERE clause
func (b *Builder) buildWhereClause(conditions []WhereCondition) (string, []interface{}) {
	if len(conditions) == 0 {
		return "", nil
	}

	clause := getStringBuilder()
	defer putStringBuilder(clause)
	
	args := make([]interface{}, 0, len(conditions))

	for i, cond := range conditions {
		if i > 0 {
			clause.WriteString(" ")
			clause.WriteString(cond.Logic)
			clause.WriteString(" ")
		}

		clause.WriteString(cond.Column)
		clause.WriteString(" ")
		clause.WriteString(cond.Operator)

		if cond.Operator == "IS NULL" || cond.Operator == "IS NOT NULL" {
			// No value needed
		} else if strings.Contains(cond.Operator, "IN") {
			if values, ok := cond.Value.([]interface{}); ok {
				args = append(args, values...)
			}
		} else {
			clause.WriteString(" ?")
			args = append(args, cond.Value)
		}
	}

	return clause.String(), args
}

// Get retrieves all records
func (b *Builder) Get() (*sql.Rows, error) {
	query, args := b.buildSelectQuery()

	if b.tx != nil {
		return b.tx.Query(query, args...)
	}
	return b.db.Query(query, args...)
}

// First retrieves the first record
func (b *Builder) First() (*sql.Row, error) {
	b.limitVal = 1
	query, args := b.buildSelectQuery()

	if b.tx != nil {
		return b.tx.QueryRow(query, args...), nil
	}
	return b.db.QueryRow(query, args...), nil
}

// Count counts the number of records
func (b *Builder) Count() (int64, error) {
	originalCols := b.selectCols
	b.selectCols = []string{"COUNT(*) as count"}

	query, args := b.buildSelectQuery()
	b.selectCols = originalCols

	var count int64
	var row *sql.Row

	if b.tx != nil {
		row = b.tx.QueryRow(query, args...)
	} else {
		row = b.db.QueryRow(query, args...)
	}

	err := row.Scan(&count)
	return count, err
}

// Insert performs INSERT with prepared statement
func (b *Builder) Insert(data map[string]interface{}) (sql.Result, error) {
	columns := make([]string, 0, len(data))
	placeholders := make([]string, 0, len(data))
	values := make([]interface{}, 0, len(data))

	for col, val := range data {
		columns = append(columns, col)
		placeholders = append(placeholders, "?")
		values = append(values, val)
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		b.table,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "))

	if b.tx != nil {
		return b.tx.Exec(query, values...)
	}
	return b.db.Exec(query, values...)
}

// InsertBulk performs efficient bulk insert
func (b *Builder) InsertBulk(data []map[string]interface{}) error {
	if len(data) == 0 {
		return nil
	}

	// Get columns from first data row
	firstRow := data[0]
	numCols := len(firstRow)
	columns := make([]string, 0, numCols)
	for col := range firstRow {
		columns = append(columns, col)
	}

	// Pre-allocate with exact capacity
	numRows := len(data)
	allValues := make([]interface{}, 0, numRows*numCols)
	
	// Use string builder from pool
	query := getStringBuilder()
	defer putStringBuilder(query)
	
	// Build query efficiently
	query.WriteString("INSERT INTO ")
	query.WriteString(b.table)
	query.WriteString(" (")
	query.WriteString(strings.Join(columns, ", "))
	query.WriteString(") VALUES ")
	
	// Build VALUES clause
	for i, row := range data {
		if i > 0 {
			query.WriteString(", ")
		}
		query.WriteString("(")
		
		// Add placeholders and values
		for j, col := range columns {
			if j > 0 {
				query.WriteString(", ")
			}
			query.WriteString("?")
			allValues = append(allValues, row[col])
		}
		query.WriteString(")")
	}

	queryStr := query.String()

	if b.tx != nil {
		_, err := b.tx.Exec(queryStr, allValues...)
		return err
	}
	_, err := b.db.Exec(queryStr, allValues...)
	return err
}

// Update performs UPDATE with WHERE conditions
func (b *Builder) Update(data map[string]interface{}) (sql.Result, error) {
	setClauses := make([]string, 0, len(data))
	args := make([]interface{}, 0, len(data))

	for col, val := range data {
		setClauses = append(setClauses, col+" = ?")
		args = append(args, val)
	}

	query := "UPDATE " + b.table + " SET " + strings.Join(setClauses, ", ")

	if len(b.whereConds) > 0 {
		whereClause, whereArgs := b.buildWhereClause(b.whereConds)
		query += " WHERE " + whereClause
		args = append(args, whereArgs...)
	}

	if b.tx != nil {
		return b.tx.Exec(query, args...)
	}
	return b.db.Exec(query, args...)
}

// UpdateBulk performs efficient bulk update
func (b *Builder) UpdateBulk(updates []map[string]interface{}, keyColumn string) error {
	if len(updates) == 0 {
		return nil
	}

	// CASE WHEN implementation for bulk update
	columns := make(map[string]bool)
	for _, update := range updates {
		for col := range update {
			if col != keyColumn {
				columns[col] = true
			}
		}
	}

	setClauses := make([]string, 0)
	args := make([]interface{}, 0)
	keyValues := make([]interface{}, len(updates))

	for col := range columns {
		caseClause := col + " = CASE " + keyColumn
		for _, update := range updates {
			caseClause += " WHEN ? THEN ?"
			args = append(args, update[keyColumn], update[col])
		}
		caseClause += " ELSE " + col + " END"
		setClauses = append(setClauses, caseClause)
	}

	for i, update := range updates {
		keyValues[i] = update[keyColumn]
	}

	// Create IN clause for WHERE
	inPlaceholders := make([]string, len(keyValues))
	for i := range keyValues {
		inPlaceholders[i] = "?"
	}

	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s IN (%s)",
		b.table,
		strings.Join(setClauses, ", "),
		keyColumn,
		strings.Join(inPlaceholders, ", "))

	args = append(args, keyValues...)

	if b.tx != nil {
		_, err := b.tx.Exec(query, args...)
		return err
	}
	_, err := b.db.Exec(query, args...)
	return err
}

// Delete performs DELETE with WHERE conditions
func (b *Builder) Delete() (sql.Result, error) {
	query := "DELETE FROM " + b.table
	args := make([]interface{}, 0)

	if len(b.whereConds) > 0 {
		whereClause, whereArgs := b.buildWhereClause(b.whereConds)
		query += " WHERE " + whereClause
		args = append(args, whereArgs...)
	}

	if b.tx != nil {
		return b.tx.Exec(query, args...)
	}
	return b.db.Exec(query, args...)
}

// Transaction methods
func (b *Builder) BeginTransaction() error {
	tx, err := b.db.Begin()
	if err != nil {
		return err
	}
	b.tx = tx
	return nil
}

func (b *Builder) CommitTransaction() error {
	if b.tx == nil {
		return fmt.Errorf("no active transaction")
	}
	err := b.tx.Commit()
	b.tx = nil
	return err
}

func (b *Builder) RollbackTransaction() error {
	if b.tx == nil {
		return fmt.Errorf("no active transaction")
	}
	err := b.tx.Rollback()
	b.tx = nil
	return err
}

// WithTransaction runs operations within transaction context
func (b *Builder) WithTransaction(fn func(*Builder) error) error {
	if err := b.BeginTransaction(); err != nil {
		return err
	}

	if err := fn(b); err != nil {
		b.RollbackTransaction()
		return err
	}

	return b.CommitTransaction()
}

// CreateOrUpdate performs UPSERT operation
func (b *Builder) CreateOrUpdate(data map[string]interface{}, conflictColumns []string) (sql.Result, error) {
	// MySQL implementation using ON DUPLICATE KEY UPDATE
	columns := make([]string, 0, len(data))
	placeholders := make([]string, 0, len(data))
	values := make([]interface{}, 0, len(data))
	updateClauses := make([]string, 0)

	for col, val := range data {
		columns = append(columns, col)
		placeholders = append(placeholders, "?")
		values = append(values, val)

		// Skip conflict columns in update clause
		isConflictCol := false
		for _, conflictCol := range conflictColumns {
			if col == conflictCol {
				isConflictCol = true
				break
			}
		}
		if !isConflictCol {
			updateClauses = append(updateClauses, col+" = VALUES("+col+")")
		}
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s) ON DUPLICATE KEY UPDATE %s",
		b.table,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
		strings.Join(updateClauses, ", "))

	if b.tx != nil {
		return b.tx.Exec(query, values...)
	}
	return b.db.Exec(query, values...)
}

// PrintSQL for debugging - displays the SQL to be executed
func (b *Builder) PrintSQL() string {
	query, args := b.buildSelectQuery()

	// Replace placeholders with values for debugging
	for i, arg := range args {
		placeholder := "?"
		var value string

		switch v := arg.(type) {
		case string:
			value = "'" + strings.Replace(v, "'", "''", -1) + "'"
		case time.Time:
			value = "'" + v.Format("2006-01-02 15:04:05") + "'"
		case nil:
			value = "NULL"
		default:
			value = fmt.Sprintf("%v", v)
		}

		if i == 0 {
			query = strings.Replace(query, placeholder, value, 1)
		} else {
			query = strings.Replace(query, placeholder, value, 1)
		}
	}

	return query
}

// Aggregate functions
func (b *Builder) Sum(column string) (float64, error) {
	b.selectCols = []string{"SUM(" + column + ") as sum"}
	query, args := b.buildSelectQuery()

	var sum sql.NullFloat64
	var row *sql.Row

	if b.tx != nil {
		row = b.tx.QueryRow(query, args...)
	} else {
		row = b.db.QueryRow(query, args...)
	}

	err := row.Scan(&sum)
	if err != nil {
		return 0, err
	}

	if sum.Valid {
		return sum.Float64, nil
	}
	return 0, nil
}

func (b *Builder) Max(column string) (interface{}, error) {
	b.selectCols = []string{"MAX(" + column + ") as max"}
	query, args := b.buildSelectQuery()

	var max interface{}
	var row *sql.Row

	if b.tx != nil {
		row = b.tx.QueryRow(query, args...)
	} else {
		row = b.db.QueryRow(query, args...)
	}

	err := row.Scan(&max)
	return max, err
}

func (b *Builder) Min(column string) (interface{}, error) {
	b.selectCols = []string{"MIN(" + column + ") as min"}
	query, args := b.buildSelectQuery()

	var min interface{}
	var row *sql.Row

	if b.tx != nil {
		row = b.tx.QueryRow(query, args...)
	} else {
		row = b.db.QueryRow(query, args...)
	}

	err := row.Scan(&min)
	return min, err
}

func (b *Builder) Avg(column string) (float64, error) {
	b.selectCols = []string{"AVG(" + column + ") as avg"}
	query, args := b.buildSelectQuery()

	var avg sql.NullFloat64
	var row *sql.Row

	if b.tx != nil {
		row = b.tx.QueryRow(query, args...)
	} else {
		row = b.db.QueryRow(query, args...)
	}

	err := row.Scan(&avg)
	if err != nil {
		return 0, err
	}

	if avg.Valid {
		return avg.Float64, nil
	}
	return 0, nil
}

// ToArray converts query results to slice of maps
func (b *Builder) ToArray() ([]map[string]interface{}, error) {
	rows, err := b.Get()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var results []map[string]interface{}

	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))

		for i := range columns {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			row[col] = values[i]
		}

		results = append(results, row)
	}

	return results, nil
}

// Clone creates a copy of builder for reuse
func (b *Builder) Clone() *Builder {
	clone := &Builder{
		db:        b.db,
		table:     b.table,
		limitVal:  b.limitVal,
		offsetVal: b.offsetVal,
		tx:        b.tx,
	}

	// Only allocate slices if they have content
	if len(b.selectCols) > 0 {
		clone.selectCols = make([]string, len(b.selectCols))
		copy(clone.selectCols, b.selectCols)
	} else {
		clone.selectCols = []string{"*"}
	}

	if len(b.whereConds) > 0 {
		clone.whereConds = make([]WhereCondition, len(b.whereConds))
		copy(clone.whereConds, b.whereConds)
	}

	if len(b.joins) > 0 {
		clone.joins = make([]JoinCondition, len(b.joins))
		copy(clone.joins, b.joins)
	}

	if len(b.orderBy) > 0 {
		clone.orderBy = make([]OrderCondition, len(b.orderBy))
		copy(clone.orderBy, b.orderBy)
	}

	if len(b.groupBy) > 0 {
		clone.groupBy = make([]string, len(b.groupBy))
		copy(clone.groupBy, b.groupBy)
	}

	if len(b.having) > 0 {
		clone.having = make([]WhereCondition, len(b.having))
		copy(clone.having, b.having)
	}

	if len(b.args) > 0 {
		clone.args = make([]interface{}, len(b.args))
		copy(clone.args, b.args)
	} else {
		clone.args = make([]interface{}, 0)
	}

	return clone
}
