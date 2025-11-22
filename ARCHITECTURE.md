# GLive Architecture

## Overview

GLive is a tool that automatically clones and runs GitHub projects with AI-powered setup and dependency management. It consists of three main components:

1. **CLI Tool** - Command-line interface for direct usage
2. **Local Agent** - Background API server for web integration
3. **Web Dashboard (PWA)** - Progressive web app for visual management

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                         User Layer                          │
├─────────────────────┬───────────────────┬──────────────────┤
│   CLI Interface     │   Web Dashboard   │  Desktop App     │
│   (Cobra)           │   (Next.js PWA)   │  (Future)        │
└──────────┬──────────┴─────────┬─────────┴──────────────────┘
           │                    │
           │                    │ HTTP/WebSocket
           │                    │
           ▼                    ▼
┌──────────────────────────────────────────────────────────────┐
│                     Core Library (Go)                        │
├──────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌──────────────┐  ┌─────────────────┐    │
│  │  Repository │  │   Analyzer   │  │   AI Client     │    │
│  │   Handler   │  │              │  │  (DeepSeek)     │    │
│  └─────────────┘  └──────────────┘  └─────────────────┘    │
│  ┌─────────────┐  ┌──────────────┐  ┌─────────────────┐    │
│  │  Executor   │  │    Scanner   │  │  State Manager  │    │
│  │             │  │  (Security)  │  │                 │    │
│  └─────────────┘  └──────────────┘  └─────────────────┘    │
│  ┌─────────────────────────────────────────────────────┐    │
│  │           Configuration Manager                     │    │
│  └─────────────────────────────────────────────────────┘    │
└──────────────────────────────────────────────────────────────┘
           │                    │
           ▼                    ▼
┌──────────────────┐  ┌──────────────────────────────────────┐
│  Local Agent     │  │      External Services               │
│  (Fiber Server)  │  │  ┌─────────────┐  ┌──────────────┐  │
│                  │  │  │ DeepSeek AI │  │   GitHub     │  │
│  - REST API      │  │  └─────────────┘  └──────────────┘  │
│  - WebSocket     │  └──────────────────────────────────────┘
└──────────────────┘
           │
           ▼
┌──────────────────────────────────────────────────────────────┐
│                    File System                               │
│  - Cloned repositories (~/glive-workspace/)                 │
│  - State files (.glive/)                                     │
│  - Configuration (~/.glive.json)                            │
└──────────────────────────────────────────────────────────────┘
```

## Component Details

### 1. CLI Tool (`cmd/glive`)

**Purpose**: Direct command-line interface for users

**Key Features**:
- Parse GitHub URLs
- Execute in different modes (auto, assisted, manual)
- Display progress and logs
- Manage configuration

**Commands**:
- `glive <url>` - Run a project
- `glive config` - Manage configuration
- `glive status` - Show project status
- `glive cleanup` - Clean up projects
- `glive list` - List all projects

### 2. Core Library (`pkg/core`)

**Purpose**: Shared business logic used by both CLI and Agent

#### Repository Handler (`repo/`)
- Parse GitHub URLs (HTTPS, SSH, shorthand)
- Clone repositories
- Validate repository structure

#### Analyzer (`analyzer/`)
- Detect project type (Node.js, Python, Go, etc.)
- Identify package managers
- Find entry points
- Extract dependencies
- Parse README files
- Generate setup commands

#### AI Client (`ai/`)
- Interface with AI APIs (DeepSeek, OpenAI, Claude)
- Analyze projects with AI
- Debug errors with AI suggestions
- Generate setup instructions

#### Executor (`executor/`)
- Execute shell commands
- Stream output in real-time
- Handle errors and retries
- Check command availability
- Get installed versions

#### Scanner (`scanner/`)
- Detect suspicious code patterns
- Security analysis
- Flag dangerous operations
- Generate security reports

#### State Manager (`state/`)
- Persist project state
- Track operations for rollback
- Manage project lifecycle
- Support idempotent operations

#### Config Manager (`config/`)
- Load/save configuration
- Manage API keys
- Set preferences
- Initialize workspace

### 3. Local Agent (`pkg/agent`)

**Purpose**: API server for web dashboard integration

**Technology**: Fiber (Go web framework)

**API Endpoints**:
```
GET  /health                     - Health check
GET  /api/v1/projects            - List projects
POST /api/v1/projects            - Create project
GET  /api/v1/projects/:id        - Get project
DELETE /api/v1/projects/:id      - Delete project
POST /api/v1/projects/:id/start  - Start project
POST /api/v1/projects/:id/stop   - Stop project
POST /api/v1/projects/:id/cleanup - Cleanup project
GET  /api/v1/config              - Get configuration
PUT  /api/v1/config              - Update configuration
WS   /api/v1/ws/:project_id      - WebSocket for real-time updates
```

### 4. Web Dashboard (`web/`)

**Purpose**: Visual interface for managing projects

**Technology**:
- Next.js 14 (App Router)
- TypeScript
- Tailwind CSS
- PWA (installable)

**Features**:
- Project input with URL validation
- Mode selection (auto/assisted/manual)
- Project list with status
- Real-time progress updates via WebSocket
- Settings management
- Dark mode support

## Data Flow

### Running a New Project

```
1. User Input
   └─> CLI or Web Dashboard

