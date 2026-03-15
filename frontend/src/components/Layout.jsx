import { Outlet, Link } from 'react-router-dom';

export default function Layout() {
  return (
    <div style={{ display: 'flex', flexDirection: 'column', minHeight: '100vh' }}>
      <header style={{ 
        borderBottom: '1px solid var(--color-border)', 
        background: 'var(--color-surface)',
        padding: 'var(--space-4) 0'
      }}>
        <div className="container" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <Link to="/" style={{ fontSize: 'var(--font-size-xl)', fontWeight: 'bold', color: 'var(--color-primary)' }}>
            VideoCompare
          </Link>
          <Link to="/admin/login" className="btn btn-ghost" style={{ padding: 'var(--space-2) var(--space-4)' }}>
            Admin
          </Link>
        </div>
      </header>
      
      <main className="container" style={{ flex: 1, padding: 'var(--space-6) var(--space-5)' }}>
        <Outlet />
      </main>
      
      <footer style={{ 
        borderTop: '1px solid var(--color-border)', 
        padding: 'var(--space-5) 0',
        textAlign: 'center',
        color: 'var(--color-text-muted)',
        fontSize: 'var(--font-size-sm)'
      }}>
        &copy; {new Date().getFullYear()} VideoCompare Service
      </footer>
    </div>
  );
}
