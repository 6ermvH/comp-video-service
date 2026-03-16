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
          Инструкции
        </h1>

        {instructionsText ? (
          <div className="card" style={{ marginBottom: '24px', whiteSpace: 'pre-line', lineHeight: 1.8 }}>
            {instructionsText}
          </div>
        ) : (
          <div className="card" style={{ marginBottom: '24px' }}>
            <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>

              <section>
                <h3 style={{ fontSize: '17px', marginBottom: '10px', color: 'var(--color-primary-h)' }}>
                  Что вам предстоит делать
                </h3>
                <p style={{ color: 'var(--color-text-muted)', lineHeight: 1.8 }}>
                  Вы увидите пары коротких видеороликов с визуальными эффектами (наводнение, взрыв).
                  Видео <strong>A</strong> и <strong>B</strong> — два разных метода симуляции одного
                  и того же события. Выберите, какое из них выглядит <strong>более реалистично</strong>.
                </p>
              </section>

              <hr style={{ border: 'none', borderTop: '1px solid var(--color-border)' }} />

              <section>
                <h3 style={{ fontSize: '17px', marginBottom: '10px', color: 'var(--color-primary-h)' }}>
                  Как оценивать
                </h3>
                <ul style={{ paddingLeft: '20px', color: 'var(--color-text-muted)', lineHeight: 2 }}>
                  <li>Смотрите на <strong>реалистичность движения</strong> частиц / воды / огня</li>
                  <li>Обращайте внимание на <strong>артефакты и дёргание</strong></li>
                  <li>Оценивайте <strong>общее визуальное качество</strong></li>
                  <li>Если оба видео кажутся одинаковыми — выберите «<strong>Равны</strong>»</li>
                </ul>
              </section>

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

              <hr style={{ border: 'none', borderTop: '1px solid var(--color-border)' }} />

              <section>
                <h3 style={{ fontSize: '17px', marginBottom: '10px', color: 'var(--color-primary-h)' }}>
                  Повтор
                </h3>
                <p style={{ color: 'var(--color-text-muted)', lineHeight: 1.8 }}>
                  Вы можете повторить просмотр неограниченное количество раз. Пожалуйста,
                  не спешите — качество ваших оценок важнее скорости.
                </p>
              </section>

            </div>
          </div>
        )}

        <div style={{ textAlign: 'center' }}>
          <button
            className="btn btn-primary"
            onClick={() => navigate('/practice')}
            style={{ padding: '14px 40px', fontSize: '16px', width: '100%' }}
          >
            Понятно, начнём тренировку →
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
