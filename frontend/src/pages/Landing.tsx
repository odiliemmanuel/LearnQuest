import { Link } from 'react-router-dom'
import { ThemeToggle } from '../components/ThemeToggle'

const features = [
  ['Instant feedback', 'Understand every answer while the question is still fresh.'],
  ['Curriculum-led practice', 'Study lessons and questions matched to your class and term.'],
  ['A visible learning path', 'Turn weak topics into strong ones with progress and recovery missions.'],
]

const steps = [
  ['01', 'Choose your class', 'Your lessons, questions, and explanations are matched to where you are.'],
  ['02', 'Learn and practise', 'Read a focused lesson, then take a short assessment.'],
  ['03', 'Get feedback now', 'See the correct working, teacher notes, and AI tutor feedback.'],
  ['04', 'Build mastery', 'Keep your streak, earn XP, and revisit topics that need attention.'],
]

export default function Landing() {
  return (
    <div className="min-h-screen bg-white text-slate-900 transition-colors dark:bg-slate-950 dark:text-slate-100">
      <header className="sticky top-0 z-20 border-b border-slate-100 bg-white/85 backdrop-blur-xl dark:border-slate-800 dark:bg-slate-950/85">
        <nav className="mx-auto flex max-w-7xl items-center justify-between px-5 py-4 sm:px-8">
          <Brand dark={false} />
          <div className="hidden items-center gap-7 text-sm font-medium text-slate-600 dark:text-slate-300 md:flex">
            <a href="#how-it-works" className="transition hover:text-indigo-700 dark:hover:text-indigo-300">How it works</a>
            <a href="#features" className="transition hover:text-indigo-700 dark:hover:text-indigo-300">Features</a>
          </div>
          <div className="flex items-center gap-3">
            <ThemeToggle />
            <Link to="/login" className="px-2 text-sm font-semibold text-slate-600 transition hover:text-indigo-700 dark:text-slate-300 dark:hover:text-indigo-300">Log in</Link>
            <Link to="/register" className="rounded-xl bg-gradient-to-br from-indigo-600 to-violet-800 px-4 py-2.5 text-sm font-semibold text-white shadow-lg shadow-indigo-600/20 transition hover:-translate-y-0.5 hover:shadow-indigo-600/30">
              Get started free
            </Link>
          </div>
        </nav>
      </header>

      <main>
        <section className="mx-auto grid max-w-7xl items-center gap-12 px-5 pb-24 pt-16 sm:px-8 lg:grid-cols-2 lg:gap-16 lg:pb-28 lg:pt-24">
          <div>
            <div className="inline-flex items-center gap-2 rounded-full bg-indigo-50 px-4 py-1.5 text-sm font-semibold text-indigo-700 dark:bg-indigo-500/15 dark:text-indigo-300">
              <span className="grid h-5 w-5 place-items-center rounded-full bg-indigo-600 text-[10px] text-white">*</span>
              Built for Nigerian secondary schools
            </div>
            <h1 className="mt-6 max-w-2xl text-5xl font-bold leading-[1.03] tracking-tight sm:text-6xl">
              Practice smarter.
              <span className="block bg-gradient-to-br from-indigo-600 to-violet-800 bg-clip-text text-transparent">Master each topic.</span>
            </h1>
            <p className="mt-6 max-w-xl text-lg leading-relaxed text-slate-600 dark:text-slate-300">
              LearnQuest gives students lessons, assessments, and clear feedback in one focused learning journey.
            </p>
            <div className="mt-8 flex flex-wrap gap-4">
              <Link to="/register" className="rounded-xl bg-gradient-to-br from-indigo-600 to-violet-800 px-6 py-3.5 font-semibold text-white shadow-xl shadow-indigo-600/20 transition hover:-translate-y-0.5">
                Start learning free <span aria-hidden="true">→</span>
              </Link>
              <a href="#how-it-works" className="rounded-xl border border-slate-200 px-6 py-3.5 font-semibold text-slate-700 transition hover:border-indigo-200 hover:bg-indigo-50 dark:border-slate-700 dark:text-slate-200 dark:hover:border-indigo-500/50 dark:hover:bg-slate-900">See how it works</a>
            </div>
            <div className="mt-9 flex flex-wrap gap-x-6 gap-y-2 text-sm text-slate-500 dark:text-slate-400">
              <span>Free to start</span><span>JSS1 to SS3</span><span>No card required</span>
            </div>
          </div>
          <HeroQuizMockup />
        </section>

        <section id="how-it-works" className="bg-slate-50 py-20 dark:bg-slate-900/50 sm:py-24">
          <div className="mx-auto max-w-7xl px-5 sm:px-8">
            <div className="mx-auto max-w-2xl text-center">
              <h2 className="text-3xl font-bold tracking-tight sm:text-4xl">How LearnQuest works</h2>
              <p className="mt-4 text-slate-600 dark:text-slate-300">Four practical steps from a difficult topic to confident practice.</p>
            </div>
            <div className="mt-14 grid gap-8 sm:grid-cols-2 lg:grid-cols-4">
              {steps.map(([number, title, description]) => (
                <article key={number}>
                  <span className="text-5xl font-bold tracking-tight text-indigo-100 dark:text-slate-800">{number}</span>
                  <h3 className="mt-3 text-lg font-bold">{title}</h3>
                  <p className="mt-2 text-sm leading-relaxed text-slate-600 dark:text-slate-400">{description}</p>
                </article>
              ))}
            </div>
          </div>
        </section>

        <section id="features" className="mx-auto max-w-7xl px-5 py-20 sm:px-8 sm:py-24">
          <div className="mx-auto max-w-2xl text-center">
            <h2 className="text-3xl font-bold tracking-tight sm:text-4xl">Everything needed to improve deliberately</h2>
            <p className="mt-4 text-slate-600 dark:text-slate-300">More than a quiz app: a clear loop for learning, feedback, and recovery.</p>
          </div>
          <div className="mt-14 grid gap-5 md:grid-cols-3">
            {features.map(([title, description], index) => (
              <article key={title} className="rounded-2xl border border-slate-200 bg-slate-50 p-6 transition hover:-translate-y-1 hover:border-indigo-200 hover:bg-white hover:shadow-lg dark:border-slate-800 dark:bg-slate-900 dark:hover:border-indigo-500/50 dark:hover:bg-slate-800">
                <div className="grid h-11 w-11 place-items-center rounded-xl bg-gradient-to-br from-indigo-600 to-violet-800 text-sm font-bold text-white">0{index + 1}</div>
                <h3 className="mt-5 text-lg font-bold">{title}</h3>
                <p className="mt-2 text-sm leading-relaxed text-slate-600 dark:text-slate-400">{description}</p>
              </article>
            ))}
          </div>
        </section>

        <section className="mx-auto max-w-7xl px-5 pb-20 sm:px-8 sm:pb-24">
          <div className="relative overflow-hidden rounded-[2rem] bg-gradient-to-br from-indigo-600 to-violet-900 px-7 py-14 text-center sm:px-12 sm:py-16">
            <div className="absolute -bottom-24 -left-20 h-72 w-72 rounded-full bg-white/10 blur-3xl" />
            <div className="relative mx-auto max-w-2xl">
              <h2 className="text-3xl font-bold tracking-tight text-white sm:text-4xl">Ready to learn with more clarity?</h2>
              <p className="mt-4 text-indigo-100">Create a free account and start your personalised learning journey.</p>
              <Link to="/register" className="mt-8 inline-block rounded-xl bg-white px-6 py-3.5 font-bold text-indigo-700 shadow-xl transition hover:-translate-y-0.5">Create your free account →</Link>
            </div>
          </div>
        </section>
      </main>

      <footer className="border-t border-slate-100 py-8 text-center text-sm text-slate-500 dark:border-slate-800 dark:text-slate-400">© 2026 LearnQuest. Built for Nigerian students.</footer>
    </div>
  )
}

