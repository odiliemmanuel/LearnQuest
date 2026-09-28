import { useState } from 'react'
import { Link } from 'react-router-dom'
import { useEffect } from 'react'
import { useAuth } from '../context/AuthContext'
import { api } from '../lib/api'
import type { ClassLevel, EducationLevel, Subject, SubjectProgress } from '../lib/types'
import { ErrorBanner, Spinner } from '../components/ui'

export default function Subjects() {
  const { profile } = useAuth()

  const [levels, setLevels] = useState<EducationLevel[]>([])
  const [classes, setClasses] = useState<ClassLevel[]>([])
  const [levelId, setLevelId] = useState<string>(String(profile?.educationLevelId ?? ''))
  const [classId, setClassId] = useState<string>(String(profile?.classLevelId ?? ''))
  const [subjects, setSubjects] = useState<Subject[]>([])
  const [progress, setProgress] = useState<SubjectProgress[]>([])
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    api
      .get<EducationLevel[]>('/api/v1/curriculum/levels')
      .then(setLevels)
      .catch(() => setError('Could not load education levels.'))
  }, [])

  useEffect(() => {
    if (!levelId) return
    api
      .get<ClassLevel[]>(`/api/v1/curriculum/classes?levelId=${levelId}`)
      .then(setClasses)
      .catch(() => setError('Could not load classes.'))
  }, [levelId])

  useEffect(() => {
    if (!levelId || !classId) return
    setLoading(true)
    Promise.all([
      api.get<Subject[]>(`/api/v1/curriculum/subjects-for-class?classId=${classId}`),
      api.get<SubjectProgress[]>('/api/v1/progress').catch(() => []),
    ])
      .then(([subs, prog]) => {
        setSubjects(subs)
        setProgress(prog)
        setLoading(false)
      })
      .catch(() => {
        setError('Could not load subjects.')
        setLoading(false)
      })
  }, [levelId, classId])

  const pctFor = (subjectId: number) => progress.find((p) => p.subject?.id === subjectId)?.percentage ?? 0

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-slate-900">Subjects</h1>
        <p className="mt-1 text-sm text-slate-500">Pick a subject to explore its topics and lessons.</p>
      </div>

      <div className="flex flex-wrap gap-3">
        <div className="min-w-48">
          <label className="label">Education level</label>
          <select
            className="input"
            value={levelId}
            onChange={(e) => {
              setLevelId(e.target.value)
              setClassId('')
            }}
          >
            <option value="">Choose…</option>
            {levels.map((l) => (
              <option key={l.id} value={l.id}>
                {l.name}
              </option>
            ))}
          </select>
        </div>
        <div className="min-w-48">
          <label className="label">Class</label>
          <select className="input" value={classId} onChange={(e) => setClassId(e.target.value)} disabled={!levelId}>
            <option value="">Choose…</option>
            {classes.map((c) => (
              <option key={c.id} value={c.id}>
                {c.name}
              </option>
            ))}
          </select>
        </div>
      </div>

      <ErrorBanner message={error} />
      {!classId && <p className="text-sm text-slate-500">Select your class to see its subjects.</p>}
      {loading && <Spinner />}

      {!loading && classId && (
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {subjects.map((s) => (
            <Link key={s.id} to={`/subjects/${s.id}/topics?classId=${classId}`} className="card block hover:border-indigo-300">
              <div className="flex items-center justify-between">
                <span className="flex h-10 w-10 items-center justify-center rounded-xl bg-indigo-600/10 text-sm font-bold text-indigo-700">
                  {s.code.slice(0, 3)}
                </span>
                <span className="rounded-full bg-slate-100 px-2.5 py-0.5 text-xs font-semibold text-slate-500">
                  {Math.round(pctFor(s.id) * 100)}%
                </span>
              </div>
              <div className="mt-3 font-semibold text-slate-900">{s.name}</div>
              <div className="mt-3 h-1.5 w-full overflow-hidden rounded-full bg-slate-200">
                <div className="h-full rounded-full bg-indigo-600" style={{ width: `${pctFor(s.id) * 100}%` }} />
              </div>
            </Link>
          ))}
        </div>
      )}
    </div>
  )
}
