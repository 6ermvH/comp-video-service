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

  // Pair builder state
  const [freeAssets, setFreeAssets]       = useState([])
  const [pairForm, setPairForm]           = useState({ baseline_video_id: '', candidate_video_id: '', group_id: '', pair_code: '', difficulty: '' })
  const [creatingPair, setCreatingPair]   = useState(false)
  const [showPairBuilder, setShowPairBuilder] = useState(false)

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
      api.getFreeAssets(),
    ])
      .then(([gData, sData, aData]) => {
        setGroups(gData?.groups ?? [])
        setSourceItems(sData?.source_items ?? [])
        setFreeAssets(aData?.assets ?? [])
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

  const handleCreatePair = async (e) => {
    e.preventDefault()
    if (!selectedStudy) return
    if (!pairForm.baseline_video_id || !pairForm.candidate_video_id) return
    setCreatingPair(true)
    setError(null)
    try {
      await apiCall(() => api.createPair(selectedStudy, {
        group_id: pairForm.group_id,
        baseline_video_id: pairForm.baseline_video_id,
        candidate_video_id: pairForm.candidate_video_id,
        pair_code: pairForm.pair_code,
        difficulty: pairForm.difficulty,
      }))
      setSuccessMsg('Пара создана')
      setPairForm({ baseline_video_id: '', candidate_video_id: '', group_id: '', pair_code: '', difficulty: '' })
      setShowPairBuilder(false)
      const [sData, aData] = await Promise.all([
        api.getSourceItems({ study_id: selectedStudy }),
        api.getFreeAssets(),
      ])
      setSourceItems(sData?.source_items ?? [])
      setFreeAssets(aData?.assets ?? [])
    } catch (err) {
      setError(err.message)
    } finally {
      setCreatingPair(false)
    }
  }

  const handleDeletePair = async (id) => {
    if (!window.confirm('Удалить пару? Видео вернутся в библиотеку.')) return
    try {
      await api.deletePair(id)
      setSuccessMsg('Пара удалена, видео возвращены в библиотеку')
      const data = await api.getSourceItems({ study_id: selectedStudy })
      setSourceItems(data?.source_items ?? [])
    } catch (err) {
      if (err.status === 409) {
        setError('Нельзя удалить: есть ответы участников для этой пары.')
      } else {
        setError(err.message)
      }
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
                <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(240px, 1fr))', gap: '12px' }}>
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
              <div style={{ overflowX: 'auto' }}>
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
              </div>
                <p style={{ marginTop: '10px', fontSize: '13px', color: 'var(--color-text-muted)' }}>
                  Скопируйте UUID нужной группы в первую колонку CSV для импорта пар.
                </p>
              </>
            )}
          </div>

          {/* ── Pair builder ─────────────────────────────────── */}
          {selectedStudy && (
            <div className="card">
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: showPairBuilder ? '16px' : 0 }}>
                <h2 style={{ fontSize: '16px', fontWeight: 600 }}>Создать пару из библиотеки</h2>
                <button className="btn btn-ghost" style={{ fontSize: '13px' }}
                  onClick={() => setShowPairBuilder(!showPairBuilder)}>
                  {showPairBuilder ? '▲ Свернуть' : '▼ Развернуть'}
                </button>
              </div>

              {showPairBuilder && (
                <form onSubmit={handleCreatePair} style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
                  <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))', gap: '12px' }}>
                    <div>
                      <label className="label">Группа *</label>
                      <select className="input" required value={pairForm.group_id}
                        onChange={(e) => setPairForm({ ...pairForm, group_id: e.target.value })}>
                        <option value="">— выберите группу —</option>
                        {groups.map((g) => <option key={g.id} value={g.id}>{g.name}</option>)}
                      </select>
                    </div>
                    <div>
                      <label className="label">Baseline видео *</label>
                      <VideoCombobox
                        options={freeAssets.filter((a) => a.method_type === 'baseline')}
                        value={pairForm.baseline_video_id}
                        onChange={(id) => setPairForm({ ...pairForm, baseline_video_id: id })}
                        placeholder="Поиск baseline..."
                      />
                    </div>
                    <div>
                      <label className="label">Candidate видео *</label>
                      <VideoCombobox
                        options={freeAssets.filter((a) => a.method_type === 'candidate')}
                        value={pairForm.candidate_video_id}
                        onChange={(id) => setPairForm({ ...pairForm, candidate_video_id: id })}
                        placeholder="Поиск candidate..."
                      />
                    </div>
                  </div>
                  <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))', gap: '12px' }}>
                    <div>
                      <label className="label">Код пары</label>
                      <input className="input" value={pairForm.pair_code}
                        onChange={(e) => setPairForm({ ...pairForm, pair_code: e.target.value })} />
                    </div>
                    <div>
                      <label className="label">Сложность</label>
                      <select className="input" value={pairForm.difficulty}
                        onChange={(e) => setPairForm({ ...pairForm, difficulty: e.target.value })}>
                        <option value="">—</option>
                        <option value="easy">easy</option>
                        <option value="medium">medium</option>
                        <option value="hard">hard</option>
                      </select>
                    </div>
                  </div>
                  <div style={{ display: 'flex', justifyContent: 'flex-end' }}>
                    <button type="submit" className="btn btn-primary" disabled={creatingPair}>
                      {creatingPair ? 'Создание…' : '+ Создать пару'}
                    </button>
                  </div>
                </form>
              )}
            </div>
          )}

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
                      {['Pair Code', 'Группа', 'Сложность', 'Ассеты', 'Ответы', 'QC', 'Attention', ''].map((h) => (
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
                          <td style={{ padding: '10px 12px' }}>
                            <button
                              className="btn btn-ghost"
                              style={{ fontSize: '12px', padding: '4px 8px', color: '#ff6584' }}
                              onClick={() => handleDeletePair(item.id)}
                            >
                              🗑 Удалить
                            </button>
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

function VideoCombobox({ options, value, onChange, placeholder }) {
  const [query, setQuery] = useState('')
  const [open, setOpen] = useState(false)

  const selected = options.find((o) => o.id === value)
  const filtered = options
    .filter((o) => (o.title || o.s3_key).toLowerCase().includes(query.toLowerCase()))
    .slice(0, 30)

  const handleSelect = (id) => {
    onChange(id)
    setQuery('')
    setOpen(false)
  }

  return (
    <div style={{ position: 'relative' }}>
      <input
        className="input"
        placeholder={placeholder}
        value={open ? query : (selected ? (selected.title || selected.s3_key) : '')}
        onChange={(e) => { setQuery(e.target.value); onChange(''); setOpen(true) }}
        onFocus={() => setOpen(true)}
        onBlur={() => setTimeout(() => setOpen(false), 150)}
      />
      {open && (
        <div style={{
          position: 'absolute', zIndex: 20, width: '100%',
          background: 'var(--color-surface)', border: '1px solid var(--color-border)',
          borderRadius: 'var(--radius-sm)', maxHeight: '220px', overflowY: 'auto',
          boxShadow: '0 4px 16px rgba(0,0,0,0.3)', marginTop: '2px',
        }}>
          {filtered.length === 0 ? (
            <div style={{ padding: '10px 12px', fontSize: '13px', color: 'var(--color-text-muted)' }}>
              Ничего не найдено
            </div>
          ) : filtered.map((o) => (
            <div
              key={o.id}
              onMouseDown={() => handleSelect(o.id)}
              style={{
                padding: '8px 12px', cursor: 'pointer', fontSize: '13px',
                background: o.id === value ? 'rgba(108,99,255,0.15)' : 'transparent',
              }}
              onMouseEnter={(e) => { e.currentTarget.style.background = 'rgba(108,99,255,0.1)' }}
              onMouseLeave={(e) => { e.currentTarget.style.background = o.id === value ? 'rgba(108,99,255,0.15)' : 'transparent' }}
            >
              {o.title || o.s3_key}
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
