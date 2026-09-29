import { useEffect, useState } from 'react'
import { api } from '../lib/api'
import type { AdminUserRow } from '../lib/types'
import { ErrorBanner, Spinner } from '../components/ui'

export default function AdminUsers() {
  const [users, setUsers] = useState<AdminUserRow[]>([])
  const [query, setQuery] = useState('')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [deleting, setDeleting] = useState<number | null>(null)

  useEffect(() => {
    const timer = setTimeout(() => {
      setLoading(true)
      setError('')
      api
        .get<AdminUserRow[]>(`/api/v1/admin/users?q=${encodeURIComponent(query)}`)
        .then(setUsers)
        .catch((err: unknown) => {
          setError(err instanceof Error ? err.message : 'Could not load students.')
          setUsers([])
        })
        .finally(() => setLoading(false))
    }, 250)
    return () => clearTimeout(timer)
  }, [query])

  async function removeUser(user: AdminUserRow) {
    if (user.role === 'ADMIN') return
    const ok = window.confirm(`Soft-delete ${user.name} (${user.email})?\n\nTheir data and history stay in the database but they lose access to the app.`)
    if (!ok) return
    setDeleting(user.id)
    setError('')
    try {
      await api.delete(`/api/v1/admin/users/${user.id}`)
      setUsers((current) => current.filter((u) => u.id !== user.id))
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Could not delete user.')
    } finally {
      setDeleting(null)
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold text-slate-900 dark:text-white">Students</h1>
          <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">Accounts, activity and performance across the platform.</p>
        </div>
        <input
          className="input w-full sm:w-72"
          placeholder="Search by name or email…"
          value={query}
          onChange={(event) => setQuery(event.target.value)}
        />
      </div>

      <ErrorBanner message={error} />

      {loading ? (
        <Spinner label="Loading students…" />
      ) : users.length === 0 ? (
        <div className="rounded-3xl border-2 border-dashed border-slate-200 p-8 text-center text-sm text-slate-500 dark:border-slate-700 dark:text-slate-400">
          No students match.
        </div>
      ) : (
        <div className="card overflow-x-auto">
          <table className="w-full min-w-[720px] text-left text-sm">
            <thead>
              <tr className="border-b border-slate-200 text-xs uppercase tracking-wide text-slate-400 dark:border-slate-800">
                <th className="px-4 py-3 font-semibold">Student</th>
                <th className="px-4 py-3 font-semibold">Status</th>
                <th className="px-4 py-3 text-right font-semibold">Quizzes</th>
                <th className="px-4 py-3 text-right font-semibold">Avg score</th>
                <th className="px-4 py-3 text-right font-semibold">XP</th>
                <th className="px-4 py-3 text-right font-semibold">Strong topics</th>
                <th className="px-4 py-3 font-semibold">Last active</th>
                <th className="px-4 py-3 font-semibold" aria-label="Actions" />
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100 dark:divide-slate-800">
              {users.map((user) => (
                <tr key={user.id} className="hover:bg-slate-50 dark:hover:bg-slate-800/50">
                  <td className="px-4 py-3">
                    <div className="font-semibold text-slate-900 dark:text-white">{user.name}</div>
                    <div className="text-xs text-slate-500 dark:text-slate-400">{user.email}</div>
                  </td>
                  <td className="px-4 py-3">
                    <div className="flex items-center gap-2">
                      {user.role === 'ADMIN' && (
                        <span className="rounded-full bg-indigo-100 px-2.5 py-0.5 text-xs font-semibold text-indigo-700 dark:bg-indigo-500/15 dark:text-indigo-300">Admin</span>
                      )}
                      <span className={`rounded-full px-2.5 py-0.5 text-xs font-semibold ${user.emailVerified ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-300' : 'bg-amber-100 text-amber-700 dark:bg-amber-500/15 dark:text-amber-300'}`}>
                        {user.emailVerified ? 'Verified' : 'Pending'}
                      </span>
                      {user.onboarded && (
                        <span className="rounded-full bg-sky-100 px-2.5 py-0.5 text-xs font-semibold text-sky-700 dark:bg-sky-500/15 dark:text-sky-300">Onboarded</span>
                      )}
                    </div>
                  </td>
                  <td className="px-4 py-3 text-right text-slate-700 dark:text-slate-300">{user.submittedCount}<span className="text-slate-400">/{user.quizCount}</span></td>
                  <td className="px-4 py-3 text-right text-slate-700 dark:text-slate-300">{user.submittedCount ? `${Math.round(user.avgScore)}%` : '—'}</td>
                  <td className="px-4 py-3 text-right font-semibold text-slate-900 dark:text-white">{user.xp}</td>
                  <td className="px-4 py-3 text-right text-slate-700 dark:text-slate-300">{user.masteredTopics}</td>
                  <td className="px-4 py-3 text-slate-500 dark:text-slate-400">{user.lastActiveAt ? new Date(user.lastActiveAt).toLocaleDateString() : 'Never'}</td>
                  <td className="px-4 py-3 text-right">
                    {user.role === 'ADMIN' ? (
                      <span className="text-xs text-slate-400" title="Admin accounts cannot be deleted">—</span>
                    ) : (
                      <button
                        onClick={() => removeUser(user)}
                        disabled={deleting === user.id}
                        className="rounded-lg px-2.5 py-1 text-xs font-semibold text-rose-600 transition hover:bg-rose-50 disabled:opacity-50 dark:text-rose-400 dark:hover:bg-rose-500/10"
                      >
                        {deleting === user.id ? 'Deleting…' : 'Delete'}
                      </button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}