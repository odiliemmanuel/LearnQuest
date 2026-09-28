import { useEffect, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'
import { api } from '../lib/api'
import type { GamificationSummary, Recommendation, SubjectProgress } from '../lib/types'
import { ProgressBar, Spinner } from '../components/ui'

export default function Dashboard() {
  const { user, profile } = useAuth()
  const navigate = useNavigate()
  const [summary, setSummary] = useState<GamificationSummary | null>(null)
  const [recommendations, setRecommendations] = useState<Recommendation[]>([])
  const [progress, setProgress] = useState<SubjectProgress[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    Promise.all([
      api.get<GamificationSummary>('/api/v1/gamification/summary').catch(() => null),
      api.get<Recommendation[]>('/api/v1/recommendations').catch(() => []),
      api.get<SubjectProgress[]>('/api/v1/progress').catch(() => []),
    ]).then(([nextSummary, nextRecommendations, nextProgress]) => {
      setSummary(nextSummary)
      setRecommendations(nextRecommendations)
      setProgress(nextProgress)
      setLoading(false)
    })
  }, [])

  if (loading) return <Spinner label="Building your learning space…" />

  const firstName = user?.name.split(' ')[0] ?? 'Student'
  const lead = recommendations[0]
  const answered = progress.reduce((total, item) => total + item.totalCount, 0)
  const mastered = progress.reduce((total, item) => total + item.topicsMastered, 0)

  return (
    <div className="space-y-8">
      <section className="relative overflow-hidden rounded-[2rem] bg-gradient-to-br from-indigo-600 via-indigo-700 to-violet-950 p-6 shadow-xl shadow-indigo-950/15 sm:p-8">
        <div className="absolute -right-16 -top-24 h-72 w-72 rounded-full bg-white/10 blur-3xl" />
        <div className="absolute -bottom-28 left-1/3 h-64 w-64 rounded-full bg-violet-300/15 blur-3xl" />
        <div className="relative flex flex-col justify-between gap-8 lg:flex-row lg:items-center">
          <div>
            <div className="text-sm font-semibold text-indigo-200">{profile?.classLevel?.name ?? 'Your personalised curriculum'}</div>
            <h1 className="mt-2 text-3xl font-bold tracking-tight text-white sm:text-4xl">Welcome back, {firstName}.</h1>
            <p className="mt-2 max-w-xl text-sm leading-relaxed text-indigo-100 sm:text-base">
              {summary?.streak ? `You are on a ${summary.streak}-day streak. Keep the momentum going with one focused session.` : 'A focused practice session today is the easiest way to start your learning streak.'}
            </p>
          </div>
          <div className="grid grid-cols-3 gap-2 sm:gap-3">
            <HeroStat value={summary?.level ?? 1} label="Level" />
            <HeroStat value={summary?.streak ?? 0} label="Day streak" />
            <HeroStat value={summary?.xp ?? 0} label="XP earned" />
          </div>
        </div>
      </section>

      <section className="grid gap-5 lg:grid-cols-3">
        <article className="relative overflow-hidden rounded-3xl border border-indigo-100 bg-gradient-to-br from-indigo-50 to-violet-50 p-6 dark:border-indigo-500/25 dark:from-indigo-500/15 dark:to-violet-500/10 lg:col-span-2">
          <div className="absolute -right-12 -bottom-20 h-48 w-48 rounded-full bg-indigo-200/50 blur-2xl dark:bg-indigo-500/15" />
          <div className="relative">
            <div className="flex items-center justify-between gap-3">
              <span className="rounded-full bg-white/80 px-3 py-1 text-xs font-bold uppercase tracking-wide text-indigo-700 shadow-sm dark:bg-slate-900/70 dark:text-indigo-300">Continue learning</span>
              {lead && <span className={`rounded-full px-2.5 py-1 text-xs font-bold ${lead.kind === 'RECOVERY' ? 'bg-rose-100 text-rose-700 dark:bg-rose-500/15 dark:text-rose-300' : 'bg-amber-100 text-amber-700 dark:bg-amber-500/15 dark:text-amber-300'}`}>{lead.kind === 'RECOVERY' ? 'Recovery mission' : 'Recommended'}</span>}
            </div>
            {lead ? (
              <>
                <div className="mt-5 text-sm font-semibold text-indigo-700 dark:text-indigo-300">{lead.topic.subject.name} · {lead.topic.className}</div>
                <h2 className="mt-1 text-2xl font-bold text-slate-900 dark:text-white">{lead.topic.name}</h2>
                <p className="mt-2 max-w-xl text-sm leading-relaxed text-slate-600 dark:text-slate-300">{lead.reason}</p>
                <button onClick={() => navigate(`/topics/${lead.topic.id}`)} className="btn-primary mt-5">Open learning plan <span aria-hidden="true">→</span></button>
              </>
            ) : (
              <>
                <h2 className="mt-5 text-2xl font-bold text-slate-900 dark:text-white">Find your next topic</h2>
                <p className="mt-2 max-w-xl text-sm leading-relaxed text-slate-600 dark:text-slate-300">Choose a subject, read a lesson, and take a short quiz to begin building your personalised learning plan.</p>
                <Link to="/subjects" className="btn-primary mt-5">Browse subjects <span aria-hidden="true">→</span></Link>
              </>
            )}
          </div>
        </article>

        <article className="card flex flex-col justify-between">
          <div>
            <div className="text-xs font-bold uppercase tracking-wide text-slate-400">Learning snapshot</div>
            <div className="mt-5 grid grid-cols-2 gap-4">
              <Snapshot value={answered} label="Questions answered" />
              <Snapshot value={mastered} label="Topics strong" />
            </div>
          </div>
          <Link to="/progress" className="mt-6 text-sm font-bold text-indigo-600 transition hover:text-indigo-700 dark:text-indigo-300 dark:hover:text-indigo-200">Open progress map →</Link>
        </article>
      </section>

      <section>
        <div className="mb-4 flex items-end justify-between gap-4">
          <div>
            <div className="text-xs font-bold uppercase tracking-wide text-slate-400">Your subjects</div>
            <h2 className="mt-1 text-xl font-bold text-slate-900 dark:text-white">Progress at a glance</h2>
          </div>
          <Link to="/subjects" className="text-sm font-semibold text-indigo-600 hover:underline dark:text-indigo-300">Browse all subjects</Link>
        </div>
        {progress.length === 0 ? (
          <div className="rounded-3xl border-2 border-dashed border-indigo-200 bg-indigo-50/50 p-8 text-center dark:border-indigo-500/30 dark:bg-indigo-500/10">
            <h3 className="font-bold text-slate-900 dark:text-white">Your first assessment starts the map</h3>
            <p className="mt-1 text-sm text-slate-600 dark:text-slate-300">Complete a quiz and this space will show mastery across your subjects.</p>
            <Link to="/subjects" className="btn-secondary mt-4">Choose a subject</Link>
          </div>
        ) : (
          <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
            {progress.map((item) => <SubjectProgressCard key={item.id} item={item} />)}
          </div>
        )}
      </section>

      <section className="grid gap-4 md:grid-cols-3">
        <ToolkitCard number="01" title="Explore subjects" description="Find lessons and assessments made for your class." to="/subjects" />
        <ToolkitCard number="02" title="Recover weak topics" description="Turn feedback into a targeted recovery mission." to="/recovery" />
        <ToolkitCard number="03" title="Take on challenges" description="Build your streak and collect more XP." to="/challenges" />
      </section>
    </div>
  )
}

