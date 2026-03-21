import { useState, useEffect, useRef } from 'react'
import { api } from '../api/client.js'
import { useApiCall } from '../hooks/useApiCall.js'

const STATUS_COLORS = {
  draft:    { bg: 'rgba(108,99,255,0.15)', text: '#a78bfa' },
  active:   { bg: 'rgba(67,217,139,0.15)', text: '#43d98b' },
  paused:   { bg: 'rgba(240,180,41,0.15)', text: '#f0b429' },
  archived: { bg: 'rgba(139,147,168,0.1)', text: '#8a93a8' },
}

const STATUS_TRANSITIONS = {
  draft:    ['active'],
  active:   ['paused', 'archived'],
  paused:   ['active', 'archived'],
  archived: ['active'],
}

const STATUS_LABELS = {
  draft: 'Черновик', active: 'Активно', paused: 'Пауза', archived: 'Архив',
}

const DEFAULT_EFFECT_TYPES = ['flooding', 'explosion', 'mixed']

const DEFAULT_INSTRUCTIONS = `Будут представлены видео А и B, выберите лучшее видео в целом.

Обращайте внимание на следующие факторы:
  • реализм
  • стабильность (плавность и согласованность движения)
  • отсутствие артефактов
  • общая визуальная целостность (одна связная сцена)

Если затрудняетесь определить лучшее видео, выберите «Затрудняюсь ответить»

Вы можете повторить просмотр неограниченное количество раз. Пожалуйста, не спешите — качество ваших оценок важнее скорости.

После подтверждения выбора и перехода к следующей паре видео, вы не сможете вернуться назад и изменить свой выбор.`

