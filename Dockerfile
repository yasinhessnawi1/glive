# Build stage
FROM golang:1.24-bookworm AS builder

WORKDIR /app

# Copy the entire source code
COPY . .

# Build the agent from its directory
WORKDIR /app/pkg/agent
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/agent .

# Run stage
FROM debian:bookworm-slim

WORKDIR /app

# Install runtime dependencies
RUN apt-get update && apt-get install -y \
    ca-certificates \
    git \
    curl \
    wget \
    nodejs \
    npm \
    python3 \
    python3-pip \
    zip \
    unzip \
    && rm -rf /var/lib/apt/lists/*

# Copy the pre-built binary file from the previous stage
COPY --from=builder /app/agent .

# Create a directory for the workspace
RUN mkdir -p /app/workspace
ENV WORKSPACE_DIR=/app/workspace

# Expose port 8080 to the outside world
EXPOSE 8080

# Command to run the executable
CMD ["./agent", "-port", "8080"]
