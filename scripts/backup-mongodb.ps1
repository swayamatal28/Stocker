[CmdletBinding()]
param(
    [string]$MongoURI = $env:MONGODB_URI,
    [string]$Database = $env:MONGODB_DATABASE,
    [string]$OutputDirectory = "backups"
)
$ErrorActionPreference = "Stop"
if (-not $MongoURI -or -not $Database) { throw "MONGODB_URI and MONGODB_DATABASE are required" }
$root = Split-Path -Parent $PSScriptRoot
$directory = [IO.Path]::GetFullPath((Join-Path $root $OutputDirectory))
if (-not $directory.StartsWith([IO.Path]::GetFullPath($root), [StringComparison]::OrdinalIgnoreCase)) { throw "Backup directory must remain inside the workspace" }
New-Item -ItemType Directory -Force -Path $directory | Out-Null
$stamp = (Get-Date).ToUniversalTime().ToString("yyyyMMddTHHmmssZ")
$archive = Join-Path $directory "stocker-$stamp.archive.gz"
mongodump --uri $MongoURI --db $Database --archive=$archive --gzip
if ($LASTEXITCODE -ne 0) { throw "mongodump failed" }
$hash = (Get-FileHash -Algorithm SHA256 -LiteralPath $archive).Hash
[ordered]@{ database = $Database; createdAt = (Get-Date).ToUniversalTime().ToString("o"); archive = (Split-Path -Leaf $archive); sha256 = $hash; encryptedInTransit = $MongoURI -match 'mongodb\+srv://|tls=true' } | ConvertTo-Json | Set-Content -Encoding UTF8 "$archive.manifest.json"
Write-Host "Backup and checksum manifest created: $archive"
