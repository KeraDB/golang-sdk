package benchmark

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"testing"

	"github.com/keradb/golang-sdk"
	_ "modernc.org/sqlite"
)

// Benchmark configuration
const (
	numDocs   = 1000
	batchSize = 100
)

// ============================================================================
// KeraDB Setup
// ============================================================================

func setupKeraDB(b *testing.B) *keradb.Client {
	dbPath := fmt.Sprintf("bench_keradb_%d.ndb", rand.Int())
	client, err := keradb.Connect(dbPath)
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() {
		client.Close()
		os.Remove(dbPath)
	})
	return client
}

// ============================================================================
// SQLite Setup
// ============================================================================

func setupSQLite(b *testing.B) *sql.DB {
	dbPath := fmt.Sprintf("bench_sqlite_%d.db", rand.Int())
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		b.Fatal(err)
	}

	// Create documents table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS documents (
			id TEXT PRIMARY KEY,
			name TEXT,
			age INTEGER,
			email TEXT,
			active INTEGER,
			metadata TEXT
		)
	`)
	if err != nil {
		b.Fatal(err)
	}

	// Create index for faster lookups
	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_id ON documents(id)`)
	if err != nil {
		b.Fatal(err)
	}

	b.Cleanup(func() {
		db.Close()
		os.Remove(dbPath)
	})

	return db
}

// ============================================================================
// Document Benchmarks
// ============================================================================

