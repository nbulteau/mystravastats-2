#!/bin/zsh

# Start time
start_time=$(date +%s)

echo "🚀 Starting build process..."
echo "⌛ Building UI project..."

# Build the UI project
docker run --rm -v "$PWD:/app" -w /app/ui node:latest \
    sh -c "npm install -g npm@11.1.0 2>/dev/null && npm install && VITE_CJS_TRACE=false NODE_OPTIONS='--no-deprecation' npm run build 2>/dev/null"

echo "📦 Copying UI build to back-go/public..."
# Copy the UI build to the back-go/public directory
mkdir -p back-go/public
cp -r ui/dist/* back-go/public/

echo "🔨 Building macOS binary..."
# Build for macOS
docker run --rm -v "$PWD:/app" -w /app golang:latest \
    sh -c "cd back-go && GOOS=darwin GOARCH=amd64 go build -o ../mystravastats-mac"

# End time
end_time=$(date +%s)
elapsed_time=$((end_time - start_time))

echo "✅ Build process completed in $elapsed_time seconds."