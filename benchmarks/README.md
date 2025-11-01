# Hyundai Logger Benchmarks

This directory contains comprehensive benchmark tests for the Hyundai Logger application's critical performance components.

## Running Benchmarks

### Run All Benchmarks

```bash
go test -bench=. -benchmem ./benchmarks/
```

### Run Specific Benchmark Suite

```bash
# Cache benchmarks only
go test -bench=BenchmarkCache -benchmem ./benchmarks/

# Circuit breaker benchmarks only
go test -bench=BenchmarkCircuitBreaker -benchmem ./benchmarks/

# Retry benchmarks only
go test -bench=BenchmarkRetry -benchmem ./benchmarks/

# API benchmarks only
go test -bench=BenchmarkAPI -benchmem ./benchmarks/

# Metrics benchmarks only
go test -bench=BenchmarkMetrics -benchmem ./benchmarks/
```

### Run Specific Benchmark

```bash
go test -bench=BenchmarkCacheGet -benchmem ./benchmarks/
```

### Advanced Options

```bash
# Run benchmarks for 10 seconds each
go test -bench=. -benchtime=10s ./benchmarks/

# Run with CPU profiling
go test -bench=. -cpuprofile=cpu.prof ./benchmarks/
go tool pprof cpu.prof

# Run with memory profiling
go test -bench=. -memprofile=mem.prof ./benchmarks/
go tool pprof mem.prof

# Run benchmarks multiple times for statistical significance
go test -bench=. -count=10 ./benchmarks/

# Compare benchmarks before/after changes
go test -bench=. ./benchmarks/ > old.txt
# Make changes...
go test -bench=. ./benchmarks/ > new.txt
benchcmp old.txt new.txt  # Requires: go get golang.org/x/tools/cmd/benchcmp
```

## Benchmark Suites

### Cache Benchmarks (`cache_bench_test.go`)

Tests the in-memory cache implementation performance:

- **BenchmarkCacheSet**: Write operations
- **BenchmarkCacheGet**: Read operations (cache hits)
- **BenchmarkCacheGetMiss**: Read operations (cache misses)
- **BenchmarkCacheGetExpired**: Expired entry cleanup
- **BenchmarkCacheSetParallel**: Concurrent write operations
- **BenchmarkCacheGetParallel**: Concurrent read operations
- **BenchmarkCacheMixedOperations**: Realistic 80% read / 20% write workload
- **BenchmarkCacheCleanExpired**: Bulk expired entry removal
- **BenchmarkCacheGetStats**: Statistics gathering overhead

**Expected Performance Targets:**
- Cache Get: < 100 ns/op
- Cache Set: < 500 ns/op
- Parallel Get: < 200 ns/op

### Circuit Breaker Benchmarks (`circuitbreaker_bench_test.go`)

Tests the circuit breaker pattern implementation:

- **BenchmarkCircuitBreakerSuccess**: Successful operations (closed state)
- **BenchmarkCircuitBreakerFailure**: Failed operations
- **BenchmarkCircuitBreakerOpen**: Operations when circuit is open (fast-fail)
- **BenchmarkCircuitBreakerParallel**: Concurrent operations
- **BenchmarkCircuitBreakerGetState**: State checking overhead
- **BenchmarkCircuitBreakerGetStats**: Statistics retrieval
- **BenchmarkCircuitBreakerMixedWorkload**: Realistic 90% success rate

**Expected Performance Targets:**
- Success path: < 500 ns/op
- Open circuit (fast-fail): < 100 ns/op
- State check: < 50 ns/op

### Retry Benchmarks (`retry_bench_test.go`)

Tests the retry mechanism with exponential backoff:

- **BenchmarkRetrySuccess**: Successful operations (no retries)
- **BenchmarkRetryFailure**: Operations that always fail (max retries)
- **BenchmarkRetryEventualSuccess**: Operations succeeding on 2nd attempt
- **BenchmarkRetryWithResult**: DoWithResult method
- **BenchmarkRetryWithResultFailure**: DoWithResult with failures
- **BenchmarkRetryParallel**: Concurrent retry operations
- **BenchmarkRetryBackoff**: Exponential backoff calculation
- **BenchmarkRetryContextCancellation**: Context cancellation handling

**Expected Performance Targets:**
- Success (no retry): < 1 µs/op
- With retries: depends on backoff configuration

### API Benchmarks (`api_bench_test.go`)

Tests API client operations:

- **BenchmarkAPIClientCreation**: Client instantiation overhead
- **BenchmarkJSONMarshal**: JSON encoding performance
- **BenchmarkJSONUnmarshal**: JSON decoding performance
- **BenchmarkHTTPServerResponse**: Mock server response time
- **BenchmarkContextCreation**: Context creation overhead
- **BenchmarkStructAllocation**: VehicleStatus allocation

