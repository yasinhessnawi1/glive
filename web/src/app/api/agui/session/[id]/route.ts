import { NextRequest, NextResponse } from 'next/server'
import { auth } from '@clerk/nextjs/server'
import { usageTracker } from '@/lib/token-counter'
import { sessions, userSessions } from '@/lib/agui-sessions'

// GET /api/agui/session/[id] - Get specific session
export async function GET(request: NextRequest, { params }: { params: Promise<{ id: string }> }) {
    try {
        const { userId } = await auth()
        const { id: sessionId } = await params

        const session = sessions.get(sessionId)
        if (!session) {
            return NextResponse.json({
                error: 'Session not found',
            }, { status: 404 })
        }

        // Check ownership
        if (userId && session.userId !== userId) {
            return NextResponse.json({
                error: 'Access denied',
            }, { status: 403 })
        }

        return NextResponse.json({
            id: session.id,
            status: session.status,
            startTime: session.startTime,
            lastActivity: session.lastActivity,
            configuration: session.configuration,
            metadata: session.metadata,
            stats: session.stats,
        })

    } catch (error: any) {
        console.error('Failed to get session:', error)
        return NextResponse.json({
            error: 'Failed to get session',
            message: error.message,
        }, { status: 500 })
    }
}

// DELETE /api/agui/session/[id] - Delete session
export async function DELETE(request: NextRequest, { params }: { params: Promise<{ id: string }> }) {
    try {
        const { userId } = await auth()
        const { id: sessionId } = await params

        const session = sessions.get(sessionId)
        if (!session) {
            return NextResponse.json({
                error: 'Session not found',
            }, { status: 404 })
        }

        // Check ownership
        if (userId && session.userId !== userId) {
            return NextResponse.json({
                error: 'Access denied',
            }, { status: 403 })
        }

        // Remove from storage
        sessions.delete(sessionId)

        if (session.userId) {
            const userSessionIds = userSessions.get(session.userId) || []
            userSessions.set(
                session.userId,
                userSessionIds.filter(id => id !== sessionId)
            )
        }

        // Track session deletion
        usageTracker.track({
            inputTokens: 0,
            outputTokens: 0,
            totalTokens: 0,
            cost: 0,
            modelId: session.configuration.model,
            operation: 'session_delete',
            userId: session.userId,
            sessionId,
        })

        return NextResponse.json({
            id: sessionId,
            deleted: true,
        })

    } catch (error: any) {
        console.error('Failed to delete session:', error)
        return NextResponse.json({
            error: 'Failed to delete session',
            message: error.message,
        }, { status: 500 })
    }
}
