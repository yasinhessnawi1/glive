#!/bin/bash

# GLive Initialization Script
# This script sets up the development environment

set -e

echo "🚀 GLive - Initialization Script"
echo "================================="
echo ""

# Check Go installation
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed!"
    echo "Please install Go from https://go.dev/dl/"
    echo ""
    echo "Windows: choco install golang"
    echo "macOS: brew install go"
    echo "Linux: sudo apt install golang-go"
    exit 1
fi

echo "✅ Go $(go version | awk '{print $3}') found"

# Check Node.js installation
if ! command -v node &> /dev/null; then
    echo "❌ Node.js is not installed!"
    echo "Please install Node.js from https://nodejs.org/"
    exit 1
fi

echo "✅ Node.js $(node --version) found"
echo ""

# Initialize Go modules
echo "📦 Initializing Go modules..."

echo "  - pkg/core"
cd pkg/core
go mod tidy

echo "  - pkg/agent"
cd ../agent
go mod tidy

echo "  - cmd/glive"
cd ../../cmd/glive
go mod tidy

cd ../..

echo "✅ Go modules initialized"
echo ""

# Install web dependencies
echo "📦 Installing web dependencies..."
cd web
npm install
cd ..

echo "✅ Web dependencies installed"
echo ""

# Create workspace directory
WORKSPACE_DIR="$HOME/glive-workspace"
if [ ! -d "$WORKSPACE_DIR" ]; then
    mkdir -p "$WORKSPACE_DIR"
    echo "✅ Created workspace directory: $WORKSPACE_DIR"
fi

# Create state directory
STATE_DIR="$HOME/.glive"
if [ ! -d "$STATE_DIR" ]; then
    mkdir -p "$STATE_DIR"
    echo "✅ Created state directory: $STATE_DIR"
fi

echo ""
echo "🎉 Initialization complete!"
echo ""
echo "Next steps:"
echo "1. Configure your API key:"
echo "   cd cmd/glive && go run main.go config set api-key YOUR_DEEPSEEK_API_KEY"
echo ""
echo "2. Build the CLI:"
echo "   cd cmd/glive && go build -o glive"
echo ""
echo "3. Run the agent (optional, for web dashboard):"
echo "   cd pkg/agent && go run main.go"
echo ""
echo "4. Run the web dashboard (optional):"
echo "   cd web && npm run dev"
echo ""
echo "5. Test the CLI:"
echo "   cd cmd/glive && go run main.go version"
echo ""
