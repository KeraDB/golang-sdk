package benchmark

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"testing"

	"github.com/keradb/golang-sdk"
	_ "modernc.org/sqlite"
)

// Test data structures
type Document struct {
	ID       string                 `json:"_id,omitempty"`
	Name     string                 `json:"name"`
	Age      int                    `json:"age"`
	Email    string                 `json:"email"`
	Active   bool                   `json:"active"`
	Metadata map[string]interface{} `json:"metadata"`
}

// Benchmark configuration
const (
	numDocs         = 10000
	batchSize       = 100
	vectorDimension = 128
	numVectors      = 5000
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
		os.Remove(dbPath + ".vectors.ndb")
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

	// Create vectors table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS vectors (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			embedding TEXT,
			metadata TEXT
		)
	`)
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

// ============================================================================
// Vector Benchmarks
// ============================================================================

func generateRandomVector(dim int) keradb.Embedding {
	vec := make(keradb.Embedding, dim)
	var sumSquares float32
	for i := 0; i < dim; i++ {
		val := rand.Float32()*2 - 1
		vec[i] = val
		sumSquares += val * val
	}
	// Normalize
	norm := float32(math.Sqrt(float64(sumSquares)))
	for i := 0; i < dim; i++ {
		vec[i] /= norm
	}
	return vec
}

func BenchmarkKeraDB_VectorInsert(b *testing.B) {
	client := setupKeraDB(b)

	config := keradb.NewVectorConfig(vectorDimension).
		WithDistance(keradb.Cosine).
		WithM(16)

	err := client.CreateVectorCollection("embeddings", config)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		vec := generateRandomVector(vectorDimension)
		metadata := keradb.M{"index": i}
		_, err := client.InsertVector("embeddings", vec, metadata)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSQLite_VectorInsert(b *testing.B) {
	db := setupSQLite(b)

	stmt, err := db.Prepare("INSERT INTO vectors (embedding, metadata) VALUES (?, ?)")
	if err != nil {
		b.Fatal(err)
	}
	defer stmt.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		vec := generateRandomVector(vectorDimension)
		vecJSON, _ := json.Marshal(vec)
		metadata := map[string]interface{}{"index": i}
		metadataJSON, _ := json.Marshal(metadata)

		_, err := stmt.Exec(string(vecJSON), string(metadataJSON))
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkKeraDB_VectorSearch(b *testing.B) {
	client := setupKeraDB(b)

	config := keradb.NewVectorConfig(vectorDimension).
		WithDistance(keradb.Cosine).
		WithM(16).
		WithEfSearch(50)

	err := client.CreateVectorCollection("embeddings", config)
	if err != nil {
		b.Fatal(err)
	}

	// Insert test vectors
	for i := 0; i < numVectors; i++ {
		vec := generateRandomVector(vectorDimension)
		metadata := keradb.M{"index": i}
		_, err := client.InsertVector("embeddings", vec, metadata)
		if err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		queryVec := generateRandomVector(vectorDimension)
		_, err := client.VectorSearch("embeddings", queryVec, 10)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSQLite_VectorSearch_Naive(b *testing.B) {
	db := setupSQLite(b)

	// Insert test vectors
	stmt, err := db.Prepare("INSERT INTO vectors (embedding, metadata) VALUES (?, ?)")
	if err != nil {
		b.Fatal(err)
	}

	vectors := make([]keradb.Embedding, numVectors)
	for i := 0; i < numVectors; i++ {
		vec := generateRandomVector(vectorDimension)
		vectors[i] = vec
		vecJSON, _ := json.Marshal(vec)
		metadata := map[string]interface{}{"index": i}
		metadataJSON, _ := json.Marshal(metadata)

		_, err := stmt.Exec(string(vecJSON), string(metadataJSON))
		if err != nil {
			b.Fatal(err)
		}
	}
	stmt.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		queryVec := generateRandomVector(vectorDimension)

		// Naive linear search with cosine similarity
		rows, err := db.Query("SELECT id, embedding FROM vectors")
		if err != nil {
			b.Fatal(err)
		}

		type result struct {
			id    int
			score float32
		}
		results := make([]result, 0, numVectors)

		for rows.Next() {
			var id int
			var embeddingJSON string
			if err := rows.Scan(&id, &embeddingJSON); err != nil {
				rows.Close()
				b.Fatal(err)
			}

			var embedding keradb.Embedding
			json.Unmarshal([]byte(embeddingJSON), &embedding)

			// Compute cosine similarity
			var dot, normA, normB float32
			for j := 0; j < vectorDimension; j++ {
				dot += queryVec[j] * embedding[j]
				normA += queryVec[j] * queryVec[j]
				normB += embedding[j] * embedding[j]
			}
			similarity := dot / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))

			results = append(results, result{id: id, score: similarity})
		}
		rows.Close()

		// Sort top 10 (simplified - just find max 10 times)
		for k := 0; k < 10 && k < len(results); k++ {
			maxIdx := k
			for j := k + 1; j < len(results); j++ {
				if results[j].score > results[maxIdx].score {
					maxIdx = j
				}
			}
			results[k], results[maxIdx] = results[maxIdx], results[k]
		}
	}
}

// ============================================================================
// Compression Benchmarks
// ============================================================================

func BenchmarkKeraDB_VectorInsert_WithCompression(b *testing.B) {
	client := setupKeraDB(b)

	config := keradb.NewVectorConfig(vectorDimension).
		WithDistance(keradb.Cosine).
		WithM(16).
		WithDeltaCompression()

	err := client.CreateVectorCollection("embeddings", config)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		vec := generateRandomVector(vectorDimension)
		metadata := keradb.M{"index": i}
		_, err := client.InsertVector("embeddings", vec, metadata)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkKeraDB_VectorSearch_WithCompression(b *testing.B) {
	client := setupKeraDB(b)

	config := keradb.NewVectorConfig(vectorDimension).
		WithDistance(keradb.Cosine).
		WithM(16).
		WithEfSearch(50).
		WithDeltaCompression()

	err := client.CreateVectorCollection("embeddings", config)
	if err != nil {
		b.Fatal(err)
	}

	// Insert test vectors
	for i := 0; i < numVectors; i++ {
		vec := generateRandomVector(vectorDimension)
		metadata := keradb.M{"index": i}
		_, err := client.InsertVector("embeddings", vec, metadata)
		if err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		queryVec := generateRandomVector(vectorDimension)
		_, err := client.VectorSearch("embeddings", queryVec, 10)
		if err != nil {
			b.Fatal(err)
		}
	}
}
