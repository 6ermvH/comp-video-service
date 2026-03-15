# Задачи для Агента 2 — Frontend

## Задача 1 — CRITICAL: Пустая страница при выборе исследования на /admin/pairs

**Файл:** `frontend/src/pages/AdminPairsPage.jsx`, строки 43 и 63

**Причина:** бэкенд возвращает `{"source_items": null}` когда пар нет.
Текущий код: `data?.source_items || data || []`
При `null || data` подставляется весь объект `{source_items: null}` — это не массив, `.map()` крашится, страница пустая.

**Фикс строка 43:**
```js
// было
.then((data) => setSourceItems(data?.source_items ?? []))
```

**Фикс строка 63** (в `handleImport`, то же самое):
```js
setSourceItems(data?.source_items ?? [])
```

---

## Задача 2 — HIGH: Добавить секцию управления группами на /admin/pairs

**Файл:** `frontend/src/pages/AdminPairsPage.jsx`

**Проблема:** чтобы импортировать CSV с парами, первая колонка должна содержать `group_id` (UUID). Сейчас нигде нет ни формы создания группы, ни отображения UUID существующих групп.

**Что нужно добавить** — новую секцию между "Выбор исследования" и "CSV-импорт":

### Состояние
```js
const [groups, setGroups] = useState([])
const [showGroupForm, setShowGroupForm] = useState(false)
const [groupForm, setGroupForm] = useState({ name: '', description: '', priority: 0, target_votes_per_pair: 10 })
const [creatingGroup, setCreatingGroup] = useState(false)
```

### Загрузка групп при смене исследования
Добавить в useEffect при смене `selectedStudy`:
```js
api.getGroups(selectedStudy)
  .then((data) => setGroups(data?.groups ?? []))
  .catch(() => setGroups([]))
```

### API клиент (`frontend/src/api/client.js`)
Добавить два метода:
```js
getGroups: (studyId) =>
  request(`/admin/studies/${studyId}/groups`),

createGroup: (studyId, body) =>
  request(`/admin/studies/${studyId}/groups`, { method: 'POST', body: JSON.stringify(body) }),
```

### UI секции групп

**Список групп** — таблица/карточки с колонками: Название, Приоритет, Цель ответов, UUID (моноширинный, с кнопкой "Копировать").

**Форма создания группы** — кнопка "+ Группа" раскрывает форму с полями:
- Название (required)
- Описание (optional)
- Приоритет (number, default 0)
- Цель ответов на пару (number, default 10)

После создания — обновить список групп.

**Подсказка под списком групп:**
```
Скопируйте UUID нужной группы в первую колонку CSV для импорта пар.
```

---

## Задача 3 — HIGH: Заменить input UUID на select в форме загрузки ассета

**Файл:** `frontend/src/pages/AdminPairsPage.jsx`, строки 172–174

**Фикс:** заменить `<input>` на `<select>` из `sourceItems`. Когда `sourceItems.length === 0` — показать текст вместо формы:

```jsx
{sourceItems.length === 0 ? (
  <p style={{ color: 'var(--color-text-muted)', fontSize: '14px' }}>
    Сначала импортируйте пары через CSV — затем здесь можно привязать видео к паре.
  </p>
) : (
  // вся форма handleUploadAsset
)}
```

Внутри формы заменить поле Source Item ID:
```jsx
// label: "Пара *" вместо "Source Item ID *"
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
```
