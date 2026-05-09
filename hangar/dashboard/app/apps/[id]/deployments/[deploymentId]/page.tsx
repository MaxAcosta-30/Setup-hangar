import { getApp, getDeployments } from '@/app/lib/api'
import LogStream from '@/app/components/LogStream'

export const dynamic = 'force-dynamic'

export default async function DeploymentPage({
  params,
}: {
  params: { id: string; deploymentId: string }
}) {
  const [app, deployments] = await Promise.all([
    getApp(params.id),
    getDeployments(params.id),
  ])

  const deployment = deployments.find(d => d.id === params.deploymentId)

  if (!deployment) {
    return (
      <div className="text-center py-20 text-[var(--muted)]">
        deployment no encontrado
      </div>
    )
  }

  return (
    <div className="space-y-8">

      {/* Breadcrumb */}
      <div className="flex items-center gap-2 text-xs text-[var(--muted)]">
        <a href="/" className="hover:text-white transition-colors">apps</a>
        <span>/</span>
        <a href={`/apps/${app.id}`} className="hover:text-white transition-colors">{app.name}</a>
        <span>/</span>
        <span className="text-white">{deployment.id.slice(0, 8)}</span>
      </div>

      {/* Título */}
      <div>
        <h1 className="text-white text-lg font-medium">deployment logs</h1>
        <p className="text-[var(--muted)] text-xs mt-0.5">
          {app.name} - {new Date(deployment.created_at).toLocaleString()}
        </p>
      </div>

      {/* Stream de logs en tiempo real */}
      <LogStream
        deploymentId={deployment.id}
        initialStatus={deployment.status}
      />

      {/* Link de vuelta */}
      <a
        href={`/apps/${app.id}`}
        className="inline-block text-xs text-[var(--muted)] hover:text-[var(--accent)] transition-colors"
      >
        <- volver a {app.name}
      </a>
    </div>
  )
}
