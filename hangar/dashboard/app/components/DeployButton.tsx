'use client'

import { useState } from 'react'
import { useRouter } from 'next/navigation'
import { triggerDeploy } from '@/app/lib/api'

export default function DeployButton({ appId }: { appId: string }) {
  const [loading, setLoading] = useState(false)
  const router = useRouter()

  async function handleDeploy() {
    setLoading(true)
    try {
      const data = await triggerDeploy(appId)
      // Redirige a la vista de logs del nuevo deployment
      router.push(`/apps/${appId}/deployments/${data.deployment.id}`)
    } catch (e) {
      console.error('deploy error', e)
      setLoading(false)
    }
  }

  return (
    <button
      onClick={handleDeploy}
      disabled={loading}
      className="
        px-4 py-2 text-sm font-mono
        bg-[var(--accent)] text-white rounded
        hover:bg-blue-400 active:bg-blue-700
        disabled:opacity-50 disabled:cursor-not-allowed
        transition-colors
      "
    >
      {loading ? 'deploying...' : 'deploy'}
    </button>
  )
}
