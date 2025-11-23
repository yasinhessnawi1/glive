import { NextRequest, NextResponse } from 'next/server'
import { auth } from '@clerk/nextjs/server'
import { sessions } from '@/lib/agui-sessions'

// POST /api/agui/session/[id]/pause - Pause session
export async function POST(request: NextRequest, { params }: { params: Promise<{ id: string }> }) {
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

        session.status = 'paused'
        session.lastActivity = new Date()
        sessions.set(sessionId, session)

        return NextResponse.json({
            id: sessionId,
            status: session.status,
            lastActivity: session.lastActivity,
        })

    } catch (error: any) {
        console.error('Failed to pause session:', error)
        return NextResponse.json({
            error: 'Failed to pause session',
            message: error.message,
        }, { status: 500 })
    }
}
