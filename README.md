# keradb Go SDK

Go SDK for keradb - a lightweight, embedded NoSQL document database.

## Installation

```bash
go get github.com/KeraDB/golang-sdk
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
    
    "github.com/KeraDB/golang-sdk"
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
