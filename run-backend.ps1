[CmdletBinding()]
param([switch]$CheckOnly)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $MyInvocation.MyCommand.Path
$runtime = Join-Path $root ".cache\runtime"
$bin = Join-Path $runtime "bin"
$logs = Join-Path $runtime "logs"
$mongoData = Join-Path $root "data\db"

function Import-DotEnv([string]$Path) {
    if (-not (Test-Path -LiteralPath $Path -PathType Leaf)) {
        throw ".env was not found at $Path"
    }
    foreach ($line in Get-Content -LiteralPath $Path) {
        $trimmed = $line.Trim()
        if (-not $trimmed -or $trimmed.StartsWith("#")) { continue }
        if ($trimmed -notmatch '^([A-Za-z_][A-Za-z0-9_]*)=(.*)$') { continue }
        $name = $matches[1]
        $value = $matches[2].Trim().Trim('"').Trim("'")
        [Environment]::SetEnvironmentVariable($name, $value, "Process")
    }
}

function Test-LocalPort([int]$Port) {
    $client = [Net.Sockets.TcpClient]::new()
    try {
        $connection = $client.ConnectAsync("127.0.0.1", $Port)
        return $connection.Wait(350) -and $client.Connected
    }
    catch { return $false }
    finally { $client.Dispose() }
}

function Wait-LocalPort([int]$Port, [string]$Name, [int]$Seconds = 20) {
    $deadline = (Get-Date).AddSeconds($Seconds)
    while ((Get-Date) -lt $deadline) {
        if (Test-LocalPort $Port) { return }
        Start-Sleep -Milliseconds 300
    }
    throw "$Name did not begin listening on port $Port within $Seconds seconds. Check $logs."
}

function Start-HiddenProcess([string]$FilePath, [string[]]$Arguments, [string]$Name) {
    $stdout = Join-Path $logs "$Name.out.log"
    $stderr = Join-Path $logs "$Name.err.log"
    $options = @{ FilePath = $FilePath; WorkingDirectory = $root; WindowStyle = "Hidden"; RedirectStandardOutput = $stdout; RedirectStandardError = $stderr; PassThru = $true }
    if ($Arguments.Count -gt 0) { $options.ArgumentList = $Arguments }
    return Start-Process @options
}

Import-DotEnv (Join-Path $root ".env")
$env:GOCACHE = Join-Path $root ".cache\go-build"
$env:GOMODCACHE = Join-Path $root ".cache\go\pkg\mod"

$go = Get-Command go -ErrorAction SilentlyContinue
$mongod = Get-Command mongod -ErrorAction SilentlyContinue
if (-not $go) { throw "Go is not installed or is not available in PATH." }
if (-not $mongod -and -not (Test-LocalPort 27017)) { throw "mongod is not installed or is not available in PATH." }

if ($CheckOnly) {
    Write-Host "Backend prerequisites and .env loading are valid."
    exit 0
}

New-Item -ItemType Directory -Force -Path $runtime, $bin, $logs, $mongoData | Out-Null

if (-not (Test-LocalPort 27017)) {
    Write-Host "Starting MongoDB on 127.0.0.1:27017..."
    $null = Start-HiddenProcess $mongod.Source @("--dbpath", $mongoData, "--bind_ip", "127.0.0.1", "--port", "27017") "mongodb"
    Wait-LocalPort 27017 "MongoDB"
} else {
    Write-Host "MongoDB is already running on port 27017."
}

$redisAvailable = Test-LocalPort 6379
if (-not $redisAvailable) {
    $redisServer = Get-Command redis-server -ErrorAction SilentlyContinue
    if ($redisServer) {
        Write-Host "Starting Redis on 127.0.0.1:6379..."
        $null = Start-HiddenProcess $redisServer.Source @("--bind", "127.0.0.1", "--port", "6379") "redis"
        Wait-LocalPort 6379 "Redis"
        $redisAvailable = $true
    }
}
if (-not $redisAvailable -and $env:REDIS_REQUIRED -eq "true") {
    throw "Redis is required by .env but redis-server is unavailable. Install/start Redis and rerun this command."
}
if (-not $redisAvailable) {
    Write-Warning "Redis is unavailable. The API will run, but ingestion, analysis queues and distributed live updates are disabled."
}

Write-Host "Building backend executables..."
$targets = [ordered]@{
    "stocker-api.exe"         = "./apps/api"
    "stocker-maintenance.exe" = "./workers/maintenance"
}
if ($redisAvailable) {
    $targets["stocker-ingestion.exe"] = "./workers/ingestion"
    $targets["stocker-analysis.exe"] = "./workers/analysis"
}
foreach ($target in $targets.GetEnumerator()) {
    $output = Join-Path $bin $target.Key
    & $go.Source build -trimpath -o $output $target.Value
    if ($LASTEXITCODE -ne 0) { throw "Failed to build $($target.Value)" }
}

$children = New-Object System.Collections.Generic.List[System.Diagnostics.Process]
try {
    $children.Add((Start-HiddenProcess (Join-Path $bin "stocker-maintenance.exe") @() "maintenance"))
    if ($redisAvailable) {
        $children.Add((Start-HiddenProcess (Join-Path $bin "stocker-ingestion.exe") @() "ingestion"))
        $children.Add((Start-HiddenProcess (Join-Path $bin "stocker-analysis.exe") @() "analysis"))
    }
    Write-Host "Backend is starting at http://localhost:8080"
    Write-Host "Worker logs: $logs"
    Write-Host "Press Ctrl+C to stop the API and workers. MongoDB/Redis remain available for the next run."
    & (Join-Path $bin "stocker-api.exe")
}
finally {
    foreach ($process in $children) {
        if ($process -and -not $process.HasExited) {
            Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
        }
    }
}
