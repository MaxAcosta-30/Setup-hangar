import { ETAPAS, type EstadoEtapa } from '@/app/lib/pipeline'

const TEXTO: Record<EstadoEtapa, string> = { pendiente: 'En espera', activo: 'En curso', listo: 'Listo', fallo: 'Falló' }

// Riel del pipeline: la pieza firma de Hangar. Cada etapa se enciende con los logs reales.
export default function Riel({ estados }: { estados: EstadoEtapa[] }) {
  return (
    <ol className="riel" aria-label="Etapas del despliegue">
      {ETAPAS.map((etapa, i) => (
        <li key={etapa.id} className="riel-paso" data-estado={estados[i]}>
          <span className="mono text-[11px] text-[var(--muted)]">{String(i + 1).padStart(2, '0')}</span>
          <p className="rotulo mt-1 text-lg leading-none">{etapa.nombre}</p>
          <p className="mono mt-1.5 truncate text-[11px] text-[var(--muted)]">{etapa.detalle}</p>
          <p className={`mt-2 text-xs font-medium ${estados[i] === 'fallo' ? 'text-[var(--red)]' : estados[i] === 'listo' ? 'text-[var(--accent)]' : 'text-[var(--muted)]'}`}>
            {TEXTO[estados[i]]}
          </p>
        </li>
      ))}
    </ol>
  )
}
