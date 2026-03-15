import { useState } from 'react'

const IS_MOBILE = /Mobi|Android|iPhone|iPad/i.test(navigator.userAgent)

/**
 * Shows an overlay warning on mobile devices (desktop-first study).
 * User can dismiss it to continue anyway.
 */
export default function MobileWarning() {
  const [dismissed, setDismissed] = useState(false)

  if (!IS_MOBILE || dismissed) return null

  return (
    <div style={{
      position: 'fixed', inset: 0, zIndex: 9999,
      background: 'rgba(13,15,20,0.97)',
      display: 'flex', alignItems: 'center', justifyContent: 'center',
      padding: '24px',
    }}>
      <div style={{ maxWidth: '380px', textAlign: 'center' }}>
        <div style={{ fontSize: '48px', marginBottom: '16px' }}>🖥️</div>
        <h2 style={{ fontSize: '20px', fontWeight: 700, marginBottom: '12px' }}>
          Откройте на компьютере
        </h2>
        <p style={{ color: 'var(--color-text-muted)', lineHeight: 1.7, marginBottom: '24px', fontSize: '14px' }}>
          Это исследование требует сравнения двух видео рядом. На мобильном устройстве
          этот процесс будет значительно сложнее. Пожалуйста, откройте ссылку на ПК
          или ноутбуке.
        </p>
        <button
          className="btn btn-ghost"
          onClick={() => setDismissed(true)}
          style={{ fontSize: '13px' }}
        >
          Всё равно продолжить
        </button>
      </div>
    </div>
  )
}
