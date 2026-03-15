/**
 * ProgressBar — shows "Сравнение N из M" with a visual bar.
 */
export default function ProgressBar({ current, total, isPractice }) {
  const pct = total > 0 ? Math.min(100, (current / total) * 100) : 0

  return (
    <div style={{ width: '100%' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between',
        fontSize: '13px', color: 'var(--color-text-muted)', marginBottom: '6px' }}>
        <span>
          {isPractice ? '🎓 Тренировка' : `Сравнение ${current} из ${total}`}
        </span>
        {!isPractice && (
          <span>{Math.round(pct)}%</span>
        )}
      </div>
      <div style={{
        height: '6px',
        background: 'var(--color-surface-2)',
        borderRadius: '99px',
        overflow: 'hidden',
      }}>
        <div style={{
          height: '100%',
          width: `${pct}%`,
          background: isPractice ? 'var(--color-warning)' : 'var(--color-primary)',
          borderRadius: '99px',
          transition: 'width 0.4s ease',
        }} />
      </div>
    </div>
  )
}
