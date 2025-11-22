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
} from '@/types/glive';

// Derive WebSocket URL from API URL (convert http -> ws, https -> wss)
const getDefaultWsUrl = () => {
  const apiUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';
  return apiUrl.replace(/^http/, 'ws');
};

const DEFAULT_WS_URL = process.env.NEXT_PUBLIC_GLIVE_WS_URL || getDefaultWsUrl();

export type WSMessageHandler = (message: WSMessage) => void;
export type ConnectionState = 'connecting' | 'connected' | 'disconnected' | 'error';

export class GliveWebSocket {
  private ws: WebSocket | null = null;
  private url: string;
  private projectId: string;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private reconnectDelay = 1000;
  private handlers: Map<WSMessageType | '*', WSMessageHandler[]> = new Map();
  private connectionStateHandlers: ((state: ConnectionState) => void)[] = [];
  private shouldReconnect = true;

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

    this.updateConnectionState('connecting');

    try {
      this.ws = new WebSocket(this.url);

      this.ws.onopen = () => {
        this.reconnectAttempts = 0;
        this.updateConnectionState('connected');
        console.log('WebSocket connected:', this.url);
      };

      this.ws.onmessage = (event) => {
        try {
          const message = JSON.parse(event.data) as WSMessage;
          this.handleMessage(message);
        } catch (error) {
          console.error('Failed to parse WebSocket message:', error, event.data);
        }
      };

      this.ws.onerror = (error) => {
        console.error('WebSocket error:', error);
        console.error('WebSocket URL:', this.url);
        console.error('WebSocket readyState:', this.ws?.readyState);
        this.updateConnectionState('error');
      };

      this.ws.onclose = (event) => {
        console.log('WebSocket closed:', event.code, event.reason);
        this.updateConnectionState('disconnected');
        if (this.shouldReconnect && this.reconnectAttempts < this.maxReconnectAttempts) {
          this.reconnectAttempts++;
          console.log(`Reconnecting in ${this.reconnectDelay * this.reconnectAttempts}ms (attempt ${this.reconnectAttempts})`);
          setTimeout(() => {
            this.connect();
          }, this.reconnectDelay * this.reconnectAttempts);
        }
      };
    } catch (error) {
      console.error('Failed to create WebSocket connection:', error);
      this.updateConnectionState('error');
    }
  }

  disconnect(): void {
    this.shouldReconnect = false;
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
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

