param (
    [switch]$Verbose
)

# Record the start time of the build process
$start_time = Get-Date

Write-Output "🚀 Starting build process..."

# Function to write output only if verbose mode is enabled
function Write-VerboseOutput {
    param (
        [string]$Message
    )
    if ($Verbose) {
        Write-Output $Message
    }
}

# Build the UI project using Docker
Write-VerboseOutput "⌛ Building front-vue project..."
docker run --rm -v "${PWD}:/app" -w /app/front-vue node:latest `
 sh -c "npm install -g npm@11.4.2 2>/dev/null && npm install && VITE_CJS_TRACE=false NODE_OPTIONS='--no-deprecation' npm run build 2>/dev/null" > $null 2>&1

# Copy the build artifacts to the back-go/public directory
Write-VerboseOutput "📦 Copying UI build to back-go/public..."
# Remove the existing back-go/public directory if it exists
if (Test-Path -Path "back-go/public") {
    Remove-Item -Recurse -Force "back-go/public"
    Write-VerboseOutput "🗑️ Removed existing back-go/public directory."
}
# Recreate the back-go/public directory
New-Item -ItemType Directory -Path "back-go/public" | Out-Null
# Copy the build artifacts
Copy-Item -Recurse -Force -Path "front-vue/dist/*" -Destination "back-go/public/"

# Remove the old binary before building the new one
if (Test-Path -Path "mystravastats.exe") {
    Remove-Item -Force "mystravastats.exe"
    Write-VerboseOutput "🗑️ Removed old mystravastats.exe binary."
}

# Build the back-go project for Windows using Docker
Write-VerboseOutput "⌛ Building back-go project..."
docker run --rm -v "${PWD}:/app" -w /app golang:1.24.4 `
 sh -c "cd back-go; GOOS=windows GOARCH=amd64 go build -o ../mystravastats.exe" > $null 2>&1

# Check if the new binary was created successfully
if (-Not (Test-Path -Path "mystravastats.exe")) {
    Write-Output "❌ Build failed: mystravastats.exe binary not found."
    exit 1
}

# Ensure the strava-cache directory exists
if (-Not (Test-Path -Path "strava-cache")) {
    New-Item -ItemType Directory -Path "strava-cache"
    Write-VerboseOutput "📁 Created strava-cache directory."
}

# Copy the famous-climb directory to the strava-cache directory
Copy-Item -Recurse -Force -Path "back-go/famous-climb" -Destination "strava-cache/"

# Ensure the .strava file exists in the strava-cache directory
$stravaFilePath = "strava-cache/.strava"
if (-Not (Test-Path -Path $stravaFilePath)) {
    Set-Content -Path $stravaFilePath -Value "clientId=`nclientSecret="
    Write-Output "ℹ️ Any registered Strava user can obtain an `access_token` by first creating an application at [Strava API Settings](https://www.strava.com/settings/api)."
    Write-Output "🔑 Please add your Strava API credentials to strava-cache/.strava file."
}

# Ensure the .env file exists and add the STRAVA_CACHE_PATH variable
$envFilePath = ".env"
if (-Not (Test-Path -Path $envFilePath)) {
    $currentDirectory = (Get-Location).Path
    Set-Content -Path $envFilePath -Value "STRAVA_CACHE_PATH=$currentDirectory\strava-cache"
    Write-VerboseOutput "📄 Created .env file."
}

# Record the end time and calculate the elapsed time
$end_time = Get-Date
$elapsed_time = $end_time - $start_time

Write-Output "✅ Build process completed in $($elapsed_time.TotalSeconds) seconds."
