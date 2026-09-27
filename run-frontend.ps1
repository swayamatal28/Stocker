[CmdletBinding()]
param()

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $MyInvocation.MyCommand.Path
$web = Join-Path $root "apps\web"
$npm = Get-Command npm.cmd -ErrorAction SilentlyContinue
if (-not $npm) { throw "Node.js/npm is not installed or npm.cmd is not available in PATH." }

Push-Location $web
try {
    if (-not (Test-Path -LiteralPath (Join-Path $web "node_modules"))) {
        Write-Host "Installing frontend dependencies..."
        & $npm.Source ci
        if ($LASTEXITCODE -ne 0) { throw "Frontend dependency installation failed." }
    }
    Write-Host "Frontend is starting at http://localhost:5173"
    & $npm.Source run dev
}
finally {
    Pop-Location
}
