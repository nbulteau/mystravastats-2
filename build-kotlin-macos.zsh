#!/bin/zsh

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
OUTPUT_BINARY="${OUTPUT_BINARY_NAME:-mystravastats-kotlin-macos-arm64}"
NATIVE_IMAGE_OPTIONS_VALUE="${NATIVE_IMAGE_OPTIONS:---parallelism=4 -J-Xms1g -J-Xmx8g}"
GRADLE_WORKERS_MAX="${GRADLE_WORKERS_MAX:-2}"
GRADLE_OPTS_VALUE="-Dorg.gradle.vfs.watch=false -Dorg.gradle.workers.max=$GRADLE_WORKERS_MAX"
GRADLE_USER_HOME_DIR="${GRADLE_USER_HOME_OVERRIDE:-$ROOT_DIR/.gradle-home-macos}"
SKIP_FRONT_BUILD="${SKIP_FRONT_BUILD:-0}"

start_time=$(date +%s)
echo "🚀 Starting Kotlin native-image build for macOS (local)..."
echo "⚙️ Native image options: $NATIVE_IMAGE_OPTIONS_VALUE"
echo "⚙️ Gradle workers max: $GRADLE_WORKERS_MAX"
echo "⚙️ Gradle user home: $GRADLE_USER_HOME_DIR"
if [[ "$SKIP_FRONT_BUILD" == "1" ]]; then
  echo "⚙️ Front build: disabled (SKIP_FRONT_BUILD=1)"
else
  echo "⚙️ Front build: enabled (front-vue -> public/)"
fi

if [[ "$(uname -s)" != "Darwin" ]]; then
  echo "❌ This script targets macOS and must be run on Darwin."
  exit 1
fi

if [[ "$(uname -m)" != "arm64" ]]; then
  echo "⚠️ This script is optimized for Apple Silicon (arm64)."
  echo "It will continue, but your output may not match arm64 expectations."
fi

if [[ ! -x "$BACK_DIR/gradlew" ]]; then
  echo "❌ back-kotlin/gradlew is missing or not executable."
  exit 1
fi

if [[ "$SKIP_FRONT_BUILD" != "1" ]]; then
  if ! command -v docker >/dev/null 2>&1; then
    echo "❌ Docker is required to build front-vue in this script."
    echo "   (Or run with SKIP_FRONT_BUILD=1 if you already have public/ ready.)"
    exit 1
  fi

  echo "⌛ Building front-vue project with Docker..."
  if [[ $VERBOSE -eq 1 ]]; then
    docker run --rm -v "$ROOT_DIR:/app" -w /app/front-vue node:latest \
      sh -c "npm install -g npm@11.6.2 && npm install && VITE_CJS_TRACE=false NODE_OPTIONS='--no-deprecation' npm run build"
  else
    docker run --rm -v "$ROOT_DIR:/app" -w /app/front-vue node:latest \
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

if ! command -v native-image >/dev/null 2>&1; then
  echo "ℹ️ 'native-image' was not found in PATH."
  echo "Gradle will try to auto-provision a local GraalVM toolchain (Java 25)."
fi

if [[ -d "$GRADLE_USER_HOME_DIR/jdks" ]]; then
  repaired_wrapper=0
  while IFS= read -r wrapper; do
    jdk_home="$(dirname "$(dirname "$wrapper")")"
    source_bin="$jdk_home/lib/svm/bin/native-image"
    if [[ -s "$source_bin" ]]; then
      cp "$source_bin" "$wrapper"
      chmod +x "$wrapper"
      repaired_wrapper=1
    fi
  done < <(find "$GRADLE_USER_HOME_DIR/jdks" -type f -path "*/bin/native-image" -size 0 2>/dev/null)

  while IFS= read -r wrapper; do
    jdk_home="$(dirname "$(dirname "$wrapper")")"
    source_bin="$jdk_home/lib/svm/bin/native-image-configure"
    if [[ -s "$source_bin" ]]; then
      cp "$source_bin" "$wrapper"
      chmod +x "$wrapper"
      repaired_wrapper=1
    fi
  done < <(find "$GRADLE_USER_HOME_DIR/jdks" -type f -path "*/bin/native-image-configure" -size 0 2>/dev/null)

  if [[ "$repaired_wrapper" -eq 1 ]]; then
    echo "🔧 Repaired Gradle GraalVM wrapper binaries in $GRADLE_USER_HOME_DIR/jdks."
  fi

  # Some extracted binaries can lose the executable bit on macOS; fix when possible.
  find "$GRADLE_USER_HOME_DIR/jdks" -type f -path "*/bin/native-image" -size +0 -exec chmod +x {} \; 2>/dev/null || true
  find "$GRADLE_USER_HOME_DIR/jdks" -type f -path "*/bin/native-image-configure" -size +0 -exec chmod +x {} \; 2>/dev/null || true
fi

echo "🔨 Building Kotlin native executable for macOS (this can take a while)..."
if [[ $VERBOSE -eq 1 ]]; then
  (
    cd "$BACK_DIR"
    GRADLE_USER_HOME="$GRADLE_USER_HOME_DIR" \
      GRADLE_OPTS="$GRADLE_OPTS_VALUE" \
      NATIVE_IMAGE_OPTIONS="$NATIVE_IMAGE_OPTIONS_VALUE" \
      ./gradlew --no-daemon -Dorg.gradle.java.installations.auto-download=true clean nativeCompile
  )
else
  if ! (
    cd "$BACK_DIR"
    GRADLE_USER_HOME="$GRADLE_USER_HOME_DIR" \
      GRADLE_OPTS="$GRADLE_OPTS_VALUE" \
      NATIVE_IMAGE_OPTIONS="$NATIVE_IMAGE_OPTIONS_VALUE" \
      ./gradlew --no-daemon -Dorg.gradle.java.installations.auto-download=true clean nativeCompile
  ) >/dev/null 2>&1; then
    echo "❌ Native build failed. Re-run with --verbose for details."
    echo "ℹ️ If the error mentions native-image startup, retry once with:"
    echo "   rm -rf \"$GRADLE_USER_HOME_DIR/jdks\" && ./build-kotlin-macos.zsh --verbose"
    exit 1
  fi
fi

if [[ ! -f "$NATIVE_BINARY" ]]; then
  echo "❌ Native build failed: binary not found at $NATIVE_BINARY"
  exit 1
fi

cp "$NATIVE_BINARY" "$ROOT_DIR/$OUTPUT_BINARY"
chmod +x "$ROOT_DIR/$OUTPUT_BINARY"

# On recent macOS versions, native executables can be killed by taskgated
# after copy with "Code Signature Invalid". Re-sign ad-hoc to stabilize launch.
if command -v codesign >/dev/null 2>&1; then
  if ! codesign --force --sign - --timestamp=none "$ROOT_DIR/$OUTPUT_BINARY" >/dev/null 2>&1; then
    echo "⚠️ Could not ad-hoc sign ./$OUTPUT_BINARY (it may still run)."
  fi
fi

echo "📦 Native macOS binary ready: ./$OUTPUT_BINARY"

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
echo "✅ Kotlin macOS local native build completed in $elapsed_time seconds."
echo "ℹ️ Run with: ./$OUTPUT_BINARY"