2. URL Parsing
   └─> Repository Handler validates and parses GitHub URL

3. Cloning
   └─> Repository Handler clones to workspace

4. Analysis
   ├─> Analyzer detects project type
   ├─> Scanner checks for security issues
   └─> AI Client analyzes README and code

5. Setup Commands
   └─> Analyzer generates command sequence

6. Execution
   ├─> Executor runs commands
   ├─> State Manager records operations
   └─> Progress updates sent to user

7. Completion
   ├─> Project marked as ready
   └─> User notified with next steps
```

## Execution Modes

### Auto Mode
- Fully automatic execution
- AI debugs and fixes errors
- Minimal user interaction
- Best for trusted repositories

### Assisted Mode
- User confirms each action
- Review commands before execution
- AI explains each step
- Balance of automation and control

### Manual Mode
- Instructions only
- User executes commands
- Full transparency
- Best for learning or untrusted code

## State Management

### Project State
```json
{
  "id": "project-uuid",
  "name": "repo-name",
  "github_url": "https://github.com/user/repo",
  "local_path": "/home/user/glive-workspace/repo",
  "type": "nodejs",
  "status": "ready",
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:05:00Z"
}
```

### Operations Log
```json
{
  "operations": [
    {
      "id": "op-1",
      "type": "clone",
      "description": "Clone repository",
      "timestamp": "2024-01-01T00:00:00Z",
      "reversible": true
    },
    {
      "id": "op-2",
      "type": "command",
      "description": "npm install",
      "timestamp": "2024-01-01T00:02:00Z",
      "reversible": false
    }
  ]
}
```

## Security Considerations

### Code Scanning
- Detect `eval()`, `exec()`, dangerous file operations
- Flag network calls to unknown hosts
- Warn about privilege escalation
- Check for crypto mining patterns

### Sandboxing (Future)
- Docker container isolation
- Resource limits
- Network restrictions
- Filesystem isolation

### User Confirmation
- Required for high-severity findings
- Transparent about all operations
- Rollback capability
- Clear security warnings

## Configuration

### User Configuration (~/.glive.json)
```json
{
  "api_key": "your-api-key",
  "api_provider": "deepseek",
  "api_endpoint": "",
  "default_mode": "auto",
  "workspace_dir": "~/glive-workspace",
  "max_concurrent": 3,
  "enable_sandbox": false,
  "agent_port": 8080
}
```

## Future Enhancements

### Phase 2
- Docker support for sandboxing
- More AI providers (Anthropic Claude, OpenAI)
- Database project support (MySQL, PostgreSQL)
- Enhanced rollback capabilities

### Phase 3
- Desktop app (Electron or Tauri)
- Project templates
- Shareable configurations
- Team collaboration features

### Phase 4
- CI/CD integration
- Cloud deployment
- Custom AI training
- Plugin system

## Performance Considerations

### Optimization Strategies
- Concurrent dependency downloads
- Caching of AI responses
- Incremental analysis
- Lazy loading of modules

### Resource Management
- Limit concurrent projects
- Clean up old workspaces
- Stream large outputs
- Efficient state serialization

## Testing Strategy

### Unit Tests
- Each core module independently
- Mock external dependencies
- Test error cases

### Integration Tests
- End-to-end flows
- Real GitHub repositories
- Various project types

### Security Tests
- Scanner effectiveness
- Malicious code detection
- Sandbox escape prevention

## Deployment

### CLI Distribution
- Go binaries for all platforms
- Package managers (npm, homebrew, chocolatey)
- Auto-update mechanism

### Web Dashboard
- Static export from Next.js
- Served by local agent
- PWA for offline capability

### Agent
- Background service
- Auto-start on system boot
- Graceful shutdown
