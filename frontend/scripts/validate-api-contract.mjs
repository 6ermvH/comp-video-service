#!/usr/bin/env node
import { readFileSync } from 'fs'
import { resolve, dirname } from 'path'
import { fileURLToPath } from 'url'
import { parse } from 'yaml'

const __dirname = dirname(fileURLToPath(import.meta.url))
const root = resolve(__dirname, '../..')

// --- Load OpenAPI spec ---
const specPath = resolve(root, 'backend/docs/openapi.yaml')
let spec
try {
  spec = parse(readFileSync(specPath, 'utf8'))
} catch (_) {
  console.error(`Cannot read OpenAPI spec at ${specPath}`)
  console.error('Run: cd backend && swag init -g cmd/server/main.go -o docs --outputTypes yaml,json')
  process.exit(1)
}

// Build set of "METHOD /basepath/path" from spec
const basePath = (spec.basePath || '').replace(/\/$/, '')
const specEndpoints = new Set()
for (const [path, methods] of Object.entries(spec.paths || {})) {
  for (const method of Object.keys(methods)) {
    if (['get', 'post', 'put', 'patch', 'delete'].includes(method)) {
      const normalized = path.replace(/\{(\w+)\}/g, ':$1')
      specEndpoints.add(`${method.toUpperCase()} ${basePath}${normalized}`)
    }
  }
}

// --- Parse api/client.js ---
const clientPath = resolve(__dirname, '../src/api/client.js')
const clientSrc = readFileSync(clientPath, 'utf8')

// Extract paths from request() / requestMultipart() / download() calls.
// client.js passes paths like /admin/... or /participant/... to request(),
// which prepends BASE_URL (/api), giving final URL /api/admin/... etc.
const clientEndpoints = []
const lines = clientSrc.split('\n')

for (const line of lines) {
  // Only process lines that are actual API calls (not href assignments, comments, etc.)
  const isApiCall = line.includes('request(') || line.includes('download(')
  if (!isApiCall) continue

  // requestMultipart defaults to POST (its function signature has method = 'POST')
  const isMultipart = line.includes('requestMultipart(')

  // Match single-quoted paths: '/admin/...'
  const singleQ = line.match(/'(\/[^'\n]+)'/)
  // Match double-quoted paths: "/admin/..."
  const doubleQ = line.match(/"(\/[^"\n]+)"/)
  // Match template-literal paths: `/admin/...` (use RegExp to avoid backtick-in-regex issues)
  const tmplLit = line.match(new RegExp('`(/[^`\\n]+)`'))

  const m = singleQ || doubleQ || tmplLit
  if (!m) continue

  const rawPath = m[1]
  // Only process paths that look like API routes
  if (!rawPath.startsWith('/admin/') && !rawPath.startsWith('/participant/')) continue

  const normalizedPath = rawPath
    .replace(/\$\{[^}]+\}/g, ':param')   // ${studyId} -> :param
    .split('?')[0]                         // strip static query strings
    .replace(/[^/]:param.*$/, (s) => s[0]) // strip query-string-style ${} not preceded by /

  let method = isMultipart ? 'POST' : 'GET'
  if (line.includes("method: 'POST'")   || line.includes('method: "POST"'))   method = 'POST'
  if (line.includes("method: 'PATCH'")  || line.includes('method: "PATCH"'))  method = 'PATCH'
  if (line.includes("method: 'DELETE'") || line.includes('method: "DELETE"')) method = 'DELETE'
  if (line.includes("method: 'PUT'")    || line.includes('method: "PUT"'))    method = 'PUT'

  // Prepend /api to match OpenAPI spec paths
  clientEndpoints.push(`${method} /api${normalizedPath}`)
}

// Deduplicate
const uniqueClientEndpoints = [...new Set(clientEndpoints)]

// --- Compare ---
const errors = []
for (const endpoint of uniqueClientEndpoints) {
  // Normalize dynamic segments for comparison (:param vs :id vs :studyId etc.)
  const normalizedClient = endpoint.replace(/:[^/]+/g, ':param')
  const found = [...specEndpoints].some((se) =>
    se.replace(/:[^/]+/g, ':param') === normalizedClient
  )
  if (!found) {
    errors.push(`  x ${endpoint} — not found in OpenAPI spec`)
  }
}

// Report
console.log(`OpenAPI spec endpoints: ${specEndpoints.size}`)
console.log(`Frontend client endpoints: ${uniqueClientEndpoints.length}`)
console.log()

if (errors.length > 0) {
  console.error('API contract violations:')
  errors.forEach((e) => console.error(e))
  console.error()
  console.error(`${errors.length} endpoint(s) in api/client.js have no matching route in the OpenAPI spec.`)
  console.error('Either add the route to the backend and regenerate the spec, or fix the URL in client.js.')
  process.exit(1)
} else {
  console.log('All frontend API calls match the OpenAPI spec.')
}
