'use client';

import { useEffect, useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { useGliveAPI } from '@/hooks/use-glive-api';
import type { Project } from '@/types/glive';
import { 
  ArrowLeft, 
  ExternalLink, 
  Play, 
  Square, 
  Trash2, 
  RefreshCw,
  Loader2,
  Calendar,
  Folder,
  GitBranch
} from 'lucide-react';
import Link from 'next/link';

export default function ProjectDetailsPage() {
  const params = useParams();
  const router = useRouter();
  const { getProject, startProject, stopProject, deleteProject, cleanupProject, loading } = useGliveAPI();
  const [project, setProject] = useState<Project | null>(null);
  const [actionLoading, setActionLoading] = useState<string | null>(null);

  const projectId = params.id as string;

  useEffect(() => {
    const loadProject = async () => {
      const data = await getProject(projectId);
      if (data) {
        setProject(data);
      }
    };
    loadProject();
    
    // Refresh every 5 seconds
    const interval = setInterval(loadProject, 5000);
    return () => clearInterval(interval);
  }, [projectId, getProject]);

  const handleStart = async () => {
    setActionLoading('start');
    try {
      const success = await startProject(projectId);
      if (success) {
        const updated = await getProject(projectId);
        if (updated) setProject(updated);
      } else {
        alert('Failed to start project. Please check the console for details.');
      }
    } catch (error) {
      console.error('Error starting project:', error);
      alert('Error starting project: ' + (error instanceof Error ? error.message : 'Unknown error'));
    } finally {
      setActionLoading(null);
    }
  };

  const handleStop = async () => {
    setActionLoading('stop');
    try {
      const success = await stopProject(projectId);
      if (success) {
        const updated = await getProject(projectId);
        if (updated) setProject(updated);
      } else {
        alert('Failed to stop project. Please check the console for details.');
      }
    } catch (error) {
      console.error('Error stopping project:', error);
      alert('Error stopping project: ' + (error instanceof Error ? error.message : 'Unknown error'));
    } finally {
      setActionLoading(null);
    }
  };

  const handleDelete = async () => {
    if (!confirm('Are you sure you want to delete this project? This action cannot be undone.')) {
      return;
    }
    setActionLoading('delete');
    try {
      const success = await deleteProject(projectId);
      if (success) {
        router.push('/dashboard');
      } else {
        alert('Failed to delete project. Please check the console for details.');
      }
    } catch (error) {
      console.error('Error deleting project:', error);
      alert('Error deleting project: ' + (error instanceof Error ? error.message : 'Unknown error'));
    } finally {
      setActionLoading(null);
    }
  };

  const handleCleanup = async () => {
    setActionLoading('cleanup');
    try {
      const success = await cleanupProject(projectId);
      if (success) {
        const updated = await getProject(projectId);
        if (updated) setProject(updated);
        alert('Project resources cleaned up successfully.');
      } else {
        alert('Failed to cleanup project. Please check the console for details.');
      }
    } catch (error) {
      console.error('Error cleaning up project:', error);
      alert('Error cleaning up project: ' + (error instanceof Error ? error.message : 'Unknown error'));
    } finally {
      setActionLoading(null);
    }
  };

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

  if (loading && !project) {
    return (
      <div className="min-h-screen bg-slate-50 dark:bg-slate-900 flex items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-blue-600" />
      </div>
    );
  }

  if (!project) {
    return (
      <div className="min-h-screen bg-slate-50 dark:bg-slate-900">
        <div className="max-w-4xl mx-auto py-8 px-4">
          <Card>
            <CardHeader>
              <CardTitle>Project Not Found</CardTitle>
              <CardDescription>
                The project you're looking for doesn't exist or has been deleted.
              </CardDescription>
            </CardHeader>
            <CardContent>
              <Link href="/dashboard">
                <Button>Back to Dashboard</Button>
              </Link>
            </CardContent>
          </Card>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-slate-50 dark:bg-slate-900">
      <header className="bg-white dark:bg-slate-800 shadow-sm border-b border-slate-200 dark:border-slate-700">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex items-center h-16 gap-4">
            <Link href="/dashboard">
              <Button variant="ghost" size="sm">
                <ArrowLeft className="h-4 w-4 mr-2" />
                Back
              </Button>
            </Link>
            <h1 className="text-xl font-semibold text-slate-900 dark:text-slate-100">
              {project.name || project.github_url?.split('/').pop() || 'Project Details'}
            </h1>
            <Badge className={getStatusColor(project.status)} variant="outline">
              {project.status}
            </Badge>
          </div>
        </div>
      </header>

      <main className="max-w-7xl mx-auto py-6 sm:px-6 lg:px-8">
        <div className="px-4 py-6 sm:px-0">
          <div className="grid gap-6 md:grid-cols-2">
            {/* Project Information */}
            <Card>
              <CardHeader>
                <CardTitle>Project Information</CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                <div>
                  <div className="flex items-center gap-2 text-sm text-muted-foreground mb-1">
                    <GitBranch className="h-4 w-4" />
                    Repository
                  </div>
                  <div className="flex items-center gap-2">
                    <a
                      href={project.github_url}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="text-blue-600 hover:underline flex items-center gap-1"
                    >
                      {project.github_url}
                      <ExternalLink className="h-3 w-3" />
                    </a>
                  </div>
                </div>

                <div>
                  <div className="flex items-center gap-2 text-sm text-muted-foreground mb-1">
                    <Folder className="h-4 w-4" />
                    Local Path
                  </div>
                  <p className="text-sm font-mono">{project.local_path || 'N/A'}</p>
                </div>

                <div>
                  <div className="flex items-center gap-2 text-sm text-muted-foreground mb-1">
                    Type
                  </div>
                  <Badge variant="outline">{project.type || 'unknown'}</Badge>
                  {project.detected_types && project.detected_types.length > 0 && (
                    <div className="mt-2 flex flex-wrap gap-2">
                      {project.detected_types.map((type) => (
                        <Badge key={type} variant="secondary" className="text-xs">
                          {type}
                        </Badge>
                      ))}
                    </div>
                  )}
                </div>

                <div>
                  <div className="flex items-center gap-2 text-sm text-muted-foreground mb-1">
                    <Calendar className="h-4 w-4" />
                    Created
                  </div>
                  <p className="text-sm">
                    {new Date(project.created_at).toLocaleString()}
                  </p>
                </div>

                <div>
                  <div className="flex items-center gap-2 text-sm text-muted-foreground mb-1">
                    Last Updated
                  </div>
                  <p className="text-sm">
                    {new Date(project.updated_at).toLocaleString()}
                  </p>
                </div>
              </CardContent>
            </Card>

            {/* Actions */}
            <Card>
              <CardHeader>
                <CardTitle>Actions</CardTitle>
                <CardDescription>
                  Manage your project execution and resources
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-3">
                {project.status === 'ready' && (
                  <Button
                    onClick={handleStart}
                    disabled={actionLoading !== null}
                    className="w-full"
                  >
                    {actionLoading === 'start' ? (
                      <>
                        <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                        Starting...
                      </>
                    ) : (
                      <>
                        <Play className="h-4 w-4 mr-2" />
                        Start Project
                      </>
                    )}
                  </Button>
                )}

                {['running', 'cloning', 'analyzing', 'installing'].includes(project.status) && (
                  <Button
                    onClick={handleStop}
                    disabled={actionLoading !== null}
                    variant="destructive"
                    className="w-full"
                  >
                    {actionLoading === 'stop' ? (
                      <>
                        <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                        Stopping...
                      </>
                    ) : (
                      <>
                        <Square className="h-4 w-4 mr-2" />
                        Stop Project
                      </>
                    )}
                  </Button>
                )}

                <Button
                  onClick={handleCleanup}
                  disabled={actionLoading !== null}
                  variant="outline"
                  className="w-full"
                >
                  {actionLoading === 'cleanup' ? (
                    <>
                      <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                      Cleaning...
                    </>
                  ) : (
                    <>
                      <RefreshCw className="h-4 w-4 mr-2" />
                      Cleanup Resources
                    </>
                  )}
                </Button>

                <Link href={`/dashboard/projects/${projectId}/execution`}>
                  <Button variant="outline" className="w-full">
                    View Execution
                  </Button>
                </Link>

                <Button
                  onClick={handleDelete}
                  disabled={actionLoading !== null}
                  variant="destructive"
                  className="w-full"
                >
                  {actionLoading === 'delete' ? (
                    <>
                      <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                      Deleting...
                    </>
                  ) : (
                    <>
                      <Trash2 className="h-4 w-4 mr-2" />
                      Delete Project
                    </>
                  )}
                </Button>
              </CardContent>
            </Card>
          </div>
        </div>
      </main>
    </div>
  );
}

