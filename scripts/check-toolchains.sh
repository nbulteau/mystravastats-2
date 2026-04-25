#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

fail() {
  echo "toolchain check failed: $*" >&2
  exit 1
}

require_file_contains() {
  local file="$1"
  local expected="$2"
  if ! grep -Fq "$expected" "$file"; then
    fail "$file does not contain expected text: $expected"
  fi
}

go_version="$(awk '/^go / {print $2; exit}' back-go/go.mod)"
[[ -n "$go_version" ]] || fail "unable to read Go version from back-go/go.mod"

java_version="$(sed -n 's/.*JavaLanguageVersion\.of(\([0-9][0-9]*\)).*/\1/p' back-kotlin/build.gradle.kts | head -n 1)"
[[ -n "$java_version" ]] || fail "unable to read Java version from back-kotlin/build.gradle.kts"

gradle_version="$(sed -n 's#.*gradle-\([0-9][0-9.]*\)-bin\.zip.*#\1#p' back-kotlin/gradle/wrapper/gradle-wrapper.properties | head -n 1)"
[[ -n "$gradle_version" ]] || fail "unable to read Gradle version from back-kotlin/gradle/wrapper/gradle-wrapper.properties"

node_version="$(sed -n 's/.*"node"[[:space:]]*:[[:space:]]*">=\([0-9][0-9.]*\)".*/\1/p' front-vue/package.json | head -n 1)"
[[ -n "$node_version" ]] || fail "unable to read Node.js version from front-vue/package.json"

require_file_contains back-go/Dockerfile "FROM golang:${go_version}-alpine AS build"
require_file_contains build-go-macos.zsh "golang:${go_version}"
require_file_contains build-go-ubuntu.sh "golang:${go_version}"
require_file_contains build-go-windows.ps1 "golang:${go_version}"
require_file_contains .github/workflows/ci.yml "go-version: \"${go_version}\""
require_file_contains .github/workflows/build-go-manual.yml "go-version: \"${go_version}\""

require_file_contains back-kotlin/Dockerfile "FROM gradle:${gradle_version}-jdk${java_version} AS build"
require_file_contains back-kotlin/Dockerfile "FROM eclipse-temurin:${java_version}-jre AS runtime"
require_file_contains .github/workflows/ci.yml "java-version: \"${java_version}\""

require_file_contains front-vue/Dockerfile "FROM node:${node_version}-alpine AS build"
require_file_contains build-go-macos.zsh "node:${node_version}"
require_file_contains build-go-ubuntu.sh "node:${node_version}"
require_file_contains build-go-windows.ps1 "node:${node_version}"
require_file_contains .github/workflows/ci.yml "node-version: \"${node_version}\""
require_file_contains .github/workflows/build-go-manual.yml "node-version: \"${node_version}\""

echo "Toolchains aligned: Go ${go_version}, Java ${java_version}, Gradle ${gradle_version}, Node.js ${node_version}"
