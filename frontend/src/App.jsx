import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { SessionProvider } from './context/SessionContext.jsx'
import { ToastProvider } from './context/ToastContext.jsx'
import AdminLayout from './components/AdminLayout.jsx'

import WelcomePage      from './pages/WelcomePage.jsx'
import InstructionsPage from './pages/InstructionsPage.jsx'
import PracticePage     from './pages/PracticePage.jsx'
import TaskPage         from './pages/TaskPage.jsx'
import BreakPage        from './pages/BreakPage.jsx'
import CompletionPage   from './pages/CompletionPage.jsx'

import LoginPage           from './pages/LoginPage.jsx'
import AdminStudiesPage    from './pages/AdminStudiesPage.jsx'
import AdminPairsPage      from './pages/AdminPairsPage.jsx'
import AdminAnalyticsPage  from './pages/AdminAnalyticsPage.jsx'

function App() {
  return (
    <ToastProvider>
    <SessionProvider>
      <BrowserRouter>
        <Routes>
          {/* ── Participant flow ── */}
          <Route path="/"            element={<WelcomePage />} />
          <Route path="/instructions" element={<InstructionsPage />} />
          <Route path="/practice"    element={<PracticePage />} />
          <Route path="/task"        element={<TaskPage />} />
          <Route path="/break"       element={<BreakPage />} />
          <Route path="/complete"    element={<CompletionPage />} />

          {/* ── Admin ── */}
          <Route path="/admin/login" element={<LoginPage />} />
          <Route element={<AdminLayout />}>
            <Route path="/admin/studies"   element={<AdminStudiesPage />} />
            <Route path="/admin/pairs"     element={<AdminPairsPage />} />
            <Route path="/admin/analytics" element={<AdminAnalyticsPage />} />
          </Route>

          {/* ── Fallback ── */}
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </BrowserRouter>
    </SessionProvider>
    </ToastProvider>
  )
}

export default App
