/**
 * API Client — Video Comparison Service
 * Covers both participant (public) and admin (JWT) endpoints.
 */

const BASE_URL = import.meta.env.VITE_API_BASE_URL
  ? `${import.meta.env.VITE_API_BASE_URL}/api`
  : '/api'

const ADMIN_TOKEN_KEY   = 'cvs_admin_token'
const CSRF_TOKEN_KEY    = 'cvs_csrf_token'
const SESSION_TOKEN_KEY = 'cvs_session_token'

// ── Auth helpers (admin JWT) ───────────────────────────────
export const auth = {
  setToken:  (token) => localStorage.setItem(ADMIN_TOKEN_KEY, token),
  getToken:  ()      => localStorage.getItem(ADMIN_TOKEN_KEY),
  clearToken: () => {
    localStorage.removeItem(ADMIN_TOKEN_KEY)
    localStorage.removeItem(CSRF_TOKEN_KEY)
  },
  isLoggedIn: () => !!localStorage.getItem(ADMIN_TOKEN_KEY),
}

// ── CSRF helpers ───────────────────────────────────────────
export const csrf = {
  setToken: (token) => localStorage.setItem(CSRF_TOKEN_KEY, token),
  getToken: ()      => localStorage.getItem(CSRF_TOKEN_KEY),
}

// ── Session helpers (participant token) ────────────────────
export const sessionStorage = {
  setToken:  (token) => localStorage.setItem(SESSION_TOKEN_KEY, token),
  getToken:  ()      => localStorage.getItem(SESSION_TOKEN_KEY),
  clearToken: ()     => localStorage.removeItem(SESSION_TOKEN_KEY),
}

// ── Unauthorized callback (registered by AdminLayout) ─────
let _onUnauthorized = null
export function setUnauthorizedHandler(fn) { _onUnauthorized = fn }

// ── API error with HTTP status attached ───────────────────
function apiError(message, status, extra = {}) {
  return Object.assign(new Error(message), { status, ...extra })
}

const CSRF_METHODS  = new Set(['POST', 'PUT', 'PATCH', 'DELETE'])
const ADMIN_PATH_RE = /^\/admin\//

// ── Shared response handler ────────────────────────────────
async function handleResponse(res, path) {
  if (res.ok) {
    if (res.status === 204) return null
    return res.json()
  }

  let message = `HTTP ${res.status}`
  let retryAfter = null
  try {
    const body = await res.json()
    message = body.error || body.message || message
    retryAfter = body.retry_after ?? null
  } catch (_) { /* ignore */ }

  const status = res.status

  // 401/403 on admin routes → clear session and hand off to registered handler
  if ((status === 401 || status === 403) && ADMIN_PATH_RE.test(path)) {
    auth.clearToken()
    if (_onUnauthorized) _onUnauthorized()
    else window.location.href = '/admin/login'
  }

  throw apiError(message, status, retryAfter != null ? { retryAfter } : {})
}

// ── Base fetch wrapper ─────────────────────────────────────
async function request(path, options = {}) {
  const headers = { 'Content-Type': 'application/json', ...options.headers }

  const token = auth.getToken()
  if (token) headers['Authorization'] = `Bearer ${token}`

  const method = (options.method || 'GET').toUpperCase()
  if (CSRF_METHODS.has(method) && ADMIN_PATH_RE.test(path)) {
    const csrfToken = csrf.getToken()
    if (csrfToken) headers['X-CSRF-Token'] = csrfToken
  }

  const res = await fetch(`${BASE_URL}${path}`, { ...options, headers })
  return handleResponse(res, path)
}

// ── Multipart helper ───────────────────────────────────────
async function requestMultipart(path, formData, method = 'POST') {
  const headers = {}
  const token = auth.getToken()
  if (token) headers['Authorization'] = `Bearer ${token}`

  const m = method.toUpperCase()
  if (CSRF_METHODS.has(m) && ADMIN_PATH_RE.test(path)) {
    const csrfToken = csrf.getToken()
    if (csrfToken) headers['X-CSRF-Token'] = csrfToken
  }

  const res = await fetch(`${BASE_URL}${path}`, { method, headers, body: formData })
  return handleResponse(res, path)
}

