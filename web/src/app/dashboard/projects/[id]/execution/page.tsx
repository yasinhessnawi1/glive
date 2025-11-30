'use client';

import { useEffect, useState, useRef, useMemo, useCallback } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Input } from '@/components/ui/input';
import { useExecutionStream } from '@/hooks/use-execution-stream';
import { useGliveAPI } from '@/hooks/use-glive-api';
import type { ExecutionEvent, LogLevel, Project, ExecutionMilestone, AISuggestion, MilestoneStatus } from '@/types/glive';
import {
  ArrowLeft,
  Play,
  Square,
  RefreshCw,
  Loader2,
  CheckCircle2,
  XCircle,
  AlertCircle,
  Terminal,
  Download,
  Search,
  Copy,
  Wifi,
  WifiOff,
  BarChart3,
  Filter,
  ChevronDown,
  ChevronRight,
  ExternalLink,
  Lightbulb,
  Milestone,
  Globe,
  Clock,
  Zap,
  Info,
  AlertTriangle,
  FileArchive,
  FileText,
  Code2,
  FolderOpen,
} from 'lucide-react';
import Link from 'next/link';
import {
  formatTimestamp,
  formatDuration,
  getEventText,
  getEventLevel,
  filterEventsByLevel,
  searchEvents,
  exportEventsToText,
  downloadTextFile,
  calculateStatistics,
  copyToClipboard,
} from '@/lib/terminal-utils';

