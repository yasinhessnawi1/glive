import { NextResponse } from 'next/server'
import { sessions, userSessions } from '@/lib/agui-sessions'

// GET /api/agui/session/health - Health check
export async function GET() {
    return NextResponse.json({
        status: 'healthy',
        sessionCount: sessions.size,
        userCount: userSessions.size,
        availableModels: ['gpt-4o', 'gpt-4', 'gpt-3.5-turbo', 'claude-3-sonnet'],
        availableTools: [
            'web_search',
            'calculator',
            'file_manager',
            'database_query',
            'email_sender',
            'image_generator',
            'code_executor',
            'pdf_reader',
        ],
        timestamp: new Date().toISOString(),
    })
}
