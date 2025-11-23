import { HttpAgent, EventType, RunAgentInput, BaseEvent } from '@ag-ui/client'

export interface AgentClientConfig {
  url?: string
  apiKey?: string
  headers?: Record<string, string>
}

export interface AgentMessage {
  id: string
  type: 'user' | 'agent' | 'system' | 'tool'
  content: string
  timestamp: Date
  metadata?: Record<string, any>
}

export interface ToolCall {
  id: string
  name: string
  params: any
  status: 'pending' | 'running' | 'completed' | 'failed'
  result?: any
  error?: string
}

export function createHttpAgent(config: AgentClientConfig = {}): HttpAgent {
  const defaultUrl = process.env.NEXT_PUBLIC_AGENT_URL || 'http://localhost:8000/agent'

  const client = new HttpAgent({
    url: config.url || defaultUrl,
    headers: {
      'Content-Type': 'application/json',
      ...(config.apiKey && {
        'Authorization': `Bearer ${config.apiKey}`,
      }),
      ...(process.env.AGENT_API_KEY && {
        'Authorization': `Bearer ${process.env.AGENT_API_KEY}`,
      }),
      ...config.headers,
    },
  })

  return client
}

export class AgentSession {
  private client: HttpAgent
  private messages: AgentMessage[] = []
  private toolCalls: ToolCall[] = []
  private listeners: Map<string, Function[]> = new Map()
  private sessionId: string
  private abortController: AbortController | null = null

  constructor(config: AgentClientConfig = {}) {
    this.client = createHttpAgent(config)
    this.sessionId = `session_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`
  }

  async runAgent(input: Partial<RunAgentInput>): Promise<void> {
    this.abortController = new AbortController()

    const fullInput: RunAgentInput = {
      threadId: this.sessionId,
      runId: `run_${Date.now()}`,
      messages: [],
      tools: [],
      context: [],
      ...input,
    }

    try {
      const result = await this.client.runAgent(fullInput)

      // Handle the result - it may contain events or a final response
      if (result && typeof result === 'object') {
        if ('events' in result && Array.isArray((result as any).events)) {
          for (const event of (result as any).events) {
            this.handleEvent(event)
          }
        }
      }

      this.emit('processingEnd')
    } catch (error: any) {
      if (error.name !== 'AbortError') {
        this.emit('error', { message: error.message || 'Unknown error' })
      }
    }
  }

  private handleEvent(event: BaseEvent): void {
    switch (event.type) {
      case EventType.TEXT_MESSAGE_START:
      case EventType.TEXT_MESSAGE_CONTENT:
      case EventType.TEXT_MESSAGE_END:
        this.handleTextMessage(event)
        break
      case EventType.TOOL_CALL_START:
      case EventType.TOOL_CALL_ARGS:
      case EventType.TOOL_CALL_END:
        this.handleToolCall(event)
        break
      case EventType.RUN_STARTED:
        this.emit('processingStart')
        break
      case EventType.RUN_FINISHED:
        this.emit('processingEnd')
        break
      case EventType.RUN_ERROR:
        this.emit('error', { message: (event as any).error || 'Run error' })
        break
      default:
        // Handle unknown event types
        break
    }
  }

  private handleTextMessage(event: BaseEvent): void {
    if (event.type === EventType.TEXT_MESSAGE_CONTENT) {
      const messageEvent = event as any
      const message: AgentMessage = {
        id: messageEvent.messageId || `msg_${Date.now()}`,
        type: 'agent',
        content: messageEvent.delta || '',
        timestamp: new Date(),
        metadata: {},
      }
      this.messages.push(message)
      this.emit('message', message)
    }
  }

  private handleToolCall(event: BaseEvent): void {
    const toolEvent = event as any

    if (event.type === EventType.TOOL_CALL_START) {
      const toolCall: ToolCall = {
        id: toolEvent.toolCallId || `tool_${Date.now()}`,
        name: toolEvent.toolCallName || '',
        params: {},
        status: 'running',
      }
      this.toolCalls.push(toolCall)
      this.emit('toolCall', toolCall)
    } else if (event.type === EventType.TOOL_CALL_END) {
      const existingCall = this.toolCalls.find(tc => tc.id === toolEvent.toolCallId)
      if (existingCall) {
        existingCall.status = 'completed'
        this.emit('toolResult', existingCall)
      }
    }
  }

  async sendMessage(content: string, metadata?: Record<string, any>): Promise<void> {
    const messageId = `msg_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`

    const message: AgentMessage = {
      id: messageId,
      type: 'user',
      content,
      timestamp: new Date(),
      metadata,
    }

    this.messages.push(message)
    this.emit('message', message)

    await this.runAgent({
      messages: [{ id: messageId, role: 'user', content }],
    })
  }

