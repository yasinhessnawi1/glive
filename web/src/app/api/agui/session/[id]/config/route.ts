import { NextRequest, NextResponse } from 'next/server'
import { auth } from '@clerk/nextjs/server'
import { z } from 'zod'
import { sessions, UpdateConfigSchema } from '@/lib/agui-sessions'

// PATCH /api/agui/session/[id]/config - Update session configuration
export async function PATCH(request: NextRequest, { params }: { params: Promise<{ id: string }> }) {
    try {
        const { userId } = await auth()
        const { id: sessionId } = await params
        const body = await request.json()

        const updates = UpdateConfigSchema.parse(body)

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

        // Update configuration
        session.configuration = {
            ...session.configuration,
            ...updates,
            safety: {
                ...session.configuration.safety,
                ...updates.safety,
            },
        }
        session.lastActivity = new Date()

        sessions.set(sessionId, session)

        return NextResponse.json({
            id: sessionId,
            configuration: session.configuration,
            lastActivity: session.lastActivity,
        })

    } catch (error: any) {
        console.error('Failed to update session configuration:', error)

        if (error instanceof z.ZodError) {
            return NextResponse.json({
                error: 'Invalid configuration format',
                details: error.errors,
            }, { status: 400 })
        }

        return NextResponse.json({
            error: 'Failed to update configuration',
            message: error.message,
        }, { status: 500 })
    }
}
