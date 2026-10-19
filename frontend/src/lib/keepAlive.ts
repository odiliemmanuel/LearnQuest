import { API_BASE } from './api'

// Render's free tier puts the backend to sleep after ~15 minutes without
// traffic, and the next request then takes 30-60s to wake it. While a tab has
// the app open we ping /health on load, every PING_INTERVAL_MS, and whenever
// the tab becomes visible again, so the backend stays warm.
const PING_INTERVAL_MS = 13 * 60 * 1000

export function pingHealth() {
  fetch(`${API_BASE}/health`, { headers: { Accept: 'application/json' } }).catch(() => {
    // Best-effort: waking the backend is all we need; a failure is harmless.
  })
}

export function startKeepAlive() {
  pingHealth()
  const timer = setInterval(pingHealth, PING_INTERVAL_MS)
  const onVisible = () => {
    if (document.visibilityState === 'visible') pingHealth()
  }
  document.addEventListener('visibilitychange', onVisible)
  return () => {
    clearInterval(timer)
    document.removeEventListener('visibilitychange', onVisible)
  }
}
