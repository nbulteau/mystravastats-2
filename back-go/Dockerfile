# Use the same Go version declared in go.mod.
FROM golang:1.26.2-alpine AS build

# Set the working directory
WORKDIR /app

# Copy the Go modules files
COPY go.mod go.sum ./

# Download the Go modules
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN go build -o mystravastats .

# Use a minimal image for the runtime
FROM alpine:latest AS runtime

# Install CA certificates for Strava HTTPS calls and curl for container healthchecks.
RUN apk add --no-cache ca-certificates curl

ENV SERVER_HOST=0.0.0.0
ENV PORT=8080
ENV OPEN_BROWSER=false

# Set the working directory
WORKDIR /app

# Copy the built application from the build stage
COPY --from=build /app/mystravastats /app/mystravastats
COPY famous-climb/ /app/famous-climb/

# Expose the port the application runs on
EXPOSE 8080
EXPOSE 8090

# Run the application
ENTRYPOINT ["/app/mystravastats"]
