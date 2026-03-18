/**
 * ConfidenceRating — 1-5 star-like scale for confidence in the choice.
 */
const LABELS = {
  1: 'Совсем не уверен',
  2: 'Слабая уверенность',
  3: 'Умеренная уверенность',
  4: 'Уверен',
  5: 'Очень уверен',
}

export default function ConfidenceRating({ value, onChange, disabled }) {
  return (
    <div>
      <p style={{ fontSize: '12px', color: 'var(--color-text-muted)', marginBottom: '4px' }}>
        Уверенность (необязательно):
      </p>
      <div style={{ display: 'flex', gap: '6px', alignItems: 'center', flexWrap: 'wrap' }}>
        {[1, 2, 3, 4, 5].map((n) => {
          const active = value === n
          return (
            <button
              key={n}
              onClick={() => !disabled && onChange(n)}
              disabled={disabled}
              title={LABELS[n]}
              aria-label={LABELS[n]}
              style={{
                width: '32px',
                height: '32px',
                border: `2px solid ${active ? 'var(--color-warning)' : 'var(--color-border)'}`,
                borderRadius: 'var(--radius-sm)',
                background: active ? 'rgba(240,180,41,0.15)' : 'var(--color-surface-2)',
                color: active ? 'var(--color-warning)' : 'var(--color-text-muted)',
                fontSize: '13px',
                fontWeight: 700,
                cursor: disabled ? 'not-allowed' : 'pointer',
                transition: 'all 0.15s ease',
                fontFamily: 'var(--font-family)',
              }}
            >
              {n}
            </button>
          )
        })}
      </div>
    </div>
  )
}
