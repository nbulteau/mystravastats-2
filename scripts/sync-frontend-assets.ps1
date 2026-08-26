param (
    [Parameter(Mandatory = $true)]
    [ValidateSet("go", "kotlin")]
    [string] $Target,

    [switch] $SkipBuild
)

$ErrorActionPreference = "Stop"

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$RepoRoot = Split-Path -Parent $ScriptDir
$FrontDir = Join-Path $RepoRoot "front-vue"
$DistDir = Join-Path $FrontDir "dist"

switch ($Target) {
    "go" {
        $Destination = Join-Path $RepoRoot "back-go/public"
    }
    "kotlin" {
        $Destination = Join-Path $RepoRoot "back-kotlin/build/generated/frontend-static"
    }
}

if (-not $SkipBuild) {
    $npmCommand = Get-Command "npm" -ErrorAction SilentlyContinue
    if (-not $npmCommand) {
        throw "npm is required to build front-vue."
    }

    if (-not (Test-Path -Path (Join-Path $FrontDir "package.json"))) {
        throw "Missing front-vue/package.json."
    }

    Push-Location $FrontDir
    try {
        & npm run build
        if ($LASTEXITCODE -ne 0) {
            throw "front-vue build failed."
        }
    } finally {
        Pop-Location
    }
}

$IndexPath = Join-Path $DistDir "index.html"
if (-not (Test-Path -Path $IndexPath)) {
    throw "Missing front-vue/dist/index.html; run npm run build first."
}

if (Test-Path -Path $Destination) {
    Remove-Item -Recurse -Force $Destination
}

New-Item -ItemType Directory -Path $Destination | Out-Null
Copy-Item -Recurse -Force -Path (Join-Path $DistDir "*") -Destination $Destination

$RelativeDestination = Resolve-Path -Path $Destination -Relative
Write-Output "Synced front-vue/dist to $RelativeDestination"
