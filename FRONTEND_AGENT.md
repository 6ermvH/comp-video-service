# Frontend Agent Guide

This document describes the frontend agent's responsibilities, code conventions, and rules for extending or refactoring the codebase.

---

## Scope

The agent owns **`frontend/src/`** and its supporting config files:

- `frontend/src/` — all React source code
- `frontend/eslint.config.js`
- `frontend/scripts/` — build-time scripts (e.g. `validate-api-contract.mjs`)
- `frontend/package.json`, `frontend/vite.config.js`

The agent does **not touch** backend code, Docker, CI/CD, or database migrations.

---

## Project Structure

```
src/
  api/
    client.js              — single source of all HTTP calls
  components/
    AdminLayout.jsx        — admin shell (sidebar + mobile burger drawer)
    ChoicePanel.jsx        — A / B / "Can't decide" buttons
    ConfidenceRating.jsx
    ProgressBar.jsx
    ReasonsSelector.jsx
    StatsChart.jsx         — recharts wrapper
    SyncVideoPlayer.jsx    — synchronized dual-video player
  context/
    SessionContext.jsx     — participant session state (token, tasks, meta)
    ToastContext.jsx       — global toast notifications
  hooks/
    useApiCall.js          — API call wrapper with automatic 429/500 toasts
    useWindowWidth.js      — reactive window width for mobile layout
  pages/
    WelcomePage.jsx
    InstructionsPage.jsx
    PracticePage.jsx
    TaskPage.jsx
    BreakPage.jsx
    CompletionPage.jsx
    LoginPage.jsx
    AdminStudiesPage.jsx
    AdminPairsPage.jsx
    AdminAnalyticsPage.jsx
  App.jsx                  — routing
  index.css                — design tokens and global styles
  main.jsx
scripts/
  validate-api-contract.mjs  — validates client.js calls against OpenAPI spec
```

---

## Backend Contract

The source of truth is `backend/API_CONTRACT.md` (or `backend/docs/swagger.yaml` once generated).

**Rules:**

- All HTTP calls go through `src/api/client.js` via the `api.*` object only
- Paths passed to `request()` omit the `/api` prefix: `request('/admin/studies')` → `/api/admin/studies`
- Mutating admin requests (POST/PUT/PATCH/DELETE) automatically get the `X-CSRF-Token` header
- 401/403 on admin routes → automatic logout via `setUnauthorizedHandler`
- HTTP 409 in `submitResponse` → idempotent, treat as success (do not throw)
- HTTP 204 → returns `null`

**Task shape normalization** (done in `SessionContext.normalizeTask`):
```js
{
  presentation_id,   // not pair_presentation_id
  task_order,
  is_practice,
  is_attention_check,
  left_video_url,    // from raw.left.presigned_url
  right_video_url,   // from raw.right.presigned_url
}
```

---

## Adding Code

### New API endpoint

1. Add a method to the `api` object in `src/api/client.js`:
   ```js
   getMyData: (id) => request(`/admin/my-resource/${id}`),
   ```
2. If `backend/docs/swagger.yaml` exists, run `npm run validate:api` to confirm it matches the spec.

### New page

1. Create `src/pages/MyPage.jsx`
2. Add a route in `src/App.jsx`
3. Admin pages → nest inside `<Route element={<AdminLayout />}>`
4. Participant pages → add a `sessionToken` guard (redirect to `/` if missing)

### New component

- Place in `src/components/`
- Styles: inline `style={{}}` or classes from `index.css` (`.btn`, `.card`, `.input`, `.label`)
- Do not introduce CSS modules or styled-components — the project uses a single `index.css`

### New hook

- Place in `src/hooks/`
- Example: `useWindowWidth.js` returns `window.innerWidth` and re-renders on resize

### Toast notifications

```js
const { addToast } = useToast()
addToast('Message', 'error')                          // auto-dismiss
addToast('Message', 'warning', { sticky: true })      // stays until closed
addToast('Failed', 'error', { retryFn: handleSubmit }) // with retry button
```

### API error handling

Use `useApiCall` for automatic 429/500 handling:
```js
const apiCall = useApiCall()
const data = await apiCall(() => api.getStudies(), { onRetry: load })
```

For manual handling, check `.status` on the caught error:
```js
catch (err) {
  if (err.status === 429) { /* rate limited */ }
  if (err.status >= 500) { /* server error */ }
}
```

---

## Mobile Adaptation

- Breakpoint: `768px` (CSS variable `--bp-mobile`)
- For conditional rendering use the `useWindowWidth` hook:
  ```js
  const isMobile = useWindowWidth() <= 768
  ```
- CSS utilities: `.hide-mobile`, `.hide-desktop`
- Minimum touch target height: `44px`
- Admin tables must always be wrapped in `<div style={{ overflowX: 'auto' }}>`
- Two-column form grids: `gridTemplateColumns: 'repeat(auto-fit, minmax(240px, 1fr))'`
- Fixed max-widths on participant pages: use `min(Npx, 100%)` instead of a hard `Npx`

---

## Refactoring Rules

### Principles

1. **Don't refactor without a reason.** If the task is a bug fix, don't touch surrounding code.
2. **Read the file before editing.** Never write changes from memory.
3. **Minimal diff.** Prefer `Edit` with a targeted replacement over a full file rewrite.
4. **No premature abstractions.** Three similar blocks of code is better than a helper used once.
5. **Don't add JSDoc or comments** to code that wasn't changed.
6. **Don't add error handling for impossible cases.** Trust internal code and framework guarantees.

### When a full file rewrite (Write) is acceptable

- The file's logic is entirely outdated (not just cosmetic)
- Five or more unrelated changes are needed in one file
- The component's hook structure needs to change (new state, reordered hooks)

### Renaming / moving files

- `Grep` for all imports of the file before deleting it
- Check `App.jsx` for affected routes

### After every change

```bash
npm run lint    # must produce 0 warnings and 0 errors
npm run build   # must succeed
```

---

## Linter (ESLint 9 flat config)

Config: `frontend/eslint.config.js`

Active rules:
- `no-unused-vars` — warn; variables prefixed with `_` are ignored
- `no-console` — warn; `console.warn` and `console.error` are allowed
- `react-hooks/rules-of-hooks` — error (hooks must come before any conditional `return`)
- `react-refresh/only-export-components` — warn

Ignored paths: `dist/**`, `node_modules/**`, `scripts/**`

---

## API Contract Validation

```bash
cd frontend && npm run validate:api
```

The script reads `backend/docs/swagger.yaml` and verifies that every call in `api/client.js` has a matching route in the spec. Run it after:
- modifying `api/client.js`
- regenerating the swagger spec on the backend side

---

## Pre-completion Checklist

- [ ] `npm run lint` — clean
- [ ] `npm run build` — succeeds
- [ ] If `api/client.js` changed and `backend/docs/swagger.yaml` exists — `npm run validate:api`
- [ ] Mobile layout considered (`isMobile` / `min()` / `auto-fit`)
- [ ] No hardcoded colors — use CSS variables from `index.css`
- [ ] New tables wrapped in `overflowX: auto`
- [ ] No imports of deleted components or files
