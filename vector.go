package keradb

/*
#cgo LDFLAGS: -L${SRCDIR}/../../../target/release -lkeradb
#cgo linux LDFLAGS: -lkeradb -lm -ldl -lpthread
#cgo darwin LDFLAGS: -lkeradb -lm -ldl -lpthread
#cgo windows LDFLAGS: -lkeradb -lws2_32 -luserenv -lbcrypt -lntdll

#include <stdlib.h>

typedef void* KeraDB;

// Vector FFI functions
char* keradb_create_vector_collection(KeraDB db, const char* name, const char* config_json);
char* keradb_list_vector_collections(KeraDB db);
int keradb_drop_vector_collection(KeraDB db, const char* name);
char* keradb_insert_vector(KeraDB db, const char* collection, const char* vector_json, const char* metadata_json);
char* keradb_insert_text(KeraDB db, const char* collection, const char* text, const char* metadata_json);
char* keradb_vector_search(KeraDB db, const char* collection, const char* query_vector_json, int k);
char* keradb_vector_search_text(KeraDB db, const char* collection, const char* query_text, int k);
char* keradb_vector_search_filtered(KeraDB db, const char* collection, const char* query_vector_json, int k, const char* filter_json);
char* keradb_get_vector(KeraDB db, const char* collection, unsigned long long id);
int keradb_delete_vector(KeraDB db, const char* collection, unsigned long long id);
char* keradb_vector_stats(KeraDB db, const char* collection);
void keradb_free_string(char* s);
*/
import "C"
import (
	"encoding/json"
	"fmt"
	"unsafe"
)

// ============================================================================
// Vector Types
// ============================================================================

// VectorID is a unique identifier for a vector document
type VectorID uint64

// Embedding is a vector of float32 values
type Embedding []float32

// Distance metric for vector similarity
type Distance string

const (
	// Cosine distance (default) - range [0, 2] where 0 = identical
	Cosine Distance = "cosine"
	// Euclidean distance (L2 norm)
	Euclidean Distance = "euclidean"
	// DotProduct distance (negative dot product for similarity ranking)
	DotProduct Distance = "dot_product"
	// Manhattan distance (L1 norm)
	Manhattan Distance = "manhattan"
)

// CompressionMode defines how vectors are compressed
type CompressionMode string

const (
	// NoCompression stores full vectors
	NoCompression CompressionMode = "none"
	// DeltaCompression stores sparse differences from neighbors
	DeltaCompression CompressionMode = "delta"
	// QuantizedDelta uses aggressive quantized deltas
	QuantizedDelta CompressionMode = "quantized_delta"
)

// CompressionConfig defines compression parameters
type CompressionConfig struct {
	Mode              CompressionMode `json:"mode,omitempty"`
	SparsityThreshold *float32        `json:"sparsity_threshold,omitempty"`
	MaxDensity        *float32        `json:"max_density,omitempty"`
	AnchorFrequency   *int            `json:"anchor_frequency,omitempty"`
	QuantizationBits  *int            `json:"quantization_bits,omitempty"`
}

// VectorConfig defines configuration for a vector collection
type VectorConfig struct {
	Dimensions      int                `json:"dimensions"`
	Distance        Distance           `json:"distance,omitempty"`
	M               *int               `json:"m,omitempty"`                // HNSW M parameter (default 16)
	EfConstruction  *int               `json:"ef_construction,omitempty"`  // Build quality (default 200)
	EfSearch        *int               `json:"ef_search,omitempty"`        // Query quality (default 50)
	LazyEmbedding   *bool              `json:"lazy_embedding,omitempty"`   // Enable lazy recomputation
	EmbeddingModel  *string            `json:"embedding_model,omitempty"`  // Model name
	Compression     *CompressionConfig `json:"compression,omitempty"`
}

