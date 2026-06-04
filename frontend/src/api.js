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

export const api = {
  health: () => req('/health'),
  scenarios: () => req('/scenarios'),
  scenario: (id) => req('/scenarios/' + id),
  startSession: (scenarioId) =>
    req('/session', { method: 'POST', body: JSON.stringify({ scenario_id: scenarioId }) }),
  chat: (sessionId, message) =>
    req('/chat', { method: 'POST', body: JSON.stringify({ session_id: sessionId, message }) }),
  grade: (sessionId, answer) =>
    req('/grade', { method: 'POST', body: JSON.stringify({ session_id: sessionId, answer }) }),
}