  stop(): void {
    if (this.abortController) {
      this.abortController.abort()
      this.abortController = null
    }
  }

  // Connect method for compatibility - ag-ui doesn't require explicit connection
  async connect(): Promise<void> {
    this.emit('connected')
  }

  // Disconnect method for compatibility
  async disconnect(): Promise<void> {
    this.stop()
    this.emit('disconnected')
  }

  // Call a tool - sends a message requesting tool execution
  async callTool(toolName: string, params: any): Promise<string> {
    const toolCallId = `tool_${Date.now()}_${Math.random().toString(36).substring(2, 11)}`

    const toolCall: ToolCall = {
      id: toolCallId,
      name: toolName,
      params,
      status: 'pending',
    }

    this.toolCalls.push(toolCall)
    this.emit('toolCall', toolCall)

    // Send a message to trigger tool execution
    await this.runAgent({
      messages: [{ id: `msg_${Date.now()}`, role: 'user', content: `Execute tool: ${toolName}` }],
      tools: [{ name: toolName, description: `Tool ${toolName}`, parameters: params }],
    })

    return toolCallId
  }

  // Request a completion
  async requestCompletion(prompt: string, options?: any): Promise<void> {
    await this.runAgent({
      messages: [{ id: `msg_${Date.now()}`, role: 'user', content: prompt }],
      ...options,
    })
  }

  on(event: string, callback: Function): void {
    if (!this.listeners.has(event)) {
      this.listeners.set(event, [])
    }
    this.listeners.get(event)!.push(callback)
  }

  off(event: string, callback: Function): void {
    const eventListeners = this.listeners.get(event)
    if (eventListeners) {
      const index = eventListeners.indexOf(callback)
      if (index > -1) {
        eventListeners.splice(index, 1)
      }
    }
  }

  private emit(event: string, ...args: any[]): void {
    const eventListeners = this.listeners.get(event)
    if (eventListeners) {
      eventListeners.forEach(callback => {
        try {
          callback(...args)
        } catch (error) {
          console.error(`Error in event listener for ${event}:`, error)
        }
      })
    }
  }

  // Getters
  get allMessages(): AgentMessage[] {
    return [...this.messages]
  }

  get allToolCalls(): ToolCall[] {
    return [...this.toolCalls]
  }

  get id(): string {
    return this.sessionId
  }

  // Message management
  clearMessages(): void {
    this.messages = []
    this.emit('messagesCleared')
  }

  getMessageById(id: string): AgentMessage | undefined {
    return this.messages.find(msg => msg.id === id)
  }

  removeMessage(id: string): boolean {
    const index = this.messages.findIndex(msg => msg.id === id)
    if (index > -1) {
      this.messages.splice(index, 1)
      this.emit('messageRemoved', id)
      return true
    }
    return false
  }

  // Tool call management
  getToolCallById(id: string): ToolCall | undefined {
    return this.toolCalls.find(tc => tc.id === id)
  }

  getToolCallsByName(name: string): ToolCall[] {
    return this.toolCalls.filter(tc => tc.name === name)
  }

  getPendingToolCalls(): ToolCall[] {
    return this.toolCalls.filter(tc => tc.status === 'pending' || tc.status === 'running')
  }
}

// Utility functions for common operations
export async function quickAgentMessage(
  message: string,
  config: AgentClientConfig = {}
): Promise<AgentMessage[]> {
  const session = new AgentSession(config)
  const responses: AgentMessage[] = []

  return new Promise((resolve, reject) => {
    const timeout = setTimeout(() => {
      session.stop()
      reject(new Error('Agent response timeout'))
    }, 30000)

    session.on('message', (msg: AgentMessage) => {
      if (msg.type === 'agent') {
        responses.push(msg)
      }
    })

    session.on('processingEnd', () => {
      clearTimeout(timeout)
      resolve(responses)
    })

    session.on('error', (error: any) => {
      clearTimeout(timeout)
      reject(error)
    })

    session.sendMessage(message).catch(reject)
  })
}

export async function quickToolCall(
  toolName: string,
  params: any,
  config: AgentClientConfig = {}
): Promise<ToolCall> {
  const session = new AgentSession(config)

  return new Promise((resolve, reject) => {
    const timeout = setTimeout(() => {
      session.stop()
      reject(new Error('Tool call timeout'))
    }, 30000)

    session.on('toolResult', (toolCall: ToolCall) => {
      if (toolCall.name === toolName) {
        clearTimeout(timeout)
        resolve(toolCall)
      }
    })

    session.on('error', (error: any) => {
      clearTimeout(timeout)
      reject(error)
    })

    // Send a message requesting the tool call
    session.sendMessage(`Call tool: ${toolName} with params: ${JSON.stringify(params)}`).catch(reject)
  })
}

// Event type re-exports for convenience
export { EventType } from '@ag-ui/client'
