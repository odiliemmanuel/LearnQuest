import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { api } from '../lib/api'
import type { KnowledgeGroup, SubjectProgress } from '../lib/types'
import { ErrorBanner, MasteryBadge, ProgressBar, Spinner } from '../components/ui'

interface KnowledgeMapResponse {
  subjects: KnowledgeGroup[]
}

export default function Progress() {
  const [progress, setProgress] = useState<SubjectProgress[]>([])
  const [groups, setGroups] = useState<KnowledgeGroup[]>([])
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    Promise.all([api.get<SubjectProgress[]>('/api/v1/progress'), api.get<KnowledgeMapResponse>('/api/v1/knowledge-map')])
      .then(([subjectProgress, map]) => {
        setProgress(subjectProgress)
        setGroups(map.subjects)
      })
      .catch((err: unknown) => setError(err instanceof Error ? err.message : 'Could not load progress.'))
      .finally(() => setLoading(false))
  }, [])

  if (loading) return <Spinner label="Loading progress…" />

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-slate-900">Your progress</h1>
        <p className="mt-1 text-sm text-slate-500">Your knowledge map grows every time you complete a quiz.</p>
      </div>
      <ErrorBanner message={error} />

      <section className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
        {progress.map((item) => (
          <div key={item.id} className="card">
            <div className="flex items-center justify-between gap-3">
              <h2 className="font-semibold text-slate-900">{item.subject?.name ?? 'Subject'}</h2>
              <span className="text-sm font-bold text-indigo-600">{Math.round(item.percentage * 100)}%</span>
            </div>
            <ProgressBar value={item.percentage} className="mt-3" />
            <p className="mt-3 text-xs text-slate-500">
              {item.topicsMastered}/{item.topicsTotal} topics strong · {item.correctCount}/{item.totalCount} answers correct
            </p>
          </div>
        ))}
      </section>

      {groups.length === 0 ? (
        <div className="card text-center">
          <h2 className="font-semibold text-slate-900">Your knowledge map is waiting</h2>
          <p className="mt-1 text-sm text-slate-500">Take a quiz to see topic-level mastery here.</p>
          <Link to="/subjects" className="btn-primary mt-4">Browse subjects</Link>
        </div>
      ) : (
        <section className="space-y-4">
          <h2 className="text-lg font-bold text-slate-900">Knowledge map</h2>
          {groups.map((group) => (
            <div key={group.subject.id} className="card">
              <div className="flex flex-wrap items-center justify-between gap-2">
                <div>
                  <h3 className="font-semibold text-slate-900">{group.subject.name}</h3>
                  <p className="text-xs text-slate-500">{group.mastered} strong · {group.improving} improving · {group.weak} needs work</p>
                </div>
                <span className="text-sm font-bold text-indigo-600">{Math.round(group.percentage * 100)}%</span>
              </div>
              <ProgressBar value={group.percentage} className="mt-3" />
              <div className="mt-4 grid gap-2 sm:grid-cols-2">
                {group.topics.map((topic) => (
                  <Link key={topic.id} to={`/topics/${topic.id}`} className="rounded-xl border border-slate-200 p-3 hover:border-indigo-300 dark:border-slate-700 dark:hover:border-indigo-500/50">
                    <div className="flex items-start justify-between gap-2">
                      <span className="text-sm font-medium text-slate-800">{topic.name}</span>
                      <MasteryBadge state={topic.state} mastery={topic.mastery} />
                    </div>
                    <p className="mt-2 text-xs text-slate-500">{topic.correct}/{topic.attempts} correct · {topic.termName}</p>
                  </Link>
                ))}
              </div>
            </div>
          ))}
        </section>
      )}
    </div>
  )
}