export default function ExecutionPage() {
  const params = useParams();
  const router = useRouter();
  const projectId = params.id as string;

  // State declarations first
  const [project, setProject] = useState<Project | null>(null);
  const [autoScroll, setAutoScroll] = useState(true);
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedLevels, setSelectedLevels] = useState<LogLevel[]>([]);
  const [showStats, setShowStats] = useState(true); // Show stats by default
  const [showFilters, setShowFilters] = useState(false);
  const [copiedIndex, setCopiedIndex] = useState<number | null>(null);
  const [collapsedCommands, setCollapsedCommands] = useState<Set<string>>(new Set());
  const [actionInProgress, setActionInProgress] = useState<'start' | 'stop' | 'reclone' | null>(null);
  const [zipDownloadError, setZipDownloadError] = useState<string | null>(null);
  const outputEndRef = useRef<HTMLDivElement>(null);

  // Active statuses that can be stopped
  const activeStatuses = ['running', 'cloning', 'analyzing', 'installing', 'pending'];
  const isProjectActive = project?.status && activeStatuses.includes(project.status);
  const canStart = project?.status && ['ready', 'stopped', 'failed'].includes(project.status);

  // Handle project not found by recreating it
  const handleProjectNotFound = useCallback(async () => {
    if (!project?.github_url) {
      // If we don't have the GitHub URL, navigate to projects page
      router.push('/dashboard/projects');
      return;
    }

    try {
      // Create a new project with the same GitHub URL
      const response = await fetch(`${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'}/api/v1/projects`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          github_url: project.github_url,
          mode: 'auto',
          force_execution: false,
        }),
      });

      if (response.ok) {
        const newProject = await response.json();
        // Navigate to the new project's execution page
        router.push(`/dashboard/projects/${newProject.id}/execution`);
      } else {
        // If recreation fails, navigate to projects page
        router.push('/dashboard/projects');
      }
    } catch (error) {
      console.error('Failed to recreate project:', error);
      router.push('/dashboard/projects');
    }
  }, [project, router]);

  const { events, connectionState, metrics, isConnected, clearEvents, reconnect } = useExecutionStream(projectId);
  const { getProject, startProject, stopProject, createProject, deleteProject, loading, getDownloadZipURL, getVSCodeURLs, getExecutionReportMarkdownURL } = useGliveAPI();

  // VS Code URLs state
  const [vscodeUrls, setVscodeUrls] = useState<{
    local_vscode_url: string;
    web_vscode_url: string;
    github_dev_url: string;
    project_path: string;
  } | null>(null);

  // Load VS Code URLs when project is ready
  useEffect(() => {
    const loadVSCodeURLs = async () => {
      if (project?.status === 'ready' || project?.status === 'running' || project?.status === 'stopped') {
        const urls = await getVSCodeURLs(projectId);
        if (urls) setVscodeUrls(urls);
      }
    };
    loadVSCodeURLs();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [project?.status, projectId]);

  // Derive milestones from events and project status
  const executionSummary = useMemo(() => {
    const milestones: ExecutionMilestone[] = [];
    const suggestions: AISuggestion[] = [];
    let previewUrl: string | undefined;
    let previewPort: number | undefined;
    let executionStartTime: number | undefined;
    let executionEndTime: number | undefined;

    // Check if this is a web project
    const webProjectTypes = ['nodejs', 'python', 'go', 'rust', 'java'];
    const isWebProject = project?.type ? webProjectTypes.includes(project.type) : false;

    // Extract preview URL from logs
    for (const event of events) {
      if (event.type === 'output_line' && event.line) {
        // Look for localhost URLs in the output
        const urlMatch = event.line.match(/https?:\/\/localhost[:\d]*(\/\S*)?/i) ||
                        event.line.match(/https?:\/\/127\.0\.0\.1[:\d]*(\/\S*)?/i) ||
                        event.line.match(/https?:\/\/0\.0\.0\.0[:\d]*(\/\S*)?/i);
        if (urlMatch) {
          previewUrl = urlMatch[0];
          const portMatch = previewUrl.match(/:(\d+)/);
          if (portMatch) {
            previewPort = parseInt(portMatch[1], 10);
          }
        }
      }
    }

    // Build milestones based on project status and events
    const statusToMilestone: Record<string, { name: string; description: string }> = {
      cloning: { name: 'Clone Repository', description: 'Downloading source code from GitHub' },
      analyzing: { name: 'Analyze Project', description: 'Detecting project type and dependencies' },
      installing: { name: 'Install Dependencies', description: 'Installing required packages' },
      running: { name: 'Start Application', description: 'Running the project' },
      ready: { name: 'Ready', description: 'Project is ready to run' },
      stopped: { name: 'Stopped', description: 'Project execution stopped' },
      failed: { name: 'Failed', description: 'Project execution failed' },
    };

    // Create milestones based on what we see in events
    const seenStages = new Set<string>();
    for (const event of events) {
      if (event.type === 'output_line' && event.line) {
        if (event.line.includes('📦') || event.line.includes('cloning')) seenStages.add('cloning');
        if (event.line.includes('📂') || event.line.includes('Project path')) seenStages.add('analyzing');
        if (event.line.includes('📍') || event.line.includes('Allocated port')) seenStages.add('installing');
        if (event.line.includes('🚀') || event.line.includes('Starting project')) {
          seenStages.add('running');
          executionStartTime = event.timestamp;
        }
        if (event.line.includes('Compiled successfully') || event.line.includes('webpack compiled')) {
          executionEndTime = event.timestamp || Date.now();
        }
      }
    }

    // Standard milestones for execution flow
    const standardMilestones = ['cloning', 'analyzing', 'installing', 'running'];
    standardMilestones.forEach((stage, index) => {
      const info = statusToMilestone[stage];
      if (info) {
        let status: MilestoneStatus = 'pending';
        if (seenStages.has(stage)) {
          status = 'completed';
        } else if (project?.status === stage) {
          status = 'in_progress';
        } else if (project?.status === 'failed') {
          // Mark subsequent milestones as skipped if failed
          const currentIndex = standardMilestones.indexOf(project.status || '');
          if (currentIndex >= 0 && index > currentIndex) {
            status = 'skipped';
          }
        }
        milestones.push({
          id: stage,
          name: info.name,
          description: info.description,
          status,
        });
      }
    });

    // Generate AI suggestions based on project state
    if (project?.status === 'running' && previewUrl) {
      suggestions.push({
        id: 'preview',
        type: 'link',
        title: 'View Your Application',
        description: `Your application is running and accessible at ${previewUrl}`,
        actionUrl: previewUrl,
        actionLabel: 'Open Preview',
        priority: 'high',
      });
    }

    if (project?.status === 'running' && !previewUrl && isWebProject) {
      suggestions.push({
        id: 'check-terminal',
        type: 'info',
        title: 'Watch Terminal Output',
        description: 'Monitor the terminal above for server startup messages and any errors.',
        priority: 'medium',
      });
    }

    if (project?.status === 'failed') {
      suggestions.push({
        id: 'retry',
        type: 'action',
        title: 'Retry Execution',
        description: 'The project failed to start. Check the error messages above and try again.',
        actionLabel: 'Retry',
        priority: 'high',
      });
    }

    if (project?.status === 'running') {
      suggestions.push({
        id: 'logs',
        type: 'info',
        title: 'Live Logs',
        description: 'The terminal shows real-time output. Look for errors or success messages.',
        priority: 'low',
      });
    }

    // Next steps based on status
    const nextSteps: string[] = [];
    if (project?.status === 'running') {
      if (previewUrl) {
        nextSteps.push(`Visit ${previewUrl} to see your application`);
      }
      nextSteps.push('Monitor the terminal for any runtime errors');
      nextSteps.push('Use the Stop button to gracefully stop the application');
    } else if (project?.status === 'ready' || project?.status === 'stopped') {
      nextSteps.push('Click Start Execution to run the project');
    } else if (project?.status === 'failed') {
      nextSteps.push('Review the error messages in the terminal');
      nextSteps.push('Fix any issues and try again');
    }

    return {
      projectId: project?.id || '',
      projectName: project?.name || '',
      projectType: project?.type || 'unknown',
      status: (project?.status === 'running' ? 'running' :
               project?.status === 'failed' ? 'failed' :
               project?.status === 'stopped' ? 'stopped' : 'completed') as 'running' | 'completed' | 'failed' | 'stopped',
      startedAt: executionStartTime,
      completedAt: executionEndTime,
      totalDuration: executionStartTime && executionEndTime ? executionEndTime - executionStartTime : undefined,
      milestones,
      suggestions,
      previewUrl,
      previewPort,
      isWebProject,
      nextSteps,
    };
  }, [events, project]);

  // Load project initially and poll for status updates
  useEffect(() => {
    const loadProject = async () => {
      const data = await getProject(projectId);
      if (data) setProject(data);
    };
    loadProject();

    // Poll for project status updates every 2 seconds when active
    const pollInterval = setInterval(async () => {
      const data = await getProject(projectId);
      if (data) setProject(data);
    }, 2000);

    return () => clearInterval(pollInterval);
  }, [projectId, getProject]);

  useEffect(() => {
    if (autoScroll) {
      outputEndRef.current?.scrollIntoView({ behavior: 'smooth' });
    }
  }, [events, autoScroll]);

  const handleStart = async () => {
    setActionInProgress('start');
    // Clear events BEFORE starting so we don't miss any new events
    clearEvents();
    try {
      // If project is already running, stop it first
      if (project?.status === 'running') {
        await stopProject(projectId);
        // Wait a bit for the stop to complete
        await new Promise(resolve => setTimeout(resolve, 1000));
      }

      const success = await startProject(projectId);
      if (success) {
        const updated = await getProject(projectId);
        if (updated) setProject(updated);
      }
    } catch (error) {
      console.error('Error starting project:', error);
    } finally {
      setActionInProgress(null);
    }
  };

  const handleStop = async () => {
    setActionInProgress('stop');
    try {
      const success = await stopProject(projectId);
      if (success) {
        const updated = await getProject(projectId);
        if (updated) setProject(updated);
      }
    } catch (error) {
      console.error('Error stopping project:', error);
    } finally {
      setActionInProgress(null);
    }
  };

  const handleReclone = async () => {
    if (!project?.github_url) return;

    setActionInProgress('reclone');
    clearEvents();
    try {
      // Stop if running
      if (project?.status === 'running') {
        await stopProject(projectId);
        await new Promise(resolve => setTimeout(resolve, 1000));
      }

      // Delete the old project
      await deleteProject(projectId);
      await new Promise(resolve => setTimeout(resolve, 500));

      // Create a new project with the same GitHub URL and force execution
      const newProject = await createProject({
        github_url: project.github_url,
        mode: 'auto',
        force_execution: true,
      });

      if (newProject) {
        // Navigate to the new project's execution page
        router.push(`/dashboard/projects/${newProject.id}/execution`);
      }
    } catch (error) {
      console.error('Error recloning project:', error);
    } finally {
      setActionInProgress(null);
    }
  };

  const handleDownloadZip = async () => {
    setZipDownloadError(null);
    try {
      const response = await fetch(getDownloadZipURL(projectId));
      if (!response.ok) {
        const errorData = await response.json().catch(() => ({ error: { message: 'Failed to download ZIP file' } }));
        if (response.status === 404 || errorData.error?.code === 'PROJECT_DIR_NOT_FOUND') {
          setZipDownloadError('Project directory not found. The project files may have been deleted. Please reclone the project.');
        } else {
          setZipDownloadError(errorData.error?.message || 'Failed to download ZIP file');
        }
        return;
      }

      // If successful, trigger download
      const blob = await response.blob();
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `${project?.name || 'project'}.zip`;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);
    } catch (error) {
      console.error('Error downloading ZIP:', error);
      setZipDownloadError('An error occurred while downloading the ZIP file. Please try again.');
    }
  };

  const handleExport = () => {
    const text = exportEventsToText(filteredEvents);
    const filename = `execution-${projectId}-${Date.now()}.txt`;
    downloadTextFile(text, filename);
  };

  const handleCopyLine = async (index: number, text: string) => {
    const success = await copyToClipboard(text);
    if (success) {
      setCopiedIndex(index);
      setTimeout(() => setCopiedIndex(null), 2000);
    }
  };

  const toggleLevel = (level: LogLevel) => {
    setSelectedLevels((prev: LogLevel[]) =>
      prev.includes(level) ? prev.filter((l: LogLevel) => l !== level) : [...prev, level]
    );
  };

  const toggleCommandCollapse = (commandId: string) => {
    setCollapsedCommands((prev: Set<string>) => {
      const next = new Set(prev);
      if (next.has(commandId)) {
        next.delete(commandId);
      } else {
        next.add(commandId);
      }
      return next;
    });
  };

  // Filter and search events
  const filteredEvents = useMemo(() => {
    // Deduplicate events based on content and timestamp
    const seen = new Set<string>();
    const deduplicated = events.filter((event) => {
      // Create a unique key for each event
      const key = JSON.stringify({
        type: event.type,
        timestamp: 'timestamp' in event ? event.timestamp : undefined,
        line: 'line' in event ? event.line : undefined,
        command: 'command' in event ? event.command : undefined,
        command_id: 'command_id' in event ? event.command_id : undefined,
      });

      if (seen.has(key)) {
        return false; // Skip duplicate
      }
      seen.add(key);
      return true;
    });

    // Filter out empty lines
    const nonEmpty = deduplicated.filter((event) => {
      if (event.type === 'output_line' && 'line' in event) {
        const line = event.line?.trim();
        return line && line.length > 0;
      }
      return true;
    });

    let filtered = nonEmpty;
    if (selectedLevels.length > 0) {
      filtered = filterEventsByLevel(filtered, selectedLevels);
    }
    if (searchQuery.trim()) {
      filtered = searchEvents(filtered, searchQuery);
    }
    return filtered;
  }, [events, selectedLevels, searchQuery]);

  // Calculate statistics
  const stats = useMemo(() => calculateStatistics(events), [events]);

  const getEventIcon = (event: ExecutionEvent) => {
    switch (event.type) {
      case 'command_started':
        return <Play className="h-4 w-4 text-blue-400" />;
      case 'command_completed':
        return event.success ? (
          <CheckCircle2 className="h-4 w-4 text-green-400" />
        ) : (
          <XCircle className="h-4 w-4 text-red-400" />
        );
      case 'recovery_triggered':
        return <AlertCircle className="h-4 w-4 text-purple-400" />;
      case 'execution_completed':
        return event.status === 'success' ? (
          <CheckCircle2 className="h-4 w-4 text-green-400" />
        ) : (
          <XCircle className="h-4 w-4 text-red-400" />
        );
      case 'error':
        return <XCircle className="h-4 w-4 text-red-400" />;
      default:
        return null;
    }
  };

  const getEventColor = (event: ExecutionEvent) => {
    switch (event.type) {
      case 'command_started':
        return 'border-blue-500/30 bg-blue-500/10';
      case 'command_completed':
        return event.success
          ? 'border-green-500/30 bg-green-500/10'
          : 'border-red-500/30 bg-red-500/10';
      case 'recovery_triggered':
        return 'border-purple-500/30 bg-purple-500/10';
      case 'execution_completed':
        return event.status === 'success'
          ? 'border-green-500/30 bg-green-500/10'
          : 'border-red-500/30 bg-red-500/10';
      case 'error':
        return 'border-red-500/30 bg-red-500/10';
      default:
        return 'border-slate-700 bg-slate-800/50';
    }
  };

  const getConnectionStatusColor = () => {
    switch (connectionState) {
      case 'connected':
        return 'bg-green-500';
      case 'connecting':
        return 'bg-yellow-500';
      case 'error':
        return 'bg-red-500';
      default:
        return 'bg-gray-500';
    }
  };

  const renderEvent = (event: ExecutionEvent, index: number) => {
    const timestamp = 'timestamp' in event ? event.timestamp : undefined;
    const eventText = getEventText(event);

    // Track current command ID from command_started events
    const currentCommandId = event.type === 'command_started' ? event.command_id : undefined;

    // Find the command_id for output_line events by looking backwards
    let associatedCommandId: string | undefined;
    if (event.type === 'output_line') {
      // Look backward to find the most recent command_started
      for (let i = index - 1; i >= 0; i--) {
        const prevEvent = filteredEvents[i];
        if (prevEvent.type === 'command_started') {
          associatedCommandId = prevEvent.command_id;
          break;
        }
        if (prevEvent.type === 'command_completed') {
          break; // Stop if we hit a completed command
        }
      }
    }

    const isCollapsed = event.type === 'command_started' && collapsedCommands.has(event.command_id);
    const isPartOfCollapsedCommand = associatedCommandId && collapsedCommands.has(associatedCommandId);

    // Hide output lines that belong to a collapsed command
    if (event.type === 'output_line' && isPartOfCollapsedCommand) {
      return null;
    }

    return (
      <div
        key={index}
        className={`group relative p-3 rounded border-l-2 ${getEventColor(event)} hover:bg-slate-800/70 transition-colors`}
      >
        <div className="flex items-start gap-2">
          {event.type === 'command_started' && (
            <button
              onClick={() => toggleCommandCollapse(event.command_id)}
              className="mt-0.5 hover:text-blue-400 transition-colors"
              title={isCollapsed ? "Expand command output" : "Collapse command output"}
            >
              {isCollapsed ? (
                <ChevronRight className="h-4 w-4" />
              ) : (
                <ChevronDown className="h-4 w-4" />
              )}
            </button>
          )}
          {getEventIcon(event)}
          <div className="flex-1 min-w-0">
            <div className="flex items-start justify-between gap-2">
              <div className="flex-1">
                {event.type === 'command_started' && (
                  <div>
                    <span className="text-blue-400">▶</span> Starting: <span className="font-semibold">{event.command}</span>
                    {isCollapsed && <span className="text-slate-500 ml-2 text-sm">(output hidden)</span>}
                  </div>
                )}
                {event.type === 'output_line' && (
                  <div className={event.stream === 'stderr' ? 'text-red-400' : 'text-slate-300'}>
                    <pre className="whitespace-pre-wrap font-mono text-sm">{event.line}</pre>
                  </div>
                )}
                {event.type === 'command_completed' && (
                  <div>
                    <span className={event.success ? 'text-green-400' : 'text-red-400'}>
                      {event.success ? '✓' : '✗'} Command completed
                    </span>
                    {event.exit_code !== undefined && (
                      <span className="text-slate-500 ml-2">(exit code: {event.exit_code})</span>
                    )}
                    {event.duration && (
                      <span className="text-slate-500 ml-2">
                        in {formatDuration(event.duration)}
                      </span>
                    )}
                  </div>
                )}
                {event.type === 'recovery_triggered' && (
                  <div className="text-purple-400">
                    🔄 AI Recovery triggered: {event.reason}
                    {event.error_output && (
                      <pre className="mt-2 text-xs text-red-400 bg-slate-900/50 p-2 rounded overflow-x-auto">
                        {event.error_output}
                      </pre>
                    )}
                  </div>
                )}
                {event.type === 'recovery_plan' && (
                  <div className="text-purple-400">
                    <div>📋 Recovery plan generated</div>
                    <div className="mt-1 text-sm text-slate-300">{event.plan.analysis}</div>
                    <div className="mt-2 text-xs text-slate-400">
                      Confidence: {Math.round(event.plan.confidence * 100)}%
                    </div>
                  </div>
                )}
                {event.type === 'recovery_step' && (
                  <div className="text-purple-400">🔧 Recovery step: {event.step.status}</div>
                )}
                {event.type === 'execution_completed' && (
                  <div className={event.status === 'success' ? 'text-green-400' : 'text-red-400'}>
                    {event.status === 'success' ? '✓' : '✗'} Execution {event.status}
                    {event.message && <span className="ml-2">- {event.message}</span>}
                  </div>
                )}
                {event.type === 'error' && (
                  <div className="text-red-400">
                    <div>❌ Error: {event.error.message}</div>
                    {event.error.details && (
                      <pre className="mt-2 text-xs bg-slate-900/50 p-2 rounded overflow-x-auto">
                        {JSON.stringify(event.error.details, null, 2)}
                      </pre>
                    )}
                  </div>
                )}
              </div>
              <div className="flex items-center gap-2 text-xs text-slate-500">
                <span>{formatTimestamp(timestamp)}</span>
                <button
                  onClick={() => handleCopyLine(index, eventText)}
                  className="opacity-0 group-hover:opacity-100 transition-opacity hover:text-slate-300"
                  title="Copy to clipboard"
                >
                  {copiedIndex === index ? (
                    <CheckCircle2 className="h-3 w-3 text-green-400" />
                  ) : (
                    <Copy className="h-3 w-3" />
                  )}
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    );
  };

  return (
    <div className="min-h-screen bg-slate-50 dark:bg-slate-900">
      <header className="dark:bg-slate-800 shadow-sm border-b border-slate-200 dark:border-slate-700">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex items-center justify-between h-16">
            <div className="flex items-center gap-4">
              <Link href={`/dashboard/projects/${projectId}`}>
                <Button variant="ghost" size="sm">
                  <ArrowLeft className="h-4 w-4 mr-2" />
                  Back
                </Button>
              </Link>
              <h1 className="text-xl font-semibold text-slate-900 dark:text-slate-100">
                Execution Monitor
              </h1>
            </div>
            <div className="flex items-center gap-3">
              <div className="flex items-center gap-2 text-sm">
                <div className={`w-2 h-2 rounded-full ${getConnectionStatusColor()}`} />
                <span className="text-slate-600 dark:text-slate-400">{connectionState}</span>
              </div>
              {metrics.reconnectAttempts > 0 && (
                <Badge variant="outline" className="text-xs">
                  Reconnects: {metrics.reconnectAttempts}
                </Badge>
              )}
            </div>
          </div>
        </div>
      </header>

      <main className="max-w-7xl mx-auto py-6 sm:px-6 lg:px-8">
        <div className="px-4 py-6 sm:px-0 space-y-6">
          {/* Controls */}
          <Card>
            <CardHeader className="pb-3">
              <div className="flex items-center justify-between">
                <div>
                  <CardTitle>Execution Controls</CardTitle>
                  <CardDescription>Start, stop, or monitor project execution</CardDescription>
                </div>
                {project && (
                  <Badge
                    variant={isProjectActive ? 'default' : 'secondary'}
                    className={isProjectActive ? 'bg-green-600 hover:bg-green-700' : ''}
                  >
                    {project.status}
                  </Badge>
                )}
              </div>
            </CardHeader>
            <CardContent className="flex flex-wrap gap-3">
              {/* Start Button - only show when can start */}
              {canStart && (
                <Button
                  onClick={handleStart}
                  disabled={actionInProgress !== null}
                  className="bg-green-600 hover:bg-green-700"
                >
                  {actionInProgress === 'start' ? (
                    <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                  ) : (
                    <Play className="h-4 w-4 mr-2" />
                  )}
                  Start Execution
                </Button>
              )}

              {/* Stop Button - show when project is active */}
              {isProjectActive && (
                <Button
                  onClick={handleStop}
                  variant="destructive"
                  disabled={actionInProgress !== null}
                >
                  {actionInProgress === 'stop' ? (
                    <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                  ) : (
                    <Square className="h-4 w-4 mr-2" />
                  )}
                  Stop Execution
                </Button>
              )}

              {/* Reclone Button - show when project exists */}
              {project && (
                <Button
                  onClick={handleReclone}
                  variant="outline"
                  disabled={actionInProgress !== null}
                  title="Re-clone and setup project from scratch"
                >
                  {actionInProgress === 'reclone' ? (
                    <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                  ) : (
                    <RefreshCw className="h-4 w-4 mr-2" />
                  )}
                  Reclone
                </Button>
              )}

              <div className="h-8 w-px bg-slate-200 dark:bg-slate-700 mx-1" />

              <Button onClick={clearEvents} variant="outline" size="default">
                <RefreshCw className="h-4 w-4 mr-2" />
                Clear Logs
              </Button>
              <Button onClick={handleExport} variant="outline" disabled={events.length === 0}>
                <Download className="h-4 w-4 mr-2" />
                Export
              </Button>
              <Button
                onClick={reconnect}
                variant="outline"
                disabled={isConnected}
                title="Manually reconnect WebSocket"
              >
                {isConnected ? <Wifi className="h-4 w-4 mr-2 text-green-500" /> : <WifiOff className="h-4 w-4 mr-2 text-red-500" />}
                {isConnected ? 'Connected' : 'Reconnect'}
              </Button>
              <Button
                onClick={() => setShowStats(!showStats)}
                variant={showStats ? 'default' : 'outline'}
                className="ml-auto"
              >
                <BarChart3 className="h-4 w-4 mr-2" />
                Stats
              </Button>
            </CardContent>
          </Card>

          {/* Connection Status - User Friendly */}
          {metrics.lastError && !isConnected && (
            <Card className="border-amber-200 dark:border-amber-800 bg-amber-50 dark:bg-amber-950/30">
              <CardContent className="pt-6">
                <div className="flex items-start gap-4">
                  {/* Icon */}
                  <div className="flex-shrink-0 w-12 h-12 rounded-full bg-amber-100 dark:bg-amber-900/50 flex items-center justify-center">
                    {metrics.lastError.code === 'WEBSOCKET_ERROR' && metrics.lastError.details?.readyState === 3 ? (
                      <WifiOff className="h-6 w-6 text-amber-600 dark:text-amber-400" />
                    ) : (
                      <AlertCircle className="h-6 w-6 text-amber-600 dark:text-amber-400" />
                    )}
                  </div>

                  {/* Content */}
                  <div className="flex-1 min-w-0">
                    <h3 className="text-lg font-semibold text-amber-800 dark:text-amber-200">
                      {metrics.lastError.code === 'WEBSOCKET_ERROR' && metrics.lastError.details?.readyState === 3
                        ? 'Unable to Connect to Server'
                        : metrics.lastError.code === 'CONNECTION_TIMEOUT'
                        ? 'Connection Timed Out'
                        : metrics.lastError.code === 'HEARTBEAT_TIMEOUT'
                        ? 'Connection Lost'
                        : 'Connection Issue'}
                    </h3>

                    <p className="mt-1 text-sm text-amber-700 dark:text-amber-300">
                      {metrics.lastError.code === 'WEBSOCKET_ERROR' && metrics.lastError.details?.readyState === 3 ? (
                        'The GLive backend server appears to be offline. Start the server and try again.'
                      ) : metrics.lastError.code === 'CONNECTION_TIMEOUT' ? (
                        'The server took too long to respond. It might be starting up or experiencing high load.'
                      ) : metrics.lastError.code === 'HEARTBEAT_TIMEOUT' ? (
                        'The connection to the server was interrupted. This could be due to network issues.'
                      ) : (
                        'Something went wrong with the connection. Please try reconnecting.'
                      )}
                    </p>

                    {/* How to fix */}
                    {metrics.lastError.code === 'WEBSOCKET_ERROR' && metrics.lastError.details?.readyState === 3 && (
                      <div className="mt-3 p-3 bg-white dark:bg-slate-800 rounded-lg border border-amber-200 dark:border-amber-800">
                        <p className="text-xs font-medium text-slate-600 dark:text-slate-400 mb-2">To start the server, run:</p>
                        <code className="text-xs bg-slate-100 dark:bg-slate-900 px-2 py-1 rounded font-mono text-slate-800 dark:text-slate-200">
                          glive agent start
                        </code>
                      </div>
                    )}

                    {/* Actions */}
                    <div className="mt-4 flex items-center gap-3">
                      <Button
                        size="sm"
                        onClick={reconnect}
                        className="bg-amber-600 hover:bg-amber-700 text-white"
                      >
                        <RefreshCw className="h-4 w-4 mr-2" />
                        Reconnect
                      </Button>
                      {metrics.reconnectAttempts > 0 && (
                        <span className="text-xs text-amber-600 dark:text-amber-400">
                          Attempted {metrics.reconnectAttempts} time{metrics.reconnectAttempts !== 1 ? 's' : ''}
                        </span>
                      )}
                    </div>
                  </div>
                </div>
              </CardContent>
            </Card>
          )}

          {/* Statistics */}
          {showStats && (
            <Card className="bg-gradient-to-br from-slate-50 to-slate-100 dark:from-slate-800 dark:to-slate-900">
              <CardHeader className="pb-2">
                <CardTitle className="text-lg">Execution Statistics</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                  <div className="p-3 rounded-lg bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 shadow-sm">
                    <div className="text-2xl font-bold text-blue-600 dark:text-blue-400">{stats.totalEvents}</div>
                    <div className="text-xs font-medium text-slate-500 dark:text-slate-400">Total Events</div>
                  </div>
                  <div className="p-3 rounded-lg bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 shadow-sm">
                    <div className="text-2xl font-bold text-green-600 dark:text-green-400">{stats.totalCommands}</div>
                    <div className="text-xs font-medium text-slate-500 dark:text-slate-400">Commands Executed</div>
                  </div>
                  <div className="p-3 rounded-lg bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 shadow-sm">
                    <div className="text-2xl font-bold text-purple-600 dark:text-purple-400">
                      {stats.totalCommands > 0
                        ? Math.round((stats.successfulCommands / stats.totalCommands) * 100)
                        : stats.successfulCommands > 0 ? 100 : 0}%
                    </div>
                    <div className="text-xs font-medium text-slate-500 dark:text-slate-400">Success Rate</div>
                  </div>
                  <div className="p-3 rounded-lg bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 shadow-sm">
                    <div className="text-2xl font-bold text-amber-600 dark:text-yellow-400">
                      {stats.averageDuration ? formatDuration(stats.averageDuration) : '—'}
                    </div>
                    <div className="text-xs font-medium text-slate-500 dark:text-slate-400">Avg Duration</div>
                  </div>
                  <div className="p-3 rounded-lg bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 shadow-sm">
                    <div className="text-2xl font-bold text-cyan-600 dark:text-cyan-400">{stats.totalOutputLines}</div>
                    <div className="text-xs font-medium text-slate-500 dark:text-slate-400">Output Lines</div>
                  </div>
                  <div className="p-3 rounded-lg bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 shadow-sm">
                    <div className="text-2xl font-bold text-red-600 dark:text-red-400">{stats.errorCount}</div>
                    <div className="text-xs font-medium text-slate-500 dark:text-slate-400">Errors</div>
                  </div>
                  <div className="p-3 rounded-lg bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 shadow-sm">
                    <div className="text-2xl font-bold text-violet-600 dark:text-purple-400">{stats.recoveryCount}</div>
                    <div className="text-xs font-medium text-slate-500 dark:text-slate-400">AI Recoveries</div>
                  </div>
                  <div className="p-3 rounded-lg bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 shadow-sm">
                    <div className="text-2xl font-bold text-slate-600 dark:text-slate-400">{metrics.messageCount}</div>
                    <div className="text-xs font-medium text-slate-500 dark:text-slate-400">WS Messages</div>
                  </div>
                </div>
              </CardContent>
            </Card>
          )}

          {/* Live Output */}
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <div>
                  <CardTitle className="flex items-center gap-2">
                    <Terminal className="h-5 w-5" />
                    Live Output
                  </CardTitle>
                  <CardDescription>Real-time execution logs and command output</CardDescription>
                </div>
                <div className="flex items-center gap-2">
                  <Badge variant={isConnected ? 'default' : 'secondary'}>
                    {filteredEvents.length} / {events.length} events
                  </Badge>
                  <Button
                    size="sm"
                    variant="outline"
                    onClick={() => setShowFilters(!showFilters)}
                  >
                    <Filter className="h-4 w-4 mr-2" />
                    Filters
                  </Button>
                </div>
              </div>
            </CardHeader>
            <CardContent className="space-y-4">
              {/* Filters */}
              {showFilters && (
                <div className="space-y-4 p-4 bg-slate-100 dark:bg-slate-800/50 rounded-lg border border-slate-200 dark:border-slate-700">
                  {/* Search */}
                  <div className="space-y-2">
                    <label className="text-sm font-medium text-slate-700 dark:text-slate-300">Search Logs</label>
                    <div className="relative">
                      <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-slate-400" />
                      <Input
                        placeholder="Search in logs..."
                        value={searchQuery}
                        onChange={(e: React.ChangeEvent<HTMLInputElement>) => setSearchQuery(e.target.value)}
                        className="pl-10 bg-white dark:bg-slate-900 border-slate-300 dark:border-slate-600"
                      />
                    </div>
                  </div>

                  {/* Level Filters */}
                  <div className="space-y-2">
                    <label className="text-sm font-medium text-slate-700 dark:text-slate-300">Filter by Level</label>
                    <div className="flex flex-wrap gap-2">
                      {(['info', 'success', 'warning', 'error'] as LogLevel[]).map((level) => {
                        const isSelected = selectedLevels.includes(level);
                        const levelColors: Record<LogLevel, string> = {
                          info: isSelected ? 'bg-blue-600 hover:bg-blue-700 text-white' : 'border-blue-400 text-blue-600 dark:text-blue-400 hover:bg-blue-50 dark:hover:bg-blue-900/20',
                          success: isSelected ? 'bg-green-600 hover:bg-green-700 text-white' : 'border-green-400 text-green-600 dark:text-green-400 hover:bg-green-50 dark:hover:bg-green-900/20',
                          warning: isSelected ? 'bg-yellow-600 hover:bg-yellow-700 text-white' : 'border-yellow-400 text-yellow-600 dark:text-yellow-400 hover:bg-yellow-50 dark:hover:bg-yellow-900/20',
                          error: isSelected ? 'bg-red-600 hover:bg-red-700 text-white' : 'border-red-400 text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20',
                        };
                        return (
                          <Button
                            key={level}
                            size="sm"
                            variant={isSelected ? 'default' : 'outline'}
                            onClick={() => toggleLevel(level)}
                            className={`capitalize ${levelColors[level]}`}
                          >
                            {level}
                          </Button>
                        );
                      })}
                      {selectedLevels.length > 0 && (
                        <Button
                          size="sm"
                          variant="ghost"
                          onClick={() => setSelectedLevels([])}
                          className="text-slate-500 hover:text-slate-700 dark:hover:text-slate-300"
                        >
                          Clear All
                        </Button>
                      )}
                    </div>
                  </div>

                  {/* Options */}
                  <div className="flex items-center justify-between pt-2 border-t border-slate-200 dark:border-slate-700">
                    <div className="flex items-center gap-2">
                      <input
                        type="checkbox"
                        id="autoScroll"
                        checked={autoScroll}
                        onChange={(e) => setAutoScroll(e.target.checked)}
                        className="rounded border-slate-300 dark:border-slate-600 text-blue-600"
                      />
                      <label htmlFor="autoScroll" className="text-sm text-slate-600 dark:text-slate-400 cursor-pointer">
                        Auto-scroll to new entries
                      </label>
                    </div>
                    <span className="text-xs text-slate-500">
                      Showing {filteredEvents.length} of {events.length} events
                    </span>
                  </div>
                </div>
              )}

              {/* Terminal Output */}
              <div className="bg-slate-900 text-green-400 text-sm rounded-lg p-4 h-[600px] overflow-y-auto" style={{ fontFamily: 'ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, "Liberation Mono", monospace, "Apple Color Emoji", "Segoe UI Emoji", "Segoe UI Symbol", "Noto Color Emoji"' }}>
                {filteredEvents.length === 0 ? (
                  <div className="text-slate-500 text-center py-8">
                    {events.length === 0 ? (
                      <div className="space-y-2">
                        <Terminal className="h-12 w-12 mx-auto opacity-50" />
                        <div>
                          {isConnected
                            ? 'Waiting for execution events...'
                            : 'Not connected. Start execution to see live output.'}
                        </div>
                        {!isConnected && connectionState === 'error' && (
                          <div className="mt-4">
                            <Button onClick={reconnect} variant="outline" size="sm">
                              <WifiOff className="h-4 w-4 mr-2" />
                              Try Reconnecting
                            </Button>
                          </div>
                        )}
                      </div>
                    ) : (
                      <div>No events match your filters</div>
                    )}
                  </div>
                ) : (
                  <div className="space-y-2">
                    {filteredEvents.map((event: ExecutionEvent, index: number) => renderEvent(event, index))}
                    <div ref={outputEndRef} />
                  </div>
                )}
              </div>
            </CardContent>
          </Card>

          {/* Milestones & Summary Card */}
          <Card className="bg-gradient-to-br from-slate-50 to-slate-100 dark:from-slate-800 dark:to-slate-900">
            <CardHeader>
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <Milestone className="h-5 w-5 text-blue-500" />
                  <CardTitle>Execution Progress & Summary</CardTitle>
                </div>
                {executionSummary.previewUrl && project?.status === 'running' && (
                  <a
                    href={executionSummary.previewUrl}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="inline-flex items-center gap-2 px-4 py-2 bg-green-600 hover:bg-green-700 text-white rounded-lg text-sm font-medium transition-colors"
                  >
                    <Globe className="h-4 w-4" />
                    Open Preview
                    <ExternalLink className="h-3 w-3" />
                  </a>
                )}
              </div>
              <CardDescription>Track execution milestones and get AI-powered suggestions</CardDescription>
            </CardHeader>
            <CardContent className="space-y-6">
              {/* Milestones Progress */}
              <div className="space-y-4">
                <h3 className="text-sm font-semibold text-slate-700 dark:text-slate-300 flex items-center gap-2">
                  <Clock className="h-4 w-4" />
                  Execution Milestones
                </h3>
                <div className="relative">
                  {/* Progress line */}
                  <div className="absolute left-4 top-0 bottom-0 w-0.5 bg-slate-200 dark:bg-slate-700" />

                  <div className="space-y-4">
                    {executionSummary.milestones.map((milestone, index) => (
                      <div key={milestone.id} className="relative flex items-start gap-4 pl-2">
                        {/* Status indicator */}
                        <div className={`relative z-10 flex items-center justify-center w-5 h-5 rounded-full border-2 ${
                          milestone.status === 'completed'
                            ? 'bg-green-500 border-green-500'
                            : milestone.status === 'in_progress'
                            ? 'bg-blue-500 border-blue-500 animate-pulse'
                            : milestone.status === 'failed'
                            ? 'bg-red-500 border-red-500'
                            : milestone.status === 'skipped'
                            ? 'bg-slate-400 border-slate-400'
                            : 'bg-white dark:bg-slate-800 border-slate-300 dark:border-slate-600'
                        }`}>
                          {milestone.status === 'completed' && (
                            <CheckCircle2 className="h-3 w-3 text-white" />
                          )}
                          {milestone.status === 'in_progress' && (
                            <Loader2 className="h-3 w-3 text-white animate-spin" />
                          )}
                          {milestone.status === 'failed' && (
                            <XCircle className="h-3 w-3 text-white" />
                          )}
                        </div>

                        {/* Milestone content */}
                        <div className="flex-1 pb-4">
                          <div className="flex items-center gap-2">
                            <span className={`font-medium ${
                              milestone.status === 'completed'
                                ? 'text-green-700 dark:text-green-400'
                                : milestone.status === 'in_progress'
                                ? 'text-blue-700 dark:text-blue-400'
                                : milestone.status === 'failed'
                                ? 'text-red-700 dark:text-red-400'
                                : 'text-slate-500 dark:text-slate-400'
                            }`}>
                              {milestone.name}
                            </span>
                            {milestone.status === 'in_progress' && (
                              <Badge variant="outline" className="text-xs bg-blue-50 dark:bg-blue-900/30 border-blue-200 dark:border-blue-800 text-blue-700 dark:text-blue-300">
                                In Progress
                              </Badge>
                            )}
                          </div>
                          <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5">
                            {milestone.description}
                          </p>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              </div>

              {/* AI Suggestions */}
              {executionSummary.suggestions.length > 0 && (
                <div className="space-y-4">
                  <h3 className="text-sm font-semibold text-slate-700 dark:text-slate-300 flex items-center gap-2">
                    <Lightbulb className="h-4 w-4 text-yellow-500" />
                    Suggestions
                  </h3>
                  <div className="space-y-3">
                    {executionSummary.suggestions.map((suggestion) => (
                      <div
                        key={suggestion.id}
                        className={`p-4 rounded-lg border ${
                          suggestion.type === 'link'
                            ? 'bg-green-50 dark:bg-green-900/20 border-green-200 dark:border-green-800'
                            : suggestion.type === 'warning'
                            ? 'bg-amber-50 dark:bg-amber-900/20 border-amber-200 dark:border-amber-800'
                            : suggestion.type === 'action'
                            ? 'bg-blue-50 dark:bg-blue-900/20 border-blue-200 dark:border-blue-800'
                            : 'bg-slate-50 dark:bg-slate-800/50 border-slate-200 dark:border-slate-700'
                        }`}
                      >
                        <div className="flex items-start gap-3">
                          <div className={`flex-shrink-0 w-8 h-8 rounded-full flex items-center justify-center ${
                            suggestion.type === 'link'
                              ? 'bg-green-100 dark:bg-green-800'
                              : suggestion.type === 'warning'
                              ? 'bg-amber-100 dark:bg-amber-800'
                              : suggestion.type === 'action'
                              ? 'bg-blue-100 dark:bg-blue-800'
                              : 'bg-slate-100 dark:bg-slate-700'
                          }`}>
                            {suggestion.type === 'link' && <Globe className="h-4 w-4 text-green-600 dark:text-green-400" />}
                            {suggestion.type === 'warning' && <AlertTriangle className="h-4 w-4 text-amber-600 dark:text-amber-400" />}
                            {suggestion.type === 'action' && <Zap className="h-4 w-4 text-blue-600 dark:text-blue-400" />}
                            {suggestion.type === 'info' && <Info className="h-4 w-4 text-slate-600 dark:text-slate-400" />}
                          </div>
                          <div className="flex-1 min-w-0">
                            <h4 className="text-sm font-medium text-slate-800 dark:text-slate-200">
                              {suggestion.title}
                            </h4>
                            <p className="text-xs text-slate-600 dark:text-slate-400 mt-1">
                              {suggestion.description}
                            </p>
                            {suggestion.actionUrl && (
                              <a
                                href={suggestion.actionUrl}
                                target="_blank"
                                rel="noopener noreferrer"
                                className="inline-flex items-center gap-1 mt-2 text-sm font-medium text-green-600 dark:text-green-400 hover:underline"
                              >
                                {suggestion.actionLabel || 'Open'}
                                <ExternalLink className="h-3 w-3" />
                              </a>
                            )}
                            {suggestion.type === 'action' && suggestion.actionLabel && !suggestion.actionUrl && (
                              <Button
                                size="sm"
                                variant="outline"
                                className="mt-2"
                                onClick={handleStart}
                                disabled={actionInProgress !== null}
                              >
                                {suggestion.actionLabel}
                              </Button>
                            )}
                          </div>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              )}

              {/* Next Steps */}
              {executionSummary.nextSteps.length > 0 && (
                <div className="space-y-3">
                  <h3 className="text-sm font-semibold text-slate-700 dark:text-slate-300 flex items-center gap-2">
                    <ChevronRight className="h-4 w-4" />
                    Next Steps
                  </h3>
                  <ul className="space-y-2">
                    {executionSummary.nextSteps.map((step, index) => (
                      <li key={index} className="flex items-start gap-2 text-sm text-slate-600 dark:text-slate-400">
                        <span className="flex-shrink-0 w-5 h-5 rounded-full bg-slate-200 dark:bg-slate-700 flex items-center justify-center text-xs font-medium">
                          {index + 1}
                        </span>
                        <span>{step}</span>
                      </li>
                    ))}
                  </ul>
                </div>
              )}

              {/* Preview URL Banner */}
              {executionSummary.previewUrl && project?.status === 'running' && (
                <div className="p-4 bg-gradient-to-r from-green-500 to-emerald-600 rounded-lg text-white">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-3">
                      <div className="w-10 h-10 rounded-full bg-white/20 flex items-center justify-center">
                        <Globe className="h-5 w-5" />
                      </div>
                      <div>
                        <h4 className="font-semibold">Application Running!</h4>
                        <p className="text-sm text-green-100">
                          Your application is live at: <code className="bg-white/20 px-2 py-0.5 rounded">{executionSummary.previewUrl}</code>
                        </p>
                      </div>
                    </div>
                    <a
                      href={executionSummary.previewUrl}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="inline-flex items-center gap-2 px-4 py-2 bg-white text-green-600 rounded-lg text-sm font-medium hover:bg-green-50 transition-colors"
                    >
                      Open in Browser
                      <ExternalLink className="h-4 w-4" />
                    </a>
                  </div>
                </div>
              )}

              {/* Security Notice for Web Projects */}
              {executionSummary.isWebProject && project?.status === 'running' && (
                <div className="p-3 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg">
                  <div className="flex items-start gap-2">
                    <Info className="h-4 w-4 text-blue-600 dark:text-blue-400 mt-0.5 flex-shrink-0" />
                    <div className="text-xs text-blue-700 dark:text-blue-300">
                      <strong>Security Note:</strong> The preview URL is only accessible from your local machine.
                      For production deployment, consider using proper hosting services with HTTPS.
                    </div>
                  </div>
                </div>
              )}
            </CardContent>
          </Card>

          {/* Quick Actions Card */}
          {(project?.status === 'ready' || project?.status === 'running' || project?.status === 'stopped' || project?.status === 'failed') && (
            <Card>
              <CardHeader>
                <div className="flex items-center gap-2">
                  <FolderOpen className="h-5 w-5 text-purple-500" />
                  <CardTitle>Quick Actions</CardTitle>
                </div>
                <CardDescription>Download, open in editor, or export project</CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                {/* ZIP Download Error Alert */}
                {zipDownloadError && (
                  <div className="col-span-full p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg">
                    <div className="flex items-start gap-2">
                      <AlertTriangle className="h-4 w-4 text-red-600 dark:text-red-400 mt-0.5 flex-shrink-0" />
                      <div className="flex-1 text-xs text-red-700 dark:text-red-300">
                        <strong>Error:</strong> {zipDownloadError}
                      </div>
                      <button
                        onClick={() => setZipDownloadError(null)}
                        className="text-red-600 dark:text-red-400 hover:text-red-800 dark:hover:text-red-200"
                      >
                        <XCircle className="h-4 w-4" />
                      </button>
                    </div>
                    {zipDownloadError.includes('directory not found') && (
                      <Button
                        onClick={handleReclone}
                        variant="outline"
                        size="sm"
                        className="mt-2 text-xs"
                        disabled={actionInProgress !== null}
                      >
                        <RefreshCw className="h-3 w-3 mr-1" />
                        Reclone Project
                      </Button>
                    )}
                  </div>
                )}

                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
                  {/* Download ZIP */}
                  <button
                    onClick={handleDownloadZip}
                    className="flex flex-col items-center gap-3 p-4 bg-slate-50 dark:bg-slate-800/50 rounded-lg border border-slate-200 dark:border-slate-700 hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors group"
                  >
                    <div className="w-12 h-12 rounded-full bg-blue-100 dark:bg-blue-900/50 flex items-center justify-center group-hover:bg-blue-200 dark:group-hover:bg-blue-900 transition-colors">
                      <FileArchive className="h-6 w-6 text-blue-600 dark:text-blue-400" />
                    </div>
                    <div className="text-center">
                      <h4 className="text-sm font-medium text-slate-800 dark:text-slate-200">Download ZIP</h4>
                      <p className="text-xs text-slate-500 dark:text-slate-400 mt-1">Get project files without node_modules</p>
                    </div>
                  </button>

                  {/* Open in VS Code (Local) */}
                  {vscodeUrls?.local_vscode_url && (
                    <a
                      href={vscodeUrls.local_vscode_url}
                      className="flex flex-col items-center gap-3 p-4 bg-slate-50 dark:bg-slate-800/50 rounded-lg border border-slate-200 dark:border-slate-700 hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors group"
                    >
                      <div className="w-12 h-12 rounded-full bg-purple-100 dark:bg-purple-900/50 flex items-center justify-center group-hover:bg-purple-200 dark:group-hover:bg-purple-900 transition-colors">
                        <Code2 className="h-6 w-6 text-purple-600 dark:text-purple-400" />
                      </div>
                      <div className="text-center">
                        <h4 className="text-sm font-medium text-slate-800 dark:text-slate-200">Open in VS Code</h4>
                        <p className="text-xs text-slate-500 dark:text-slate-400 mt-1">Open project in local editor</p>
                      </div>
                    </a>
                  )}

                  {/* Open in VS Code Web */}
                  {vscodeUrls?.web_vscode_url && (
                    <a
                      href={vscodeUrls.web_vscode_url}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="flex flex-col items-center gap-3 p-4 bg-slate-50 dark:bg-slate-800/50 rounded-lg border border-slate-200 dark:border-slate-700 hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors group"
                    >
                      <div className="w-12 h-12 rounded-full bg-green-100 dark:bg-green-900/50 flex items-center justify-center group-hover:bg-green-200 dark:group-hover:bg-green-900 transition-colors">
                        <Globe className="h-6 w-6 text-green-600 dark:text-green-400" />
                      </div>
                      <div className="text-center">
                        <h4 className="text-sm font-medium text-slate-800 dark:text-slate-200">VS Code Web</h4>
                        <p className="text-xs text-slate-500 dark:text-slate-400 mt-1">Edit on vscode.dev</p>
                      </div>
                    </a>
                  )}

                  {/* GitHub.dev */}
                  {vscodeUrls?.github_dev_url && (
                    <a
                      href={vscodeUrls.github_dev_url}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="flex flex-col items-center gap-3 p-4 bg-slate-50 dark:bg-slate-800/50 rounded-lg border border-slate-200 dark:border-slate-700 hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors group"
                    >
                      <div className="w-12 h-12 rounded-full bg-slate-100 dark:bg-slate-700 flex items-center justify-center group-hover:bg-slate-200 dark:group-hover:bg-slate-600 transition-colors">
                        <svg className="h-6 w-6 text-slate-700 dark:text-slate-300" viewBox="0 0 24 24" fill="currentColor">
                          <path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z"/>
                        </svg>
                      </div>
                      <div className="text-center">
                        <h4 className="text-sm font-medium text-slate-800 dark:text-slate-200">GitHub.dev</h4>
                        <p className="text-xs text-slate-500 dark:text-slate-400 mt-1">Edit on github.dev</p>
                      </div>
                    </a>
                  )}

                  {/* Download Report */}
                  <a
                    href={getExecutionReportMarkdownURL(projectId)}
                    className="flex flex-col items-center gap-3 p-4 bg-slate-50 dark:bg-slate-800/50 rounded-lg border border-slate-200 dark:border-slate-700 hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors group"
                  >
                    <div className="w-12 h-12 rounded-full bg-amber-100 dark:bg-amber-900/50 flex items-center justify-center group-hover:bg-amber-200 dark:group-hover:bg-amber-900 transition-colors">
                      <FileText className="h-6 w-6 text-amber-600 dark:text-amber-400" />
                    </div>
                    <div className="text-center">
                      <h4 className="text-sm font-medium text-slate-800 dark:text-slate-200">Download Report</h4>
                      <p className="text-xs text-slate-500 dark:text-slate-400 mt-1">Step-by-step setup guide</p>
                    </div>
                  </a>
                </div>

                {/* Project Path Info */}
                {vscodeUrls?.project_path && (
                  <div className="p-3 bg-slate-100 dark:bg-slate-800 rounded-lg">
                    <div className="flex items-center justify-between gap-2">
                      <div className="flex items-center gap-2 min-w-0">
                        <FolderOpen className="h-4 w-4 text-slate-500 flex-shrink-0" />
                        <code className="text-xs text-slate-600 dark:text-slate-400 truncate">{vscodeUrls.project_path}</code>
                      </div>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => copyToClipboard(vscodeUrls.project_path)}
                        className="flex-shrink-0"
                      >
                        <Copy className="h-3 w-3" />
                      </Button>
                    </div>
                  </div>
                )}
              </CardContent>
            </Card>
          )}
        </div>
      </main>
    </div>
  );
}
