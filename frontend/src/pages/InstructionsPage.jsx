import { useNavigate } from 'react-router-dom'
import { useSession } from '../context/SessionContext.jsx'

export default function InstructionsPage() {
  const navigate = useNavigate()
  const { sessionToken, studyMeta } = useSession()
  const instructionsText = studyMeta?.instructions_text || ''

  // Guard: if no session, redirect to welcome
  if (!sessionToken) {
    navigate('/', { replace: true })
    return null
  }

  return (
    <div style={{
      minHeight: '100vh',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      padding: 'clamp(16px, 4vw, 24px)',
      background: 'var(--color-bg)',
    }}>
      <div style={{ maxWidth: 'min(680px, 100%)', width: '100%' }}>

        <h1 style={{ fontSize: '26px', fontWeight: 700, marginBottom: '24px', textAlign: 'center' }}>
          Инструкция
        </h1>

        {instructionsText ? (
          <div className="card" style={{ marginBottom: '24px', whiteSpace: 'pre-line', lineHeight: 1.8 }}>
            {instructionsText}
          </div>
        ) : (
          <div className="card" style={{ marginBottom: '24px' }}>
            <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
              <section>
                <ul style={{ paddingLeft: '20px', color: 'var(--color-text-muted)', lineHeight: 2 }}>
                  <li>Будут представлены видео А и B, выберите лучшее видео в целом.</li>
                  <li>
                    Обращайте внимание на следующие факторы:
                    <ul style={{ marginTop: '4px', paddingLeft: '22px', lineHeight: 1.9 }}>
                      <li>реализм</li>
                      <li>стабильность (плавность и согласованность движения)</li>
                      <li>отсутствие артефактов</li>
                      <li>общая визуальная целостность (видео выглядит как одна связная сцена)</li>
                    </ul>
                  </li>
                  <li>Если затрудняетесь определить лучшее видео, выберите «<strong>Затрудняюсь ответить</strong>»</li>
                  <li>Вы можете повторить просмотр неограниченное количество раз. Пожалуйста, не спешите — качество ваших оценок важнее скорости.</li>
                  <li>После подтверждения выбора и перехода к следующей паре видео, вы не сможете вернуться назад и изменить свой выбор.</li>
                </ul>
              </section>

              {/*
              <hr style={{ border: 'none', borderTop: '1px solid var(--color-border)' }} />

              <section>
                <h3 style={{ fontSize: '17px', marginBottom: '10px', color: 'var(--color-primary-h)' }}>
                  Управление
                </h3>
                <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(220px, 1fr))', gap: '8px',
                  fontSize: '14px', color: 'var(--color-text-muted)' }}>
                  <span><kbd style={kbdStyle}>1</kbd> — Видео A лучше</span>
                  <span><kbd style={kbdStyle}>2</kbd> — Видео B лучше</span>
                  <span><kbd style={kbdStyle}>0</kbd> — Равны</span>
                  <span><kbd style={kbdStyle}>R</kbd> — Повторить оба</span>
                  <span><kbd style={kbdStyle}>N</kbd> / Enter — Следующее</span>
                  <span><kbd style={kbdStyle}>Space</kbd> — Пауза / Воспроизведение</span>
                </div>
              </section>
              */}
            </div>
          </div>
        )}

        <div style={{ textAlign: 'center' }}>
          <button
            className="btn btn-primary"
            onClick={() => navigate('/practice')}
            style={{ padding: '14px 40px', fontSize: '16px', width: '100%' }}
          >
            Далее
          </button>
        </div>
      </div>
    </div>
  )
}

const kbdStyle = {
  display: 'inline-block',
  padding: '2px 8px',
  background: 'var(--color-surface-2)',
  border: '1px solid var(--color-border)',
  borderRadius: '4px',
  fontFamily: 'monospace',
  fontSize: '13px',
  color: 'var(--color-text)',
  marginRight: '6px',
}
