import type { Metadata } from 'next'
import { Big_Shoulders_Display, Instrument_Sans, JetBrains_Mono } from 'next/font/google'
import Logo from '@/app/components/Logo'
import './globals.css'

// Fuentes autoalojadas por next/font (antes se pedían a Google en tiempo de ejecución).
const display = Big_Shoulders_Display({ subsets: ['latin'], weight: ['700', '800'], variable: '--font-display' })
const sans = Instrument_Sans({ subsets: ['latin'], variable: '--font-sans' })
const mono = JetBrains_Mono({ subsets: ['latin'], variable: '--font-mono' })

export const metadata: Metadata = {
  title: 'Hangar',
  description: 'Plataforma de despliegue autoalojada: de un repositorio Git a una app con subdominio.',
}

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="es" className={`${display.variable} ${sans.variable} ${mono.variable}`}>
      <body className="min-h-screen">
        <header className="border-b border-[var(--border)]">
          <nav className="mx-auto flex max-w-6xl items-center justify-between px-6 py-4">
            <a href="/" aria-label="Hangar, inicio" className="rounded">
              <Logo />
            </a>
            <span className="hidden text-sm text-[var(--muted)] sm:inline">PaaS autoalojado · Go · Temporal · Docker</span>
          </nav>
        </header>
        <main className="mx-auto max-w-6xl px-6 py-10">{children}</main>
      </body>
    </html>
  )
}
