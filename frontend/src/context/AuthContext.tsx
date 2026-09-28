import { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react'
import { api, clearToken, getStoredUser, getToken, SESSION_EXPIRED_EVENT, setStoredUser, setToken } from '../lib/api'
import type { AuthResponse, RegisterResponse, StudentProfile, User } from '../lib/types'

interface AuthState {
  user: User | null
  profile: StudentProfile | null
  onboarded: boolean
  loading: boolean
  login: (email: string, password: string) => Promise<void>
  register: (name: string, email: string, password: string) => Promise<string>
  verifyEmail: (email: string, code: string) => Promise<void>
  logout: () => void
  refresh: () => Promise<void>
}

const AuthContext = createContext<AuthState | null>(null)

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [profile, setProfile] = useState<StudentProfile | null>(null)
  const [onboarded, setOnboarded] = useState(false)
  const [loading, setLoading] = useState(true)

  const applyAuth = useCallback((res: AuthResponse) => {
	setToken(res.token)
	setStoredUser(res.user)
    setUser(res.user)
    setOnboarded(res.onboarded)
  }, [])

  const refresh = useCallback(async () => {
    if (!getToken()) {
      setLoading(false)
      return
    }
    try {
      const data = await api.get<{ profile: StudentProfile; onboarded: boolean }>('/api/v1/student/profile')
      setProfile(data.profile)
      setOnboarded(data.onboarded)
		setUser(getStoredUser<User>())
    } catch {
      clearToken()
      setUser(null)
      setProfile(null)
      setOnboarded(false)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    refresh()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  useEffect(() => {
    function clearExpiredSession() {
      clearToken()
      setUser(null)
      setProfile(null)
      setOnboarded(false)
    }
    window.addEventListener(SESSION_EXPIRED_EVENT, clearExpiredSession)
    return () => window.removeEventListener(SESSION_EXPIRED_EVENT, clearExpiredSession)
  }, [])

  const login = useCallback(
    async (email: string, password: string) => {
      const res = await api.post<AuthResponse>('/api/v1/auth/login', { email, password })
      applyAuth(res)
      const data = await api.get<{ profile: StudentProfile; onboarded: boolean }>('/api/v1/student/profile')
      setProfile(data.profile)
      setOnboarded(data.onboarded)
    },
    [applyAuth],
  )

  const register = useCallback(
    async (name: string, email: string, password: string) => {
      const res = await api.post<RegisterResponse>('/api/v1/auth/register', { name, email, password })
      return res.email
    },
    [],
  )

  const verifyEmail = useCallback(
    async (email: string, code: string) => {
      const res = await api.post<AuthResponse>('/api/v1/auth/verify-email', { email, code })
      applyAuth(res)
      const data = await api.get<{ profile: StudentProfile; onboarded: boolean }>('/api/v1/student/profile')
      setProfile(data.profile)
      setOnboarded(data.onboarded)
    },
    [applyAuth],
  )

  const logout = useCallback(() => {
    clearToken()
    setUser(null)
    setProfile(null)
    setOnboarded(false)
  }, [])

  const value = useMemo(
		() => ({ user, profile, onboarded, loading, login, register, verifyEmail, logout, refresh }),
		[user, profile, onboarded, loading, login, register, verifyEmail, logout, refresh],
  )

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth(): AuthState {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used within AuthProvider')
  return ctx
}