function Brand({ dark }: { dark: boolean }) {
  return (
    <div className="flex items-center gap-2.5">
      <span className={`grid h-9 w-9 place-items-center rounded-xl bg-gradient-to-br from-indigo-600 to-violet-800 text-xs font-bold text-white ${dark ? 'ring-1 ring-white/20' : ''}`}>LQ</span>
      <span className={`text-xl font-bold tracking-tight ${dark ? 'text-white' : 'text-slate-900 dark:text-slate-100'}`}>LearnQuest</span>
    </div>
  )
}

function HeroQuizMockup() {
  return (
    <div className="relative mx-auto w-full max-w-xl">
      <div className="absolute -inset-5 rounded-[2.5rem] bg-gradient-to-br from-indigo-200/60 to-violet-200/50 blur-2xl dark:from-indigo-500/20 dark:to-violet-500/20" />
      <div className="relative rounded-3xl border border-slate-100 bg-white p-5 shadow-2xl shadow-indigo-950/15 dark:border-slate-800 dark:bg-slate-900 sm:p-7">
        <div className="flex items-center justify-between text-sm font-semibold"><span className="text-slate-400">SS2 · Physics · Waves</span><span className="text-indigo-600">Question 3 of 10</span></div>
        <p className="mt-6 text-lg font-semibold leading-snug">What happens to wave speed when frequency increases and wavelength stays constant?</p>
        <div className="mt-5 space-y-3 text-sm font-medium">
          <div className="flex items-center justify-between rounded-xl border-2 border-emerald-500 bg-emerald-50 p-4 text-emerald-800"><span>Wave speed increases</span><span className="grid h-6 w-6 place-items-center rounded-full bg-emerald-600 text-xs text-white">✓</span></div>
          <div className="rounded-xl border border-slate-200 p-4 text-slate-400 dark:border-slate-700">Wave speed decreases</div>
          <div className="rounded-xl border border-slate-200 p-4 text-slate-400 dark:border-slate-700">Wave speed stays the same</div>
        </div>
        <div className="mt-5 rounded-xl bg-indigo-50 p-4 text-sm leading-relaxed text-slate-600 dark:bg-indigo-500/15 dark:text-slate-300"><span className="font-bold text-indigo-700 dark:text-indigo-300">AI tutor: </span><span className="font-semibold text-slate-900 dark:text-white">Correct.</span> Since v = fλ, speed rises when frequency rises and wavelength does not change.</div>
      </div>
    </div>
  )
}
