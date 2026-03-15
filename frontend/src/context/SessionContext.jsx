import { createContext, useContext, useState, useCallback } from 'react'
import { api, sessionStorage } from '../api/client.js'

const SessionContext = createContext(null)

/**
 * Normalize raw task payload (from first_task or next-task) to a stable internal shape.
 * Contract fields: presentation_id, source_item_id, task_order, is_attention_check,
 *                  is_practice, left.presigned_url, right.presigned_url
 */
function normalizeTask(raw) {
  if (!raw) return null
  return {
    presentation_id:    raw.presentation_id,
    source_item_id:     raw.source_item_id,
    task_order:         raw.task_order,
    is_attention_check: raw.is_attention_check ?? false,
    is_practice:        raw.is_practice ?? false,
    left_video_url:     raw.left?.presigned_url,
    right_video_url:    raw.right?.presigned_url,
  }
}

export function SessionProvider({ children }) {
  const [sessionToken, setSessionToken]   = useState(() => sessionStorage.getToken())
  const [studyMeta, setStudyMeta]         = useState(null)   // meta block from session/start
  const [currentTask, setCurrentTask]     = useState(null)
  const [tasksTotal, setTasksTotal]       = useState(0)
  const [completionCode, setCompletionCode] = useState(null)
  const [loading, setLoading]             = useState(false)
  const [error, setError]                 = useState(null)

  const startSession = useCallback(async (studyId, participantData) => {
    setLoading(true)
    setError(null)
    try {
      const res = await api.startSession({
        study_id:    studyId,
        device_type: participantData.deviceType,
        browser:     navigator.userAgent.slice(0, 100),
        role:        participantData.role,
        experience:  participantData.experience,
      })

      sessionStorage.setToken(res.session_token)
      setSessionToken(res.session_token)
      setStudyMeta(res.meta || null)
      setTasksTotal(res.assigned || 0)

      // first_task is already included in session/start — use it immediately
      if (res.first_task) {
        setCurrentTask(normalizeTask(res.first_task))
      }

      return res
    } catch (err) {
      setError(err.message)
      throw err
    } finally {
      setLoading(false)
    }
  }, [])

  /**
   * Fetch the next task after submitting a response.
   * Returns normalized task, or null when 204 (no more tasks).
   */
  const loadNextTask = useCallback(async () => {
    const tok = sessionToken
    if (!tok) throw new Error('No session token')
    setLoading(true)
    setError(null)
    try {
      // 204 → request() returns null → no more tasks
      const raw = await api.getNextTask(tok)
      if (!raw) {
        setCurrentTask(null)
        return null
      }
      const task = normalizeTask(raw)
      setCurrentTask(task)
      return task
    } catch (err) {
      setError(err.message)
      throw err
    } finally {
      setLoading(false)
    }
  }, [sessionToken])

  const submitResponse = useCallback(async (presentationId, data) => {
    setError(null)
    try {
      return await api.submitResponse(presentationId, data)
    } catch (err) {
      // 409 = duplicate response — treat as idempotent success, don't block the user
      if (err.status === 409) return { duplicate: true }
      setError(err.message)
      throw err
    }
  }, [])

  const logEvent = useCallback(async (presentationId, eventType, payload = {}) => {
    try {
      await api.logEvent(presentationId, { event_type: eventType, payload_json: payload })
    } catch (_) { /* best-effort, don't block UI */ }
  }, [])

  const completeSession = useCallback(async () => {
    if (!sessionToken) return null
    setLoading(true)
    try {
      const res = await api.completeSession(sessionToken)
      // res = { completion_code: "CVS-xxxxxxxx" }
      setCompletionCode(res?.completion_code || null)
      sessionStorage.clearToken()
      return res
    } catch (err) {
      setError(err.message)
      throw err
    } finally {
      setLoading(false)
    }
  }, [sessionToken])

  const clearSession = useCallback(() => {
    sessionStorage.clearToken()
    setSessionToken(null)
    setStudyMeta(null)
    setCurrentTask(null)
    setCompletionCode(null)
    setError(null)
  }, [])

  return (
    <SessionContext.Provider value={{
      sessionToken,
      studyMeta,           // { tie_option_enabled, reasons_enabled, confidence_enabled, ... }
      currentTask,         // normalized task shape
      tasksTotal,          // from res.assigned
      completionCode,
      loading,
      error,
      startSession,
      loadNextTask,
      submitResponse,
      logEvent,
      completeSession,
      clearSession,
    }}>
      {children}
    </SessionContext.Provider>
  )
}

// eslint-disable-next-line react-refresh/only-export-components
export function useSession() {
  const ctx = useContext(SessionContext)
  if (!ctx) throw new Error('useSession must be used within SessionProvider')
  return ctx
}
