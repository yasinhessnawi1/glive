/**
 * AG-UI Session Management API
 * Creates and manages AG-UI sessions with streaming capabilities
 */

import { NextRequest, NextResponse } from 'next/server'
import { auth } from '@clerk/nextjs/server'
import { usageTracker } from '@/lib/token-counter'
import { z } from 'zod'
import {
  sessions,
  userSessions,
  defaultConfiguration,
  CreateSessionSchema,
  AGUISession
} from '@/lib/agui-sessions'

// POST /api/agui/session - Create new session
export async function POST(request: NextRequest) {
  try {
    const { userId } = await auth()
    const body = await request.json()

    const { userId: requestUserId, configuration } = CreateSessionSchema.parse(body)
    const finalUserId = userId || requestUserId

    // Generate session ID
    const sessionId = `agui-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`

    // Merge configuration with defaults
    const sessionConfig = {
      ...defaultConfiguration,
      ...configuration,
      safety: {
        ...defaultConfiguration.safety,
        ...configuration?.safety,
      },
    }

    // Create session
    const session: AGUISession = {
      id: sessionId,
      userId: finalUserId,
      status: 'active',
      startTime: new Date(),
      lastActivity: new Date(),
      configuration: sessionConfig,
      metadata: {
        userAgent: request.headers.get('user-agent'),
        ipAddress: request.headers.get('x-forwarded-for') || 'unknown',
        createdAt: new Date().toISOString(),
      },
      stats: {
        messageCount: 0,
        toolCallCount: 0,
        tokensUsed: 0,
        cost: 0,
      },
    }

    // Store session
    sessions.set(sessionId, session)

    if (finalUserId) {
      const existingSessions = userSessions.get(finalUserId) || []
      userSessions.set(finalUserId, [...existingSessions, sessionId])
    }

    // Track session creation
    usageTracker.track({
      inputTokens: 0,
      outputTokens: 0,
      totalTokens: 0,
      cost: 0,
      modelId: sessionConfig.model,
      operation: 'session_create',
      userId: finalUserId,
      sessionId,
    })

    return NextResponse.json({
      id: sessionId,
      status: session.status,
      configuration: sessionConfig,
      metadata: session.metadata,
    })

  } catch (error: any) {
    console.error('Failed to create AG-UI session:', error)

    if (error instanceof z.ZodError) {
      return NextResponse.json({
        error: 'Invalid request format',
        details: error.errors,
      }, { status: 400 })
    }

    return NextResponse.json({
      error: 'Failed to create session',
      message: error.message,
    }, { status: 500 })
  }
}

// GET /api/agui/session - List user sessions
export async function GET(request: NextRequest) {
  try {
    const { userId } = await auth()
    if (!userId) {
      return NextResponse.json({
        error: 'Authentication required',
      }, { status: 401 })
    }

    const userSessionIds = userSessions.get(userId) || []
    const userSessionData = userSessionIds
      .map(id => sessions.get(id))
      .filter(Boolean)
      .map(session => ({
        id: session!.id,
        status: session!.status,
        startTime: session!.startTime,
        lastActivity: session!.lastActivity,
        stats: session!.stats,
        configuration: {
          model: session!.configuration.model,
          streaming: session!.configuration.streaming,
        },
      }))

    return NextResponse.json({
      sessions: userSessionData,
      total: userSessionData.length,
    })

  } catch (error: any) {
    console.error('Failed to list sessions:', error)
    return NextResponse.json({
      error: 'Failed to list sessions',
      message: error.message,
    }, { status: 500 })
  }
}