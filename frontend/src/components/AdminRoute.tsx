import { Navigate, Outlet, useLocation } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'

export default function AdminRoute() {
  const { user, loading } = useAuth()
  const location = useLocation()

  if (loading) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <div className="text-sm text-slate-500">Loading…</div>
      </div>
    )
  }

  if (!user || user.role !== 'ADMIN') {
    return <Navigate to="/dashboard" state={{ from: location }} replace />
  }

  return <Outlet />
}