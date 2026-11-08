import { getApp, getDeployments } from '@/app/lib/api'
import LogStream from '@/app/components/LogStream'

export const dynamic = 'force-dynamic'

export default async function DeploymentPage({ params }: { params: { id: string; deploymentId: string } }) {
  const [app, deployments] = await Promise.all([getApp(params.id), getDeployments(params.id)])
  const deployment = deployments.find((d) => d.id === params.deploymentId)

  if (!deployment) {
    return (
      <div className="py-20 text-center">
        <p className="rotulo text-3xl">Despliegue no encontrado</p>
        <a href={`/apps/${params.id}`} className="mt-3 inline-block text-sm text-[var(--accent)] hover:underline">Volver a la aplicación</a>
      </div>
    )
  }

  return (
    <div className="space-y-8">
      <nav aria-label="Ruta" className="flex items-center gap-2 text-sm text-[var(--muted)]">
        <a href="/" className="hover:text-[var(--text)]">Aplicaciones</a>
        <span aria-hidden="true">/</span>
        <a href={`/apps/${app.id}`} className="hover:text-[var(--text)]">{app.name}</a>
        <span aria-hidden="true">/</span>
        <span className="mono text-[var(--text)]" aria-current="page">{deployment.id.slice(0, 8)}</span>
      </nav>

      <div>
        <h1 className="rotulo text-5xl leading-none">Despliegue de {app.name}</h1>
        <p className="mt-3 text-[var(--muted)]">
          {new Date(deployment.created_at).toLocaleString('es-MX', { dateStyle: 'medium', timeStyle: 'short' })} · Temporal coordina cada etapa y reintenta si algo falla.
        </p>
      </div>

      <LogStream deploymentId={deployment.id} initialStatus={deployment.status} />
    </div>
  )
}
