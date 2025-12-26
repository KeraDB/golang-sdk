#!/bin/bash

echo "======================================"
echo "KeraDB vs SQLite Benchmark Suite"
echo "======================================"
echo ""

# Build the KeraDB library first
echo "Building KeraDB library..."
cd ../../..
cargo build --release
if [ $? -ne 0 ]; then
    echo "Failed to build KeraDB library"
    exit 1
fi
cd sdks/go/benchmark

# Set library path
case "$(uname -s)" in
    Linux*)
        export LD_LIBRARY_PATH=$LD_LIBRARY_PATH:$(pwd)/../../../target/release
        ;;
    Darwin*)
        export DYLD_LIBRARY_PATH=$DYLD_LIBRARY_PATH:$(pwd)/../../../target/release
        ;;
esac

echo "✓ Library built"
echo ""

# Download dependencies
echo "Downloading Go dependencies..."
go mod download
echo "✓ Dependencies ready"
echo ""

# Run benchmarks
echo "======================================"
echo "Running Benchmarks..."
echo "======================================"
echo ""

# Document operations
echo "--- Document Operations ---"
go test -bench="BenchmarkKeraDB_Insert$" -benchmem -benchtime=1000x
go test -bench="BenchmarkSQLite_Insert$" -benchmem -benchtime=1000x
echo ""

go test -bench="BenchmarkKeraDB_InsertBatch" -benchmem -benchtime=100x
go test -bench="BenchmarkSQLite_InsertBatch" -benchmem -benchtime=100x
echo ""

go test -bench="BenchmarkKeraDB_FindByID" -benchmem -benchtime=10000x
go test -bench="BenchmarkSQLite_FindByID" -benchmem -benchtime=10000x
echo ""

go test -bench="BenchmarkKeraDB_Update" -benchmem -benchtime=1000x
go test -bench="BenchmarkSQLite_Update" -benchmem -benchtime=1000x
echo ""

# Vector operations
echo "--- Vector Operations ---"
go test -bench="BenchmarkKeraDB_VectorInsert$" -benchmem -benchtime=1000x
go test -bench="BenchmarkSQLite_VectorInsert" -benchmem -benchtime=1000x
echo ""

go test -bench="BenchmarkKeraDB_VectorSearch$" -benchmem -benchtime=100x
go test -bench="BenchmarkSQLite_VectorSearch" -benchmem -benchtime=10x
echo ""

# Compression benchmarks
echo "--- Vector With Compression ---"
go test -bench="BenchmarkKeraDB_VectorInsert_WithCompression" -benchmem -benchtime=1000x
go test -bench="BenchmarkKeraDB_VectorSearch_WithCompression" -benchmem -benchtime=100x
echo ""

echo "======================================"
echo "Benchmark Complete!"
echo "======================================"
