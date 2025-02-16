#!/bin/zsh

# Start time
start_time=$(date +%s)

echo "🚀 Starting build process..."

# Build the UI project
echo "⌛ Building front-vue project..."
docker run --rm -v "$PWD:/app" -w /app/front-vue node:latest \
    sh -c "npm install -g npm@11.1.0 2>/dev/null && npm install && VITE_CJS_TRACE=false NODE_OPTIONS='--no-deprecation' npm run build 2>/dev/null" > $null 2>&1

# Copy the UI build to the back-go/public directory
echo "📦 Copying UI build to back-go/public..."
mkdir -p back-go/public
cp -r front-vue/dist/* back-go/public/

# Build for macOS
echo "🔨 Building macOS binary..."
docker run --rm -v "$PWD:/app" -w /app golang:latest \
    sh -c "cd back-go && GOOS=darwin GOARCH=amd64 go build -o ../mystravastats" > $null 2>&1

# Ensure strava-cache directory exists
if [ ! -d strava-cache ]; then
    mkdir strava-cache
fi

# Copy the famous-climb directory to strava-cache
cp -r famous-climb strava-cache/

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
    if ! grep -q "STRAVA_CACHE_PATH" .env; then
        echo "STRAVA_CACHE_PATH=strava-cache" >> .env
    fi
fi

# End time
end_time=$(date +%s)
elapsed_time=$((end_time - start_time))

echo "✅ Build process completed in $elapsed_time seconds."