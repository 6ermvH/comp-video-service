import { useEffect } from 'react'
import { Outlet, Link, useNavigate, useLocation, Navigate } from 'react-router-dom'
import { auth, setUnauthorizedHandler } from '../api/client.js'

const NAV_ITEMS = [
  { path: '/admin/studies',   label: 'Исследования', icon: '🔬' },
  { path: '/admin/pairs',     label: 'Пары и ассеты', icon: '🎬' },
  { path: '/admin/analytics', label: 'Аналитика',     icon: '📊' },
]

export default function AdminLayout() {
  const navigate = useNavigate()
  const location = useLocation()

  // Register handler so client.js can trigger logout on 401/403
  // (must be before any conditional returns — Rules of Hooks)
  useEffect(() => {
    setUnauthorizedHandler(() => {
      auth.clearToken()
      navigate('/admin/login', { replace: true })
    })
    return () => setUnauthorizedHandler(null)
  }, [navigate])

  if (!auth.isLoggedIn()) {
    return <Navigate to="/admin/login" replace state={{ from: location }} />
  }

  const handleLogout = () => {
    auth.clearToken()
    navigate('/admin/login')
  }

  return (
    <div style={{ display: 'flex', minHeight: '100vh', background: 'var(--color-bg)' }}>
      <aside style={{
        width: '240px',
        background: 'var(--color-surface)',
        borderRight: '1px solid var(--color-border)',
        padding: '24px 16px',
        display: 'flex',
        flexDirection: 'column',
        flexShrink: 0,
      }}>
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

      <main style={{ flex: 1, overflowY: 'auto' }}>
        <div style={{ padding: '32px', maxWidth: '1100px', margin: '0 auto' }}>
          <Outlet />
        </div>
      </main>
    </div>
  )
}
