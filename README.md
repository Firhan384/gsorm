# GSORM - High-Performance Go SQL Query Builder

[![Go Report Card](https://goreportcard.com/badge/github.com/Firhan384/gsorm)](https://goreportcard.com/report/github.com/Firhan384/gsorm)
[![GoDoc](https://godoc.org/github.com/Firhan384/gsorm?status.svg)](https://godoc.org/github.com/Firhan384/gsorm)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## Overview

GSORM is a Go library specifically designed to provide maximum security against SQL injection while maintaining high performance. This library implements the singleton pattern for efficient database connection management, focusing on:

- **🔒 100% SQL Injection Protection** - Uses prepared statements for all operations
- **🏗️ Singleton Pattern** - One database connection instance for the entire application
- **🔄 Thread Safe** - Safe for concurrent access with sync.Once
- **⚡ High Performance** - Optimized for high performance with object pooling
- **📊 Comprehensive Testing** - 33 unit tests + extensive benchmarks
- **🧬 Clone Support** - Each query uses an isolated instance

## 🚀 Performance Highlights

- **Sub-microsecond query building** (~287ns)
- **47% faster** query generation vs baseline
- **48% less memory usage** in core operations
- **44% fewer allocations** with object pooling
- **Excellent scalability** across operation complexity

## Installation

```bash
go get github.com/Firhan384/gsorm
```

## Quick Start

### Basic Setup

```go
package main

import (
    "database/sql"
    "fmt"
    "log"
    _ "github.com/mattn/go-sqlite3" // or your preferred driver
    "github.com/Firhan384/gsorm"
)

func main() {
    // Database connection
    db, err := sql.Open("sqlite3", "./database.db")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    // Initialize singleton (call once at application start)
    gsorm.Set(db)

    // Use DB() to get fresh instance anywhere in your app
    users, err := gsorm.DB().Table("users").
        Where("status", "=", "active").
        ToArray()
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Active users: %+v\n", users)
}
```

### Web Application Example

```go
// main.go
func main() {
    db := setupDatabase()
    gsorm.Set(db) // Initialize once
    
    router := setupRoutes()
    router.Run(":8080")
}

// handlers/user.go
func GetUsers(c *gin.Context) {
    // Direct usage without re-initialization
    users, err := gsorm.DB().Table("users").
        Where("active", "=", 1).
        OrderBy("created_at", "DESC").
        Paginate(1, 20).
        ToArray()
    
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    c.JSON(200, users)
}
```

## 📚 API Reference

### 🔍 Query Operations

#### Select Data

```go
// Simple select
users, err := gsorm.DB().Table("users").
    Select("id", "name", "email").
    ToArray()

// Complex conditions
results, err := gsorm.DB().Table("users").
    Where("age", ">=", 18).
    Where("status", "=", "active").
    OrWhere("role", "=", "admin").
    WhereIn("department", []interface{}{"IT", "Marketing"}).
    WhereNotNull("email_verified_at").
    OrderBy("created_at", "DESC").
    Limit(20).
    ToArray()

// First record
user, err := gsorm.DB().Table("users").
    Where("email", "=", "john@example.com").
    First()

// Count records
count, err := gsorm.DB().Table("users").
    Where("status", "=", "active").
    Count()
```

#### Join Operations

```go
// INNER JOIN
userPosts, err := gsorm.DB().Table("users").
    Select("users.name", "posts.title", "posts.created_at").
    InnerJoin("posts", "posts.user_id = users.id").
    Where("users.active", "=", 1).
    ToArray()

// LEFT JOIN with multiple tables
orders, err := gsorm.DB().Table("orders").
    Select("orders.*", "users.name", "products.title").
    LeftJoin("users", "users.id = orders.user_id").
    LeftJoin("order_items", "order_items.order_id = orders.id").
    LeftJoin("products", "products.id = order_items.product_id").
    Where("orders.status", "=", "completed").
    ToArray()
```

### ✏️ Insert Operations

```go
// Single insert
userData := map[string]interface{}{
    "name":       "John Doe",
    "email":      "john@example.com",
    "status":     "active",
    "created_at": time.Now(),
}

result, err := gsorm.DB().Table("users").Insert(userData)
if err != nil {
    log.Fatal(err)
}

lastID, _ := result.LastInsertId()

// High-performance bulk insert
bulkData := []map[string]interface{}{
    {"name": "Alice Johnson", "email": "alice@example.com"},
    {"name": "Bob Smith", "email": "bob@example.com"},
    {"name": "Carol Davis", "email": "carol@example.com"},
}

// Inserts all records in a single optimized query
err = gsorm.DB().Table("users").InsertBulk(bulkData)
```

### 🔄 Update Operations

```go
// Single update
result, err := gsorm.DB().Table("users").
    Where("email", "=", "john@example.com").
    Update(map[string]interface{}{
        "status":     "verified",
        "updated_at": time.Now(),
    })

// Bulk update (high performance)
bulkUpdates := []map[string]interface{}{
    {"id": 1, "status": "premium", "updated_at": time.Now()},
    {"id": 2, "status": "verified", "updated_at": time.Now()},
    {"id": 3, "status": "suspended", "updated_at": time.Now()},
}

err = gsorm.DB().Table("users").UpdateBulk(bulkUpdates, "id")
```

### 🗑️ Delete Operations

```go
// Safe delete with WHERE conditions
result, err := gsorm.DB().Table("users").
    Where("status", "=", "inactive").
    Where("last_login", "<", "2023-01-01").
    Delete()

rowsDeleted, _ := result.RowsAffected()
fmt.Printf("Deleted %d rows\n", rowsDeleted)
```

### 📊 Aggregate Functions

```go
// Count active users
activeCount, err := gsorm.DB().Table("users").
    Where("status", "=", "active").
    Count()

// Sum order amounts
totalSales, err := gsorm.DB().Table("orders").
    Where("status", "=", "completed").
    Sum("amount")

// Average age
avgAge, err := gsorm.DB().Table("users").Avg("age")

// Min/Max values
oldestUser, err := gsorm.DB().Table("users").Min("created_at")
newestUser, err := gsorm.DB().Table("users").Max("created_at")

// GROUP BY with HAVING
topCustomers, err := gsorm.DB().Table("orders").
    Select("user_id", "COUNT(*) as order_count", "SUM(amount) as total").
    Where("status", "=", "completed").
    GroupBy("user_id").
    Having("COUNT(*)", ">", 5).
    Having("SUM(amount)", ">", 1000).
    OrderBy("total", "DESC").
    Limit(10).
    ToArray()
```

### 📄 Pagination

```go
// Method 1: Manual pagination
page1, err := gsorm.DB().Table("products").
    Where("active", "=", 1).
    OrderBy("created_at", "DESC").
    Limit(20).
    Offset(0).
    ToArray()

// Method 2: Helper pagination
page2, err := gsorm.DB().Table("products").
    Where("active", "=", 1).
    OrderBy("created_at", "DESC").
    Paginate(2, 20). // Page 2, 20 items per page
    ToArray()
```

### 🔄 Transactions

#### Manual Transaction Control

```go
builder := gsorm.DB()

// Begin transaction
err := builder.BeginTransaction()
if err != nil {
    log.Fatal(err)
}

// Perform operations
_, err = builder.Table("accounts").
    Where("id", "=", 1).
    Update(map[string]interface{}{"balance": "balance - 100"})
if err != nil {
    builder.RollbackTransaction()
    log.Fatal(err)
}

_, err = builder.Table("accounts").
    Where("id", "=", 2).
    Update(map[string]interface{}{"balance": "balance + 100"})
if err != nil {
    builder.RollbackTransaction()
    log.Fatal(err)
}

// Commit transaction
err = builder.CommitTransaction()
```

#### Transaction Helper

```go
err := gsorm.DB().WithTransaction(func(tx *gsorm.Builder) error {
    // All operations in this function are within transaction
    
    // Transfer money between accounts
    _, err := tx.Table("accounts").
        Where("id", "=", 1).
        Update(map[string]interface{}{"balance": "balance - 100"})
    if err != nil {
        return err // Will trigger automatic rollback
    }
    
    _, err = tx.Table("accounts").
        Where("id", "=", 2).
        Update(map[string]interface{}{"balance": "balance + 100"})
    if err != nil {
        return err // Will trigger automatic rollback
    }
    
    // Log transaction
    _, err = tx.Table("transactions").Insert(map[string]interface{}{
        "from_account": 1,
        "to_account":   2,
        "amount":       100,
        "type":         "transfer",
        "created_at":   time.Now(),
    })
    
    return err // Will commit if no error, rollback if error
})

if err != nil {
    log.Printf("Transaction failed: %v", err)
}
```

### 🛠️ Utility Functions

#### Query Builder Cloning

```go
// Create a base query
baseQuery := gsorm.DB().Table("products").
    Where("status", "=", "active").
    Where("stock", ">", 0)

// Clone for different categories
electronics := baseQuery.Clone().
    Where("category", "=", "electronics").
    OrderBy("price", "ASC").
    ToArray()

clothing := baseQuery.Clone().
    Where("category", "=", "clothing").
    OrderBy("name", "ASC").
    ToArray()
```

#### Debug SQL Output

```go
query := gsorm.DB().Table("users").
    Where("age", ">=", 18).
    Where("status", "=", "active").
    WhereIn("role", []interface{}{"admin", "user"}).
    OrderBy("created_at", "DESC").
    Limit(20)

fmt.Printf("Generated SQL: %s\n", query.PrintSQL())
// Output: SELECT * FROM users WHERE age >= 18 AND status = 'active' AND role IN ('admin', 'user') ORDER BY created_at DESC LIMIT 20
```

## 🔒 Security Features

### SQL Injection Prevention

GSORM menggunakan prepared statements untuk **100% perlindungan** dari SQL injection:

```go
// ❌ DANGEROUS: Raw SQL (never do this)
// query := "SELECT * FROM users WHERE name = '" + userInput + "'"

// ✅ SAFE: GSORM (always use this)
users, err := gsorm.DB().Table("users").
    Where("name", "=", userInput). // userInput is automatically escaped
    ToArray()
```

### Tested Against Advanced Attacks

GSORM telah diuji terhadap berbagai jenis serangan SQL injection:

- ✅ Basic injection attempts (`'; DROP TABLE users; --`)
- ✅ Union-based attacks (`1' UNION SELECT password FROM admin --`) 
- ✅ Boolean-based attacks (`1' OR '1'='1`)
- ✅ Time-based attacks (`1'; WAITFOR DELAY '00:00:10'; --`)
- ✅ Second-order injection
- ✅ Unicode normalization attacks
- ✅ Hex encoding attacks

### Input Validation

```go
// Automatic ORDER BY direction validation
gsorm.DB().Table("users").
    OrderBy("name", "INVALID_DIRECTION") // Automatically defaults to "ASC"

// Safe operator validation
validOperators := []string{"=", "!=", "<>", ">", ">=", "<", "<=", "LIKE", "NOT LIKE", "IN", "IS NULL", "IS NOT NULL"}
```

## ⚡ Performance Benchmarks

### Test Environment
- **Platform**: Apple Silicon (ARM64)
- **Go Version**: 1.21.1
- **Database**: SQLite (in-memory)
- **Dataset**: 1,000 test records

### 🏆 Core Operations Performance

| Operation | Time (ns/op) | Memory (B/op) | Allocations | Performance Level |
|-----------|--------------|---------------|-------------|-------------------|
| **Query Building** | 181.3 | 512 | 6 | 🟢 Excellent |
| **Build Select Query** | 287.9 | 344 | 10 | 🟢 Excellent |
| **Clone Builder** | 121.7 | 416 | 4 | 🟢 Excellent |
| **Print SQL Debug** | 567.7 | 688 | 15 | 🟢 Excellent |

### 🗄️ Database Operations Performance

| Operation | Time (ns/op) | Memory (B/op) | Allocations | Performance Level |
|-----------|--------------|---------------|-------------|-------------------|
| **Simple Select** | 2,456 | 736 | 18 | 🟢 Excellent |
| **Select with WHERE** | 3,130 | 1,016 | 26 | 🟢 Excellent |
| **Multiple WHERE** | 4,521 | 1,800 | 34 | 🟡 Very Good |
| **Count Query** | 2,755 | 928 | 32 | 🟢 Excellent |
| **First Record** | 1,778 | 844 | 23 | 🟢 Excellent |
| **Insert Operation** | 9,951 | 1,576 | 39 | 🟡 Good |
| **Bulk Insert (100)** | 441,003 | 129,891 | 1,926 | 🟡 Good |
| **Update Operation** | 3,533 | 915 | 31 | 🟢 Excellent |
| **Transaction** | 14,463 | 2,218 | 69 | 🟡 Very Good |

### 📈 Performance Improvements After Optimization

| Metric | Before | After | Improvement |
|--------|--------|-------|-------------|
| **Query Building Speed** | 547.7 ns | 287.9 ns | 🚀 **47.4% faster** |
| **Memory Usage** | 672 B | 344 B | 🧠 **48.8% less memory** |
| **Allocations** | 18 allocs | 10 allocs | 📊 **44.4% fewer allocs** |

### 🔧 Performance Optimizations

1. **String Builder Pool**: Reduces GC pressure with object pooling
2. **Optimized Clone()**: Conditional allocation for better memory efficiency  
3. **Efficient Query Building**: Using `strings.Builder` instead of concatenation
4. **Smart WHERE Clauses**: Direct building without intermediate arrays

## 🧪 Testing

### Unit Tests Coverage

- ✅ **33 comprehensive unit tests**
- ✅ **All CRUD operations**
- ✅ **Transaction handling**
- ✅ **Error scenarios**
- ✅ **Security validations**
- ✅ **Edge cases**

### Benchmark Suite

- ✅ **30+ performance benchmarks**
- ✅ **Memory allocation analysis**
- ✅ **Concurrent operation testing**
- ✅ **Scalability testing**

```bash
# Run tests
go test -v

# Run benchmarks
go test -bench=. -benchmem

# Run specific benchmarks
go test -bench=BenchmarkSelect -benchmem
```

## 🚀 Advanced Features

### Bulk Operations

```go
// High-performance bulk insert
users := make([]map[string]interface{}, 1000)
for i := 0; i < 1000; i++ {
    users[i] = map[string]interface{}{
        "name":  fmt.Sprintf("User %d", i),
        "email": fmt.Sprintf("user%d@example.com", i),
    }
}

// Single optimized query for 1000 records
err := gsorm.DB().Table("users").InsertBulk(users)

// Bulk update with CASE WHEN optimization
updates := []map[string]interface{}{
    {"id": 1, "status": "premium"},
    {"id": 2, "status": "verified"},
    {"id": 3, "status": "suspended"},
}

err = gsorm.DB().Table("users").UpdateBulk(updates, "id")
```

### Complex Queries

```go
// Advanced reporting query
report, err := gsorm.DB().Table("orders").
    Select(
        "DATE(created_at) as date",
        "COUNT(*) as total_orders", 
        "SUM(amount) as total_revenue",
        "AVG(amount) as avg_order_value",
    ).
    LeftJoin("users", "users.id = orders.user_id").
    LeftJoin("user_segments", "user_segments.user_id = users.id").
    Where("orders.status", "=", "completed").
    Where("orders.created_at", ">=", "2024-01-01").
    WhereIn("user_segments.segment", []interface{}{"premium", "vip"}).
    GroupBy("DATE(created_at)").
    Having("COUNT(*)", ">", 10).
    OrderBy("date", "DESC").
    Limit(30).
    ToArray()
```

## 🏗️ Architecture

### Singleton Pattern Benefits

1. **Single Database Connection**: Efficient resource usage
2. **Thread Safety**: Safe concurrent access with `sync.Once`
3. **Memory Efficiency**: No connection duplication
4. **State Isolation**: Each `DB()` call returns isolated clone

### Memory Management

- **Object Pooling**: Reuses string builders to reduce allocations
- **Conditional Allocation**: Only allocates slices when needed
- **Efficient Cloning**: Smart copying of builder state
- **GC Friendly**: Minimal garbage collection pressure

## 📖 Best Practices

### 1. Initialization

```go
// ✅ Good: Initialize once at application start
func main() {
    db := setupDatabase()
    gsorm.Set(db) // Call once
    
    startServer()
}

// ❌ Bad: Multiple initializations
func badExample() {
    gsorm.Set(db1) // Don't do this
    gsorm.Set(db2) // Or this
}
```

### 2. Query Building

```go
// ✅ Good: Method chaining
users := gsorm.DB().Table("users").
    Where("active", "=", 1).
    Where("age", ">=", 18).
    OrderBy("created_at", "DESC").
    Limit(20).
    ToArray()

// ✅ Good: Reusable base queries  
baseQuery := gsorm.DB().Table("products").Where("active", "=", 1)
electronics := baseQuery.Clone().Where("category", "=", "electronics").ToArray()
clothing := baseQuery.Clone().Where("category", "=", "clothing").ToArray()
```

### 3. Error Handling

```go
// ✅ Good: Always check errors
users, err := gsorm.DB().Table("users").ToArray()
if err != nil {
    log.Printf("Database error: %v", err)
    return err
}

// ✅ Good: Transaction error handling
err := gsorm.DB().WithTransaction(func(tx *gsorm.Builder) error {
    if _, err := tx.Table("orders").Insert(orderData); err != nil {
        return fmt.Errorf("failed to insert order: %w", err)
    }
    return nil
})
```

### 4. Performance Optimization

```go
// ✅ Good: Use bulk operations for multiple records
err := gsorm.DB().Table("users").InsertBulk(largeDataset)

// ❌ Avoid: Multiple single inserts
for _, user := range largeDataset {
    gsorm.DB().Table("users").Insert(user) // Inefficient
}

// ✅ Good: Use specific columns
users := gsorm.DB().Table("users").
    Select("id", "name", "email"). // Only needed columns
    ToArray()

// ❌ Avoid: Select all when not needed
users := gsorm.DB().Table("users").ToArray() // Selects all columns
```

## 🤝 Contributing

Contributions are welcome! Please read our contributing guidelines:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Write tests for your changes
4. Run the test suite (`go test -v`)
5. Run benchmarks (`go test -bench=.`)
6. Commit your changes (`git commit -m 'Add amazing feature'`)
7. Push to the branch (`git push origin feature/amazing-feature`)
8. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Inspired by Laravel's Eloquent ORM
- Built with performance in mind using Go best practices
- Comprehensive testing ensures reliability and security

---

**Made with ❤️ for the Go community**

For more examples and advanced usage, check out the [examples](examples/) directory and [performance documentation](PERFORMANCE.md).