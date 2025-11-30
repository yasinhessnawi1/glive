"use client";

import { useState, useEffect } from "react";
import Link from "next/link";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  Download,
  Terminal,
  Apple,
  Monitor,
  CheckCircle2,
  Copy,
  ExternalLink,
  Github,
  Package,
  Cpu,
  ArrowRight,
  Rocket
} from "lucide-react";
import { cn } from "@/lib/utils";

type Platform = "windows" | "macos" | "linux";

export default function DownloadPage() {
  const [detectedPlatform, setDetectedPlatform] = useState<Platform>("linux");
  const [copiedCommand, setCopiedCommand] = useState<string | null>(null);

  // Detect user's platform
  useEffect(() => {
    const userAgent = navigator.userAgent.toLowerCase();
    if (userAgent.includes("win")) {
      setDetectedPlatform("windows");
    } else if (userAgent.includes("mac")) {
      setDetectedPlatform("macos");
    } else {
      setDetectedPlatform("linux");
    }
  }, []);

  const copyToClipboard = async (text: string, id: string) => {
    await navigator.clipboard.writeText(text);
    setCopiedCommand(id);
    setTimeout(() => setCopiedCommand(null), 2000);
  };

  const CopyButton = ({ text, id }: { text: string; id: string }) => (
    <Button
      variant="ghost"
      size="sm"
      onClick={() => copyToClipboard(text, id)}
      className="absolute right-2 top-2 h-8 w-8 p-0"
    >
      {copiedCommand === id ? (
        <CheckCircle2 className="h-4 w-4 text-green-500" />
      ) : (
        <Copy className="h-4 w-4" />
      )}
    </Button>
  );

  const installCommands = {
    windows: {
      quick: `irm https://raw.githubusercontent.com/yasinhessnawi1/glive/main/install.ps1 | iex`,
      manual: `# Download from GitHub Releases
# 1. Go to https://github.com/yasinhessnawi1/glive/releases
# 2. Download glive_windows_amd64.zip
# 3. Extract to a folder (e.g., C:\\Program Files\\glive)
# 4. Add the folder to your PATH`,
      go: `go install github.com/yasinhessnawi1/glive/cmd/glive@latest`,
    },
    macos: {
      quick: `curl -sSL https://raw.githubusercontent.com/yasinhessnawi1/glive/main/install.sh | bash`,
      homebrew: `# Coming soon!
brew install glive`,
      manual: `# Download from GitHub Releases
curl -sL https://github.com/yasinhessnawi1/glive/releases/latest/download/glive_darwin_arm64.tar.gz | tar xz
sudo mv glive /usr/local/bin/`,
      go: `go install github.com/yasinhessnawi1/glive/cmd/glive@latest`,
    },
    linux: {
      quick: `curl -sSL https://raw.githubusercontent.com/yasinhessnawi1/glive/main/install.sh | bash`,
      manual: `# Download from GitHub Releases
curl -sL https://github.com/yasinhessnawi1/glive/releases/latest/download/glive_linux_amd64.tar.gz | tar xz
sudo mv glive /usr/local/bin/`,
      go: `go install github.com/yasinhessnawi1/glive/cmd/glive@latest`,
    },
  };

  const platformIcons = {
    windows: Monitor,
    macos: Apple,
    linux: Terminal,
  };

  const platformNames = {
    windows: "Windows",
    macos: "macOS",
    linux: "Linux",
  };

  return (
    <div className="container mx-auto px-4 py-8 max-w-5xl">
      {/* Header */}
      <div className="text-center mb-12">
        <div className="flex items-center justify-center gap-2 mb-4">
          <Download className="h-10 w-10 text-primary" />
        </div>
        <h1 className="text-4xl font-bold mb-4">Download GLive</h1>
        <p className="text-xl text-muted-foreground mb-6">
          Get started with GLive CLI in seconds
        </p>
        <div className="flex justify-center gap-2 flex-wrap">
          <Badge variant="secondary">
            <Cpu className="mr-1 h-3 w-3" />
            Cross-Platform
          </Badge>
          <Badge variant="secondary">
            <Package className="mr-1 h-3 w-3" />
            Single Binary
          </Badge>
          <Badge variant="secondary">
            <Rocket className="mr-1 h-3 w-3" />
            No Dependencies
          </Badge>
        </div>
      </div>

      {/* Detected Platform Banner */}
      <Card className="mb-8 border-primary/50 bg-primary/5">
        <CardContent className="pt-6">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              {(() => {
                const Icon = platformIcons[detectedPlatform];
                return <Icon className="h-8 w-8 text-primary" />;
              })()}
              <div>
                <p className="text-sm text-muted-foreground">Detected Platform</p>
                <p className="font-semibold text-lg">{platformNames[detectedPlatform]}</p>
              </div>
            </div>
            <Badge variant="outline" className="text-primary border-primary">
              Recommended
            </Badge>
          </div>
        </CardContent>
      </Card>

      {/* Platform Tabs */}
      <Tabs defaultValue={detectedPlatform} className="w-full">
        <TabsList className="grid w-full grid-cols-3 mb-8">
          <TabsTrigger value="windows" className="flex items-center gap-2">
            <Monitor className="h-4 w-4" />
            Windows
          </TabsTrigger>
          <TabsTrigger value="macos" className="flex items-center gap-2">
            <Apple className="h-4 w-4" />
            macOS
          </TabsTrigger>
          <TabsTrigger value="linux" className="flex items-center gap-2">
            <Terminal className="h-4 w-4" />
            Linux
          </TabsTrigger>
        </TabsList>

        {/* Windows */}
        <TabsContent value="windows" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Rocket className="h-5 w-5 text-green-500" />
                Quick Install (Recommended)
              </CardTitle>
              <CardDescription>
                One-line PowerShell command - installs GLive and adds it to your PATH
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="relative">
                <pre className="bg-muted p-4 rounded-lg overflow-x-auto text-sm">
                  <code>{installCommands.windows.quick}</code>
                </pre>
                <CopyButton text={installCommands.windows.quick} id="win-quick" />
              </div>
              <p className="text-sm text-muted-foreground mt-3">
                Run this in PowerShell as Administrator
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Using Go</CardTitle>
              <CardDescription>
                If you have Go installed (1.21+)
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="relative">
                <pre className="bg-muted p-4 rounded-lg overflow-x-auto text-sm">
                  <code>{installCommands.windows.go}</code>
                </pre>
                <CopyButton text={installCommands.windows.go} id="win-go" />
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Manual Installation</CardTitle>
              <CardDescription>
                Download the binary manually from GitHub Releases
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="relative">
                <pre className="bg-muted p-4 rounded-lg overflow-x-auto text-sm whitespace-pre-wrap">
                  <code>{installCommands.windows.manual}</code>
                </pre>
              </div>
              <Link href="https://github.com/yasinhessnawi1/glive/releases" target="_blank">
                <Button variant="outline" className="mt-4">
                  <Github className="mr-2 h-4 w-4" />
                  View GitHub Releases
                  <ExternalLink className="ml-2 h-3 w-3" />
                </Button>
              </Link>
            </CardContent>
          </Card>
        </TabsContent>

        {/* macOS */}
        <TabsContent value="macos" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Rocket className="h-5 w-5 text-green-500" />
                Quick Install (Recommended)
              </CardTitle>
              <CardDescription>
                One-line curl command - installs GLive to /usr/local/bin
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="relative">
                <pre className="bg-muted p-4 rounded-lg overflow-x-auto text-sm">
                  <code>{installCommands.macos.quick}</code>
                </pre>
                <CopyButton text={installCommands.macos.quick} id="mac-quick" />
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Using Go</CardTitle>
              <CardDescription>
                If you have Go installed (1.21+)
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="relative">
                <pre className="bg-muted p-4 rounded-lg overflow-x-auto text-sm">
                  <code>{installCommands.macos.go}</code>
                </pre>
                <CopyButton text={installCommands.macos.go} id="mac-go" />
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Manual Installation</CardTitle>
              <CardDescription>
                Download and install manually
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="relative">
                <pre className="bg-muted p-4 rounded-lg overflow-x-auto text-sm whitespace-pre-wrap">
                  <code>{installCommands.macos.manual}</code>
                </pre>
                <CopyButton text={installCommands.macos.manual} id="mac-manual" />
              </div>
              <Link href="https://github.com/yasinhessnawi1/glive/releases" target="_blank">
                <Button variant="outline" className="mt-4">
                  <Github className="mr-2 h-4 w-4" />
                  View GitHub Releases
                  <ExternalLink className="ml-2 h-3 w-3" />
                </Button>
              </Link>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Linux */}
        <TabsContent value="linux" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Rocket className="h-5 w-5 text-green-500" />
                Quick Install (Recommended)
              </CardTitle>
              <CardDescription>
                One-line curl command - installs GLive to /usr/local/bin
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="relative">
                <pre className="bg-muted p-4 rounded-lg overflow-x-auto text-sm">
                  <code>{installCommands.linux.quick}</code>
                </pre>
                <CopyButton text={installCommands.linux.quick} id="linux-quick" />
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Using Go</CardTitle>
              <CardDescription>
                If you have Go installed (1.21+)
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="relative">
                <pre className="bg-muted p-4 rounded-lg overflow-x-auto text-sm">
                  <code>{installCommands.linux.go}</code>
                </pre>
                <CopyButton text={installCommands.linux.go} id="linux-go" />
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Manual Installation</CardTitle>
              <CardDescription>
                Download and install manually
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="relative">
                <pre className="bg-muted p-4 rounded-lg overflow-x-auto text-sm whitespace-pre-wrap">
                  <code>{installCommands.linux.manual}</code>
                </pre>
                <CopyButton text={installCommands.linux.manual} id="linux-manual" />
              </div>
              <Link href="https://github.com/yasinhessnawi1/glive/releases" target="_blank">
                <Button variant="outline" className="mt-4">
                  <Github className="mr-2 h-4 w-4" />
                  View GitHub Releases
                  <ExternalLink className="ml-2 h-3 w-3" />
                </Button>
              </Link>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>

      {/* Verify Installation */}
      <Card className="mt-8">
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <CheckCircle2 className="h-5 w-5 text-green-500" />
            Verify Installation
          </CardTitle>
          <CardDescription>
            After installation, verify GLive is working correctly
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div>
            <p className="text-sm font-medium mb-2">1. Check the version:</p>
            <div className="relative">
              <pre className="bg-muted p-4 rounded-lg text-sm">
                <code>glive --version</code>
              </pre>
              <CopyButton text="glive --version" id="verify-version" />
            </div>
          </div>
          <div>
            <p className="text-sm font-medium mb-2">2. Run your first project:</p>
            <div className="relative">
              <pre className="bg-muted p-4 rounded-lg text-sm">
                <code>glive https://github.com/facebook/react</code>
              </pre>
              <CopyButton text="glive https://github.com/facebook/react" id="verify-run" />
            </div>
          </div>
          <div>
            <p className="text-sm font-medium mb-2">3. Or use the interactive TUI:</p>
            <div className="relative">
              <pre className="bg-muted p-4 rounded-lg text-sm">
                <code>glive</code>
              </pre>
              <CopyButton text="glive" id="verify-tui" />
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Quick Start Guide */}
      <Card className="mt-8">
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Rocket className="h-5 w-5" />
            Quick Start Guide
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid gap-4 md:grid-cols-3">
            <div className="space-y-2">
              <div className="flex items-center gap-2">
                <Badge variant="outline">1</Badge>
                <span className="font-medium">Clone & Run</span>
              </div>
              <p className="text-sm text-muted-foreground">
                Just pass any GitHub URL and GLive handles the rest
              </p>
              <pre className="bg-muted p-2 rounded text-xs">
                glive user/repo
              </pre>
            </div>
            <div className="space-y-2">
              <div className="flex items-center gap-2">
                <Badge variant="outline">2</Badge>
                <span className="font-medium">Run Existing Project</span>
              </div>
              <p className="text-sm text-muted-foreground">
                Navigate to any project and run it
              </p>
              <pre className="bg-muted p-2 rounded text-xs">
                glive run ./my-project
              </pre>
            </div>
            <div className="space-y-2">
              <div className="flex items-center gap-2">
                <Badge variant="outline">3</Badge>
                <span className="font-medium">Interactive Mode</span>
              </div>
              <p className="text-sm text-muted-foreground">
                Use the TUI for a visual experience
              </p>
              <pre className="bg-muted p-2 rounded text-xs">
                glive
              </pre>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* CTA */}
      <div className="mt-12 text-center space-y-4">
        <p className="text-muted-foreground">
          Need help? Check out the documentation or use the web dashboard.
        </p>
        <div className="flex justify-center gap-4">
          <Link href="/docs">
            <Button variant="outline">
              View Documentation
              <ArrowRight className="ml-2 h-4 w-4" />
            </Button>
          </Link>
          <Link href="/dashboard">
            <Button>
              Open Dashboard
              <ArrowRight className="ml-2 h-4 w-4" />
            </Button>
          </Link>
        </div>
      </div>
    </div>
  );
}
