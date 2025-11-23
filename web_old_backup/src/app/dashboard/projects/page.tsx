'use client';

import { useEffect, useState } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Input } from '@/components/ui/input';
import { useGliveAPI } from '@/hooks/use-glive-api';
import type { Project } from '@/types/glive';
import Link from 'next/link';
import { Plus, Search, ExternalLink, ArrowRight } from 'lucide-react';

export default function ProjectsPage() {
  const { listProjects, loading } = useGliveAPI();
  const [projects, setProjects] = useState<Project[]>([]);
  const [searchQuery, setSearchQuery] = useState('');

  useEffect(() => {
    const loadProjects = async () => {
      const data = await listProjects();
      setProjects(data);
    };
    loadProjects();
    const interval = setInterval(loadProjects, 10000);
    return () => clearInterval(interval);
  }, [listProjects]);

  const filteredProjects = projects.filter(project =>
    project.github_url.toLowerCase().includes(searchQuery.toLowerCase()) ||
    project.name?.toLowerCase().includes(searchQuery.toLowerCase()) ||
    project.id.toLowerCase().includes(searchQuery.toLowerCase())
  );

  const getStatusColor = (status: Project['status']) => {
    switch (status) {
      case 'ready':
        return 'bg-green-100 text-green-800 dark:bg-green-900/20 dark:text-green-300';
      case 'running':
      case 'cloning':
      case 'analyzing':
      case 'installing':
        return 'bg-blue-100 text-blue-800 dark:bg-blue-900/20 dark:text-blue-300';
      case 'failed':
        return 'bg-red-100 text-red-800 dark:bg-red-900/20 dark:text-red-300';
      case 'stopped':
        return 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-300';
      default:
        return 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/20 dark:text-yellow-300';
    }
  };

  return (
    <div className="min-h-screen bg-slate-50 dark:bg-slate-900">
      <header className="bg-white dark:bg-slate-800 shadow-sm border-b border-slate-200 dark:border-slate-700">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between items-center h-16">
            <h1 className="text-xl font-semibold text-slate-900 dark:text-slate-100">
              All Projects
            </h1>
            <Link href="/dashboard/projects/new">
              <Button>
                <Plus className="h-4 w-4 mr-2" />
                New Project
              </Button>
            </Link>
          </div>
        </div>
      </header>

      <main className="max-w-7xl mx-auto py-6 sm:px-6 lg:px-8">
        <div className="px-4 py-6 sm:px-0">
          {/* Search */}
          <div className="mb-6">
            <div className="relative">
              <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <Input
                placeholder="Search projects by name, URL, or ID..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="pl-10"
              />
            </div>
          </div>

          {loading && projects.length === 0 ? (
            <Card>
              <CardContent className="py-12 text-center">
                <p className="text-muted-foreground">Loading projects...</p>
              </CardContent>
            </Card>
          ) : filteredProjects.length === 0 ? (
            <Card>
              <CardContent className="py-12 text-center">
                <p className="text-muted-foreground mb-4">
                  {searchQuery ? 'No projects found matching your search.' : 'No projects yet.'}
                </p>
                <Link href="/dashboard/projects/new">
                  <Button>
                    <Plus className="h-4 w-4 mr-2" />
                    Create Your First Project
                  </Button>
                </Link>
              </CardContent>
            </Card>
          ) : (
            <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
              {filteredProjects.map((project) => (
                <Card key={project.id} className="hover:shadow-lg transition-all duration-200 group border-slate-200 dark:border-slate-700">
                  <CardHeader className="pb-3">
                    <div className="flex items-start justify-between gap-4">
                      <div className="flex-1 min-w-0 space-y-1">
                        <CardTitle className="truncate text-lg font-semibold text-slate-900 dark:text-slate-100">
                          {project.name || project.github_url.split('/').pop()}
                        </CardTitle>
                        <CardDescription className="truncate text-sm text-slate-500 dark:text-slate-400">
                          {project.github_url}
                        </CardDescription>
                      </div>
                      <Badge className={`${getStatusColor(project.status)} shrink-0 capitalize`} variant="outline">
                        {project.status}
                      </Badge>
                    </div>
                  </CardHeader>
                  <CardContent className="space-y-4">
                    <div className="flex items-center gap-2 text-sm text-muted-foreground bg-slate-50 dark:bg-slate-800/50 p-2 rounded-md">
                      <span className="font-medium">Type:</span>
                      <Badge variant="secondary" className="text-xs uppercase tracking-wider">
                        {project.type || 'unknown'}
                      </Badge>
                    </div>
                    <div className="flex gap-3 pt-2">
                      <Link href={`/dashboard/projects/${project.id}`} className="flex-1">
                        <Button className="w-full bg-slate-900 hover:bg-slate-800 text-white dark:bg-slate-100 dark:hover:bg-slate-200 dark:text-slate-900" size="sm">
                          View Details
                          <ArrowRight className="h-3 w-3 ml-2" />
                        </Button>
                      </Link>
                      <a
                        href={project.github_url}
                        target="_blank"
                        rel="noopener noreferrer"
                      >
                        <Button variant="outline" size="sm" className="px-3" title="View on GitHub">
                          <ExternalLink className="h-4 w-4" />
                        </Button>
                      </a>
                    </div>
                  </CardContent>
                </Card>
              ))}
            </div>
          )}
        </div>
      </main>
    </div>
  );
}
