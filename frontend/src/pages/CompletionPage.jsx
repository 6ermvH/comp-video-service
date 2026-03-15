import { useSession } from '../context/SessionContext.jsx'
import { useNavigate } from 'react-router-dom'
import { useEffect } from 'react'

export default function CompletionPage() {
  const navigate = useNavigate()
  const { completionCode, clearSession } = useSession()

  // If somehow arrived here without a completion code, redirect home
  useEffect(() => {
    if (!completionCode) {
      navigate('/', { replace: true })
    }
  }, [completionCode, navigate])

  const handleCopy = () => {
    if (completionCode) navigator.clipboard.writeText(completionCode)
  }

  return (
    <div style={{
      minHeight: '100vh',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      padding: '24px',
      background: 'var(--color-bg)',
    }}>
      <div style={{ maxWidth: '520px', width: '100%', textAlign: 'center' }}>

        <div style={{ fontSize: '72px', marginBottom: '24px' }}>🎉</div>

        <h1 style={{ fontSize: '28px', fontWeight: 700, marginBottom: '16px' }}>
          Спасибо за участие!
        </h1>

        <p style={{ color: 'var(--color-text-muted)', lineHeight: 1.8, marginBottom: '32px' }}>
          Вы успешно завершили исследование. Ваши ответы очень помогут нам
          улучшить качество алгоритмов симуляции.
        </p>

        {completionCode && (
          <div className="card" style={{ marginBottom: '32px' }}>
            <p style={{ fontSize: '14px', color: 'var(--color-text-muted)', marginBottom: '12px' }}>
              Ваш код завершения:
            </p>
            <div style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              gap: '12px',
            }}>
              <code style={{
                fontSize: '22px',
                fontWeight: 700,
                letterSpacing: '0.12em',
                color: 'var(--color-success)',
                fontFamily: 'monospace',
                padding: '8px 16px',
                background: 'rgba(67,217,139,0.1)',
                border: '1px solid rgba(67,217,139,0.2)',
                borderRadius: 'var(--radius-sm)',
              }}>
                {completionCode}
              </code>
              <button
                className="btn btn-ghost"
                onClick={handleCopy}
                title="Скопировать код"
              >
                📋 Копировать
              </button>
            </div>
            <p style={{ fontSize: '13px', color: 'var(--color-text-muted)', marginTop: '12px' }}>
              Сохраните этот код — он подтверждает ваше участие.
            </p>
          </div>
        )}

        <button
          className="btn btn-ghost"
          onClick={() => { clearSession(); navigate('/') }}
        >
          Начать новую сессию
        </button>
      </div>
    </div>
  )
}
