'use client';

import { StatsOverview } from '@/components/glive/dashboard/stats-overview';
import { RecentProjects } from '@/components/glive/dashboard/recent-projects';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import Link from 'next/link';
import { Plus, Settings, FileText, Activity } from 'lucide-react';

export default function DashboardPage() {
  return (
    <div className="min-h-screen bg-slate-50 dark:bg-slate-900">
      <header className="bg-white dark:bg-slate-800 shadow-sm border-b border-slate-200 dark:border-slate-700">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between items-center h-16">
            <div className="flex items-center gap-4">
              <Link href="/">
                <h1 className="text-xl font-semibold text-slate-900 dark:text-slate-100">
                  GLive Dashboard
                </h1>
              </Link>
            </div>
            <nav className="flex items-center gap-4">
              <Link href="/dashboard/projects/new">
                <Button size="sm">
                  <Plus className="h-4 w-4 mr-2" />
                  New Project
                </Button>
              </Link>
              <Link href="/dashboard/settings">
                <Button variant="ghost" size="sm">
                  <Settings className="h-4 w-4" />
                </Button>
              </Link>
            </nav>
          </div>
        </div>
      </header>

      <main className="max-w-7xl mx-auto py-6 sm:px-6 lg:px-8">
        <div className="px-4 py-6 sm:px-0">
          <div className="mb-8">
            <h2 className="text-3xl font-bold text-slate-900 dark:text-slate-100 mb-2">
              Welcome to GLive
            </h2>
            <p className="text-slate-600 dark:text-slate-400">
              Manage your GitHub projects, monitor executions, and track AI recovery attempts.
            </p>
          </div>

          {/* Stats Overview */}
          <div className="mb-8">
            <StatsOverview />
          </div>

          {/* Quick Actions */}
          <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-4 mb-8">
            <Link href="/dashboard/projects/new">
              <Card className="h-full hover:shadow-lg transition-shadow cursor-pointer group">
                <CardHeader>
                  <div className="flex items-center justify-between">
                    <Plus className="h-8 w-8 text-blue-600" />
                  </div>
                  <CardTitle>Create Project</CardTitle>
                  <CardDescription>
                    Start a new project from a GitHub repository
                  </CardDescription>
                </CardHeader>
              </Card>
            </Link>

            <Link href="/dashboard/projects">
              <Card className="h-full hover:shadow-lg transition-shadow cursor-pointer group">
                <CardHeader>
                  <div className="flex items-center justify-between">
                    <Activity className="h-8 w-8 text-green-600" />
                  </div>
                  <CardTitle>All Projects</CardTitle>
                  <CardDescription>
                    View and manage all your projects
                  </CardDescription>
                </CardHeader>
              </Card>
            </Link>

            <Link href="/dashboard/logs">
              <Card className="h-full hover:shadow-lg transition-shadow cursor-pointer group">
                <CardHeader>
                  <div className="flex items-center justify-between">
                    <FileText className="h-8 w-8 text-purple-600" />
                  </div>
                  <CardTitle>Logs</CardTitle>
                  <CardDescription>
                    View execution logs and system events
                  </CardDescription>
                </CardHeader>
              </Card>
            </Link>

            <Link href="/dashboard/settings">
              <Card className="h-full hover:shadow-lg transition-shadow cursor-pointer group">
                <CardHeader>
                  <div className="flex items-center justify-between">
                    <Settings className="h-8 w-8 text-orange-600" />
                  </div>
                  <CardTitle>Settings</CardTitle>
                  <CardDescription>
                    Configure API, sandbox, and AI settings
                  </CardDescription>
                </CardHeader>
              </Card>
            </Link>
          </div>

          {/* Recent Projects */}
          <div className="mb-8">
            <RecentProjects />
          </div>
        </div>
      </main>
    </div>
  );
}

