#!/bin/zsh

# Start time
start_time=$(date +%s)

echo "🚀 Starting build process..."

# Build the UI project silently
echo "⌛ Building front-vue project..."
docker run --rm -v "$PWD:/app" -w /app/front-vue node:latest \
    sh -c "npm install -g npm@11.4.2 >/dev/null 2>&1 && npm install >/dev/null 2>&1 && VITE_CJS_TRACE=false NODE_OPTIONS='--no-deprecation' npm run build >/dev/null 2>&1" 

# Copy the UI build to the back-go/public directory
echo "📦 Copying UI build to back-go/public..."
rm -rf back-go/public
mkdir -p back-go/public
cp -r front-vue/dist/* back-go/public/

# Remove old binary before building
if [ -f mystravastats ]; then
    rm mystravastats
    echo "🗑️ Removed old mystravastats binary."
fi

# Build back for macOS silently
echo "🔨 Building macOS binary..."
docker run --rm -v "$PWD:/app" -w /app golang:1.24.4 \
    sh -c "cd back-go && GOOS=darwin GOARCH=arm64 go build -o ../mystravastats" >/dev/null 2>&1

# Check if new binary was created
if [ ! -f mystravastats ]; then
    echo "❌ Build failed: mystravastats binary not found."
    exit 1
fi

# Ensure strava-cache directory exists
if [ ! -d strava-cache ]; then
    mkdir strava-cache
    echo "📁 Created strava-cache directory."
fi

# Copy the famous-climb directory to strava-cache
cp -r back-go/famous-climb strava-cache/

# Ensure .strava file exists in strava-cache directory
strava_file_path="strava-cache/.strava"
if [ ! -f "$strava_file_path" ]; then
    echo "clientId=\nclientSecret=" > "$strava_file_path"
    echo "ℹ️ Any registered Strava user can obtain an access_token by first creating an application at [Strava API Settings](https://www.strava.com/settings/api)."
    echo "🔑 Please add your Strava API credentials to strava-cache/.strava file."
fi

# Ensure .env file exists and add STRAVA_CACHE_PATH
if [ ! -f .env ]; then
    touch .env
    echo "STRAVA_CACHE_PATH=$PWD/strava-cache" >> .env
    echo "📁 Created '.env' file."
fi

# End time
end_time=$(date +%s)
elapsed_time=$((end_time - start_time))

echo "✅ Build process completed in $elapsed_time seconds."