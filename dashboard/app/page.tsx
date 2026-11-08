import { getApps, getDeployments, getLogs } from '@/app/lib/api'
import { ETAPAS, estadoEtapas } from '@/app/lib/pipeline'
import StatusBadge from '@/app/components/StatusBadge'
import CreateAppForm from '@/app/components/CreateAppForm'

export const dynamic = 'force-dynamic'

const fecha = (iso: string) => new Date(iso).toLocaleString('es-MX', { day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit' })

export default async function HomePage() {
  const apps = await getApps()
  // El estado que importa es el del último despliegue (la app guarda solo el de creación).
  const ultimos = await Promise.all(apps.map(async (app) => (await getDeployments(app.id))[0] ?? null))
  const enLinea = ultimos.filter((d) => d?.status === 'success').length
  const etapas = await Promise.all(
    ultimos.map(async (d) => (d ? estadoEtapas((await getLogs(d.id)).map((l) => l.message), d.status) : null)),
  )

  return (
    <div className="space-y-8">
      <div className="flex flex-wrap items-end justify-between gap-6">
        <div>
          <h1 className="rotulo text-4xl leading-none sm:text-5xl">Aplicaciones</h1>
          <p className="mt-3 text-[var(--muted)]">
            {apps.length} {apps.length === 1 ? 'registrada' : 'registradas'} · <span className="text-[var(--text)]">{enLinea} en línea</span>
          </p>
        </div>
        <CreateAppForm />
      </div>

      {apps.length === 0 ? (
        <div className="rounded-lg border border-dashed border-[var(--border)] p-14 text-center">
          <p className="rotulo text-2xl">El hangar está vacío</p>
          <p className="mt-2 text-sm text-[var(--muted)]">Registra un repositorio con Dockerfile y Hangar lo convierte en una app con subdominio.</p>
        </div>
      ) : (
        <div className="overflow-hidden rounded-lg border border-[var(--border)] bg-[var(--surface)]">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-[var(--border)] text-left text-xs text-[var(--muted)]">
                <th className="px-5 py-3 font-medium">Aplicación</th>
                <th className="hidden px-5 py-3 font-medium sm:table-cell">Subdominio</th>
                <th className="px-5 py-3 font-medium">Último despliegue</th>
                <th className="hidden px-5 py-3 font-medium sm:table-cell">Creada</th>
                <th className="px-5 py-3"><span className="sr-only">Acciones</span></th>
              </tr>
            </thead>
            <tbody>
              {apps.map((app, i) => (
                <tr key={app.id} className="group border-b border-[var(--border)] transition-colors last:border-0 hover:bg-[var(--surface-2)]">
                  <td className="px-5 py-4">
                    <a href={`/apps/${app.id}`} className="font-semibold text-[var(--text)] group-hover:text-[var(--accent)]">
                      {app.name}
                    </a>
                  </td>
                  <td className="mono hidden px-5 py-4 text-xs text-[var(--muted)] sm:table-cell">{app.subdomain}.hangar.local</td>
                  <td className="px-5 py-4">
                    <div className="flex items-center gap-4">
                      <StatusBadge status={ultimos[i]?.status ?? app.status} />
                      {/* Mini riel: las cuatro etapas del último despliegue */}
                      {etapas[i] && (
                        <span className="hidden gap-1 md:flex" aria-label={`Etapas: ${ETAPAS.map((e, k) => `${e.nombre} ${etapas[i]![k]}`).join(', ')}`}>
                          {etapas[i]!.map((estado, k) => (
                            <span
                              key={k}
                              className="h-1.5 w-6 rounded-full"
                              style={{ background: estado === 'listo' ? 'var(--accent)' : estado === 'fallo' ? 'var(--red)' : 'var(--border)' }}
                            />
                          ))}
                        </span>
                      )}
                    </div>
                  </td>
                  <td className="hidden px-5 py-4 text-xs text-[var(--muted)] sm:table-cell">{fecha(app.created_at)}</td>
                  <td className="px-5 py-4 text-right">
                    <a href={`/apps/${app.id}`} className="text-xs font-medium text-[var(--muted)] hover:text-[var(--accent)]">
                      Ver detalle<span className="sr-only"> de {app.name}</span>
                    </a>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}
