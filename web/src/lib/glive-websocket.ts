// GLive WebSocket Client
import type {
  WSMessage,
  WSMessageType,
  ProjectStatusPayload,
  CommandOutputPayload,
  CommandCompletePayload,
  CommandStartedPayload,
  RecoveryTriggeredPayload,
  RecoveryPlanPayload,
  RecoveryStepPayload,
  ExecutionCompletedPayload,
  ErrorPayload,
  ConnectionMetrics,
  ConnectionError,
} from '@/types/glive';

// Derive WebSocket URL from API URL (convert http -> ws, https -> wss)
const getDefaultWsUrl = () => {
  const apiUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';
  return apiUrl.replace(/^http/, 'ws');
};

const DEFAULT_WS_URL = process.env.NEXT_PUBLIC_GLIVE_WS_URL || getDefaultWsUrl();

// Connection configuration
const CONNECTION_TIMEOUT = 30000; // 30 seconds
const HEARTBEAT_INTERVAL = 25000; // 25 seconds (less than server's 30s ping)
const HEARTBEAT_TIMEOUT = 60000; // 60 seconds timeout (matching server)
const MAX_RECONNECT_ATTEMPTS = 20; // More attempts
const BASE_RECONNECT_DELAY = 500; // 500ms (faster initial reconnect)
const MAX_RECONNECT_DELAY = 10000; // 10 seconds max

export type WSMessageHandler = (message: WSMessage) => void;
export type ConnectionState = 'connecting' | 'connected' | 'disconnected' | 'error';
export type MetricsHandler = (metrics: ConnectionMetrics) => void;

export class GliveWebSocket {
  private ws: WebSocket | null = null;
  private url: string;
  private projectId: string;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = MAX_RECONNECT_ATTEMPTS;
  private handlers: Map<WSMessageType | '*', WSMessageHandler[]> = new Map();
  private connectionStateHandlers: ((state: ConnectionState) => void)[] = [];
  private metricsHandlers: MetricsHandler[] = [];
  private shouldReconnect = true;

  // Connection management
  private connectionTimeout: NodeJS.Timeout | null = null;
  private heartbeatInterval: NodeJS.Timeout | null = null;
  private lastPongTime: number = 0;

  // Metrics
  private metrics: ConnectionMetrics = {
    reconnectAttempts: 0,
    messageCount: 0,
  };

  constructor(projectId: string, baseURL: string = DEFAULT_WS_URL) {
    this.projectId = projectId;
    // Convert http:// to ws:// and https:// to wss://
    const wsBase = baseURL.replace(/^http/, 'ws');
    this.url = `${wsBase}/api/v1/ws/${projectId}`;
  }

