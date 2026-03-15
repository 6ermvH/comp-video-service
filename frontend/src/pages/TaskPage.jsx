import { useState, useEffect, useRef, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { useSession } from '../context/SessionContext.jsx'
import { useToast } from '../context/ToastContext.jsx'
import SyncVideoPlayer from '../components/SyncVideoPlayer.jsx'
import ChoicePanel from '../components/ChoicePanel.jsx'
import ReasonsSelector from '../components/ReasonsSelector.jsx'
import ConfidenceRating from '../components/ConfidenceRating.jsx'
import ProgressBar from '../components/ProgressBar.jsx'

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

  return (
    <div style={{
      minHeight: '100vh',
      background: 'var(--color-bg)',
      padding: '20px 24px',
      display: 'flex',
      flexDirection: 'column',
      gap: '16px',
      maxWidth: '1100px',
      margin: '0 auto',
    }}>

      {/* Progress */}
      <ProgressBar
        current={currentTask.task_order}
        total={tasksTotal}
        isPractice={currentTask.is_practice}
      />

      {/* Synchronized video player */}
      <SyncVideoPlayer
        ref={playerRef}
        leftUrl={currentTask.left_video_url}
        rightUrl={currentTask.right_video_url}
        onReplay={handleReplay}
        onEnded={() => { /* tracking hook */ }}
      />

      {/* Response panel */}
      <div className="card" style={{ padding: '20px', display: 'flex', flexDirection: 'column', gap: '16px' }}>

        <div style={{ textAlign: 'center' }}>
          <p style={{ fontSize: '15px', color: 'var(--color-text-muted)', marginBottom: '12px' }}>
            Какое видео выглядит более реалистично?
          </p>
          <ChoicePanel
            choice={choice}
            onChange={setChoice}
            disabled={submitting}
            tieEnabled={tieEnabled}
          />
        </div>

        {showReasons && choice && choice !== 'tie' && (
          <ReasonsSelector
            selected={reasons}
            onChange={setReasons}
            disabled={submitting}
          />
        )}

        {showConfidence && choice && (
          <ConfidenceRating
            value={confidence}
            onChange={setConfidence}
            disabled={submitting}
          />
        )}

        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <div style={{ fontSize: '13px', color: 'var(--color-text-muted)' }}>
            {replayCount > 0 && `↺ ${replayCount} повтор${replayCount === 1 ? '' : 'а'}`}
          </div>
          <button
            className="btn btn-primary"
            onClick={handleSubmit}
            disabled={!choice || submitting}
            style={{ minWidth: '160px', padding: '12px 24px' }}
          >
            {submitting ? 'Отправка…' : 'Следующее →'}
          </button>
        </div>

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

      {/* Keyboard hint */}
      <p style={{ textAlign: 'center', fontSize: '12px', color: 'var(--color-text-muted)' }}>
        Клавиши: <strong>1</strong> — A, <strong>2</strong> — B, <strong>0</strong> — равны,
        {' '}<strong>R</strong> — повтор, <strong>N</strong> / Enter — далее
      </p>
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
