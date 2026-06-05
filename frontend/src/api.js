// Thin API client for the Go backend. All calls go through /api (proxied in dev).
const BASE = '/api'

async function req(path, opts = {}) {
  const res = await fetch(BASE + path, {
    headers: { 'Content-Type': 'application/json' },
    ...opts,
  })
  const data = await res.json().catch(() => ({}))
  if (!res.ok) throw new Error(data.error || `HTTP ${res.status}`)
  return data
}

async function reqForm(path, formData, opts = {}) {
  const res = await fetch(BASE + path, {
    method: opts.method || 'POST',
    body: formData,
  })
  const data = await res.json().catch(() => ({}))
  if (!res.ok) throw new Error(data.error || `HTTP ${res.status}`)
  return data
}

export const api = {
  health: () => req('/health'),
  scenarios: () => req('/scenarios'),
  scenario: (id) => req('/scenarios/' + id),
  teacherScenarios: () => req('/teacher/scenarios'),
  teacherScenario: (id) => req('/teacher/scenarios/' + id),
  teacherDocuments: () => req('/teacher/documents'),
  teacherDocument: (id) => req('/teacher/documents/' + id),
  uploadTeacherDocument: (formData) => reqForm('/teacher/documents', formData),
  generateScenarioFromDocument: (id, payload) =>
    req('/teacher/documents/' + id + '/generate-scenario', { method: 'POST', body: JSON.stringify(payload) }),
  generateTeacherScenario: (payload) =>
    req('/teacher/scenarios', { method: 'POST', body: JSON.stringify(payload) }),
  updateTeacherScenario: (id, payload) =>
    req('/teacher/scenarios/' + id, { method: 'PUT', body: JSON.stringify(payload) }),
  approveTeacherScenario: (id) =>
    req('/teacher/scenarios/' + id + '/approve', { method: 'PATCH', body: '{}' }),
  startSession: (scenarioId) =>
    req('/session', { method: 'POST', body: JSON.stringify({ scenario_id: scenarioId }) }),
  chat: (sessionId, message) =>
    req('/chat', { method: 'POST', body: JSON.stringify({ session_id: sessionId, message }) }),
  grade: (sessionId, answer) =>
    req('/grade', { method: 'POST', body: JSON.stringify({ session_id: sessionId, answer }) }),
  sandboxSubmit: (sessionId, code) =>
    req('/sandbox/submit', { method: 'POST', body: JSON.stringify({ session_id: sessionId, code }) }),
}
