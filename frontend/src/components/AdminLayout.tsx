import { NavLink, Outlet, useNavigate } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'
import { ThemeToggle } from './ThemeToggle'

const navItems = [
  { to: '/admin', label: 'Overview', end: true },
  { to: '/admin/users', label: 'Students', end: false },
]

export function AdminLayout() {
  const { logout } = useAuth()
  const navigate = useNavigate()

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
            <span className="hidden rounded-full border border-indigo-200 px-2 py-0.5 text-[10px] font-bold uppercase tracking-wide text-indigo-600 sm:inline dark:border-indigo-500/40 dark:text-indigo-300">
              Admin
            </span>
          </button>

          <nav className="hidden items-center gap-1 md:flex">
            {navItems.map((n) => (
              <NavLink key={n.to} to={n.to} end={n.end} className={linkCls}>
                {n.label}
              </NavLink>
            ))}
          </nav>

          <div className="flex items-center gap-2">
            <ThemeToggle />
            <button onClick={() => navigate('/dashboard')} className="btn-secondary !px-3 !py-1.5 text-xs">
              Back to app
            </button>
            <button onClick={() => { logout(); navigate('/login') }} className="btn-primary !px-3 !py-1.5 text-xs">
              Sign out
            </button>
          </div>
        </div>
      </header>

      <main className="mx-auto max-w-7xl px-4 py-8 sm:px-6 sm:py-10">
        <Outlet />
      </main>
    </div>
  )
}