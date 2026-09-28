import { useEffect, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { api } from '../lib/api'
import type { AIEnvelope, ExplainFeedback, Question, StartQuizResponse, TopicDetail as TopicDetailType } from '../lib/types'
import { ErrorBanner, MasteryBadge, Spinner } from '../components/ui'

export default function TopicDetail() {
  const { topicId } = useParams()
  const navigate = useNavigate()
  const [topic, setTopic] = useState<TopicDetailType | null>(null)
  const [questions, setQuestions] = useState<Question[]>([])
  const [count, setCount] = useState(5)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const [explain, setExplain] = useState<ExplainFeedback | null>(null)
  const [explainBusy, setExplainBusy] = useState(false)

  useEffect(() => {
    Promise.all([
      api.get<TopicDetailType>(`/api/v1/curriculum/topics/${topicId}`),
      api.get<Question[]>(`/api/v1/topics/${topicId}/questions`).catch(() => []),
    ])
      .then(([t, qs]) => {
        setTopic(t)
        setQuestions(qs)
        if (qs.length > 0) setCount(Math.min(5, qs.length))
      })
      .catch(() => setError('Could not load this topic.'))
  }, [topicId])

  async function startQuiz(recovery = false) {
    setBusy(true)
    setError('')
    try {
      const res = await api.post<StartQuizResponse>(`/api/v1/topics/${topicId}/quiz/start`, {
        questionCount: count,
        recovery,
      })
      const quizState = { quizId: res.quizId, questions: res.questions, topicName: res.topic.name }
      sessionStorage.setItem(`quiz:${res.quizId}`, JSON.stringify(quizState))
      navigate(`/quiz/${res.quizId}`, { state: quizState })
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Could not start the quiz.')
    } finally {
      setBusy(false)
    }
  }

  async function askExplain() {
    setExplainBusy(true)
    setError('')
    try {
      const res = await api.post<AIEnvelope<ExplainFeedback>>('/api/v1/ai/explain', { topicId: Number(topicId) })
      setExplain(res.data)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'AI explanation unavailable.')
    } finally {
      setExplainBusy(false)
    }
  }

  if (!topic && !error) return <Spinner />
  if (!topic) return <ErrorBanner message={error} />

  return (
    <div className="space-y-6">
      <div className="card">
        <div className="flex items-center justify-between">
          <div className="text-sm font-medium text-indigo-600">
            {topic.subject.name} · {topic.classLevel.name} · {topic.term.name}
          </div>
          <MasteryBadge state={topic.state} mastery={topic.mastery} />
        </div>
        <h1 className="mt-1 text-2xl font-bold text-slate-900">{topic.name}</h1>
        <p className="mt-2 text-sm text-slate-600">{topic.description || 'Master this topic with a lesson and a quiz.'}</p>

        <div className="mt-6 grid gap-3 sm:grid-cols-2">
          {topic.hasLesson && topic.lessonId && (
            <button onClick={() => navigate(`/lessons/${topic.lessonId}`)} className="btn-secondary">
              Read the lesson
            </button>
          )}
          {topic.hasLesson === false && (
            <div className="text-sm text-slate-400">A full lesson is not available yet for this topic.</div>
          )}

          {questions.length > 0 ? (
            <div className="flex flex-col gap-3 sm:flex-row sm:items-center">
              <select className="input !w-36" value={count} onChange={(e) => setCount(Number(e.target.value))}>
                {Array.from(new Set([Math.min(5, questions.length), Math.min(10, questions.length), questions.length]))
                  .filter((n) => n > 0)
                  .map((n) => (
                    <option key={n} value={n}>
                      {n} questions
                    </option>
                  ))}
              </select>
              <button className="btn-primary flex-1" disabled={busy} onClick={() => startQuiz(false)}>
                {busy ? 'Starting…' : 'Start quiz'}
              </button>
            </div>
          ) : (
            <div className="text-sm text-slate-400">No quiz questions are ready for this topic yet.</div>
          )}
        </div>
      </div>

      <div>
        <button className="btn-secondary" onClick={askExplain} disabled={explainBusy}>
          {explainBusy ? 'Asking AI…' : explain ? 'Regenerate AI explanation' : 'Get an AI explanation'}
        </button>
        <p className="mt-1 text-xs text-slate-400">A tutor-style explanation tailored to your class.</p>
        {explain && (
          <div className="card mt-3 space-y-3">
            <h3 className="text-lg font-bold text-slate-900">{explain.title || topic.name}</h3>
            <p className="text-sm leading-relaxed text-slate-700">{explain.explanation}</p>
            {explain.keyPoints.length > 0 && (
              <div>
                <div className="mb-1 text-sm font-semibold text-slate-700">Key points</div>
                <ul className="list-inside list-disc space-y-1 text-sm text-slate-600">
                  {explain.keyPoints.map((k, i) => (
                    <li key={i}>{k}</li>
                  ))}
                </ul>
              </div>
            )}
            {explain.example && (
              <div className="rounded-xl bg-indigo-50 p-3 text-sm text-slate-700">
                <span className="font-semibold text-indigo-700">Example: </span>
                {explain.example}
              </div>
            )}
          </div>
        )}
      </div>

      <ErrorBanner message={error} />
    </div>
  )
}
