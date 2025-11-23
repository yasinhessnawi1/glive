'use client';

import { useEffect, useState, useRef, useMemo } from 'react';
import { useParams } from 'next/navigation';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Input } from '@/components/ui/input';
import { useExecutionStream } from '@/hooks/use-execution-stream';
import { useGliveAPI } from '@/hooks/use-glive-api';
import type { ExecutionEvent, LogLevel } from '@/types/glive';
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
  const projectId = params.id as string;
  const { events, connectionState, metrics, isConnected, clearEvents, reconnect } = useExecutionStream(projectId);
  const { getProject, startProject, stopProject, loading } = useGliveAPI();
  const [project, setProject] = useState<any>(null);
  const outputEndRef = useRef<HTMLDivElement>(null);
  const [autoScroll, setAutoScroll] = useState(true);
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedLevels, setSelectedLevels] = useState<LogLevel[]>([]);
  const [showStats, setShowStats] = useState(false);
  const [showFilters, setShowFilters] = useState(false);
  const [copiedIndex, setCopiedIndex] = useState<number | null>(null);
  const [collapsedCommands, setCollapsedCommands] = useState<Set<string>>(new Set());

  useEffect(() => {
    const loadProject = async () => {
      const data = await getProject(projectId);
      if (data) setProject(data);
    };
    loadProject();
  }, [projectId, getProject]);

  useEffect(() => {
    if (autoScroll) {
      outputEndRef.current?.scrollIntoView({ behavior: 'smooth' });
    }
  }, [events, autoScroll]);

  const handleStart = async () => {
    try {
      const success = await startProject(projectId);
      if (success) {
        const updated = await getProject(projectId);
        if (updated) setProject(updated);
      }
    } catch (error) {
      console.error('Error starting project:', error);
    }
  };

  const handleStop = async () => {
    try {
      const success = await stopProject(projectId);
      if (success) {
        const updated = await getProject(projectId);
        if (updated) setProject(updated);
      }
    } catch (error) {
      console.error('Error stopping project:', error);
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
    setSelectedLevels((prev) =>
      prev.includes(level) ? prev.filter((l) => l !== level) : [...prev, level]
    );
  };

  const toggleCommandCollapse = (commandId: string) => {
    setCollapsedCommands((prev) => {
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
    let filtered = events;
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
    const isCollapsed = event.type === 'command_started' && collapsedCommands.has(event.command_id);

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
                    <span className="text-blue-400">▶</span> Starting: {event.command}
                  </div>
                )}
                {event.type === 'output_line' && (
                  <div className={event.stream === 'stderr' ? 'text-red-400' : 'text-green-400'}>
                    {event.line}
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
            <CardHeader>
              <CardTitle>Execution Controls</CardTitle>
              <CardDescription>Start, stop, or monitor project execution</CardDescription>
            </CardHeader>
            <CardContent className="flex flex-wrap gap-4">
              <Button onClick={handleStart} disabled={loading || project?.status === 'running'}>
                <Play className="h-4 w-4 mr-2" />
                Start Execution
              </Button>
              <Button
                onClick={handleStop}
                variant="destructive"
                disabled={loading || project?.status !== 'running'}
              >
                <Square className="h-4 w-4 mr-2" />
                Stop Execution
              </Button>
              <Button onClick={clearEvents} variant="outline">
                <RefreshCw className="h-4 w-4 mr-2" />
                Clear Logs
              </Button>
              <Button onClick={handleExport} variant="outline" disabled={events.length === 0}>
                <Download className="h-4 w-4 mr-2" />
                Export Logs
              </Button>
              <Button
                onClick={reconnect}
                variant="outline"
                disabled={isConnected}
                title="Manually reconnect WebSocket"
              >
                {isConnected ? <Wifi className="h-4 w-4 mr-2" /> : <WifiOff className="h-4 w-4 mr-2" />}
                Reconnect
              </Button>
              <Button
                onClick={() => setShowStats(!showStats)}
                variant="outline"
                className="ml-auto"
              >
                <BarChart3 className="h-4 w-4 mr-2" />
                {showStats ? 'Hide' : 'Show'} Stats
              </Button>
            </CardContent>
          </Card>

          {/* Connection Status Details */}
          {metrics.lastError && (
            <Card className="border-red-500/50 bg-red-500/5">
              <CardHeader>
                <CardTitle className="text-red-400 flex items-center gap-2">
                  <AlertCircle className="h-5 w-5" />
                  Connection Error
                </CardTitle>
              </CardHeader>
              <CardContent>
                <div className="space-y-2 text-sm">
                  <div>
                    <span className="font-semibold">Code:</span> {metrics.lastError.code}
                  </div>
                  <div>
                    <span className="font-semibold">Message:</span> {metrics.lastError.message}
                  </div>
                  <div className="text-xs text-slate-500">
                    {new Date(metrics.lastError.timestamp).toLocaleString()}
                  </div>
                  {metrics.lastError.details && (
                    <pre className="mt-2 text-xs bg-slate-900/50 p-2 rounded overflow-x-auto">
                      {JSON.stringify(metrics.lastError.details, null, 2)}
                    </pre>
                  )}
                </div>
              </CardContent>
            </Card>
          )}

          {/* Statistics */}
          {showStats && (
            <Card>
              <CardHeader>
                <CardTitle>Execution Statistics</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                  <div className="space-y-1">
                    <div className="text-2xl font-bold text-blue-400">{stats.totalEvents}</div>
                    <div className="text-xs text-slate-500">Total Events</div>
                  </div>
                  <div className="space-y-1">
                    <div className="text-2xl font-bold text-green-400">{stats.totalCommands}</div>
                    <div className="text-xs text-slate-500">Commands Executed</div>
                  </div>
                  <div className="space-y-1">
                    <div className="text-2xl font-bold text-purple-400">
                      {stats.totalCommands > 0
                        ? Math.round((stats.successfulCommands / stats.totalCommands) * 100)
                        : 0}
                      %
                    </div>
                    <div className="text-xs text-slate-500">Success Rate</div>
                  </div>
                  <div className="space-y-1">
                    <div className="text-2xl font-bold text-yellow-400">
                      {stats.averageDuration ? formatDuration(stats.averageDuration) : 'N/A'}
                    </div>
                    <div className="text-xs text-slate-500">Avg Duration</div>
                  </div>
                  <div className="space-y-1">
                    <div className="text-2xl font-bold text-cyan-400">{stats.totalOutputLines}</div>
                    <div className="text-xs text-slate-500">Output Lines</div>
                  </div>
                  <div className="space-y-1">
                    <div className="text-2xl font-bold text-red-400">{stats.errorCount}</div>
                    <div className="text-xs text-slate-500">Errors</div>
                  </div>
                  <div className="space-y-1">
                    <div className="text-2xl font-bold text-purple-400">{stats.recoveryCount}</div>
                    <div className="text-xs text-slate-500">Recoveries</div>
                  </div>
                  <div className="space-y-1">
                    <div className="text-2xl font-bold text-slate-400">{metrics.messageCount}</div>
                    <div className="text-xs text-slate-500">WS Messages</div>
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
                <div className="space-y-3 p-4 bg-slate-800/50 rounded-lg border border-slate-700">
                  <div className="flex items-center gap-2">
                    <Search className="h-4 w-4 text-slate-400" />
                    <Input
                      placeholder="Search logs..."
                      value={searchQuery}
                      onChange={(e) => setSearchQuery(e.target.value)}
                      className="flex-1 bg-slate-900 border-slate-700"
                    />
                  </div>
                  <div className="flex flex-wrap gap-2">
                    <span className="text-sm text-slate-400">Filter by level:</span>
                    {(['info', 'success', 'warning', 'error'] as LogLevel[]).map((level) => (
                      <Button
                        key={level}
                        size="sm"
                        variant={selectedLevels.includes(level) ? 'default' : 'outline'}
                        onClick={() => toggleLevel(level)}
                        className="text-xs"
                      >
                        {level}
                      </Button>
                    ))}
                    {selectedLevels.length > 0 && (
                      <Button
                        size="sm"
                        variant="ghost"
                        onClick={() => setSelectedLevels([])}
                        className="text-xs"
                      >
                        Clear
                      </Button>
                    )}
                  </div>
                  <div className="flex items-center gap-2">
                    <input
                      type="checkbox"
                      id="autoScroll"
                      checked={autoScroll}
                      onChange={(e) => setAutoScroll(e.target.checked)}
                      className="rounded"
                    />
                    <label htmlFor="autoScroll" className="text-sm text-slate-400 cursor-pointer">
                      Auto-scroll to bottom
                    </label>
                  </div>
                </div>
              )}

              {/* Terminal Output */}
              <div className="bg-slate-900 text-green-400 font-mono text-sm rounded-lg p-4 h-[600px] overflow-y-auto">
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
                    {filteredEvents.map((event, index) => renderEvent(event, index))}
                    <div ref={outputEndRef} />
                  </div>
                )}
              </div>
            </CardContent>
          </Card>
        </div>
      </main>
    </div>
  );
}
