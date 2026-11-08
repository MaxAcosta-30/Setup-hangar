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
        setError(data.error ?? 'No se pudo registrar la app')
      }
    } catch {
      setError('No hay conexión con la API de Hangar')
    } finally {
      setLoading(false)
    }
  }

  if (!open) {
    return (
      <button
        onClick={() => setOpen(true)}
        className="rounded-md border border-[var(--border)] bg-[var(--surface)] px-4 py-2.5 text-sm font-semibold text-[var(--text)] transition-colors hover:border-[var(--accent)] hover:text-[var(--accent)]"
      >
        Registrar app
      </button>
    )
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 p-4" role="dialog" aria-modal="true" aria-labelledby="nueva-app">
      <div className="w-full max-w-md rounded-lg border border-[var(--border)] bg-[var(--surface)] p-6">
        <h2 id="nueva-app" className="rotulo mb-1 text-3xl">Registrar app</h2>
        <p className="mb-6 text-sm text-[var(--muted)]">El repositorio necesita un Dockerfile en la raíz.</p>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label htmlFor="nombre-app" className="mb-1.5 block text-sm font-medium">Nombre</label>
            <input
              type="text"
              value={name}
              onChange={e => setName(e.target.value)}
              placeholder="landing-panaderia"
              id="nombre-app"
              required
              className="w-full bg-[var(--bg)] border border-[var(--border)] rounded px-3 py-2 text-sm font-mono text-[var(--text)] focus:border-[var(--accent)] focus:outline-none"
            />
          </div>
          <div>
            <label htmlFor="git-url" className="mb-1.5 block text-sm font-medium">Repositorio Git</label>
            <input
              type="text"
              value={gitUrl}
              onChange={e => setGitUrl(e.target.value)}
              placeholder="https://github.com/usuario/repo"
              id="git-url"
              required
              className="w-full bg-[var(--bg)] border border-[var(--border)] rounded px-3 py-2 text-sm font-mono text-[var(--text)] focus:border-[var(--accent)] focus:outline-none"
            />
          </div>
          {error && <p role="alert" className="text-sm text-[var(--red)]">{error}</p>}
          <div className="flex gap-3 pt-2">
            <button
              type="submit"
              disabled={loading}
              className="flex-1 rounded-md bg-[var(--accent)] py-2.5 text-sm font-semibold text-[var(--on-accent)] transition-[filter] hover:brightness-110 disabled:opacity-60"
            >
              {loading ? 'Registrando…' : 'Registrar'}
            </button>
            <button
              type="button"
              onClick={() => setOpen(false)}
              className="rounded-md border border-[var(--border)] px-4 py-2.5 text-sm text-[var(--muted)] transition-colors hover:text-[var(--text)]"
            >
              Cancelar
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
