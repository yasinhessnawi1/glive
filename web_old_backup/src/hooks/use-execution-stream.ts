// React hook for execution WebSocket stream
import { useState, useEffect, useRef, useCallback } from 'react';
import { GliveWebSocket, type ConnectionState } from '@/lib/glive-websocket';
import type { ExecutionEvent, ConnectionMetrics } from '@/types/glive';
import { isValidEvent, deduplicateEvents } from '@/lib/terminal-utils';

const MAX_EVENTS = 10000; // Limit to prevent memory issues

export function useExecutionStream(projectId: string | null) {
  const [events, setEvents] = useState<ExecutionEvent[]>([]);
  const [connectionState, setConnectionState] = useState<ConnectionState>('disconnected');
  const [metrics, setMetrics] = useState<ConnectionMetrics>({
    reconnectAttempts: 0,
    messageCount: 0,
  });
  const wsRef = useRef<GliveWebSocket | null>(null);
  const eventBufferRef = useRef<ExecutionEvent[]>([]);
  const bufferTimerRef = useRef<NodeJS.Timeout | null>(null);

  // Flush buffered events periodically
  const flushEventBuffer = useCallback(() => {
    if (eventBufferRef.current.length > 0) {
      setEvents((prev) => {
        const combined = [...prev, ...eventBufferRef.current];
        // Deduplicate and limit size
        const deduplicated = deduplicateEvents(combined);
        const limited = deduplicated.slice(-MAX_EVENTS);
        eventBufferRef.current = [];
        return limited;
      });
    }
  }, []);

  // Add event to buffer
  const addEvent = useCallback((event: ExecutionEvent) => {
    // Validate event before adding
    if (!isValidEvent(event)) {
      console.warn('Invalid event received:', event);
      return;
    }

    eventBufferRef.current.push(event);

    // Flush buffer if it gets too large
    if (eventBufferRef.current.length >= 50) {
      flushEventBuffer();
    }
  }, [flushEventBuffer]);

  useEffect(() => {
    if (!projectId) {
      return;
    }

    const ws = new GliveWebSocket(projectId);
    wsRef.current = ws;

    // Connection state handler
    ws.onConnectionStateChange((state) => {
      setConnectionState(state);
    });

    // Metrics handler
    ws.onMetricsChange((newMetrics) => {
      setMetrics(newMetrics);
    });

    // Command started
    ws.onCommandStarted((payload) => {
      addEvent({
        type: 'command_started',
        command: payload.command || 'Unknown command',
        command_id: payload.command_id || 'unknown',
        timestamp: payload.timestamp || Date.now(),
      });
    });

    // Command output
    ws.onCommandOutput((payload) => {
      addEvent({
        type: 'output_line',
        stream: payload.stream || 'stdout',
        line: payload.output || '',
        timestamp: payload.timestamp || Date.now(),
      });
    });

    // Command complete
    ws.onCommandComplete((payload) => {
      addEvent({
        type: 'command_completed',
        exit_code: payload.exit_code ?? -1,
        success: payload.success ?? false,
        duration: payload.duration,
      });
    });

    // Recovery triggered
    ws.onRecoveryTriggered((payload) => {
      addEvent({
        type: 'recovery_triggered',
        reason: payload.reason || 'Unknown reason',
        error_output: payload.error_output,
      });
    });

    // Recovery plan
    ws.onRecoveryPlan((payload) => {
      addEvent({
        type: 'recovery_plan',
        plan: payload,
      });
    });

    // Recovery step
    ws.onRecoveryStep((payload) => {
      addEvent({
        type: 'recovery_step',
        step: payload,
      });
    });

    // Execution completed
    ws.onExecutionCompleted((payload) => {
      addEvent({
        type: 'execution_completed',
        status: payload.status || 'failed',
        message: payload.message,
      });
    });

    // Error
    ws.onError((payload) => {
      addEvent({
        type: 'error',
        error: payload,
      });
    });

    ws.connect();

    // Set up buffer flush interval
    bufferTimerRef.current = setInterval(flushEventBuffer, 100);

    return () => {
      if (bufferTimerRef.current) {
        clearInterval(bufferTimerRef.current);
      }
      flushEventBuffer(); // Flush any remaining events
      ws.disconnect();
      wsRef.current = null;
    };
  }, [projectId, addEvent, flushEventBuffer]);

  const clearEvents = useCallback(() => {
    setEvents([]);
    eventBufferRef.current = [];
  }, []);

  const reconnect = useCallback(() => {
    wsRef.current?.reconnect();
  }, []);

  return {
    events,
    connectionState,
    metrics,
    isConnected: connectionState === 'connected',
    clearEvents,
    reconnect,
  };
}

