# Start time
$start_time = Get-Date

Write-Output "🚀 Starting build process..."

# Build the UI project
Write-Output "⌛ Building front-vue project..."
docker run --rm -v "${PWD}:/app" -w /app/front-vue node:latest `
    sh -c "npm install -g npm@11.1.0 2>/dev/null && npm install && VITE_CJS_TRACE=false NODE_OPTIONS='--no-deprecation' npm run build 2>/dev/null" > $null 2>&1


Write-Output "📦 Copying UI build to back-go/public..."
# Copy the UI build to the back-go/public directory
if (-Not (Test-Path -Path "back-go/public")) {
    New-Item -ItemType Directory -Path "back-go/public"
}
Copy-Item -Recurse -Force -Path "front-vue/dist/*" -Destination "back-go/public/"

# Build for Windows
Write-Output "⌛ Building back-go project..."
docker run --rm -v "${PWD}:/app" -w /app golang:latest `
    sh -c "cd back-go && GOOS=windows GOARCH=amd64 go build -o ../mystravastats.exe" > $null 2>&1

# Ensure strava-cache directory exists
if (-Not (Test-Path -Path "strava-cache")) {
    New-Item -ItemType Directory -Path "strava-cache"
    Write-Output "📁 Created strava-cache directory."
}

# Copy the famous-climb directory to strava-cache
Copy-Item -Recurse -Force -Path "back-go/famous-climb" -Destination "strava-cache/"

# Ensure .strava file exists in strava-cache directory
$stravaFilePath = "strava-cache/.strava"
if (-Not (Test-Path -Path $stravaFilePath)) {
    Set-Content -Path $stravaFilePath -Value "clientId=`nclientSecret="
    Write-Output "ℹ️ Any registered Strava user can obtain an `access_token` by first creating an application at [Strava API Settings](https://www.strava.com/settings/api)."
    Write-Output "🔑 Please add your Strava API credentials to strava-cache/.strava file."
}

# Ensure .env file exists and add STRAVA_CACHE_PATH
$envFilePath = ".env"
if (-Not (Test-Path -Path $envFilePath)) {
    $currentDirectory = (Get-Location).Path
    Set-Content -Path $envFilePath -Value "STRAVA_CACHE_PATH=$currentDirectory\strava-cache"
    Write-Output "📁 Created .env file."
}

# End time
$end_time = Get-Date
$elapsed_time = $end_time - $start_time

Write-Output "✅ Build process completed in $($elapsed_time.TotalSeconds) seconds."