export default function AdminStudiesPage() {
  const apiCall = useApiCall()
  const [studies, setStudies] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [showForm, setShowForm] = useState(false)
  const [form, setForm] = useState({
    name: '',
    effect_type: 'flooding',
    max_tasks_per_participant: 20,
    instructions_text: '',
    tie_option_enabled: true,
    reasons_enabled: true,
    confidence_enabled: true,
  })
  const [creating, setCreating] = useState(false)
  const [editStudy, setEditStudy] = useState(null)
  const [editForm, setEditForm] = useState({})
  const [saving, setSaving] = useState(false)

  // Import archive state
  const [showImportForm, setShowImportForm] = useState(false)
  const [importForm, setImportForm] = useState({
    name: '',
    effect_type: 'flooding',
    max_tasks_per_participant: 20,
    tie_option_enabled: true,
    reasons_enabled: true,
    confidence_enabled: true,
  })
  const [importFile, setImportFile] = useState(null)
  const [importing, setImporting] = useState(false)
  const [importResult, setImportResult] = useState(null)
  const [importError, setImportError] = useState(null)
  const [dragOver, setDragOver] = useState(false)
  const importFileRef = useRef(null)

  const uniqueEffectTypes = studies.length > 0
    ? [...new Set(studies.map((s) => s.effect_type).filter(Boolean))]
    : DEFAULT_EFFECT_TYPES

  const load = async () => {
    setLoading(true)
    setError(null)
    try {
      const data = await apiCall(() => api.getStudies(), { onRetry: load })
      setStudies(data?.studies || data || [])
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  // eslint-disable-next-line react-hooks/exhaustive-deps
  useEffect(() => { load() }, [])

  const handleCreate = async (e) => {
    e.preventDefault()
    setCreating(true)
    try {
      await apiCall(() => api.createStudy({
        ...form,
        max_tasks_per_participant: Number(form.max_tasks_per_participant),
      }))
      setShowForm(false)
      setForm({
        name: '', effect_type: 'flooding', max_tasks_per_participant: 20,
        instructions_text: '', tie_option_enabled: true,
        reasons_enabled: true, confidence_enabled: true,
      })
      load()
    } catch (err) {
      setError(err.message)
    } finally {
      setCreating(false)
    }
  }

  const handleEditSave = async (e) => {
    e.preventDefault()
    setSaving(true)
    try {
      await apiCall(() => api.updateStudy(editStudy.id, {
        ...editForm,
        max_tasks_per_participant: Number(editForm.max_tasks_per_participant),
      }))
      setEditStudy(null)
      load()
    } catch (err) {
      setError(err.message)
    } finally {
      setSaving(false)
    }
  }

  const handleDelete = async (study) => {
    if (!window.confirm('Удалить исследование и все связанные данные (группы, пары, ответы)? Это действие необратимо!')) return
    try {
      await api.deleteStudy(study.id)
      load()
    } catch (err) {
      setError(err.message)
    }
  }

  const handleStatusChange = async (studyId, newStatus) => {
    try {
      await api.updateStudy(studyId, { status: newStatus })
      load()
    } catch (err) {
      setError(err.message)
    }
  }

  const handleImport = async (e) => {
    e.preventDefault()
    if (!importFile) return
    setImporting(true)
    setImportError(null)
    setImportResult(null)
    try {
      const fd = new FormData()
      fd.append('file', importFile)
      fd.append('name', importForm.name)
      fd.append('effect_type', importForm.effect_type)
      fd.append('max_tasks_per_participant', String(Number(importForm.max_tasks_per_participant)))
      fd.append('tie_option_enabled', String(importForm.tie_option_enabled))
      fd.append('reasons_enabled', String(importForm.reasons_enabled))
      fd.append('confidence_enabled', String(importForm.confidence_enabled))
      const result = await apiCall(() => api.importStudyArchive(fd))
      setImportResult(result)
      setImportFile(null)
      setImportForm({
        name: '', effect_type: 'flooding', max_tasks_per_participant: 20,
        tie_option_enabled: true, reasons_enabled: true, confidence_enabled: true,
      })
      if (importFileRef.current) importFileRef.current.value = ''
      load()
    } catch (err) {
      setImportError(err.message)
    } finally {
      setImporting(false)
    }
  }

  const handleImportDrop = (e) => {
    e.preventDefault()
    setDragOver(false)
    const file = e.dataTransfer.files[0]
    if (file && (file.name.endsWith('.zip') || file.type === 'application/zip')) {
      setImportFile(file)
    }
  }

  const copyLink = (studyId) => {
    const url = `${window.location.origin}/?study_id=${studyId}`
    navigator.clipboard.writeText(url)
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '24px' }}>

      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <h1 style={{ fontSize: '24px', fontWeight: 700 }}>Исследования</h1>
        <div style={{ display: 'flex', gap: '8px' }}>
          <button className="btn btn-ghost" onClick={load}>↻ Обновить</button>
          <button
            className="btn btn-ghost"
            onClick={() => {
              setShowImportForm(!showImportForm)
              setShowForm(false)
              setImportResult(null)
              setImportError(null)
            }}
          >
            ⬆ Импорт из архива
          </button>
          <button className="btn btn-primary" onClick={() => { setShowForm(!showForm); setShowImportForm(false) }}>
            + Создать
          </button>
        </div>
      </div>

      {error && <ErrorBox message={error} onClose={() => setError(null)} />}

      {/* Import archive form */}
      {showImportForm && (
        <div className="card">
          <h2 style={{ fontSize: '18px', fontWeight: 600, marginBottom: '4px' }}>
            Импорт исследования из архива
          </h2>
          <p style={{ fontSize: '13px', color: 'var(--color-text-muted)', marginBottom: '20px' }}>
            Архив должен содержать MP4 файлы с именами в формате:{' '}
            <code style={{ fontFamily: 'monospace', background: 'var(--color-surface-2)',
              padding: '2px 6px', borderRadius: '4px' }}>
              &lt;группа&gt;_&lt;название&gt;_&lt;candidate|baseline&gt;.mp4
            </code>
          </p>

          {importError && (
            <div style={{ marginBottom: '16px' }}>
              <ErrorBox message={importError} onClose={() => setImportError(null)} />
            </div>
          )}

          {importResult && (
            <ImportResultBox result={importResult} onClose={() => setImportResult(null)} />
          )}

          <form onSubmit={handleImport} style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
            {/* Drop zone */}
            <div
              onDragOver={(e) => { e.preventDefault(); setDragOver(true) }}
              onDragLeave={() => setDragOver(false)}
              onDrop={handleImportDrop}
              onClick={() => importFileRef.current?.click()}
              style={{
                border: `2px dashed ${dragOver ? '#6c63ff' : 'var(--color-border)'}`,
                borderRadius: 'var(--radius-sm)',
                padding: '28px 16px',
                textAlign: 'center',
                cursor: 'pointer',
                background: dragOver ? 'rgba(108,99,255,0.07)' : 'transparent',
                transition: 'border-color 0.15s, background 0.15s',
              }}
            >
              <input
                ref={importFileRef}
                type="file"
                accept=".zip,application/zip"
                style={{ display: 'none' }}
                onChange={(e) => setImportFile(e.target.files[0] || null)}
              />
              {importFile ? (
                <div>
                  <div style={{ fontSize: '20px', marginBottom: '4px' }}>📦</div>
                  <div style={{ fontSize: '14px', fontWeight: 500 }}>{importFile.name}</div>
                  <div style={{ fontSize: '12px', color: 'var(--color-text-muted)', marginTop: '4px' }}>
                    {(importFile.size / 1024 / 1024).toFixed(1)} МБ
                  </div>
                  <button
                    type="button"
                    onClick={(e) => { e.stopPropagation(); setImportFile(null); if (importFileRef.current) importFileRef.current.value = '' }}
                    style={{ marginTop: '8px', fontSize: '12px', background: 'none', border: 'none',
                      color: 'var(--color-text-muted)', cursor: 'pointer', textDecoration: 'underline' }}
                  >
                    Выбрать другой
                  </button>
                </div>
              ) : (
                <div>
                  <div style={{ fontSize: '20px', marginBottom: '4px' }}>📁</div>
                  <div style={{ fontSize: '14px', color: 'var(--color-text-muted)' }}>
                    Перетащите ZIP-архив или{' '}
                    <span style={{ color: '#6c63ff', textDecoration: 'underline' }}>выберите файл</span>
                  </div>
                </div>
              )}
            </div>

            {/* Name + Effect type */}
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(240px, 1fr))', gap: '16px' }}>
              <div>
                <label className="label">Название исследования *</label>
                <input
                  className="input"
                  required
                  value={importForm.name}
                  onChange={(e) => setImportForm({ ...importForm, name: e.target.value })}
                />
              </div>
              <div>
                <label className="label">Тип эффекта</label>
                <input
                  className="input"
                  list="effect-types-import"
                  placeholder="Введите или выберите тип..."
                  value={importForm.effect_type}
                  onChange={(e) => setImportForm({ ...importForm, effect_type: e.target.value })}
                />
                <datalist id="effect-types-import">
                  {uniqueEffectTypes.map((t) => <option key={t} value={t} />)}
                </datalist>
              </div>
            </div>

            {/* Max tasks */}
            <div>
              <label className="label">Заданий на участника</label>
              <input
                className="input"
                type="number"
                min={1}
                max={100}
                value={importForm.max_tasks_per_participant}
                onChange={(e) => setImportForm({ ...importForm, max_tasks_per_participant: e.target.value })}
                style={{ maxWidth: '160px' }}
              />
            </div>

            {/* Checkboxes */}
            <div style={{ display: 'flex', gap: '24px', flexWrap: 'wrap' }}>
              {[
                { key: 'tie_option_enabled', label: 'Опция "Равны"' },
                { key: 'reasons_enabled', label: 'Причины выбора' },
                { key: 'confidence_enabled', label: 'Уверенность' },
              ].map(({ key, label }) => (
                <label key={key} style={{ display: 'flex', alignItems: 'center', gap: '8px', cursor: 'pointer', fontSize: '14px' }}>
                  <input
                    type="checkbox"
                    checked={importForm[key]}
                    onChange={(e) => setImportForm({ ...importForm, [key]: e.target.checked })}
                  />
                  {label}
                </label>
              ))}
            </div>

            <div style={{ display: 'flex', gap: '8px', justifyContent: 'flex-end' }}>
              <button
                type="button"
                className="btn btn-ghost"
                onClick={() => { setShowImportForm(false); setImportResult(null); setImportError(null) }}
              >
                Отмена
              </button>
              <button
                type="submit"
                className="btn btn-primary"
                disabled={importing || !importFile}
              >
                {importing ? (
                  <span style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                    <span className="spinner" style={{ width: '14px', height: '14px', borderWidth: '2px' }} />
                    Импортируется…
                  </span>
                ) : 'Импортировать'}
              </button>
            </div>
          </form>
        </div>
      )}

      {/* Create form */}
      {showForm && (
        <div className="card">
          <h2 style={{ fontSize: '18px', fontWeight: 600, marginBottom: '20px' }}>
            Новое исследование
          </h2>
          <form onSubmit={handleCreate} style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(240px, 1fr))', gap: '16px' }}>
              <div>
                <label className="label">Название *</label>
                <input className="input" required value={form.name}
                  onChange={(e) => setForm({ ...form, name: e.target.value })} />
              </div>
              <div>
                <label className="label">Тип эффекта</label>
                <input
                  className="input"
                  list="effect-types-create"
                  placeholder="Введите или выберите тип..."
                  value={form.effect_type}
                  onChange={(e) => setForm({ ...form, effect_type: e.target.value })}
                />
                <datalist id="effect-types-create">
                  {uniqueEffectTypes.map((t) => <option key={t} value={t} />)}
                </datalist>
              </div>
            </div>

            <div>
              <label className="label">Заданий на участника</label>
              <input className="input" type="number" min={1} max={100}
                value={form.max_tasks_per_participant}
                onChange={(e) => setForm({ ...form, max_tasks_per_participant: e.target.value })} />
            </div>

            <div>
              <label className="label">Текст инструкций (необязательно)</label>
              <textarea className="input" rows={4}
                placeholder="Оставьте пустым для стандартных инструкций"
                value={form.instructions_text}
                onChange={(e) => setForm({ ...form, instructions_text: e.target.value })}
                style={{ resize: 'vertical' }} />
              <details style={{ marginTop: '6px' }}>
                <summary style={{ fontSize: '12px', color: 'var(--color-text-muted)', cursor: 'pointer', userSelect: 'none' }}>
                  Инструкция по умолчанию ↓
                </summary>
                <pre style={{
                  marginTop: '8px', padding: '12px', fontSize: '12px', lineHeight: 1.6,
                  background: 'var(--color-surface-2)', border: '1px solid var(--color-border)',
                  borderRadius: 'var(--radius-sm)', whiteSpace: 'pre-wrap', color: 'var(--color-text-muted)',
                }}>
                  {DEFAULT_INSTRUCTIONS}
                </pre>
              </details>
            </div>

            <div style={{ display: 'flex', gap: '24px', flexWrap: 'wrap' }}>
              {[
                { key: 'tie_option_enabled', label: 'Опция "Равны"' },
                { key: 'reasons_enabled', label: 'Причины выбора' },
                { key: 'confidence_enabled', label: 'Уверенность' },
              ].map(({ key, label }) => (
                <label key={key} style={{ display: 'flex', alignItems: 'center', gap: '8px', cursor: 'pointer', fontSize: '14px' }}>
                  <input type="checkbox" checked={form[key]}
                    onChange={(e) => setForm({ ...form, [key]: e.target.checked })} />
                  {label}
                </label>
              ))}
            </div>

            <div style={{ display: 'flex', gap: '8px', justifyContent: 'flex-end' }}>
              <button type="button" className="btn btn-ghost" onClick={() => setShowForm(false)}>
                Отмена
              </button>
              <button type="submit" className="btn btn-primary" disabled={creating}>
                {creating ? 'Создание…' : 'Создать'}
              </button>
            </div>
          </form>
        </div>
      )}

      {/* Studies list */}
      {loading ? (
        <div style={{ display: 'flex', justifyContent: 'center', padding: '48px' }}>
          <div className="spinner" />
        </div>
      ) : studies.length === 0 ? (
        <div style={{ textAlign: 'center', padding: '48px', color: 'var(--color-text-muted)' }}>
          Нет исследований. Создайте первое.
        </div>
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
          {studies.map((study) => {
            const sc = STATUS_COLORS[study.status] || STATUS_COLORS.draft
            const transitions = STATUS_TRANSITIONS[study.status] || []
            return (
              <div key={study.id} className="card" style={{ padding: '20px' }}>
                <div style={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', gap: '16px' }}>
                  <div style={{ flex: 1 }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '10px', marginBottom: '8px' }}>
                      <h3 style={{ fontSize: '16px', fontWeight: 600 }}>{study.name}</h3>
                      <span style={{
                        padding: '2px 10px', borderRadius: '99px', fontSize: '12px', fontWeight: 600,
                        background: sc.bg, color: sc.text,
                      }}>
                        {STATUS_LABELS[study.status]}
                      </span>
                      <span style={{ fontSize: '12px', color: 'var(--color-text-muted)', padding: '2px 8px',
                        border: '1px solid var(--color-border)', borderRadius: '4px' }}>
                        {study.effect_type}
                      </span>
                    </div>
                    <div style={{ display: 'flex', gap: '20px', fontSize: '13px', color: 'var(--color-text-muted)' }}>
                      <span>Заданий: {study.max_tasks_per_participant}</span>
                      <span>Участников: {study.participant_count ?? '—'}</span>
                      <span>Ответов: {study.response_count ?? '—'}</span>
                      <span style={{ fontFamily: 'monospace', fontSize: '11px' }}>{study.id}</span>
                    </div>
                  </div>

                  <div style={{ display: 'flex', gap: '8px', flexWrap: 'wrap', alignItems: 'center' }}>
                    <button className="btn btn-ghost" style={{ fontSize: '12px', padding: '6px 10px' }}
                      onClick={() => copyLink(study.id)} title="Скопировать ссылку для участников">
                      🔗 Ссылка
                    </button>
                    <button
                      className="btn btn-ghost"
                      style={{ fontSize: '12px', padding: '6px 10px' }}
                      onClick={() => {
                        setEditStudy(study)
                        setEditForm({
                          name: study.name,
                          effect_type: study.effect_type,
                          max_tasks_per_participant: study.max_tasks_per_participant,
                          instructions_text: study.instructions_text ?? '',
                          tie_option_enabled: study.tie_option_enabled,
                          reasons_enabled: study.reasons_enabled,
                          confidence_enabled: study.confidence_enabled,
                        })
                      }}
                    >
                      ✎ Редактировать
                    </button>
                    {transitions.map((s) => (
                      <button key={s} className="btn btn-ghost"
                        style={{ fontSize: '12px', padding: '6px 10px' }}
                        onClick={() => handleStatusChange(study.id, s)}>
                        → {STATUS_LABELS[s]}
                      </button>
                    ))}
                    {(study.status === 'draft' || study.status === 'archived') && (
                      <button
                        className="btn btn-ghost"
                        style={{ fontSize: '12px', padding: '6px 10px', color: '#ff4d6d' }}
                        onClick={() => handleDelete(study)}
                      >
                        🗑 Удалить
                      </button>
                    )}
                  </div>
                </div>
              </div>
            )
          })}
        </div>
      )}

      {/* Edit modal */}
      {editStudy && (
        <div style={{
          position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.6)',
          display: 'flex', alignItems: 'center', justifyContent: 'center',
          zIndex: 1000, padding: '16px',
        }}>
          <div className="card" style={{ width: '100%', maxWidth: '560px', padding: '28px' }}>
            <h2 style={{ fontSize: '18px', fontWeight: 600, marginBottom: '20px' }}>
              Редактировать исследование
            </h2>
            <form onSubmit={handleEditSave} style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
              <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(240px, 1fr))', gap: '16px' }}>
                <div>
                  <label className="label">Название *</label>
                  <input className="input" required value={editForm.name}
                    onChange={(e) => setEditForm({ ...editForm, name: e.target.value })} />
                </div>
                <div>
                  <label className="label">Тип эффекта</label>
                  <input
                    className="input"
                    list="effect-types-edit"
                    placeholder="Введите или выберите тип..."
                    value={editForm.effect_type}
                    onChange={(e) => setEditForm({ ...editForm, effect_type: e.target.value })}
                  />
                  <datalist id="effect-types-edit">
                    {uniqueEffectTypes.map((t) => <option key={t} value={t} />)}
                  </datalist>
                </div>
              </div>

              <div>
                <label className="label">Заданий на участника</label>
                <input className="input" type="number" min={1} max={100}
                  value={editForm.max_tasks_per_participant}
                  onChange={(e) => setEditForm({ ...editForm, max_tasks_per_participant: e.target.value })} />
              </div>

              <div>
                <label className="label">Текст инструкций</label>
                <textarea className="input" rows={4}
                  placeholder="Оставьте пустым для стандартных инструкций"
                  value={editForm.instructions_text}
                  onChange={(e) => setEditForm({ ...editForm, instructions_text: e.target.value })}
                  style={{ resize: 'vertical' }} />
                <details style={{ marginTop: '6px' }}>
                  <summary style={{ fontSize: '12px', color: 'var(--color-text-muted)', cursor: 'pointer', userSelect: 'none' }}>
                    Инструкция по умолчанию ↓
                  </summary>
                  <pre style={{
                    marginTop: '8px', padding: '12px', fontSize: '12px', lineHeight: 1.6,
                    background: 'var(--color-surface-2)', border: '1px solid var(--color-border)',
                    borderRadius: 'var(--radius-sm)', whiteSpace: 'pre-wrap', color: 'var(--color-text-muted)',
                  }}>
                    {DEFAULT_INSTRUCTIONS}
                  </pre>
                </details>
              </div>

              <div style={{ display: 'flex', gap: '24px', flexWrap: 'wrap' }}>
                {[
                  { key: 'tie_option_enabled', label: 'Опция "Равны"' },
                  { key: 'reasons_enabled', label: 'Причины выбора' },
                  { key: 'confidence_enabled', label: 'Уверенность' },
                ].map(({ key, label }) => (
                  <label key={key} style={{ display: 'flex', alignItems: 'center', gap: '8px', cursor: 'pointer', fontSize: '14px' }}>
                    <input type="checkbox" checked={editForm[key]}
                      onChange={(e) => setEditForm({ ...editForm, [key]: e.target.checked })} />
                    {label}
                  </label>
                ))}
              </div>

              <div style={{ display: 'flex', gap: '8px', justifyContent: 'flex-end' }}>
                <button type="button" className="btn btn-ghost" onClick={() => setEditStudy(null)}>
                  Отмена
                </button>
                <button type="submit" className="btn btn-primary" disabled={saving}>
                  {saving ? 'Сохранение…' : 'Сохранить'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}

