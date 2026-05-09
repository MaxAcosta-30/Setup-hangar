'use client'

import { useEffect, useRef, useState } from 'react'

interface LogLine {
  type: 'log' | 'status' | 'done' | 'error'
  message: string
  time: string
}

interface Props {
  deploymentId: string
  initialStatus: string
}

export default function LogStream({ deploymentId, initialStatus }: Props) {
  const [lines, setLines] = useState<LogLine[]>([])
  const [status, setStatus] = useState(initialStatus)
  const [connected, setConnected] = useState(false)
  const bottomRef = useRef<HTMLDivElement>(null)
  const wsRef = useRef<WebSocket | null>(null)

  useEffect(() => {
    // WS va directo al backend en :3000 (no pasa por Next.js proxy)
    // porque los rewrites de Next.js no aplican a WebSockets
    const wsUrl = `ws://localhost:3000/api/v1/deployments/${deploymentId}/stream`
    const ws = new WebSocket(wsUrl)
    wsRef.current = ws

    ws.onopen = () => setConnected(true)

    ws.onmessage = (event) => {
      const msg: LogLine = JSON.parse(event.data)
      if (msg.type === 'done') {
        setStatus(msg.message) // 'success' o 'failed'
        setConnected(false)
      } else {
        setLines(prev => [...prev, msg])
      }
    }

    ws.onclose = () => setConnected(false)
    ws.onerror = () => setConnected(false)

    return () => ws.close()
  }, [deploymentId])

  // Auto-scroll al último log
  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [lines])

  const statusColor =
    status === 'success' ? 'text-[var(--green)]' :
    status === 'failed'  ? 'text-[var(--red)]' :
    'text-[var(--yellow)]'

  return (
    <div className="space-y-4">

      {/* Header con estado de conexión */}
      <div className="flex items-center gap-3">
        <span className={`text-sm font-mono ${statusColor}`}>
          {status === 'success' ? 'success' :
           status === 'failed'  ? 'failed' :
           'running'}
        </span>
        {connected && (
          <span className="flex items-center gap-1 text-xs text-[var(--muted)]">
            <span className="w-1.5 h-1.5 rounded-full bg-[var(--green)] animate-pulse inline-block" />
            live
          </span>
        )}
      </div>

      {/* Terminal de logs */}
      <div className="bg-[#0d0d0d] border border-[var(--border)] rounded-lg overflow-hidden">

        {/* Barra de título del terminal */}
        <div className="flex items-center gap-1.5 px-4 py-2.5 border-b border-[var(--border)] bg-[var(--surface)]">
          <span className="w-3 h-3 rounded-full bg-[#ff5f56]" />
          <span className="w-3 h-3 rounded-full bg-[#ffbd2e]" />
          <span className="w-3 h-3 rounded-full bg-[#27c93f]" />
          <span className="ml-3 text-xs text-[var(--muted)]">
            deploy - {deploymentId.slice(0, 8)}
          </span>
        </div>

        {/* Líneas de log */}
        <div className="p-4 h-96 overflow-y-auto font-mono text-xs leading-relaxed">
          {lines.length === 0 && (
            <span className="text-[var(--muted)]">esperando logs...</span>
          )}
          {lines.map((line, i) => (
            <div key={i} className="flex gap-3 hover:bg-white/5 px-1 -mx-1 rounded">
              <span className="text-[var(--muted)] shrink-0 select-none">
                {new Date(line.time).toLocaleTimeString('en-US', { hour12: false })}
              </span>
              <span className={
                line.message.includes('[done]') ? 'text-[var(--green)]' :
                line.message.includes('error') ? 'text-[var(--red)]' :
                line.message.startsWith('[build]') ? 'text-[var(--muted)]' :
                'text-[var(--text)]'
              }>
                {line.message}
              </span>
            </div>
          ))}

          {/* Cursor parpadeante mientras está conectado */}
          {connected && (
            <div className="flex gap-3 px-1">
              <span className="text-[var(--muted)] select-none">
                {new Date().toLocaleTimeString('en-US', { hour12: false })}
              </span>
              <span className="animate-pulse text-[var(--accent)]">_</span>
            </div>
          )}

          <div ref={bottomRef} />
        </div>
      </div>
    </div>
  )
}
