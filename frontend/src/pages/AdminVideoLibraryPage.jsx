import { useState, useEffect, useRef, useCallback } from 'react'
import { api } from '../api/client.js'
import { useApiCall } from '../hooks/useApiCall.js'

const METHOD_COLORS = {
  baseline:  { bg: 'rgba(108,99,255,0.15)', text: '#a78bfa' },
  candidate: { bg: 'rgba(67,217,139,0.15)',  text: '#43d98b' },
}

// ── Pagination helpers ─────────────────────────────────────────────────────

function buildPageNumbers(current, total) {
  if (total <= 7) return Array.from({ length: total }, (_, i) => i + 1)
  const pages = []
  if (current <= 4) {
    pages.push(1, 2, 3, 4, 5, '...', total)
  } else if (current >= total - 3) {
    pages.push(1, '...', total - 4, total - 3, total - 2, total - 1, total)
  } else {
    pages.push(1, '...', current - 1, current, current + 1, '...', total)
  }
  return pages
}

// ── Video modal ────────────────────────────────────────────────────────────

function VideoModal({ asset, onClose }) {
  const [url, setUrl] = useState(null)
  const [loadingUrl, setLoadingUrl] = useState(true)
  const [urlError, setUrlError] = useState(null)
  const backdropRef = useRef(null)

  useEffect(() => {
    let cancelled = false
    setLoadingUrl(true)
    api.getAssetUrl(asset.id)
      .then((data) => { if (!cancelled) { setUrl(data.url); setLoadingUrl(false) } })
      .catch((err) => { if (!cancelled) { setUrlError(err.message); setLoadingUrl(false) } })
    return () => { cancelled = true }
  }, [asset.id])

  useEffect(() => {
    const handleKey = (e) => { if (e.key === 'Escape') onClose() }
    document.addEventListener('keydown', handleKey)
    return () => document.removeEventListener('keydown', handleKey)
  }, [onClose])

  const handleBackdropClick = (e) => {
    if (e.target === backdropRef.current) onClose()
  }

  return (
    <div
      ref={backdropRef}
      onClick={handleBackdropClick}
      style={{
        position: 'fixed', inset: 0, zIndex: 1000,
        background: 'rgba(0,0,0,0.75)',
        display: 'flex', alignItems: 'center', justifyContent: 'center',
        padding: '24px',
      }}
    >
      <div style={{
        background: 'var(--color-surface)',
        border: '1px solid var(--color-border)',
        borderRadius: 'var(--radius)',
        width: '100%', maxWidth: '900px',
        display: 'flex', flexDirection: 'column', gap: '16px',
        padding: '20px',
      }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
          <div>
            <div style={{ fontWeight: 600, fontSize: '16px' }}>{asset.title || asset.s3_key}</div>
            <div style={{ fontSize: '11px', color: 'var(--color-text-muted)', fontFamily: 'monospace', marginTop: '2px' }}>
              {asset.id}
            </div>
          </div>
          <button
            onClick={onClose}
            style={{
              background: 'none', border: 'none', cursor: 'pointer',
              color: 'var(--color-text-muted)', fontSize: '20px', lineHeight: 1,
              padding: '0 4px', flexShrink: 0,
            }}
          >
            ✕
          </button>
        </div>

        <div style={{
          background: '#000', borderRadius: 'var(--radius-sm)',
          minHeight: '200px', display: 'flex', alignItems: 'center', justifyContent: 'center',
        }}>
          {loadingUrl && <div className="spinner" />}
          {urlError && (
            <div style={{ color: '#ff6584', fontSize: '14px', padding: '16px' }}>
              Ошибка загрузки URL: {urlError}
            </div>
          )}
          {url && (
            <video
              controls
              autoPlay={false}
              src={url}
              style={{ width: '100%', maxHeight: '70vh', borderRadius: 'var(--radius-sm)' }}
            />
          )}
        </div>
      </div>
    </div>
  )
}

// ── Main page ──────────────────────────────────────────────────────────────

export default function AdminVideoLibraryPage() {
  const apiCall = useApiCall()
  const [assets, setAssets] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [successMsg, setSuccessMsg] = useState(null)

  // Pagination
  const [page, setPage]   = useState(1)
  const [total, setTotal] = useState(0)
  const PER_PAGE = 20

  // Search
  const [search, setSearch]         = useState('')
  const [searchInput, setSearchInput] = useState('')
  const debounceRef = useRef(null)

  // Upload form visibility
  const [uploadOpen, setUploadOpen] = useState(false)

  // Upload form state
  const [baselineFile, setBaselineFile]   = useState(null)
  const [baselineTitle, setBaselineTitle] = useState('')
  const [candidateFile, setCandidateFile]   = useState(null)
  const [candidateTitle, setCandidateTitle] = useState('')
  const [uploading, setUploading] = useState(false)

  // Video modal
  const [previewAsset, setPreviewAsset] = useState(null)

  const load = useCallback(async (p, s) => {
    setLoading(true)
    try {
      const data = await apiCall(
        () => api.getAssets(p, PER_PAGE, s),
        { onRetry: () => load(p, s) }
      )
      setAssets(data?.assets ?? [])
      setTotal(data?.total ?? 0)
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  useEffect(() => { load(page, search) }, [page, search, load])

  const handleSearchInput = (e) => {
    const val = e.target.value
    setSearchInput(val)
    if (debounceRef.current) clearTimeout(debounceRef.current)
    debounceRef.current = setTimeout(() => {
      setPage(1)
      setSearch(val)
    }, 300)
  }

  const handleDeleteAsset = async (id) => {
    if (!window.confirm('Удалить видео из библиотеки?')) return
    try {
      await api.deleteAsset(id)
      setSuccessMsg('Видео удалено')
      if (assets.length === 1 && page > 1) {
        setPage((p) => p - 1)
      } else {
        load(page, search)
      }
    } catch (err) {
      if (err.status === 409) {
        setError('Нельзя удалить: видео привязано к паре. Сначала удалите пару.')
      } else {
        setError(err.message)
      }
    }
  }

  const handleUpload = async (e) => {
    e.preventDefault()
    if (!baselineFile || !candidateFile) return
    setUploading(true)
    setError(null)
    try {
      if (baselineFile) {
        const fd = new FormData()
        fd.append('file', baselineFile)
        fd.append('method_type', 'baseline')
        fd.append('title', baselineTitle || baselineFile.name.replace(/\.[^.]+$/, ''))
        await apiCall(() => api.uploadAsset(fd))
      }
      if (candidateFile) {
        const fd = new FormData()
        fd.append('file', candidateFile)
        fd.append('method_type', 'candidate')
        fd.append('title', candidateTitle || candidateFile.name.replace(/\.[^.]+$/, ''))
        await apiCall(() => api.uploadAsset(fd))
      }
      setSuccessMsg('Baseline и candidate загружены')
      setBaselineFile(null); setBaselineTitle('')
      setCandidateFile(null); setCandidateTitle('')
      e.target.reset()
      setUploadOpen(false)
      load(page, search)
    } catch (err) {
      setError(err.message)
    } finally {
      setUploading(false)
    }
  }

  const totalPages = Math.ceil(total / PER_PAGE)
  const firstShown = total === 0 ? 0 : (page - 1) * PER_PAGE + 1
  const lastShown  = Math.min(page * PER_PAGE, total)

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '24px' }}>

      {/* Header */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', flexWrap: 'wrap', gap: '10px' }}>
        <h1 style={{ fontSize: '24px', fontWeight: 700 }}>Видеотека</h1>
        <div style={{ display: 'flex', gap: '8px' }}>
          <button className="btn btn-ghost" onClick={() => load(page, search)}>↻ Обновить</button>
          <button
            className="btn btn-primary"
            onClick={() => setUploadOpen((v) => !v)}
          >
            {uploadOpen ? '✕ Скрыть форму' : '⬆ Загрузить видео'}
          </button>
        </div>
      </div>

      {error && <ErrorBox message={error} onClose={() => setError(null)} />}
      {successMsg && (
        <div style={{ padding: '12px 16px', background: 'rgba(67,217,139,0.1)',
          border: '1px solid rgba(67,217,139,0.3)', borderRadius: 'var(--radius-sm)',
          color: '#43d98b', fontSize: '14px', display: 'flex', justifyContent: 'space-between' }}>
          <span>{successMsg}</span>
          <button onClick={() => setSuccessMsg(null)}
            style={{ background: 'none', border: 'none', color: '#43d98b', cursor: 'pointer' }}>✕</button>
        </div>
      )}

      {/* Upload form — collapsible */}
      <div style={{
        overflow: 'hidden',
        maxHeight: uploadOpen ? '600px' : '0',
        transition: 'max-height 0.35s ease',
      }}>
        <div className="card">
          <h2 style={{ fontSize: '16px', fontWeight: 600, marginBottom: '16px' }}>
            Загрузить видео
          </h2>
          <form onSubmit={handleUpload} style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '16px' }}>
              <div style={{ padding: '16px', border: '1px solid rgba(108,99,255,0.3)',
                borderRadius: 'var(--radius-sm)', background: 'rgba(108,99,255,0.05)' }}>
                <div style={{ fontSize: '13px', fontWeight: 600, color: '#a78bfa', marginBottom: '10px' }}>
                  Baseline
                </div>
                <input type="file" accept="video/mp4"
                  onChange={(e) => setBaselineFile(e.target.files[0])} />
                {baselineFile && (
                  <div style={{ fontSize: '12px', color: 'var(--color-text-muted)', marginTop: '6px' }}>
                    {baselineFile.name}
                  </div>
                )}
                <input className="input" placeholder="Название (необязательно)"
                  value={baselineTitle}
                  onChange={(e) => setBaselineTitle(e.target.value)}
                  style={{ marginTop: '8px', fontSize: '13px' }} />
              </div>

              <div style={{ padding: '16px', border: '1px solid rgba(67,217,139,0.3)',
                borderRadius: 'var(--radius-sm)', background: 'rgba(67,217,139,0.05)' }}>
                <div style={{ fontSize: '13px', fontWeight: 600, color: '#43d98b', marginBottom: '10px' }}>
                  Candidate
                </div>
                <input type="file" accept="video/mp4"
                  onChange={(e) => setCandidateFile(e.target.files[0])} />
                {candidateFile && (
                  <div style={{ fontSize: '12px', color: 'var(--color-text-muted)', marginTop: '6px' }}>
                    {candidateFile.name}
                  </div>
                )}
                <input className="input" placeholder="Название (необязательно)"
                  value={candidateTitle}
                  onChange={(e) => setCandidateTitle(e.target.value)}
                  style={{ marginTop: '8px', fontSize: '13px' }} />
              </div>
            </div>

            <div style={{ display: 'flex', justifyContent: 'flex-end' }}>
              <button type="submit" className="btn btn-primary"
                disabled={uploading || !baselineFile || !candidateFile}>
                {uploading ? 'Загрузка…' : '⬆ Загрузить'}
              </button>
            </div>
          </form>
        </div>
      </div>

      {/* Search */}
      <div style={{ position: 'relative', maxWidth: '360px' }}>
        <span style={{
          position: 'absolute', left: '12px', top: '50%', transform: 'translateY(-50%)',
          color: 'var(--color-text-muted)', fontSize: '14px', pointerEvents: 'none',
        }}>
          ⌕
        </span>
        <input
          className="input"
          placeholder="Поиск по названию..."
          value={searchInput}
          onChange={handleSearchInput}
          style={{ paddingLeft: '32px' }}
        />
      </div>

      {/* Assets list */}
      {loading ? (
        <div style={{ display: 'flex', justifyContent: 'center', padding: '48px' }}>
          <div className="spinner" />
        </div>
      ) : assets.length === 0 ? (
        <div style={{ textAlign: 'center', padding: '48px', color: 'var(--color-text-muted)' }}>
          {search ? 'Ничего не найдено.' : 'Нет загруженных видео.'}
        </div>
      ) : (
        <div className="card" style={{ padding: 0, overflow: 'hidden' }}>
          <div style={{ overflowX: 'auto' }}>
            <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '14px' }}>
              <thead>
                <tr style={{ borderBottom: '1px solid var(--color-border)' }}>
                  {['Название', 'Тип', 'Статус', 'Привязка к паре', 'Добавлено', ''].map((h) => (
                    <th key={h} style={{ textAlign: 'left', padding: '12px 16px',
                      color: 'var(--color-text-muted)', fontWeight: 500, whiteSpace: 'nowrap' }}>{h}</th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {assets.map((a) => {
                  const mc = METHOD_COLORS[a.method_type] || METHOD_COLORS.baseline
                  return (
                    <tr key={a.id} style={{ borderBottom: '1px solid var(--color-border)' }}>
                      <td style={{ padding: '12px 16px' }}>
                        <div
                          role="button"
                          tabIndex={0}
                          onClick={() => setPreviewAsset(a)}
                          onKeyDown={(e) => e.key === 'Enter' && setPreviewAsset(a)}
                          style={{
                            fontWeight: 500, cursor: 'pointer',
                            textDecoration: 'underline',
                            textDecorationColor: 'transparent',
                            transition: 'text-decoration-color 0.15s',
                          }}
                          onMouseEnter={(e) => { e.currentTarget.style.textDecorationColor = 'currentColor' }}
                          onMouseLeave={(e) => { e.currentTarget.style.textDecorationColor = 'transparent' }}
                        >
                          {a.title || a.s3_key}
                        </div>
                        <div style={{ fontSize: '11px', color: 'var(--color-text-muted)', fontFamily: 'monospace' }}>
                          {a.id}
                        </div>
                      </td>
                      <td style={{ padding: '12px 16px' }}>
                        <span style={{ padding: '2px 8px', borderRadius: '99px', fontSize: '12px',
                          fontWeight: 600, background: mc.bg, color: mc.text }}>
                          {a.method_type}
                        </span>
                      </td>
                      <td style={{ padding: '12px 16px', color: 'var(--color-text-muted)' }}>
                        {a.status}
                      </td>
                      <td style={{ padding: '12px 16px' }}>
                        {a.source_item_id ? (
                          <span style={{ fontSize: '12px', color: '#43d98b' }}>✓ привязано</span>
                        ) : (
                          <span style={{ fontSize: '12px', color: 'var(--color-text-muted)' }}>— свободно</span>
                        )}
                      </td>
                      <td style={{ padding: '12px 16px', color: 'var(--color-text-muted)',
                        fontSize: '12px', whiteSpace: 'nowrap' }}>
                        {new Date(a.created_at).toLocaleDateString()}
                      </td>
                      <td style={{ padding: '12px 16px' }}>
                        {!a.source_item_id && (
                          <button
                            className="btn btn-ghost"
                            style={{ fontSize: '12px', padding: '4px 8px', color: '#ff6584' }}
                            onClick={() => handleDeleteAsset(a.id)}
                          >
                            Удалить
                          </button>
                        )}
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* Pagination */}
      {totalPages > 1 && (
        <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: '8px', marginTop: '8px' }}>
          <div style={{ fontSize: '13px', color: 'var(--color-text-muted)' }}>
            Показано {firstShown}–{lastShown} из {total}
          </div>
          <div style={{ display: 'flex', alignItems: 'center', gap: '4px', flexWrap: 'wrap', justifyContent: 'center' }}>
            {/* First page */}
            <button
              className="btn btn-ghost"
              style={{ padding: '4px 8px', fontSize: '13px' }}
              disabled={page === 1}
              onClick={() => setPage(1)}
            >
              «
            </button>
            {/* Prev */}
            <button
              className="btn btn-ghost"
              style={{ padding: '4px 10px', fontSize: '13px' }}
              disabled={page === 1}
              onClick={() => setPage((p) => p - 1)}
            >
              ‹
            </button>

            {buildPageNumbers(page, totalPages).map((p, idx) =>
              p === '...' ? (
                <span key={`ellipsis-${idx}`} style={{ padding: '4px 6px', color: 'var(--color-text-muted)', fontSize: '13px' }}>
                  …
                </span>
              ) : (
                <button
                  key={p}
                  className="btn btn-ghost"
                  style={{
                    padding: '4px 10px', fontSize: '13px', minWidth: '36px',
                    ...(p === page ? {
                      background: 'var(--color-primary)',
                      color: '#fff',
                      borderColor: 'var(--color-primary)',
                    } : {}),
                  }}
                  onClick={() => setPage(p)}
                  disabled={p === page}
                >
                  {p}
                </button>
              )
            )}

            {/* Next */}
            <button
              className="btn btn-ghost"
              style={{ padding: '4px 10px', fontSize: '13px' }}
              disabled={page === totalPages}
              onClick={() => setPage((p) => p + 1)}
            >
              ›
            </button>
            {/* Last page */}
            <button
              className="btn btn-ghost"
              style={{ padding: '4px 8px', fontSize: '13px' }}
              disabled={page === totalPages}
              onClick={() => setPage(totalPages)}
            >
              »
            </button>
          </div>
        </div>
      )}

      {/* Video preview modal */}
      {previewAsset && (
        <VideoModal asset={previewAsset} onClose={() => setPreviewAsset(null)} />
      )}
    </div>
  )
}

function ErrorBox({ message, onClose }) {
  return (
    <div style={{ padding: '12px 16px', background: 'rgba(255,77,109,0.1)',
      border: '1px solid rgba(255,77,109,0.3)', borderRadius: 'var(--radius-sm)',
      color: '#ff6584', fontSize: '14px', display: 'flex', justifyContent: 'space-between' }}>
      <span>{message}</span>
      <button onClick={onClose} style={{ background: 'none', border: 'none', color: '#ff6584', cursor: 'pointer' }}>✕</button>
    </div>
  )
}
