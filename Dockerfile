# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN make build

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy binary from builder
COPY --from=builder /app/azure-pr-cli .

# Set environment variables
ENV AZURE_DEVOPS_ORG=""
ENV AZURE_DEVOPS_PROJECT=""
ENV AZURE_DEVOPS_PAT=""

ENTRYPOINT ["./azure-pr-cli"]
CMD ["--help"]
