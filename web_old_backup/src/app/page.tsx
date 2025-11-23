'use client'

import { useEffect } from 'react'
import { useRouter } from 'next/navigation'
import Link from 'next/link'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { 
  Rocket, 
  Brain, 
  Shield, 
  ArrowRight,
  Zap,
  GitBranch
} from 'lucide-react'

export default function Home() {
  const router = useRouter()

  return (
    <main className="min-h-screen bg-gradient-to-br from-slate-50 to-blue-50 dark:from-slate-900 dark:to-slate-800">
      <header className="bg-white/80 dark:bg-slate-800/80 backdrop-blur-sm shadow-sm border-b border-slate-200 dark:border-slate-700">
        <div className="container mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between items-center h-16">
            <div className="flex items-center">
              <h1 className="text-xl font-semibold text-slate-900 dark:text-slate-100">
                GLive
              </h1>
            </div>
            <nav className="flex items-center gap-4">
              <Link href="/dashboard">
                <Button variant="ghost">Dashboard</Button>
              </Link>
              <Link href="/dashboard/projects/new">
                <Button>Get Started</Button>
              </Link>
            </nav>
          </div>
        </div>
      </header>

      <div className="container mx-auto px-4 py-16 max-w-6xl">
        <div className="text-center mb-16">
          <h1 className="text-5xl md:text-6xl font-bold mb-6 bg-gradient-to-r from-blue-600 to-cyan-600 bg-clip-text text-transparent">
            GLive
          </h1>
          <p className="text-xl md:text-2xl text-gray-600 dark:text-gray-300 mb-4">
            GitHub to Live - Run any project instantly
          </p>
          <p className="text-lg text-gray-500 dark:text-gray-400 mb-8">
            AI-powered setup • Automatic dependencies • One command
          </p>
          <div className="flex gap-4 justify-center">
            <Link href="/dashboard">
              <Button size="lg" className="gap-2">
                Go to Dashboard
                <ArrowRight className="h-4 w-4" />
              </Button>
            </Link>
            <Link href="/dashboard/projects/new">
              <Button size="lg" variant="outline" className="gap-2">
                Create Project
                <GitBranch className="h-4 w-4" />
              </Button>
            </Link>
          </div>
        </div>

        <div className="grid md:grid-cols-3 gap-6 mb-16">
          <Card className="hover:shadow-lg transition-shadow">
            <CardHeader>
              <Rocket className="h-10 w-10 text-blue-600 mb-2" />
              <CardTitle>Instant Setup</CardTitle>
              <CardDescription>
                Clone and run projects with a single command. No manual configuration needed.
              </CardDescription>
            </CardHeader>
          </Card>

          <Card className="hover:shadow-lg transition-shadow">
            <CardHeader>
              <Brain className="h-10 w-10 text-purple-600 mb-2" />
              <CardTitle>AI-Powered</CardTitle>
              <CardDescription>
                Smart analysis and automatic troubleshooting. AI recovery handles errors intelligently.
              </CardDescription>
            </CardHeader>
          </Card>

          <Card className="hover:shadow-lg transition-shadow">
            <CardHeader>
              <Shield className="h-10 w-10 text-green-600 mb-2" />
              <CardTitle>Safe & Secure</CardTitle>
              <CardDescription>
                Safety checks and sandboxing built-in. Your system stays protected.
              </CardDescription>
            </CardHeader>
          </Card>
        </div>

        <div className="text-center">
          <h2 className="text-2xl font-semibold mb-4 text-gray-800 dark:text-gray-200">
            Ready to get started?
          </h2>
          <Link href="/dashboard">
            <Button size="lg" variant="outline">
              Open Dashboard
            </Button>
          </Link>
        </div>
      </div>
    </main>
  )
}
