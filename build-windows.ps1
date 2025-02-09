# Start time
start_time=$(date +%s)

echo "🚀 Starting build process..."
echo "⌛ Building UI project..."

# Build the UI project
docker run --rm -v "${PWD}:/app" -w /app/ui node:latest `
    sh -c "npm install -g npm@11.1.0 && NODE_OPTIONS='--no-deprecation' npm run build"

echo "📦 Copying UI build to back-go/public..."
# Copy the UI build to the back-go/public directory
if (-Not (Test-Path -Path "back-go/public")) {
    New-Item -ItemType Directory -Path "back-go/public"
}
Copy-Item -Recurse -Force -Path "ui/dist/*" -Destination "back-go/public/"

# Build for Windows
docker run --rm -v "${PWD}:/app" -w /app golang:latest `
    sh -c "cd back-go && GOOS=windows GOARCH=amd64 go build -o ../mystravastats.exe"

# End time
end_time=$(date +%s)
elapsed_time=$((end_time - start_time))

echo "Build process completed in $elapsed_time seconds."