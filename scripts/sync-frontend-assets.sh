#!/usr/bin/env sh

set -eu

usage() {
  cat <<'USAGE'
Usage: scripts/sync-frontend-assets.sh <go|kotlin> [--skip-build]

Copies the Vite production build into the backend-specific generated assets
directory. By default the frontend is rebuilt first. Use --skip-build only when
the caller has already produced a fresh front-vue/dist artifact in this run.
USAGE
}

fail() {
  echo "sync-frontend-assets: $*" >&2
  exit 1
}

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
repo_root=$(CDPATH= cd -- "$script_dir/.." && pwd)

target="${1:-}"
skip_build=0

if [ "$target" = "" ]; then
  usage
  exit 2
fi

if [ "${2:-}" = "--skip-build" ]; then
  skip_build=1
elif [ "${2:-}" != "" ]; then
  usage
  exit 2
fi

front_dir="$repo_root/front-vue"
dist_dir="$front_dir/dist"

case "$target" in
  go)
    destination="$repo_root/back-go/public"
    ;;
  kotlin)
    destination="$repo_root/back-kotlin/build/generated/frontend-static"
    ;;
  *)
    usage
    exit 2
    ;;
esac

if [ "$skip_build" -eq 0 ]; then
  command -v npm >/dev/null 2>&1 || fail "npm is required to build front-vue"
  [ -f "$front_dir/package.json" ] || fail "missing front-vue/package.json"
  (
    cd "$front_dir"
    npm run build
  )
fi

[ -f "$dist_dir/index.html" ] || fail "missing front-vue/dist/index.html; run npm run build first"

rm -rf "$destination"
mkdir -p "$destination"
cp -R "$dist_dir"/. "$destination"/

echo "Synced front-vue/dist to ${destination#$repo_root/}"
