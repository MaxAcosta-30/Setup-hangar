'use client'

import { useEffect, useRef, useState } from 'react'
import Riel from '@/app/components/Riel'
import StatusBadge from '@/app/components/StatusBadge'
import { estadoEtapas } from '@/app/lib/pipeline'

interface LogLine {
  type: 'log' | 'status' | 'done' | 'error'
  message: string
  time: string
}

interface Props {
  deploymentId: string
  initialStatus: string
}

// El WebSocket va directo al backend: los rewrites de Next.js no aplican a WebSockets.
const WS_URL = process.env.NEXT_PUBLIC_HANGAR_WS_URL ?? 'ws://localhost:3000'

// Separa la etiqueta de origen ([hangar], [build]) y la marca [DONE] del texto del log.
function partes(message: string) {
  const etiqueta = /^\[(\w+)\]\s*/.exec(message)
  const resto = etiqueta ? message.slice(etiqueta[0].length) : message
  const listo = resto.includes('[DONE]')
  return { origen: etiqueta?.[1] ?? '', texto: resto.replace('[DONE]', '').trim(), listo }
}

export default function LogStream({ deploymentId, initialStatus }: Props) {
  const [lines, setLines] = useState<LogLine[]>([])
  const [status, setStatus] = useState(initialStatus)
  const [connected, setConnected] = useState(false)
  const panel = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const ws = new WebSocket(`${WS_URL}/api/v1/deployments/${deploymentId}/stream`)
    ws.onopen = () => setConnected(true)
    ws.onmessage = (event) => {
      const msg: LogLine = JSON.parse(event.data)
      if (msg.type === 'done') {
        setStatus(msg.message) // 'success' o 'failed'
        setConnected(false)
      } else {
        setLines((prev) => [...prev, msg])
      }
    }
    ws.onclose = () => setConnected(false)
    ws.onerror = () => setConnected(false)
    return () => ws.close()
  }, [deploymentId])

  // Sigue el final del log sin mover la página entera.
  useEffect(() => {
    const p = panel.current
    if (p) p.scrollTop = p.scrollHeight
  }, [lines])

  const estados = estadoEtapas(lines.map((l) => l.message), connected ? 'running' : status)

  return (
    <div className="space-y-6">
      <Riel estados={estados} />

      <div className="overflow-hidden rounded-lg border border-[var(--border)] bg-[#0a0b0d]">
        <div className="flex items-center justify-between gap-4 border-b border-[var(--border)] bg-[var(--surface)] px-4 py-2.5">
          <span className="mono text-xs text-[var(--muted)]">deploy · {deploymentId.slice(0, 8)} · {lines.length} líneas</span>
          <div className="flex items-center gap-3">
            {connected && (
              <span className="flex items-center gap-1.5 text-xs text-[var(--text)]">
                <span className="h-1.5 w-1.5 animate-pulse rounded-full bg-[var(--green)]" aria-hidden="true" />
                En vivo
              </span>
            )}
            <StatusBadge status={connected ? 'running' : status} />
          </div>
        </div>

        <div ref={panel} className="mono h-[26rem] overflow-y-auto py-3 text-[12.5px] leading-relaxed" role="log" aria-live="polite">
          {lines.length === 0 && <p className="px-4 text-[var(--muted)]">Esperando los primeros logs del worker…</p>}
          {lines.map((line, i) => {
            const { origen, texto, listo } = partes(line.message)
            const error = /error|falló|fail/i.test(texto)
            return (
              <div key={i} className="grid grid-cols-[3.25rem_4.5rem_4.25rem_1fr] items-baseline gap-2 px-4 hover:bg-white/[0.03]">
                <span className="select-none text-right text-[var(--border)]">{i + 1}</span>
                <span className="select-none text-[var(--muted)]">{new Date(line.time).toLocaleTimeString('es-MX', { hour12: false })}</span>
                <span className={origen === 'build' ? 'text-[var(--muted)]' : 'text-[var(--accent)]'}>{origen}</span>
                <span className={error ? 'text-[var(--red)]' : origen === 'build' ? 'text-[var(--muted)]' : 'text-[var(--text)]'}>
                  {texto}
                  {listo && <span className="ml-2 rounded bg-[var(--accent)] px-1.5 py-px text-[10px] font-bold text-[var(--on-accent)]">LISTO</span>}
                </span>
              </div>
            )
          })}
        </div>
      </div>
    </div>
  )
}
