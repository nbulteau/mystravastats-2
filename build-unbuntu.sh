#!/bin/bash

# Start time
start_time=$(date +%s)

echo "🚀 Starting build process..."
echo "⌛ Building front-vue project..."

# Build the UI project
docker run --rm -v "$PWD:/app" -w /app/front-vue node:latest \
    sh -c "npm install -g npm@11.1.0 2>/dev/null && npm install && VITE_CJS_TRACE=false NODE_OPTIONS='--no-deprecation' npm run build 2>/dev/null"

echo "📦 Copying UI build to back-go/public..."
# Copy the UI build to the back-go/public directory
mkdir -p back-go/public
cp -r front-vue/dist/* back-go/public/

echo "🔨 Building Ubuntu binary..."
# Build for Ubuntu
docker run --rm -v "$PWD:/app" -w /app golang:latest \
 sh -c "cd back-go && GOOS=linux GOARCH=amd64 go build -o ../mystravastats -buildvcs=false"

# End time
end_time=$(date +%s)
elapsed_time=$((end_time - start_time))

echo "✅ Build process completed in $elapsed_time seconds."

# Change owner of the built binary to the current user
echo "🔄 Change owner of mystravastats to current user: 'sudo chown $(whoami):$(whoami) mystravastats'"