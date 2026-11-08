// Estado con texto y forma, nunca solo color: punto lleno = en marcha, anillo = detenido, rombo = error.
const config: Record<string, { label: string; color: string; shape: 'dot' | 'ring' | 'diamond' | 'pulse' }> = {
  idle:     { label: 'Sin desplegar', color: 'var(--muted)',  shape: 'ring' },
  building: { label: 'Construyendo',  color: 'var(--yellow)', shape: 'pulse' },
  running:  { label: 'En línea',      color: 'var(--green)',  shape: 'dot' },
  failed:   { label: 'Falló',         color: 'var(--red)',    shape: 'diamond' },
  stopped:  { label: 'Detenida',      color: 'var(--muted)',  shape: 'ring' },
  pending:  { label: 'En cola',       color: 'var(--yellow)', shape: 'pulse' },
  success:  { label: 'Desplegado',    color: 'var(--green)',  shape: 'dot' },
  canceled: { label: 'Cancelado',     color: 'var(--muted)',  shape: 'ring' },
}

export default function StatusBadge({ status }: { status: string }) {
  const s = config[status] ?? { label: status, color: 'var(--muted)', shape: 'ring' as const }
  const marca =
    s.shape === 'diamond' ? 'h-2 w-2 rotate-45' :
    s.shape === 'ring' ? 'h-2 w-2 rounded-full border-2' :
    s.shape === 'pulse' ? 'h-2 w-2 rounded-full animate-pulse' :
    'h-2 w-2 rounded-full'
  return (
    <span className="inline-flex items-center gap-2 rounded-full border border-[var(--border)] bg-[var(--surface-2)] px-2.5 py-1 text-xs font-medium text-[var(--text)]">
      <span
        aria-hidden="true"
        className={marca}
        style={s.shape === 'ring' ? { borderColor: s.color } : { background: s.color }}
      />
      {s.label}
    </span>
  )
}