function ErrorBox({ message, onClose }) {
  return (
    <div style={{
      padding: '12px 16px', background: 'rgba(255,77,109,0.1)',
      border: '1px solid rgba(255,77,109,0.3)', borderRadius: 'var(--radius-sm)',
      color: '#ff6584', fontSize: '14px', display: 'flex', justifyContent: 'space-between',
    }}>
      <span>{message}</span>
      <button onClick={onClose} style={{ background: 'none', border: 'none', color: '#ff6584', cursor: 'pointer' }}>✕</button>
    </div>
  )
}

function ImportResultBox({ result, onClose }) {
  return (
    <div style={{
      padding: '16px', background: 'rgba(67,217,139,0.08)',
      border: '1px solid rgba(67,217,139,0.3)', borderRadius: 'var(--radius-sm)',
      fontSize: '14px', marginBottom: '16px',
    }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
        <div style={{ fontWeight: 600, color: '#43d98b', marginBottom: '10px' }}>
          Импорт завершён успешно
        </div>
        <button onClick={onClose} style={{ background: 'none', border: 'none', color: 'var(--color-text-muted)', cursor: 'pointer' }}>✕</button>
      </div>
      <div style={{ display: 'flex', gap: '24px', flexWrap: 'wrap' }}>
        {[
          { label: 'Групп создано', value: result.groups_created },
          { label: 'Пар создано', value: result.pairs_created },
          { label: 'Видео загружено', value: result.videos_uploaded },
        ].map(({ label, value }) => (
          <div key={label} style={{ textAlign: 'center' }}>
            <div style={{ fontSize: '24px', fontWeight: 700, color: '#43d98b' }}>{value ?? '—'}</div>
            <div style={{ fontSize: '12px', color: 'var(--color-text-muted)' }}>{label}</div>
          </div>
        ))}
      </div>
      {result.errors && result.errors.length > 0 && (
        <div style={{ marginTop: '12px' }}>
          <div style={{ fontSize: '13px', fontWeight: 600, color: '#f0b429', marginBottom: '6px' }}>
            Предупреждения ({result.errors.length}):
          </div>
          <ul style={{ margin: 0, paddingLeft: '20px', fontSize: '13px', color: 'var(--color-text-muted)' }}>
            {result.errors.map((err, i) => (
              <li key={i}>{err}</li>
            ))}
          </ul>
        </div>
      )}
    </div>
  )
}
