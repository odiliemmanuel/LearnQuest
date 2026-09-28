import { useCallback, useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { api } from '../lib/api'
import type { AnswerFeedback, QuizResultResponse } from '../lib/types'
import { ErrorBanner, Spinner } from '../components/ui'

export default function QuizResult() {
  const { quizId } = useParams()
  const [res, setRes] = useState<QuizResultResponse | null>(null)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(true)

  const fetchResult = useCallback(async () => {
    const data = await api.get<QuizResultResponse>(`/api/v1/quizzes/${quizId}/result`)
    setRes(data)
    return data
  }, [quizId])

  useEffect(() => {
    let stopped = false
    ;(async () => {
      try {
        let data = await fetchResult()
        // Poll a few times while the AI feedback is still generating.
        for (let i = 0; i < 5 && data.feedback.some((f) => f.ai.status === 'PENDING'); i++) {
          await new Promise((r) => setTimeout(r, 2000))
          if (stopped) return
          data = await fetchResult()
        }
        setLoading(false)
      } catch (err) {
        if (!stopped) {
          setError(err instanceof Error ? err.message : 'Could not load results.')
          setLoading(false)
        }
      }
    })()
    return () => {
      stopped = true
    }
  }, [fetchResult])

  if (loading) return <Spinner label="Loading results…" />
  if (!res) return <ErrorBanner message={error} />

  const score = Math.round(res.quiz.score * 100) / 100
  const passed = res.quiz.score >= 60
  return (
    <div className="space-y-6">
      <div className={`card text-center ${passed ? 'border-emerald-200' : 'border-rose-200'}`}>
        <div className={`mx-auto flex h-20 w-20 items-center justify-center rounded-full text-2xl font-bold ${
          passed ? 'bg-emerald-100 text-emerald-700' : 'bg-rose-100 text-rose-700'
        }`}>
          {Math.round(score)}%
        </div>
        <h1 className="mt-3 text-2xl font-bold text-slate-900">{passed ? 'Well done!' : 'Keep practising'}</h1>
        <p className="mt-1 text-sm text-slate-500">
          {res.className} · {res.subject.name} · {res.topic.name}
        </p>
        <p className="mt-2 text-sm text-slate-600">
          {res.quiz.correctCount} of {res.quiz.totalCount} correct · {res.quiz.scoreMarks}/{res.quiz.totalMarks} marks
        </p>
        <div className="mt-5 flex flex-wrap items-center justify-center gap-3">
          <Link to={`/topics/${res.topic.id}`} className="btn-secondary">
            Review topic
          </Link>
          {!passed && (
            <Link to="/recovery" className="btn-primary">
              Try recovery mission
            </Link>
          )}
        </div>
      </div>

      <ErrorBanner message={error} />

      <div className="space-y-4">
        <h2 className="text-lg font-bold text-slate-900">Question review</h2>
        {res.feedback.map((f, i) => (
          <div key={i} className={`card border-l-4 ${f.isCorrect ? 'border-l-emerald-500' : 'border-l-rose-500'}`}>
            <div className="flex items-center justify-between">
              <span className={`rounded-full px-2.5 py-0.5 text-xs font-bold ${
                f.isCorrect ? 'bg-emerald-100 text-emerald-700' : 'bg-rose-100 text-rose-700'
              }`}>
                {f.isCorrect ? 'Correct' : 'Incorrect'}
              </span>
              <span className="text-xs text-slate-400">{f.receivedMarks}/{f.question.marks} marks</span>
            </div>

            <div className="mt-2 font-medium text-slate-900">{f.question.prompt}</div>

            <div className="mt-3 space-y-2 text-sm">
              {f.question.type === 'MCQ' ? (
                <>
                  <div className="text-slate-600">
                    <span className="font-semibold">Your answer: </span>
                    {(f.question.options.find((o) => o.id === f.selectedOptionId)?.text ?? f.studentAnswer) || 'No answer'}
                  </div>
                  <div className="text-emerald-700">
                    <span className="font-semibold">Correct answer: </span>
                    {f.correctAnswer}
                  </div>
                </>
              ) : (
                <>
                  <div className="text-slate-600">
                    <span className="font-semibold">Your answer: </span>
                    {f.studentAnswer || 'No answer'}
                  </div>
                  <div className="text-emerald-700">
                    <span className="font-semibold">Correct answer: </span>
                    {f.correctAnswer}
                  </div>
                </>
              )}
              {f.explanation && (
                <div className="rounded-xl bg-slate-50 p-3 text-slate-600 dark:bg-slate-800">
                  <span className="font-semibold">Teacher note: </span>
                  {f.explanation}
                </div>
              )}
            </div>

            {!f.isCorrect && (
              <AiFeedbackCard ai={f.ai} working={f.working} />
            )}
          </div>
        ))}
      </div>
    </div>
  )
}

function AiFeedbackCard({ ai }: { ai: AnswerFeedback['ai']; working?: string }) {
  if (ai.status === 'PENDING') {
    return <div className="mt-3 rounded-xl bg-indigo-50 p-3 text-sm text-indigo-600">AI feedback is being prepared…</div>
  }
  if (ai.status === 'FAILED') {
    return (
      <div className="mt-3 rounded-xl bg-amber-50 p-3 text-sm text-amber-700">
        AI analysis is temporarily unavailable. The correct answer and teacher note above are still available.
      </div>
    )
  }
  if (ai.status === '' || !ai.feedback) return null

  const f = ai.feedback as Record<string, unknown>
  const text = (k: string) => (typeof f[k] === 'string' ? (f[k] as string) : '')

  return (
    <div className="mt-3 rounded-xl border border-indigo-200 bg-indigo-50/60 p-4 dark:border-indigo-500/30 dark:bg-indigo-500/10">
      <div className="mb-2 flex items-center gap-2 text-sm font-bold text-indigo-700">
        <span className="flex h-5 w-5 items-center justify-center rounded-full bg-indigo-600 text-[10px] text-white">AI</span>
        Tutor feedback
        {ai.status === 'FALLBACK' && <span className="text-[10px] font-medium text-amber-600">(simplified)</span>}
      </div>
      <div className="space-y-3 text-sm text-slate-700">
        {text('whatStudentDid') && (
          <div>
            <span className="font-semibold">What happened: </span>
            {text('whatStudentDid')}
          </div>
        )}
        {text('explanation') && (
          <div>
            <span className="font-semibold">Explanation: </span>
            {text('explanation')}
          </div>
        )}
        {text('correctWorking') && (
          <div className="whitespace-pre-line rounded-lg bg-white/70 p-3 dark:bg-slate-900/70">
            <span className="font-semibold">Correct working: </span>
            {text('correctWorking')}
          </div>
        )}
        {text('recommendedAction') && (
          <div className="text-indigo-700">
            <span className="font-semibold">Next step: </span>
            {text('recommendedAction')}
          </div>
        )}
      </div>
    </div>
  )
}