func BenchmarkKeraDB_Insert(b *testing.B) {
	client := setupKeraDB(b)
	db := client.Database()
	coll := db.Collection("users")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		doc := keradb.M{
			"name":   fmt.Sprintf("User%d", i),
			"age":    20 + (i % 50),
			"email":  fmt.Sprintf("user%d@example.com", i),
			"active": i%2 == 0,
		}
		_, err := coll.InsertOne(doc)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSQLite_Insert(b *testing.B) {
	db := setupSQLite(b)

	stmt, err := db.Prepare("INSERT INTO documents (id, name, age, email, active, metadata) VALUES (?, ?, ?, ?, ?, ?)")
	if err != nil {
		b.Fatal(err)
	}
	defer stmt.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metadata := map[string]interface{}{}
		metadataJSON, _ := json.Marshal(metadata)

		_, err := stmt.Exec(
			fmt.Sprintf("doc%d", i),
			fmt.Sprintf("User%d", i),
			20+(i%50),
			fmt.Sprintf("user%d@example.com", i),
			i%2,
			string(metadataJSON),
		)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkKeraDB_InsertBatch(b *testing.B) {
	client := setupKeraDB(b)
	db := client.Database()
	coll := db.Collection("users")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		docs := make([]interface{}, batchSize)
		for j := 0; j < batchSize; j++ {
			docs[j] = keradb.M{
				"name":   fmt.Sprintf("User%d", i*batchSize+j),
				"age":    20 + ((i*batchSize + j) % 50),
				"email":  fmt.Sprintf("user%d@example.com", i*batchSize+j),
				"active": (i*batchSize+j)%2 == 0,
			}
		}
		_, err := coll.InsertMany(docs)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSQLite_InsertBatch(b *testing.B) {
	db := setupSQLite(b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tx, err := db.Begin()
		if err != nil {
			b.Fatal(err)
		}

		stmt, err := tx.Prepare("INSERT INTO documents (id, name, age, email, active, metadata) VALUES (?, ?, ?, ?, ?, ?)")
		if err != nil {
			b.Fatal(err)
		}

		for j := 0; j < batchSize; j++ {
			idx := i*batchSize + j
			metadata := map[string]interface{}{}
			metadataJSON, _ := json.Marshal(metadata)

			_, err := stmt.Exec(
				fmt.Sprintf("doc%d", idx),
				fmt.Sprintf("User%d", idx),
				20+(idx%50),
				fmt.Sprintf("user%d@example.com", idx),
				idx%2,
				string(metadataJSON),
			)
			if err != nil {
				stmt.Close()
				tx.Rollback()
				b.Fatal(err)
			}
		}

		stmt.Close()
		if err := tx.Commit(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkKeraDB_FindByID(b *testing.B) {
	client := setupKeraDB(b)
	db := client.Database()
	coll := db.Collection("users")

	// Insert test data
	ids := make([]string, numDocs)
	for i := 0; i < numDocs; i++ {
		doc := keradb.M{
			"name":  fmt.Sprintf("User%d", i),
			"age":   20 + (i % 50),
			"email": fmt.Sprintf("user%d@example.com", i),
		}
		result, err := coll.InsertOne(doc)
		if err != nil {
			b.Fatal(err)
		}
		ids[i] = result.InsertedID
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		id := ids[i%numDocs]
		result := coll.FindOne(keradb.M{"_id": id})
		if result.Err() != nil {
			b.Fatal(result.Err())
		}
	}
}

func BenchmarkSQLite_FindByID(b *testing.B) {
	db := setupSQLite(b)

	// Insert test data
	stmt, err := db.Prepare("INSERT INTO documents (id, name, age, email, active, metadata) VALUES (?, ?, ?, ?, ?, ?)")
	if err != nil {
		b.Fatal(err)
	}

	ids := make([]string, numDocs)
	for i := 0; i < numDocs; i++ {
		id := fmt.Sprintf("doc%d", i)
		ids[i] = id
		metadata := map[string]interface{}{}
		metadataJSON, _ := json.Marshal(metadata)

		_, err := stmt.Exec(id, fmt.Sprintf("User%d", i), 20+(i%50), fmt.Sprintf("user%d@example.com", i), i%2, string(metadataJSON))
		if err != nil {
			b.Fatal(err)
		}
	}
	stmt.Close()

	selectStmt, err := db.Prepare("SELECT id, name, age, email, active, metadata FROM documents WHERE id = ?")
	if err != nil {
		b.Fatal(err)
	}
	defer selectStmt.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		id := ids[i%numDocs]
		var docID, name, email, metadata string
		var age, active int
		err := selectStmt.QueryRow(id).Scan(&docID, &name, &age, &email, &active, &metadata)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkKeraDB_Update(b *testing.B) {
	client := setupKeraDB(b)
	db := client.Database()
	coll := db.Collection("users")

	// Insert test data
	ids := make([]string, numDocs)
	for i := 0; i < numDocs; i++ {
		doc := keradb.M{
			"name":  fmt.Sprintf("User%d", i),
			"age":   20 + (i % 50),
			"email": fmt.Sprintf("user%d@example.com", i),
		}
		result, err := coll.InsertOne(doc)
		if err != nil {
			b.Fatal(err)
		}
		ids[i] = result.InsertedID
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		id := ids[i%numDocs]
		_, err := coll.UpdateOne(
			keradb.M{"_id": id},
			keradb.M{"$set": keradb.M{"age": 30 + (i % 40)}},
		)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSQLite_Update(b *testing.B) {
	db := setupSQLite(b)

	// Insert test data
	stmt, err := db.Prepare("INSERT INTO documents (id, name, age, email, active, metadata) VALUES (?, ?, ?, ?, ?, ?)")
	if err != nil {
		b.Fatal(err)
	}

	ids := make([]string, numDocs)
	for i := 0; i < numDocs; i++ {
		id := fmt.Sprintf("doc%d", i)
		ids[i] = id
		metadata := map[string]interface{}{}
		metadataJSON, _ := json.Marshal(metadata)

		_, err := stmt.Exec(id, fmt.Sprintf("User%d", i), 20+(i%50), fmt.Sprintf("user%d@example.com", i), i%2, string(metadataJSON))
		if err != nil {
			b.Fatal(err)
		}
	}
	stmt.Close()

	updateStmt, err := db.Prepare("UPDATE documents SET age = ? WHERE id = ?")
	if err != nil {
		b.Fatal(err)
	}
	defer updateStmt.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		id := ids[i%numDocs]
		_, err := updateStmt.Exec(30+(i%40), id)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkKeraDB_Delete(b *testing.B) {
	client := setupKeraDB(b)
	db := client.Database()
	coll := db.Collection("users")

	// Insert test data
	ids := make([]string, b.N)
	for i := 0; i < b.N; i++ {
		doc := keradb.M{
			"name":  fmt.Sprintf("User%d", i),
			"age":   20 + (i % 50),
			"email": fmt.Sprintf("user%d@example.com", i),
		}
		result, err := coll.InsertOne(doc)
		if err != nil {
			b.Fatal(err)
		}
		ids[i] = result.InsertedID
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := coll.DeleteOne(keradb.M{"_id": ids[i]})
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSQLite_Delete(b *testing.B) {
	db := setupSQLite(b)

	// Insert test data
	insertStmt, err := db.Prepare("INSERT INTO documents (id, name, age, email, active, metadata) VALUES (?, ?, ?, ?, ?, ?)")
	if err != nil {
		b.Fatal(err)
	}

	ids := make([]string, b.N)
	for i := 0; i < b.N; i++ {
		id := fmt.Sprintf("doc%d", i)
		ids[i] = id
		metadata := map[string]interface{}{}
		metadataJSON, _ := json.Marshal(metadata)

		_, err := insertStmt.Exec(id, fmt.Sprintf("User%d", i), 20+(i%50), fmt.Sprintf("user%d@example.com", i), i%2, string(metadataJSON))
		if err != nil {
			b.Fatal(err)
		}
	}
	insertStmt.Close()

	deleteStmt, err := db.Prepare("DELETE FROM documents WHERE id = ?")
	if err != nil {
		b.Fatal(err)
	}
	defer deleteStmt.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := deleteStmt.Exec(ids[i])
		if err != nil {
			b.Fatal(err)
		}
	}
}
