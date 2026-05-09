'use client'

import { useState } from 'react'
import { useRouter } from 'next/navigation'
import { createApp } from '@/app/lib/api'

export default function CreateAppForm() {
  const [open, setOpen] = useState(false)
  const [name, setName] = useState('')
  const [gitUrl, setGitUrl] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const router = useRouter()

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setLoading(true)
    setError('')
    try {
      const data = await createApp(name, gitUrl)
      if (data.app) {
        setOpen(false)
        setName('')
        setGitUrl('')
        router.refresh()
      } else {
        setError(data.error ?? 'Error creando app')
      }
    } catch {
      setError('Error de conexión con la API')
    } finally {
      setLoading(false)
    }
  }

  if (!open) {
    return (
      <button
        onClick={() => setOpen(true)}
        className="px-4 py-2 text-sm font-mono border border-[var(--border)] text-[var(--text)] rounded hover:border-[var(--accent)] hover:text-[var(--accent)] transition-colors"
      >
        + new app
      </button>
    )
  }

  return (
    <div className="fixed inset-0 bg-black/70 flex items-center justify-center z-50">
      <div className="bg-[var(--surface)] border border-[var(--border)] rounded-lg p-6 w-full max-w-md">
        <h2 className="text-white font-medium mb-6">new app</h2>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="block text-xs text-[var(--muted)] mb-1">app name</label>
            <input
              type="text"
              value={name}
              onChange={e => setName(e.target.value)}
              placeholder="my-app"
              required
              className="w-full bg-[var(--bg)] border border-[var(--border)] rounded px-3 py-2 text-sm text-white font-mono focus:outline-none focus:border-[var(--accent)]"
            />
          </div>
          <div>
            <label className="block text-xs text-[var(--muted)] mb-1">git url</label>
            <input
              type="text"
              value={gitUrl}
              onChange={e => setGitUrl(e.target.value)}
              placeholder="https://github.com/user/repo"
              required
              className="w-full bg-[var(--bg)] border border-[var(--border)] rounded px-3 py-2 text-sm text-white font-mono focus:outline-none focus:border-[var(--accent)]"
            />
          </div>
          {error && <p className="text-[var(--red)] text-xs">{error}</p>}
          <div className="flex gap-3 pt-2">
            <button
              type="submit"
              disabled={loading}
              className="flex-1 py-2 text-sm bg-[var(--accent)] text-white rounded hover:bg-blue-400 disabled:opacity-50 transition-colors"
            >
              {loading ? 'creating...' : 'create'}
            </button>
            <button
              type="button"
              onClick={() => setOpen(false)}
              className="px-4 py-2 text-sm border border-[var(--border)] text-[var(--muted)] rounded hover:text-white transition-colors"
            >
              cancel
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
