import { useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { useSession } from '../context/SessionContext.jsx'

export default function WelcomePage() {
  const [searchParams] = useSearchParams()
  const navigate = useNavigate()
  const { startSession, loading, error } = useSession()

  const [studyId, setStudyId] = useState(searchParams.get('study_id') || '')
  const [consent, setConsent] = useState(false)

  const deviceType = /Mobi|Android/i.test(navigator.userAgent) ? 'mobile' : 'desktop'

  const handleStart = async (e) => {
    e.preventDefault()
    if (!consent) return

    try {
      await startSession(studyId, { role: '', experience: '', deviceType })
      navigate('/instructions')
    } catch (_) { /* error shown via context */ }
  }

  const isValid = studyId.trim() && consent
  const introWidth = 'min(620px, 100%)'
  const introFontSize = '16px'

  return (
    <div style={{
      minHeight: '100vh',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      padding: 'clamp(16px, 4vw, 24px)',
      background: 'var(--color-bg)',
    }}>
      <div style={{ maxWidth: 'min(520px, 100%)', width: '100%' }}>

        <div style={{ textAlign: 'center', marginBottom: '24px' }}>
          <h1 style={{ fontSize: '28px', fontWeight: 700, marginBottom: '28px', marginTop: '-40px' }}>
            Оценка качества генерации видео
          </h1>
          <p style={{
            color: 'var(--color-text-muted)',
            fontSize: introFontSize,
            lineHeight: 1.7,
            textAlign: 'left',
            maxWidth: introWidth,
            margin: '0 auto',
          }}>
            В данном исследовании будут представлены пары видео, сгенерированные
            разными способами, но имеющие одинаковое исходное изображение.
            Необходимо выбрать лучшее на ваш взгляд видео в каждой паре.
          </p>
          <p style={{
            color: 'var(--color-text-muted)',
            fontSize: introFontSize,
            lineHeight: 1.7,
            textAlign: 'left',
            maxWidth: introWidth,
            margin: '12px auto 0',
          }}>
            Участие является добровольным, исследование полностью анонимно.
            Полученные данные будут использованы только для научных целей.
          </p>
          <div style={{
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            gap: '12px',
            marginTop: '12px',
            fontSize: introFontSize,
            lineHeight: 1.7,
            color: 'var(--color-text-muted)',
            width: introWidth,
            marginLeft: 'auto',
            marginRight: 'auto',
            alignItems: 'flex-start',
          }}>
            <span>Средняя продолжительность исследования — 15 минут.</span>
            <span>Звук не требуется.</span>
          </div>
        </div>

        <form onSubmit={handleStart} style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>

          {!searchParams.get('study_id') && (
            <div>
              <label className="label">ID исследования *</label>
              <input
                className="input"
                type="text"
                placeholder="Введите ID исследования"
                value={studyId}
                onChange={(e) => setStudyId(e.target.value)}
                required
              />
            </div>
          )}

          <label style={{
            display: 'flex', gap: '12px', alignItems: 'flex-start',
            cursor: 'pointer', fontSize: introFontSize, lineHeight: 1.7,
            marginTop: '18px',
          }}>
            <input
              type="checkbox"
              checked={consent}
              onChange={(e) => setConsent(e.target.checked)}
              style={{ marginTop: '3px', width: '16px', height: '16px', flexShrink: 0 }}
            />
            <span style={{ color: 'var(--color-text-muted)' }}>
              Я ознакомился(-ась) с информацией и согласен(-на) принять участие
              в исследовании.
            </span>
          </label>

          {error && (
            <div style={{
              padding: '12px 16px',
              background: 'rgba(255,77,109,0.1)',
              border: '1px solid rgba(255,77,109,0.3)',
              borderRadius: 'var(--radius-sm)',
              color: '#ff6584',
              fontSize: '14px',
            }}>
              {error}
            </div>
          )}

          <button
            type="submit"
            className="btn btn-primary"
            disabled={!isValid || loading}
            style={{ padding: '16px', fontSize: '16px', marginTop: '8px', width: '100%' }}
          >
            {loading ? 'Запуск…' : 'Далее'}
          </button>
        </form>
      </div>
    </div>
  )
}
