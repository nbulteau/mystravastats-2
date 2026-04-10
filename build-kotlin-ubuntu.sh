#!/bin/bash

set -euo pipefail

# Check for --verbose flag
VERBOSE=0
for arg in "$@"; do
  if [[ "$arg" == "--verbose" ]]; then
    VERBOSE=1
    set -x
  fi
done

ROOT_DIR=$(cd -- "$(dirname "$0")" && pwd)
BACK_DIR="$ROOT_DIR/back-kotlin"
NATIVE_BINARY="$BACK_DIR/build/native/nativeCompile/mystravastats-kotlin"
OUTPUT_BINARY="${OUTPUT_BINARY_NAME:-mystravastats-kotlin-ubuntu}"
DOCKER_IMAGE="${GRAALVM_IMAGE:-ghcr.io/graalvm/native-image-community:25}"
NODE_DOCKER_IMAGE="${NODE_DOCKER_IMAGE:-node:latest}"
DOCKER_PLATFORM="${NATIVE_DOCKER_PLATFORM:-linux/amd64}"
HOST_USER="$(id -un 2>/dev/null || echo user)"
NATIVE_IMAGE_OPTIONS_VALUE="${NATIVE_IMAGE_OPTIONS:---parallelism=2 -J-Xms1g -J-Xmx6g}"
GRADLE_WORKERS_MAX="${GRADLE_WORKERS_MAX:-1}"
GRADLE_OPTS_VALUE="-Dorg.gradle.vfs.watch=false -Dorg.gradle.workers.max=$GRADLE_WORKERS_MAX"
GRADLE_NATIVE_TASKS="${GRADLE_NATIVE_TASKS:-nativeCompile}"
GRADLE_USER_HOME_DIR="${GRADLE_USER_HOME_OVERRIDE:-/workspace/.gradle-home-ubuntu}"
SKIP_FRONT_BUILD="${SKIP_FRONT_BUILD:-0}"

start_time=$(date +%s)
echo "🚀 Starting Kotlin native-image build for Ubuntu with Docker..."
echo "🐳 Image: $DOCKER_IMAGE"
echo "🏗️  Target platform: $DOCKER_PLATFORM"
echo "⚙️ Native image options: $NATIVE_IMAGE_OPTIONS_VALUE"
echo "⚙️ Gradle workers max: $GRADLE_WORKERS_MAX"
echo "⚙️ Gradle user home: $GRADLE_USER_HOME_DIR"
if [[ "$SKIP_FRONT_BUILD" == "1" ]]; then
  echo "⚙️ Front build: disabled (SKIP_FRONT_BUILD=1)"
else
  echo "⚙️ Front build: enabled (front-vue -> public/)"
fi

DOCKER_BIN="$(command -v docker || true)"
if [[ -z "$DOCKER_BIN" ]]; then
  for candidate in \
    /opt/homebrew/bin/docker \
    /usr/local/bin/docker \
    /Applications/Docker.app/Contents/Resources/bin/docker; do
    if [[ -x "$candidate" ]]; then
      DOCKER_BIN="$candidate"
      break
    fi
  done
fi

if [[ -z "$DOCKER_BIN" ]]; then
  echo "❌ Docker is required but was not found in PATH."
  exit 1
fi
echo "🐋 Docker binary: $DOCKER_BIN"

if [[ ! -x "$BACK_DIR/gradlew" ]]; then
  echo "❌ back-kotlin/gradlew is missing or not executable."
  exit 1
fi

if [[ "$SKIP_FRONT_BUILD" != "1" ]]; then
  echo "⌛ Building front-vue project with Docker..."
  if [[ $VERBOSE -eq 1 ]]; then
    "$DOCKER_BIN" run --rm -v "$ROOT_DIR:/app" -w /app/front-vue "$NODE_DOCKER_IMAGE" \
      sh -c "npm install -g npm@11.6.2 && npm install && VITE_CJS_TRACE=false NODE_OPTIONS='--no-deprecation' npm run build"
  else
    "$DOCKER_BIN" run --rm -v "$ROOT_DIR:/app" -w /app/front-vue "$NODE_DOCKER_IMAGE" \
      sh -c "npm install -g npm@11.6.2 >/dev/null 2>&1 && npm install >/dev/null 2>&1 && VITE_CJS_TRACE=false NODE_OPTIONS='--no-deprecation' npm run build >/dev/null 2>&1"
  fi

  if [[ ! -d "$ROOT_DIR/front-vue/dist" ]]; then
    echo "❌ front-vue build failed: dist/ not found."
    exit 1
  fi

  # Kotlin backend serves static files from file:public/
  echo "📦 Copying UI build from front-vue/dist to public/..."
  rm -rf "$ROOT_DIR/public"
  mkdir -p "$ROOT_DIR/public"
  cp -r "$ROOT_DIR/front-vue/dist/"* "$ROOT_DIR/public/"
