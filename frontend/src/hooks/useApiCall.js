import { useToast } from '../context/ToastContext.jsx'

/**
 * Returns a wrapper that calls an async API function and handles common HTTP errors:
 *   429 → toast with retry button
 *   500 → toast with retry button
 *   other → re-throws (caller handles)
 *
 * Usage:
 *   const call = useApiCall()
 *   await call(() => api.getStudies(), { onRetry: load })
 */
export function useApiCall() {
  const { addToast } = useToast()

  return async function call(fn, { onRetry } = {}) {
    try {
      return await fn()
    } catch (err) {
      const status = err.status

      if (status === 429) {
        const after = err.retryAfter ? ` Повторите через ${err.retryAfter}с.` : ''
        addToast(`Слишком много запросов.${after}`, 'warning', {
          sticky: true,
          retryFn: onRetry,
        })
        throw err
      }

      if (status >= 500) {
        addToast(`Ошибка сервера (${status}). Попробуйте ещё раз.`, 'error', {
          retryFn: onRetry,
        })
        throw err
      }

      throw err
    }
  }
}
