# KeraDB vs SQLite Benchmark

Comprehensive benchmark suite comparing KeraDB against SQLite for both document operations and vector search.

## What's Tested

### Document Operations
- **Insert**: Single document insertion
- **Insert Batch**: Batch insertion (100 documents)
- **Find By ID**: Document retrieval by primary key
- **Update**: Single document update

### Vector Operations
- **Vector Insert**: Insert 128-dimensional vectors
- **Vector Search**: HNSW-based approximate nearest neighbor search (k=10)
- **Vector Search (SQLite)**: Naive linear scan with cosine similarity
- **Vector with Compression**: Delta compression performance

## Running the Benchmarks

### Prerequisites

1. Build the KeraDB native library:
```bash
cd ../../../
cargo build --release
```

2. Ensure Go 1.21+ is installed

### Run All Benchmarks

**Linux/macOS:**
```bash
chmod +x run_benchmark.sh
./run_benchmark.sh
```

**Windows:**
```cmd
run_benchmark.bat
```

### Run Individual Benchmarks

```bash
cd sdks/go/benchmark

# Document operations
go test -bench=BenchmarkKeraDB_Insert -benchmem
go test -bench=BenchmarkSQLite_Insert -benchmem

# Vector search
go test -bench=BenchmarkKeraDB_VectorSearch -benchmem
go test -bench=BenchmarkSQLite_VectorSearch -benchmem

# All benchmarks with custom iterations
go test -bench=. -benchmem -benchtime=1000x
```

## Benchmark Configuration

```go
const (
    numDocs         = 10000  // Documents for setup
    batchSize       = 100    // Batch insert size
    vectorDimension = 128    // Vector dimensions
    numVectors      = 5000   // Vectors for search tests
)
```

## Expected Results

### Document Operations

KeraDB should show:
- **Faster single inserts**: Optimized for embedded use
- **Faster batch inserts**: Efficient bulk operations
- **Faster ID lookups**: Native key-value optimization
- **Faster updates**: In-place update support

### Vector Operations

KeraDB should show:
- **Similar insert speed**: Both store JSON/binary
- **100-1000x faster search**: HNSW vs naive linear scan
- **Lower memory usage**: With delta compression enabled

### Compression

With delta compression enabled:
- **85-95% storage reduction**: Sparse delta encoding
- **Minimal search overhead**: ~5-10% slower vs uncompressed
- **Faster insert**: Less data to write

## Understanding Output

```
BenchmarkKeraDB_Insert-8         1000     12345 ns/op     1024 B/op     15 allocs/op
                        │        │        │               │             │
                        │        │        │               │             └─ Allocations per op
                        │        │        │               └─────────────── Bytes allocated per op
                        │        │        └─────────────────────────────── Nanoseconds per op
                        │        └──────────────────────────────────────── Iterations run
                        └───────────────────────────────────────────────── CPU cores used
```

Lower ns/op, B/op, and allocs/op are better.

## Interpreting Results

### Document Operations
- **Insert performance**: KeraDB optimized for embedded use cases
- **Query performance**: Both are fast for ID lookups (indexed)
- **Batch operations**: KeraDB's batch API reduces overhead

### Vector Search
- **Search latency**: KeraDB HNSW dramatically faster than SQLite linear scan
- **Scalability**: HNSW is O(log n), linear scan is O(n)
- **Quality**: HNSW provides >95% recall at 10x-100x speedup

### Memory Usage
- **SQLite**: Stores full vectors as JSON text
- **KeraDB**: Binary storage with optional compression
- **Compression**: 85-95% reduction with delta encoding

## Notes

- SQLite vector search is intentionally naive (no index) for comparison
- Real applications would use SQLite with vector extensions (pgvector-like)
- KeraDB's HNSW is built-in and optimized for embedded scenarios
- Benchmarks use random data; real-world performance varies by use case

## Troubleshooting

**Library not found errors:**
```bash
# Linux/macOS
export LD_LIBRARY_PATH=$LD_LIBRARY_PATH:$(pwd)/../../../target/release
export DYLD_LIBRARY_PATH=$DYLD_LIBRARY_PATH:$(pwd)/../../../target/release

# Windows
set PATH=%PATH%;%cd%\..\..\..\target\release
```

**Go module errors:**
```bash
go mod download
go mod tidy
```
