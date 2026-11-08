'use client'

import { useState } from 'react'
import { useRouter } from 'next/navigation'
import { triggerDeploy } from '@/app/lib/api'

export default function DeployButton({ appId }: { appId: string }) {
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const router = useRouter()

  async function handleDeploy() {
    setLoading(true)
    setError('')
    try {
      const data = await triggerDeploy(appId)
      // Lleva a la vista de logs del despliegue nuevo
      router.push(`/apps/${appId}/deployments/${data.deployment.id}`)
    } catch {
      setError('No se pudo iniciar el despliegue. ¿Está corriendo la API?')
      setLoading(false)
    }
  }

  return (
    <div className="flex flex-col items-end gap-2">
      <button
        onClick={handleDeploy}
        disabled={loading}
        className="rotulo rounded-md bg-[var(--accent)] px-6 py-3 text-lg leading-none text-[var(--on-accent)] transition-[transform,filter] duration-150 hover:brightness-110 active:scale-[0.97] disabled:cursor-not-allowed disabled:opacity-60"
      >
        {loading ? 'Despegando…' : 'Desplegar'}
      </button>
      {error && <p role="alert" className="text-xs text-[var(--red)]">{error}</p>}
    </div>
  )
}
