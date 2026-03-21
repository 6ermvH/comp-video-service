import { useState, useEffect } from 'react'
import { api } from '../api/client.js'
import StatsChart from '../components/StatsChart.jsx'
import { useApiCall } from '../hooks/useApiCall.js'

export default function AdminAnalyticsPage() {
  const apiCall = useApiCall()
  const [overview, setOverview] = useState(null)
  const [qcReport, setQcReport] = useState(null)
  const [studies, setStudies] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [exporting, setExporting] = useState(null) // 'csv' | 'json' | null

  const [selectedStudyId, setSelectedStudyId] = useState(null)
  const [pairStats, setPairStats] = useState([])
  const [loadingPairs, setLoadingPairs] = useState(false)
  const [exportingStudyCsv, setExportingStudyCsv] = useState(false)

  const load = async () => {
    setLoading(true)
    setError(null)
    try {
      const [ov, qc, st] = await Promise.allSettled([
        apiCall(() => api.getAnalyticsOverview(), { onRetry: load }),
        apiCall(() => api.getQCReport(), { onRetry: load }),
        api.getStudies(),
      ])
      if (ov.status === 'fulfilled') setOverview(ov.value)
      if (qc.status === 'fulfilled') setQcReport(qc.value)
      if (st.status === 'fulfilled') setStudies(st.value?.studies ?? st.value ?? [])
      if (ov.status === 'rejected') setError(ov.reason.message)
    } finally {
      setLoading(false)
    }
  }

  // eslint-disable-next-line react-hooks/exhaustive-deps
  useEffect(() => { load() }, [])

  useEffect(() => {
    if (!selectedStudyId) { setPairStats([]); return }
    setLoadingPairs(true)
    api.getAnalyticsPairs(selectedStudyId)
      .then((data) => setPairStats(Array.isArray(data) ? data : []))
      .catch(() => setPairStats([]))
      .finally(() => setLoadingPairs(false))
  }, [selectedStudyId])

  const downloadBlob = (blob, filename) => {
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = filename
    a.click()
    URL.revokeObjectURL(url)
  }

  const handleExport = async (format) => {
    setExporting(format)
    try {
      if (format === 'csv') {
        const blob = await api.exportCSV()
        downloadBlob(blob, 'responses_export.csv')
      } else {
        const blob = await api.exportJSON()
        downloadBlob(blob, 'responses_export.json')
      }
    } catch (err) {
      setError(err.message)
    } finally {
      setExporting(null)
    }
  }

  const ov = overview || {}

  const winRateData = (ov.effects ?? []).map((e) => ({
    name: e.effect_type,
    win_rate: Math.round(e.candidate_win_rate * 100),
  }))

  const groupData = (ov.groups ?? []).map((g) => ({
    name: g.group_name,
    responses: g.responses,
  }))

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '24px' }}>

      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <h1 style={{ fontSize: '24px', fontWeight: 700 }}>Аналитика</h1>
        <div style={{ display: 'flex', gap: '8px' }}>
          <button className="btn btn-ghost" onClick={load} disabled={loading}>
            ↻ Обновить
          </button>
          <button className="btn btn-ghost"
            onClick={() => handleExport('csv')} disabled={exporting === 'csv'}>
            {exporting === 'csv' ? '…' : '⬇ CSV'}
          </button>
          <button className="btn btn-ghost"
            onClick={() => handleExport('json')} disabled={exporting === 'json'}>
            {exporting === 'json' ? '…' : '⬇ JSON'}
          </button>
        </div>
      </div>

      {error && (
        <div style={{ padding: '12px 16px', background: 'rgba(255,77,109,0.1)',
          border: '1px solid rgba(255,77,109,0.3)', borderRadius: 'var(--radius-sm)',
          color: '#ff6584', fontSize: '14px' }}>
          {error}
        </div>
      )}

      {loading ? (
        <div style={{ display: 'flex', justifyContent: 'center', padding: '64px' }}>
          <div className="spinner" />
        </div>
      ) : (
        <>
          {/* KPI row */}
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(180px, 1fr))', gap: '16px' }}>
            {[
              { label: 'Всего ответов',       value: ov.total_responses ?? '—' },
              { label: 'Участников',           value: ov.total_participants ?? '—' },
              { label: 'Пар',                  value: ov.total_source_items ?? '—' },
              { label: 'Candidate win rate',   value: ov.candidate_win_rate != null ? `${Math.round(ov.candidate_win_rate * 100)}%` : '—' },
              { label: 'Completion rate',      value: ov.completion_rate != null ? `${Math.round(ov.completion_rate * 100)}%` : '—' },
              { label: 'Tie rate',             value: ov.tie_rate != null ? `${Math.round(ov.tie_rate * 100)}%` : '—' },
            ].map(({ label, value }) => (
              <div key={label} className="card" style={{ padding: '20px', textAlign: 'center' }}>
                <div style={{ fontSize: '28px', fontWeight: 700, marginBottom: '6px' }}>{value}</div>
                <div style={{ fontSize: '13px', color: 'var(--color-text-muted)' }}>{label}</div>
              </div>
            ))}
          </div>

          {/* Charts row */}
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(320px, 1fr))', gap: '16px' }}>
            <div className="card">
              <h2 style={{ fontSize: '16px', fontWeight: 600, marginBottom: '16px' }}>
                Win rate candidate по типу эффекта
              </h2>
              {winRateData.length > 0 ? (
                <StatsChart
                  data={winRateData}
                  xDataKey="name"
                  barDataKey="win_rate"
                  barColor="var(--color-primary)"
                />
              ) : (
                <EmptyChart />
              )}
            </div>

            <div className="card">
              <h2 style={{ fontSize: '16px', fontWeight: 600, marginBottom: '16px' }}>
                Ответы по группам
              </h2>
              {groupData.length > 0 ? (
                <StatsChart
                  data={groupData}
                  xDataKey="name"
                  barDataKey="responses"
                  barColor="var(--color-success)"
                />
              ) : (
                <EmptyChart />
              )}
            </div>
          </div>

          {/* Per-pair breakdown */}
          <div className="card">
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '16px' }}>
              <h2 style={{ fontSize: '16px', fontWeight: 600 }}>
                Статистика по парам
              </h2>
              {selectedStudyId && (
                <button
                  className="btn btn-ghost"
                  disabled={exportingStudyCsv}
                  onClick={async () => {
                    setExportingStudyCsv(true)
                    try {
                      const blob = await api.exportStudyCSV(selectedStudyId)
                      downloadBlob(blob, `pairs_export_${selectedStudyId}.csv`)
                    } catch (err) {
                      setError(err.message)
                    } finally {
                      setExportingStudyCsv(false)
                    }
                  }}
                >
                  {exportingStudyCsv ? '…' : '⬇ CSV по парам'}
                </button>
              )}
            </div>
            <select
              value={selectedStudyId ?? ''}
              onChange={(e) => setSelectedStudyId(e.target.value || null)}
              className="input"
              style={{ maxWidth: '300px', marginBottom: '16px' }}
            >
              <option value="">— Выбрать исследование —</option>
              {studies.map((s) => (
                <option key={s.id} value={s.id}>{s.name}</option>
              ))}
            </select>

            {loadingPairs ? (
              <div style={{ display: 'flex', justifyContent: 'center', padding: '32px' }}>
                <div className="spinner" />
              </div>
            ) : pairStats.length > 0 ? (
              <div style={{ overflowX: 'auto' }}>
                <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '14px' }}>
                  <thead>
                    <tr style={{ borderBottom: '1px solid var(--color-border)' }}>
                      {['Пара', 'Группа', 'Сложность', 'Ответов', 'Candidate wins', 'Baseline wins', 'Tie', 'Win rate'].map((h) => (
                        <th key={h} style={{ textAlign: 'left', padding: '8px 12px',
                          color: 'var(--color-text-muted)', fontWeight: 500 }}>{h}</th>
                      ))}
                    </tr>
                  </thead>
                  <tbody>
                    {pairStats.map((p) => (
                      <tr key={p.source_item_id} style={{ borderBottom: '1px solid var(--color-border)' }}>
                        <td style={{ padding: '10px 12px', fontFamily: 'monospace', fontSize: '12px' }}>{p.pair_code ?? '—'}</td>
                        <td style={{ padding: '10px 12px' }}>{p.group_name}</td>
                        <td style={{ padding: '10px 12px' }}>{p.difficulty ?? '—'}</td>
                        <td style={{ padding: '10px 12px' }}>{p.total_responses}</td>
                        <td style={{ padding: '10px 12px', color: 'var(--color-success)' }}>{p.candidate_wins}</td>
                        <td style={{ padding: '10px 12px' }}>{p.baseline_wins}</td>
                        <td style={{ padding: '10px 12px' }}>{p.tie_count}</td>
                        <td style={{ padding: '10px 12px', fontWeight: 600 }}>
                          {p.total_responses > 0 ? `${Math.round(p.candidate_win_rate * 100)}%` : '—'}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            ) : selectedStudyId ? (
              <p style={{ fontSize: '14px', color: 'var(--color-text-muted)' }}>Нет данных для выбранного исследования.</p>
            ) : null}
          </div>

          {/* QC Report */}
          {qcReport && (
            <div className="card">
              <h2 style={{ fontSize: '16px', fontWeight: 600, marginBottom: '16px' }}>
                QC-отчёт
              </h2>
              <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(160px, 1fr))', gap: '16px', marginBottom: '20px' }}>
                {[
                  { label: 'Быстрые ответы',    value: qcReport.fast_responses ?? '—',           warn: true },
                  { label: 'Прямолинейные',       value: qcReport.straight_lining ?? '—',          warn: true },
                  { label: 'Провал attention',    value: qcReport.attention_check_failures ?? '—', warn: true },
                  { label: 'Помечено suspect',    value: qcReport.suspect_count ?? '—',            warn: false },
                ].map(({ label, value, warn }) => (
                  <div key={label} style={{
                    padding: '16px', background: 'var(--color-surface-2)',
                    borderRadius: 'var(--radius-sm)', textAlign: 'center',
                  }}>
                    <div style={{
                      fontSize: '22px', fontWeight: 700, marginBottom: '4px',
                      color: warn && value > 0 ? 'var(--color-warning)' : 'var(--color-text)',
                    }}>
                      {value}
                    </div>
                    <div style={{ fontSize: '12px', color: 'var(--color-text-muted)' }}>{label}</div>
                  </div>
                ))}
              </div>

              {qcReport.flagged_participants?.length > 0 && (
                <>
                  <h3 style={{ fontSize: '14px', fontWeight: 600, marginBottom: '12px', color: 'var(--color-text-muted)' }}>
                    Подозрительные участники
                  </h3>
                  <div style={{ overflowX: 'auto' }}>
                    <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '13px' }}>
                      <thead>
                        <tr style={{ borderBottom: '1px solid var(--color-border)' }}>
                          {['Participant ID', 'Причина', 'Ответов', 'Ср. время (мс)'].map((h) => (
                            <th key={h} style={{ textAlign: 'left', padding: '6px 10px',
                              color: 'var(--color-text-muted)', fontWeight: 500 }}>{h}</th>
                          ))}
                        </tr>
                      </thead>
                      <tbody>
                        {qcReport.flagged_participants.map((p) => (
                          <tr key={p.id} style={{ borderBottom: '1px solid var(--color-border)' }}>
                            <td style={{ padding: '8px 10px', fontFamily: 'monospace', fontSize: '11px' }}>{p.id}</td>
                            <td style={{ padding: '8px 10px', color: 'var(--color-warning)' }}>{p.flag_reason}</td>
                            <td style={{ padding: '8px 10px' }}>{p.response_count}</td>
                            <td style={{ padding: '8px 10px' }}>{p.avg_response_ms}</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                </>
              )}
            </div>
          )}
        </>
      )}
    </div>
  )
}

function EmptyChart() {
  return (
    <div style={{ height: '200px', display: 'flex', alignItems: 'center', justifyContent: 'center',
      color: 'var(--color-text-muted)', fontSize: '14px', background: 'var(--color-surface-2)',
      borderRadius: 'var(--radius-sm)' }}>
      Нет данных
    </div>
  )
}
