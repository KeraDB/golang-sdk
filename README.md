# keradb Go SDK

Go SDK for keradb - a lightweight, embedded NoSQL document database with vector search capabilities.

## Installation

```bash
go get github.com/keradb/golang-sdk
```

For development from source:

```bash
cd sdks/go
go mod download
go build
```

## Prerequisites

The native keradb library must be built first:

```bash
# From the root of the keradb project
cargo build --release
```

Make sure the library is in your library path or set the appropriate environment variables:

**Linux/macOS:**
```bash
export LD_LIBRARY_PATH=$LD_LIBRARY_PATH:$(pwd)/../../target/release
export DYLD_LIBRARY_PATH=$DYLD_LIBRARY_PATH:$(pwd)/../../target/release
```

**Windows:**
```powershell
$env:PATH += ";$(pwd)\..\..\target\release"
```

## Usage

### Basic Example

```go
package main

import (
    "fmt"
    "log"

    "github.com/keradb/golang-sdk"
)

func main() {
    // Create a new database
    db, err := keradb.Create("mydata.ndb")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()
    
    // Insert a document
    id, err := db.Insert("users", map[string]interface{}{
        "name":  "Alice",
        "age":   30,
        "email": "alice@example.com",
    })
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Inserted document with ID: %s\n", id)
    
    // Find by ID
    doc, err := db.FindByID("users", id)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Found: %v\n", doc)
    
    // Update
    updated, err := db.Update("users", id, map[string]interface{}{
        "name":  "Alice",
        "age":   31,
        "email": "alice@example.com",
    })
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Updated: %v\n", updated)
    
    // Find all
    allDocs, err := db.FindAll("users", nil)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("All documents: %v\n", allDocs)
    
    // Count
    count := db.Count("users")
    fmt.Printf("Count: %d\n", count)
    
    // List collections
    collections, err := db.ListCollections()
    if err != nil {
        log.Fatal(err)
    }
    for _, col := range collections {
        fmt.Printf("%s: %d documents\n", col.Name, col.Count)
    }
    
    // Delete
    if err := db.Delete("users", id); err != nil {
        log.Fatal(err)
    }
    
    // Sync to disk
    if err := db.Sync(); err != nil {
        log.Fatal(err)
    }
}
```

### Pagination

```go
// Get first 10 documents
limit := 10
skip := 0
page1, err := db.FindAll("users", &keradb.FindAllOptions{
    Limit: &limit,
    Skip:  &skip,
})

// Get next 10 documents
skip = 10
page2, err := db.FindAll("users", &keradb.FindAllOptions{
    Limit: &limit,
    Skip:  &skip,
})
```

### Working with Documents

```go
doc, err := db.FindByID("users", id)
if err != nil {
    log.Fatal(err)
}

// Access document ID
fmt.Printf("ID: %s\n", doc.ID())

// Access document fields
name := doc["name"].(string)
age := doc["age"].(float64)
fmt.Printf("Name: %s, Age: %.0f\n", name, age)
```

### Error Handling

```go
doc, err := db.FindByID("users", "non-existent-id")
if err != nil {
    // Handle error
    fmt.Printf("Error: %v\n", err)
    return
}
```

### Struct Mapping

```go
type User struct {
    Name  string `json:"name"`
    Age   int    `json:"age"`
    Email string `json:"email"`
}

// Insert
user := User{
    Name:  "Bob",
    Age:   25,
    Email: "bob@example.com",
}

data := map[string]interface{}{
    "name":  user.Name,
    "age":   user.Age,
    "email": user.Email,
}

id, err := db.Insert("users", data)

// Or use json marshaling
jsonData, _ := json.Marshal(user)
var mapData map[string]interface{}
json.Unmarshal(jsonData, &mapData)
id, err = db.Insert("users", mapData)
```

## API Reference

### Types

#### Database

```go
type Database struct {
    // private fields
}
```

#### Document

```go
type Document map[string]interface{}
```

