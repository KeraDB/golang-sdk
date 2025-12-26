// Package keradb provides a MongoDB-compatible Go client for KeraDB.
//
// Usage:
//
//	import "github.com/keradb/golang-sdk"
//
//	client, _ := keradb.Connect("mydb.ndb")
//	defer client.Close()
//
//	db := client.Database()
//	users := db.Collection("users")
//
//	// Insert
//	result, _ := users.InsertOne(map[string]interface{}{"name": "Alice", "age": 30})
//	fmt.Println("Inserted:", result.InsertedID)
//
//	// Find
//	var user map[string]interface{}
//	users.FindOne(keradb.M{"_id": result.InsertedID}).Decode(&user)
//
//	// Update
//	users.UpdateOne(keradb.M{"_id": result.InsertedID}, keradb.M{"$set": keradb.M{"age": 31}})
//
//	// Delete
//	users.DeleteOne(keradb.M{"_id": result.InsertedID})
package keradb

/*
#cgo LDFLAGS: -L${SRCDIR}/../../../target/release -lkeradb
#cgo linux LDFLAGS: -lkeradb -lm -ldl -lpthread
#cgo darwin LDFLAGS: -lkeradb -lm -ldl -lpthread
#cgo windows LDFLAGS: -lkeradb -lws2_32 -luserenv -lbcrypt -lntdll

#include <stdlib.h>

typedef void* KeraDB;

KeraDB keradb_create(const char* path);
KeraDB keradb_open(const char* path);
void keradb_close(KeraDB db);
char* keradb_insert(KeraDB db, const char* collection, const char* json_data);
char* keradb_find_by_id(KeraDB db, const char* collection, const char* doc_id);
char* keradb_update(KeraDB db, const char* collection, const char* doc_id, const char* json_data);
int keradb_delete(KeraDB db, const char* collection, const char* doc_id);
char* keradb_find_all(KeraDB db, const char* collection, int limit, int skip);
int keradb_count(KeraDB db, const char* collection);
char* keradb_list_collections(KeraDB db);
int keradb_sync(KeraDB db);
char* keradb_last_error();
void keradb_free_string(char* s);
*/
import "C"
import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"unsafe"
)

// M is a shorthand for map[string]interface{}, similar to MongoDB's bson.M
type M map[string]interface{}

// D is a shorthand for ordered document (slice of E)
type D []E

// E is a single element in D
type E struct {
	Key   string
	Value interface{}
}

// Document represents a document in the database
type Document map[string]interface{}

// ID returns the document ID
func (d Document) ID() string {
	if id, ok := d["_id"].(string); ok {
		return id
	}
	return ""
}

// ============================================================================
// Result Types (MongoDB-compatible)
// ============================================================================

// InsertOneResult is the result of an InsertOne operation
type InsertOneResult struct {
	InsertedID string
}

// InsertManyResult is the result of an InsertMany operation
type InsertManyResult struct {
	InsertedIDs []string
}

// UpdateResult is the result of an Update operation
type UpdateResult struct {
	MatchedCount  int64
	ModifiedCount int64
}

// DeleteResult is the result of a Delete operation
type DeleteResult struct {
	DeletedCount int64
}

// ============================================================================
// Helper Functions
// ============================================================================

func getLastError() string {
	cErr := C.keradb_last_error()
	if cErr == nil {
		return "Unknown error"
	}
	defer C.keradb_free_string(cErr)
	return C.GoString(cErr)
}

