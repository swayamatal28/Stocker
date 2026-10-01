[CmdletBinding()]
param([string]$BenchTime = "2s")
$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
Push-Location $root
try {
    $env:GOCACHE = Join-Path $root ".cache\go-build"
    $env:GOMODCACHE = Join-Path $root ".cache\go\pkg\mod"
    go test ./internal/httpapi -run '^TestHubFanoutLoadPreservesUserIsolation$' -count=1
    if ($LASTEXITCODE -ne 0) { throw "SSE isolation load test failed" }
    go test ./internal/httpapi -run '^$' -bench '^BenchmarkHubFanout$' -benchmem -benchtime $BenchTime
    if ($LASTEXITCODE -ne 0) { throw "SSE benchmark failed" }
    go test ./internal/ingest -run '^$' -bench '^BenchmarkProcessorBatch1000$' -benchmem -benchtime $BenchTime
    if ($LASTEXITCODE -ne 0) { throw "ingestion benchmark failed" }
}
finally { Pop-Location }
