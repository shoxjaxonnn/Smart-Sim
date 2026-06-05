<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { api } from '../api'

const props = defineProps({
  scenario: { type: Object, required: true },
  sessionId: { type: String, required: true },
  availableScenarios: { type: Array, default: () => [] },
})
const emit = defineEmits(['finished', 'switch-scenario'])

const messages = ref([])
const input = ref('')
const sending = ref(false)
const grading = ref(false)
const confirming = ref(false)
const sandboxing = ref(false)
const errorMsg = ref('')
const sandboxMsg = ref('')
const studentCode = ref('')
const sandboxResult = ref(null)
const atBottom = ref(true)
const codeOpen = ref(false)

const streamEl = ref(null)
const taEl = ref(null)
let nextId = 0
let onKeydownGlobal = () => {}

const starters = [
  'Server error logida nima yozilgan?',
  'Qaysi jadval hujumga uchragan?',
  "Bu qanday hujum turi deb o'ylaysan?",
]

const userTurns = computed(() => messages.value.filter((m) => m.role === 'user').length)
const showStarters = computed(() => userTurns.value === 0 && !sending.value)
const codeUnlocked = computed(() => {
  const round = Number(props.scenario?.code_challenge_after_round || 0)
  if (!round) return Boolean(props.scenario?.code_challenge?.tests)
  return userTurns.value >= round
})

watch(
  () => props.scenario?.id,
  () => {
    messages.value = []
    nextId = 0
    errorMsg.value = ''
    sandboxMsg.value = ''
    sandboxResult.value = null
    confirming.value = false
    studentCode.value = ''
    codeOpen.value = false
  },
  { immediate: true }
)

watch(
  codeUnlocked,
  (open) => {
    if (open && props.scenario?.code_challenge?.buggy_code && !studentCode.value.trim()) {
      studentCode.value = props.scenario.code_challenge.buggy_code
    }
  },
  { immediate: true }
)

onMounted(() => {
  pushMsg(
    'assistant',
    `"${props.scenario.title}" simulyatsiyasi boshlandi. Vaziyatni tahlil qil, savol ber, keyin code challenge ochiladi.`
  )
  autoGrow()
  onKeydownGlobal = (event) => {
    if (event.key === 'Escape') {
      codeOpen.value = false
    }
  }
  window.addEventListener('keydown', onKeydownGlobal)
})

onBeforeUnmount(() => {
  window.removeEventListener('keydown', onKeydownGlobal)
})

function now() {
  return new Date().toLocaleTimeString('uz', { hour: '2-digit', minute: '2-digit' })
}

function pushMsg(role, content) {
  messages.value.push({ id: nextId++, role, content, time: now() })
}

