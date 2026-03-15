import { useState } from 'react';
import { useNavigate, useLocation } from 'react-router-dom';
import { api, auth, csrf } from '../api/client.js';

export default function LoginPage() {
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  
  const navigate = useNavigate();
  const location = useLocation();

  const handleLogin = async (e) => {
    e.preventDefault();
    setLoading(true);
    setError('');
    try {
      const res = await api.login(username, password);
      const { token } = res;
      if (token) {
        auth.setToken(token);
        if (res.csrf_token) csrf.setToken(res.csrf_token);
        const from = location.state?.from?.pathname || '/admin/studies';
        navigate(from, { replace: true });
      } else {
        setError('Login failed: Invalid response from server');
      }
    } catch (err) {
      setError(err.message || 'Login failed');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', minHeight: '80vh' }}>
      <form onSubmit={handleLogin} className="card fade-in" style={{ width: '100%', maxWidth: '400px', display: 'flex', flexDirection: 'column', gap: 'var(--space-4)' }}>
        <div style={{ textAlign: 'center', marginBottom: 'var(--space-2)' }}>
          <h1 style={{ fontSize: 'var(--font-size-2xl)' }}>Admin Login</h1>
          <p style={{ color: 'var(--color-text-muted)' }}>Enter your credentials to manage comparisons.</p>
        </div>
        
        {error && (
          <div style={{ color: 'var(--color-accent)', padding: 'var(--space-3)', background: 'rgba(255, 101, 132, 0.1)', border: '1px solid rgba(255, 101, 132, 0.2)', borderRadius: 'var(--radius-sm)' }}>
            {error}
          </div>
        )}
        
        <div style={{ display: 'flex', flexDirection: 'column' }}>
          <label className="label">Username</label>
          <input 
            type="text" 
            className="input" 
            value={username} 
            onChange={e => setUsername(e.target.value)} 
            required
            autoFocus 
          />
        </div>
        
        <div style={{ display: 'flex', flexDirection: 'column' }}>
          <label className="label">Password</label>
          <input 
            type="password" 
            className="input" 
            value={password} 
            onChange={e => setPassword(e.target.value)} 
            required 
          />
        </div>
        
        <button type="submit" className="btn btn-primary" disabled={loading} style={{ marginTop: 'var(--space-3)' }}>
          {loading ? 'Logging in...' : 'Login'}
        </button>
      </form>
    </div>
  );
}
