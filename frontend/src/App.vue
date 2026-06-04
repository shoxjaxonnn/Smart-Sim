<script setup>
import { ref, onMounted } from 'vue'
import { api } from './api'
import ScenarioIntro from './components/ScenarioIntro.vue'
import ChatPanel from './components/ChatPanel.vue'
import GradePanel from './components/GradePanel.vue'

const view = ref('intro') // intro | chat | grade
const scenarios = ref([])
const scenario = ref(null)
const sessionId = ref(null)
const grade = ref(null)
const offline = ref(false)
const error = ref('')

onMounted(async () => {
  try {
    await api.health()
    scenarios.value = await api.scenarios()
  } catch (e) {
    offline.value = true
    error.value = 'Backend ulanmadi. `cd backend && go run .` ishga tushiring.'
  }
})

async function startScenario(brief) {
  error.value = ''
  try {
    scenario.value = await api.scenario(brief.id)
    const s = await api.startSession(brief.id)
    sessionId.value = s.session_id
    view.value = 'chat'
  } catch (e) {
    error.value = e.message
  }
}

function onFinished(g) {
  grade.value = g
  view.value = 'grade'
}

function restart() {
  scenario.value = null
  sessionId.value = null
  grade.value = null
  view.value = 'intro'
}
</script>

<template>
  <div class="shell">
    <header class="topbar">
      <div class="brand">
        <span class="logo">◆</span>
        <div>
          <div class="brand-name">Smart Sim</div>
          <div class="brand-sub">AI Simulation Platform</div>
        </div>
      </div>
      <div class="top-right">
        <span v-if="scenario" class="pill">{{ scenario.subject }}</span>
        <span class="pill" :class="offline ? 'pill-bad' : 'pill-good'">
          {{ offline ? 'Backend offline' : 'Backend online' }}
        </span>
      </div>
    </header>

    <main class="stage">
      <p v-if="error" class="error">{{ error }}</p>

      <Transition name="fade" mode="out-in">
        <ScenarioIntro
          v-if="view === 'intro'"
          :scenarios="scenarios"
          :offline="offline"
          @start="startScenario"
          key="intro"
        />
        <ChatPanel
          v-else-if="view === 'chat'"
          :scenario="scenario"
          :session-id="sessionId"
          @finished="onFinished"
          key="chat"
        />
        <GradePanel
          v-else
          :scenario="scenario"
          :grade="grade"
          @restart="restart"
          key="grade"
        />
      </Transition>
    </main>
  </div>
</template>

<style scoped>
.shell {
  height: 100%;
  display: flex;
  flex-direction: column;
  max-width: 1180px;
  margin: 0 auto;
  padding: 0 24px;
}

.topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 20px 4px;
}
.brand { display: flex; align-items: center; gap: 14px; }
.logo {
  width: 44px; height: 44px;
  display: grid; place-items: center;
  font-size: 22px;
  border-radius: 12px;
  background: linear-gradient(135deg, var(--accent), var(--accent-2));
  box-shadow: 0 6px 20px rgba(139, 92, 246, 0.4);
}
.brand-name { font-size: 19px; font-weight: 700; letter-spacing: .2px; }
.brand-sub { font-size: 12px; color: var(--text-dim); }

.top-right { display: flex; gap: 10px; align-items: center; }
.pill {
  font-size: 12px;
  padding: 6px 12px;
  border-radius: 999px;
  background: var(--panel-2);
  border: 1px solid var(--border);
  color: var(--text-dim);
}
.pill-good { color: var(--good); border-color: rgba(52, 211, 153, .4); }
.pill-bad { color: var(--bad); border-color: rgba(248, 113, 113, .4); }

.stage {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  padding-bottom: 24px;
}
.error {
  background: rgba(248, 113, 113, .12);
  border: 1px solid rgba(248, 113, 113, .4);
  color: #fecaca;
  padding: 12px 16px;
  border-radius: 10px;
  margin: 0 0 14px;
  font-size: 14px;
}

@media (max-width: 760px) {
  .shell { padding: 0 14px; }
  .brand-sub { display: none; }
}
</style>