function fmt(text) {
  const esc = text
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
  return esc
    .replace(/`([^`]+)`/g, '<code>$1</code>')
    .replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>')
}

async function scrollDown() {
  await nextTick()
  if (streamEl.value) streamEl.value.scrollTop = streamEl.value.scrollHeight
}

function onScroll() {
  const el = streamEl.value
  if (!el) return
  atBottom.value = el.scrollHeight - el.scrollTop - el.clientHeight < 60
}

function autoGrow() {
  const el = taEl.value
  if (!el) return
  el.style.height = 'auto'
  el.style.height = Math.min(el.scrollHeight, 140) + 'px'
}

async function send(preset) {
  const text = (preset ?? input.value).trim()
  if (!text || sending.value) return
  errorMsg.value = ''
  pushMsg('user', text)
  input.value = ''
  await nextTick()
  autoGrow()
  sending.value = true
  scrollDown()
  try {
    const r = await api.chat(props.sessionId, text)
    pushMsg('assistant', r.reply)
  } catch (e) {
    errorMsg.value = e.message
    pushMsg('assistant', 'Xatolik: ' + e.message)
  } finally {
    sending.value = false
    scrollDown()
  }
}

function finishClick() {
  if (grading.value) return
  if (!confirming.value) {
    confirming.value = true
    return
  }
  doGrade()
}

async function doGrade() {
  grading.value = true
  errorMsg.value = ''
  try {
    const g = await api.grade(props.sessionId, '')
    emit('finished', g)
  } catch (e) {
    errorMsg.value = e.message
    confirming.value = false
  } finally {
    grading.value = false
  }
}

function resetCode() {
  studentCode.value = props.scenario?.code_challenge?.buggy_code || ''
  sandboxMsg.value = ''
  sandboxResult.value = null
}

async function runSandbox(mode = 'run') {
  if (!studentCode.value.trim() || sandboxing.value) return
  sandboxing.value = true
  sandboxMsg.value = ''
  sandboxResult.value = null
  try {
    const res = await api.sandboxSubmit(props.sessionId, studentCode.value)
    sandboxResult.value = res
    if (res.passed) {
      sandboxMsg.value = mode === 'submit'
        ? 'Topshirildi. Testlar o‘tdi.'
        : "Testlar o'tdi. Yechim ishladi."
    } else {
      sandboxMsg.value = res.timed_out
        ? 'Kod timeout berdi.'
        : 'Testlar yiqildi. Xatoni tekshir.'
    }
  } catch (e) {
    sandboxMsg.value = e.message
    sandboxResult.value = { passed: false, stderr: e.message }
  } finally {
    sandboxing.value = false
  }
}

function onKeydown(e) {
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    send()
  }
}

function toggleCodePane() {
  codeOpen.value = !codeOpen.value
}
</script>

<template>
  <div class="sim-shell" :class="{ 'code-open': codeOpen }">
    <Transition name="fade">
      <button v-if="codeOpen" class="drawer-backdrop" aria-label="Close code pane" @click="toggleCodePane"></button>
    </Transition>

    <div class="split">
      <section class="chat-pane">
        <div class="chat-head">
          <div>
            <div class="kicker">AI Tutor</div>
            <h3>{{ scenario.title }}</h3>
          </div>
          <div class="chat-tools">
            <select
              v-if="availableScenarios.length > 1"
              :value="scenario.id"
              @change="emit('switch-scenario', availableScenarios.find((s) => s.id === $event.target.value))"
            >
              <option v-for="s in availableScenarios" :key="s.id" :value="s.id">{{ s.title }}</option>
            </select>
            <span class="meta-pill">{{ userTurns }} turn</span>
            <button class="btn-ghost code-toggle" :aria-expanded="codeOpen" @click="toggleCodePane">
              {{ codeOpen ? 'Hide code' : 'Code pane' }}
            </button>
          </div>
        </div>

        <div class="stream" ref="streamEl" @scroll="onScroll">
          <TransitionGroup name="msg">
            <div v-for="m in messages" :key="m.id" class="msg" :class="m.role">
              <div class="avatar">{{ m.role === 'user' ? 'Sen' : 'AI' }}</div>
              <div class="msg-body">
                <div class="bubble" v-html="fmt(m.content)"></div>
                <span class="time">{{ m.time }}</span>
              </div>
            </div>
          </TransitionGroup>

          <div v-if="sending" class="msg assistant">
            <div class="avatar">AI</div>
            <div class="msg-body">
              <div class="bubble typing"><span></span><span></span><span></span></div>
            </div>
          </div>
        </div>

        <Transition name="fade">
          <button v-if="!atBottom" class="scroll-fab" @click="scrollDown">↓</button>
        </Transition>

        <Transition name="fade">
          <div v-if="showStarters" class="starters">
            <button v-for="s in starters" :key="s" class="chip" @click="send(s)">{{ s }}</button>
          </div>
        </Transition>

        <p v-if="errorMsg" class="chat-error">{{ errorMsg }}</p>

        <div class="composer">
          <textarea
            ref="taEl"
            v-model="input"
            rows="1"
            placeholder="Javob yoz..."
            @keydown="onKeydown"
            @input="autoGrow"
          ></textarea>
          <button class="btn-primary send-btn" :disabled="sending || !input.trim()" @click="send()">
            <span>Yuborish</span>
          </button>
        </div>

        <div class="finish-zone">
          <div class="finish-btns">
            <button v-if="confirming" class="btn-ghost" :disabled="grading" @click="confirming = false">Bekor</button>
            <button class="btn-primary finish-btn" :disabled="grading || userTurns === 0" @click="finishClick">
              {{ grading ? 'Baholanmoqda…' : confirming ? 'Ha, bahola' : 'Yakunlash va baho olish' }}
            </button>
          </div>
        </div>
      </section>

      <aside class="terminal-pane" :class="{ open: codeOpen }" aria-label="Code pane">
        <div class="terminal-head">
          <div>
            <div class="kicker">Terminal</div>
            <div class="terminal-title">{{ scenario.code_language || 'python' }} challenge</div>
          </div>
          <div class="terminal-tools">
            <button class="btn-ghost small" :disabled="!codeUnlocked" @click="resetCode">Reset</button>
            <button class="btn-ghost small" @click="toggleCodePane">Close</button>
          </div>
        </div>

        <div v-if="scenario.code_challenge && (scenario.code_challenge.buggy_code || scenario.code_challenge.tests)" class="code-card">
          <div v-if="!codeUnlocked" class="code-locked">
            Code challenge {{ scenario.code_challenge_after_round }}-roundda ochiladi.
          </div>
          <template v-else>
            <div class="editor-actions">
              <span class="language-chip">Python</span>
              <button class="btn-primary" :disabled="sandboxing || !studentCode.trim()" @click="runSandbox('run')">
                {{ sandboxing ? 'Running…' : 'Yurish' }}
              </button>
              <button class="btn-primary" :disabled="sandboxing || !studentCode.trim()" @click="runSandbox('submit')">
                Yuborish
              </button>
            </div>

            <textarea
              v-model="studentCode"
              class="code-input"
              rows="13"
              :placeholder="scenario.code_language || 'python'"
            ></textarea>

            <div class="test-tabs">
              <span class="active">Testlar</span>
              <span>Natijalar</span>
            </div>

            <div class="console-grid">
              <div>
                <div class="test-title">Test case</div>
                <pre class="code-tests">{{ scenario.code_challenge.tests }}</pre>
              </div>
              <div>
                <div class="test-title">Output</div>
                <div v-if="sandboxMsg" class="code-result">{{ sandboxMsg }}</div>
                <div v-else class="code-result muted">Hali run qilinmagan.</div>
                <div v-if="sandboxResult" class="sandbox-meta">
                  <span :class="sandboxResult.passed ? 'good' : 'bad'">
                    {{ sandboxResult.passed ? 'PASSED' : 'FAILED' }}
                  </span>
                  <span v-if="sandboxResult.timed_out">timeout</span>
                  <span>exit {{ sandboxResult.exit_code }}</span>
                  <span>{{ sandboxResult.duration_ms }}ms</span>
                </div>
                <pre v-if="sandboxResult?.stderr" class="code-tests">{{ sandboxResult.stderr }}</pre>
                <pre v-if="sandboxResult?.stdout" class="code-tests">{{ sandboxResult.stdout }}</pre>
              </div>
            </div>

            <details v-if="scenario.code_challenge.buggy_code" class="buggy-block">
              <summary>Broken code</summary>
              <pre class="code-tests">{{ scenario.code_challenge.buggy_code }}</pre>
            </details>
            <div v-if="scenario.code_challenge.hint" class="code-hint">{{ scenario.code_challenge.hint }}</div>
          </template>
        </div>
        <div v-else class="empty terminal-empty">
          Code challenge yo'q. Chatga fokus qil.
        </div>
      </aside>
    </div>
  </div>
</template>

<style scoped>
.sim-shell {
  flex: 1;
  min-height: 0;
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.kicker {
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: 1px;
  color: var(--text-dim);
  font-weight: 800;
}
h3 {
  margin: 4px 0 0;
  font-size: 20px;
  line-height: 1.15;
}
.meta-pill {
  padding: 8px 10px;
  border-radius: 8px;
  background: var(--panel-2);
  border: 1px solid var(--border);
  color: var(--text-dim);
  font-size: 12px;
  white-space: nowrap;
}
.split {
  flex: 1;
  min-height: 0;
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  border: 1px solid var(--border-strong);
  background: var(--panel);
  border-radius: var(--radius);
  overflow: hidden;
}
.chat-pane,
.terminal-pane {
  background: transparent;
  border: 0;
  border-radius: 0;
  display: flex;
  flex-direction: column;
  min-height: 0;
  overflow: hidden;
  position: relative;
}
.chat-head,
.terminal-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  padding: 14px 16px;
  border-bottom: 1px solid var(--border-strong);
  background: var(--panel-2);
}
.terminal-pane {
  position: absolute;
  top: 0;
  right: 0;
  bottom: 0;
  width: min(460px, 92vw);
  border-left: 1px solid var(--border-strong);
  background: linear-gradient(180deg, var(--panel-2), var(--bg-soft));
  box-shadow: -24px 0 60px rgba(0, 0, 0, .28);
  transform: translateX(102%);
  opacity: 0;
  pointer-events: none;
  transition: transform .22s ease, opacity .22s ease;
  z-index: 3;
}
.terminal-pane.open {
  transform: translateX(0);
  opacity: 1;
  pointer-events: auto;
}
.drawer-backdrop {
  position: absolute;
  inset: 0;
  z-index: 2;
  border: 0;
  border-radius: var(--radius);
  background: rgba(8, 10, 8, .42);
  backdrop-filter: blur(2px);
}
.terminal-tools {
  display: flex;
  align-items: center;
  gap: 8px;
}
.terminal-title,
.chat-head-title {
  font-weight: 800;
  font-size: 15px;
}
.chat-tools {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
  justify-content: flex-end;
}
.chat-tools select {
  min-width: 170px;
  padding: 8px 10px;
  border-radius: 8px;
  border: 1px solid var(--border);
  background: var(--panel-2);
  color: var(--text);
}
.code-toggle {
  white-space: nowrap;
}
.stream {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 16px;
  scroll-behavior: smooth;
}
.msg { display: flex; gap: 10px; max-width: 84%; }
.msg.user { align-self: flex-end; flex-direction: row-reverse; }
.msg-body { display: flex; flex-direction: column; gap: 4px; min-width: 0; }
.msg.user .msg-body { align-items: flex-end; }
.avatar {
  flex: none;
  width: 34px; height: 34px;
  border-radius: 8px;
  display: grid; place-items: center;
  font-size: 11px; font-weight: 800;
  background: var(--panel-2);
  border: 1px solid var(--border);
  color: var(--text-dim);
}
.msg.user .avatar {
  background: var(--ink);
  color: var(--paper); border-color: var(--ink);
}
.bubble {
  background: var(--panel-2);
  border: 1px solid var(--border);
  padding: 12px 14px;
  border-radius: 8px;
  line-height: 1.55;
  font-size: 15px;
  white-space: pre-wrap;
  word-break: break-word;
}
.msg.user .bubble {
  background: var(--success-bg);
  border-color: var(--success-border);
}
.bubble :deep(code) {
  background: var(--panel-2);
  border: 1px solid var(--border);
  padding: 1px 6px;
  border-radius: 5px;
  font-size: 13px;
  font-family: var(--mono);
}
.bubble :deep(strong) { color: var(--ink); }
.time { font-size: 11px; color: var(--text-dim); padding: 0 4px; }
.typing { display: flex; gap: 5px; align-items: center; padding: 16px 15px; }
.typing span {
  width: 7px; height: 7px; border-radius: 50%;
  background: var(--text-dim);
  animation: blink 1.2s infinite ease-in-out;
}
.typing span:nth-child(2) { animation-delay: .2s; }
.typing span:nth-child(3) { animation-delay: .4s; }
@keyframes blink { 0%, 60%, 100% { opacity: .25; } 30% { opacity: 1; } }
.scroll-fab {
  position: absolute;
  right: 18px;
  bottom: 145px;
  width: 38px; height: 38px;
  border-radius: 8px;
  background: var(--panel-2);
  border: 1px solid var(--border);
  color: var(--text);
  font-size: 18px;
}
.starters {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
  padding: 0 16px 10px;
}
.chip {
  background: var(--panel-2);
  border: 1px solid var(--border);
  color: var(--text);
  font-size: 13px;
  padding: 8px 13px;
  border-radius: 8px;
}
.chat-error { color: var(--bad); font-size: 13px; padding: 8px 18px 0; margin: 0; }
.composer {
  display: flex;
  gap: 10px;
  padding: 14px 16px;
  border-top: 1px solid var(--border-strong);
  background: var(--bg-soft);
  align-items: flex-end;
}
.composer textarea {
  flex: 1;
  resize: none;
  padding: 12px 14px;
  font-size: 16px;
  line-height: 1.5;
  max-height: 140px;
  overflow-y: auto;
}
.send-btn { padding: 12px 20px; height: 46px; }
.finish-zone {
  padding: 0 16px 14px;
  background: var(--bg-soft);
  border-top: 0;
}
.finish-btns { display: flex; gap: 10px; }
.finish-btn { flex: 1; }
.terminal-pane {
  padding-bottom: 0;
}
.code-card {
  padding: 14px;
  display: flex;
  flex-direction: column;
  gap: 10px;
  min-height: 0;
  overflow: auto;
}
.code-locked {
  padding: 18px;
  border-radius: 8px;
  background: var(--panel);
  border: 1px dashed var(--border-strong);
  color: var(--text-dim);
  font-size: 14px;
}
.code-input {
  min-height: 320px;
  width: 100%;
  resize: vertical;
  font-family: var(--mono);
  font-size: 14px;
  line-height: 1.6;
  background: var(--panel-2);
  color: var(--text);
}
.editor-actions {
  display: grid;
  grid-template-columns: 1fr auto auto;
  gap: 10px;
  align-items: center;
}
.small {
  padding: 8px 12px;
  font-size: 13px;
}
.language-chip {
  justify-self: start;
  padding: 9px 11px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--panel);
  font-weight: 800;
  font-size: 13px;
}
.test-tabs {
  display: flex;
  gap: 6px;
  border-bottom: 1px solid var(--border);
}
.test-tabs span {
  padding: 9px 12px;
  font-size: 13px;
  color: var(--text-dim);
  border: 1px solid transparent;
  border-bottom: 0;
}
.test-tabs .active {
  background: var(--panel);
  color: var(--text);
  border-color: var(--border);
  border-radius: 8px 8px 0 0;
  font-weight: 800;
}
.console-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}
.test-title {
  font-size: 12px;
  text-transform: uppercase;
  letter-spacing: .6px;
  color: var(--text-dim);
  font-weight: 700;
  margin-bottom: 10px;
}
.code-tests,
.code-result {
  margin: 0;
  color: var(--text);
  white-space: pre-wrap;
  word-break: break-word;
  font-size: 13px;
  line-height: 1.55;
  font-family: var(--mono);
}
.code-result {
  font-family: var(--font);
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 10px;
}
.muted {
  color: var(--text-dim);
}
.sandbox-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 8px;
  font-size: 12px;
  color: var(--text-dim);
}
.sandbox-meta .good { color: var(--good); font-weight: 700; }
.sandbox-meta .bad { color: var(--bad); font-weight: 700; }
.buggy-block {
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 12px;
}
.buggy-block summary {
  cursor: pointer;
  font-weight: 800;
  color: var(--text-dim);
  margin-bottom: 8px;
}
.code-hint {
  margin-top: 2px;
  font-size: 13px;
  color: var(--accent);
  line-height: 1.5;
}
.terminal-empty {
  margin: 16px;
}

@media (max-width: 900px) {
  .split { border: 0; background: transparent; }
  .chat-pane,
  .terminal-pane {
    border: 1px solid var(--border-strong);
    border-radius: var(--radius);
    background: var(--panel);
  }
  .terminal-pane { width: min(100%, 92vw); }
  .msg { max-width: 92%; }
  .console-grid { grid-template-columns: 1fr; }
  .scroll-fab { bottom: 140px; }
  .chat-tools select {
    min-width: 0;
    flex: 1 1 170px;
  }
  .code-toggle {
    flex: 1 1 120px;
  }
}

@media (max-width: 560px) {
  .chat-head { align-items: stretch; flex-direction: column; }
  .chat-tools { justify-content: space-between; }
  .composer { flex-direction: column; align-items: stretch; }
  .send-btn { width: 100%; }
  .editor-actions { grid-template-columns: 1fr; }
  .editor-actions .btn-primary { width: 100%; }
  .code-input { min-height: 260px; }
  .terminal-pane {
    width: 100%;
    border-left: 0;
  }
  .drawer-backdrop {
    border-radius: 0;
  }
}
</style>
