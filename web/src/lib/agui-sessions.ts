import { z } from 'zod'

// Types for AG-UI sessions
export interface AGUISession {
    id: string
    userId?: string
    status: 'initializing' | 'active' | 'paused' | 'completed' | 'error'
    startTime: Date
    lastActivity: Date
    configuration: AGUIConfiguration
    metadata: Record<string, any>
    stats: {
        messageCount: number
        toolCallCount: number
        tokensUsed: number
        cost: number
    }
}

export interface AGUIConfiguration {
    model: string
    temperature: number
    maxTokens: number
    systemPrompt?: string
    tools: string[]
    capabilities: string[]
    streaming: boolean
    safety: {
        contentFiltering: boolean
        rateLimiting: boolean
        maxRequestsPerMinute: number
    }
}

// Validation schemas
export const CreateSessionSchema = z.object({
    userId: z.string().optional(),
    configuration: z.object({
        model: z.string().default('gpt-4o'),
        temperature: z.number().min(0).max(2).default(0.7),
        maxTokens: z.number().min(1).max(4000).default(2000),
        systemPrompt: z.string().optional(),
        tools: z.array(z.string()).default([]),
        capabilities: z.array(z.string()).default([]),
        streaming: z.boolean().default(true),
        safety: z.object({
            contentFiltering: z.boolean().default(true),
            rateLimiting: z.boolean().default(true),
            maxRequestsPerMinute: z.number().default(60),
        }).default({}),
    }).optional(),
})

export const UpdateConfigSchema = z.object({
    model: z.string().optional(),
    temperature: z.number().min(0).max(2).optional(),
    maxTokens: z.number().min(1).max(4000).optional(),
    systemPrompt: z.string().optional(),
    tools: z.array(z.string()).optional(),
    capabilities: z.array(z.string()).optional(),
    streaming: z.boolean().optional(),
    safety: z.object({
        contentFiltering: z.boolean().optional(),
        rateLimiting: z.boolean().optional(),
        maxRequestsPerMinute: z.number().optional(),
    }).optional(),
})

// In-memory session storage (in production, use Redis or database)
export const sessions = new Map<string, AGUISession>()
export const userSessions = new Map<string, string[]>() // userId -> sessionIds

// Default configuration
export const defaultConfiguration: AGUIConfiguration = {
    model: 'gpt-4o',
    temperature: 0.7,
    maxTokens: 2000,
    systemPrompt: 'You are an AI assistant for Roomicor. Help users with tasks, automation, and business operations.',
    tools: ['web_search', 'calculator', 'file_manager', 'database_query', 'email_sender'],
    capabilities: [
        'web_access',
        'file_operations',
        'email_integration',
        'database_access',
        'automation_tools',
    ],
    streaming: true,
    safety: {
        contentFiltering: true,
        rateLimiting: true,
        maxRequestsPerMinute: 60,
    },
}

// Export session getter for other modules
export function getSession(sessionId: string): AGUISession | undefined {
    return sessions.get(sessionId)
}

export function updateSessionStats(sessionId: string, stats: Partial<AGUISession['stats']>): void {
    const session = sessions.get(sessionId)
    if (session) {
        session.stats = { ...session.stats, ...stats }
        session.lastActivity = new Date()
        sessions.set(sessionId, session)
    }
}