**Expected Performance Targets:**
- Client creation: < 10 µs/op
- JSON Marshal: < 2 µs/op
- JSON Unmarshal: < 5 µs/op

### Metrics Benchmarks (`metrics_bench_test.go`)

Tests the metrics collection system:

- **BenchmarkMetricsCollectorRecordPoll**: Poll recording
- **BenchmarkMetricsCollectorRecordError**: Error recording
- **BenchmarkMetricsCollectorRecordAPICall**: API call recording
- **BenchmarkMetricsCollectorUpdateVehicleCount**: Vehicle count updates
- **BenchmarkMetricsCollectorGetStats**: Statistics retrieval
- **BenchmarkMetricsCollectorParallelRecording**: Concurrent recording
- **BenchmarkMetricsCollectorMixedOperations**: Realistic workload simulation
- **BenchmarkMetricsCalculation**: Statistics calculation overhead

**Expected Performance Targets:**
- Record operation: < 500 ns/op
- GetStats: < 5 µs/op
- Parallel recording: < 1 µs/op

## Understanding Benchmark Results

### Output Format

```
BenchmarkCacheGet-8     10000000    125 ns/op    32 B/op    2 allocs/op
```

- `BenchmarkCacheGet-8`: Benchmark name with GOMAXPROCS value
- `10000000`: Number of iterations
- `125 ns/op`: Average time per operation
- `32 B/op`: Bytes allocated per operation
- `2 allocs/op`: Number of allocations per operation

### Performance Indicators

**Good Performance:**
- Low ns/op values
- Zero or minimal allocations
- Consistent results across multiple runs

**Areas for Improvement:**
- High allocation counts (> 10 allocs/op for simple operations)
- High memory usage per operation
- Large variance in timing across runs

## Continuous Performance Monitoring

### Baseline Creation

Create a baseline for comparison:

```bash
go test -bench=. -benchmem ./benchmarks/ | tee baseline.txt
```

### Regression Testing

After making changes, compare against baseline:

```bash
go test -bench=. -benchmem ./benchmarks/ | tee new.txt
benchcmp baseline.txt new.txt
```

### CI/CD Integration

Add to your CI pipeline:

```bash
# Run benchmarks and fail if performance degrades > 20%
go test -bench=. -benchmem ./benchmarks/ > new.txt
benchcmp baseline.txt new.txt || echo "Performance regression detected"
```

## Performance Optimization Tips

### Cache Optimization
- Minimize lock contention with RWMutex
- Use appropriate TTL values
- Implement background cleanup for expired entries
- Consider cache size limits for memory-constrained environments

### Circuit Breaker Optimization
- Fast-fail path should have minimal overhead
- State transitions should be thread-safe but efficient
- Avoid unnecessary allocations in hot paths

### Retry Optimization
- Use minimal initial delays for benchmarks
- Implement backoff capping to prevent excessive delays
- Consider context cancellation for long-running operations

### API Client Optimization
- Reuse HTTP clients and connections
- Enable connection pooling
- Use efficient JSON encoding/decoding
- Implement request/response caching

### Metrics Optimization
- Use atomic operations for counters
- Minimize lock contention for concurrent updates
- Calculate derived metrics lazily
- Consider sampling for high-frequency events

## Profiling

### CPU Profiling

```bash
go test -bench=BenchmarkCacheGet -cpuprofile=cpu.prof ./benchmarks/
go tool pprof -http=:8080 cpu.prof
```

### Memory Profiling

```bash
go test -bench=BenchmarkCacheSet -memprofile=mem.prof ./benchmarks/
go tool pprof -http=:8080 mem.prof
```

### Trace Analysis

```bash
go test -bench=BenchmarkCircuitBreaker -trace=trace.out ./benchmarks/
go tool trace trace.out
```

## Best Practices

1. **Run Multiple Times**: Use `-count=10` for statistical significance
2. **Consistent Environment**: Run on same hardware for comparisons
3. **Isolate Tests**: Close other applications during benchmarking
4. **Warm-up**: Some benchmarks include pre-population for realistic scenarios
5. **Document Baselines**: Keep performance baselines in version control
6. **Regular Monitoring**: Run benchmarks regularly to catch regressions early

## Contributing

When adding new benchmarks:

1. Follow existing naming conventions: `Benchmark<Component><Operation>`
2. Include parallel variants for concurrent operations: `Benchmark<Name>Parallel`
3. Document expected performance targets in this README
4. Add realistic workload scenarios when possible
5. Keep benchmarks focused and simple
6. Avoid external dependencies (databases, networks) when possible

## References

- [Go Benchmark Documentation](https://pkg.go.dev/testing#hdr-Benchmarks)
- [Profiling Go Programs](https://go.dev/blog/pprof)
- [Benchmarking Best Practices](https://dave.cheney.net/2013/06/30/how-to-write-benchmarks-in-go)
