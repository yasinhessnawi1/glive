/**
 * Terminal utility functions for log processing, formatting, and export
 */

import type { ExecutionEvent } from '@/types/glive';

/**
 * ANSI color codes mapping
 */
const ANSI_COLORS: Record<string, string> = {
    '30': 'text-gray-900',
    '31': 'text-red-500',
    '32': 'text-green-500',
    '33': 'text-yellow-500',
    '34': 'text-blue-500',
    '35': 'text-purple-500',
    '36': 'text-cyan-500',
    '37': 'text-gray-300',
    '90': 'text-gray-600',
    '91': 'text-red-400',
    '92': 'text-green-400',
    '93': 'text-yellow-400',
    '94': 'text-blue-400',
    '95': 'text-purple-400',
    '96': 'text-cyan-400',
    '97': 'text-white',
};

/**
 * Parse ANSI escape codes and convert to React elements with Tailwind classes
 */
export function parseAnsiCodes(text: string): { text: string; className?: string } {
    // Remove ANSI codes for now - can be enhanced later for full support
    const cleanText = text.replace(/\x1b\[[0-9;]*m/g, '');

    // Simple color detection - can be enhanced
    const colorMatch = text.match(/\x1b\[([0-9;]+)m/);
    if (colorMatch) {
        const code = colorMatch[1].split(';').pop() || '';
        return {
            text: cleanText,
            className: ANSI_COLORS[code],
        };
    }

    return { text: cleanText };
}

/**
 * Format timestamp to HH:MM:SS format
 */
export function formatTimestamp(timestamp?: number): string {
    const date = timestamp ? new Date(timestamp) : new Date();
    return date.toLocaleTimeString('en-US', {
        hour12: false,
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit',
    });
}

/**
 * Format duration in milliseconds to human-readable format
 */
export function formatDuration(ms?: number): string {
    if (!ms) return '0ms';

    if (ms < 1000) {
        return `${ms}ms`;
    }

    const seconds = Math.floor(ms / 1000);
    const minutes = Math.floor(seconds / 60);
    const hours = Math.floor(minutes / 60);

    if (hours > 0) {
        return `${hours}h ${minutes % 60}m ${seconds % 60}s`;
    }
    if (minutes > 0) {
        return `${minutes}m ${seconds % 60}s`;
    }
    return `${seconds}s`;
}

/**
 * Get event text representation for searching and export
 */
export function getEventText(event: ExecutionEvent): string {
    switch (event.type) {
        case 'command_started':
            return `▶ Starting: ${event.command}`;
        case 'output_line':
            return event.line;
        case 'command_completed':
            return `${event.success ? '✓' : '✗'} Command completed (exit code: ${event.exit_code})`;
        case 'recovery_triggered':
            return `🔄 AI Recovery triggered: ${event.reason}`;
        case 'recovery_plan':
            return `📋 Recovery plan: ${event.plan.analysis}`;
        case 'recovery_step':
            return `🔧 Recovery step: ${event.step.status}`;
        case 'execution_completed':
            return `${event.status === 'success' ? '✓' : '✗'} Execution ${event.status}${event.message ? ` - ${event.message}` : ''}`;
        case 'error':
            return `❌ Error: ${event.error.message}`;
        default:
            return '';
    }
}

/**
 * Get event severity level for filtering
 */
export type LogLevel = 'info' | 'warning' | 'error' | 'success';

export function getEventLevel(event: ExecutionEvent): LogLevel {
    switch (event.type) {
        case 'command_started':
            return 'info';
        case 'output_line':
            return event.stream === 'stderr' ? 'warning' : 'info';
        case 'command_completed':
            return event.success ? 'success' : 'error';
        case 'recovery_triggered':
            return 'warning';
        case 'recovery_plan':
        case 'recovery_step':
            return 'info';
        case 'execution_completed':
            return event.status === 'success' ? 'success' : 'error';
        case 'error':
            return 'error';
        default:
            return 'info';
    }
}

/**
 * Filter events by log level
 */
export function filterEventsByLevel(
    events: ExecutionEvent[],
    levels: LogLevel[]
): ExecutionEvent[] {
    if (levels.length === 0) return events;
    return events.filter((event) => levels.includes(getEventLevel(event)));
}

/**
 * Search events by text query
 */
export function searchEvents(
    events: ExecutionEvent[],
    query: string
): ExecutionEvent[] {
    if (!query.trim()) return events;

    const lowerQuery = query.toLowerCase();
    return events.filter((event) => {
        const text = getEventText(event).toLowerCase();
        return text.includes(lowerQuery);
    });
}

/**
 * Export events to text format
 */
export function exportEventsToText(events: ExecutionEvent[]): string {
    const lines = events.map((event, index) => {
        const timestamp = formatTimestamp(
            'timestamp' in event ? event.timestamp : undefined
        );
        const text = getEventText(event);
        return `[${timestamp}] ${text}`;
    });

    return lines.join('\n');
}

/**
 * Download text content as file
 */
export function downloadTextFile(content: string, filename: string): void {
    const blob = new Blob([content], { type: 'text/plain' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = filename;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
}

/**
 * Deduplicate consecutive identical events
 */
export function deduplicateEvents(events: ExecutionEvent[]): ExecutionEvent[] {
    if (events.length === 0) return events;

    const deduplicated: ExecutionEvent[] = [events[0]];

    for (let i = 1; i < events.length; i++) {
        const prev = events[i - 1];
        const curr = events[i];

        // Only deduplicate output_line events with identical content
        if (
            prev.type === 'output_line' &&
            curr.type === 'output_line' &&
            prev.line === curr.line &&
            prev.stream === curr.stream
        ) {
            continue;
        }

        deduplicated.push(curr);
    }

    return deduplicated;
}

/**
 * Validate event payload has required fields
 */
export function isValidEvent(event: unknown): event is ExecutionEvent {
    if (!event || typeof event !== 'object') return false;

    const e = event as Record<string, unknown>;

    if (!e.type || typeof e.type !== 'string') return false;

    // Validate based on event type
    switch (e.type) {
        case 'command_started':
            return typeof e.command === 'string' && typeof e.command_id === 'string';
        case 'output_line':
            return typeof e.line === 'string' && (e.stream === 'stdout' || e.stream === 'stderr');
        case 'command_completed':
            return typeof e.exit_code === 'number' && typeof e.success === 'boolean';
        case 'recovery_triggered':
            return typeof e.reason === 'string';
        case 'recovery_plan':
            return e.plan !== null && typeof e.plan === 'object';
        case 'recovery_step':
            return e.step !== null && typeof e.step === 'object';
        case 'execution_completed':
            return e.status === 'success' || e.status === 'failed';
        case 'error':
            return e.error !== null && typeof e.error === 'object';
        default:
            return false;
    }
}

/**
 * Calculate event statistics
 */
export interface EventStatistics {
    totalEvents: number;
    totalCommands: number;
    successfulCommands: number;
    failedCommands: number;
    totalOutputLines: number;
    errorCount: number;
    recoveryCount: number;
    averageDuration?: number;
}

export function calculateStatistics(events: ExecutionEvent[]): EventStatistics {
    const stats: EventStatistics = {
        totalEvents: events.length,
        totalCommands: 0,
        successfulCommands: 0,
        failedCommands: 0,
        totalOutputLines: 0,
        errorCount: 0,
        recoveryCount: 0,
    };

    const durations: number[] = [];

    events.forEach((event) => {
        switch (event.type) {
            case 'command_started':
                stats.totalCommands++;
                break;
            case 'command_completed':
                if (event.success) {
                    stats.successfulCommands++;
                } else {
                    stats.failedCommands++;
                }
                if (event.duration) {
                    durations.push(event.duration);
                }
                break;
            case 'output_line':
                stats.totalOutputLines++;
                break;
            case 'error':
                stats.errorCount++;
                break;
            case 'recovery_triggered':
                stats.recoveryCount++;
                break;
        }
    });

    if (durations.length > 0) {
        stats.averageDuration = durations.reduce((a, b) => a + b, 0) / durations.length;
    }

    return stats;
}

/**
 * Copy text to clipboard
 */
export async function copyToClipboard(text: string): Promise<boolean> {
    try {
        await navigator.clipboard.writeText(text);
        return true;
    } catch (error) {
        console.error('Failed to copy to clipboard:', error);
        return false;
    }
}
