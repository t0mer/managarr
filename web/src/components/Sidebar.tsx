// web/src/components/Sidebar.tsx
import { NavLink } from 'react-router-dom'
import {
  LayoutDashboard, ScrollText, AlertCircle, Server,
  Archive, RefreshCw, Settings,
} from 'lucide-react'
import { cn } from '@/lib/utils'

const nav = [
  { to: '/',        label: 'Dashboard', icon: LayoutDashboard },
  { to: '/logs',    label: 'Logs',      icon: ScrollText },
  { to: '/issues',  label: 'Issues',    icon: AlertCircle },
  { to: '/apps',    label: 'Apps',      icon: Server },
  { to: '/backup',  label: 'Backup',    icon: Archive },
  { to: '/sync',    label: 'Sync',      icon: RefreshCw },
  { to: '/settings',label: 'Settings',  icon: Settings },
]

export function Sidebar() {
  return (
    <aside className="w-56 shrink-0 h-full border-r border-[var(--border)] bg-[var(--sidebar-bg)] flex flex-col">
      <div className="px-4 py-5 border-b border-[var(--border)]">
        <span className="text-lg font-bold tracking-tight">Galactica</span>
      </div>
      <nav className="flex-1 overflow-y-auto py-3 px-2 space-y-0.5">
        {nav.map(({ to, label, icon: Icon }) => (
          <NavLink
            key={to}
            to={to}
            end={to === '/'}
            className={({ isActive }) =>
              cn(
                'flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors',
                isActive
                  ? 'bg-blue-600 text-white'
                  : 'text-[var(--fg)] hover:bg-[var(--border)]',
              )
            }
          >
            <Icon size={16} />
            {label}
          </NavLink>
        ))}
      </nav>
    </aside>
  )
}
