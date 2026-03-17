import { useState, useEffect, useRef, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { useSession } from '../context/SessionContext.jsx'
import { useToast } from '../context/ToastContext.jsx'
import SyncVideoPlayer from '../components/SyncVideoPlayer.jsx'
import ChoicePanel from '../components/ChoicePanel.jsx'
import ReasonsSelector from '../components/ReasonsSelector.jsx'
import ConfidenceRating from '../components/ConfidenceRating.jsx'
import ProgressBar from '../components/ProgressBar.jsx'
import { useWindowWidth } from '../hooks/useWindowWidth.js'

// Show break page every N real tasks
const BREAK_EVERY = 10

export default function TaskPage({ isPractice = false }) {
  const navigate = useNavigate()
  const { addToast } = useToast()
  const {
    sessionToken, studyMeta, currentTask, tasksTotal,
    loadNextTask, submitResponse, logEvent, completeSession,
  } = useSession()

  const [choice, setChoice]       = useState(null)  // 'left' | 'right' | 'tie'
  const [reasons, setReasons]     = useState([])
  const [confidence, setConfidence] = useState(null)
  const [replayCount, setReplayCount] = useState(0)
  const [submitting, setSubmitting] = useState(false)
  const [taskStartTs, setTaskStartTs] = useState(null)

  const playerRef    = useRef(null)
  const taskCountRef = useRef(0)
  const isMobile = useWindowWidth() <= 768

  // Guard: no session → back to welcome
  useEffect(() => {
    if (!sessionToken) navigate('/', { replace: true })
  }, [sessionToken, navigate])

  // first_task may be null when session starts with no available tasks yet —
  // fetch the first task on mount in that case
  useEffect(() => {
    if (sessionToken && !currentTask) {
      loadNextTask().catch(() => {})
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [sessionToken])

  // Reset UI state when task changes (keyed on presentation_id per contract)
  useEffect(() => {
    if (!currentTask) return
    setChoice(null)
    setReasons([])
    setConfidence(null)
    setReplayCount(0)
    setTaskStartTs(Date.now())
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [currentTask?.presentation_id])

  // Keyboard shortcuts
  const handleKey = useCallback((e) => {
    if (submitting) return
    switch (e.key) {
      case '1': case 'a': case 'A': setChoice('left'); break
      case '2': case 'b': case 'B': setChoice('right'); break
      case '0': case 't': case 'T': setChoice('tie'); break
      case 'r': case 'R': playerRef.current?.replay(); break
      case ' ': e.preventDefault(); playerRef.current?.togglePlayPause(); break
      case 'n': case 'N': case 'Enter': if (choice) handleSubmit(); break
    }
  }, [choice, submitting]) // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    window.addEventListener('keydown', handleKey)
    return () => window.removeEventListener('keydown', handleKey)
  }, [handleKey])

  const handleReplay = () => {
    setReplayCount((c) => c + 1)
    if (currentTask) {
      logEvent(currentTask.presentation_id, 'replay_clicked', { count: replayCount + 1 })
    }
  }

  const handleSubmit = async () => {
    if (!choice || !currentTask || submitting) return

    setSubmitting(true)
    const responseTimeMs = taskStartTs ? Date.now() - taskStartTs : 0

    try {
      // Submit response — task id is presentation_id per contract
      await submitResponse(currentTask.presentation_id, {
        choice,
        reason_codes: reasons,
        confidence:   confidence || null,
        response_time_ms: responseTimeMs,
        replay_count: replayCount,
      })

      taskCountRef.current += 1

      // Fetch next task — returns null on 204 (no more tasks)
      const next = await loadNextTask()

      if (!next) {
        // 204: all tasks done → complete session
        await completeSession()
        navigate('/complete', { replace: true })
        return
      }

      // Break every N real tasks (not practice)
      if (!isPractice && taskCountRef.current % BREAK_EVERY === 0) {
        navigate('/break')
        return
      }

      // Transition from practice to real tasks
      if (isPractice && !next.is_practice) {
        navigate('/task', { replace: true })
      }

    } catch (err) {
      if (err.status === 429) {
        addToast('Слишком много запросов. Подождите и попробуйте снова.', 'warning')
      } else if (err.status >= 500) {
        addToast('Ошибка сервера. Ответ не сохранён — попробуйте ещё раз.', 'error', { retryFn: handleSubmit })
      } else {
        console.error('Submit failed:', err)
      }
    } finally {
      setSubmitting(false)
    }
  }

  if (!currentTask) {
    return (
      <div style={centerStyle}>
        <div className="spinner" />
        <p style={{ color: 'var(--color-text-muted)', marginTop: '16px' }}>Загрузка задания…</p>
      </div>
    )
  }

  const tieEnabled    = studyMeta?.tie_option_enabled ?? true
  const showReasons   = studyMeta?.reasons_enabled    ?? true
  const showConfidence = studyMeta?.confidence_enabled ?? true
  const hasDetailedFeedback = choice && choice !== 'tie'

  return (
    <div style={{
      minHeight: '100vh',
      background: 'var(--color-bg)',
      padding: '10px 4px 20px',
      display: 'flex',
      flexDirection: 'column',
      gap: '4px',
      width: '100%',
      maxWidth: 'none',
      margin: '0 auto',
    }}>

      {/* Synchronized video player */}
      <div style={isMobile ? {
        width: '100%',
        display: 'flex',
        flexDirection: 'column',
      } : {
        width: 'calc(100vw - 8px)',
        maxWidth: 'none',
        marginLeft: 'calc(50% - 50vw + 4px)',
        marginRight: 'calc(50% - 50vw + 4px)',
        display: 'flex',
        flexDirection: 'column',
      }}>
        <SyncVideoPlayer
          ref={playerRef}
          leftUrl={currentTask.left_video_url}
          rightUrl={currentTask.right_video_url}
          onReplay={handleReplay}
          onEnded={() => { /* tracking hook */ }}
        />
        {isMobile && (
          <button
            className="btn btn-ghost"
            style={{ width: '100%', marginTop: '8px' }}
            onClick={() => playerRef.current?.replay()}
          >
            ↺ Повторить оба видео
          </button>
        )}
      </div>

      {/* Response panel */}
      <div className="card" style={{
        width: '100%',
        maxWidth: '980px',
        margin: '60px auto 0',
        padding: '12px 16px',
        display: 'flex',
        flexDirection: 'column',
        gap: '10px',
      }}>

        <div style={{ textAlign: 'center' }}>
          <p style={{ fontSize: '13px', color: 'var(--color-text-muted)', marginBottom: '8px', whiteSpace: 'pre-line', lineHeight: 1.5 }}>
            {'Какое видео выглядит лучше в целом ?\nОриентируйтесь на правдоподобие эффекта, стабильность, отсутствие артефактов и общую визуальную целостность.'}
          </p>
          <ChoicePanel
            choice={choice}
            onChange={setChoice}
            disabled={submitting}
            tieEnabled={tieEnabled}
          />
        </div>

        {showReasons && hasDetailedFeedback && (
          <ReasonsSelector
            selected={reasons}
            onChange={setReasons}
            disabled={submitting}
          />
        )}

        {showConfidence && hasDetailedFeedback ? (
          <div style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'flex-end',
            gap: '16px',
            flexWrap: 'wrap',
          }}>
            <ConfidenceRating
              value={confidence}
              onChange={setConfidence}
              disabled={submitting}
            />
            <button
              className="btn btn-primary"
              onClick={handleSubmit}
              disabled={!choice || submitting}
              style={{ minWidth: '140px', padding: '10px 18px' }}
            >
              {submitting ? 'Отправка…' : 'Следующее →'}
            </button>
          </div>
        ) : (
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <div />
            <button
              className="btn btn-primary"
              onClick={handleSubmit}
              disabled={!choice || submitting}
              style={{ minWidth: '140px', padding: '10px 18px' }}
            >
              {submitting ? 'Отправка…' : 'Следующее →'}
            </button>
          </div>
        )}

        {currentTask.is_practice && (
          <div style={{
            padding: '8px 14px',
            background: 'rgba(240,180,41,0.1)',
            border: '1px solid rgba(240,180,41,0.2)',
            borderRadius: 'var(--radius-sm)',
            fontSize: '13px', color: 'var(--color-warning)',
          }}>
            🎓 Тренировочное задание — ответы не учитываются
          </div>
        )}

        {currentTask.is_attention_check && (
          <div style={{
            padding: '8px 14px',
            background: 'rgba(67,217,139,0.08)',
            border: '1px solid rgba(67,217,139,0.2)',
            borderRadius: 'var(--radius-sm)',
            fontSize: '13px', color: 'var(--color-success)',
          }}>
            ✓ Внимательно сравните оба видео
          </div>
        )}
      </div>

      {/* Progress */}
      <div style={{ width: '100%', maxWidth: '1280px', margin: 'auto auto 0', paddingTop: '8px' }}>
        <ProgressBar
          current={currentTask.task_order}
          total={tasksTotal}
          isPractice={currentTask.is_practice}
        />
      </div>

    </div>
  )
}

const centerStyle = {
  minHeight: '100vh',
  display: 'flex',
  flexDirection: 'column',
  alignItems: 'center',
  justifyContent: 'center',
  background: 'var(--color-bg)',
}
