import { createContext, useContext, useState, useCallback, useRef } from 'react'

const ToastContext = createContext(null)

const AUTO_DISMISS_MS = 5000

export function ToastProvider({ children }) {
  const [toasts, setToasts] = useState([])
  const nextId = useRef(1)

  const dismiss = useCallback((id) => {
    setToasts((t) => t.filter((x) => x.id !== id))
  }, [])

  const addToast = useCallback((message, type = 'error', opts = {}) => {
    const id = nextId.current++
    setToasts((t) => [...t, { id, message, type, retryFn: opts.retryFn ?? null }])
    if (!opts.sticky) {
      setTimeout(() => dismiss(id), AUTO_DISMISS_MS)
    }
    return id
  }, [dismiss])

  return (
    <ToastContext.Provider value={{ addToast, dismiss }}>
      {children}
      <ToastList toasts={toasts} onDismiss={dismiss} />
    </ToastContext.Provider>
  )
}

// eslint-disable-next-line react-refresh/only-export-components
export function useToast() {
  const ctx = useContext(ToastContext)
  if (!ctx) throw new Error('useToast must be used within ToastProvider')
  return ctx
}

// ── Toast list UI ──────────────────────────────────────────
const TYPE_STYLES = {
  error:   { border: 'rgba(244,63,94,0.4)',   bg: 'rgba(244,63,94,0.12)',   icon: '✕' },
  warning: { border: 'rgba(234,179,8,0.4)',   bg: 'rgba(234,179,8,0.1)',    icon: '⚠' },
  success: { border: 'rgba(34,197,94,0.4)',   bg: 'rgba(34,197,94,0.1)',    icon: '✓' },
  info:    { border: 'rgba(99,102,241,0.4)',  bg: 'rgba(99,102,241,0.1)',   icon: 'ℹ' },
}

function ToastList({ toasts, onDismiss }) {
  if (toasts.length === 0) return null

  return (
    <div style={{
      position: 'fixed',
      bottom: '24px',
      right: '24px',
      zIndex: 9000,
      display: 'flex',
      flexDirection: 'column',
      gap: '10px',
      maxWidth: '380px',
      width: '100%',
    }}>
      {toasts.map((toast) => {
        const s = TYPE_STYLES[toast.type] || TYPE_STYLES.error
        return (
          <div key={toast.id} style={{
            display: 'flex',
            alignItems: 'flex-start',
            gap: '10px',
            padding: '12px 14px',
            background: s.bg,
            border: `1px solid ${s.border}`,
            borderRadius: 'var(--radius-md)',
            boxShadow: 'var(--shadow-md)',
            animation: 'fadeIn 0.2s ease both',
            color: 'var(--color-text)',
            fontSize: '14px',
            lineHeight: 1.5,
          }}>
            <span style={{ flexShrink: 0, fontWeight: 700 }}>{s.icon}</span>
            <span style={{ flex: 1 }}>{toast.message}</span>
            <div style={{ display: 'flex', gap: '8px', flexShrink: 0, alignItems: 'center' }}>
              {toast.retryFn && (
                <button
                  onClick={() => { toast.retryFn(); onDismiss(toast.id) }}
                  style={{
                    background: 'none', border: '1px solid var(--color-border)',
                    borderRadius: '4px', color: 'var(--color-text)', cursor: 'pointer',
                    fontSize: '12px', padding: '2px 8px', fontFamily: 'var(--font-family)',
                  }}
                >
                  Повторить
                </button>
              )}
              <button
                onClick={() => onDismiss(toast.id)}
                style={{
                  background: 'none', border: 'none', color: 'var(--color-text-muted)',
                  cursor: 'pointer', fontSize: '16px', lineHeight: 1, padding: '0 2px',
                }}
              >
                ×
              </button>
            </div>
          </div>
        )
      })}
    </div>
  )
}
