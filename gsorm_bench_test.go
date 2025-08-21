package gsorm

import (
	"database/sql"
	"fmt"
	"math/rand"
	"sync"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func setupBenchDB(b *testing.B) *sql.DB {
	// Reset singleton for each benchmark
	gsormInstance = nil
	gsormOnce = sync.Once{}

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		b.Fatalf("Failed to open bench database: %v", err)
	}

	// Initialize GSORM first
	Set(db)

	// Create test table with indexes for better performance testing
	_, err = db.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			email TEXT UNIQUE NOT NULL,
			age INTEGER,
			salary REAL,
			department_id INTEGER,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX idx_users_age ON users(age);
		CREATE INDEX idx_users_department ON users(department_id);
		CREATE INDEX idx_users_name ON users(name);
	`)
	if err != nil {
		b.Fatalf("Failed to create bench table: %v", err)
	}

	// Create a larger dataset for meaningful benchmarks
	data := make([]map[string]interface{}, 1000)
	departments := []int{1, 2, 3, 4, 5}
	names := []string{"John", "Jane", "Bob", "Alice", "Charlie", "Diana", "Eve", "Frank"}

	for i := 0; i < 1000; i++ {
		data[i] = map[string]interface{}{
			"name":          fmt.Sprintf("%s_%d", names[i%len(names)], i),
			"email":         fmt.Sprintf("user%d@example.com", i),
			"age":           20 + (i % 50),
			"salary":        30000.0 + float64(i*100),
			"department_id": departments[i%len(departments)],
		}
	}

	err = DB().Table("users").InsertBulk(data)
	if err != nil {
		b.Fatalf("Failed to insert bench data: %v", err)
	}

	return db
}

func BenchmarkSetAndDB(b *testing.B) {
	db := setupBenchDB(b)
	defer db.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Set(db)
		DB()
	}
}

func BenchmarkQueryBuilder(b *testing.B) {
	db := setupBenchDB(b)
	defer db.Close()
	Set(db)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = DB().Table("users").
			Select("name", "email", "age").
			Where("age", ">", 25).
			Where("department_id", "=", 1).
			OrderBy("name", "ASC").
			Limit(10)
	}
}

func BenchmarkBuildSelectQuery(b *testing.B) {
	db := setupBenchDB(b)
	defer db.Close()
	Set(db)

	builder := DB().Table("users").
		Select("name", "email", "age").
		Where("age", ">", 25).
		Where("department_id", "=", 1).
		OrderBy("name", "ASC").
		Limit(10)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = builder.buildSelectQuery()
	}
}

func BenchmarkSimpleSelect(b *testing.B) {
	db := setupBenchDB(b)
	defer db.Close()
	Set(db)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rows, err := DB().Table("users").Get()
		if err != nil {
			b.Fatal(err)
		}
		rows.Close()
	}
}

func BenchmarkSelectWithWhere(b *testing.B) {
	db := setupBenchDB(b)
	defer db.Close()
	Set(db)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rows, err := DB().Table("users").Where("age", ">", 30).Get()
		if err != nil {
			b.Fatal(err)
		}
		rows.Close()
	}
}

func BenchmarkSelectWithMultipleWhere(b *testing.B) {
	db := setupBenchDB(b)
	defer db.Close()
	Set(db)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rows, err := DB().Table("users").
			Where("age", ">", 25).
			Where("department_id", "=", 1).
			OrWhere("salary", ">", 50000).
			Get()
		if err != nil {
			b.Fatal(err)
		}
		rows.Close()
	}
}

func BenchmarkSelectWithJoin(b *testing.B) {
	db := setupBenchDB(b)
	defer db.Close()
	Set(db)

	// Create departments table for join
	_, err := db.Exec(`
		CREATE TABLE departments (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL
		);
		INSERT INTO departments (id, name) VALUES 
		(1, 'Engineering'), (2, 'Marketing'), (3, 'Sales'), (4, 'HR'), (5, 'Finance');
	`)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rows, err := DB().Table("users").
			Select("users.name", "users.email", "departments.name as dept_name").
			LeftJoin("departments", "users.department_id = departments.id").
			Where("users.age", ">", 30).
			Get()
		if err != nil {
			b.Fatal(err)
		}
		rows.Close()
	}
}

func BenchmarkSelectWithOrderBy(b *testing.B) {
	db := setupBenchDB(b)
	defer db.Close()
	Set(db)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rows, err := DB().Table("users").
			OrderBy("name", "ASC").
			OrderBy("age", "DESC").
			Get()
		if err != nil {
			b.Fatal(err)
		}
		rows.Close()
	}
}

func BenchmarkSelectWithLimitOffset(b *testing.B) {
	db := setupBenchDB(b)
	defer db.Close()
	Set(db)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rows, err := DB().Table("users").
			OrderBy("id", "ASC").
			Limit(20).
			Offset(i % 100).
			Get()
		if err != nil {
			b.Fatal(err)
		}
		rows.Close()
	}
}

func BenchmarkSelectWithPagination(b *testing.B) {
	db := setupBenchDB(b)
	defer db.Close()
	Set(db)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		page := (i % 50) + 1
		rows, err := DB().Table("users").
			OrderBy("id", "ASC").
			Paginate(page, 20).
			Get()
		if err != nil {
			b.Fatal(err)
		}
		rows.Close()
	}
}

func BenchmarkCount(b *testing.B) {
	db := setupBenchDB(b)
	defer db.Close()
	Set(db)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := DB().Table("users").Count()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCountWithWhere(b *testing.B) {
	db := setupBenchDB(b)
	defer db.Close()
	Set(db)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := DB().Table("users").Where("age", ">", 30).Count()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFirst(b *testing.B) {
	db := setupBenchDB(b)
	defer db.Close()
	Set(db)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := DB().Table("users").Where("id", "=", (i%1000)+1).First()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkInsert(b *testing.B) {
	db := setupBenchDB(b)
	defer db.Close()
	Set(db)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data := map[string]interface{}{
			"name":          fmt.Sprintf("BenchUser_%d", i),
			"email":         fmt.Sprintf("bench%d@example.com", i),
			"age":           25 + (i % 40),
			"salary":        40000.0 + float64(i*50),
			"department_id": (i % 5) + 1,
		}
		_, err := DB().Table("users").Insert(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkInsertBulk(b *testing.B) {
	db := setupBenchDB(b)
	defer db.Close()
	Set(db)

	batchSize := 100
	data := make([]map[string]interface{}, batchSize)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < batchSize; j++ {
			idx := i*batchSize + j
			data[j] = map[string]interface{}{
				"name":          fmt.Sprintf("BulkUser_%d", idx),
				"email":         fmt.Sprintf("bulk%d@example.com", idx),
				"age":           25 + (idx % 40),
				"salary":        40000.0 + float64(idx*50),
				"department_id": (idx % 5) + 1,
			}
		}
		err := DB().Table("users").InsertBulk(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUpdate(b *testing.B) {
	db := setupBenchDB(b)
	defer db.Close()
	Set(db)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data := map[string]interface{}{
			"salary": 45000.0 + float64(i*10),
		}
		_, err := DB().Table("users").Where("id", "=", (i%1000)+1).Update(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUpdateBulk(b *testing.B) {
	db := setupBenchDB(b)
	defer db.Close()
	Set(db)

	batchSize := 50
	updates := make([]map[string]interface{}, batchSize)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < batchSize; j++ {
			idx := (i*batchSize + j) % 1000
			updates[j] = map[string]interface{}{
				"id":     idx + 1,
				"salary": 50000.0 + float64(idx*20),
			}
		}
		err := DB().Table("users").UpdateBulk(updates, "id")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDelete(b *testing.B) {
	db := setupBenchDB(b)
	defer db.Close()
	Set(db)

	// Add extra users for deletion
	data := make([]map[string]interface{}, b.N)
	for i := 0; i < b.N; i++ {
		data[i] = map[string]interface{}{
			"name":          fmt.Sprintf("DeleteUser_%d", i),
			"email":         fmt.Sprintf("delete%d@example.com", i),
			"age":           25,
			"salary":        30000.0,
			"department_id": 1,
		}
	}
	err := DB().Table("users").InsertBulk(data)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := DB().Table("users").Where("email", "=", fmt.Sprintf("delete%d@example.com", i)).Delete()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAggregateSum(b *testing.B) {
	db := setupBenchDB(b)
	defer db.Close()
	Set(db)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := DB().Table("users").Sum("salary")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAggregateAvg(b *testing.B) {
	db := setupBenchDB(b)
	defer db.Close()
	Set(db)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := DB().Table("users").Avg("age")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAggregateMax(b *testing.B) {
	db := setupBenchDB(b)
	defer db.Close()
	Set(db)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := DB().Table("users").Max("salary")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAggregateMin(b *testing.B) {
	db := setupBenchDB(b)
	defer db.Close()
	Set(db)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := DB().Table("users").Min("age")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkToArray(b *testing.B) {
	db := setupBenchDB(b)
	defer db.Close()
	Set(db)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := DB().Table("users").Select("name", "email", "age").Limit(100).ToArray()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkClone(b *testing.B) {
	db := setupBenchDB(b)
	defer db.Close()
	Set(db)

	original := DB().Table("users").
		Select("name", "email").
		Where("age", ">", 25).
		Where("department_id", "=", 1).
		OrderBy("name", "ASC")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = original.Clone()
	}
}

func BenchmarkTransaction(b *testing.B) {
	db := setupBenchDB(b)
	defer db.Close()
	Set(db)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := DB().WithTransaction(func(tx *Builder) error {
			data := map[string]interface{}{
				"name":          fmt.Sprintf("TxUser_%d", i),
				"email":         fmt.Sprintf("tx%d@example.com", i),
				"age":           30,
				"salary":        45000.0,
				"department_id": 1,
			}
			_, err := tx.Table("users").Insert(data)
			return err
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkWhereIn(b *testing.B) {
	db := setupBenchDB(b)
	defer db.Close()
	Set(db)

	ids := make([]interface{}, 20)
	for i := 0; i < 20; i++ {
		ids[i] = rand.Intn(1000) + 1
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rows, err := DB().Table("users").WhereIn("id", ids).Get()
		if err != nil {
			b.Fatal(err)
		}
		rows.Close()
	}
}

func BenchmarkComplexQuery(b *testing.B) {
	db := setupBenchDB(b)
	defer db.Close()
	Set(db)

	// Create departments table for complex join
	_, err := db.Exec(`
		CREATE TABLE departments (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL
		);
		INSERT INTO departments (id, name) VALUES 
		(1, 'Engineering'), (2, 'Marketing'), (3, 'Sales'), (4, 'HR'), (5, 'Finance');
	`)
	if err != nil {
		b.Fatal(err)
	}

	ageValues := []interface{}{25, 30, 35, 40}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rows, err := DB().Table("users").
			Select("users.name", "users.email", "users.salary", "departments.name as dept").
			LeftJoin("departments", "users.department_id = departments.id").
			Where("users.salary", ">", 40000).
			WhereIn("users.age", ageValues).
			OrWhere("departments.name", "=", "Engineering").
			GroupBy("departments.id").
			Having("COUNT(*)", ">", 5).
			OrderBy("users.salary", "DESC").
			OrderBy("users.name", "ASC").
			Limit(50).
			Get()
		if err != nil {
			b.Fatal(err)
		}
		rows.Close()
	}
}

func BenchmarkCreateOrUpdate(b *testing.B) {
	db := setupBenchDB(b)
	defer db.Close()
	Set(db)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data := map[string]interface{}{
			"email":         fmt.Sprintf("upsert%d@example.com", i%100), // Reuse emails for upsert
			"name":          fmt.Sprintf("UpsertUser_%d", i),
			"age":           25 + (i % 30),
			"salary":        35000.0 + float64(i*100),
			"department_id": (i % 5) + 1,
		}
		_, err := DB().Table("users").CreateOrUpdate(data, []string{"email"})
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPrintSQL(b *testing.B) {
	db := setupBenchDB(b)
	defer db.Close()
	Set(db)

	builder := DB().Table("users").
		Select("name", "email", "age").
		Where("age", ">", 25).
		Where("department_id", "=", 1).
		OrderBy("name", "ASC").
		Limit(10)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = builder.PrintSQL()
	}
}

func BenchmarkConcurrentQueries(b *testing.B) {
	db := setupBenchDB(b)
	defer db.Close()
	Set(db)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			rows, err := DB().Table("users").Where("age", ">", 30).Limit(10).Get()
			if err != nil {
				b.Fatal(err)
			}
			rows.Close()
		}
	})
}

func BenchmarkMemoryUsage(b *testing.B) {
	db := setupBenchDB(b)
	defer db.Close()
	Set(db)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		results, err := DB().Table("users").Select("name", "email").Limit(1000).ToArray()
		if err != nil {
			b.Fatal(err)
		}
		_ = results // Use results to prevent optimization
	}
}
