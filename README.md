# GLive - GitHub to Live

> Automatically clone and run any GitHub project with AI-powered setup and dependency management.

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://go.dev)
[![Node Version](https://img.shields.io/badge/Node-18+-339933?logo=node.js)](https://nodejs.org)

## 🎯 Problem

GitHub is filled with amazing projects, but getting them to run locally is often frustrating:
- Missing setup instructions
- Dependency conflicts
- Architecture mismatches
- Unclear error messages
- Time-consuming troubleshooting

**GLive solves this.** It uses AI to understand any project and get it running with a single command.

## ✨ Features

- **🚀 One-Command Setup** - Clone and run projects instantly
- **🤖 AI-Powered Analysis** - DeepSeek analyzes README, code, and dependencies
- **🛡️ Safety First** - Security scanning for suspicious patterns
- **🔄 Smart Recovery** - Rollback capabilities and idempotent operations
- **🎯 Three Modes**:
  - **Auto**: Fully automatic (AI fixes errors)
  - **Assisted**: Approve each action
  - **Manual**: Instructions only
- **📊 Web Dashboard** - Visual interface with real-time progress
- **🌐 PWA Support** - Install as a desktop app
- **🔌 Multi-Platform** - Support for Node.js, Python, Go, Rust, Java

## 🚦 Quick Start

### Installation

#### Option 1: Using Go (CLI only) recomanded for quick setup
```bash
# Install Go from https://go.dev/dl/ if not already installed
go install github.com/yasinhessnawi1/GLive/cmd/GLive@latest
```

#### Option 2: From Source (Full stack)
```bash
# Clone the repository
git clone https://github.com/yasinhessnawi1/GLive.git
cd GLive

# Build CLI
cd cmd/glive
go build -o glive
sudo mv glive /usr/local/bin/  # or add to PATH

# Build and run agent (optional, for web dashboard)
cd ../../pkg/agent
go run main.go
```

### Configuration

```bash
# Set your DeepSeek API key (get one at https://platform.deepseek.com)
glive config set api-key YOUR_DEEPSEEK_API_KEY

# Optional: Configure workspace directory
glive config set workspace-dir ~/my-glive-projects

# View configuration
glive config list
```

### Usage

#### CLI

```bash
# Run a project (auto mode)
glive https://github.com/user/awesome-project

# Run with assisted mode (approve each action)
glive https://github.com/user/project --mode assisted

# Run with manual mode (instructions only)
glive user/project --mode manual

# List all projects
glive list

# Check project status
glive status

# Clean up a project
glive cleanup project-id
```

#### Web Dashboard

1. Start the GLive agent:
```bash
cd pkg/agent
go run main.go
# Agent runs on http://localhost:8080
```

2. Start the web dashboard:
```bash
cd web
npm install
npm run dev
# Dashboard available at http://localhost:3000
```

3. Open your browser and enter a GitHub URL!

## 📁 Project Structure

```
glive/
├── cmd/glive/          # CLI application
├── pkg/
│   ├── core/          # Shared core library
│   │   ├── analyzer/  # Project type detection & analysis
│   │   ├── executor/  # Command execution
│   │   ├── ai/        # AI client (DeepSeek)
│   │   ├── scanner/   # Security scanner
│   │   ├── state/     # State management
│   │   ├── config/    # Configuration
│   │   └── repo/      # Git operations
│   └── agent/         # Local API server (Fiber)
└── web/               # Next.js PWA dashboard
```

See [ARCHITECTURE.md](./ARCHITECTURE.md) for detailed architecture.

## 🔧 How It Works

1. **Clone**: GLive clones the GitHub repository
2. **Analyze**: Detects project type, package managers, dependencies
3. **AI Review**: DeepSeek reads README and code to understand setup
4. **Security Scan**: Checks for suspicious patterns
5. **Execute**: Runs setup commands (install dependencies, build, etc.)
6. **Debug**: If errors occur, AI suggests fixes
7. **Ready**: Project is running and ready to use

## 🎨 Supported Project Types

- ✅ Node.js (npm, yarn, pnpm)
- ✅ Python (pip, pipenv, poetry)
- ✅ Go (go modules)
- ✅ Rust (cargo)
- ✅ Java (Maven, Gradle)
- ✅ Docker (docker-compose)
- 🔄 More coming soon!

## 🛡️ Security

GLive includes built-in security scanning to detect:
- Dangerous commands (`rm -rf`, `eval()`, `exec()`)
- Privilege escalation (`sudo`)
- Suspicious network calls
- Code injection patterns

You'll always be warned about potential security issues before execution.

## 🤝 Contributing

We welcome contributions! See [CONTRIBUTING.md](./CONTRIBUTING.md) for:
- Development setup
- Coding standards
- How to add new features
- Testing guidelines

## 📖 Documentation

- [Architecture](./ARCHITECTURE.md) - System design and components
- [Contributing](./CONTRIBUTING.md) - Development guide
- [Setup](./SETUP.md) - Detailed setup instructions

## 🗺️ Roadmap

### v0.2.0
- [ ] Docker sandbox support
- [ ] More AI providers (OpenAI, Claude)
- [ ] Database setup (PostgreSQL, MySQL)
- [ ] Enhanced rollback

### v0.3.0
- [ ] Desktop app (Electron/Tauri)
- [ ] Project templates
- [ ] Team collaboration
- [ ] Plugin system

### v1.0.0
- [ ] Production-ready
- [ ] Complete test coverage
- [ ] Performance optimizations
- [ ] Enterprise features

## 💡 Examples

```bash
# Run a React app
glive https://github.com/facebook/create-react-app

# Run a Python ML project
glive https://github.com/tensorflow/models

# Run a Go API server
glive https://github.com/gofiber/recipes

# Run with assisted mode for learning
glive https://github.com/some/project --mode assisted
```

## 📝 License

MIT License - see [LICENSE](./LICENSE) for details

## 🙏 Acknowledgments

- Built with [Go](https://go.dev), [Next.js](https://nextjs.org), and [Fiber](https://gofiber.io)
- Powered by [DeepSeek AI](https://www.deepseek.com)
- Inspired by the pain of setting up projects

## 📧 Support

- 🐛 [Report a bug](https://github.com/yourusername/GLive/issues)
- 💡 [Request a feature](https://github.com/yourusername/GLive/issues)
- 💬 [Discussions](https://github.com/yourusername/GLive/discussions)

---

Made with ❤️ for developers who want things to just work
