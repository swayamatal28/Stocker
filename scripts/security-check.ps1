[CmdletBinding()]
param([switch]$RequireExternalTools)
$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
$safeRoot = $root -replace '\\', '/'
Push-Location $root
try {
    $env:GOCACHE = Join-Path $root ".cache\go-build"
    $env:GOMODCACHE = Join-Path $root ".cache\go\pkg\mod"
    go vet ./apps/api ./internal/... ./workers/...
    if ($LASTEXITCODE -ne 0) { throw "go vet failed" }
    npm.cmd --prefix apps/web audit --omit=dev
    if ($LASTEXITCODE -ne 0) { throw "npm production dependency audit failed" }
    $trackedSecrets = @(git -c "safe.directory=$safeRoot" ls-files | Where-Object { $_ -match '(^|/)(\.env|id_rsa|.*\.(pem|p12|pfx))$' })
    if ($trackedSecrets.Count -gt 0) { throw "Secret-like files are tracked. File names are intentionally suppressed." }
    $matches = @(git -c "safe.directory=$safeRoot" grep -I -l -E '(BEGIN (RSA|OPENSSH|EC) PRIVATE KEY|AIza[0-9A-Za-z_-]{30,}|sk-[0-9A-Za-z]{20,})' -- ':!scripts/security-check.ps1' 2>$null)
    if ($matches.Count -gt 0) { throw "Potential secret material detected. Matching contents are intentionally suppressed." }
    if (Get-Command govulncheck -ErrorAction SilentlyContinue) {
        govulncheck ./...
        if ($LASTEXITCODE -ne 0) { throw "govulncheck failed" }
    } elseif ($RequireExternalTools) {
        go run golang.org/x/vuln/cmd/govulncheck@v1.8.0 ./...
        if ($LASTEXITCODE -ne 0) { throw "govulncheck failed" }
    } else { Write-Warning "govulncheck not installed; rerun CI with -RequireExternalTools" }
    & "$PSScriptRoot\generate-sbom.ps1"
}
finally { Pop-Location }
