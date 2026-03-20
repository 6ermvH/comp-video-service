import { useNavigate } from 'react-router-dom'
import { useSession } from '../context/SessionContext.jsx'

export default function InstructionsPage() {
  const navigate = useNavigate()
  const { sessionToken, studyMeta } = useSession()
  const instructionsText = studyMeta?.instructions_text || ''
  const bulletStyle = { marginBottom: '14px', lineHeight: 1.45 }
  const nestedBulletStyle = { marginBottom: '6px', lineHeight: 1.35 }
  const instructionsTextColor = 'rgba(232, 237, 248, 0.9)'

  // Guard: if no session, redirect to welcome
  if (!sessionToken) {
    navigate('/', { replace: true })
    return null
  }

  return (
    <div style={{
      minHeight: '100vh',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      padding: 'clamp(16px, 4vw, 24px)',
      background: 'var(--color-bg)',
    }}>
      <div style={{ maxWidth: 'min(880px, 100%)', width: '100%' }}>

        <h1 style={{ fontSize: '26px', fontWeight: 700, marginBottom: '24px', textAlign: 'center' }}>
          Инструкция
        </h1>

        {instructionsText ? (
          <div className="card" style={{ marginBottom: '24px', whiteSpace: 'pre-line', lineHeight: 1.8, color: instructionsTextColor }}>
            {instructionsText}
          </div>
        ) : (
          <div className="card" style={{ marginBottom: '24px' }}>
            <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
              <section>
                <ul style={{ paddingLeft: '20px', color: instructionsTextColor, margin: 0 }}>
                  <li style={bulletStyle}>Будут представлены видео А и B, выберите лучшее видео в целом.</li>
                  <li style={bulletStyle}>
                    Обращайте внимание на следующие факторы:
                    <ul style={{ marginTop: '4px', marginBottom: 0, paddingLeft: '22px' }}>
                      <li style={nestedBulletStyle}>реализм движения объектов в видео - движение выглядит так, как оно могло бы происходить в реальности; скорость, направление и развитие движения выглядят убедительно</li>
                      <li style={nestedBulletStyle}>отсутствие артефактов: искажение формы объекта, появление или исчезновение деталей без причины, нелогичные текстуры и другие</li>
                      <li style={nestedBulletStyle}>детализация изображения - четкость и проработанность мелких элементов, текстур и границ объектов в кадре</li>
                      <li style={{ ...nestedBulletStyle, marginBottom: 0 }}>общая визуальная целостность - видео воспринимается как единая, согласованная сцена, а не как набор отдельных, плохо сочетающихся друг с другом элементов.</li>
                    </ul>
                  </li>
                  <li style={bulletStyle}>Для выбранного видео можете опционально указать факторы, оказавшие наибольшее влияние на ваш выбор.</li>
                  <li style={bulletStyle}>
                    Также можете опционально указать степень уверенности вашего выбора по 5-балльной шкале:
                    <ul style={{ marginTop: '4px', marginBottom: 0, paddingLeft: '22px' }}>
                      <li style={nestedBulletStyle}>1 - разница между видео неясна или практически не ощущается</li>
                      <li style={nestedBulletStyle}>2 - есть слабое ощущение, что один вариант лучше, но уверенности в этом почти нет</li>
                      <li style={nestedBulletStyle}>3 - выбор сделан осознанно, но без полной уверенности</li>
                      <li style={nestedBulletStyle}>4 - есть уверенность, что выбранный вариант действительно лучше</li>
                      <li style={{ ...nestedBulletStyle, marginBottom: 0 }}>5 - выбор однозначный, и сомнений практически нет.</li>
                    </ul>
                  </li>
                  <li style={bulletStyle}>Если затрудняетесь определить лучшее видео, выберите «Затрудняюсь ответить».</li>
                  <li style={bulletStyle}>Вы можете повторить просмотр неограниченное количество раз. Пожалуйста, не спешите - качество ваших оценок важнее скорости.</li>
                  <li style={{ ...bulletStyle, marginBottom: 0 }}>После подтверждения выбора и перехода к следующей паре видео, вы не сможете вернуться назад и изменить свой выбор.</li>
                </ul>
              </section>

            </div>
          </div>
        )}

        <div style={{ textAlign: 'center' }}>
          <button
            className="btn btn-primary"
            onClick={() => navigate('/practice')}
            style={{ padding: '14px 40px', fontSize: '16px', width: '100%' }}
          >
            Далее
          </button>
        </div>
      </div>
    </div>
  )
}
