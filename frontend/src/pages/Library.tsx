import { useEffect, useMemo, useState } from 'react'
import { api } from '../lib/api'
import type { LibrarySubject } from '../lib/types'
import { ErrorBanner, Spinner } from '../components/ui'

export default function Library() {
  const [subjects, setSubjects] = useState<LibrarySubject[]>([])
  const [activeSubject, setActiveSubject] = useState<string>('')
  const [activeTopic, setActiveTopic] = useState<string>('')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    api
      .get<LibrarySubject[]>('/api/v1/library')
      .then((data) => {
        setSubjects(data)
        const first = data.find((s) => s.count > 0)
        if (first) {
          setActiveSubject(first.name)
          setActiveTopic(first.notes[0].topic)
        }
      })
      .catch((err: unknown) => setError(err instanceof Error ? err.message : 'Could not load the library.'))
      .finally(() => setLoading(false))
  }, [])

  const activeNotes = useMemo(() => subjects.find((s) => s.name === activeSubject)?.notes ?? [], [subjects, activeSubject])
  const note = useMemo(() => activeNotes.find((n) => n.topic === activeTopic), [activeNotes, activeTopic])

  const chapterNav = useMemo(() => {
    const idx = activeNotes.findIndex((n) => n.topic === activeTopic)
    if (idx === -1) return { prev: undefined, next: undefined, position: 0 }
    return {
      prev: idx > 0 ? activeNotes[idx - 1] : undefined,
      next: idx < activeNotes.length - 1 ? activeNotes[idx + 1] : undefined,
      position: idx + 1,
    }
  }, [activeNotes, activeTopic])

  if (loading) return <Spinner label="Opening the library…" />

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-slate-900 dark:text-white">Library</h1>
        <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">
          Condensed study notes for every subject and topic — your personal textbook, written for LearnQuest.
        </p>
      </div>

      <ErrorBanner message={error} />

      {subjects.length === 0 ? (
        <div className="rounded-3xl border-2 border-dashed border-slate-200 p-10 text-center dark:border-slate-700">
          <h3 className="font-bold text-slate-900 dark:text-white">No notes yet</h3>
          <p className="mt-1 text-sm text-slate-600 dark:text-slate-300">The library is still being written. Check back soon.</p>
        </div>
      ) : (
        <div className="grid gap-6 lg:grid-cols-[260px_1fr]">
          <nav className="card h-fit p-2 lg:sticky lg:top-24">
            <div className="px-3 py-2 text-xs font-bold uppercase tracking-wide text-slate-400">Subjects</div>
            <ul className="space-y-1">
              {subjects.filter((s) => s.count > 0).map((subject) => (
                <li key={subject.name}>
                  <button
                    onClick={() => {
                      setActiveSubject(subject.name)
                      setActiveTopic(subject.notes[0].topic)
                    }}
                    className={`flex w-full items-center justify-between gap-2 rounded-xl px-3 py-2 text-left text-sm font-medium transition ${
                      subject.name === activeSubject
                        ? 'bg-indigo-50 text-indigo-700 dark:bg-indigo-500/15 dark:text-indigo-300'
                        : 'text-slate-600 hover:bg-slate-100 hover:text-slate-900 dark:text-slate-300 dark:hover:bg-slate-800 dark:hover:text-white'
                    }`}
                  >
                    <span className="truncate">{subject.name}</span>
                    <span className="shrink-0 text-xs text-slate-400">{subject.count}</span>
                  </button>
                </li>
              ))}
            </ul>
          </nav>

          <section className="min-w-0">
            <div className="mb-4 flex flex-wrap items-center gap-2">
              {activeNotes.map((n) => (
                <button
                  key={n.id}
                  onClick={() => setActiveTopic(n.topic)}
                  className={`rounded-full px-3 py-1 text-xs font-semibold transition ${
                    n.topic === activeTopic
                      ? 'bg-indigo-600 text-white'
                      : 'bg-slate-100 text-slate-600 hover:bg-slate-200 dark:bg-slate-800 dark:text-slate-300 dark:hover:bg-slate-700'
                  }`}
                >
                  {n.topic}
                </button>
              ))}
            </div>

            {note ? (
              <article className="relative overflow-hidden rounded-3xl border border-slate-200 bg-white shadow-sm dark:border-slate-800 dark:bg-slate-900">
                <div className="absolute inset-x-0 top-0 h-1.5 bg-gradient-to-r from-indigo-500 via-violet-500 to-fuchsia-500" />
                <div className="px-6 py-8 sm:px-10 sm:py-10">
                  <div className="text-xs font-bold uppercase tracking-wide text-indigo-600 dark:text-indigo-300">
                    {note.subject} · {note.topic}
                  </div>
                  <h2 className="mt-2 text-2xl font-bold leading-tight text-slate-900 dark:text-white sm:text-3xl">{note.title}</h2>
                  <div className="mt-6 space-y-4 border-t border-slate-100 pt-6 dark:border-slate-800">
                    {note.content.split(/\n\n+/).map((paragraph, index) => (
                      <p key={index} className="text-[15px] leading-relaxed text-slate-700 dark:text-slate-300">
                        {paragraph}
                      </p>
                    ))}
                  </div>
                  <div className="mt-10 flex items-center justify-between gap-3 border-t border-slate-100 pt-6 dark:border-slate-800">
                    {chapterNav.prev ? (
                      <button
                        onClick={() => setActiveTopic(chapterNav.prev!.topic)}
                        className="rounded-xl border border-slate-200 px-4 py-2 text-sm font-semibold text-slate-600 transition hover:bg-slate-100 dark:border-slate-700 dark:text-slate-300 dark:hover:bg-slate-800"
                      >
                        ← {chapterNav.prev.topic}
                      </button>
                    ) : (
                      <span />
                    )}
                    <span className="text-xs font-semibold text-slate-400">
                      {chapterNav.position} of {activeNotes.length}
                    </span>
                    {chapterNav.next ? (
                      <button
                        onClick={() => setActiveTopic(chapterNav.next!.topic)}
                        className="rounded-xl border border-slate-200 px-4 py-2 text-sm font-semibold text-slate-600 transition hover:bg-slate-100 dark:border-slate-700 dark:text-slate-300 dark:hover:bg-slate-800"
                      >
                        {chapterNav.next.topic} →
                      </button>
                    ) : (
                      <span />
                    )}
                  </div>
                </div>
              </article>
            ) : (
              <div className="rounded-3xl border-2 border-dashed border-slate-200 p-10 text-center text-sm text-slate-500 dark:border-slate-700 dark:text-slate-400">
                Pick a note to start reading.
              </div>
            )}
          </section>
        </div>
      )}
    </div>
  )
}