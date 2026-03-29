# Start from the official Go Alpine image (lightweight)
FROM golang:alpine

# Set the working directory inside the container
WORKDIR /app

# Add timezone support
RUN apk add --no-cache tzdata

# Copy the Go module files first (for better caching)
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY src/ ./src/

# Build the application inside the container
RUN go build -o fix ./src

# Expose port 8080 for the Spotify Authentication Callback
EXPOSE 8080

# Set the working directory to where the JSON files should go
WORKDIR /app/run

# Command to run the executable
CMD ["/app/fix"]