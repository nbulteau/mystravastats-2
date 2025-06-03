# Use an official Golang image to build the application
FROM golang:1.24.3-alpine AS build

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