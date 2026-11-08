import { getApp, getDeployments, getLogs } from '@/app/lib/api'
import StatusBadge from '@/app/components/StatusBadge'
import DeployButton from '@/app/components/DeployButton'
import Riel from '@/app/components/Riel'
import { estadoEtapas } from '@/app/lib/pipeline'

export const dynamic = 'force-dynamic'

const fecha = (iso: string) => new Date(iso).toLocaleString('es-MX', { day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit' })

export default async function AppPage({ params }: { params: { id: string } }) {
  const [app, deployments] = await Promise.all([getApp(params.id), getDeployments(params.id)])
  const ultimo = deployments[0]
  const logs = ultimo ? await getLogs(ultimo.id) : []

  return (
    <div className="space-y-10">
      <nav aria-label="Ruta" className="flex items-center gap-2 text-sm text-[var(--muted)]">
        <a href="/" className="hover:text-[var(--text)]">Aplicaciones</a>
        <span aria-hidden="true">/</span>
        <span className="text-[var(--text)]" aria-current="page">{app.name}</span>
      </nav>

      <div className="flex flex-wrap items-end justify-between gap-6">
        <div>
          <div className="flex flex-wrap items-center gap-4">
            <h1 className="rotulo text-4xl leading-none sm:text-5xl">{app.name}</h1>
            <StatusBadge status={ultimo?.status ?? app.status} />
          </div>
          <p className="mono mt-3 text-sm text-[var(--muted)]">{app.subdomain}.hangar.local</p>
        </div>
        <DeployButton appId={app.id} />
      </div>

      {ultimo && (
        <section className="space-y-3">
          <h2 className="text-sm font-semibold text-[var(--muted)]">Último despliegue · {fecha(ultimo.created_at)}</h2>
          <Riel estados={estadoEtapas(logs.map((l) => l.message), ultimo.status)} />
        </section>
      )}

      <dl className="grid gap-px overflow-hidden rounded-lg border border-[var(--border)] bg-[var(--border)] sm:grid-cols-2">
        {[
          { label: 'Repositorio', value: app.git_url },
          { label: 'ID de la app', value: app.id },
        ].map(({ label, value }) => (
          <div key={label} className="bg-[var(--surface)] p-4">
            <dt className="text-xs text-[var(--muted)]">{label}</dt>
            <dd className="mono mt-1 truncate text-sm">{value}</dd>
          </div>
        ))}
      </dl>

      <section className="space-y-3">
        <h2 className="rotulo text-2xl">Despliegues</h2>
        {deployments.length === 0 ? (
          <div className="rounded-lg border border-dashed border-[var(--border)] p-10 text-center text-sm text-[var(--muted)]">
            Todavía no hay despliegues. Usa «Desplegar» para lanzar el primero.
          </div>
        ) : (
          <div className="overflow-hidden rounded-lg border border-[var(--border)] bg-[var(--surface)]">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-[var(--border)] text-left text-xs text-[var(--muted)]">
                  <th className="px-5 py-3 font-medium">Despliegue</th>
                  <th className="px-5 py-3 font-medium">Estado</th>
                  <th className="hidden px-5 py-3 font-medium sm:table-cell">Commit</th>
                  <th className="hidden px-5 py-3 font-medium sm:table-cell">Fecha</th>
                  <th className="px-5 py-3"><span className="sr-only">Logs</span></th>
                </tr>
              </thead>
              <tbody>
                {deployments.map((d) => (
                  <tr key={d.id} className="border-b border-[var(--border)] last:border-0 hover:bg-[var(--surface-2)]">
                    <td className="mono px-5 py-3.5 text-xs">{d.id.slice(0, 8)}</td>
                    <td className="px-5 py-3.5"><StatusBadge status={d.status} /></td>
                    <td className="mono hidden px-5 py-3.5 text-xs text-[var(--muted)] sm:table-cell">{d.commit_sha ? d.commit_sha.slice(0, 7) : '—'}</td>
                    <td className="hidden px-5 py-3.5 text-xs text-[var(--muted)] sm:table-cell">{fecha(d.created_at)}</td>
                    <td className="px-5 py-3.5 text-right">
                      <a href={`/apps/${app.id}/deployments/${d.id}`} className="text-xs font-medium text-[var(--muted)] hover:text-[var(--accent)]">
                        Ver logs<span className="sr-only"> del despliegue {d.id.slice(0, 8)}</span>
                      </a>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>
    </div>
  )
}
