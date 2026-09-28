import { useEffect, useState } from 'react'
import { useAuth } from '../context/AuthContext'
import { api } from '../lib/api'
import type { ClassLevel, EducationLevel } from '../lib/types'
import { ErrorBanner, Spinner } from '../components/ui'

export default function Profile() {
  const { user, profile, refresh } = useAuth()
  const [levels, setLevels] = useState<EducationLevel[]>([])
  const [classes, setClasses] = useState<ClassLevel[]>([])
  const [levelId, setLevelId] = useState(String(profile?.educationLevelId ?? ''))
  const [classId, setClassId] = useState(String(profile?.classLevelId ?? ''))
  const [error, setError] = useState('')
  const [message, setMessage] = useState('')
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    api
      .get<EducationLevel[]>('/api/v1/curriculum/levels')
      .then(setLevels)
      .catch((err: unknown) => setError(err instanceof Error ? err.message : 'Could not load education levels.'))
      .finally(() => setLoading(false))
  }, [])

  useEffect(() => {
    if (!levelId) {
      setClasses([])
      return
    }
    api
      .get<ClassLevel[]>(`/api/v1/curriculum/classes?levelId=${levelId}`)
      .then(setClasses)
      .catch((err: unknown) => setError(err instanceof Error ? err.message : 'Could not load classes.'))
  }, [levelId])

  async function save(event: React.FormEvent) {
    event.preventDefault()
    if (!levelId || !classId) {
      setError('Choose an education level and class.')
      return
    }
    setSaving(true)
    setError('')
    setMessage('')
    try {
      await api.put('/api/v1/student/profile', { educationLevelId: Number(levelId), classLevelId: Number(classId) })
      await refresh()
      setMessage('Your class details have been updated.')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Could not save your profile.')
    } finally {
      setSaving(false)
    }
  }

  if (loading) return <Spinner label="Loading profile…" />

  return (
    <div className="mx-auto max-w-xl space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-slate-900">Profile</h1>
        <p className="mt-1 text-sm text-slate-500">Keep your curriculum aligned with your current class.</p>
      </div>
      <div className="card">
        <dl className="grid gap-3 text-sm sm:grid-cols-2">
          <div><dt className="text-slate-400">Name</dt><dd className="font-semibold text-slate-800">{user?.name ?? 'Student'}</dd></div>
          <div><dt className="text-slate-400">Email</dt><dd className="font-semibold text-slate-800">{user?.email ?? 'Unavailable'}</dd></div>
        </dl>
      </div>
      <form onSubmit={save} className="card space-y-4">
        <h2 className="font-semibold text-slate-900">Curriculum</h2>
        <div>
          <label className="label">Education level</label>
          <select className="input" value={levelId} onChange={(event) => { setLevelId(event.target.value); setClassId('') }}>
            <option value="">Choose…</option>
            {levels.map((level) => <option key={level.id} value={level.id}>{level.name}</option>)}
          </select>
        </div>
        <div>
          <label className="label">Class</label>
          <select className="input" disabled={!levelId} value={classId} onChange={(event) => setClassId(event.target.value)}>
            <option value="">Choose…</option>
            {classes.map((classLevel) => <option key={classLevel.id} value={classLevel.id}>{classLevel.name}</option>)}
          </select>
        </div>
        <ErrorBanner message={error} />
        {message && <p className="text-sm font-medium text-emerald-700">{message}</p>}
        <button className="btn-primary" disabled={saving}>{saving ? 'Saving…' : 'Save changes'}</button>
      </form>
    </div>
  )
}
