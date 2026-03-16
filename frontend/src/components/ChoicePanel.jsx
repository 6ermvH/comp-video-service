/**
 * ChoicePanel — A / Tie / B selection buttons.
 * Keyboard: 1 or A = left, 0 or T = tie, 2 or B = right.
 */
export default function ChoicePanel({ choice, onChange, disabled, tieEnabled = true }) {
  const options = [
    { key: 'left',  label: 'A', hint: '1', color: '#3b82f6' },
    { key: 'right', label: 'B', hint: '2', color: '#10b981' },
    ...(tieEnabled ? [{ key: 'tie', label: 'Затрудняюсь ответить', hint: '0', color: '#6b7280' }] : []),
  ]

  return (
    <div style={{ display: 'flex', gap: '8px', justifyContent: 'center', flexWrap: 'wrap' }}>
      {options.map((opt) => {
        const selected = choice === opt.key
        return (
          <button
            key={opt.key}
            onClick={() => !disabled && onChange(opt.key)}
            disabled={disabled}
            style={{
              minWidth: '112px',
              padding: '10px 14px',
              border: `2px solid ${selected ? opt.color : 'var(--color-border)'}`,
              borderRadius: 'var(--radius-md)',
              background: selected ? `${opt.color}22` : 'var(--color-surface-2)',
              color: selected ? opt.color : 'var(--color-text)',
              fontFamily: 'var(--font-family)',
              fontSize: '14px',
              fontWeight: 600,
              cursor: disabled ? 'not-allowed' : 'pointer',
              transition: 'all 0.15s ease',
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'center',
              gap: '2px',
              outline: 'none',
            }}
            onFocus={(e) => { e.target.style.boxShadow = `0 0 0 3px ${opt.color}44` }}
            onBlur={(e) => { e.target.style.boxShadow = 'none' }}
          >
            <span>{opt.label}</span>
          </button>
        )
      })}
    </div>
  )
}
