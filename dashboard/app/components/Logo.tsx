// Marca de Hangar: la nave en arco con su portón abierto; la franja naranja es la app que sale.
export default function Logo() {
  return (
    <span className="inline-flex items-center gap-2.5">
      <svg width="30" height="30" viewBox="0 0 32 32" aria-hidden="true" focusable="false">
        <rect width="32" height="32" rx="6" fill="#ff6a13" />
        <path d="M5 25V16a11 9 0 0 1 22 0v9" fill="none" stroke="#160a02" strokeWidth="2.6" strokeLinejoin="round" />
        <path d="M10 25v-7h12v7" fill="none" stroke="#160a02" strokeWidth="2.2" />
        <path d="M10 21h12" stroke="#160a02" strokeWidth="1.6" />
        <rect x="4" y="24.5" width="24" height="2.5" rx="1" fill="#160a02" />
      </svg>
      <span className="rotulo text-[22px] leading-none text-[var(--text)]">Hangar</span>
    </span>
  )
}