else
  echo "⏭️ Skipping front-vue build and copy because SKIP_FRONT_BUILD=1."
fi

echo "🔨 Building Kotlin native executable for Ubuntu inside Docker (this can take a while)..."

if [[ $VERBOSE -eq 1 ]]; then
  "$DOCKER_BIN" run --rm \
    --platform "$DOCKER_PLATFORM" \
    --entrypoint /bin/bash \
    -u "$(id -u):$(id -g)" \
    -e GRADLE_USER_HOME="$GRADLE_USER_HOME_DIR" \
    -e GRADLE_OPTS="$GRADLE_OPTS_VALUE" \
    -e NATIVE_IMAGE_OPTIONS="$NATIVE_IMAGE_OPTIONS_VALUE" \
    -e USER="$HOST_USER" \
    -e LOGNAME="$HOST_USER" \
    -v "$ROOT_DIR:/workspace" \
    -w /workspace/back-kotlin \
    "$DOCKER_IMAGE" \
    -lc "./gradlew --no-daemon clean $GRADLE_NATIVE_TASKS"
else
  if ! "$DOCKER_BIN" run --rm \
    --platform "$DOCKER_PLATFORM" \
    --entrypoint /bin/bash \
    -u "$(id -u):$(id -g)" \
    -e GRADLE_USER_HOME="$GRADLE_USER_HOME_DIR" \
    -e GRADLE_OPTS="$GRADLE_OPTS_VALUE" \
    -e NATIVE_IMAGE_OPTIONS="$NATIVE_IMAGE_OPTIONS_VALUE" \
    -e USER="$HOST_USER" \
    -e LOGNAME="$HOST_USER" \
    -v "$ROOT_DIR:/workspace" \
    -w /workspace/back-kotlin \
    "$DOCKER_IMAGE" \
    -lc "./gradlew --no-daemon clean $GRADLE_NATIVE_TASKS" >/dev/null 2>&1; then
    echo "❌ Native build failed. Re-run with --verbose for details."
    exit 1
  fi
fi

if [[ ! -f "$NATIVE_BINARY" ]]; then
  echo "❌ Native build failed: binary not found at $NATIVE_BINARY"
  exit 1
fi

cp "$NATIVE_BINARY" "$ROOT_DIR/$OUTPUT_BINARY"
chmod +x "$ROOT_DIR/$OUTPUT_BINARY"
echo "📦 Native Ubuntu binary ready: ./$OUTPUT_BINARY"
echo "ℹ️  This binary is built for Linux/Ubuntu (Docker target: $DOCKER_PLATFORM)."

# Ensure strava-cache directory exists
if [[ ! -d "$ROOT_DIR/strava-cache" ]]; then
  mkdir -p "$ROOT_DIR/strava-cache"
  echo "📁 Created strava-cache directory."
fi

# Copy the famous-climb directory to strava-cache
cp -r "$BACK_DIR/famous-climb" "$ROOT_DIR/strava-cache/" 2>/dev/null || true

# Ensure .strava file exists in strava-cache directory
strava_file_path="$ROOT_DIR/strava-cache/.strava"
if [[ ! -f "$strava_file_path" ]]; then
  cat <<EOF > "$strava_file_path"
clientId=
clientSecret=
useCache=false
EOF
  echo "🔑 Please add your Strava API credentials to strava-cache/.strava"
fi

# Ensure .env file exists and add STRAVA_CACHE_PATH if missing
if [[ ! -f "$ROOT_DIR/.env" ]]; then
  touch "$ROOT_DIR/.env"
fi
if ! rg -q "^STRAVA_CACHE_PATH=" "$ROOT_DIR/.env" 2>/dev/null; then
  echo "STRAVA_CACHE_PATH=$ROOT_DIR/strava-cache" >> "$ROOT_DIR/.env"
  echo "📄 Added STRAVA_CACHE_PATH to .env"
fi

end_time=$(date +%s)
elapsed_time=$((end_time - start_time))
echo "✅ Kotlin Ubuntu Docker native build completed in $elapsed_time seconds."
echo "ℹ️ Run with: ./$OUTPUT_BINARY"
