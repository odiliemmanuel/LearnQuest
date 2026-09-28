import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { api } from '../lib/api'
import type { Lesson as LessonType } from '../lib/types'
import { ErrorBanner, Spinner } from '../components/ui'

export default function Lesson() {
  const { lessonId } = useParams()
  const [lesson, setLesson] = useState<LessonType | null>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    api
      .get<LessonType>(`/api/v1/lessons/${lessonId}`)
      .then(setLesson)
      .catch(() => setError('Could not load this lesson.'))
  }, [lessonId])

  if (error) return <ErrorBanner message={error} />
  if (!lesson) return <Spinner />

  return (
    <div className="mx-auto max-w-3xl space-y-5">
      <div>
        <Link to={`/topics/${lesson.topicId}`} className="text-sm font-semibold text-indigo-600 hover:underline">
          ← Back to topic
        </Link>
        <div className="mt-2 text-xs font-medium uppercase tracking-wide text-indigo-500">
          {lesson.subjectName} · {lesson.className} · {lesson.termName}
        </div>
        <h1 className="mt-1 text-2xl font-bold text-slate-900">{lesson.title}</h1>
      </div>

      {lesson.introduction && (
        <div className="card border-l-4 border-l-indigo-500">
          <p className="text-sm leading-relaxed text-slate-700">{lesson.introduction}</p>
        </div>
      )}

      {lesson.formulas.length > 0 && (
        <div className="card">
          <h2 className="text-base font-bold text-slate-900">Formulas</h2>
          <div className="mt-3 space-y-3">
            {lesson.formulas.map((f, i) => (
              <div key={i} className="rounded-xl border border-slate-200 p-3 dark:border-slate-700">
                <div className="text-sm font-bold text-slate-800">{f.name}</div>
                <code className="mt-1 block rounded-lg bg-slate-100 px-3 py-1.5 text-sm text-indigo-700">{f.expression}</code>
                {f.meaning && <div className="mt-1 text-xs text-slate-500">{f.meaning}</div>}
              </div>
            ))}
          </div>
        </div>
      )}

      {lesson.explanation && (
        <div className="card">
          <h2 className="text-base font-bold text-slate-900">Explanation</h2>
          <div className="mt-2 whitespace-pre-line text-sm leading-relaxed text-slate-700">{lesson.explanation}</div>
        </div>
      )}

      {lesson.examples.length > 0 && (
        <div className="card">
          <h2 className="text-base font-bold text-slate-900">Worked examples</h2>
          <div className="mt-3 space-y-4">
            {lesson.examples.map((ex, i) => (
              <div key={i} className="rounded-xl border border-slate-200 p-3 dark:border-slate-700">
                <div className="font-semibold text-slate-800">
                  {i + 1}. {ex.title}
                </div>
                <div className="mt-1 text-sm text-slate-600">
                  <span className="font-semibold">Problem: </span>
                  {ex.problem}
                </div>
                <div className="mt-1 rounded-lg bg-emerald-50 p-2 text-sm text-emerald-800">
                  <span className="font-semibold">Solution: </span>
                  {ex.solution}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {lesson.keyPoints.length > 0 && (
        <div className="card">
          <h2 className="text-base font-bold text-slate-900">Key points</h2>
          <ul className="mt-2 list-inside list-disc space-y-1 text-sm text-slate-700">
            {lesson.keyPoints.map((k, i) => (
              <li key={i}>{k}</li>
            ))}
          </ul>
        </div>
      )}

      <div className="flex gap-3">
        <Link to={`/topics/${lesson.topicId}`} className="btn-primary">
          Start the quiz
        </Link>
      </div>
    </div>
  )
}
