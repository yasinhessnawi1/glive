'use client'

import { useEffect, useState } from 'react'
import ProjectCard from './ProjectCard'

interface Project {
  id: string
  github_url: string
  status: string
  name?: string
}

export default function ProjectList() {
  const [projects, setProjects] = useState<Project[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    fetchProjects()
  }, [])

  const fetchProjects = async () => {
    try {
      const apiUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'
      const response = await fetch(`${apiUrl}/api/v1/projects`)
      if (response.ok) {
        const data = await response.json()
        setProjects(data.projects || [])
      } else {
        setError('Failed to load projects')
      }
    } catch (err) {
      setError('Cannot connect to GLive agent')
    } finally {
      setLoading(false)
    }
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 text-yellow-800 dark:text-yellow-300 px-4 py-3 rounded-lg">
        <p className="font-semibold">Connection Error</p>
        <p className="text-sm mt-1">{error}</p>
        <p className="text-sm mt-2">Make sure the GLive agent is running on port 8080.</p>
      </div>
    )
  }

  if (projects.length === 0) {
    return (
      <div className="text-center py-12 bg-white dark:bg-slate-800 rounded-lg">
        <div className="text-6xl mb-4">📦</div>
        <p className="text-gray-600 dark:text-gray-400">
          No projects yet. Enter a GitHub URL above to get started!
        </p>
      </div>
    )
  }

  return (
    <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-6">
      {projects.map((project) => (
        <ProjectCard key={project.id} project={project} onUpdate={fetchProjects} />
      ))}
    </div>
  )
}