// ── Download helper (returns Blob) ─────────────────────────
async function download(path) {
  const headers = {}
  const token = auth.getToken()
  if (token) headers['Authorization'] = `Bearer ${token}`

  const res = await fetch(`${BASE_URL}${path}`, { headers })
  if (!res.ok) {
    const status = res.status
    if ((status === 401 || status === 403) && ADMIN_PATH_RE.test(path)) {
      auth.clearToken()
      if (_onUnauthorized) _onUnauthorized()
      else window.location.href = '/admin/login'
    }
    throw apiError(`HTTP ${status}`, status)
  }
  return res.blob()
}

export const api = {
  // ── Participant (public) ──────────────────────────────────

  /** POST /session/start → { session_token, assigned, meta, first_task } */
  startSession: (data) =>
    request('/session/start', { method: 'POST', body: JSON.stringify(data) }),

  /** GET /session/:token/next-task → task object | null (204) */
  getNextTask: (token) =>
    request(`/session/${token}/next-task`),

  /**
   * POST /task/:id/response
   * Throws with status=409 on duplicate (caller handles as idempotent).
   */
  submitResponse: (presentationId, data) =>
    request(`/task/${presentationId}/response`, { method: 'POST', body: JSON.stringify(data) }),

  /** POST /task/:id/event */
  logEvent: (presentationId, data) =>
    request(`/task/${presentationId}/event`, { method: 'POST', body: JSON.stringify(data) }),

  /** POST /session/:token/complete → { completion_code } */
  completeSession: (token) =>
    request(`/session/${token}/complete`, { method: 'POST' }),

  // ── Admin auth ────────────────────────────────────────────

  /** POST /admin/login → { token, csrf_token, admin } */
  login: (username, password) =>
    request('/admin/login', { method: 'POST', body: JSON.stringify({ username, password }) }),

  // ── Admin: Studies ────────────────────────────────────────

  getStudies: () => request('/admin/studies'),

  createStudy: (data) =>
    request('/admin/studies', { method: 'POST', body: JSON.stringify(data) }),

  updateStudy: (id, data) =>
    request(`/admin/studies/${id}`, { method: 'PATCH', body: JSON.stringify(data) }),

  // ── Admin: Groups & Pairs ─────────────────────────────────

  getGroups: (studyId) =>
    request(`/admin/studies/${studyId}/groups`),

  createGroup: (studyId, data) =>
    request(`/admin/studies/${studyId}/groups`, { method: 'POST', body: JSON.stringify(data) }),

  /**
   * POST /admin/assets/upload — multipart fields:
   *   file (mp4), method_type, optional: source_item_id, title, description
   *   When source_item_id is omitted, video is uploaded to the library unlinked.
   */
  uploadAsset: (formData) =>
    requestMultipart('/admin/assets/upload', formData),

  getAssets: () => request('/admin/assets'),

  deleteAsset: (id) => request(`/admin/assets/${id}`, { method: 'DELETE' }),
  deletePair:  (id) => request(`/admin/source-items/${id}`, { method: 'DELETE' }),

  createPair: (studyId, body) =>
    request(`/admin/studies/${studyId}/pairs`, { method: 'POST', body: JSON.stringify(body) }),

  getSourceItems: (params = {}) => {
    const qs = new URLSearchParams(params).toString()
    return request(`/admin/source-items${qs ? '?' + qs : ''}`)
  },

  // ── Admin: Analytics ─────────────────────────────────────

  getAnalyticsOverview:  ()         => request('/admin/analytics/overview'),
  getAnalyticsPairs:     (studyId)  => request(`/admin/analytics/study/${studyId}/pairs`),
  getStudyAnalytics:     (id)       => request(`/admin/analytics/study/${id}`),
  getQCReport:           ()         => request('/admin/analytics/qc'),

  // ── Admin: Export ─────────────────────────────────────────

  exportCSV:  () => download('/admin/export/csv'),
  exportJSON: () => download('/admin/export/json'),
}
