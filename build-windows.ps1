# Build the UI project
docker run --rm -v "${PWD}:/app" -w /app/ui node:latest `
    sh -c "npm install && npm run build"

# Copy the UI build to the back-go/public directory
if (-Not (Test-Path -Path "back-go/public")) {
    New-Item -ItemType Directory -Path "back-go/public"
}
Copy-Item -Recurse -Force -Path "ui/dist/*" -Destination "back-go/public/"

# Build for Windows
docker run --rm -v "${PWD}:/app" -w /app golang:latest `
    sh -c "cd back-go && GOOS=windows GOARCH=amd64 go build -o ../mystravastats.exe"