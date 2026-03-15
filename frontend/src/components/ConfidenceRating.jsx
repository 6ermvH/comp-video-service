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
      <p style={{ fontSize: '13px', color: 'var(--color-text-muted)', marginBottom: '10px' }}>
        Уверенность (необязательно):
      </p>
      <div style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
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
                width: '38px',
                height: '38px',
                border: `2px solid ${active ? 'var(--color-warning)' : 'var(--color-border)'}`,
                borderRadius: 'var(--radius-sm)',
                background: active ? 'rgba(240,180,41,0.15)' : 'var(--color-surface-2)',
                color: active ? 'var(--color-warning)' : 'var(--color-text-muted)',
                fontSize: '15px',
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
        {value && (
          <span style={{ fontSize: '13px', color: 'var(--color-text-muted)', marginLeft: '8px' }}>
            {LABELS[value]}
          </span>
        )}
      </div>
    </div>
  )
}
