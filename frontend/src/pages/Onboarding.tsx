import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'
import { api } from '../lib/api'
import type { ClassLevel, EducationLevel } from '../lib/types'
import { ErrorBanner, Spinner } from '../components/ui'
import { ThemeToggle } from '../components/ThemeToggle'

export default function Onboarding() {
  const { refresh, user } = useAuth()
  const navigate = useNavigate()
  const [levels, setLevels] = useState<EducationLevel[]>([])
  const [classes, setClasses] = useState<ClassLevel[]>([])
  const [levelId, setLevelId] = useState('')
  const [classId, setClassId] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

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

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!levelId || !classId) {
      setError('Please choose your level and class.')
      return
    }
    setBusy(true)
    setError('')
    try {
      await api.put('/api/v1/student/profile', {
        educationLevelId: Number(levelId),
        classLevelId: Number(classId),
      })
      await refresh()
      navigate('/dashboard', { replace: true })
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Could not save your details.')
    } finally {
      setBusy(false)
    }
  }

  if (levels.length === 0 && !error) return <Spinner label="Loading curriculum…" />

  return (
    <div className="relative flex min-h-screen items-center justify-center bg-gradient-to-br from-indigo-600 via-indigo-500 to-violet-600 px-4">
      <div className="absolute right-5 top-5"><ThemeToggle /></div>
      <div className="w-full max-w-md">
        <div className="card">
          <h1 className="text-xl font-bold text-slate-900">Tell us where you are</h1>
          <p className="mt-1 text-sm text-slate-500">
            Hi {user?.name.split(' ')[0]}. We’ll tailor your curriculum, lessons and quizzes to your class.
          </p>
          <form onSubmit={onSubmit} className="mt-6 space-y-4">
            <div>
              <label className="label">Education level</label>
              <select className="input" value={levelId} onChange={(e) => { setLevelId(e.target.value); setClassId('') }}>
                <option value="">Choose…</option>
                {levels.map((l) => (
                  <option key={l.id} value={l.id}>
                    {l.name}
                  </option>
                ))}
              </select>
            </div>
            <div>
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
            <ErrorBanner message={error} />
            <button className="btn-primary w-full" disabled={busy || !levelId || !classId}>
              {busy ? 'Saving…' : 'Start learning'}
            </button>
          </form>
        </div>
      </div>
    </div>
  )
}
