'use client';

import { useEffect, useState, useRef } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { useExecutionStream } from '@/hooks/use-execution-stream';
import { useGliveAPI } from '@/hooks/use-glive-api';
import type { ExecutionEvent } from '@/types/glive';
import {
  ArrowLeft,
  Play,
  Square,
  RefreshCw,
  Loader2,
  CheckCircle2,
  XCircle,
  AlertCircle,
  Terminal
} from 'lucide-react';
import Link from 'next/link';

export default function ExecutionPage() {
  const params = useParams();
  const router = useRouter();
  const projectId = params.id as string;
  const { events, connectionState, isConnected, clearEvents } = useExecutionStream(projectId);
  const { getProject, startProject, stopProject, loading } = useGliveAPI();
  const [project, setProject] = useState<any>(null);
  const outputEndRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const loadProject = async () => {
      const data = await getProject(projectId);
      if (data) setProject(data);
    };
    loadProject();
  }, [projectId, getProject]);

  useEffect(() => {
    outputEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [events]);

  const handleStart = async () => {
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
    }
  };

  const handleStop = async () => {
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
    }
  };

  const getEventIcon = (event: ExecutionEvent) => {
    switch (event.type) {
      case 'command_started':
        return <Play className="h-4 w-4 text-blue-600" />;
      case 'command_completed':
        return event.success ?
          <CheckCircle2 className="h-4 w-4 text-green-600" /> :
          <XCircle className="h-4 w-4 text-red-600" />;
      case 'recovery_triggered':
        return <AlertCircle className="h-4 w-4 text-purple-600" />;
      case 'execution_completed':
        return event.status === 'success' ?
          <CheckCircle2 className="h-4 w-4 text-green-600" /> :
          <XCircle className="h-4 w-4 text-red-600" />;
      default:
        return null;
    }
  };

  const getEventColor = (event: ExecutionEvent) => {
    switch (event.type) {
      case 'command_started':
        return 'border-blue-900/30 bg-blue-900/10';
      case 'command_completed':
        return event.success ?
          'border-green-900/30 bg-green-900/10' :
          'border-red-900/30 bg-red-900/10';
      case 'recovery_triggered':
        return 'border-purple-900/30 bg-purple-900/10';
      case 'execution_completed':
        return event.status === 'success' ?
          'border-green-900/30 bg-green-900/10' :
          'border-red-900/30 bg-red-900/10';
      default:
        return 'border-slate-800 bg-slate-900/50';
    }
  };

  return (
    <div className="min-h-screen bg-slate-50 dark:bg-slate-900">
      <header className="dark:bg-slate-800 shadow-sm border-b border-slate-200 dark:border-slate-700">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex items-center h-16 gap-4">
            <Link href={`/dashboard/projects/${projectId}`}>
              <Button variant="ghost" size="sm">
                <ArrowLeft className="h-4 w-4 mr-2" />
                Back
              </Button>
            </Link>
            <h1 className="text-xl font-semibold text-slate-900 dark:text-slate-100">
              Execution Monitor
            </h1>
            <Badge variant={isConnected ? 'default' : 'secondary'}>
              {isConnected ? 'Connected' : connectionState}
            </Badge>
          </div>
        </div>
      </header>

      <main className="max-w-7xl mx-auto py-6 sm:px-6 lg:px-8">
        <div className="px-4 py-6 sm:px-0 space-y-6">
          {/* Controls */}
          <Card>
            <CardHeader>
              <CardTitle>Execution Controls</CardTitle>
              <CardDescription>
                Start, stop, or monitor project execution
              </CardDescription>
            </CardHeader>
            <CardContent className="flex gap-4">
              <Button
                onClick={handleStart}
                disabled={loading || project?.status === 'running'}
              >
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
              <Button
                onClick={clearEvents}
                variant="outline"
              >
                <RefreshCw className="h-4 w-4 mr-2" />
                Clear Logs
              </Button>
            </CardContent>
          </Card>

          {/* Live Output */}
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <div>
                  <CardTitle className="flex items-center gap-2">
                    <Terminal className="h-5 w-5" />
                    Live Output
                  </CardTitle>
                  <CardDescription>
                    Real-time execution logs and command output
                  </CardDescription>
                </div>
                <Badge variant={isConnected ? 'default' : 'secondary'}>
                  {events.length} events
                </Badge>
              </div>
            </CardHeader>
            <CardContent>
              <div className="bg-slate-900 text-green-400 font-mono text-sm rounded-lg p-4 h-[600px] overflow-y-auto">
                {events.length === 0 ? (
                  <div className="text-slate-500 text-center py-8">
                    {isConnected ?
                      'Waiting for execution events...' :
                      'Not connected. Start execution to see live output.'}
                  </div>
                ) : (
                  <div className="space-y-2">
                    {events.map((event, index) => (
                      <div
                        key={index}
                        className={`p-2 rounded border-l-2 ${getEventColor(event)}`}
                      >
                        <div className="flex items-start gap-2">
                          {getEventIcon(event)}
                          <div className="flex-1">
                            {event.type === 'command_started' && (
                              <div>
                                <span className="text-blue-400">▶</span> Starting: {event.command}
                              </div>
                            )}
                            {event.type === 'output_line' && (
                              <div className={event.stream === 'stderr' ? 'text-red-400' : ''}>
                                {event.line}
                              </div>
                            )}
                            {event.type === 'command_completed' && (
                              <div>
                                <span className={event.success ? 'text-green-400' : 'text-red-400'}>
                                  {event.success ? '✓' : '✗'} Command completed
                                </span>
                                {event.exit_code !== undefined && (
                                  <span className="text-slate-500 ml-2">
                                    (exit code: {event.exit_code})
                                  </span>
                                )}
                              </div>
                            )}
                            {event.type === 'recovery_triggered' && (
                              <div className="text-purple-400">
                                🔄 AI Recovery triggered: {event.reason}
                              </div>
                            )}
                            {event.type === 'recovery_plan' && (
                              <div className="text-purple-400">
                                📋 Recovery plan generated: {event.plan.analysis}
                              </div>
                            )}
                            {event.type === 'recovery_step' && (
                              <div className="text-purple-400">
                                🔧 Recovery step: {event.step.status}
                              </div>
                            )}
                            {event.type === 'execution_completed' && (
                              <div className={event.status === 'success' ? 'text-green-400' : 'text-red-400'}>
                                {event.status === 'success' ? '✓' : '✗'} Execution {event.status}
                                {event.message && <span className="ml-2">- {event.message}</span>}
                              </div>
                            )}
                            {event.type === 'error' && (
                              <div className="text-red-400">
                                ❌ Error: {event.error.message}
                              </div>
                            )}
                          </div>
                        </div>
                      </div>
                    ))}
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

