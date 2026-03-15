import { useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { useSession } from '../context/SessionContext.jsx'

const ROLES = [
  { value: 'general_viewer',   label: 'Обычный зритель' },
  { value: 'ml_practitioner',  label: 'ML / AI-специалист' },
  { value: 'vfx_artist',       label: 'VFX / CGI-художник' },
  { value: 'researcher',       label: 'Исследователь' },
  { value: 'other',            label: 'Другое' },
]

const EXPERIENCE = [
  { value: 'none',     label: 'Нет опыта' },
  { value: 'limited',  label: 'Небольшой опыт' },
  { value: 'moderate', label: 'Умеренный опыт' },
  { value: 'strong',   label: 'Большой опыт' },
]

export default function WelcomePage() {
  const [searchParams] = useSearchParams()
  const navigate = useNavigate()
  const { startSession, loading, error } = useSession()

  const [studyId, setStudyId] = useState(searchParams.get('study_id') || '')
  const [role, setRole] = useState('')
  const [experience, setExperience] = useState('')
  const [consent, setConsent] = useState(false)

  const deviceType = /Mobi|Android/i.test(navigator.userAgent) ? 'mobile' : 'desktop'

  const handleStart = async (e) => {
    e.preventDefault()
    if (!consent) return

    try {
      await startSession(studyId, { role, experience, deviceType })
      navigate('/instructions')
    } catch (_) { /* error shown via context */ }
  }

  const isValid = studyId.trim() && role && experience && consent

  return (
    <div style={{
      minHeight: '100vh',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      padding: '24px',
      background: 'var(--color-bg)',
    }}>
      <div style={{ maxWidth: '520px', width: '100%' }}>

        <div style={{ textAlign: 'center', marginBottom: '40px' }}>
          <h1 style={{ fontSize: '28px', fontWeight: 700, marginBottom: '12px' }}>
            Оценка качества видеоэффектов
          </h1>
          <p style={{ color: 'var(--color-text-muted)', lineHeight: 1.7 }}>
            Вам будут показаны пары видео. Ваша задача — оценить, какое из двух
            видео выглядит более реалистично и качественно.
          </p>
          <div style={{ display: 'flex', justifyContent: 'center', gap: '24px',
            marginTop: '16px', fontSize: '14px', color: 'var(--color-text-muted)' }}>
            <span>⏱ ~15–20 минут</span>
            <span>🖥 Только ПК / ноутбук</span>
            <span>🔊 Звук не требуется</span>
          </div>
        </div>

        <form onSubmit={handleStart} style={{ display: 'flex', flexDirection: 'column', gap: '20px' }}>

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

          <div>
            <label className="label">Ваша роль *</label>
            <select
              className="input"
              value={role}
              onChange={(e) => setRole(e.target.value)}
              required
            >
              <option value="">Выберите роль…</option>
              {ROLES.map((r) => (
                <option key={r.value} value={r.value}>{r.label}</option>
              ))}
            </select>
          </div>

          <div>
            <label className="label">Опыт работы с VFX / симуляциями *</label>
            <select
              className="input"
              value={experience}
              onChange={(e) => setExperience(e.target.value)}
              required
            >
              <option value="">Выберите уровень…</option>
              {EXPERIENCE.map((e) => (
                <option key={e.value} value={e.value}>{e.label}</option>
              ))}
            </select>
          </div>

          <label style={{
            display: 'flex', gap: '12px', alignItems: 'flex-start',
            cursor: 'pointer', fontSize: '14px', lineHeight: 1.6,
          }}>
            <input
              type="checkbox"
              checked={consent}
              onChange={(e) => setConsent(e.target.checked)}
              style={{ marginTop: '3px', width: '16px', height: '16px', flexShrink: 0 }}
            />
            <span style={{ color: 'var(--color-text-muted)' }}>
              Я согласен(на) на участие в исследовании. Мои анонимные ответы будут
              использованы только для научных целей.
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
            style={{ padding: '16px', fontSize: '16px', marginTop: '8px' }}
          >
            {loading ? 'Запуск…' : 'Начать участие →'}
          </button>
        </form>
      </div>
    </div>
  )
}
