# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Install dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o bot .

# Final stage
FROM alpine:3.19

WORKDIR /app

# Install CA certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

# Copy the binary from the builder
COPY --from=builder /app/bot .

# Create a data directory for persistent state
RUN mkdir /data

# Set environment variables
ENV STATE_FILE_PATH=/data/state.json

# Run the bot
CMD ["./bot"]
