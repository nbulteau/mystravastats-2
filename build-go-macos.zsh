#!/bin/zsh

set -euo pipefail

# Check for --verbose flag
VERBOSE=0
for arg in "$@"; do
  if [[ "$arg" == "--verbose" ]]; then
    VERBOSE=1
    set -x  # Enable shell debug output
  fi
done

# Start time
start_time=$(date +%s)
FRONT_NODE_IMAGE="${FRONT_NODE_IMAGE:-node:26.5.0}"
GO_IMAGE="${GO_IMAGE:-golang:1.26.5}"

echo "🚀 Starting build process..."

# Build the UI project silently or verbosely
echo "⌛ Building front-vue project..."
if [[ $VERBOSE -eq 1 ]]; then
  docker run --rm -v "$PWD:/app" -v mystravastats-front-node-modules:/app/front-vue/node_modules -w /app/front-vue "$FRONT_NODE_IMAGE" \
    sh -c "npm ci --loglevel=error --no-audit --no-fund --update-notifier=false && VITE_CJS_TRACE=false NODE_OPTIONS='--no-deprecation' npm run type-check && VITE_CJS_TRACE=false NODE_OPTIONS='--no-deprecation' npm run build-only"
else
  docker run --rm -v "$PWD:/app" -v mystravastats-front-node-modules:/app/front-vue/node_modules -w /app/front-vue "$FRONT_NODE_IMAGE" \
    sh -c "npm ci --loglevel=error --no-audit --no-fund --update-notifier=false >/dev/null && VITE_CJS_TRACE=false NODE_OPTIONS='--no-deprecation' npm run --silent type-check && VITE_CJS_TRACE=false NODE_OPTIONS='--no-deprecation' npm run --silent build-only -- --logLevel error"
fi

# Copy the freshly built UI into the Go embed directory.
echo "📦 Copying UI build to back-go/public..."
./scripts/sync-frontend-assets.sh go --skip-build

# Remove old binary before building
if [ -f mystravastats ]; then
    rm mystravastats
    echo "🗑️ Removed old mystravastats binary."
fi

# Build back for macOS silently or verbosely
echo "🔨 Building macOS binary..."
if [[ $VERBOSE -eq 1 ]]; then
  docker run --rm -v "$PWD:/app" -w /app "$GO_IMAGE" \
    sh -c "cd back-go && GOOS=darwin GOARCH=arm64 go build -o ../mystravastats"
else
  docker run --rm -v "$PWD:/app" -w /app "$GO_IMAGE" \
    sh -c "cd back-go && GOOS=darwin GOARCH=arm64 go build -o ../mystravastats"
fi

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
