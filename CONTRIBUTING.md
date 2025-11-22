# Contributing to GLive

Thank you for your interest in contributing to GLive!

## Development Setup

### Prerequisites

1. **Go 1.21+** - [Install Go](https://go.dev/dl/)
2. **Node.js 18+** - [Install Node.js](https://nodejs.org/)
3. **Git** - [Install Git](https://git-scm.com/)

### Initial Setup

1. Clone the repository:
```bash
git clone https://github.com/yourusername/glive.git
cd glive
```

2. Initialize Go modules:
```bash
# Core library
cd pkg/core
go mod download

# Agent/Server
cd ../agent
go mod download

# CLI
cd ../../cmd/glive
go mod download
```

3. Install web dependencies:
```bash
cd ../../web
npm install
```

## Project Structure

```
glive/
├── cmd/glive/          # CLI application
│   ├── cmd/           # Cobra commands
│   └── main.go        # Entry point
├── pkg/
│   ├── core/          # Shared core library
│   │   ├── analyzer/  # Project analysis
│   │   ├── executor/  # Command execution
│   │   ├── ai/        # AI client
│   │   ├── scanner/   # Security scanner
│   │   ├── state/     # State management
│   │   ├── config/    # Configuration
│   │   └── repo/      # Git operations
│   └── agent/         # Local API server
│       ├── server/    # Fiber server
│       └── handlers/  # API handlers
└── web/               # Next.js PWA frontend
    ├── src/
    │   ├── app/       # Next.js app directory
    │   ├── components/# React components
    │   └── lib/       # Utilities
    └── public/        # Static assets
```

## Development Workflow

### Running the CLI

```bash
cd cmd/glive
go run main.go --help
```

### Running the Agent

```bash
cd pkg/agent
go run main.go
```

### Running the Web App

```bash
cd web
npm run dev
# Open http://localhost:3000
```

## Making Changes

1. Create a new branch:
```bash
git checkout -b feature/your-feature-name
```

2. Make your changes following our coding standards

3. Test your changes:
```bash
# Go code
go test ./...

# Web code
npm run lint
npm run build
```

4. Commit with a clear message:
```bash
git commit -m "Add: description of your feature"
```

5. Push and create a pull request

## Coding Standards

### Go Code
- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Use `gofmt` to format code
- Add comments for exported functions
- Write tests for new functionality

### TypeScript/React
- Use TypeScript for type safety
- Follow React best practices
- Use functional components with hooks
- Keep components small and focused

## Adding New Features

### Adding a New Project Type

1. Update `pkg/core/types.go` with new project type
2. Add detection logic in `pkg/core/analyzer/analyzer.go`
3. Add command generation in analyzer
4. Update tests

### Adding a New AI Provider

1. Update `pkg/core/ai/client.go`
2. Add provider-specific endpoint and model
3. Update configuration options
4. Test with the new provider

### Adding New CLI Commands

1. Create command file in `cmd/glive/cmd/`
2. Register in `root.go`
3. Implement command logic
4. Update documentation

## Testing

### Unit Tests
```bash
# Test all packages
go test ./...

# Test specific package
go test ./pkg/core/analyzer
```

### Integration Tests
Coming soon

## Documentation

- Update README.md for user-facing changes
- Update CONTRIBUTING.md for developer changes
- Add inline comments for complex logic
- Update API documentation for endpoint changes

## Need Help?

- Open an issue for bugs or feature requests
- Join our community discussions
- Check existing issues and PRs

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
