/**
 * ChoicePanel — A / Tie / B selection buttons.
 * Keyboard: 1 or A = left, 0 or T = tie, 2 or B = right.
 */
export default function ChoicePanel({ choice, onChange, disabled, tieEnabled = true }) {
  const options = [
    { key: 'left',  label: 'A лучше',   hint: '1', color: '#3b82f6' },
    ...(tieEnabled ? [{ key: 'tie', label: 'Равны', hint: '0', color: '#6b7280' }] : []),
    { key: 'right', label: 'B лучше',   hint: '2', color: '#10b981' },
  ]

  return (
    <div style={{ display: 'flex', gap: '12px', justifyContent: 'center', flexWrap: 'wrap' }}>
      {options.map((opt) => {
        const selected = choice === opt.key
        return (
          <button
            key={opt.key}
            onClick={() => !disabled && onChange(opt.key)}
            disabled={disabled}
            style={{
              minWidth: '130px',
              padding: '14px 20px',
              border: `2px solid ${selected ? opt.color : 'var(--color-border)'}`,
              borderRadius: 'var(--radius-md)',
              background: selected ? `${opt.color}22` : 'var(--color-surface-2)',
              color: selected ? opt.color : 'var(--color-text)',
              fontFamily: 'var(--font-family)',
              fontSize: '15px',
              fontWeight: 600,
              cursor: disabled ? 'not-allowed' : 'pointer',
              transition: 'all 0.15s ease',
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'center',
              gap: '4px',
              outline: 'none',
            }}
            onFocus={(e) => { e.target.style.boxShadow = `0 0 0 3px ${opt.color}44` }}
            onBlur={(e) => { e.target.style.boxShadow = 'none' }}
          >
            <span>{opt.label}</span>
            <span style={{ fontSize: '11px', opacity: 0.6, fontWeight: 400 }}>
              клавиша {opt.hint}
            </span>
          </button>
        )
      })}
    </div>
  )
}
