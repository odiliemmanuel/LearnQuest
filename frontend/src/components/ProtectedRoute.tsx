import { Navigate, Outlet, useLocation } from 'react-router-dom'
import { getToken } from '../lib/api'
import { useAuth } from '../context/AuthContext'

export default function ProtectedRoute() {
  const { loading } = useAuth()
  const location = useLocation()

  if (loading) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <div className="text-sm text-slate-500">Loading…</div>
      </div>
    )
  }

  if (!getToken()) {
    return <Navigate to="/login" state={{ from: location }} replace />
  }

  return <Outlet />
}
