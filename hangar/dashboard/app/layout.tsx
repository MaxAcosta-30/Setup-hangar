import type { Metadata } from 'next'
import './globals.css'

export const metadata: Metadata = {
  title: 'Hangar',
  description: 'Self-hosted deployment platform',
}

export default function RootLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <html lang="en">
      <body className="min-h-screen">
        <nav className="border-b border-[var(--border)] px-6 py-4 flex items-center justify-between">
          <a href="/" className="text-white font-medium tracking-tight flex items-center gap-2">
            <span className="text-[var(--accent)]"></span>
            <span>hangar</span>
          </a>
          <span className="text-[var(--muted)] text-xs">self-hosted paas</span>
        </nav>
        <main className="max-w-5xl mx-auto px-6 py-8">
          {children}
        </main>
      </body>
    </html>
  )
}
