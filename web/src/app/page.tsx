import Link from "next/link";
import Image from "next/image";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Zap, Shield, Palette, Bot, Workflow, CreditCard, Code, Github, Star, Users, Rocket, Brain, GitBranch } from "lucide-react";

export default function Home() {
  const features = [
    {
      icon: Rocket,
      title: "Instant Setup",
      description: "Clone and run projects with a single command. No manual configuration needed.",
      techs: ["One Command", "Auto-Clone", "Instant Run"]
    },
    {
      icon: Brain,
      title: "AI-Powered",
      description: "Smart analysis and automatic troubleshooting. AI recovery handles errors intelligently.",
      techs: ["AI Analysis", "Auto-Fix", "Smart Recovery"]
    },
    {
      icon: Shield,
      title: "Safe & Secure",
      description: "Safety checks and sandboxing built-in. Your system stays protected.",
      techs: ["Sandboxed", "Security Checks", "Isolated"]
    }
  ];

  return (
    <div className="min-h-screen bg-gradient-to-br from-background via-background to-muted/20">
      {/* Hero Section */}
      <section className="container mx-auto space-y-6 pb-8 pt-6 md:pb-12 md:pt-10 lg:py-32">
        <div className="mx-auto flex max-w-[64rem] flex-col items-center space-y-4 text-center">
          <Badge variant="outline" className="text-sm">
            <Rocket className="mr-2 h-3 w-3" />
            GitHub to Live
          </Badge>

          <h1 className="font-heading text-3xl sm:text-5xl md:text-6xl lg:text-7xl font-bold tracking-tight">
            <span className="bg-gradient-to-r from-blue-600 to-cyan-600 bg-clip-text text-transparent">
              GLive
            </span>
          </h1>
          <h2 className="text-2xl sm:text-3xl md:text-4xl font-bold tracking-tight text-foreground/80">
            Run any project instantly
          </h2>

          <p className="max-w-[42rem] leading-normal text-muted-foreground sm:text-xl sm:leading-8">
            AI-powered setup • Automatic dependencies • One command.
            Stop struggling with setup instructions. Just GLive it.
          </p>

          <div className="flex flex-col sm:flex-row gap-4 pt-4">
            <Link href="/dashboard/projects/new">
              <Button size="lg" className="text-lg px-8">
                <Zap className="mr-2 h-5 w-5" />
                Get Started
              </Button>
            </Link>
            <Link href="/dashboard">
              <Button variant="outline" size="lg" className="text-lg px-8">
                <GitBranch className="mr-2 h-5 w-5" />
                Go to Dashboard
              </Button>
            </Link>
          </div>

          <div className="flex items-center space-x-4 pt-4 text-sm text-muted-foreground">
            <div className="flex items-center space-x-1">
              <Star className="h-4 w-4 fill-current" />
              <span>Open Source</span>
            </div>
            <div className="flex items-center space-x-1">
              <Users className="h-4 w-4" />
              <span>MIT License</span>
            </div>
          </div>
        </div>
      </section>

      {/* Features Section */}
      <section className="container mx-auto space-y-6 py-8 md:py-12 lg:py-24">
        <div className="mx-auto flex max-w-[58rem] flex-col items-center space-y-4 text-center">
          <h2 className="font-heading text-3xl leading-[1.1] sm:text-3xl md:text-6xl font-bold">
            Why GLive?
          </h2>
          <p className="max-w-[85%] leading-normal text-muted-foreground sm:text-lg sm:leading-7">
            The fastest way to go from GitHub repo to running application.
          </p>
        </div>

        <div className="mx-auto grid justify-center gap-4 sm:grid-cols-2 md:max-w-[64rem] md:grid-cols-3">
          {features.map((feature, index) => (
            <Card key={index} className="relative overflow-hidden border-0 bg-background/60 backdrop-blur supports-[backdrop-filter]:bg-background/60">
              <CardHeader>
                <div className="flex items-center space-x-2">
                  <div className="rounded-lg bg-primary/10 p-2">
                    <feature.icon className="h-6 w-6 text-primary" />
                  </div>
                </div>
                <CardTitle className="text-xl">{feature.title}</CardTitle>
                <CardDescription className="text-sm text-muted-foreground">
                  {feature.description}
                </CardDescription>
              </CardHeader>
              <CardContent>
                <div className="flex flex-wrap gap-1">
                  {feature.techs.map((tech, techIndex) => (
                    <Badge key={techIndex} variant="secondary" className="text-xs">
                      {tech}
                    </Badge>
                  ))}
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      </section>

      {/* Run It Your Way Section */}
      <section className="container mx-auto space-y-6 py-8 md:py-12 lg:py-24">
        <div className="mx-auto flex max-w-[58rem] flex-col items-center space-y-4 text-center">
          <h2 className="font-heading text-3xl leading-[1.1] sm:text-3xl md:text-6xl font-bold">
            Run It Your Way
          </h2>
          <p className="max-w-[85%] leading-normal text-muted-foreground sm:text-lg sm:leading-7">
            Choose the mode that fits your workflow. From command line to full dashboard.
          </p>
        </div>

        <div className="mx-auto grid max-w-5xl grid-cols-1 gap-4 md:grid-cols-3">
          <Card className="relative overflow-hidden border-0 bg-background/60 backdrop-blur supports-[backdrop-filter]:bg-background/60">
            <CardHeader>
              <div className="flex items-center space-x-2">
                <div className="rounded-lg bg-primary/10 p-2">
                  <Code className="h-6 w-6 text-primary" />
                </div>
              </div>
              <CardTitle className="text-xl">CLI Mode</CardTitle>
              <CardDescription>
                Fast, scriptable, and powerful.
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="rounded-md bg-muted p-4 font-mono text-xs">
                glive https://github.com/user/repo
              </div>
            </CardContent>
          </Card>

          <Card className="relative overflow-hidden border-0 bg-background/60 backdrop-blur supports-[backdrop-filter]:bg-background/60">
            <CardHeader>
              <div className="flex items-center space-x-2">
                <div className="rounded-lg bg-primary/10 p-2">
                  <Workflow className="h-6 w-6 text-primary" />
                </div>
              </div>
              <CardTitle className="text-xl">TUI Mode</CardTitle>
              <CardDescription>
                Interactive terminal experience.
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="rounded-md bg-muted p-4 font-mono text-xs">
                glive
              </div>
            </CardContent>
          </Card>

          <Card className="relative overflow-hidden border-0 bg-background/60 backdrop-blur supports-[backdrop-filter]:bg-background/60">
            <CardHeader>
              <div className="flex items-center space-x-2">
                <div className="rounded-lg bg-primary/10 p-2">
                  <Palette className="h-6 w-6 text-primary" />
                </div>
              </div>
              <CardTitle className="text-xl">Web App</CardTitle>
              <CardDescription>
                Visual dashboard with history.
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="rounded-md bg-muted p-4 font-mono text-xs">
                <Link href="/dashboard">
                  Get Started Now
                </Link>
              </div>
            </CardContent>
          </Card>
        </div>

        <div className="mx-auto max-w-4xl pt-8">
          <Card className="border-0 bg-background/60 backdrop-blur supports-[backdrop-filter]:bg-background/60">
            <CardHeader>
              <CardTitle className="text-center">Installation</CardTitle>
              <CardDescription className="text-center">
                Get started with a single command
              </CardDescription>
            </CardHeader>
            <CardContent className="flex flex-col items-center space-y-4">
              <div className="rounded-lg bg-muted p-4 font-mono text-sm w-full max-w-lg text-center">
                go install github.com/glive/cmd/glive@latest
              </div>
              <Link href="/docs">
                <Button variant="outline">
                  View Full Documentation
                </Button>
              </Link>
            </CardContent>
          </Card>
        </div>
      </section>


    </div>
  );
}
