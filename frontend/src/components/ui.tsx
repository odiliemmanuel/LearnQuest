import type { MasteryState } from '../lib/types'

export function MasteryBadge({ state, mastery }: { state?: MasteryState; mastery?: number }) {
  const map: Record<string, string> = {
    STRONG: 'bg-emerald-100 text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-300',
    IMPROVING: 'bg-amber-100 text-amber-700 dark:bg-amber-500/15 dark:text-amber-300',
    WEAK: 'bg-rose-100 text-rose-700 dark:bg-rose-500/15 dark:text-rose-300',
    NEW: 'bg-slate-200 text-slate-600 dark:bg-slate-700 dark:text-slate-300',
  }
  const label: Record<string, string> = {
    STRONG: 'Strong',
    IMPROVING: 'Improving',
    WEAK: 'Needs work',
    NEW: 'New',
  }
  if (!state) return null
  return (
    <span className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-semibold ${map[state] ?? map.NEW}`}>
      {label[state] ?? state}
      {typeof mastery === 'number' && ` · ${Math.round(mastery * 100)}%`}
    </span>
  )
}

export function ProgressBar({ value, className = '' }: { value: number; className?: string }) {
  const pct = Math.max(0, Math.min(100, Math.round(value * 100)))
  return (
    <div className={`h-2 w-full overflow-hidden rounded-full bg-slate-200 dark:bg-slate-800 ${className}`}>
      <div className="h-full rounded-full bg-indigo-600 transition-all" style={{ width: `${pct}%` }} />
    </div>
  )
}

export function StatCard({ label, value, sub }: { label: string; value: React.ReactNode; sub?: string }) {
  return (
    <div className="card">
      <div className="text-xs font-medium uppercase tracking-wide text-slate-400">{label}</div>
      <div className="mt-1 text-2xl font-bold text-slate-900 dark:text-slate-100">{value}</div>
      {sub && <div className="mt-0.5 text-xs text-slate-500 dark:text-slate-400">{sub}</div>}
    </div>
  )
}

export function ErrorBanner({ message }: { message: string }) {
  if (!message) return null
  return (
    <div className="rounded-xl border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-700 dark:border-rose-500/30 dark:bg-rose-500/10 dark:text-rose-300" role="alert">
      {message}
    </div>
  )
}

export function Spinner({ label = 'Loading…' }: { label?: string }) {
  return (
    <div className="flex items-center justify-center gap-2 py-10 text-sm text-slate-500 dark:text-slate-400">
      <span className="h-4 w-4 animate-spin rounded-full border-2 border-slate-300 border-t-indigo-600 dark:border-slate-700" />
      {label}
    </div>
  )
}
