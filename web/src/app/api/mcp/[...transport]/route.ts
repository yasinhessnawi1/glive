import { NextRequest, NextResponse } from 'next/server'

// Authentication middleware
function authenticate(req: NextRequest): boolean {
  if (!process.env.MCP_AUTH_TOKEN) {
    return true // Allow access if no auth token is configured
  }

  const authHeader = req.headers.get('authorization')
  const token = authHeader?.replace('Bearer ', '')

  return token === process.env.MCP_AUTH_TOKEN
}

// Tool definitions
const tools = [
  {
    name: 'calculate',
    description: 'Perform basic mathematical calculations',
    inputSchema: {
      type: 'object',
      properties: {
        operation: {
          type: 'string',
          enum: ['add', 'subtract', 'multiply', 'divide', 'power', 'sqrt'],
          description: 'The mathematical operation to perform',
        },
        a: { type: 'number', description: 'First operand' },
        b: { type: 'number', description: 'Second operand (not required for sqrt)' },
      },
      required: ['operation', 'a'],
    },
  },
  {
    name: 'format_text',
    description: 'Format text with various transformations',
    inputSchema: {
      type: 'object',
      properties: {
        text: { type: 'string', description: 'The text to format' },
        format: {
          type: 'string',
          enum: ['uppercase', 'lowercase', 'capitalize', 'reverse', 'snake_case', 'camelCase'],
          description: 'The formatting operation to apply',
        },
      },
      required: ['text', 'format'],
    },
  },
  {
    name: 'generate_uuid',
    description: 'Generate a UUID v4',
    inputSchema: {
      type: 'object',
      properties: {
        count: { type: 'number', description: 'Number of UUIDs to generate (default: 1)' },
      },
      required: [],
    },
  },
  {
    name: 'get_timestamp',
    description: 'Get current timestamp in various formats',
    inputSchema: {
      type: 'object',
      properties: {
        format: {
          type: 'string',
          enum: ['iso', 'unix', 'readable', 'utc'],
          description: 'The timestamp format to return',
        },
      },
      required: ['format'],
    },
  },
]

// Tool execution handlers
function executeCalculate(args: { operation: string; a: number; b?: number }): string {
  const { operation, a, b } = args
  let result: number

  switch (operation) {
    case 'add':
      if (b === undefined) throw new Error('Second operand required')
      result = a + b
      break
    case 'subtract':
      if (b === undefined) throw new Error('Second operand required')
      result = a - b
      break
    case 'multiply':
      if (b === undefined) throw new Error('Second operand required')
      result = a * b
      break
    case 'divide':
      if (b === undefined || b === 0) throw new Error('Invalid second operand')
      result = a / b
      break
    case 'power':
      if (b === undefined) throw new Error('Second operand required')
      result = Math.pow(a, b)
      break
    case 'sqrt':
      if (a < 0) throw new Error('Cannot sqrt negative number')
      result = Math.sqrt(a)
      break
    default:
      throw new Error(`Unknown operation: ${operation}`)
  }

  return `Result: ${result}`
}

function executeFormatText(args: { text: string; format: string }): string {
  const { text, format } = args
  let result: string

  switch (format) {
    case 'uppercase':
      result = text.toUpperCase()
      break
    case 'lowercase':
      result = text.toLowerCase()
      break
    case 'capitalize':
      result = text.replace(/\b\w/g, (char) => char.toUpperCase())
      break
    case 'reverse':
      result = text.split('').reverse().join('')
      break
    case 'snake_case':
      result = text.replace(/\s+/g, '_').toLowerCase()
      break
    case 'camelCase':
      result = text.replace(/(?:^\w|[A-Z]|\b\w)/g, (word, index) =>
        index === 0 ? word.toLowerCase() : word.toUpperCase()
      ).replace(/\s+/g, '')
      break
    default:
      throw new Error(`Unknown format: ${format}`)
  }

  return `Formatted: ${result}`
}

function executeGenerateUuid(args: { count?: number }): string {
  const count = Math.min(Math.max(args.count || 1, 1), 10)
  const uuids: string[] = []

  for (let i = 0; i < count; i++) {
    uuids.push(crypto.randomUUID())
  }

  return count === 1 ? `UUID: ${uuids[0]}` : `UUIDs:\n${uuids.join('\n')}`
}

function executeGetTimestamp(args: { format: string; timezone?: string }): string {
  const { format, timezone } = args
  const now = new Date()
  let result: string

  switch (format) {
    case 'iso':
      result = now.toISOString()
      break
    case 'unix':
      result = Math.floor(now.getTime() / 1000).toString()
      break
    case 'readable':
      result = timezone
        ? now.toLocaleString('en-US', { timeZone: timezone })
        : now.toLocaleString()
      break
    case 'utc':
      result = now.toUTCString()
      break
    default:
      throw new Error(`Unknown format: ${format}`)
  }

  return `Timestamp: ${result}`
}

// Handle tool calls
function handleToolCall(name: string, args: Record<string, unknown>) {
  try {
    switch (name) {
      case 'calculate':
        return { content: [{ type: 'text', text: executeCalculate(args as any) }] }
      case 'format_text':
        return { content: [{ type: 'text', text: executeFormatText(args as any) }] }
      case 'generate_uuid':
        return { content: [{ type: 'text', text: executeGenerateUuid(args as any) }] }
      case 'get_timestamp':
        return { content: [{ type: 'text', text: executeGetTimestamp(args as any) }] }
      default:
        return { content: [{ type: 'text', text: `Unknown tool: ${name}` }], isError: true }
    }
  } catch (error: any) {
    return { content: [{ type: 'text', text: `Error: ${error.message}` }], isError: true }
  }
}

// MCP JSON-RPC handler
async function handleMcpRequest(body: any) {
  const { method, params, id } = body

  switch (method) {
    case 'initialize':
      return {
        jsonrpc: '2.0',
        id,
        result: {
          protocolVersion: '2024-11-05',
          capabilities: { tools: {} },
          serverInfo: { name: 'roomicor-mcp-server', version: '1.0.0' },
        },
      }

    case 'tools/list':
      return { jsonrpc: '2.0', id, result: { tools } }

    case 'tools/call':
      const { name, arguments: args } = params || {}
      const result = handleToolCall(name, args || {})
      return { jsonrpc: '2.0', id, result }

    default:
      return {
        jsonrpc: '2.0',
        id,
        error: { code: -32601, message: `Method not found: ${method}` },
      }
  }
}

export async function GET(req: NextRequest) {
  if (!authenticate(req)) {
    return NextResponse.json({ error: 'Unauthorized' }, { status: 401 })
  }

  return NextResponse.json({
    name: 'roomicor-mcp-server',
    version: '1.0.0',
    capabilities: { tools: {} },
    tools: tools.map(t => t.name),
  })
}

export async function POST(req: NextRequest) {
  if (!authenticate(req)) {
    return NextResponse.json({ error: 'Unauthorized' }, { status: 401 })
  }

  try {
    const body = await req.json()
    const response = await handleMcpRequest(body)
    return NextResponse.json(response)
  } catch (error: any) {
    return NextResponse.json(
      { jsonrpc: '2.0', error: { code: -32700, message: 'Parse error' } },
      { status: 400 }
    )
  }
}
