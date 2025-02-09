#!/bin/bash

# Build the UI project
docker run --rm -v "$PWD:/app" -w /app/ui node:latest \
    sh -c "npm install && npm run build"

# Copy the UI build to the back-go/public directory
mkdir -p back-go/public
cp -r ui/dist/* back-go/public/

# Build for Ubuntu
docker run --rm -v "$PWD:/app" -w /app golang:latest \
    sh -c "cd back-go && GOOS=linux GOARCH=amd64 go build -o ../mystravastats-linux"