Methods:
- `ID() string` - Returns the document ID

#### Collection

```go
type Collection struct {
    Name  string
    Count int
}
```

#### FindAllOptions

```go
type FindAllOptions struct {
    Limit *int
    Skip  *int
}
```

### Functions

#### Create

```go
func Create(path string) (*Database, error)
```

Create a new keradb database.

#### Open

```go
func Open(path string) (*Database, error)
```

Open an existing keradb database.

### Database Methods

#### Insert

```go
func (db *Database) Insert(collection string, data map[string]interface{}) (string, error)
```

Insert a document into a collection. Returns the document ID.

#### FindByID

```go
func (db *Database) FindByID(collection, docID string) (Document, error)
```

Find a document by its ID.

#### Update

```go
func (db *Database) Update(collection, docID string, data map[string]interface{}) (Document, error)
```

Update a document. Returns the updated document.

#### Delete

```go
func (db *Database) Delete(collection, docID string) error
```

Delete a document.

#### FindAll

```go
func (db *Database) FindAll(collection string, opts *FindAllOptions) ([]Document, error)
```

Find all documents in a collection with optional pagination.

#### Count

```go
func (db *Database) Count(collection string) int
```

Count documents in a collection.

#### ListCollections

```go
func (db *Database) ListCollections() ([]Collection, error)
```

List all collections with their document counts.

#### Sync

```go
func (db *Database) Sync() error
```

Sync all changes to disk.

#### Close

```go
func (db *Database) Close() error
```

Close the database connection.

#### Path

```go
func (db *Database) Path() string
```

Get the database path.

## Vector Search

KeraDB includes built-in vector search capabilities for semantic search, similarity matching, and AI applications.

### Vector Search Features

- **HNSW (Hierarchical Navigable Small World)** index for fast approximate nearest neighbor search
- **Multiple distance metrics**: Cosine (default), Euclidean (L2), Dot Product, Manhattan (L1)
- **LEANN-style delta compression** for 85-95% storage savings
- **Metadata filtering** for hybrid search
- **Lazy embedding mode** for reduced memory footprint
- **Thread-safe** concurrent access

### Basic Vector Search Example

```go
package main

import (
    "fmt"
    "log"

    "github.com/keradb/golang-sdk"
)

func main() {
    // Connect to database
    client, err := keradb.Connect("mydb.ndb")
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // Create a vector collection with 384 dimensions
    config := keradb.NewVectorConfig(384).
        WithDistance(keradb.Cosine).
        WithDeltaCompression()

    if err := client.CreateVectorCollection("embeddings", config); err != nil {
        log.Fatal(err)
    }

    // Insert vectors with metadata
    vector1 := keradb.Embedding{0.1, 0.2, 0.3, /* ... 384 dimensions */}
    metadata1 := keradb.M{"category": "tech", "title": "AI Article"}

    id, err := client.InsertVector("embeddings", vector1, metadata1)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Inserted vector with ID: %d\n", id)

    // Search for similar vectors
    queryVector := keradb.Embedding{0.11, 0.19, 0.31, /* ... */}
    results, err := client.VectorSearch("embeddings", queryVector, 10)
    if err != nil {
        log.Fatal(err)
    }

    for _, result := range results {
        fmt.Printf("Rank %d: ID=%d, Score=%.4f, Title=%v\n",
            result.Rank, result.Document.ID, result.Score,
            result.Document.Metadata["title"])
    }

    // Get collection statistics
    stats, err := client.VectorStats("embeddings")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Collection has %d vectors, %d dimensions\n",
        stats.VectorCount, stats.Dimensions)
}
```

### Vector Configuration Options

```go
// Basic configuration
config := keradb.NewVectorConfig(768)

// With custom distance metric
config.WithDistance(keradb.Euclidean)

// HNSW parameters for quality/performance trade-offs
config.WithM(16)              // More connections = better recall, more memory
config.WithEfConstruction(200) // Higher = better index quality, slower build
config.WithEfSearch(50)        // Higher = better recall, slower search

// Enable delta compression (85-95% storage savings)
config.WithDeltaCompression()

// Or quantized compression for maximum savings
config.WithQuantizedCompression()

// Lazy embedding mode (store text, compute on-demand)
config.WithLazyEmbedding("text-embedding-ada-002")
```

