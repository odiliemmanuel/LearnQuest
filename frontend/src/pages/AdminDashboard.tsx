import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { api } from '../lib/api'
import type { AdminOverview } from '../lib/types'
import { ErrorBanner, Spinner, StatCard } from '../components/ui'

export default function AdminDashboard() {
  const [overview, setOverview] = useState<AdminOverview | null>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    api
      .get<AdminOverview>('/api/v1/admin/overview')
      .then(setOverview)
      .catch((err: unknown) => setError(err instanceof Error ? err.message : 'Could not load admin overview.'))
  }, [])

  if (!overview && !error) return <Spinner label="Loading admin overview…" />

  return (
    <div className="space-y-8">
      <div>
        <h1 className="text-2xl font-bold text-slate-900 dark:text-white">Admin overview</h1>
        <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">Platform health, activity and registrations.</p>
      </div>

      <ErrorBanner message={error} />

      {overview && (
        <>
          <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
            <StatCard label="Students" value={overview.totalUsers} sub={`${overview.verifiedUsers} verified · ${overview.onboardedUsers} onboarded`} />
            <StatCard label="Quizzes" value={overview.totalQuizzes} sub={`${overview.submittedQuizzes} submitted`} />
            <StatCard label="Avg score" value={`${Math.round(overview.avgScore)}%`} sub={`${overview.totalAnswers} answers graded`} />
            <StatCard label="Active today" value={overview.activeStudentsToday} sub={`${overview.aiAnalyses} AI analyses`} />
          </section>

          <section>
            <div className="mb-4 flex items-end justify-between gap-4">
              <div>
                <div className="text-xs font-bold uppercase tracking-wide text-slate-400">Latest sign-ups</div>
                <h2 className="mt-1 text-xl font-bold text-slate-900 dark:text-white">Recent registrations</h2>
              </div>
              <Link to="/admin/users" className="text-sm font-semibold text-indigo-600 hover:underline dark:text-indigo-300">
                Manage students
              </Link>
            </div>
            {overview.recentRegistrations.length === 0 ? (
              <div className="rounded-3xl border-2 border-dashed border-slate-200 p-8 text-center text-sm text-slate-500 dark:border-slate-700 dark:text-slate-400">
                No accounts yet.
              </div>
            ) : (
              <div className="card divide-y divide-slate-100 dark:divide-slate-800">
                {overview.recentRegistrations.map((user) => (
                  <div key={user.id} className="flex items-center justify-between gap-4 py-3">
                    <div className="min-w-0">
                      <div className="truncate font-semibold text-slate-900 dark:text-white">{user.name}</div>
                      <div className="truncate text-sm text-slate-500 dark:text-slate-400">{user.email}</div>
                    </div>
                    <div className="flex shrink-0 items-center gap-2">
                      {user.role === 'ADMIN' && (
                        <span className="rounded-full bg-indigo-100 px-2.5 py-0.5 text-xs font-semibold text-indigo-700 dark:bg-indigo-500/15 dark:text-indigo-300">Admin</span>
                      )}
                      <span className={`rounded-full px-2.5 py-0.5 text-xs font-semibold ${user.emailVerified ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-300' : 'bg-amber-100 text-amber-700 dark:bg-amber-500/15 dark:text-amber-300'}`}>
                        {user.emailVerified ? 'Verified' : 'Pending'}
                      </span>
                      <span className="text-xs text-slate-400">{new Date(user.createdAt).toLocaleDateString()}</span>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </section>
        </>
      )}
    </div>
  )
}