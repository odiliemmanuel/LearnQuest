export class ApiError extends Error {
  status: number
  constructor(message: string, status: number) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}

const TOKEN_KEY = 'lq_token'
const USER_KEY = 'lq_user'
export const SESSION_EXPIRED_EVENT = 'lq:session-expired'

export function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY)
}

export function setToken(token: string) {
  localStorage.setItem(TOKEN_KEY, token)
}

export function getStoredUser<T>(): T | null {
  try {
    return JSON.parse(localStorage.getItem(USER_KEY) ?? 'null') as T | null
  } catch {
    return null
  }
}

export function setStoredUser(user: unknown) {
  localStorage.setItem(USER_KEY, JSON.stringify(user))
}

export function clearToken() {
	localStorage.removeItem(TOKEN_KEY)
	localStorage.removeItem(USER_KEY)
}

type Body = Record<string, unknown> | unknown[]

interface Envelope {
  success?: boolean
  data?: unknown
  error?: string
}

async function request<T>(method: string, path: string, body?: Body): Promise<T> {
  const headers: Record<string, string> = { 'Content-Type': 'application/json' }
  const token = getToken()
  if (token) headers.Authorization = `Bearer ${token}`

  let res: Response
  try {
    res = await fetch(path, {
      method,
      headers,
      body: body !== undefined ? JSON.stringify(body) : undefined,
    })
  } catch {
    throw new ApiError('Cannot reach the server. Is it running?', 0)
  }

  let envelope: Envelope = {}
  try {
    envelope = (await res.json()) as Envelope
  } catch {
    /* non-JSON response */
  }

  if (!res.ok || !envelope.success) {
    if (res.status === 401) {
      clearToken()
      window.dispatchEvent(new Event(SESSION_EXPIRED_EVENT))
    }
    throw new ApiError(envelope.error || (envelope.success === false ? 'Request failed.' : `HTTP ${res.status}`), res.status)
  }
  return envelope.data as T
}

export const api = {
  get: <T>(path: string) => request<T>('GET', path),
  post: <T>(path: string, body?: Body) => request<T>('POST', path, body),
  put: <T>(path: string, body?: Body) => request<T>('PUT', path, body),
  delete: <T>(path: string) => request<T>('DELETE', path),
}
