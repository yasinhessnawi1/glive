'use client'

import { useState } from 'react'

interface ProjectInputProps {
  onProjectCreated: () => void
}

export default function ProjectInput({ onProjectCreated }: ProjectInputProps) {
  const [url, setUrl] = useState('')
  const [mode, setMode] = useState<'auto' | 'assisted' | 'manual'>('auto')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')

    if (!url.trim()) {
      setError('Please enter a GitHub URL')
      return
    }

    setLoading(true)

    try {
      const apiUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'
      const response = await fetch(`${apiUrl}/api/v1/projects`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          github_url: url,
          mode: mode,
        }),
      })

      if (response.ok) {
        setUrl('')
        onProjectCreated()
      } else {
        const data = await response.json()
        setError(data.error || 'Failed to create project')
      }
    } catch (err) {
      setError('Cannot connect to GLive agent. Make sure it\'s running on port 8080.')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="bg-white dark:bg-slate-800 rounded-lg shadow-lg p-6">
      <form onSubmit={handleSubmit} className="space-y-4">
        <div>
          <label htmlFor="url" className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
            GitHub Repository URL
          </label>
          <input
            type="text"
            id="url"
            value={url}
            onChange={(e) => setUrl(e.target.value)}
            placeholder="https://github.com/user/repo or user/repo"
            className="w-full px-4 py-3 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-slate-700 dark:text-white"
            disabled={loading}
          />
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
            Execution Mode
          </label>
          <div className="grid grid-cols-3 gap-3">
            {(['auto', 'assisted', 'manual'] as const).map((m) => (
              <button
                key={m}
                type="button"
                onClick={() => setMode(m)}
                className={`px-4 py-2 rounded-lg border-2 transition-all ${mode === m
                    ? 'border-blue-600 bg-blue-50 dark:bg-blue-900/20 text-blue-700 dark:text-blue-300'
                    : 'border-gray-200 dark:border-gray-600 hover:border-gray-300 dark:hover:border-gray-500'
                  }`}
                disabled={loading}
              >
                <div className="font-semibold capitalize">{m}</div>
                <div className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                  {m === 'auto' && 'Fully automatic'}
                  {m === 'assisted' && 'Ask for approval'}
                  {m === 'manual' && 'Show instructions'}
                </div>
              </button>
            ))}
          </div>
        </div>

        {error && (
          <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 text-red-700 dark:text-red-300 px-4 py-3 rounded-lg">
            {error}
          </div>
        )}

        <button
          type="submit"
          disabled={loading}
          className="w-full bg-blue-600 text-white py-3 rounded-lg font-semibold hover:bg-blue-700 disabled:bg-gray-400 disabled:cursor-not-allowed transition-colors"
        >
          {loading ? 'Creating...' : 'Run Project'}
        </button>
      </form>
    </div>
  )
}
