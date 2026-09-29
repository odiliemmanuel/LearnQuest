import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom'
import { AuthProvider } from './context/AuthContext'
import { ThemeProvider } from './context/ThemeProvider'
import ProtectedRoute from './components/ProtectedRoute'
import { AppLayout } from './components/AppLayout'
import AdminRoute from './components/AdminRoute'
import { AdminLayout } from './components/AdminLayout'
import Login from './pages/Login'
import Register from './pages/Register'
import VerifyEmail from './pages/VerifyEmail'
import Onboarding from './pages/Onboarding'
import Dashboard from './pages/Dashboard'
import AdminDashboard from './pages/AdminDashboard'
import AdminUsers from './pages/AdminUsers'
import Subjects from './pages/Subjects'
import Topics from './pages/Topics'
import TopicDetail from './pages/TopicDetail'
import Lesson from './pages/Lesson'
import Quiz from './pages/Quiz'
import QuizResult from './pages/QuizResult'
import Recovery from './pages/Recovery'
import Progress from './pages/Progress'
import Challenges from './pages/Challenges'
import Profile from './pages/Profile'
import Library from './pages/Library'
import Landing from './pages/Landing'

export default function App() {
  return (
    <BrowserRouter>
      <ThemeProvider>
      <AuthProvider>
        <Routes>
          <Route path="/" element={<Landing />} />
          <Route path="/login" element={<Login />} />
          <Route path="/register" element={<Register />} />
          <Route path="/verify-email" element={<VerifyEmail />} />
          <Route element={<ProtectedRoute />}>
            <Route path="/onboarding" element={<Onboarding />} />
            <Route element={<AppLayout />}>
              <Route path="/dashboard" element={<Dashboard />} />
              <Route path="/subjects" element={<Subjects />} />
              <Route path="/subjects/:subjectId/topics" element={<Topics />} />
              <Route path="/topics/:topicId" element={<TopicDetail />} />
              <Route path="/lessons/:lessonId" element={<Lesson />} />
              <Route path="/quiz/:quizId" element={<Quiz />} />
              <Route path="/quiz/:quizId/result" element={<QuizResult />} />
              <Route path="/recovery" element={<Recovery />} />
              <Route path="/progress" element={<Progress />} />
              <Route path="/challenges" element={<Challenges />} />
              <Route path="/profile" element={<Profile />} />
              <Route path="/library" element={<Library />} />
            </Route>
          </Route>
          <Route element={<AdminRoute />}>
            <Route element={<AdminLayout />}>
              <Route path="/admin" element={<AdminDashboard />} />
              <Route path="/admin/users" element={<AdminUsers />} />
            </Route>
          </Route>
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </AuthProvider>
      </ThemeProvider>
    </BrowserRouter>
  )
}