// VectorDocument represents a document in a vector collection
type VectorDocument struct {
	ID        VectorID               `json:"id"`
	Embedding *Embedding             `json:"embedding,omitempty"`
	Text      *string                `json:"text,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// VectorSearchResult represents a search result with score
type VectorSearchResult struct {
	Document VectorDocument `json:"document"`
	Score    float32        `json:"score"`
	Rank     int            `json:"rank"`
}

// VectorCollectionStats provides statistics about a vector collection
type VectorCollectionStats struct {
	VectorCount    int              `json:"vector_count"`
	Dimensions     int              `json:"dimensions"`
	Distance       Distance         `json:"distance"`
	MemoryUsage    int64            `json:"memory_usage"`
	LayerCount     int              `json:"layer_count"`
	LazyEmbedding  bool             `json:"lazy_embedding"`
	Compression    *CompressionMode `json:"compression,omitempty"`
	AnchorCount    *int             `json:"anchor_count,omitempty"`
	DeltaCount     *int             `json:"delta_count,omitempty"`
}

// MetadataFilter represents a filter condition for metadata fields
type MetadataFilter struct {
	Field     string      `json:"field"`
	Condition string      `json:"condition"` // "eq", "ne", "gt", "gte", "lt", "lte", "in", "not_in", "contains", "starts_with", "ends_with"
	Value     interface{} `json:"value"`
}

// ============================================================================
// Vector Configuration Builders
// ============================================================================

// NewVectorConfig creates a new vector configuration with required dimensions
func NewVectorConfig(dimensions int) *VectorConfig {
	return &VectorConfig{
		Dimensions: dimensions,
		Distance:   Cosine, // default
	}
}

// WithDistance sets the distance metric
func (vc *VectorConfig) WithDistance(distance Distance) *VectorConfig {
	vc.Distance = distance
	return vc
}

// WithM sets the HNSW M parameter (number of connections per node)
func (vc *VectorConfig) WithM(m int) *VectorConfig {
	vc.M = &m
	return vc
}

// WithEfConstruction sets the ef_construction parameter (build quality)
func (vc *VectorConfig) WithEfConstruction(ef int) *VectorConfig {
	vc.EfConstruction = &ef
	return vc
}

// WithEfSearch sets the ef_search parameter (query quality)
func (vc *VectorConfig) WithEfSearch(ef int) *VectorConfig {
	vc.EfSearch = &ef
	return vc
}

// WithLazyEmbedding enables lazy embedding mode with a model name
func (vc *VectorConfig) WithLazyEmbedding(model string) *VectorConfig {
	t := true
	vc.LazyEmbedding = &t
	vc.EmbeddingModel = &model
	return vc
}

// WithCompression sets compression configuration
func (vc *VectorConfig) WithCompression(config CompressionConfig) *VectorConfig {
	vc.Compression = &config
	return vc
}

// WithDeltaCompression enables delta compression with default settings
func (vc *VectorConfig) WithDeltaCompression() *VectorConfig {
	mode := DeltaCompression
	return vc.WithCompression(CompressionConfig{Mode: mode})
}

// WithQuantizedCompression enables quantized delta compression
func (vc *VectorConfig) WithQuantizedCompression() *VectorConfig {
	mode := QuantizedDelta
	return vc.WithCompression(CompressionConfig{Mode: mode})
}

// ============================================================================
// Vector Collection Operations
// ============================================================================

// CreateVectorCollection creates a new vector collection
func (c *Client) CreateVectorCollection(name string, config *VectorConfig) error {
	configJSON, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))

	cConfig := C.CString(string(configJSON))
	defer C.free(unsafe.Pointer(cConfig))

	cResult := C.keradb_create_vector_collection(c.db, cName, cConfig)
	if cResult == nil {
		return fmt.Errorf("create vector collection failed: %s", getLastError())
	}
	defer C.keradb_free_string(cResult)

	return nil
}

// ListVectorCollections returns a list of vector collections with their sizes
func (c *Client) ListVectorCollections() ([]struct {
	Name  string
	Count int
}, error) {
	cResult := C.keradb_list_vector_collections(c.db)
	if cResult == nil {
		return nil, fmt.Errorf("list vector collections failed: %s", getLastError())
	}
	defer C.keradb_free_string(cResult)

	var collections []struct {
		Name  string
		Count int
	}
	if err := json.Unmarshal([]byte(C.GoString(cResult)), &collections); err != nil {
		return nil, fmt.Errorf("failed to unmarshal collections: %w", err)
	}

	return collections, nil
}

// DropVectorCollection deletes a vector collection
func (c *Client) DropVectorCollection(name string) (bool, error) {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))

	result := C.keradb_drop_vector_collection(c.db, cName)
	return result != 0, nil
}

// InsertVector inserts a vector with optional metadata
func (c *Client) InsertVector(collection string, embedding Embedding, metadata M) (VectorID, error) {
	vectorJSON, err := json.Marshal(embedding)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal vector: %w", err)
	}

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	cCollection := C.CString(collection)
	defer C.free(unsafe.Pointer(cCollection))

	cVector := C.CString(string(vectorJSON))
	defer C.free(unsafe.Pointer(cVector))

	cMetadata := C.CString(string(metadataJSON))
	defer C.free(unsafe.Pointer(cMetadata))

	cResult := C.keradb_insert_vector(c.db, cCollection, cVector, cMetadata)
	if cResult == nil {
		return 0, fmt.Errorf("insert vector failed: %s", getLastError())
	}
	defer C.keradb_free_string(cResult)

	var id VectorID
	if err := json.Unmarshal([]byte(C.GoString(cResult)), &id); err != nil {
		return 0, fmt.Errorf("failed to unmarshal ID: %w", err)
	}

	return id, nil
}

// InsertText inserts text with optional metadata (requires embedding provider)
func (c *Client) InsertText(collection string, text string, metadata M) (VectorID, error) {
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	cCollection := C.CString(collection)
	defer C.free(unsafe.Pointer(cCollection))

	cText := C.CString(text)
	defer C.free(unsafe.Pointer(cText))

	cMetadata := C.CString(string(metadataJSON))
	defer C.free(unsafe.Pointer(cMetadata))

	cResult := C.keradb_insert_text(c.db, cCollection, cText, cMetadata)
	if cResult == nil {
		return 0, fmt.Errorf("insert text failed: %s", getLastError())
	}
	defer C.keradb_free_string(cResult)

	var id VectorID
	if err := json.Unmarshal([]byte(C.GoString(cResult)), &id); err != nil {
		return 0, fmt.Errorf("failed to unmarshal ID: %w", err)
	}

	return id, nil
}

// VectorSearch performs a vector similarity search
func (c *Client) VectorSearch(collection string, queryVector Embedding, k int) ([]VectorSearchResult, error) {
	vectorJSON, err := json.Marshal(queryVector)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query vector: %w", err)
	}

	cCollection := C.CString(collection)
	defer C.free(unsafe.Pointer(cCollection))

	cVector := C.CString(string(vectorJSON))
	defer C.free(unsafe.Pointer(cVector))

	cResult := C.keradb_vector_search(c.db, cCollection, cVector, C.int(k))
	if cResult == nil {
		return nil, fmt.Errorf("vector search failed: %s", getLastError())
	}
	defer C.keradb_free_string(cResult)

	var results []VectorSearchResult
	if err := json.Unmarshal([]byte(C.GoString(cResult)), &results); err != nil {
		return nil, fmt.Errorf("failed to unmarshal results: %w", err)
	}

	return results, nil
}

// VectorSearchText performs a text-based similarity search (requires embedding provider)
func (c *Client) VectorSearchText(collection string, queryText string, k int) ([]VectorSearchResult, error) {
	cCollection := C.CString(collection)
	defer C.free(unsafe.Pointer(cCollection))

	cText := C.CString(queryText)
	defer C.free(unsafe.Pointer(cText))

	cResult := C.keradb_vector_search_text(c.db, cCollection, cText, C.int(k))
	if cResult == nil {
		return nil, fmt.Errorf("vector search text failed: %s", getLastError())
	}
	defer C.keradb_free_string(cResult)

	var results []VectorSearchResult
	if err := json.Unmarshal([]byte(C.GoString(cResult)), &results); err != nil {
		return nil, fmt.Errorf("failed to unmarshal results: %w", err)
	}

	return results, nil
}

// VectorSearchFiltered performs a filtered vector similarity search
func (c *Client) VectorSearchFiltered(collection string, queryVector Embedding, k int, filter MetadataFilter) ([]VectorSearchResult, error) {
	vectorJSON, err := json.Marshal(queryVector)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query vector: %w", err)
	}

	filterJSON, err := json.Marshal(filter)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal filter: %w", err)
	}

	cCollection := C.CString(collection)
	defer C.free(unsafe.Pointer(cCollection))

	cVector := C.CString(string(vectorJSON))
	defer C.free(unsafe.Pointer(cVector))

	cFilter := C.CString(string(filterJSON))
	defer C.free(unsafe.Pointer(cFilter))

	cResult := C.keradb_vector_search_filtered(c.db, cCollection, cVector, C.int(k), cFilter)
	if cResult == nil {
		return nil, fmt.Errorf("vector search filtered failed: %s", getLastError())
	}
	defer C.keradb_free_string(cResult)

	var results []VectorSearchResult
	if err := json.Unmarshal([]byte(C.GoString(cResult)), &results); err != nil {
		return nil, fmt.Errorf("failed to unmarshal results: %w", err)
	}

	return results, nil
}

// GetVector retrieves a vector document by ID
func (c *Client) GetVector(collection string, id VectorID) (*VectorDocument, error) {
	cCollection := C.CString(collection)
	defer C.free(unsafe.Pointer(cCollection))

	cResult := C.keradb_get_vector(c.db, cCollection, C.ulonglong(id))
	if cResult == nil {
		return nil, nil // Not found
	}
	defer C.keradb_free_string(cResult)

	var doc VectorDocument
	if err := json.Unmarshal([]byte(C.GoString(cResult)), &doc); err != nil {
		return nil, fmt.Errorf("failed to unmarshal document: %w", err)
	}

	return &doc, nil
}

// DeleteVector deletes a vector document by ID
func (c *Client) DeleteVector(collection string, id VectorID) (bool, error) {
	cCollection := C.CString(collection)
	defer C.free(unsafe.Pointer(cCollection))

	result := C.keradb_delete_vector(c.db, cCollection, C.ulonglong(id))
	return result != 0, nil
}

// VectorStats returns statistics about a vector collection
func (c *Client) VectorStats(collection string) (*VectorCollectionStats, error) {
	cCollection := C.CString(collection)
	defer C.free(unsafe.Pointer(cCollection))

	cResult := C.keradb_vector_stats(c.db, cCollection)
	if cResult == nil {
		return nil, fmt.Errorf("vector stats failed: %s", getLastError())
	}
	defer C.keradb_free_string(cResult)

	var stats VectorCollectionStats
	if err := json.Unmarshal([]byte(C.GoString(cResult)), &stats); err != nil {
		return nil, fmt.Errorf("failed to unmarshal stats: %w", err)
	}

	return &stats, nil
}
