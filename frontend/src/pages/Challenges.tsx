import { useEffect, useState } from 'react'
import { api } from '../lib/api'
import type { BadgeRow, Challenge, GamificationSummary } from '../lib/types'
import { ErrorBanner, ProgressBar, Spinner, StatCard } from '../components/ui'

export default function Challenges() {
  const [summary, setSummary] = useState<GamificationSummary | null>(null)
  const [challenges, setChallenges] = useState<Challenge[]>([])
  const [badges, setBadges] = useState<BadgeRow[]>([])
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    Promise.all([
      api.get<GamificationSummary>('/api/v1/gamification/summary'),
      api.get<Challenge[]>('/api/v1/gamification/challenges'),
      api.get<BadgeRow[]>('/api/v1/gamification/badges'),
    ])
      .then(([nextSummary, nextChallenges, nextBadges]) => {
        setSummary(nextSummary)
        setChallenges(nextChallenges)
        setBadges(nextBadges)
      })
      .catch((err: unknown) => setError(err instanceof Error ? err.message : 'Could not load challenges.'))
      .finally(() => setLoading(false))
  }, [])

  if (loading) return <Spinner label="Loading challenges…" />

  return (
    <div className="space-y-7">
      <div>
        <h1 className="text-2xl font-bold text-slate-900">Challenges</h1>
        <p className="mt-1 text-sm text-slate-500">Build a learning habit, earn XP, and collect badges.</p>
      </div>
      <ErrorBanner message={error} />

      <section className="grid grid-cols-2 gap-3 sm:grid-cols-4">
        <StatCard label="Level" value={summary?.level ?? 1} />
        <StatCard label="XP" value={summary?.xp ?? 0} />
        <StatCard label="Current streak" value={`${summary?.streak ?? 0} days`} />
        <StatCard label="Best streak" value={`${summary?.longestStreak ?? 0} days`} />
      </section>

      <section>
        <h2 className="mb-3 text-lg font-bold text-slate-900">Active challenges</h2>
        <div className="grid gap-3 md:grid-cols-2">
          {challenges.map((challenge) => {
            const ratio = challenge.target > 0 ? challenge.progress / challenge.target : 0
            return (
              <article key={challenge.id} className={`card ${challenge.completed ? 'border-emerald-200' : ''}`}>
                <div className="flex items-start justify-between gap-3">
                  <div>
                    <h3 className="font-semibold text-slate-900">{challenge.title}</h3>
                    <p className="mt-1 text-sm text-slate-500">{challenge.description}</p>
                  </div>
                  <span className="shrink-0 rounded-full bg-amber-100 px-2.5 py-0.5 text-xs font-bold text-amber-700">+{challenge.xpReward} XP</span>
                </div>
                <div className="mt-4 flex items-center gap-3">
                  <ProgressBar value={ratio} />
                  <span className="shrink-0 text-xs font-semibold text-slate-600">{Math.min(challenge.progress, challenge.target)}/{challenge.target}</span>
                </div>
                {challenge.completed && <p className="mt-3 text-sm font-semibold text-emerald-700">Completed</p>}
              </article>
            )
          })}
        </div>
      </section>

      <section>
        <h2 className="mb-3 text-lg font-bold text-slate-900">Badges</h2>
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {badges.map((badge) => (
            <article key={badge.id} className={`card ${badge.earned ? 'border-amber-200 bg-amber-50/40' : 'opacity-60'}`}>
              <div className="flex items-center gap-3">
                <span className={`flex h-10 w-10 items-center justify-center rounded-full text-sm font-bold ${badge.earned ? 'bg-amber-200 text-amber-800' : 'bg-slate-200 text-slate-500'}`}>
                  {badge.earned ? 'XP' : '?'}
                </span>
                <div>
                  <h3 className="text-sm font-semibold text-slate-900">{badge.name}</h3>
                  <p className="mt-0.5 text-xs text-slate-500">{badge.description}</p>
                </div>
              </div>
            </article>
          ))}
        </div>
      </section>
    </div>
  )
}
