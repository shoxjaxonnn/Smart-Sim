<script setup>
import { computed, onMounted, ref } from 'vue'
import { api } from '../api'
import ChatPanel from './ChatPanel.vue'

const backendOnline = ref(false)
const currentSubject = ref('')
const currentDocument = ref('')
const scenarios = ref([])
const activeScenario = ref(null)
const sessionId = ref(null)
const error = ref('')
const loading = ref(false)

const recommendedScenarios = computed(() => {
  const subject = currentSubject.value.trim().toLowerCase()
  const source = subject
    ? scenarios.value.filter((s) => sameSubject(s.subject, currentSubject.value))
    : scenarios.value
  return source.slice(0, 3)
})

const otherScenarios = computed(() => {
  const featured = new Set(recommendedScenarios.value.map((s) => s.id))
  return scenarios.value.filter((s) => !featured.has(s.id))
})

async function loadScenarios() {
  loading.value = true
  error.value = ''
  try {
    const health = await api.health()
    backendOnline.value = true
    currentSubject.value = health.current_subject || ''
    currentDocument.value = health.current_document || ''
    scenarios.value = await api.scenarios()
    if (scenarios.value.length) {
      await startScenario(recommendedScenarios.value[0] || scenarios.value[0])
    } else {
      activeScenario.value = null
      sessionId.value = null
    }
  } catch (e) {
    backendOnline.value = false
    const message = e?.message || ''
    error.value = message.includes('404')
      ? 'Teacher data hali tayyor emas. Backend scenario route-larini tekshiring.'
      : 'Backend ulanmagan. `cd backend && go run .` ishga tushiring.'
  } finally {
    loading.value = false
  }
}

async function startScenario(brief) {
  if (!brief) return
  error.value = ''
  try {
    const full = await api.scenario(brief.id)
    const s = await api.startSession(brief.id)
    activeScenario.value = full
    sessionId.value = s.session_id
  } catch (e) {
    error.value = e.message
  }
}

async function switchScenario(brief) {
  await startScenario(brief)
}

onMounted(loadScenarios)

function sameSubject(a, b) {
  return String(a || '').trim().toLowerCase() === String(b || '').trim().toLowerCase()
}
</script>

<template>
  <section class="student-shell">
    <div class="student-head">
      <div>
        <div class="kicker">Student panel</div>
        <h2>Chat-first simulation</h2>
      </div>
      <div class="right">
        <span class="pill" :class="backendOnline ? 'pill-good' : 'pill-bad'">
          {{ backendOnline ? 'Backend online' : 'Backend offline' }}
        </span>
        <button class="btn-ghost" :disabled="loading" @click="loadScenarios">
          {{ loading ? 'Loading…' : 'Reload' }}
        </button>
      </div>
    </div>

    <div v-if="currentSubject" class="subject-banner">
      <div>
        <div class="subject-kicker">Current subject</div>
        <div class="subject-name">{{ currentSubject }}</div>
      </div>
      <div class="subject-source">
        <span class="subject-label">From teacher context</span>
        <span class="subject-doc">{{ currentDocument }}</span>
      </div>
    </div>

    <div v-if="recommendedScenarios.length" class="recommendations">
      <div class="section-row">
        <div>
          <div class="kicker">Recommended</div>
          <h2>Top 3 exact-match scenarios</h2>
        </div>
        <span class="pill">{{ recommendedScenarios.length }} shown</span>
      </div>
      <div class="scenario-grid">
        <button
          v-for="s in recommendedScenarios"
          :key="s.id"
          class="scenario-card"
          :class="{ active: activeScenario?.id === s.id }"
          @click="switchScenario(s)"
        >
          <div class="scenario-top">
            <span class="scenario-title">{{ s.title }}</span>
            <span class="scenario-status">{{ s.subject }}</span>
          </div>
          <div class="scenario-sub">{{ s.status || 'approved' }}</div>
          <div class="scenario-id">{{ s.id }}</div>
        </button>
      </div>
      <details v-if="otherScenarios.length" class="all-scenarios">
        <summary>See all</summary>
        <div class="scenario-grid all-grid">
          <button
            v-for="s in otherScenarios"
            :key="s.id"
            class="scenario-card muted-card"
            :class="{ active: activeScenario?.id === s.id }"
            @click="switchScenario(s)"
          >
            <div class="scenario-top">
              <span class="scenario-title">{{ s.title }}</span>
              <span class="scenario-status">{{ s.subject }}</span>
            </div>
            <div class="scenario-sub">{{ s.status || 'approved' }}</div>
            <div class="scenario-id">{{ s.id }}</div>
          </button>
        </div>
      </details>
    </div>

    <p v-if="error" class="error">{{ error }}</p>

    <div v-if="activeScenario && sessionId" class="chat-wrap">
      <ChatPanel
        :key="activeScenario.id + ':' + sessionId"
        :scenario="activeScenario"
        :session-id="sessionId"
        :available-scenarios="scenarios"
        @switch-scenario="switchScenario"
      />
    </div>

    <div v-else class="empty-state">
      <h3>Approved scenario yo'q</h3>
      <p>Teacher panelda yangi issue yarating va approve qiling. Keyin student chat shu yerda ochiladi.</p>
    </div>
  </section>
