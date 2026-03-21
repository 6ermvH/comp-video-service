/**
 * ReasonsSelector — pick reason tags explaining your choice.
 * Only shown after a choice (left/right) is made, not for "tie".
 */
const REASONS = [
  { code: 'motion',      label: 'Реализм движения объектов' },
  { code: 'artifacts',   label: 'Количество артефактов' },
  { code: 'overall',     label: 'Детализация изображения' },
  { code: 'integration', label: 'Визуальная целостность' },
]

export default function ReasonsSelector({ selected, onChange, disabled }) {
  const toggle = (code) => {
    if (disabled) return
    if (selected.includes(code)) {
      onChange(selected.filter((c) => c !== code))
    } else {
      onChange([...selected, code])
    }
  }

  return (
    <div>
      <p style={{ fontSize: '12px', color: 'var(--color-text-muted)', marginBottom: '4px' }}>
        Что повлияло на ваш выбор? (необязательно, можно выбрать несколько)
      </p>
      <div style={{ display: 'flex', flexWrap: 'wrap', gap: '6px' }}>
        {REASONS.map((r) => {
          const active = selected.includes(r.code)
          return (
            <button
              key={r.code}
              onClick={() => toggle(r.code)}
              disabled={disabled}
              style={{
                padding: '5px 10px',
                border: `1px solid ${active ? 'var(--color-primary)' : 'var(--color-border)'}`,
                borderRadius: '99px',
                background: active ? 'rgba(108,99,255,0.2)' : 'transparent',
                color: active ? 'var(--color-primary-h)' : 'var(--color-text)',
                fontSize: '12px',
                cursor: disabled ? 'not-allowed' : 'pointer',
                transition: 'all 0.15s ease',
                fontFamily: 'var(--font-family)',
              }}
            >
              {r.label}
            </button>
          )
        })}
      </div>
    </div>
  )
}
