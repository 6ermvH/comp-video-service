import { useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useSession } from '../context/SessionContext.jsx'
import TaskPage from './TaskPage.jsx'

/**
 * PracticePage — wrapper around TaskPage for practice mode.
 * first_task is already in context from session/start.
 * If current task is already a real task, skip straight to /task.
 */
export default function PracticePage() {
  const navigate = useNavigate()
  const { sessionToken, currentTask } = useSession()

  useEffect(() => {
    if (!sessionToken) {
      navigate('/', { replace: true })
      return
    }
    if (currentTask && !currentTask.is_practice) {
      navigate('/task', { replace: true })
    }
  }, [sessionToken, currentTask, navigate])

  return <TaskPage isPractice />
}
