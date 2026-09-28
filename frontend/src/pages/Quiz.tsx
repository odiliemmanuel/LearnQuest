import { useState } from 'react'
import { useLocation, useNavigate, useParams } from 'react-router-dom'
import { api } from '../lib/api'
import type { Question, SubmitAnswer, SubmitResult } from '../lib/types'
import { ErrorBanner } from '../components/ui'

interface QuizState {
  quizId: number
  questions: Question[]
  topicName?: string
}

function savedQuiz(quizId: string | undefined): QuizState | null {
  if (!quizId) return null
  try {
    return JSON.parse(sessionStorage.getItem(`quiz:${quizId}`) ?? 'null') as QuizState | null
  } catch {
    return null
  }
}

export default function Quiz() {
  const { quizId } = useParams()
  const location = useLocation()
  const navigate = useNavigate()
  const state = (location.state as QuizState | null) ?? savedQuiz(quizId)

  const [questions] = useState<Question[]>(state?.questions ?? [])
  const [selected, setSelected] = useState<Record<number, number>>({})
  const [calc, setCalc] = useState<Record<number, string>>({})
  const [working, setWorking] = useState<Record<number, string>>({})
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  const answered = questions.every(
    (q) => (q.type === 'MCQ' ? selected[q.id] !== undefined : (calc[q.id] ?? '').trim() !== ''),
  )

  async function submit() {
    if (!answered) return
    setBusy(true)
    setError('')
    const answers: SubmitAnswer[] = questions.map((q) =>
      q.type === 'MCQ'
        ? { questionId: q.id, selectedOptionId: selected[q.id] }
        : { questionId: q.id, answer: calc[q.id] ?? '', working: working[q.id] ?? '' },
    )
    try {
      const res = await api.post<SubmitResult>(`/api/v1/quizzes/${quizId}/submit`, { answers })
      sessionStorage.removeItem(`quiz:${quizId}`)
      navigate(`/quiz/${quizId}/result`, { state: { submitted: res }, replace: true })
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Could not submit quiz.')
    } finally {
      setBusy(false)
    }
  }

  if (questions.length === 0) {
    return <ErrorBanner message="This page needs the quiz to be started from the topic first. Please go back and start again." />
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-slate-900">{state?.topicName ?? 'Quiz'}</h1>
        <p className="mt-1 text-sm text-slate-500">
          {questions.length} questions · answer all to submit. Working is encouraged for calculation questions.
        </p>
      </div>

      <div className="space-y-5">
        {questions.map((q, i) => (
          <div key={q.id} className="card">
            <div className="flex items-center justify-between">
               <span className="rounded-full bg-indigo-50 px-2.5 py-0.5 text-xs font-bold text-indigo-600 dark:bg-indigo-500/15 dark:text-indigo-300">Q{i + 1}</span>
              <span className="text-xs text-slate-400">
                {q.marks} {q.marks === 1 ? 'mark' : 'marks'} · {q.type === 'CALCULATION' ? 'Calculation' : 'Multiple choice'}
              </span>
            </div>
            <div className="mt-2 font-medium text-slate-900">{q.prompt}</div>

            {q.type === 'MCQ' ? (
              <div className="mt-3 space-y-2">
                {q.options.map((o) => {
                  const active = selected[q.id] === o.id
                  return (
                    <button
                      key={o.id}
                      onClick={() => setSelected((s) => ({ ...s, [q.id]: o.id }))}
                      className={`flex w-full items-start gap-3 rounded-xl border px-3 py-2.5 text-left transition ${
                        active
                          ? 'border-indigo-500 bg-indigo-50 dark:bg-indigo-500/15'
                          : 'border-slate-200 bg-white hover:bg-slate-50 dark:border-slate-700 dark:bg-slate-900 dark:hover:bg-slate-800'
                      }`}
                    >
                      <span
                        className={`flex h-6 w-6 shrink-0 items-center justify-center rounded-full text-xs font-bold ${
                          active ? 'bg-indigo-600 text-white' : 'bg-slate-100 text-slate-500 dark:bg-slate-800 dark:text-slate-400'
                        }`}
                      >
                        {o.key}
                      </span>
                      <span className="text-sm text-slate-700">{o.text}</span>
                    </button>
                  )
                })}
              </div>
            ) : (
              <div className="mt-3 space-y-3">
                <input
                  className="input"
                  placeholder="Your answer (e.g. 6 m)"
                  value={calc[q.id] ?? ''}
                  onChange={(e) => setCalc((c) => ({ ...c, [q.id]: e.target.value }))}
                />
                <textarea
                  className="input"
                  rows={3}
                  placeholder="Show your working (optional, lets the AI review your steps)"
                  value={working[q.id] ?? ''}
                  onChange={(e) => setWorking((w) => ({ ...w, [q.id]: e.target.value }))}
                />
              </div>
            )}
          </div>
        ))}
      </div>

      <ErrorBanner message={error} />

      <div className="flex items-center justify-between gap-3">
        <span className="text-sm text-slate-500">{answered ? 'All questions answered.' : 'Answer every question to submit.'}</span>
        <button className="btn-primary" disabled={!answered || busy} onClick={submit}>
          {busy ? 'Submitting…' : `Submit quiz`}
        </button>
      </div>
    </div>
  )
}