  connect(): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      return;
    }

    this.clearTimers();
    this.updateConnectionState('connecting');

    try {
      this.ws = new WebSocket(this.url);

      // Set connection timeout
      this.connectionTimeout = setTimeout(() => {
        if (this.ws?.readyState !== WebSocket.OPEN) {
          console.error('WebSocket connection timeout');
          this.handleConnectionError({
            code: 'CONNECTION_TIMEOUT',
            message: 'Connection timeout after 30 seconds',
            timestamp: Date.now(),
          });
          this.ws?.close();
        }
      }, CONNECTION_TIMEOUT);

      this.ws.onopen = () => {
        this.clearConnectionTimeout();
        this.reconnectAttempts = 0;
        this.metrics.connectedAt = Date.now();
        this.metrics.reconnectAttempts = 0;
        this.updateConnectionState('connected');
        this.startHeartbeat();
        this.updateMetrics();
        console.log('WebSocket connected:', this.url);
      };

      this.ws.onmessage = (event) => {
        try {
          const message = JSON.parse(event.data) as WSMessage;

          // Handle pong messages for heartbeat
          if (message.type === 'pong' as WSMessageType) {
            this.lastPongTime = Date.now();
            if (this.metrics.connectedAt) {
              this.metrics.latency = this.lastPongTime - this.metrics.connectedAt;
            }
            return;
          }

          this.metrics.messageCount++;
          this.updateMetrics();
          this.handleMessage(message);
        } catch (error) {
          console.error('Failed to parse WebSocket message:', error, event.data);
          this.handleConnectionError({
            code: 'PARSE_ERROR',
            message: 'Failed to parse WebSocket message',
            timestamp: Date.now(),
            details: { error: String(error), data: event.data },
          });
        }
      };

      this.ws.onerror = (error) => {
        console.error('WebSocket error:', error);
        console.error('WebSocket URL:', this.url);
        console.error('WebSocket readyState:', this.ws?.readyState);

        this.handleConnectionError({
          code: 'WEBSOCKET_ERROR',
          message: 'WebSocket connection error',
          timestamp: Date.now(),
          details: { readyState: this.ws?.readyState },
        });

        this.updateConnectionState('error');
      };

      this.ws.onclose = (event) => {
        console.log('WebSocket closed:', event.code, event.reason);
        this.clearTimers();
        this.updateConnectionState('disconnected');

        if (this.shouldReconnect && this.reconnectAttempts < this.maxReconnectAttempts) {
          this.reconnectAttempts++;
          this.metrics.reconnectAttempts = this.reconnectAttempts;
          this.updateMetrics();

          // Exponential backoff with jitter
          const delay = Math.min(
            BASE_RECONNECT_DELAY * Math.pow(2, this.reconnectAttempts - 1) + Math.random() * 1000,
            MAX_RECONNECT_DELAY
          );

          console.log(`Reconnecting in ${Math.round(delay)}ms (attempt ${this.reconnectAttempts}/${this.maxReconnectAttempts})`);

          setTimeout(() => {
            this.connect();
          }, delay);
        } else if (this.reconnectAttempts >= this.maxReconnectAttempts) {
          this.handleConnectionError({
            code: 'MAX_RECONNECT_ATTEMPTS',
            message: `Failed to reconnect after ${this.maxReconnectAttempts} attempts`,
            timestamp: Date.now(),
          });
        }
      };
    } catch (error) {
      console.error('Failed to create WebSocket connection:', error);
      this.handleConnectionError({
        code: 'CONNECTION_FAILED',
        message: 'Failed to create WebSocket connection',
        timestamp: Date.now(),
        details: { error: String(error) },
      });
      this.updateConnectionState('error');
    }
  }

  disconnect(): void {
    this.shouldReconnect = false;
    this.clearTimers();
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
    this.metrics = {
      reconnectAttempts: 0,
      messageCount: 0,
    };
    this.updateMetrics();
  }

  /**
   * Manually trigger reconnection
   */
  reconnect(): void {
    this.disconnect();
    this.shouldReconnect = true;
    this.reconnectAttempts = 0;
    this.connect();
  }

  /**
   * Clear all timers
   */
  private clearTimers(): void {
    this.clearConnectionTimeout();
    this.stopHeartbeat();
  }

  /**
   * Clear connection timeout
   */
  private clearConnectionTimeout(): void {
    if (this.connectionTimeout) {
      clearTimeout(this.connectionTimeout);
      this.connectionTimeout = null;
    }
  }

  /**
   * Start heartbeat mechanism
   */
  private startHeartbeat(): void {
    this.stopHeartbeat();
    this.lastPongTime = Date.now();

    this.heartbeatInterval = setInterval(() => {
      if (this.ws?.readyState === WebSocket.OPEN) {
        // Send ping
        try {
          this.ws.send(JSON.stringify({ type: 'ping' }));

          // Check if we received pong recently (use longer timeout)
          const timeSinceLastPong = Date.now() - this.lastPongTime;
          if (timeSinceLastPong > HEARTBEAT_TIMEOUT) {
            console.warn('Heartbeat timeout, reconnecting...');
            this.handleConnectionError({
              code: 'HEARTBEAT_TIMEOUT',
              message: 'No heartbeat response from server',
              timestamp: Date.now(),
            });
            this.ws.close();
          }
        } catch (error) {
          console.error('Failed to send heartbeat:', error);
        }
      }
    }, HEARTBEAT_INTERVAL);
  }

  /**
   * Stop heartbeat mechanism
   */
  private stopHeartbeat(): void {
    if (this.heartbeatInterval) {
      clearInterval(this.heartbeatInterval);
      this.heartbeatInterval = null;
    }
  }

  /**
   * Handle connection error
   */
  private handleConnectionError(error: ConnectionError): void {
    this.metrics.lastError = error;
    this.updateMetrics();
  }

  /**
   * Update metrics and notify handlers
   */
  private updateMetrics(): void {
    this.metricsHandlers.forEach((handler) => handler({ ...this.metrics }));
  }

  private handleMessage(message: WSMessage): void {
    const handlers = this.handlers.get(message.type);
    if (handlers) {
      handlers.forEach((handler) => handler(message));
    }

    // Also call general handlers
    const allHandlers = this.handlers.get('*' as WSMessageType);
    if (allHandlers) {
      allHandlers.forEach((handler) => handler(message));
    }
  }

  on(type: WSMessageType | '*', handler: WSMessageHandler): void {
    if (!this.handlers.has(type)) {
      this.handlers.set(type, []);
    }
    this.handlers.get(type)!.push(handler);
  }

  off(type: WSMessageType | '*', handler: WSMessageHandler): void {
    const handlers = this.handlers.get(type);
    if (handlers) {
      const index = handlers.indexOf(handler);
      if (index > -1) {
        handlers.splice(index, 1);
      }
    }
  }

  onConnectionStateChange(handler: (state: ConnectionState) => void): void {
    this.connectionStateHandlers.push(handler);
  }

  /**
   * Register metrics change handler
   */
  onMetricsChange(handler: MetricsHandler): void {
    this.metricsHandlers.push(handler);
  }

  /**
   * Get current connection metrics
   */
  getMetrics(): ConnectionMetrics {
    return { ...this.metrics };
  }

  private updateConnectionState(state: ConnectionState): void {
    this.connectionStateHandlers.forEach((handler) => handler(state));
  }

  getState(): ConnectionState {
    if (!this.ws) return 'disconnected';
    switch (this.ws.readyState) {
      case WebSocket.CONNECTING:
        return 'connecting';
      case WebSocket.OPEN:
        return 'connected';
      case WebSocket.CLOSING:
      case WebSocket.CLOSED:
        return 'disconnected';
      default:
        return 'disconnected';
    }
  }

  isConnected(): boolean {
    return this.ws?.readyState === WebSocket.OPEN;
  }

  // Type-safe message handlers
  onProjectStatus(handler: (payload: ProjectStatusPayload) => void): void {
    this.on('project.status', (message) => {
      handler(message.payload as ProjectStatusPayload);
    });
  }

  onCommandOutput(handler: (payload: CommandOutputPayload) => void): void {
    this.on('command.output', (message) => {
      handler(message.payload as CommandOutputPayload);
    });
  }

  onCommandComplete(handler: (payload: CommandCompletePayload) => void): void {
    this.on('command.complete', (message) => {
      handler(message.payload as CommandCompletePayload);
    });
  }

  onCommandStarted(handler: (payload: CommandStartedPayload) => void): void {
    this.on('command.started', (message) => {
      handler(message.payload as CommandStartedPayload);
    });
  }

  onRecoveryTriggered(handler: (payload: RecoveryTriggeredPayload) => void): void {
    this.on('recovery.triggered', (message) => {
      handler(message.payload as RecoveryTriggeredPayload);
    });
  }

  onRecoveryPlan(handler: (payload: RecoveryPlanPayload) => void): void {
    this.on('recovery.plan', (message) => {
      handler(message.payload as RecoveryPlanPayload);
    });
  }

  onRecoveryStep(handler: (payload: RecoveryStepPayload) => void): void {
    this.on('recovery.step', (message) => {
      handler(message.payload as RecoveryStepPayload);
    });
  }

  onExecutionCompleted(handler: (payload: ExecutionCompletedPayload) => void): void {
    this.on('execution.completed', (message) => {
      handler(message.payload as ExecutionCompletedPayload);
    });
  }

  onError(handler: (payload: ErrorPayload) => void): void {
    this.on('error', (message) => {
      handler(message.payload as ErrorPayload);
    });
  }
}

