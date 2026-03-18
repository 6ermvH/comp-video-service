import { useState, useEffect } from 'react'
import { api } from '../api/client.js'
import { useApiCall } from '../hooks/useApiCall.js'

const METHOD_COLORS = {
  baseline:  { bg: 'rgba(108,99,255,0.15)', text: '#a78bfa' },
  candidate: { bg: 'rgba(67,217,139,0.15)',  text: '#43d98b' },
}

export default function AdminVideoLibraryPage() {
  const apiCall = useApiCall()
  const [assets, setAssets] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [successMsg, setSuccessMsg] = useState(null)

  // Upload form state
  const [baselineFile, setBaselineFile] = useState(null)
  const [candidateFile, setCandidateFile] = useState(null)
  const [uploading, setUploading] = useState(false)

  const load = async () => {
    setLoading(true)
    try {
      const data = await apiCall(() => api.getAssets(), { onRetry: load })
      setAssets(data?.assets ?? [])
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  // eslint-disable-next-line react-hooks/exhaustive-deps
  useEffect(() => { load() }, [])

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
        fd.append('title', baselineFile.name.replace(/\.[^.]+$/, ''))
        await apiCall(() => api.uploadAsset(fd))
      }
      if (candidateFile) {
        const fd = new FormData()
        fd.append('file', candidateFile)
        fd.append('method_type', 'candidate')
        fd.append('title', candidateFile.name.replace(/\.[^.]+$/, ''))
        await apiCall(() => api.uploadAsset(fd))
      }
      setSuccessMsg('Baseline и candidate загружены')
      setBaselineFile(null)
      setCandidateFile(null)
      e.target.reset()
      load()
    } catch (err) {
      setError(err.message)
    } finally {
      setUploading(false)
    }
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '24px' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <h1 style={{ fontSize: '24px', fontWeight: 700 }}>Видеотека</h1>
        <button className="btn btn-ghost" onClick={load}>↻ Обновить</button>
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

      {/* Upload form */}
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

      {/* Assets list */}
      {loading ? (
        <div style={{ display: 'flex', justifyContent: 'center', padding: '48px' }}>
          <div className="spinner" />
        </div>
      ) : assets.length === 0 ? (
        <div style={{ textAlign: 'center', padding: '48px', color: 'var(--color-text-muted)' }}>
          Нет загруженных видео.
        </div>
      ) : (
        <div className="card" style={{ padding: 0, overflow: 'hidden' }}>
          <div style={{ overflowX: 'auto' }}>
            <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '14px' }}>
              <thead>
                <tr style={{ borderBottom: '1px solid var(--color-border)' }}>
                  {['Название', 'Тип', 'Статус', 'Привязка к паре', 'Добавлено'].map((h) => (
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
                        <div style={{ fontWeight: 500 }}>{a.title || a.s3_key}</div>
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
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>
        </div>
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
