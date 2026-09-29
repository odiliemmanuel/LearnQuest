import { useEffect, useState } from 'react'
import { NavLink, Outlet, useNavigate } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'
import { api } from '../lib/api'
import type { GamificationSummary } from '../lib/types'
import { ThemeToggle } from './ThemeToggle'

export function AppLayout() {
  const { user, logout } = useAuth()
  const navigate = useNavigate()
  const [summary, setSummary] = useState<GamificationSummary | null>(null)
  const [open, setOpen] = useState(false)

  const navItems = [
    { to: '/dashboard', label: 'Dashboard' },
    { to: '/subjects', label: 'Subjects' },
    { to: '/library', label: 'Library' },
    { to: '/recovery', label: 'Recovery' },
    { to: '/progress', label: 'Progress' },
    { to: '/challenges', label: 'Challenges' },
    { to: '/profile', label: 'Profile' },
    ...(user?.role === 'ADMIN' ? [{ to: '/admin', label: 'Admin' }] : []),
  ]

  useEffect(() => {
    api
      .get<GamificationSummary>('/api/v1/gamification/summary')
      .then(setSummary)
      .catch(() => undefined)
  }, [])

  function handleLogout() {
    logout()
    navigate('/login')
  }

  const linkCls = ({ isActive }: { isActive: boolean }) =>
    `rounded-lg px-3 py-2 text-sm font-medium transition ${
      isActive
        ? 'bg-indigo-50 text-indigo-700 dark:bg-indigo-500/15 dark:text-indigo-300'
        : 'text-slate-600 hover:bg-slate-100 hover:text-slate-900 dark:text-slate-300 dark:hover:bg-slate-800 dark:hover:text-white'
    }`

  return (
    <div className="min-h-screen">
      <header className="sticky top-0 z-20 border-b border-slate-200 bg-white/90 backdrop-blur dark:border-slate-800 dark:bg-slate-950/90">
        <div className="mx-auto flex h-16 max-w-7xl items-center justify-between gap-3 px-4 sm:px-6">
          <button onClick={() => navigate('/dashboard')} className="flex items-center gap-2">
            <span className="flex h-8 w-8 items-center justify-center rounded-lg bg-indigo-600 text-sm font-bold text-white">
              LQ
            </span>
            <span className="text-base font-bold tracking-tight text-slate-900 dark:text-slate-100">LearnQuest</span>
          </button>

          <nav className="hidden items-center gap-1 md:flex">
            {navItems.map((n) => (
              <NavLink key={n.to} to={n.to} className={linkCls}>
                {n.label}
              </NavLink>
            ))}
          </nav>

          <div className="flex items-center gap-2">
            {summary && (
              <span className="hidden rounded-full bg-indigo-50 px-3 py-1 text-xs font-semibold text-indigo-700 sm:inline">
                Lv {summary.level} · {summary.xp} XP
              </span>
            )}
            <ThemeToggle />
            <button
              onClick={() => setOpen((v) => !v)}
              className="rounded-lg px-3 py-1.5 text-sm font-medium text-slate-700 hover:bg-slate-100 dark:text-slate-200 dark:hover:bg-slate-800"
            >
              {user?.name.split(' ')[0]}
            </button>
            <button onClick={handleLogout} className="btn-secondary !px-3 !py-1.5 text-xs">
              Sign out
            </button>
          </div>
        </div>

        {open && (
          <div className="border-t border-slate-100 bg-white dark:border-slate-800 dark:bg-slate-950 md:hidden">
            <div className="mx-auto max-w-7xl px-4 py-2 sm:px-6">
              {navItems.map((n) => (
                <NavLink
                  key={n.to}
                  to={n.to}
                  onClick={() => setOpen(false)}
                  className={({ isActive }) =>
                    `block rounded-lg px-3 py-2 text-sm font-medium ${isActive ? 'bg-indigo-50 text-indigo-700 dark:bg-indigo-500/15 dark:text-indigo-300' : 'text-slate-600 dark:text-slate-300'}`
                  }
                >
                  {n.label}
                </NavLink>
              ))}
            </div>
          </div>
        )}
      </header>

      <main className="mx-auto max-w-7xl px-4 py-8 sm:px-6 sm:py-10">
        <Outlet />
      </main>
    </div>
  )
}
