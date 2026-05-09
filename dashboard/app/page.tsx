import { getApps } from '@/app/lib/api'
import StatusBadge from '@/app/components/StatusBadge'
import CreateAppForm from '@/app/components/CreateAppForm'

export const dynamic = 'force-dynamic'

export default async function HomePage() {
  const apps = await getApps()

  return (
    <div className="space-y-6">

      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-white text-lg font-medium">apps</h1>
          <p className="text-[var(--muted)] text-xs mt-0.5">
            {apps.length} {apps.length === 1 ? 'app' : 'apps'} registradas
          </p>
        </div>
        <CreateAppForm />
      </div>

      {/* Tabla */}
      {apps.length === 0 ? (
        <div className="border border-dashed border-[var(--border)] rounded-lg p-12 text-center">
          <p className="text-[var(--muted)] text-sm">no hay apps todavía</p>
          <p className="text-[var(--muted)] text-xs mt-1">crea una con el botón de arriba</p>
        </div>
      ) : (
        <div className="border border-[var(--border)] rounded-lg overflow-hidden">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-[var(--border)] bg-[var(--surface)]">
                <th className="text-left px-4 py-3 text-xs text-[var(--muted)] font-normal">name</th>
                <th className="text-left px-4 py-3 text-xs text-[var(--muted)] font-normal">subdomain</th>
                <th className="text-left px-4 py-3 text-xs text-[var(--muted)] font-normal">status</th>
                <th className="text-left px-4 py-3 text-xs text-[var(--muted)] font-normal">created</th>
                <th className="px-4 py-3" />
              </tr>
            </thead>
            <tbody>
              {apps.map((app, i) => (
                <tr
                  key={app.id}
                  className={`border-b border-[var(--border)] last:border-0 hover:bg-[var(--surface)] transition-colors ${i % 2 === 0 ? '' : 'bg-white/[0.02]'}`}
                >
                  <td className="px-4 py-3">
                    <a
                      href={`/apps/${app.id}`}
                      className="text-white hover:text-[var(--accent)] transition-colors"
                    >
                      {app.name}
                    </a>
                  </td>
                  <td className="px-4 py-3 text-[var(--muted)] text-xs">
                    {app.subdomain}.hangar.local
                  </td>
                  <td className="px-4 py-3">
                    <StatusBadge status={app.status} />
                  </td>
                  <td className="px-4 py-3 text-[var(--muted)] text-xs">
                    {new Date(app.created_at).toLocaleDateString()}
                  </td>
                  <td className="px-4 py-3 text-right">
                    <a
                      href={`/apps/${app.id}`}
                      className="text-xs text-[var(--muted)] hover:text-[var(--accent)] transition-colors"
                    >
                      view ->
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
