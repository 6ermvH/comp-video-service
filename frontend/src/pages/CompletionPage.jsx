import { useSession } from '../context/SessionContext.jsx'
import { useNavigate } from 'react-router-dom'
import { useEffect } from 'react'

export default function CompletionPage() {
  const navigate = useNavigate()
  const { completionCode } = useSession()

  useEffect(() => {
    if (!completionCode) {
      navigate('/', { replace: true })
    }
  }, [completionCode, navigate])

  return (
    <div style={{
      minHeight: '100vh',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      padding: 'clamp(16px, 4vw, 24px)',
      background: 'var(--color-bg)',
    }}>
      <div style={{ maxWidth: 'min(520px, 100%)', width: '100%', textAlign: 'center' }}>
        <div style={{ fontSize: '72px', marginBottom: '24px' }}>🎉</div>
        <h1 style={{ fontSize: '28px', fontWeight: 700, marginBottom: '16px' }}>
          Спасибо за участие!
        </h1>
      </div>
    </div>
  )
}
