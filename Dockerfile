# Start from the official Go Alpine image (lightweight)
FROM golang:alpine

# Set the working directory inside the container
WORKDIR /app

# Copy the Go module files first (for better caching)
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY *.go ./

# Build the application inside the container
RUN go build -o fix main.go

# Expose port 8080 for the Spotify Authentication Callback
EXPOSE 8080

# Command to run the executable
CMD ["./fix"]