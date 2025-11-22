'use client'

import { useState } from 'react'

interface Project {
  id: string
  github_url: string
  status: string
  name?: string
}

interface ProjectCardProps {
  project: Project
  onUpdate: () => void
}

export default function ProjectCard({ project, onUpdate }: ProjectCardProps) {
  const [loading, setLoading] = useState(false)

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'ready':
      case 'running':
        return 'bg-green-100 text-green-800 dark:bg-green-900/20 dark:text-green-300'
      case 'pending':
      case 'analyzing':
        return 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/20 dark:text-yellow-300'
      case 'failed':
        return 'bg-red-100 text-red-800 dark:bg-red-900/20 dark:text-red-300'
      default:
        return 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-300'
    }
  }

  const getRepoName = (url: string) => {
    try {
      const parts = url.split('/')
      return parts[parts.length - 1] || url
    } catch {
      return url
    }
  }

  const handleDelete = async () => {
    if (!confirm('Are you sure you want to delete this project?')) {
      return
    }

    setLoading(true)
    try {
      const apiUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'
      await fetch(`${apiUrl}/api/v1/projects/${project.id}`, {
        method: 'DELETE',
      })
      onUpdate()
    } catch (err) {
      console.error('Failed to delete project:', err)
    } finally {
      setLoading(false)
    }
  }

  const handleCleanup = async () => {
    setLoading(true)
    try {
      const apiUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'
      await fetch(`${apiUrl}/api/v1/projects/${project.id}/cleanup`, {
        method: 'POST',
      })
      onUpdate()
    } catch (err) {
      console.error('Failed to cleanup project:', err)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="bg-white dark:bg-slate-800 rounded-lg shadow-md hover:shadow-lg transition-shadow p-6">
      <div className="flex items-start justify-between mb-4">
        <div className="flex-1">
          <h3 className="font-semibold text-lg text-gray-800 dark:text-gray-200 mb-1">
            {project.name || getRepoName(project.github_url)}
          </h3>
          <p className="text-sm text-gray-500 dark:text-gray-400 truncate">
            {project.github_url}
          </p>
        </div>
        <span className={`px-3 py-1 rounded-full text-xs font-semibold ${getStatusColor(project.status)}`}>
          {project.status}
        </span>
      </div>

      <div className="flex gap-2">
        <button
          onClick={() => window.open(project.github_url, '_blank')}
          className="flex-1 px-3 py-2 bg-blue-50 dark:bg-blue-900/20 text-blue-600 dark:text-blue-400 rounded-lg hover:bg-blue-100 dark:hover:bg-blue-900/30 transition-colors text-sm font-medium"
        >
          View Repo
        </button>
        <button
          onClick={handleCleanup}
          disabled={loading}
          className="px-3 py-2 bg-yellow-50 dark:bg-yellow-900/20 text-yellow-600 dark:text-yellow-400 rounded-lg hover:bg-yellow-100 dark:hover:bg-yellow-900/30 transition-colors text-sm font-medium disabled:opacity-50"
        >
          Cleanup
        </button>
        <button
          onClick={handleDelete}
          disabled={loading}
          className="px-3 py-2 bg-red-50 dark:bg-red-900/20 text-red-600 dark:text-red-400 rounded-lg hover:bg-red-100 dark:hover:bg-red-900/30 transition-colors text-sm font-medium disabled:opacity-50"
        >
          Delete
        </button>
      </div>
    </div>
  )
}
