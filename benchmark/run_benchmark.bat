@echo off
setlocal

echo ======================================
echo KeraDB vs SQLite Benchmark Suite
echo ======================================
echo.

REM Build the KeraDB library first
echo Building KeraDB library...
cd ..\..\..
cargo build --release
if errorlevel 1 (
    echo Failed to build KeraDB library
    exit /b 1
)
cd sdks\go\benchmark

REM Set library path
set PATH=%PATH%;%cd%\..\..\..\target\release

echo Library built successfully
echo.

REM Download dependencies
echo Downloading Go dependencies...
go mod download
echo Dependencies ready
echo.

echo ======================================
echo Running Benchmarks...
echo ======================================
echo.

REM Document operations
echo --- Document Operations ---
go test -bench="BenchmarkKeraDB_Insert$" -benchmem -benchtime=1000x
go test -bench="BenchmarkSQLite_Insert$" -benchmem -benchtime=1000x
echo.

go test -bench="BenchmarkKeraDB_InsertBatch" -benchmem -benchtime=100x
go test -bench="BenchmarkSQLite_InsertBatch" -benchmem -benchtime=100x
echo.

go test -bench="BenchmarkKeraDB_FindByID" -benchmem -benchtime=10000x
go test -bench="BenchmarkSQLite_FindByID" -benchmem -benchtime=10000x
echo.

go test -bench="BenchmarkKeraDB_Update" -benchmem -benchtime=1000x
go test -bench="BenchmarkSQLite_Update" -benchmem -benchtime=1000x
echo.

REM Vector operations
echo --- Vector Operations ---
go test -bench="BenchmarkKeraDB_VectorInsert$" -benchmem -benchtime=1000x
go test -bench="BenchmarkSQLite_VectorInsert" -benchmem -benchtime=1000x
echo.

go test -bench="BenchmarkKeraDB_VectorSearch$" -benchmem -benchtime=100x
go test -bench="BenchmarkSQLite_VectorSearch" -benchmem -benchtime=10x
echo.

REM Compression benchmarks
echo --- Vector With Compression ---
go test -bench="BenchmarkKeraDB_VectorInsert_WithCompression" -benchmem -benchtime=1000x
go test -bench="BenchmarkKeraDB_VectorSearch_WithCompression" -benchmem -benchtime=100x
echo.

echo ======================================
echo Benchmark Complete!
echo ======================================

endlocal
