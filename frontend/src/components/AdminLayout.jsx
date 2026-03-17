import { useEffect, useState } from 'react'
import { Outlet, Link, useNavigate, useLocation, Navigate } from 'react-router-dom'
import { auth, setUnauthorizedHandler } from '../api/client.js'
import { useWindowWidth } from '../hooks/useWindowWidth.js'

const NAV_ITEMS = [
  { path: '/admin/studies',   label: 'Исследования', icon: '🔬' },
  { path: '/admin/pairs',     label: 'Пары и ассеты', icon: '🎬' },
  { path: '/admin/library',   label: 'Видеотека',     icon: '🎞️' },
  { path: '/admin/analytics', label: 'Аналитика',     icon: '📊' },
]

export default function AdminLayout() {
  const navigate = useNavigate()
  const location = useLocation()
  const isMobile = useWindowWidth() <= 768
  const [sidebarOpen, setSidebarOpen] = useState(false)

  // Register handler so client.js can trigger logout on 401/403
  // (must be before any conditional returns — Rules of Hooks)
  useEffect(() => {
    setUnauthorizedHandler(() => {
      auth.clearToken()
      navigate('/admin/login', { replace: true })
    })
    return () => setUnauthorizedHandler(null)
  }, [navigate])

  // Close sidebar on navigation
  useEffect(() => {
    setSidebarOpen(false)
  }, [location.pathname])

  if (!auth.isLoggedIn()) {
    return <Navigate to="/admin/login" replace state={{ from: location }} />
  }

  const handleLogout = () => {
    auth.clearToken()
    navigate('/admin/login')
  }

  const sidebarStyle = {
    width: '240px',
    background: 'var(--color-surface)',
    borderRight: '1px solid var(--color-border)',
    padding: '24px 16px',
    display: 'flex',
    flexDirection: 'column',
    flexShrink: 0,
    ...(isMobile ? {
      position: 'fixed',
      top: 0,
      left: sidebarOpen ? 0 : '-240px',
      height: '100vh',
      zIndex: 200,
      transition: 'left 0.2s ease',
    } : {}),
  }

  return (
    <div style={{ display: 'flex', minHeight: '100vh', background: 'var(--color-bg)' }}>

      {/* Mobile overlay */}
      {isMobile && sidebarOpen && (
        <div
          onClick={() => setSidebarOpen(false)}
          style={{ position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.5)', zIndex: 199 }}
        />
      )}

      <aside style={sidebarStyle}>
        <div style={{ marginBottom: '32px', paddingLeft: '8px' }}>
          <Link to="/" style={{
            fontSize: '15px', fontWeight: 700, color: 'var(--color-text)',
            textDecoration: 'none', letterSpacing: '0.02em',
          }}>
            VideoCompare
          </Link>
          <div style={{ fontSize: '11px', color: 'var(--color-text-muted)', marginTop: '2px' }}>
            Admin Panel
          </div>
        </div>

        <nav style={{ display: 'flex', flexDirection: 'column', gap: '4px', flex: 1 }}>
          {NAV_ITEMS.map(({ path, label, icon }) => {
            const active = location.pathname.startsWith(path)
            return (
              <Link
                key={path}
                to={path}
                style={{
                  display: 'flex', alignItems: 'center', gap: '10px',
                  padding: '10px 12px', borderRadius: 'var(--radius-sm)',
                  textDecoration: 'none', fontSize: '14px', fontWeight: active ? 600 : 400,
                  color: active ? 'var(--color-text)' : 'var(--color-text-muted)',
                  background: active ? 'var(--color-surface-2)' : 'transparent',
                  transition: 'all 0.15s ease',
                }}
              >
                <span>{icon}</span>
                <span>{label}</span>
              </Link>
            )
          })}
        </nav>

        <button
          onClick={handleLogout}
          style={{
            display: 'flex', alignItems: 'center', gap: '8px',
            padding: '10px 12px', borderRadius: 'var(--radius-sm)',
            background: 'none', border: 'none', cursor: 'pointer',
            fontSize: '14px', color: 'var(--color-text-muted)',
            width: '100%', textAlign: 'left', transition: 'color 0.15s ease',
          }}
          onMouseEnter={(e) => e.currentTarget.style.color = 'var(--color-accent)'}
          onMouseLeave={(e) => e.currentTarget.style.color = 'var(--color-text-muted)'}
        >
          <span>🚪</span>
          <span>Выйти</span>
        </button>
      </aside>

      <main style={{ flex: 1, overflowY: 'auto', overflowX: 'hidden' }}>
        {isMobile && (
          <div style={{
            display: 'flex', alignItems: 'center', gap: '12px',
            padding: '12px 16px',
            borderBottom: '1px solid var(--color-border)',
            background: 'var(--color-surface)',
            position: 'sticky', top: 0, zIndex: 100,
          }}>
            <button
              onClick={() => setSidebarOpen((v) => !v)}
              style={{ background: 'none', border: 'none', fontSize: '20px', cursor: 'pointer', color: 'var(--color-text)', lineHeight: 1 }}
            >
              ☰
            </button>
            <span style={{ fontSize: '15px', fontWeight: 600 }}>VideoCompare Admin</span>
          </div>
        )}
        <div style={{ padding: isMobile ? '16px' : '32px', maxWidth: '1100px', margin: '0 auto' }}>
          <Outlet />
        </div>
      </main>
    </div>
  )
}
