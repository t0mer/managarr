// web/src/components/Navbar.tsx
import { useState, useEffect } from 'react'
import { Sun, Moon } from 'lucide-react'

export function Navbar() {
  const [dark, setDark] = useState(
    () => document.documentElement.classList.contains('dark'),
  )

  useEffect(() => {
    document.documentElement.classList.toggle('dark', dark)
    localStorage.setItem('theme', dark ? 'dark' : 'light')
  }, [dark])

  return (
    <header className="h-12 shrink-0 border-b border-[var(--border)] bg-[var(--bg)] flex items-center justify-end px-4 gap-2">
      <button
        onClick={() => setDark(d => !d)}
        className="p-1.5 rounded-md hover:bg-[var(--border)] transition-colors"
        aria-label="Toggle dark mode"
      >
        {dark ? <Sun size={16} /> : <Moon size={16} />}
      </button>
    </header>
  )
}
