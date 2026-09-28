import { useState } from 'react'
import { Link, useLocation, useNavigate } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'
import { ErrorBanner } from '../components/ui'
import { ThemeToggle } from '../components/ThemeToggle'

export default function Login() {
  const { login } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()
  const initialState = location.state as { email?: string; verified?: boolean; from?: { pathname: string } } | null
  const [email, setEmail] = useState(initialState?.email ?? '')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    setBusy(true)
    try {
      await login(email, password)
      const from = initialState?.from?.pathname
      navigate(onboardFrom(from), { replace: true })
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Login failed.'
      if (message.toLowerCase().includes('verify your email')) {
        navigate('/verify-email', { state: { email } })
        return
      }
      setError(message)
    } finally {
      setBusy(false)
    }
  }

  return (
    <AuthShell>
      <h1 className="text-2xl font-bold text-slate-900">Welcome back</h1>
      <p className="mt-1 text-sm text-slate-500">Sign in to continue learning.</p>
      {initialState?.verified && <p className="mt-4 rounded-xl bg-emerald-50 px-3 py-2 text-sm font-medium text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-300">Email verified. You can sign in now.</p>}
      <form onSubmit={onSubmit} className="mt-6 space-y-4">
        <div>
          <label className="label">Email</label>
          <input
            className="input"
            type="email"
            required
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            placeholder="you@example.com"
          />
        </div>
        <div>
          <label className="label">Password</label>
          <input
            className="input"
            type="password"
            required
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            placeholder="••••••••"
          />
        </div>
        <ErrorBanner message={error} />
        <button className="btn-primary w-full" disabled={busy}>
          {busy ? 'Signing in…' : 'Sign in'}
        </button>
      </form>
      <p className="mt-4 text-center text-sm text-slate-500">
        New to LearnQuest?{' '}
        <Link to="/register" className="font-semibold text-indigo-600 hover:underline">
          Create an account
        </Link>
      </p>
    </AuthShell>
  )
}

function onboardFrom(from?: string) {
  return from && from !== '/login' && from !== '/register' ? from : '/dashboard'
}

export function AuthShell({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex min-h-screen bg-slate-50 transition-colors dark:bg-slate-950">
      <aside className="relative hidden min-h-screen w-1/2 overflow-hidden bg-gradient-to-br from-indigo-600 via-indigo-700 to-violet-950 p-10 lg:flex lg:flex-col lg:justify-between xl:p-12">
        <div className="absolute -right-24 -top-24 h-96 w-96 rounded-full bg-white/10 blur-3xl" />
        <div className="absolute -bottom-28 -left-28 h-96 w-96 rounded-full bg-violet-300/20 blur-3xl" />
        <Link to="/" className="relative flex items-center gap-2.5 self-start">
          <span className="grid h-10 w-10 place-items-center rounded-xl bg-white/15 text-sm font-bold text-white ring-1 ring-white/20">LQ</span>
          <span className="text-xl font-bold tracking-tight text-white">LearnQuest</span>
        </Link>

        <div className="relative mx-auto w-full max-w-md">
          <h2 className="max-w-sm text-3xl font-bold leading-tight text-white">Feedback the moment you need it, not days later.</h2>
          <div className="relative mt-10 rotate-[-2deg] rounded-2xl bg-white p-5 shadow-2xl shadow-indigo-950/40">
            <div className="absolute -left-5 -top-6 rounded-2xl bg-white px-3 py-2 shadow-xl">
              <div className="text-sm font-bold text-slate-900">6 days</div>
              <div className="text-[10px] text-slate-400">learning streak</div>
            </div>
            <div className="absolute -right-5 -top-5 rounded-2xl bg-white px-3 py-2 shadow-xl">
              <div className="text-sm font-bold text-amber-600">+50 XP</div>
              <div className="text-[10px] text-slate-400">earned today</div>
            </div>
            <div className="flex justify-between text-xs font-semibold"><span className="text-slate-400">SS2 · Physics · Waves</span><span className="text-indigo-600">Question 3 of 10</span></div>
            <p className="mt-4 text-sm font-semibold leading-relaxed text-slate-900">What happens to wave speed when frequency increases and wavelength stays constant?</p>
            <div className="mt-4 flex items-center justify-between rounded-xl border-2 border-emerald-500 bg-emerald-50 p-3 text-sm font-medium text-emerald-800"><span>Wave speed increases</span><span className="grid h-5 w-5 place-items-center rounded-full bg-emerald-600 text-xs text-white">✓</span></div>
            <div className="mt-2 rounded-xl border border-slate-100 p-3 text-sm text-slate-300">Wave speed decreases</div>
            <div className="mt-3 rounded-xl bg-indigo-50 p-3 text-xs leading-relaxed text-slate-600"><span className="font-semibold text-indigo-700">AI tutor: </span>v = fλ, so speed rises with frequency when wavelength stays the same.</div>
          </div>
          <div className="mt-7 flex flex-wrap gap-2 text-xs font-medium text-indigo-50">
            <span className="rounded-full bg-white/10 px-3 py-1.5 ring-1 ring-white/10">Instant AI feedback</span>
            <span className="rounded-full bg-white/10 px-3 py-1.5 ring-1 ring-white/10">WAEC & NECO aligned</span>
            <span className="rounded-full bg-white/10 px-3 py-1.5 ring-1 ring-white/10">Built for JSS1 to SS3</span>
          </div>
        </div>

        <p className="relative text-sm text-indigo-200">© 2026 LearnQuest</p>
      </aside>

      <main className="relative flex min-h-screen flex-1 items-center justify-center p-5 sm:p-10">
        <div className="absolute right-5 top-5 sm:right-10 sm:top-8"><ThemeToggle /></div>
        <div className="w-full max-w-md rounded-3xl border border-white bg-white/90 p-7 shadow-xl shadow-slate-900/5 backdrop-blur dark:border-slate-800 dark:bg-slate-900/90 sm:p-10">
          <Link to="/" className="mb-8 flex items-center gap-2.5 lg:hidden">
            <span className="grid h-9 w-9 place-items-center rounded-xl bg-gradient-to-br from-indigo-600 to-violet-800 text-xs font-bold text-white">LQ</span>
            <span className="text-xl font-bold tracking-tight text-slate-900 dark:text-slate-100">LearnQuest</span>
          </Link>
          {children}
        </div>
      </main>
    </div>
  )
}
