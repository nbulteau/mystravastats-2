param (
    [switch]$Verbose
)

$ErrorActionPreference = "Stop"
$ProgressPreference = "SilentlyContinue"

$RootDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$FrontDir = Join-Path $RootDir "front-vue"
$FrontDistDir = Join-Path $FrontDir "dist"
$BackDir = Join-Path $RootDir "back-go"
$BackPublicDir = Join-Path $BackDir "public"
$OutputBinary = Join-Path $RootDir "mystravastats.exe"
$StravaCacheDir = Join-Path $RootDir "strava-cache"
$StravaFilePath = Join-Path $StravaCacheDir ".strava"
$EnvFilePath = Join-Path $RootDir ".env"
$SkipFrontBuild = if ($env:SKIP_FRONT_BUILD) { $env:SKIP_FRONT_BUILD } else { "0" }

# Record the start time of the build process
$start_time = Get-Date

Write-Output "[START] Starting build process..."
if ($SkipFrontBuild -eq "1") {
    Write-Output "[INFO] Front build: disabled (SKIP_FRONT_BUILD=1)"
} else {
    Write-Output "[INFO] Front build: enabled (front-vue -> back-go/public)"
}

# Function to write output only if verbose mode is enabled
function Write-VerboseOutput {
    param (
        [string]$Message
    )
    if ($Verbose) {
        Write-Output $Message
    }
}

$dockerCommand = Get-Command "docker" -ErrorAction SilentlyContinue
if (-not $dockerCommand) {
    throw "Docker is required to run this script."
}

# Verify that Docker is actually running
$previousErrorActionPreference = $ErrorActionPreference
$ErrorActionPreference = "Continue"
try {
    & docker version *> $null
} catch {
    throw "Docker is installed but not running. Please start Docker Desktop and try again."
}
if ($LASTEXITCODE -ne 0) {
    throw "Docker is installed but not running. Please start Docker Desktop and try again."
}
$ErrorActionPreference = $previousErrorActionPreference

if ($SkipFrontBuild -ne "1") {
    # Clean the front-vue directory
    Write-VerboseOutput "[FRONT] Cleaning front-vue directory..."
    if (Test-Path -Path (Join-Path $FrontDir "node_modules")) {
        Remove-Item -Recurse -Force (Join-Path $FrontDir "node_modules")
        Write-VerboseOutput "[CLEAN] Removed node_modules directory."
    }

    # Build the UI project using Docker
    Write-VerboseOutput "[FRONT] Building front-vue project..."
    $frontFixCommand = 'npm pkg delete "dependencies.@rolldown/binding-darwin-arm64" "devDependencies.@rolldown/binding-darwin-arm64" >/dev/null 2>&1 || true'
    if ($Verbose) {
        & docker run --rm -v "${RootDir}:/app" -w /app/front-vue node:latest `
          sh -c "$frontFixCommand && npm ci && VITE_CJS_TRACE=false NODE_OPTIONS='--no-deprecation' npm run build"
    } else {
        & docker run --rm -v "${RootDir}:/app" -w /app/front-vue node:latest `
          sh -c "$frontFixCommand && npm ci >/dev/null 2>&1 && VITE_CJS_TRACE=false NODE_OPTIONS='--no-deprecation' npm run build >/dev/null 2>&1" *> $null
    }

    if ($LASTEXITCODE -ne 0) {
        throw "front-vue build failed. Re-run with -Verbose for details."
    }

    if (-not (Test-Path -Path $FrontDistDir)) {
        throw "front-vue build failed: dist directory not found at '$FrontDistDir'."
    }

    # Copy the build artifacts to the back-go/public directory
    Write-VerboseOutput "[FRONT] Copying UI build to back-go/public..."
    # Remove the existing back-go/public directory if it exists
    if (Test-Path -Path $BackPublicDir) {
        Remove-Item -Recurse -Force $BackPublicDir
        Write-VerboseOutput "[CLEAN] Removed existing back-go/public directory."
    }
    # Recreate the back-go/public directory
    New-Item -ItemType Directory -Path $BackPublicDir | Out-Null
    # Copy the build artifacts
    Copy-Item -Recurse -Force -Path (Join-Path $FrontDistDir "*") -Destination $BackPublicDir
} else {
    Write-Output "[SKIP] Skipping front-vue build and copy because SKIP_FRONT_BUILD=1."

    if (-not (Test-Path -Path $BackPublicDir)) {
        New-Item -ItemType Directory -Path $BackPublicDir | Out-Null
    }

    $publicFiles = Get-ChildItem -Path $BackPublicDir -Recurse -File -ErrorAction SilentlyContinue
    if (-not $publicFiles) {
        $placeholderFile = Join-Path $BackPublicDir "index.html"
        Set-Content -Path $placeholderFile -Value "<!doctype html><html><body><h1>UI build skipped (SKIP_FRONT_BUILD=1)</h1></body></html>"
        Write-VerboseOutput "[INFO] Created placeholder back-go/public/index.html for go:embed."
    }
}

