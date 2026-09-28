import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { api } from '../lib/api'
import type { RecoveryResponse, Recommendation, StartQuizResponse } from '../lib/types'
import { ErrorBanner, MasteryBadge, Spinner } from '../components/ui'

export default function Recovery() {
  const navigate = useNavigate()
  const [missions, setMissions] = useState<Recommendation[]>([])
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(true)
  const [startingId, setStartingId] = useState<number | null>(null)

  useEffect(() => {
    api
      .get<RecoveryResponse>('/api/v1/recommendations/recovery')
      .then((data) => setMissions(data.missions))
      .catch((err: unknown) => setError(err instanceof Error ? err.message : 'Could not load recovery missions.'))
      .finally(() => setLoading(false))
  }, [])

  async function startMission(mission: Recommendation) {
    setStartingId(mission.id)
    setError('')
    try {
      const quiz = await api.post<StartQuizResponse>(`/api/v1/topics/${mission.topic.id}/quiz/start`, {
        questionCount: 5,
        recovery: true,
      })
      const state = { quizId: quiz.quizId, questions: quiz.questions, topicName: quiz.topic.name }
      sessionStorage.setItem(`quiz:${quiz.quizId}`, JSON.stringify(state))
      navigate(`/quiz/${quiz.quizId}`, { state })
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Could not start this recovery mission.')
      setStartingId(null)
    }
  }

  if (loading) return <Spinner label="Loading recovery missions…" />

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-slate-900">Recovery missions</h1>
        <p className="mt-1 text-sm text-slate-500">Revisit topics that need attention, then prove your progress in a short quiz.</p>
      </div>

      <ErrorBanner message={error} />

      {missions.length === 0 ? (
        <div className="card text-center">
          <h2 className="font-semibold text-slate-900">No recovery missions right now</h2>
          <p className="mt-1 text-sm text-slate-500">Complete quizzes to build your personalised recovery plan.</p>
        </div>
      ) : (
        <div className="grid gap-4 md:grid-cols-2">
          {missions.map((mission) => (
            <article key={mission.id} className="card">
              <div className="flex items-start justify-between gap-3">
                <div>
                  <div className="text-xs font-bold uppercase tracking-wide text-rose-600">{mission.topic.subject.name}</div>
                  <h2 className="mt-1 font-semibold text-slate-900">{mission.topic.name}</h2>
                </div>
                <MasteryBadge state={mission.topic.state} mastery={mission.topic.mastery} />
              </div>
              <p className="mt-3 text-sm text-slate-600">{mission.reason}</p>
              <div className="mt-4 rounded-xl bg-slate-50 p-3 text-sm text-slate-600 dark:bg-slate-800/80">
                <div className="font-semibold text-slate-700">Mission steps</div>
                <ol className="mt-1 list-inside list-decimal space-y-1">
                  {mission.steps.split(', ').map((step) => (
                    <li key={step}>{step}</li>
                  ))}
                </ol>
              </div>
              <button className="btn-primary mt-4 w-full" disabled={startingId !== null} onClick={() => startMission(mission)}>
                {startingId === mission.id ? 'Starting mission…' : 'Start recovery quiz'}
              </button>
            </article>
          ))}
        </div>
      )}
    </div>
  )
}
