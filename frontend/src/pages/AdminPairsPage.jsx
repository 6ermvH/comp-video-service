import { useState, useEffect } from 'react'
import { api } from '../api/client.js'
import { useApiCall } from '../hooks/useApiCall.js'

export default function AdminPairsPage() {
  const apiCall = useApiCall()

  const [studies, setStudies]           = useState([])
  const [selectedStudy, setSelectedStudy] = useState('')
  const [sourceItems, setSourceItems]   = useState([])
  const [loading, setLoading]           = useState(false)
  const [error, setError]               = useState(null)
  const [successMsg, setSuccessMsg]     = useState(null)

  // Groups state
  const [groups, setGroups]             = useState([])
  const [showGroupForm, setShowGroupForm] = useState(false)
  const [groupForm, setGroupForm]       = useState({ name: '', description: '', priority: 0, target_votes_per_pair: 10 })
  const [creatingGroup, setCreatingGroup] = useState(false)

  // CSV import state
  const [csvFile, setCsvFile]           = useState(null)
  const [importing, setImporting]       = useState(false)

  // Asset upload state
  const [assetFile, setAssetFile]       = useState(null)
  const [assetMeta, setAssetMeta]       = useState({ source_item_id: '', method_type: 'baseline', title: '', description: '' })
  const [uploading, setUploading]       = useState(false)

  // Load studies on mount
  useEffect(() => {
    api.getStudies()
      .then((data) => setStudies(data?.studies ?? []))
      .catch((err) => setError(err.message))
  }, [])

  // Load groups and source items when study changes
  useEffect(() => {
    if (!selectedStudy) {
      setGroups([])
      setSourceItems([])
      return
    }
    setLoading(true)
    Promise.all([
      api.getGroups(selectedStudy),
      api.getSourceItems({ study_id: selectedStudy }),
    ])
      .then(([gData, sData]) => {
        setGroups(gData?.groups ?? [])
        setSourceItems(sData?.source_items ?? [])
      })
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false))
  }, [selectedStudy])

  const handleCreateGroup = async (e) => {
    e.preventDefault()
    setCreatingGroup(true)
    setError(null)
    try {
      await apiCall(() => api.createGroup(selectedStudy, {
        ...groupForm,
        priority: Number(groupForm.priority),
        target_votes_per_pair: Number(groupForm.target_votes_per_pair),
      }))
      setGroupForm({ name: '', description: '', priority: 0, target_votes_per_pair: 10 })
      setShowGroupForm(false)
      const data = await api.getGroups(selectedStudy)
      setGroups(data?.groups ?? [])
    } catch (err) {
      setError(err.message)
    } finally {
      setCreatingGroup(false)
    }
  }

  const handleImport = async (e) => {
    e.preventDefault()
    if (!csvFile || !selectedStudy) return
    const fd = new FormData()
    fd.append('file', csvFile)
    setImporting(true)
    setError(null)
    try {
      const res = await apiCall(() => api.importPairs(selectedStudy, fd))
      setSuccessMsg(`Импортировано пар: ${res?.imported ?? '?'}`)
      setCsvFile(null)
      e.target.reset()
      const data = await api.getSourceItems({ study_id: selectedStudy })
      setSourceItems(data?.source_items ?? [])
    } catch (err) {
      setError(err.message)
    } finally {
      setImporting(false)
    }
  }

  const handleUploadAsset = async (e) => {
    e.preventDefault()
    if (!assetFile || !assetMeta.source_item_id) return
    const fd = new FormData()
    fd.append('file', assetFile)
    fd.append('method_type', assetMeta.method_type)
    if (assetMeta.source_item_id) fd.append('source_item_id', assetMeta.source_item_id)
    if (assetMeta.title)          fd.append('title', assetMeta.title)
    if (assetMeta.description)    fd.append('description', assetMeta.description)
    setUploading(true)
    setError(null)
    try {
      await apiCall(() => api.uploadAsset(fd))
      setSuccessMsg('Видео-ассет успешно загружен')
      setAssetFile(null)
      setAssetMeta({ source_item_id: '', method_type: 'baseline', title: '', description: '' })
      e.target.reset()
    } catch (err) {
      setError(err.message)
    } finally {
      setUploading(false)
    }
  }

  const copyText = (text) => navigator.clipboard.writeText(text)

  const QC_COLORS = {
    ok:      { bg: 'rgba(67,217,139,0.1)',  text: '#43d98b' },
    suspect: { bg: 'rgba(240,180,41,0.1)',  text: '#f0b429' },
    flagged: { bg: 'rgba(255,77,109,0.1)',  text: '#ff6584' },
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '24px' }}>

      <h1 style={{ fontSize: '24px', fontWeight: 700 }}>Пары и ассеты</h1>

      {error && (
        <div style={{ padding: '12px 16px', background: 'rgba(255,77,109,0.1)',
          border: '1px solid rgba(255,77,109,0.3)', borderRadius: 'var(--radius-sm)',
          color: '#ff6584', fontSize: '14px', display: 'flex', justifyContent: 'space-between' }}>
          <span>{error}</span>
          <button onClick={() => setError(null)} style={{ background: 'none', border: 'none', color: '#ff6584', cursor: 'pointer' }}>✕</button>
        </div>
      )}

      {successMsg && (
        <div style={{ padding: '12px 16px', background: 'rgba(67,217,139,0.1)',
          border: '1px solid rgba(67,217,139,0.2)', borderRadius: 'var(--radius-sm)',
          color: '#43d98b', fontSize: '14px', display: 'flex', justifyContent: 'space-between' }}>
          <span>{successMsg}</span>
          <button onClick={() => setSuccessMsg(null)} style={{ background: 'none', border: 'none', color: '#43d98b', cursor: 'pointer' }}>✕</button>
        </div>
      )}

      {/* ── Study selector ─────────────────────────────────── */}
      <div className="card" style={{ padding: '16px' }}>
        <label className="label">Выберите исследование</label>
        <select className="input" style={{ maxWidth: '400px' }}
          value={selectedStudy} onChange={(e) => setSelectedStudy(e.target.value)}>
          <option value="">— Выберите —</option>
          {studies.map((s) => (
            <option key={s.id} value={s.id}>{s.name} ({s.effect_type})</option>
          ))}
        </select>
      </div>

      {selectedStudy && (
        <>
          {/* ── Groups ───────────────────────────────────────── */}
          <div className="card">
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '16px' }}>
              <h2 style={{ fontSize: '17px', fontWeight: 600 }}>
                Группы ({groups.length})
              </h2>
              <button className="btn btn-ghost" style={{ fontSize: '13px', padding: '6px 12px' }}
                onClick={() => setShowGroupForm(!showGroupForm)}>
                + Группа
              </button>
            </div>

            {showGroupForm && (
              <form onSubmit={handleCreateGroup}
                style={{ display: 'flex', flexDirection: 'column', gap: '12px',
                  padding: '16px', background: 'var(--color-surface-2)',
                  borderRadius: 'var(--radius-sm)', marginBottom: '16px' }}>
                <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '12px' }}>
                  <div>
                    <label className="label">Название *</label>
                    <input className="input" required value={groupForm.name}
                      onChange={(e) => setGroupForm({ ...groupForm, name: e.target.value })} />
                  </div>
                  <div>
                    <label className="label">Описание</label>
                    <input className="input" value={groupForm.description}
                      onChange={(e) => setGroupForm({ ...groupForm, description: e.target.value })} />
                  </div>
                  <div>
                    <label className="label">Приоритет</label>
                    <input className="input" type="number" value={groupForm.priority}
                      onChange={(e) => setGroupForm({ ...groupForm, priority: e.target.value })} />
                  </div>
                  <div>
                    <label className="label">Цель ответов на пару</label>
                    <input className="input" type="number" min={1} value={groupForm.target_votes_per_pair}
                      onChange={(e) => setGroupForm({ ...groupForm, target_votes_per_pair: e.target.value })} />
                  </div>
                </div>
                <div style={{ display: 'flex', gap: '8px', justifyContent: 'flex-end' }}>
                  <button type="button" className="btn btn-ghost"
                    onClick={() => setShowGroupForm(false)}>Отмена</button>
                  <button type="submit" className="btn btn-primary" disabled={creatingGroup}>
                    {creatingGroup ? 'Создание…' : 'Создать группу'}
                  </button>
                </div>
              </form>
            )}

            {groups.length === 0 ? (
              <p style={{ fontSize: '14px', color: 'var(--color-text-muted)' }}>
                Нет групп. Создайте первую, чтобы затем импортировать пары.
              </p>
            ) : (
              <>
                <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '13px' }}>
                  <thead>
                    <tr style={{ borderBottom: '1px solid var(--color-border)' }}>
                      {['Название', 'Приоритет', 'Цель ответов', 'UUID'].map((h) => (
                        <th key={h} style={{ textAlign: 'left', padding: '6px 10px',
                          color: 'var(--color-text-muted)', fontWeight: 500 }}>{h}</th>
                      ))}
                    </tr>
                  </thead>
                  <tbody>
                    {groups.map((g) => (
                      <tr key={g.id} style={{ borderBottom: '1px solid var(--color-border)' }}>
                        <td style={{ padding: '8px 10px', fontWeight: 500 }}>{g.name}</td>
                        <td style={{ padding: '8px 10px', color: 'var(--color-text-muted)' }}>{g.priority}</td>
                        <td style={{ padding: '8px 10px', color: 'var(--color-text-muted)' }}>{g.target_votes_per_pair}</td>
                        <td style={{ padding: '8px 10px', display: 'flex', alignItems: 'center', gap: '8px' }}>
                          <code style={{ fontFamily: 'monospace', fontSize: '11px', color: 'var(--color-text-muted)' }}>
                            {g.id}
                          </code>
                          <button onClick={() => copyText(g.id)}
                            title="Копировать UUID"
                            style={{ background: 'none', border: 'none', cursor: 'pointer',
                              color: 'var(--color-text-muted)', fontSize: '13px', padding: '2px 4px' }}>
                            📋
                          </button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
                <p style={{ marginTop: '10px', fontSize: '13px', color: 'var(--color-text-muted)' }}>
                  Скопируйте UUID нужной группы в первую колонку CSV для импорта пар.
                </p>
              </>
            )}
          </div>

          {/* ── CSV import ───────────────────────────────────── */}
          <div className="card">
            <h2 style={{ fontSize: '17px', fontWeight: 600, marginBottom: '16px' }}>CSV-импорт пар</h2>
            <p style={{ fontSize: '13px', color: 'var(--color-text-muted)', marginBottom: '16px' }}>
              Формат: <code style={{ fontFamily: 'monospace', padding: '1px 6px',
                background: 'var(--color-surface-2)', borderRadius: '4px' }}>
                group_id, source_image_id, pair_code, difficulty, is_attention_check, notes
              </code>
            </p>
            <form onSubmit={handleImport} style={{ display: 'flex', alignItems: 'flex-end', gap: '12px' }}>
              <div style={{ flex: 1 }}>
                <label className="label">CSV-файл *</label>
                <input type="file" accept=".csv" className="input"
                  onChange={(e) => setCsvFile(e.target.files[0])} required />
              </div>
              <button type="submit" className="btn btn-primary" disabled={!csvFile || importing}>
                {importing ? 'Импорт…' : 'Импортировать'}
              </button>
            </form>
          </div>

          {/* ── Asset upload ─────────────────────────────────── */}
          <div className="card">
            <h2 style={{ fontSize: '17px', fontWeight: 600, marginBottom: '16px' }}>Загрузить видео-ассет</h2>
            {sourceItems.length === 0 ? (
              <p style={{ color: 'var(--color-text-muted)', fontSize: '14px' }}>
                Сначала импортируйте пары через CSV — затем здесь можно привязать видео к паре.
              </p>
            ) : (
              <form onSubmit={handleUploadAsset} style={{ display: 'flex', flexDirection: 'column', gap: '14px' }}>
                <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '14px' }}>
                  <div>
                    <label className="label">Пара *</label>
                    <select className="input"
                      value={assetMeta.source_item_id}
                      onChange={(e) => setAssetMeta({ ...assetMeta, source_item_id: e.target.value })}>
                      <option value="">— Выберите пару —</option>
                      {sourceItems.map((item) => (
                        <option key={item.id} value={item.id}>
                          {item.pair_code || item.source_image_id || item.id}
                        </option>
                      ))}
                    </select>
                  </div>
                  <div>
                    <label className="label">Тип метода</label>
                    <select className="input" value={assetMeta.method_type}
                      onChange={(e) => setAssetMeta({ ...assetMeta, method_type: e.target.value })}>
                      <option value="baseline">baseline</option>
                      <option value="candidate">candidate</option>
                    </select>
                  </div>
                </div>
                <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '14px' }}>
                  <div>
                    <label className="label">Название (необязательно)</label>
                    <input className="input" placeholder="Название ассета"
                      value={assetMeta.title}
                      onChange={(e) => setAssetMeta({ ...assetMeta, title: e.target.value })} />
                  </div>
                  <div>
                    <label className="label">Описание (необязательно)</label>
                    <input className="input" placeholder="Краткое описание"
                      value={assetMeta.description}
                      onChange={(e) => setAssetMeta({ ...assetMeta, description: e.target.value })} />
                  </div>
                </div>
                <div>
                  <label className="label">Видео-файл (mp4) *</label>
                  <input type="file" accept="video/mp4,video/*" className="input"
                    onChange={(e) => setAssetFile(e.target.files[0])} required />
                </div>
                <div style={{ display: 'flex', justifyContent: 'flex-end' }}>
                  <button type="submit" className="btn btn-primary"
                    disabled={!assetFile || !assetMeta.source_item_id || uploading}>
                    {uploading ? 'Загрузка…' : 'Загрузить'}
                  </button>
                </div>
              </form>
            )}
          </div>

          {/* ── Source items list ────────────────────────────── */}
          <div>
            <h2 style={{ fontSize: '17px', fontWeight: 600, marginBottom: '12px' }}>
              Пары ({sourceItems.length})
            </h2>
            {loading ? (
              <div style={{ display: 'flex', justifyContent: 'center', padding: '32px' }}>
                <div className="spinner" />
              </div>
            ) : sourceItems.length === 0 ? (
              <div style={{ textAlign: 'center', padding: '32px', color: 'var(--color-text-muted)',
                border: '1px dashed var(--color-border)', borderRadius: 'var(--radius-md)' }}>
                Нет пар. Импортируйте CSV.
              </div>
            ) : (
              <div style={{ overflowX: 'auto' }}>
                <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '13px' }}>
                  <thead>
                    <tr style={{ borderBottom: '1px solid var(--color-border)' }}>
                      {['Pair Code', 'Группа', 'Сложность', 'Ассеты', 'Ответы', 'QC', 'Attention'].map((h) => (
                        <th key={h} style={{ textAlign: 'left', padding: '8px 12px',
                          color: 'var(--color-text-muted)', fontWeight: 500 }}>{h}</th>
                      ))}
                    </tr>
                  </thead>
                  <tbody>
                    {sourceItems.map((item) => {
                      const qcColor = QC_COLORS[item.qc_flag] || QC_COLORS.ok
                      return (
                        <tr key={item.id} style={{ borderBottom: '1px solid var(--color-border)' }}>
                          <td style={{ padding: '10px 12px', fontFamily: 'monospace' }}>
                            {item.pair_code || '—'}
                          </td>
                          <td style={{ padding: '10px 12px', color: 'var(--color-text-muted)' }}>
                            {item.group_name || '—'}
                          </td>
                          <td style={{ padding: '10px 12px' }}>
                            {item.difficulty
                              ? <span style={{ padding: '2px 8px', borderRadius: '4px',
                                  background: 'var(--color-surface-2)', fontSize: '12px' }}>
                                  {item.difficulty}
                                </span>
                              : '—'}
                          </td>
                          <td style={{ padding: '10px 12px' }}>{item.asset_count ?? '—'}</td>
                          <td style={{ padding: '10px 12px' }}>{item.response_count ?? '—'}</td>
                          <td style={{ padding: '10px 12px' }}>
                            {item.qc_flag && (
                              <span style={{ padding: '2px 8px', borderRadius: '4px', fontSize: '12px',
                                background: qcColor.bg, color: qcColor.text }}>
                                {item.qc_flag}
                              </span>
                            )}
                          </td>
                          <td style={{ padding: '10px 12px', textAlign: 'center' }}>
                            {item.is_attention_check ? '✓' : ''}
                          </td>
                        </tr>
                      )
                    })}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        </>
      )}
    </div>
  )
}
