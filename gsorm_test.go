package gsorm

import (
	"database/sql"
	"sync"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// resetSingleton resets the singleton for testing
func resetSingleton() {
	gsormInstance = nil
	gsormOnce = sync.Once{}
}

func setupTestDB(t *testing.T) *sql.DB {
	// Reset singleton for each test
	resetSingleton()
	
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Create test table
	_, err = db.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			email TEXT UNIQUE NOT NULL,
			age INTEGER,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}

	// Insert test data
	_, err = db.Exec(`
		INSERT INTO users (name, email, age) VALUES 
		('John Doe', 'john@example.com', 25),
		('Jane Smith', 'jane@example.com', 30),
		('Bob Johnson', 'bob@example.com', 35),
		('Alice Brown', 'alice@example.com', 28)
	`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	// Initialize GSORM with the test database
	Set(db)
	
	return db
}

func TestSet(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	builder := Set(db)
	if builder == nil {
		t.Fatal("Set() returned nil")
	}

	if builder.db != db {
		t.Error("Builder database not set correctly")
	}
}

func TestDB(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	Set(db)
	builder := DB()
	if builder == nil {
		t.Fatal("DB() returned nil")
	}
}

func TestDBPanic(t *testing.T) {
	// Reset singleton for test
	gsormInstance = nil
	defer func() {
		if r := recover(); r == nil {
			t.Error("DB() should panic when not initialized")
		}
	}()
	DB()
}

func TestTable(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	builder := DB().Table("users")
	if builder.table != "users" {
		t.Errorf("Expected table 'users', got '%s'", builder.table)
	}
}

func TestSelect(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	builder := DB().Select("name", "email")
	expected := []string{"name", "email"}

	if len(builder.selectCols) != len(expected) {
		t.Errorf("Expected %d columns, got %d", len(expected), len(builder.selectCols))
	}

	for i, col := range expected {
		if builder.selectCols[i] != col {
			t.Errorf("Expected column '%s', got '%s'", col, builder.selectCols[i])
		}
	}
}

func TestWhere(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	builder := DB().Where("age", ">", 25)

	if len(builder.whereConds) != 1 {
		t.Errorf("Expected 1 where condition, got %d", len(builder.whereConds))
	}

	cond := builder.whereConds[0]
	if cond.Column != "age" || cond.Operator != ">" || cond.Value != 25 || cond.Logic != "AND" {
		t.Errorf("Where condition not set correctly: %+v", cond)
	}
}

func TestOrWhere(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	Set(db)
	builder := DB().Where("age", ">", 25).OrWhere("name", "=", "John")

	if len(builder.whereConds) != 2 {
		t.Errorf("Expected 2 where conditions, got %d", len(builder.whereConds))
	}

	orCond := builder.whereConds[1]
	if orCond.Logic != "OR" {
		t.Errorf("Expected OR logic, got %s", orCond.Logic)
	}
}

func TestWhereIn(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	values := []interface{}{25, 30, 35}
	Set(db)
	builder := DB().WhereIn("age", values)

	if len(builder.whereConds) != 1 {
		t.Errorf("Expected 1 where condition, got %d", len(builder.whereConds))
	}

	cond := builder.whereConds[0]
	if cond.Column != "age" || cond.Operator != "IN (?,?,?)" {
		t.Errorf("WhereIn condition not set correctly: %+v", cond)
	}
}

func TestWhereNull(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	Set(db)
	builder := DB().WhereNull("deleted_at")

	if len(builder.whereConds) != 1 {
		t.Errorf("Expected 1 where condition, got %d", len(builder.whereConds))
	}

	cond := builder.whereConds[0]
	if cond.Column != "deleted_at" || cond.Operator != "IS NULL" {
		t.Errorf("WhereNull condition not set correctly: %+v", cond)
	}
}

func TestWhereNotNull(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	Set(db)
	builder := DB().WhereNotNull("created_at")

	if len(builder.whereConds) != 1 {
		t.Errorf("Expected 1 where condition, got %d", len(builder.whereConds))
	}

	cond := builder.whereConds[0]
	if cond.Column != "created_at" || cond.Operator != "IS NOT NULL" {
		t.Errorf("WhereNotNull condition not set correctly: %+v", cond)
	}
}

func TestJoins(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	Set(db)
	builder := DB().
		LeftJoin("profiles", "users.id = profiles.user_id").
		RightJoin("orders", "users.id = orders.user_id").
		InnerJoin("categories", "orders.category_id = categories.id")

	if len(builder.joins) != 3 {
		t.Errorf("Expected 3 joins, got %d", len(builder.joins))
	}

	expectedTypes := []string{"LEFT", "RIGHT", "INNER"}
	for i, expectedType := range expectedTypes {
		if builder.joins[i].Type != expectedType {
			t.Errorf("Expected join type '%s', got '%s'", expectedType, builder.joins[i].Type)
		}
	}
}

func TestOrderBy(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	Set(db)
	builder := DB().OrderBy("name", "ASC").OrderBy("age", "DESC")

	if len(builder.orderBy) != 2 {
		t.Errorf("Expected 2 order conditions, got %d", len(builder.orderBy))
	}

	if builder.orderBy[0].Column != "name" || builder.orderBy[0].Dir != "ASC" {
		t.Errorf("First order condition not set correctly: %+v", builder.orderBy[0])
	}

	if builder.orderBy[1].Column != "age" || builder.orderBy[1].Dir != "DESC" {
		t.Errorf("Second order condition not set correctly: %+v", builder.orderBy[1])
	}
}

func TestOrderByInvalidDirection(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	Set(db)
	builder := DB().OrderBy("name", "INVALID")

	if builder.orderBy[0].Dir != "ASC" {
		t.Errorf("Expected default direction 'ASC', got '%s'", builder.orderBy[0].Dir)
	}
}

func TestGroupBy(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	Set(db)

	builder := DB().GroupBy("age", "name")

	if len(builder.groupBy) != 2 {
		t.Errorf("Expected 2 group by columns, got %d", len(builder.groupBy))
	}

	if builder.groupBy[0] != "age" || builder.groupBy[1] != "name" {
		t.Errorf("GroupBy columns not set correctly: %v", builder.groupBy)
	}
}

func TestHaving(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	Set(db)

	builder := DB().Having("COUNT(*)", ">", 1)

	if len(builder.having) != 1 {
		t.Errorf("Expected 1 having condition, got %d", len(builder.having))
	}

	cond := builder.having[0]
	if cond.Column != "COUNT(*)" || cond.Operator != ">" || cond.Value != 1 {
		t.Errorf("Having condition not set correctly: %+v", cond)
	}
}

func TestLimit(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	builder := Set(db).Limit(10)

	if builder.limitVal != 10 {
		t.Errorf("Expected limit 10, got %d", builder.limitVal)
	}
}

func TestOffset(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	builder := Set(db).Offset(5)

	if builder.offsetVal != 5 {
		t.Errorf("Expected offset 5, got %d", builder.offsetVal)
	}
}

func TestPaginate(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	builder := Set(db).Paginate(2, 10)

	if builder.limitVal != 10 {
		t.Errorf("Expected limit 10, got %d", builder.limitVal)
	}

	if builder.offsetVal != 10 {
		t.Errorf("Expected offset 10, got %d", builder.offsetVal)
	}
}

func TestPaginateInvalidValues(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	builder := Set(db).Paginate(0, -5)

	if builder.limitVal != 10 {
		t.Errorf("Expected default limit 10, got %d", builder.limitVal)
	}

	if builder.offsetVal != 0 {
		t.Errorf("Expected offset 0, got %d", builder.offsetVal)
	}
}

func TestBuildSelectQuery(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	builder := Set(db).
		Table("users").
		Select("name", "email").
		Where("age", ">", 25).
		OrderBy("name", "ASC").
		Limit(10)

	query, args := builder.buildSelectQuery()

	expectedQuery := "SELECT name, email FROM users WHERE age > ? ORDER BY name ASC LIMIT ?"
	if query != expectedQuery {
		t.Errorf("Expected query:\n%s\nGot:\n%s", expectedQuery, query)
	}

	if len(args) != 2 {
		t.Errorf("Expected 2 args, got %d", len(args))
	}

	if args[0] != 25 || args[1] != 10 {
		t.Errorf("Args not correct: %v", args)
	}
}

func TestGet(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	rows, err := Set(db).Table("users").Where("age", ">", 25).Get()
	if err != nil {
		t.Fatalf("Get() failed: %v", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		count++
	}

	if count != 3 {
		t.Errorf("Expected 3 rows, got %d", count)
	}
}

func TestFirst(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	row, err := DB().Table("users").Where("name", "=", "John Doe").First()
	if err != nil {
		t.Fatalf("First() failed: %v", err)
	}

	var id, age int
	var name, email, createdAt string
	err = row.Scan(&id, &name, &email, &age, &createdAt)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	if name != "John Doe" || email != "john@example.com" || age != 25 {
		t.Errorf("Expected John Doe data, got: %s, %s, %d", name, email, age)
	}
}

func TestCount(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	count, err := DB().Table("users").Count()
	if err != nil {
		t.Fatalf("Count() failed: %v", err)
	}

	if count != 4 {
		t.Errorf("Expected count 4, got %d", count)
	}

	count, err = DB().Table("users").Where("age", ">=", 30).Count()
	if err != nil {
		t.Fatalf("Count() with where failed: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected filtered count 2, got %d", count)
	}
}

func TestInsert(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	data := map[string]interface{}{
		"name":  "Test User",
		"email": "test@example.com",
		"age":   22,
	}

	result, err := Set(db).Table("users").Insert(data)
	if err != nil {
		t.Fatalf("Insert() failed: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		t.Fatalf("RowsAffected() failed: %v", err)
	}

	if rowsAffected != 1 {
		t.Errorf("Expected 1 row affected, got %d", rowsAffected)
	}
}

func TestInsertBulk(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	data := []map[string]interface{}{
		{"name": "User1", "email": "user1@example.com", "age": 20},
		{"name": "User2", "email": "user2@example.com", "age": 21},
		{"name": "User3", "email": "user3@example.com", "age": 22},
	}

	err := Set(db).Table("users").InsertBulk(data)
	if err != nil {
		t.Fatalf("InsertBulk() failed: %v", err)
	}

	count, err := Set(db).Table("users").Count()
	if err != nil {
		t.Fatalf("Count() after bulk insert failed: %v", err)
	}

	if count != 7 {
		t.Errorf("Expected count 7 after bulk insert, got %d", count)
	}
}

func TestInsertBulkEmpty(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	err := Set(db).Table("users").InsertBulk([]map[string]interface{}{})
	if err != nil {
		t.Errorf("InsertBulk() with empty data should not fail: %v", err)
	}
}

func TestUpdate(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	data := map[string]interface{}{
		"age": 26,
	}

	result, err := Set(db).Table("users").Where("name", "=", "John Doe").Update(data)
	if err != nil {
		t.Fatalf("Update() failed: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		t.Fatalf("RowsAffected() failed: %v", err)
	}

	if rowsAffected != 1 {
		t.Errorf("Expected 1 row affected, got %d", rowsAffected)
	}
}

func TestDelete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// First verify initial count
	initialCount, err := DB().Table("users").Count()
	if err != nil {
		t.Fatalf("Initial count failed: %v", err)
	}

	result, err := DB().Table("users").Where("name", "=", "Bob Johnson").Delete()
	if err != nil {
		t.Fatalf("Delete() failed: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		t.Fatalf("RowsAffected() failed: %v", err)
	}

	if rowsAffected != 1 {
		t.Errorf("Expected 1 row affected, got %d", rowsAffected)
	}

	count, err := DB().Table("users").Count()
	if err != nil {
		t.Fatalf("Count() after delete failed: %v", err)
	}

	expected := initialCount - 1
	if count != expected {
		t.Errorf("Expected count %d after delete, got %d", expected, count)
	}
}

func TestSum(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	sum, err := Set(db).Table("users").Sum("age")
	if err != nil {
		t.Fatalf("Sum() failed: %v", err)
	}

	expectedSum := float64(25 + 30 + 35 + 28)
	if sum != expectedSum {
		t.Errorf("Expected sum %f, got %f", expectedSum, sum)
	}
}

func TestAvg(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	avg, err := Set(db).Table("users").Avg("age")
	if err != nil {
		t.Fatalf("Avg() failed: %v", err)
	}

	expectedAvg := float64(25+30+35+28) / 4
	if avg != expectedAvg {
		t.Errorf("Expected avg %f, got %f", expectedAvg, avg)
	}
}

func TestMax(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	max, err := Set(db).Table("users").Max("age")
	if err != nil {
		t.Fatalf("Max() failed: %v", err)
	}

	if max != int64(35) {
		t.Errorf("Expected max 35, got %v", max)
	}
}

func TestMin(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	min, err := Set(db).Table("users").Min("age")
	if err != nil {
		t.Fatalf("Min() failed: %v", err)
	}

	if min != int64(25) {
		t.Errorf("Expected min 25, got %v", min)
	}
}

func TestToArray(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	results, err := Set(db).Table("users").Select("name", "age").Where("age", ">", 25).ToArray()
	if err != nil {
		t.Fatalf("ToArray() failed: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}

	for _, result := range results {
		if result["name"] == nil || result["age"] == nil {
			t.Errorf("Missing expected columns in result: %+v", result)
		}
	}
}

func TestClone(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	original := Set(db).Table("users").Where("age", ">", 25)
	clone := original.Clone()

	if clone == original {
		t.Error("Clone should return a different instance")
	}

	if clone.table != original.table {
		t.Error("Clone should have same table")
	}

	if len(clone.whereConds) != len(original.whereConds) {
		t.Error("Clone should have same where conditions")
	}
}

func TestTransaction(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	builder := Set(db)

	err := builder.WithTransaction(func(b *Builder) error {
		data := map[string]interface{}{
			"name":  "TX User",
			"email": "tx@example.com",
			"age":   25,
		}
		_, err := b.Table("users").Insert(data)
		return err
	})

	if err != nil {
		t.Fatalf("Transaction failed: %v", err)
	}

	count, err := Set(db).Table("users").Count()
	if err != nil {
		t.Fatalf("Count() after transaction failed: %v", err)
	}

	if count != 5 {
		t.Errorf("Expected count 5 after transaction, got %d", count)
	}
}

func TestTransactionRollback(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	builder := Set(db)

	err := builder.WithTransaction(func(b *Builder) error {
		data := map[string]interface{}{
			"name":  "TX User",
			"email": "duplicate@example.com",
			"age":   25,
		}
		_, err := b.Table("users").Insert(data)
		if err != nil {
			return err
		}

		// This should cause rollback due to unique constraint
		_, err = b.Table("users").Insert(data)
		return err
	})

	if err == nil {
		t.Error("Transaction should have failed due to unique constraint")
	}

	Set(db)

	count, err := DB().Table("users").Count()
	if err != nil {
		t.Fatalf("Count() after failed transaction: %v", err)
	}

	if count != 4 {
		t.Errorf("Expected count 4 after rollback, got %d", count)
	}
}

func TestPrintSQL(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	Set(db)

	builder := DB().Table("users").Where("age", ">", 25).OrderBy("name", "ASC")
	sql := builder.PrintSQL()

	expectedSQL := "SELECT * FROM users WHERE age > 25 ORDER BY name ASC"
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}
}
