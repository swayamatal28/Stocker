[CmdletBinding()]
param(
    [Parameter(Mandatory)][string]$Archive,
    [string]$MongoURI = $env:MONGODB_URI,
    [string]$SourceDatabase = $env:MONGODB_DATABASE,
    [Parameter(Mandatory)][string]$TargetDatabase
)
$ErrorActionPreference = "Stop"
if (-not $MongoURI -or -not $SourceDatabase) { throw "Mongo connection and source database are required" }
if ($TargetDatabase -notmatch '_restore_[0-9A-Za-z_-]+$') { throw "TargetDatabase must end in _restore_<drill-id>; production names are refused" }
$archivePath = [IO.Path]::GetFullPath($Archive)
if (-not (Test-Path -LiteralPath $archivePath -PathType Leaf)) { throw "Archive not found" }
$manifestPath = "$archivePath.manifest.json"
if (Test-Path -LiteralPath $manifestPath) {
    $manifest = Get-Content -Raw -LiteralPath $manifestPath | ConvertFrom-Json
    if ((Get-FileHash -Algorithm SHA256 -LiteralPath $archivePath).Hash -ne $manifest.sha256) { throw "Backup checksum mismatch" }
}
mongorestore --uri $MongoURI --archive=$archivePath --gzip --drop --nsFrom "$SourceDatabase.*" --nsTo "$TargetDatabase.*"
if ($LASTEXITCODE -ne 0) { throw "mongorestore drill failed" }
Write-Host "Restore drill completed into isolated database $TargetDatabase. Verify counts, indexes, and /health/ready before recording evidence."
