# GLive Development Setup

## Prerequisites

### 1. Install Go (Required)

**Windows:**
```bash
# Using Chocolatey
choco install golang

# Or download from https://go.dev/dl/
# Install Go 1.21 or higher
```

**Verify installation:**
```bash
go version
# Should show: go version go1.21.x or higher
```

### 2. Install Node.js (Required for web dashboard)

**Windows:**
```bash
# Using Chocolatey
choco install nodejs

# Or download from https://nodejs.org/
```

**Verify installation:**
```bash
node --version
npm --version
```

## Initial Setup

Once Go is installed, run:

```bash
# Initialize Go modules
cd pkg/core
go mod init github.com/glive/core
go mod tidy

cd ../agent
go mod init github.com/glive/agent
go mod tidy

cd ../../cmd/glive
go mod init github.com/glive/cmd/glive
go mod tidy

# Install dependencies for web
cd ../../web
npm install

# Return to root
cd ..
```

## Development

```bash
# Run CLI
cd cmd/glive
go run main.go

# Run agent
cd pkg/agent
go run main.go

# Run web dashboard
cd web
npm run dev
```

## Building

```bash
# Build CLI
cd cmd/glive
go build -o ../../bin/glive

# Build web
cd web
npm run build
```
