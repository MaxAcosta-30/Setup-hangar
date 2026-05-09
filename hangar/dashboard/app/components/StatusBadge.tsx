type Status = string

const config: Record<string, { label: string; color: string }> = {
  idle:     { label: 'idle',     color: 'text-[var(--muted)] border-[var(--border)]' },
  building: { label: 'building', color: 'text-[var(--yellow)] border-yellow-800 animate-pulse' },
  running:  { label: 'running',  color: 'text-[var(--green)] border-green-900' },
  failed:   { label: 'failed',   color: 'text-[var(--red)] border-red-900' },
  stopped:  { label: 'stopped',  color: 'text-[var(--muted)] border-[var(--border)]' },
  pending:  { label: 'pending',  color: 'text-[var(--yellow)] border-yellow-800 animate-pulse' },
  success:  { label: 'success',  color: 'text-[var(--green)] border-green-900' },
  canceled: { label: 'canceled', color: 'text-[var(--muted)] border-[var(--border)]' },
}

export default function StatusBadge({ status }: { status: Status }) {
  const s = config[status] ?? { label: status, color: 'text-[var(--muted)] border-[var(--border)]' }
  return (
    <span className={`text-xs px-2 py-0.5 border rounded font-mono ${s.color}`}>
      {s.label}
    </span>
  )
}
