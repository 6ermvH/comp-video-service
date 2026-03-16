import { useNavigate } from 'react-router-dom'
import { useSession } from '../context/SessionContext.jsx'

export default function BreakPage() {
  const navigate = useNavigate()
  const { sessionToken, currentTask, tasksTotal } = useSession()

  if (!sessionToken) {
    navigate('/', { replace: true })
    return null
  }

  const taskOrder = currentTask?.task_order || 0
  const remaining = tasksTotal - taskOrder

  return (
    <div style={{
      minHeight: '100vh',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      padding: 'clamp(16px, 4vw, 24px)',
      background: 'var(--color-bg)',
    }}>
      <div style={{ maxWidth: 'min(500px, 100%)', width: '100%', textAlign: 'center' }}>

        <div style={{ fontSize: '64px', marginBottom: '24px' }}>☕</div>

        <h1 style={{ fontSize: '26px', fontWeight: 700, marginBottom: '16px' }}>
          Небольшой перерыв
        </h1>

        <p style={{ color: 'var(--color-text-muted)', lineHeight: 1.8, marginBottom: '24px' }}>
          Отличная работа! Вы прошли несколько заданий.
          {remaining > 0 && ` Осталось ещё ${remaining} ${pluralTasks(remaining)}.`}
          <br />
          Сделайте паузу и продолжайте, когда будете готовы.
        </p>

        <div className="card" style={{ marginBottom: '24px', padding: '16px', textAlign: 'left' }}>
          <p style={{ fontSize: '14px', color: 'var(--color-text-muted)', marginBottom: '8px' }}>
            Напоминание:
          </p>
          <ul style={{ paddingLeft: '20px', color: 'var(--color-text-muted)', fontSize: '14px', lineHeight: 2 }}>
            <li>Оценивайте реалистичность, а не личное предпочтение</li>
            <li>Повторяйте видео столько раз, сколько нужно (клавиша R)</li>
            <li>Не спешите — точность важнее скорости</li>
          </ul>
        </div>

        <button
          className="btn btn-primary"
          onClick={() => navigate('/task')}
          style={{ padding: '14px 40px', fontSize: '16px', width: '100%' }}
        >
          Продолжить →
        </button>
      </div>
    </div>
  )
}

function pluralTasks(n) {
  if (n === 1) return 'задание'
  if (n >= 2 && n <= 4) return 'задания'
  return 'заданий'
}