# Remove the old binary before building the new one
if (Test-Path -Path $OutputBinary) {
    Remove-Item -Force $OutputBinary
    Write-VerboseOutput "[CLEAN] Removed old mystravastats.exe binary."
}

# Build the back-go project for Windows using Docker
Write-VerboseOutput "[BACK] Building back-go project..."
$previousErrorActionPreference = $ErrorActionPreference
$ErrorActionPreference = "Continue"
try {
    if ($Verbose) {
        & docker run --rm -v "${RootDir}:/app" -w /app golang:1.26.2`
          sh -c "cd back-go && GOOS=windows GOARCH=amd64 go build -o ../mystravastats.exe"
    } else {
        & docker run --rm -v "${RootDir}:/app" -w /app golang:1.26.2`
          sh -c "cd back-go && GOOS=windows GOARCH=amd64 go build -o ../mystravastats.exe" *> $null
    }
} finally {
    $ErrorActionPreference = $previousErrorActionPreference
}

if ($LASTEXITCODE -ne 0) {
    throw "back-go build failed. Re-run with -Verbose for details."
}

# Check if the new binary was created successfully
if (-Not (Test-Path -Path $OutputBinary)) {
    Write-Output "[ERROR] Build failed: mystravastats.exe binary not found."
    exit 1
}

# Ensure the strava-cache directory exists
if (-Not (Test-Path -Path $StravaCacheDir)) {
    New-Item -ItemType Directory -Path $StravaCacheDir | Out-Null
    Write-VerboseOutput "[INFO] Created strava-cache directory."
}

# Copy the famous-climb directory to the strava-cache directory
Copy-Item -Recurse -Force -Path (Join-Path $BackDir "famous-climb") -Destination $StravaCacheDir

# Ensure the .strava file exists in the strava-cache directory
if (-Not (Test-Path -Path $stravaFilePath)) {
    Set-Content -Path $stravaFilePath -Value "clientId=`nclientSecret="
    Write-Output "[INFO] Any registered Strava user can obtain an access_token by first creating an application at https://www.strava.com/settings/api."
    Write-Output "[TODO] Please add your Strava API credentials to strava-cache/.strava file."
}

# Ensure the .env file exists and add the STRAVA_CACHE_PATH variable
if (-Not (Test-Path -Path $envFilePath)) {
    New-Item -Path $envFilePath -ItemType File | Out-Null
    Write-VerboseOutput "[INFO] Created .env file."
}

$existingEnv = Get-Content -Path $envFilePath -ErrorAction SilentlyContinue
if (-not ($existingEnv -match '^STRAVA_CACHE_PATH=')) {
    Add-Content -Path $envFilePath -Value "STRAVA_CACHE_PATH=$StravaCacheDir"
}

# Record the end time and calculate the elapsed time
$end_time = Get-Date
$elapsed_time = $end_time - $start_time

Write-Output "[DONE] Build process completed in $($elapsed_time.TotalSeconds) seconds."
