import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import {
  Rocket,
  Terminal,
  Settings,
  Shield,
  Zap,
  HelpCircle,
  Code,
  Play,
  Square,
  FolderOpen,
  Cpu,
  AlertTriangle,
  CheckCircle2,
  FileJson,
  Bot
} from "lucide-react"

export default function DocsPage() {
  return (
    <div className="container mx-auto px-4 py-8 max-w-6xl">
      {/* Header */}
      <div className="text-center mb-12">
        <h1 className="text-4xl font-bold mb-4">GLive Documentation</h1>
        <p className="text-xl text-muted-foreground mb-6">
          Everything you need to know to use GLive effectively
        </p>
        <div className="flex justify-center gap-2 flex-wrap">
          <Badge variant="secondary">CLI Tool</Badge>
          <Badge variant="secondary">TUI Interface</Badge>
          <Badge variant="secondary">Web Dashboard</Badge>
          <Badge variant="secondary">AI-Powered</Badge>
        </div>
      </div>

      <Tabs defaultValue="getting-started" className="w-full">
        <TabsList className="grid w-full grid-cols-3 md:grid-cols-6 mb-8">
          <TabsTrigger value="getting-started">Get Started</TabsTrigger>
          <TabsTrigger value="usage">Usage</TabsTrigger>
          <TabsTrigger value="features">Features</TabsTrigger>
          <TabsTrigger value="projects">Projects</TabsTrigger>
          <TabsTrigger value="config">Config</TabsTrigger>
          <TabsTrigger value="troubleshooting">Help</TabsTrigger>
        </TabsList>

        {/* Getting Started Tab */}
        <TabsContent value="getting-started" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Rocket className="h-5 w-5" />
                What is GLive?
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <p>
                <strong>GLive</strong> is a tool that automatically clones and runs GitHub projects.
                It handles dependency installation, project detection, and can even recover from
                errors using AI. No more reading lengthy README files or debugging setup issues.
              </p>
              <div className="grid md:grid-cols-3 gap-4 mt-4">
                <div className="p-4 rounded-lg bg-muted">
                  <Zap className="h-6 w-6 text-primary mb-2" />
                  <h4 className="font-semibold">Instant</h4>
                  <p className="text-sm text-muted-foreground">One command to clone and run any project</p>
                </div>
                <div className="p-4 rounded-lg bg-muted">
                  <Bot className="h-6 w-6 text-primary mb-2" />
                  <h4 className="font-semibold">AI-Powered</h4>
                  <p className="text-sm text-muted-foreground">Smart error recovery and troubleshooting</p>
                </div>
                <div className="p-4 rounded-lg bg-muted">
                  <Shield className="h-6 w-6 text-primary mb-2" />
                  <h4 className="font-semibold">Safe</h4>
                  <p className="text-sm text-muted-foreground">Sandboxed execution with security checks</p>
                </div>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Quick Install</CardTitle>
              <CardDescription>Get GLive installed in seconds</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div>
                <h4 className="font-semibold mb-2">Linux / macOS</h4>
                <pre className="bg-muted p-4 rounded-lg text-sm overflow-x-auto">
                  <code>curl -sSL https://raw.githubusercontent.com/yasinhessnawi1/glive/main/install.sh | bash</code>
                </pre>
              </div>
              <div>
                <h4 className="font-semibold mb-2">Windows (PowerShell)</h4>
                <pre className="bg-muted p-4 rounded-lg text-sm overflow-x-auto">
                  <code>irm https://raw.githubusercontent.com/yasinhessnawi1/glive/main/install.ps1 | iex</code>
                </pre>
              </div>
              <div>
                <h4 className="font-semibold mb-2">Using Go</h4>
                <pre className="bg-muted p-4 rounded-lg text-sm overflow-x-auto">
                  <code>go install github.com/yasinhessnawi1/glive/cmd/glive@latest</code>
                </pre>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Your First Project</CardTitle>
              <CardDescription>Run a GitHub project in seconds</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <p className="font-medium">1. Run with a GitHub URL:</p>
                <pre className="bg-muted p-4 rounded-lg text-sm">
                  <code>glive https://github.com/facebook/react</code>
                </pre>
              </div>
              <div className="space-y-2">
                <p className="font-medium">2. Or use the short format:</p>
                <pre className="bg-muted p-4 rounded-lg text-sm">
                  <code>glive facebook/react</code>
                </pre>
              </div>
              <div className="space-y-2">
                <p className="font-medium">3. GLive will automatically:</p>
                <ul className="list-disc list-inside text-muted-foreground space-y-1 ml-4">
                  <li>Clone the repository</li>
                  <li>Detect the project type (Node.js, Python, Go, etc.)</li>
                  <li>Install all dependencies</li>
                  <li>Start the development server</li>
                </ul>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Usage Tab */}
        <TabsContent value="usage" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Terminal className="h-5 w-5" />
                CLI Commands
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-6">
              <div>
                <h4 className="font-semibold mb-2">Clone and Setup a Project</h4>
                <pre className="bg-muted p-4 rounded-lg text-sm">
                  <code>{`# Full URL
glive https://github.com/user/repo

# Short format
glive user/repo

# With force re-clone
glive user/repo --force`}</code>
                </pre>
              </div>

              <div>
                <h4 className="font-semibold mb-2">Run an Existing Project</h4>
                <pre className="bg-muted p-4 rounded-lg text-sm">
                  <code>{`# Run in current directory
glive run

# Run a specific project
glive run ./my-project

# Run with absolute path
glive run /home/user/projects/myapp`}</code>
                </pre>
              </div>

              <div>
                <h4 className="font-semibold mb-2">Stop a Running Project</h4>
                <pre className="bg-muted p-4 rounded-lg text-sm">
                  <code>{`# Stop by process name
glive stop node

# Stop by PID
glive stop 12345`}</code>
                </pre>
              </div>

              <div>
                <h4 className="font-semibold mb-2">Interactive TUI Mode</h4>
                <pre className="bg-muted p-4 rounded-lg text-sm">
                  <code>{`# Launch interactive dashboard
glive`}</code>
                </pre>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Execution Modes</CardTitle>
              <CardDescription>Choose how GLive handles your project</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                <div className="p-4 border rounded-lg">
                  <div className="flex items-center gap-2 mb-2">
                    <Badge>auto</Badge>
                    <span className="font-semibold">Automatic (Default)</span>
                  </div>
                  <p className="text-sm text-muted-foreground">
                    GLive handles everything automatically. Best for quick setups where you trust the project.
                  </p>
                  <pre className="bg-muted p-2 rounded text-xs mt-2">
                    <code>glive user/repo --mode auto</code>
                  </pre>
                </div>

                <div className="p-4 border rounded-lg">
                  <div className="flex items-center gap-2 mb-2">
                    <Badge variant="secondary">assisted</Badge>
                    <span className="font-semibold">Assisted</span>
                  </div>
                  <p className="text-sm text-muted-foreground">
                    GLive asks for approval before running each command. Good for learning or reviewing.
                  </p>
                  <pre className="bg-muted p-2 rounded text-xs mt-2">
                    <code>glive user/repo --mode assisted</code>
                  </pre>
                </div>

                <div className="p-4 border rounded-lg">
                  <div className="flex items-center gap-2 mb-2">
                    <Badge variant="outline">manual</Badge>
                    <span className="font-semibold">Manual</span>
                  </div>
                  <p className="text-sm text-muted-foreground">
                    GLive shows you what to do but doesnt run anything. You copy and run commands yourself.
                  </p>
                  <pre className="bg-muted p-2 rounded text-xs mt-2">
                    <code>glive user/repo --mode manual</code>
                  </pre>
                </div>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Play className="h-5 w-5 text-green-500" />
                TUI Controls
              </CardTitle>
              <CardDescription>Keyboard shortcuts in interactive mode</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="grid md:grid-cols-2 gap-4">
                <div>
                  <h4 className="font-semibold mb-2">During Setup</h4>
                  <ul className="space-y-2 text-sm">
                    <li className="flex justify-between"><span>Cancel setup</span><Badge variant="outline">X</Badge></li>
                    <li className="flex justify-between"><span>Scroll up/down</span><Badge variant="outline">Arrow Keys</Badge></li>
                    <li className="flex justify-between"><span>Jump to top/bottom</span><Badge variant="outline">Home/End</Badge></li>
                    <li className="flex justify-between"><span>Toggle auto-scroll</span><Badge variant="outline">F</Badge></li>
                    <li className="flex justify-between"><span>Quit</span><Badge variant="outline">Q</Badge></li>
                  </ul>
                </div>
                <div>
                  <h4 className="font-semibold mb-2">After Setup</h4>
                  <ul className="space-y-2 text-sm">
                    <li className="flex justify-between"><span>Run project</span><Badge variant="outline">R</Badge></li>
                    <li className="flex justify-between"><span>Open folder</span><Badge variant="outline">O</Badge></li>
                    <li className="flex justify-between"><span>Back to dashboard</span><Badge variant="outline">D</Badge></li>
                    <li className="flex justify-between"><span>Stop running project</span><Badge variant="outline">X</Badge></li>
                  </ul>
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Features Tab */}
        <TabsContent value="features" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Bot className="h-5 w-5" />
                AI-Powered Features
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="p-4 border rounded-lg">
                <h4 className="font-semibold mb-2">Smart Project Analysis</h4>
                <p className="text-sm text-muted-foreground">
                  GLive uses AI to understand your project structure, detect frameworks,
                  and determine the best way to set it up.
                </p>
              </div>
              <div className="p-4 border rounded-lg">
                <h4 className="font-semibold mb-2">Error Recovery</h4>
                <p className="text-sm text-muted-foreground">
                  When errors occur, AI analyzes the output and suggests fixes.
                  In auto mode, it can even apply fixes automatically.
                </p>
              </div>
              <div className="p-4 border rounded-lg">
                <h4 className="font-semibold mb-2">Port Conflict Resolution</h4>
                <p className="text-sm text-muted-foreground">
                  If a port is already in use, GLive automatically retries with a different port
                  (up to 5 attempts).
                </p>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Shield className="h-5 w-5" />
                Security Features
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="p-4 border rounded-lg">
                <h4 className="font-semibold mb-2">Sandboxed Execution</h4>
                <p className="text-sm text-muted-foreground">
                  Projects run in isolated environments to prevent malicious code from affecting your system.
                </p>
              </div>
              <div className="p-4 border rounded-lg">
                <h4 className="font-semibold mb-2">Security Scanning</h4>
                <p className="text-sm text-muted-foreground">
                  Before running, GLive scans the project for known vulnerabilities and suspicious patterns.
                </p>
              </div>
              <div className="p-4 border rounded-lg">
                <h4 className="font-semibold mb-2">URL Validation</h4>
                <p className="text-sm text-muted-foreground">
                  All URLs are validated to prevent SSRF attacks and ensure they point to legitimate repositories.
                </p>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Supported Projects Tab */}
        <TabsContent value="projects" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Cpu className="h-5 w-5" />
                Supported Project Types
              </CardTitle>
              <CardDescription>GLive automatically detects and handles these project types</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="grid md:grid-cols-2 gap-4">
                <div className="p-4 border rounded-lg">
                  <h4 className="font-semibold mb-2">Node.js</h4>
                  <p className="text-sm text-muted-foreground mb-2">Detected by: package.json</p>
                  <div className="flex gap-1 flex-wrap">
                    <Badge variant="secondary">npm</Badge>
                    <Badge variant="secondary">yarn</Badge>
                    <Badge variant="secondary">pnpm</Badge>
                  </div>
                </div>

                <div className="p-4 border rounded-lg">
                  <h4 className="font-semibold mb-2">Python</h4>
                  <p className="text-sm text-muted-foreground mb-2">Detected by: requirements.txt, setup.py, pyproject.toml</p>
                  <div className="flex gap-1 flex-wrap">
                    <Badge variant="secondary">pip</Badge>
                    <Badge variant="secondary">Django</Badge>
                    <Badge variant="secondary">Flask</Badge>
                    <Badge variant="secondary">FastAPI</Badge>
                  </div>
                </div>

                <div className="p-4 border rounded-lg">
                  <h4 className="font-semibold mb-2">Go</h4>
                  <p className="text-sm text-muted-foreground mb-2">Detected by: go.mod</p>
                  <div className="flex gap-1 flex-wrap">
                    <Badge variant="secondary">go modules</Badge>
                    <Badge variant="secondary">go run</Badge>
                  </div>
                </div>

                <div className="p-4 border rounded-lg">
                  <h4 className="font-semibold mb-2">Rust</h4>
                  <p className="text-sm text-muted-foreground mb-2">Detected by: Cargo.toml</p>
                  <div className="flex gap-1 flex-wrap">
                    <Badge variant="secondary">cargo</Badge>
                  </div>
                </div>

                <div className="p-4 border rounded-lg">
                  <h4 className="font-semibold mb-2">Java</h4>
                  <p className="text-sm text-muted-foreground mb-2">Detected by: pom.xml, build.gradle</p>
                  <div className="flex gap-1 flex-wrap">
                    <Badge variant="secondary">Maven</Badge>
                    <Badge variant="secondary">Gradle</Badge>
                    <Badge variant="secondary">Spring Boot</Badge>
                  </div>
                </div>

                <div className="p-4 border rounded-lg">
                  <h4 className="font-semibold mb-2">Docker</h4>
                  <p className="text-sm text-muted-foreground mb-2">Detected by: Dockerfile, docker-compose.yml</p>
                  <div className="flex gap-1 flex-wrap">
                    <Badge variant="secondary">docker build</Badge>
                    <Badge variant="secondary">docker-compose</Badge>
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Config Tab */}
        <TabsContent value="config" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <FileJson className="h-5 w-5" />
                Configuration File
              </CardTitle>
              <CardDescription>GLive stores configuration in ~/.glive.json</CardDescription>
            </CardHeader>
            <CardContent>
              <pre className="bg-muted p-4 rounded-lg text-sm overflow-x-auto">
                <code>{`{
  "api_key": "your-ai-provider-api-key",
  "api_provider": "deepseek",
  "workspace_dir": "~/glive-workspace",
  "default_mode": "auto"
}`}</code>
              </pre>
              <div className="mt-4 space-y-2">
                <p className="text-sm"><strong>api_key:</strong> Your AI provider API key (for AI features)</p>
                <p className="text-sm"><strong>api_provider:</strong> AI provider - deepseek, openai, or claude</p>
                <p className="text-sm"><strong>workspace_dir:</strong> Where projects are cloned to</p>
                <p className="text-sm"><strong>default_mode:</strong> Default execution mode</p>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Settings className="h-5 w-5" />
                Environment Variables
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="space-y-3">
                <div className="p-3 bg-muted rounded-lg">
                  <code className="text-sm font-mono">GLIVE_API_KEY</code>
                  <p className="text-sm text-muted-foreground mt-1">AI provider API key (overrides config file)</p>
                </div>
                <div className="p-3 bg-muted rounded-lg">
                  <code className="text-sm font-mono">GLIVE_API_PROVIDER</code>
                  <p className="text-sm text-muted-foreground mt-1">AI provider name: deepseek, openai, claude</p>
                </div>
                <div className="p-3 bg-muted rounded-lg">
                  <code className="text-sm font-mono">GLIVE_WORKSPACE_DIR</code>
                  <p className="text-sm text-muted-foreground mt-1">Custom workspace directory</p>
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Troubleshooting Tab */}
        <TabsContent value="troubleshooting" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <AlertTriangle className="h-5 w-5 text-yellow-500" />
                Common Issues
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="p-4 border rounded-lg">
                <h4 className="font-semibold mb-2">Port Already in Use</h4>
                <p className="text-sm text-muted-foreground mb-2">
                  GLive automatically tries different ports when this happens. If it keeps failing:
                </p>
                <ul className="list-disc list-inside text-sm text-muted-foreground">
                  <li>Kill the process using the port: <code>lsof -i :3000</code> then <code>kill PID</code></li>
                  <li>Or use: <code>glive stop</code> to stop all GLive-started processes</li>
                </ul>
              </div>

              <div className="p-4 border rounded-lg">
                <h4 className="font-semibold mb-2">Dependencies Failed to Install</h4>
                <p className="text-sm text-muted-foreground mb-2">
                  This usually means missing system dependencies:
                </p>
                <ul className="list-disc list-inside text-sm text-muted-foreground">
                  <li>For Node.js: Make sure Node.js 18+ is installed</li>
                  <li>For Python: Ensure pip is up to date: <code>pip install --upgrade pip</code></li>
                  <li>Try running with <code>--mode manual</code> to see exact commands</li>
                </ul>
              </div>

              <div className="p-4 border rounded-lg">
                <h4 className="font-semibold mb-2">Project Type Not Detected</h4>
                <p className="text-sm text-muted-foreground mb-2">
                  GLive looks for specific files to detect project types. If detection fails:
                </p>
                <ul className="list-disc list-inside text-sm text-muted-foreground">
                  <li>Check if the projects configuration file exists (package.json, go.mod, etc.)</li>
                  <li>Navigate to the project folder and run: <code>glive run</code></li>
                </ul>
              </div>

              <div className="p-4 border rounded-lg">
                <h4 className="font-semibold mb-2">glive: command not found</h4>
                <p className="text-sm text-muted-foreground mb-2">
                  GLive is not in your PATH:
                </p>
                <ul className="list-disc list-inside text-sm text-muted-foreground">
                  <li>Linux/macOS: Add <code>/usr/local/bin</code> to your PATH</li>
                  <li>Windows: Run the installer again or manually add the install folder to PATH</li>
                  <li>Go install: Add <code>$GOPATH/bin</code> to your PATH</li>
                </ul>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <HelpCircle className="h-5 w-5" />
                FAQ
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div>
                <h4 className="font-semibold">Is GLive safe to use?</h4>
                <p className="text-sm text-muted-foreground">
                  Yes! GLive includes security scanning and optional sandboxing. However, always review
                  projects from untrusted sources before running them.
                </p>
              </div>
              <div>
                <h4 className="font-semibold">Does GLive work offline?</h4>
                <p className="text-sm text-muted-foreground">
                  Basic functionality works offline, but cloning from GitHub and AI features require internet.
                </p>
              </div>
              <div>
                <h4 className="font-semibold">Can I use GLive with private repositories?</h4>
                <p className="text-sm text-muted-foreground">
                  Yes! If you have git configured with SSH keys or credentials, GLive can clone private repos.
                </p>
              </div>
              <div>
                <h4 className="font-semibold">Where are projects stored?</h4>
                <p className="text-sm text-muted-foreground">
                  By default in ~/glive-workspace. You can change this in the config file or with the
                  GLIVE_WORKSPACE_DIR environment variable.
                </p>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Need More Help?</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              <div className="flex items-center gap-3 p-3 bg-muted rounded-lg">
                <CheckCircle2 className="h-5 w-5 text-green-500" />
                <div>
                  <p className="font-medium">GitHub Issues</p>
                  <a href="https://github.com/yasinhessnawi1/glive/issues" className="text-sm text-primary hover:underline">
                    Report bugs or request features
                  </a>
                </div>
              </div>
              <div className="flex items-center gap-3 p-3 bg-muted rounded-lg">
                <CheckCircle2 className="h-5 w-5 text-green-500" />
                <div>
                  <p className="font-medium">GitHub Discussions</p>
                  <a href="https://github.com/yasinhessnawi1/glive/discussions" className="text-sm text-primary hover:underline">
                    Ask questions and share ideas
                  </a>
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  )
}
