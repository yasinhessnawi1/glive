'use client'

import Link from 'next/link'

export default function Header() {
  return (
    <header className="bg-white dark:bg-slate-800 shadow-sm">
      <nav className="container mx-auto px-4 py-4 flex items-center justify-between">
        <Link href="/" className="text-2xl font-bold text-blue-600">
          GLive
        </Link>

        <div className="flex gap-4">
          <Link
            href="/settings"
            className="px-4 py-2 text-gray-600 dark:text-gray-300 hover:text-blue-600 transition-colors"
          >
            Settings
          </Link>
          <a
            href="https://github.com/yourusername/glive"
            target="_blank"
            rel="noopener noreferrer"
            className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
          >
            GitHub
          </a>
        </div>
      </nav>
    </header>
  )
}
