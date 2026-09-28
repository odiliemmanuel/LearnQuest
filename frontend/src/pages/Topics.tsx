import { useEffect, useState } from 'react'
import { Link, useParams, useSearchParams } from 'react-router-dom'
import { api } from '../lib/api'
import type { Subject, Term, TopicView } from '../lib/types'
import { ErrorBanner, MasteryBadge, ProgressBar, Spinner } from '../components/ui'

export default function Topics() {
  const { subjectId } = useParams()
  const [params] = useSearchParams()
  const classId = params.get('classId') ?? ''

  const [subject, setSubject] = useState<Subject | null>(null)
  const [terms, setTerms] = useState<Term[]>([])
  const [termId, setTermId] = useState('')
  const [topics, setTopics] = useState<TopicView[]>([])
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    Promise.all([
      api.get<Subject[]>(`/api/v1/curriculum/subjects`).then((s) => s.find((x) => x.id === Number(subjectId)) ?? null),
      api.get<Term[]>('/api/v1/curriculum/terms'),
    ])
      .then(([subj, ts]) => {
        setSubject(subj)
        setTerms(ts)
        if (ts.length > 0) setTermId(String(ts[0].id))
      })
      .catch(() => setError('Could not load subject data.'))
  }, [subjectId])

  useEffect(() => {
    if (!classId || !termId) return
    setLoading(true)
    api
      .get<TopicView[]>(`/api/v1/curriculum/topics?classId=${classId}&subjectId=${subjectId}&termId=${termId}`)
      .then((t) => {
        setTopics(t)
        setLoading(false)
      })
      .catch(() => {
        setError('Could not load topics.')
        setLoading(false)
      })
  }, [classId, subjectId, termId])

  return (
    <div className="space-y-6">
      <div>
        <Link to="/subjects" className="text-sm font-semibold text-indigo-600 hover:underline">
          ← Subjects
        </Link>
        <h1 className="mt-2 text-2xl font-bold text-slate-900">{subject?.name ?? 'Topics'}</h1>
      </div>

      <div className="flex flex-wrap gap-2">
        {terms.map((t) => (
          <button
            key={t.id}
            onClick={() => setTermId(String(t.id))}
            className={`rounded-full px-4 py-1.5 text-sm font-semibold transition ${
              String(t.id) === termId
                ? 'bg-indigo-600 text-white'
                : 'bg-white text-slate-600 ring-1 ring-slate-200 hover:bg-slate-50 dark:bg-slate-900 dark:text-slate-300 dark:ring-slate-700 dark:hover:bg-slate-800'
            }`}
          >
            {t.name}
          </button>
        ))}
      </div>

      <ErrorBanner message={error} />
      {loading && <Spinner />}

      {!loading && topics.length === 0 && <p className="text-sm text-slate-500">No topics in this filter yet.</p>}

      <div className="grid gap-3 sm:grid-cols-2">
        {topics.map((t) => (
          <Link
            key={t.id}
            to={`/topics/${t.id}`}
            className="card block transition hover:border-indigo-300 hover:shadow"
          >
            <div className="flex items-start justify-between gap-2">
              <div>
                <div className="font-semibold text-slate-900">{t.name}</div>
                <div className="mt-1 text-sm text-slate-500">{t.description || 'Learn this topic with a lesson and quiz.'}</div>
              </div>
              <MasteryBadge state={t.state} mastery={t.mastery} />
            </div>
            <div className="mt-4 flex items-center justify-between gap-3">
              <div className="flex items-center gap-2 text-xs text-slate-500">
                {t.hasLesson && <span className="rounded-full bg-indigo-50 px-2 py-0.5 font-semibold text-indigo-600">Lesson</span>}
                {t.hasQuestions && <span className="rounded-full bg-emerald-50 px-2 py-0.5 font-semibold text-emerald-600">Quiz</span>}
              </div>
              <span className="text-xs text-slate-400">~{t.estimatedMinutes} min</span>
            </div>
            {t.mastery !== undefined && <ProgressBar value={t.mastery} className="mt-3" />}
          </Link>
        ))}
      </div>
    </div>
  )
}