function HeroStat({ value, label }: { value: number; label: string }) {
  return <div className="min-w-20 rounded-2xl bg-white/10 px-3 py-3 text-center backdrop-blur-sm ring-1 ring-white/10"><div className="text-lg font-bold text-white">{value}</div><div className="mt-0.5 text-[10px] font-medium text-indigo-100">{label}</div></div>
}

function Snapshot({ value, label }: { value: number; label: string }) {
  return <div><div className="text-2xl font-bold text-slate-900 dark:text-white">{value}</div><div className="mt-1 text-xs leading-tight text-slate-500 dark:text-slate-400">{label}</div></div>
}

function SubjectProgressCard({ item }: { item: SubjectProgress }) {
  const percentage = Math.round(item.percentage * 100)
  return (
    <Link to="/progress" className="card block transition hover:-translate-y-0.5 hover:border-indigo-300 hover:shadow-md dark:hover:border-indigo-500/50">
      <div className="flex items-start justify-between gap-4"><div><div className="text-xs font-bold uppercase tracking-wide text-indigo-600 dark:text-indigo-300">{item.topicsMastered}/{item.topicsTotal} topics strong</div><h3 className="mt-1 text-lg font-bold text-slate-900 dark:text-white">{item.subject?.name ?? 'Subject'}</h3></div><span className="text-lg font-bold text-slate-900 dark:text-white">{percentage}%</span></div>
      <ProgressBar value={item.percentage} className="mt-5" />
      <p className="mt-3 text-xs text-slate-500 dark:text-slate-400">{item.correctCount}/{item.totalCount} answers correct</p>
    </Link>
  )
}

function ToolkitCard({ number, title, description, to }: { number: string; title: string; description: string; to: string }) {
  return <Link to={to} className="group rounded-2xl border border-slate-200 bg-white p-5 transition hover:-translate-y-0.5 hover:border-indigo-300 hover:shadow-md dark:border-slate-800 dark:bg-slate-900 dark:hover:border-indigo-500/50"><div className="text-3xl font-bold text-indigo-100 dark:text-indigo-500/25">{number}</div><h3 className="mt-3 font-bold text-slate-900 dark:text-white">{title}</h3><p className="mt-1 text-sm leading-relaxed text-slate-500 dark:text-slate-400">{description}</p><div className="mt-4 text-sm font-bold text-indigo-600 dark:text-indigo-300">Open <span className="transition group-hover:ml-1">→</span></div></Link>
}