### Text-Based Vector Search

```go
// Insert text (requires embedding provider setup)
id, err := client.InsertText("embeddings", "Machine learning tutorial",
    keradb.M{"category": "education"})

// Search by text query
results, err := client.VectorSearchText("embeddings", "AI tutorials", 5)
```

### Filtered Vector Search

```go
// Search with metadata filter
filter := keradb.MetadataFilter{
    Field:     "category",
    Condition: "eq",
    Value:     "tech",
}

results, err := client.VectorSearchFiltered("embeddings", queryVector, 10, filter)
```

### Distance Metrics

| Metric | Use Case | Range |
|--------|----------|-------|
| `Cosine` | Text embeddings, normalized vectors | [0, 2] (0 = identical) |
| `Euclidean` | General purpose, image features | [0, ∞) |
| `DotProduct` | Pre-normalized vectors, fast ranking | (-∞, ∞) |
| `Manhattan` | High-dimensional spaces, robust to outliers | [0, ∞) |

### Compression

KeraDB uses LEANN-inspired delta compression:

```go
// Delta compression (stores sparse differences from neighbors)
config.WithDeltaCompression()

// Quantized delta (aggressive compression with minimal quality loss)
config.WithQuantizedCompression()

// Custom compression settings
config.WithCompression(keradb.CompressionConfig{
    Mode:              keradb.DeltaCompression,
    SparsityThreshold: float32Ptr(0.001),
    MaxDensity:        float32Ptr(0.15),
    AnchorFrequency:   intPtr(8),
})
```

**Storage savings**: 85-95% reduction in disk usage while maintaining search quality.

### Vector API Reference

#### Client Methods

```go
// Collection management
CreateVectorCollection(name string, config *VectorConfig) error
ListVectorCollections() ([]struct{Name string; Count int}, error)
DropVectorCollection(name string) (bool, error)

// Insert operations
InsertVector(collection string, embedding Embedding, metadata M) (VectorID, error)
InsertText(collection string, text string, metadata M) (VectorID, error)

// Search operations
VectorSearch(collection string, queryVector Embedding, k int) ([]VectorSearchResult, error)
VectorSearchText(collection string, queryText string, k int) ([]VectorSearchResult, error)
VectorSearchFiltered(collection string, queryVector Embedding, k int, filter MetadataFilter) ([]VectorSearchResult, error)

// Document operations
GetVector(collection string, id VectorID) (*VectorDocument, error)
DeleteVector(collection string, id VectorID) (bool, error)

// Statistics
VectorStats(collection string) (*VectorCollectionStats, error)
```

#### Types

```go
type VectorID uint64
type Embedding []float32

type VectorConfig struct {
    Dimensions      int
    Distance        Distance
    M               *int
    EfConstruction  *int
    EfSearch        *int
    LazyEmbedding   *bool
    EmbeddingModel  *string
    Compression     *CompressionConfig
}

type VectorDocument struct {
    ID        VectorID
    Embedding *Embedding
    Text      *string
    Metadata  map[string]interface{}
}

type VectorSearchResult struct {
    Document VectorDocument
    Score    float32
    Rank     int
}

type VectorCollectionStats struct {
    VectorCount    int
    Dimensions     int
    Distance       Distance
    MemoryUsage    int64
    LayerCount     int
    LazyEmbedding  bool
    Compression    *CompressionMode
    AnchorCount    *int
    DeltaCount     *int
}
```

## Building

```bash
go build
```

## Testing

```bash
go test -v
```

## Platform Support

- Linux (x64, ARM64)
- macOS (x64, Apple Silicon)
- Windows (x64)

## License

MIT License