</template>

<style scoped>
.student-shell {
  height: 100%;
  min-height: 0;
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.student-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
}
.kicker {
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: .8px;
  color: var(--accent);
  font-weight: 700;
}
h2 {
  margin: 6px 0 0;
  font-size: 22px;
}
.right {
  display: flex;
  gap: 10px;
  align-items: center;
}
.subject-banner {
  display: flex;
  justify-content: space-between;
  gap: 14px;
  align-items: center;
  background: linear-gradient(160deg, var(--panel-2), var(--panel));
  border: 1px solid var(--border);
  border-radius: var(--radius);
  padding: 14px 16px;
  box-shadow: var(--shadow);
}
.subject-kicker,
.kicker {
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: .8px;
  color: var(--accent);
  font-weight: 700;
}
.subject-name {
  margin-top: 4px;
  font-size: 18px;
  font-weight: 800;
  color: var(--text);
}
.subject-source {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 4px;
  color: var(--text-dim);
  text-align: right;
}
.subject-label {
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: .6px;
}
.subject-doc {
  font-size: 13px;
}
.recommendations {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.section-row {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  align-items: center;
}
h2 {
  margin: 4px 0 0;
  font-size: 20px;
}
.scenario-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
}
.scenario-card {
  text-align: left;
  background: linear-gradient(160deg, var(--panel-2), var(--panel));
  border: 1px solid var(--border);
  color: var(--text);
  padding: 14px;
  border-radius: 12px;
  box-shadow: var(--shadow);
}
.scenario-card.active {
  border-color: var(--accent);
  box-shadow: 0 0 0 1px var(--focus-ring) inset, var(--shadow);
}
.muted-card {
  opacity: .88;
}
.scenario-top {
  display: flex;
  justify-content: space-between;
  gap: 10px;
  align-items: start;
}
.scenario-title {
  font-weight: 800;
  font-size: 14px;
  line-height: 1.35;
}
.scenario-status {
  flex: none;
  font-size: 11px;
  padding: 4px 8px;
  border-radius: 999px;
  border: 1px solid var(--border);
  color: var(--text-dim);
}
.scenario-sub {
  margin-top: 6px;
  font-size: 12px;
  color: var(--text-dim);
}
.scenario-id {
  margin-top: 4px;
  font-size: 11px;
  color: var(--text-dim);
  opacity: .82;
  word-break: break-word;
}
.all-scenarios {
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 12px 14px;
}
.all-scenarios summary {
  cursor: pointer;
  color: var(--text);
  font-weight: 700;
}
.pill {
  font-size: 12px;
  padding: 6px 12px;
  border-radius: 999px;
  background: var(--panel-2);
  border: 1px solid var(--border);
  color: var(--text-dim);
}
.pill-good { color: var(--good); border-color: var(--success-border); background: var(--success-bg); }
.pill-bad { color: var(--bad); border-color: var(--danger-border); background: var(--danger-bg); }
.error {
  background: var(--danger-bg);
  border: 1px solid var(--danger-border);
  color: var(--danger-text);
  padding: 12px 16px;
  border-radius: 10px;
  margin: 0;
}
.chat-wrap {
  flex: 1;
  min-height: 0;
}
.empty-state {
  flex: 1;
  display: grid;
  place-items: center;
  text-align: center;
  background: linear-gradient(160deg, var(--panel-2), var(--panel));
  border: 1px solid var(--border);
  border-radius: var(--radius);
  padding: 32px;
  color: var(--text-dim);
  box-shadow: var(--shadow);
}
.empty-state h3 {
  margin: 0 0 8px;
  color: var(--text);
}

@media (max-width: 760px) {
  .student-head {
    flex-direction: column;
    align-items: stretch;
  }
  .right {
    justify-content: space-between;
  }
  .subject-banner,
  .section-row {
    flex-direction: column;
    align-items: stretch;
  }
  .subject-source {
    align-items: flex-start;
    text-align: left;
  }
  .scenario-grid {
    grid-template-columns: 1fr;
  }
}
</style>
