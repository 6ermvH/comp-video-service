/**
 * ReasonsSelector — pick up to 2 reason tags explaining your choice.
 * Only shown after a choice (left/right) is made, not for "tie".
 */
const REASONS = [
  { code: 'motion',      label: 'Более реалистичное движение' },
  { code: 'particles',   label: 'Лучшее поведение частиц' },
  { code: 'artifacts',   label: 'Меньше артефактов' },
  { code: 'integration', label: 'Лучшая интеграция в сцену' },
  { code: 'timing',      label: 'Лучший тайминг / ритм' },
  { code: 'shape',       label: 'Лучшая форма / плотность' },
  { code: 'overall',     label: 'Лучшее общее качество' },
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
      <p style={{ fontSize: '13px', color: 'var(--color-text-muted)', marginBottom: '10px' }}>
        Причина (необязательно, до {MAX_SELECTED}):
      </p>
      <div style={{ display: 'flex', flexWrap: 'wrap', gap: '8px' }}>
        {REASONS.map((r) => {
          const active = selected.includes(r.code)
          const maxed = !active && selected.length >= MAX_SELECTED
          return (
            <button
              key={r.code}
              onClick={() => toggle(r.code)}
              disabled={disabled || maxed}
              style={{
                padding: '6px 14px',
                border: `1px solid ${active ? 'var(--color-primary)' : 'var(--color-border)'}`,
                borderRadius: '99px',
                background: active ? 'rgba(108,99,255,0.2)' : 'transparent',
                color: active ? 'var(--color-primary-h)' : maxed ? 'var(--color-text-muted)' : 'var(--color-text)',
                fontSize: '13px',
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
