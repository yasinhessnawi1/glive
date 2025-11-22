// React hook for execution WebSocket stream
import { useState, useEffect, useRef, useCallback } from 'react';
import { GliveWebSocket, type ConnectionState } from '@/lib/glive-websocket';
import type { ExecutionEvent } from '@/types/glive';

export function useExecutionStream(projectId: string | null) {
  const [events, setEvents] = useState<ExecutionEvent[]>([]);
  const [connectionState, setConnectionState] = useState<ConnectionState>('disconnected');
  const wsRef = useRef<GliveWebSocket | null>(null);

  useEffect(() => {
    if (!projectId) {
      return;
    }

    // Only connect if projectId is valid
    if (!projectId) {
      return;
    }

    const ws = new GliveWebSocket(projectId);
    wsRef.current = ws;

    // Connection state handler
    ws.onConnectionStateChange((state) => {
      setConnectionState(state);
    });

    // Command started
    ws.onCommandStarted((payload) => {
      setEvents((prev) => [
        ...prev,
        {
          type: 'command_started',
          command: payload.command,
          command_id: payload.command_id,
          timestamp: payload.timestamp,
        },
      ]);
    });

    // Command output
    ws.onCommandOutput((payload) => {
      setEvents((prev) => [
        ...prev,
        {
          type: 'output_line',
          stream: payload.stream,
          line: payload.output,
          timestamp: payload.timestamp,
        },
      ]);
    });

    // Command complete
    ws.onCommandComplete((payload) => {
      setEvents((prev) => [
        ...prev,
        {
          type: 'command_completed',
          exit_code: payload.exit_code,
          success: payload.success,
          duration: payload.duration,
        },
      ]);
    });

    // Recovery triggered
    ws.onRecoveryTriggered((payload) => {
      setEvents((prev) => [
        ...prev,
        {
          type: 'recovery_triggered',
          reason: payload.reason,
          error_output: payload.error_output,
        },
      ]);
    });

    // Recovery plan
    ws.onRecoveryPlan((payload) => {
      setEvents((prev) => [
        ...prev,
        {
          type: 'recovery_plan',
          plan: payload,
        },
      ]);
    });

    // Recovery step
    ws.onRecoveryStep((payload) => {
      setEvents((prev) => [
        ...prev,
        {
          type: 'recovery_step',
          step: payload,
        },
      ]);
    });

    // Execution completed
    ws.onExecutionCompleted((payload) => {
      setEvents((prev) => [
        ...prev,
        {
          type: 'execution_completed',
          status: payload.status,
          message: payload.message,
        },
      ]);
    });

    // Error
    ws.onError((payload) => {
      setEvents((prev) => [
        ...prev,
        {
          type: 'error',
          error: payload,
        },
      ]);
    });

    ws.connect();

    return () => {
      ws.disconnect();
      wsRef.current = null;
    };
  }, [projectId]);

  const clearEvents = useCallback(() => {
    setEvents([]);
  }, []);

  return {
    events,
    connectionState,
    isConnected: connectionState === 'connected',
    clearEvents,
  };
}

