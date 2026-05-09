import { getApp, getDeployments } from '@/app/lib/api'
import StatusBadge from '@/app/components/StatusBadge'
import DeployButton from '@/app/components/DeployButton'

export const dynamic = 'force-dynamic'

export default async function AppPage({ params }: { params: { id: string } }) {
  const [app, deployments] = await Promise.all([
    getApp(params.id),
    getDeployments(params.id),
  ])

  return (
    <div className="space-y-8">

      {/* Breadcrumb */}
      <div className="flex items-center gap-2 text-xs text-[var(--muted)]">
        <a href="/" className="hover:text-white transition-colors">apps</a>
        <span>/</span>
        <span className="text-white">{app.name}</span>
      </div>

      {/* App header */}
      <div className="flex items-start justify-between">
        <div className="space-y-1">
          <div className="flex items-center gap-3">
            <h1 className="text-white text-lg font-medium">{app.name}</h1>
            <StatusBadge status={app.status} />
          </div>
          <a
            href={`http://${app.subdomain}.hangar.local`}
            target="_blank"
            rel="noopener noreferrer"
            className="text-xs text-[var(--muted)] hover:text-[var(--accent)] transition-colors"
          >
            {app.subdomain}.hangar.local ->
          </a>
        </div>
        <DeployButton appId={app.id} />
      </div>

      {/* Info */}
      <div className="grid grid-cols-2 gap-4">
        {[
          { label: 'git url', value: app.git_url },
          { label: 'app id', value: app.id },
        ].map(({ label, value }) => (
          <div key={label} className="border border-[var(--border)] rounded-lg p-4">
            <p className="text-xs text-[var(--muted)] mb-1">{label}</p>
            <p className="text-xs text-white font-mono truncate">{value}</p>
          </div>
        ))}
      </div>

      {/* Historial de deployments */}
      <div className="space-y-3">
        <h2 className="text-white text-sm font-medium">deployments</h2>

        {deployments.length === 0 ? (
          <div className="border border-dashed border-[var(--border)] rounded-lg p-8 text-center">
            <p className="text-[var(--muted)] text-xs">sin deployments todavía - usa el botón deploy</p>
          </div>
        ) : (
          <div className="border border-[var(--border)] rounded-lg overflow-hidden">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-[var(--border)] bg-[var(--surface)]">
                  <th className="text-left px-4 py-3 text-xs text-[var(--muted)] font-normal">id</th>
                  <th className="text-left px-4 py-3 text-xs text-[var(--muted)] font-normal">status</th>
                  <th className="text-left px-4 py-3 text-xs text-[var(--muted)] font-normal">commit</th>
                  <th className="text-left px-4 py-3 text-xs text-[var(--muted)] font-normal">date</th>
                  <th className="px-4 py-3" />
                </tr>
              </thead>
              <tbody>
                {deployments.map((d) => (
                  <tr
                    key={d.id}
                    className="border-b border-[var(--border)] last:border-0 hover:bg-[var(--surface)] transition-colors"
                  >
                    <td className="px-4 py-3 font-mono text-xs text-[var(--muted)]">
                      {d.id.slice(0, 8)}
                    </td>
                    <td className="px-4 py-3">
                      <StatusBadge status={d.status} />
                    </td>
                    <td className="px-4 py-3 font-mono text-xs text-[var(--muted)]">
                      {d.commit_sha ? d.commit_sha.slice(0, 7) : '-'}
                    </td>
                    <td className="px-4 py-3 text-xs text-[var(--muted)]">
                      {new Date(d.created_at).toLocaleString()}
                    </td>
                    <td className="px-4 py-3 text-right">
                      <a
                        href={`/apps/${app.id}/deployments/${d.id}`}
                        className="text-xs text-[var(--muted)] hover:text-[var(--accent)] transition-colors"
                      >
                        logs ->
                      </a>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  )
}