func matchesFilter(doc Document, filter M) bool {
	for key, value := range filter {
		if key == "$and" {
			filters, ok := value.([]M)
			if !ok {
				continue
			}
			for _, f := range filters {
				if !matchesFilter(doc, f) {
					return false
				}
			}
		} else if key == "$or" {
			filters, ok := value.([]M)
			if !ok {
				continue
			}
			matched := false
			for _, f := range filters {
				if matchesFilter(doc, f) {
					matched = true
					break
				}
			}
			if !matched {
				return false
			}
		} else if key[0] == '$' {
			// Skip unknown operators
			continue
		} else {
			docValue := doc[key]

			if opMap, ok := value.(M); ok {
				// Comparison operators
				for op, opValue := range opMap {
					switch op {
					case "$eq":
						if !reflect.DeepEqual(docValue, opValue) {
							return false
						}
					case "$ne":
						if reflect.DeepEqual(docValue, opValue) {
							return false
						}
					case "$gt":
						if !compareGT(docValue, opValue) {
							return false
						}
					case "$gte":
						if !compareGTE(docValue, opValue) {
							return false
						}
					case "$lt":
						if !compareLT(docValue, opValue) {
							return false
						}
					case "$lte":
						if !compareLTE(docValue, opValue) {
							return false
						}
					case "$in":
						if !containsValue(opValue, docValue) {
							return false
						}
					case "$nin":
						if containsValue(opValue, docValue) {
							return false
						}
					}
				}
			} else {
				// Direct equality
				if !reflect.DeepEqual(docValue, value) {
					return false
				}
			}
		}
	}
	return true
}

func compareGT(a, b interface{}) bool {
	switch av := a.(type) {
	case float64:
		if bv, ok := b.(float64); ok {
			return av > bv
		}
	case int:
		if bv, ok := b.(int); ok {
			return av > bv
		}
	case string:
		if bv, ok := b.(string); ok {
			return av > bv
		}
	}
	return false
}

func compareGTE(a, b interface{}) bool {
	return compareGT(a, b) || reflect.DeepEqual(a, b)
}

func compareLT(a, b interface{}) bool {
	switch av := a.(type) {
	case float64:
		if bv, ok := b.(float64); ok {
			return av < bv
		}
	case int:
		if bv, ok := b.(int); ok {
			return av < bv
		}
	case string:
		if bv, ok := b.(string); ok {
			return av < bv
		}
	}
	return false
}

func compareLTE(a, b interface{}) bool {
	return compareLT(a, b) || reflect.DeepEqual(a, b)
}

func containsValue(arr interface{}, val interface{}) bool {
	slice := reflect.ValueOf(arr)
	if slice.Kind() != reflect.Slice {
		return false
	}
	for i := 0; i < slice.Len(); i++ {
		if reflect.DeepEqual(slice.Index(i).Interface(), val) {
			return true
		}
	}
	return false
}

func applyUpdate(doc Document, update M) Document {
	result := make(Document)
	for k, v := range doc {
		result[k] = v
	}

	for op, fields := range update {
		switch op {
		case "$set":
			if setFields, ok := fields.(M); ok {
				for k, v := range setFields {
					result[k] = v
				}
			}
		case "$unset":
			if unsetFields, ok := fields.(M); ok {
				for k := range unsetFields {
					delete(result, k)
				}
			}
		case "$inc":
			if incFields, ok := fields.(M); ok {
				for k, v := range incFields {
					if incVal, ok := v.(float64); ok {
						if curr, ok := result[k].(float64); ok {
							result[k] = curr + incVal
						} else {
							result[k] = incVal
						}
					}
				}
			}
		case "$push":
			if pushFields, ok := fields.(M); ok {
				for k, v := range pushFields {
					if arr, ok := result[k].([]interface{}); ok {
						result[k] = append(arr, v)
					} else {
						result[k] = []interface{}{v}
					}
				}
			}
		default:
			if op[0] != '$' {
				// Replacement mode
				result = Document{"_id": doc["_id"]}
				for k, v := range update {
					result[k] = v
				}
				return result
			}
		}
	}

	return result
}

// ============================================================================
// Cursor
// ============================================================================

// Cursor allows iteration over query results
type Cursor struct {
	documents []Document
	index     int
	limit     int
	skip      int
}

