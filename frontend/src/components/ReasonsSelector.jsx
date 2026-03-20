/**
 * ReasonsSelector — pick up to 2 reason tags explaining your choice.
 * Only shown after a choice (left/right) is made, not for "tie".
 */
const REASONS = [
  { code: 'motion',      label: 'Реализм движения объектов' },
  { code: 'artifacts',   label: 'Количество артефактов' },
  { code: 'overall',     label: 'Детализация изображения' },
  { code: 'integration', label: 'Визуальная целостность' },
]

const MAX_SELECTED = 2

export default function ReasonsSelector({ selected, onChange, disabled }) {
  const toggle = (code) => {
    if (disabled) return
    if (selected.includes(code)) {
      onChange(selected.filter((c) => c !== code))
    } else if (selected.length < MAX_SELECTED) {
      onChange([...selected, code])
    }
  }

  return (
    <div>
      <p style={{ fontSize: '12px', color: 'var(--color-text-muted)', marginBottom: '4px' }}>
        Что повлияло на ваш выбор? (необязательно, до {MAX_SELECTED})
      </p>
      <div style={{ display: 'flex', flexWrap: 'wrap', gap: '6px' }}>
        {REASONS.map((r) => {
          const active = selected.includes(r.code)
          const maxed = !active && selected.length >= MAX_SELECTED
          return (
            <button
              key={r.code}
              onClick={() => toggle(r.code)}
              disabled={disabled || maxed}
              style={{
                padding: '5px 10px',
                border: `1px solid ${active ? 'var(--color-primary)' : 'var(--color-border)'}`,
                borderRadius: '99px',
                background: active ? 'rgba(108,99,255,0.2)' : 'transparent',
                color: active ? 'var(--color-primary-h)' : maxed ? 'var(--color-text-muted)' : 'var(--color-text)',
                fontSize: '12px',
                cursor: disabled || maxed ? 'not-allowed' : 'pointer',
                transition: 'all 0.15s ease',
                opacity: maxed ? 0.5 : 1,
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
