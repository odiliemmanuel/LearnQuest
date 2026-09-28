import { useState } from 'react'
import { Link, useLocation, useNavigate } from 'react-router-dom'
import { api } from '../lib/api'
import { useAuth } from '../context/AuthContext'
import { ErrorBanner } from '../components/ui'
import { AuthShell } from './Login'

export default function VerifyEmail() {
  const location = useLocation()
  const navigate = useNavigate()
  const { verifyEmail } = useAuth()
  const initialEmail = (location.state as { email?: string } | null)?.email ?? ''
  const [email, setEmail] = useState(initialEmail)
  const [code, setCode] = useState('')
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const [busy, setBusy] = useState(false)
  const [resending, setResending] = useState(false)

  async function verify(event: React.FormEvent) {
    event.preventDefault()
    setError('')
    setBusy(true)
    try {
      await verifyEmail(email, code)
      navigate('/dashboard', { replace: true })
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Could not verify your email.')
    } finally {
      setBusy(false)
    }
  }

  async function resend() {
    if (!email) {
      setError('Enter the email address you used to register.')
      return
    }
    setError('')
    setNotice('')
    setResending(true)
    try {
      await api.post('/api/v1/auth/resend-verification', { email })
      setNotice('A new verification code is on its way. Check your inbox and spam folder.')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Could not resend the code.')
    } finally {
      setResending(false)
    }
  }

  return (
    <AuthShell>
      <h1 className="text-2xl font-bold text-slate-900 dark:text-white">Verify your email</h1>
      <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">We sent a six-digit code to your email address. It expires in 10 minutes.</p>
      <form onSubmit={verify} className="mt-6 space-y-4">
        <div>
          <label className="label">Email</label>
          <input className="input" type="email" required value={email} onChange={(event) => setEmail(event.target.value)} placeholder="you@example.com" />
        </div>
        <div>
          <label className="label">Verification code</label>
          <input
            className="input text-center text-xl font-bold tracking-[0.45em]"
            inputMode="numeric"
            autoComplete="one-time-code"
            required
            maxLength={6}
            value={code}
            onChange={(event) => setCode(event.target.value.replace(/\D/g, '').slice(0, 6))}
            placeholder="000000"
          />
        </div>
        <ErrorBanner message={error} />
        {notice && <p className="rounded-xl bg-emerald-50 px-3 py-2 text-sm text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-300">{notice}</p>}
        <button className="btn-primary w-full" disabled={busy || code.length !== 6}>{busy ? 'Verifying…' : 'Verify email'}</button>
      </form>
      <div className="mt-5 flex items-center justify-between gap-3 text-sm">
        <button type="button" onClick={resend} disabled={resending} className="font-semibold text-indigo-600 hover:underline disabled:opacity-50 dark:text-indigo-300">{resending ? 'Sending…' : 'Resend code'}</button>
        <Link to="/login" className="text-slate-500 hover:text-indigo-600 dark:text-slate-400 dark:hover:text-indigo-300">Back to sign in</Link>
      </div>
    </AuthShell>
  )
}