// NewCursor creates a new cursor from documents
func NewCursor(docs []Document) *Cursor {
	return &Cursor{
		documents: docs,
		index:     0,
		limit:     -1,
		skip:      0,
	}
}

// Limit sets the maximum number of documents to return
func (c *Cursor) Limit(n int) *Cursor {
	c.limit = n
	return c
}

// Skip sets the number of documents to skip
func (c *Cursor) Skip(n int) *Cursor {
	c.skip = n
	return c
}

// All returns all documents as a slice
func (c *Cursor) All() ([]Document, error) {
	docs := c.documents
	if c.skip > 0 && c.skip < len(docs) {
		docs = docs[c.skip:]
	} else if c.skip >= len(docs) {
		docs = []Document{}
	}
	if c.limit >= 0 && c.limit < len(docs) {
		docs = docs[:c.limit]
	}
	return docs, nil
}

// Next advances the cursor and returns true if there are more documents
func (c *Cursor) Next() bool {
	docs, _ := c.All()
	return c.index < len(docs)
}

// Decode decodes the current document into the provided value
func (c *Cursor) Decode(v interface{}) error {
	docs, _ := c.All()
	if c.index >= len(docs) {
		return errors.New("cursor exhausted")
	}
	doc := docs[c.index]
	c.index++

	// Convert to JSON and back to decode into target
	data, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

// ============================================================================
// SingleResult
// ============================================================================

// SingleResult represents a single query result
type SingleResult struct {
	doc Document
	err error
}

// Decode decodes the result into the provided value
func (r *SingleResult) Decode(v interface{}) error {
	if r.err != nil {
		return r.err
	}
	if r.doc == nil {
		return errors.New("no document found")
	}
	data, err := json.Marshal(r.doc)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

// Err returns the error, if any
func (r *SingleResult) Err() error {
	return r.err
}

// ============================================================================
// Collection
// ============================================================================

// Collection represents a MongoDB-compatible collection
type Collection struct {
	db   C.KeraDB
	name string
}

// Name returns the collection name
func (c *Collection) Name() string {
	return c.name
}

// InsertOne inserts a single document
func (c *Collection) InsertOne(doc interface{}) (*InsertOneResult, error) {
	jsonData, err := json.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal document: %w", err)
	}

	cCollection := C.CString(c.name)
	defer C.free(unsafe.Pointer(cCollection))

	cJSON := C.CString(string(jsonData))
	defer C.free(unsafe.Pointer(cJSON))

	cID := C.keradb_insert(c.db, cCollection, cJSON)
	if cID == nil {
		return nil, fmt.Errorf("insert failed: %s", getLastError())
	}
	defer C.keradb_free_string(cID)

	return &InsertOneResult{
		InsertedID: C.GoString(cID),
	}, nil
}

// InsertMany inserts multiple documents
func (c *Collection) InsertMany(docs []interface{}) (*InsertManyResult, error) {
	var insertedIDs []string

	for _, doc := range docs {
		result, err := c.InsertOne(doc)
		if err != nil {
			return nil, err
		}
		insertedIDs = append(insertedIDs, result.InsertedID)
	}

	return &InsertManyResult{
		InsertedIDs: insertedIDs,
	}, nil
}

// FindOne finds a single document matching the filter
func (c *Collection) FindOne(filter M) *SingleResult {
	// Optimize for _id lookup
	if id, ok := filter["_id"].(string); ok && len(filter) == 1 {
		cCollection := C.CString(c.name)
		defer C.free(unsafe.Pointer(cCollection))

		cID := C.CString(id)
		defer C.free(unsafe.Pointer(cID))

		cDoc := C.keradb_find_by_id(c.db, cCollection, cID)
		if cDoc == nil {
			return &SingleResult{doc: nil, err: nil}
		}
		defer C.keradb_free_string(cDoc)

		var doc Document
		if err := json.Unmarshal([]byte(C.GoString(cDoc)), &doc); err != nil {
			return &SingleResult{err: err}
		}
		return &SingleResult{doc: doc}
	}

	// General filter
	cursor := c.Find(filter).Limit(1)
	docs, err := cursor.All()
	if err != nil {
		return &SingleResult{err: err}
	}
	if len(docs) == 0 {
		return &SingleResult{doc: nil}
	}
	return &SingleResult{doc: docs[0]}
}

// Find returns a cursor over documents matching the filter
func (c *Collection) Find(filter M) *Cursor {
	cCollection := C.CString(c.name)
	defer C.free(unsafe.Pointer(cCollection))

	cDocs := C.keradb_find_all(c.db, cCollection, -1, -1)
	if cDocs == nil {
		return NewCursor([]Document{})
	}
	defer C.keradb_free_string(cDocs)

	var docs []Document
	if err := json.Unmarshal([]byte(C.GoString(cDocs)), &docs); err != nil {
		return NewCursor([]Document{})
	}

	// Apply filter
	if filter != nil && len(filter) > 0 {
		var filtered []Document
		for _, doc := range docs {
			if matchesFilter(doc, filter) {
				filtered = append(filtered, doc)
			}
		}
		docs = filtered
	}

	return NewCursor(docs)
}

// UpdateOne updates a single document matching the filter
func (c *Collection) UpdateOne(filter M, update M) (*UpdateResult, error) {
	result := c.FindOne(filter)
	if result.err != nil {
		return nil, result.err
	}
	if result.doc == nil {
		return &UpdateResult{MatchedCount: 0, ModifiedCount: 0}, nil
	}

	updatedDoc := applyUpdate(result.doc, update)
	docID := result.doc.ID()

	// Remove _id from update
	delete(updatedDoc, "_id")

	jsonData, err := json.Marshal(updatedDoc)
	if err != nil {
		return nil, err
	}

	cCollection := C.CString(c.name)
	defer C.free(unsafe.Pointer(cCollection))

	cID := C.CString(docID)
	defer C.free(unsafe.Pointer(cID))

	cJSON := C.CString(string(jsonData))
	defer C.free(unsafe.Pointer(cJSON))

	cResult := C.keradb_update(c.db, cCollection, cID, cJSON)
	if cResult == nil {
		return nil, fmt.Errorf("update failed: %s", getLastError())
	}
	C.keradb_free_string(cResult)

	return &UpdateResult{MatchedCount: 1, ModifiedCount: 1}, nil
}

// UpdateMany updates all documents matching the filter
func (c *Collection) UpdateMany(filter M, update M) (*UpdateResult, error) {
	cursor := c.Find(filter)
	docs, err := cursor.All()
	if err != nil {
		return nil, err
	}

	var modifiedCount int64 = 0
	for _, doc := range docs {
		_, err := c.UpdateOne(M{"_id": doc.ID()}, update)
		if err != nil {
			return nil, err
		}
		modifiedCount++
	}

	return &UpdateResult{
		MatchedCount:  int64(len(docs)),
		ModifiedCount: modifiedCount,
	}, nil
}

// DeleteOne deletes a single document matching the filter
func (c *Collection) DeleteOne(filter M) (*DeleteResult, error) {
	result := c.FindOne(filter)
	if result.err != nil {
		return nil, result.err
	}
	if result.doc == nil {
		return &DeleteResult{DeletedCount: 0}, nil
	}

	cCollection := C.CString(c.name)
	defer C.free(unsafe.Pointer(cCollection))

	cID := C.CString(result.doc.ID())
	defer C.free(unsafe.Pointer(cID))

	deleteResult := C.keradb_delete(c.db, cCollection, cID)

	return &DeleteResult{
		DeletedCount: int64(deleteResult),
	}, nil
}

// DeleteMany deletes all documents matching the filter
func (c *Collection) DeleteMany(filter M) (*DeleteResult, error) {
	cursor := c.Find(filter)
	docs, err := cursor.All()
	if err != nil {
		return nil, err
	}

	var deletedCount int64 = 0
	for _, doc := range docs {
		result, err := c.DeleteOne(M{"_id": doc.ID()})
		if err != nil {
			return nil, err
		}
		deletedCount += result.DeletedCount
	}

	return &DeleteResult{DeletedCount: deletedCount}, nil
}

// CountDocuments counts documents matching the filter
func (c *Collection) CountDocuments(filter M) (int64, error) {
	if filter == nil || len(filter) == 0 {
		cCollection := C.CString(c.name)
		defer C.free(unsafe.Pointer(cCollection))

		count := C.keradb_count(c.db, cCollection)
		return int64(count), nil
	}

	cursor := c.Find(filter)
	docs, err := cursor.All()
	if err != nil {
		return 0, err
	}
	return int64(len(docs)), nil
}

// Drop deletes all documents in the collection
func (c *Collection) Drop() error {
	_, err := c.DeleteMany(M{})
	return err
}

// ============================================================================
// Database
// ============================================================================

// Database represents a KeraDB database
type Database struct {
	db          C.KeraDB
	collections map[string]*Collection
}

// Collection returns a collection by name
func (d *Database) Collection(name string) *Collection {
	if d.collections == nil {
		d.collections = make(map[string]*Collection)
	}
	if coll, ok := d.collections[name]; ok {
		return coll
	}
	coll := &Collection{db: d.db, name: name}
	d.collections[name] = coll
	return coll
}

// ListCollectionNames returns the names of all collections
func (d *Database) ListCollectionNames() ([]string, error) {
	cCollections := C.keradb_list_collections(d.db)
	if cCollections == nil {
		return []string{}, nil
	}
	defer C.keradb_free_string(cCollections)

	var collections [][2]interface{}
	if err := json.Unmarshal([]byte(C.GoString(cCollections)), &collections); err != nil {
		return nil, err
	}

	names := make([]string, len(collections))
	for i, c := range collections {
		names[i] = c[0].(string)
	}
	return names, nil
}

// ============================================================================
// Client
// ============================================================================

// Client is the main KeraDB client (MongoDB-compatible)
type Client struct {
	db       C.KeraDB
	path     string
	database *Database
}

// Connect creates or opens a KeraDB database
func Connect(path string) (*Client, error) {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	// Try to open first, then create if it doesn't exist
	db := C.keradb_open(cPath)
	if db == nil {
		db = C.keradb_create(cPath)
	}

	if db == nil {
		return nil, fmt.Errorf("failed to connect: %s", getLastError())
	}

	return &Client{
		db:       db,
		path:     path,
		database: &Database{db: db},
	}, nil
}

// Create creates a new KeraDB database
func Create(path string) (*Client, error) {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	db := C.keradb_create(cPath)
	if db == nil {
		return nil, fmt.Errorf("failed to create database: %s", getLastError())
	}

	return &Client{
		db:       db,
		path:     path,
		database: &Database{db: db},
	}, nil
}

// Open opens an existing KeraDB database
func Open(path string) (*Client, error) {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	db := C.keradb_open(cPath)
	if db == nil {
		return nil, fmt.Errorf("failed to open database: %s", getLastError())
	}

	return &Client{
		db:       db,
		path:     path,
		database: &Database{db: db},
	}, nil
}

// Database returns the database object
func (c *Client) Database() *Database {
	return c.database
}

// Close closes the database connection
func (c *Client) Close() error {
	if c.db != nil {
		C.keradb_close(c.db)
		c.db = nil
	}
	return nil
}

// Sync flushes all changes to disk
func (c *Client) Sync() error {
	if c.db == nil {
		return errors.New("database is closed")
	}
	C.keradb_sync(c.db)
	return nil
}

// Convenience alias for MongoDB compatibility
var MongoClient = Connect
