# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code (now includes the shell script)
COPY . .

# Make the shell script executable
RUN chmod +x calculate_checksums.sh

# Build the applications
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -o calculate_checksums ./cmd/calculate_checksums

# Runtime stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binaries and script from builder stage
COPY --from=builder /app/server .
COPY --from=builder /app/calculate_checksums .
COPY --from=builder /app/calculate_checksums.sh .

# Expose port
EXPOSE 8080

# Run the binary
CMD ["./server"]