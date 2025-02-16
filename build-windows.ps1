# Start time
$start_time = Get-Date

Write-Output "🚀 Starting build process..."
Write-Output "⌛ Building front-vue project..."

# Build the UI project
docker run --rm -v "${PWD}:/app" -w /app/front-vue node:latest `
    sh -c "npm install -g npm@11.1.0 2>/dev/null && npm install && VITE_CJS_TRACE=false NODE_OPTIONS='--no-deprecation' npm run build 2>/dev/null"

Write-Output "📦 Copying UI build to back-go/public..."
# Copy the UI build to the back-go/public directory
if (-Not (Test-Path -Path "back-go/public")) {
    New-Item -ItemType Directory -Path "back-go/public"
}
Copy-Item -Recurse -Force -Path "front-vue/dist/*" -Destination "back-go/public/"

# Build for Windows
docker run --rm -v "${PWD}:/app" -w /app golang:latest `
    sh -c "cd back-go && GOOS=windows GOARCH=amd64 go build -o ../mystravastats.exe"

# End time
$end_time = Get-Date
$elapsed_time = $end_time - $start_time

Write-Output "✅ Build process completed in $($elapsed_time.TotalSeconds) seconds."