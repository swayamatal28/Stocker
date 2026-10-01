[CmdletBinding()]
param([string]$Output = "artifacts/stocker-sbom.cdx.json")
$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
$target = Join-Path $root $Output
New-Item -ItemType Directory -Force -Path (Split-Path -Parent $target) | Out-Null
Push-Location $root
try {
    $env:GOMODCACHE = Join-Path $root ".cache\go\pkg\mod"
    $components = @()
    foreach ($line in (go list -m all)) {
        $parts = $line -split '\s+'
        if ($parts.Count -ge 2) {
            $components += [ordered]@{ type = "library"; name = $parts[0]; version = $parts[1]; purl = "pkg:golang/$($parts[0])@$($parts[1])" }
        }
    }
    $npmTree = (npm.cmd --prefix apps/web ls --omit=dev --all --json 2>$null) | Out-String | ConvertFrom-Json
    $seenNpm = @{}
    function Add-NpmDependencies([object]$dependencies) {
        if (-not $dependencies) { return }
        foreach ($property in $dependencies.PSObject.Properties) {
            $name = $property.Name
            $version = $property.Value.version
            $key = "$name@$version"
            if ($version -and -not $seenNpm.ContainsKey($key)) {
                $script:components += [ordered]@{ type = "library"; name = $name; version = $version; purl = "pkg:npm/$name@$version" }
                $seenNpm[$key] = $true
            }
            Add-NpmDependencies $property.Value.dependencies
        }
    }
    Add-NpmDependencies $npmTree.dependencies
    $bom = [ordered]@{ bomFormat = "CycloneDX"; specVersion = "1.5"; serialNumber = "urn:uuid:$([guid]::NewGuid())"; version = 1; metadata = [ordered]@{ timestamp = (Get-Date).ToUniversalTime().ToString("o"); component = [ordered]@{ type = "application"; name = "stocker" } }; components = $components }
    $bom | ConvertTo-Json -Depth 8 | Set-Content -Encoding UTF8 $target
    Write-Host "SBOM written to $target ($($components.Count) components)"
}
finally { Pop-Location }
