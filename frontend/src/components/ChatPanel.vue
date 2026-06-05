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
const codeOpen = ref(defaultCodeOpen())
const chatWidth = ref(520)
const isResizing = ref(false)

const streamEl = ref(null)
const taEl = ref(null)
const splitEl = ref(null)
const codeTaEl = ref(null)
const codeOverlayEl = ref(null)
const gutterEl = ref(null)
let nextId = 0
let onKeydownGlobal = () => {}
let onPointerMove = () => {}
let onPointerUp = () => {}

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
const hasCodeChallenge = computed(() => {
  return Boolean(props.scenario?.code_challenge?.buggy_code || props.scenario?.code_challenge?.tests)
})
const lineCount = computed(() => Math.max(1, studentCode.value.split('\n').length))
const codeLines = computed(() => Array.from({ length: lineCount.value }, (_, i) => i + 1))
const diagnostics = computed(() => inspectCode(studentCode.value, props.scenario?.code_language || 'python'))
const diagnosticsByLine = computed(() => {
  const map = new Map()
  diagnostics.value.forEach((d) => {
    if (!map.has(d.line)) map.set(d.line, [])
    map.get(d.line).push(d)
  })
  return map
})
const errorCount = computed(() => diagnostics.value.filter((d) => d.type === 'error').length)
const warningCount = computed(() => diagnostics.value.filter((d) => d.type === 'warning').length)
const highlightedCode = computed(() => highlightCode(studentCode.value, props.scenario?.code_language || 'python'))
const splitStyle = computed(() => ({
  '--chat-width': `${chatWidth.value}px`,
}))

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
    codeOpen.value = defaultCodeOpen()
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
    if (event.key === 'Escape') codeOpen.value = false
  }
  onPointerMove = (event) => {
    if (!isResizing.value || !splitEl.value) return
    const rect = splitEl.value.getBoundingClientRect()
    const minChat = 360
    const minIde = 420
    const next = Math.min(Math.max(event.clientX - rect.left, minChat), rect.width - minIde)
    chatWidth.value = Math.round(next)
  }
  onPointerUp = () => {
    isResizing.value = false
    document.body.classList.remove('is-resizing-pane')
  }
  window.addEventListener('keydown', onKeydownGlobal)
  window.addEventListener('pointermove', onPointerMove)
  window.addEventListener('pointerup', onPointerUp)
})

onBeforeUnmount(() => {
  window.removeEventListener('keydown', onKeydownGlobal)
  window.removeEventListener('pointermove', onPointerMove)
  window.removeEventListener('pointerup', onPointerUp)
})

function now() {
  return new Date().toLocaleTimeString('uz', { hour: '2-digit', minute: '2-digit' })
}

function pushMsg(role, content) {
  messages.value.push({ id: nextId++, role, content, time: now() })
}

function fmt(text) {
  const esc = escapeHtml(text)
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
        ? "Topshirildi. Testlar o'tdi."
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

function defaultCodeOpen() {
  if (typeof window === 'undefined') return true
  return !window.matchMedia('(max-width: 900px)').matches
}

function startResize(event) {
  if (window.matchMedia('(max-width: 900px)').matches) return
  isResizing.value = true
  document.body.classList.add('is-resizing-pane')
  event.preventDefault()
}

function syncCodeScroll() {
  const source = codeTaEl.value
  if (!source) return
  if (codeOverlayEl.value) {
    codeOverlayEl.value.scrollTop = source.scrollTop
    codeOverlayEl.value.scrollLeft = source.scrollLeft
  }
  if (gutterEl.value) gutterEl.value.scrollTop = source.scrollTop
}

function lineClass(line) {
  const items = diagnosticsByLine.value.get(line) || []
  if (items.some((d) => d.type === 'error')) return 'has-error'
  if (items.length) return 'has-warning'
  return ''
}

function escapeHtml(text) {
  return String(text || '')
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
}

function highlightCode(code, language) {
  const escaped = escapeHtml(code || ' ')
  const lang = String(language || '').toLowerCase()
  const keywords = lang.includes('python')
    ? /\b(def|return|if|elif|else|for|while|in|and|or|not|class|try|except|finally|import|from|as|with|pass|break|continue|True|False|None)\b/g
    : /\b(function|return|if|else|for|while|const|let|var|class|try|catch|finally|import|from|export|true|false|null|undefined|async|await)\b/g

  return escaped
    .replace(/(#.*)$/gm, '<span class="tok-comment">$1</span>')
    .replace(/(&quot;.*?&quot;|'.*?')/g, '<span class="tok-string">$1</span>')
    .replace(/\b(\d+(?:\.\d+)?)\b/g, '<span class="tok-number">$1</span>')
    .replace(keywords, '<span class="tok-keyword">$1</span>')
}

function inspectCode(code, language) {
  const lang = String(language || '').toLowerCase()
  if (!lang.includes('python')) return inspectGeneric(code)
  const diagnostics = []
  const lines = String(code || '').split('\n')
  const stack = []
  const closers = { ')': '(', ']': '[', '}': '{' }
  const openers = new Set(['(', '[', '{'])

  lines.forEach((raw, index) => {
    const lineNo = index + 1
    const line = raw.replace(/\t/g, '    ')
    const trimmed = line.trim()
    if (!trimmed || trimmed.startsWith('#')) return

    for (const char of trimmed.replace(/(['"]).*?\1/g, '')) {
      if (openers.has(char)) stack.push({ char, line: lineNo })
      if (closers[char]) {
        const prev = stack.pop()
        if (!prev || prev.char !== closers[char]) {
          diagnostics.push({ line: lineNo, type: 'error', message: `Unmatched "${char}"` })
        }
      }
    }

    if (/^(def|class|if|elif|else|for|while|try|except|finally|with)\b/.test(trimmed) && !trimmed.endsWith(':')) {
      diagnostics.push({ line: lineNo, type: 'error', message: 'Python block needs ":" at line end.' })
    }

    const previous = lines[index - 1]?.trim() || ''
    if (previous.endsWith(':') && trimmed && line.search(/\S/) <= (lines[index - 1]?.search(/\S/) ?? 0)) {
      diagnostics.push({ line: lineNo, type: 'error', message: 'Indented block expected after previous line.' })
    }

    if (/\bprint\s+[^(]/.test(trimmed)) {
      diagnostics.push({ line: lineNo, type: 'warning', message: 'Use print(...) syntax in Python 3.' })
    }
  })

  stack.forEach((item) => {
    diagnostics.push({ line: item.line, type: 'error', message: `Unclosed "${item.char}"` })
  })

  return diagnostics
}

function inspectGeneric(code) {
  const diagnostics = []
  const lines = String(code || '').split('\n')
  lines.forEach((line, index) => {
    if (/=\s*=$/.test(line.trim())) {
      diagnostics.push({ line: index + 1, type: 'warning', message: 'Incomplete comparison.' })
    }
  })
  return diagnostics
}
</script>

<template>
  <div class="sim-shell" :class="{ 'code-open': codeOpen }">
    <div class="split" ref="splitEl" :style="splitStyle">
      <section class="chat-pane">
        <div class="chat-head">
          <div>
            <div class="kicker">AI Mentor</div>
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
              {{ codeOpen ? 'Hide IDE' : 'Open IDE' }}
            </button>
          </div>
        </div>

        <div class="stream" ref="streamEl" @scroll="onScroll">
          <TransitionGroup name="msg">
            <div v-for="m in messages" :key="m.id" class="msg" :class="m.role">
              <div class="avatar">{{ m.role === 'user' ? 'ME' : 'AI' }}</div>
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
          <button v-if="!atBottom" class="scroll-fab" @click="scrollDown">v</button>
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
            placeholder="Signal yoz..."
            @keydown="onKeydown"
            @input="autoGrow"
          ></textarea>
          <button class="btn-primary send-btn" :disabled="sending || !input.trim()" @click="send()">
            Yuborish
          </button>
        </div>

        <div class="finish-zone">
          <div class="finish-btns">
            <button v-if="confirming" class="btn-ghost" :disabled="grading" @click="confirming = false">Bekor</button>
            <button class="btn-primary finish-btn" :disabled="grading || userTurns === 0" @click="finishClick">
              {{ grading ? 'Baholanmoqda...' : confirming ? 'Ha, bahola' : 'Yakunlash va baho olish' }}
            </button>
          </div>
        </div>
      </section>

      <button
        v-if="hasCodeChallenge"
        class="split-resizer"
        aria-label="Resize chat and IDE"
        @pointerdown="startResize"
      ></button>

      <aside v-if="hasCodeChallenge" class="ide-pane" :class="{ open: codeOpen }" aria-label="Code IDE">
        <div class="ide-head">
          <div>
            <div class="kicker">AI Simulator IDE</div>
            <div class="terminal-title">{{ scenario.code_language || 'python' }} challenge</div>
          </div>
          <div class="ide-status">
            <span class="status-dot" :class="{ bad: errorCount, warn: !errorCount && warningCount }"></span>
            <span>{{ errorCount }} errors</span>
            <span>{{ warningCount }} warnings</span>
          </div>
        </div>

        <div v-if="!codeUnlocked" class="code-locked">
          Code challenge {{ scenario.code_challenge_after_round }}-roundda ochiladi.
        </div>

        <template v-else>
          <div class="editor-toolbar">
            <div class="select-like">
              <span class="lang-mark">PY</span>
              <span>Python</span>
            </div>
            <div class="select-like">Aa {{ 16 }}px</div>
            <button class="btn-ghost small" @click="resetCode">Qayta boshlash</button>
            <button class="btn-primary small run" :disabled="sandboxing || !studentCode.trim()" @click="runSandbox('run')">
              > Yuritish
            </button>
            <button class="btn-primary small submit" :disabled="sandboxing || !studentCode.trim()" @click="runSandbox('submit')">
              Yuborish
            </button>
          </div>

          <div class="editor-shell">
            <div class="gutter" ref="gutterEl">
              <div v-for="line in codeLines" :key="line" class="line-no" :class="lineClass(line)">
                {{ line }}
              </div>
            </div>
            <div class="code-layer">
              <pre ref="codeOverlayEl" class="code-highlight" v-html="highlightedCode"></pre>
              <textarea
                ref="codeTaEl"
                v-model="studentCode"
                class="code-input"
                spellcheck="false"
                :placeholder="scenario.code_language || 'python'"
                @scroll="syncCodeScroll"
              ></textarea>
            </div>
          </div>

          <div class="diagnostics-strip">
            <span v-if="diagnostics.length === 0" class="diag-ok">No local syntax errors</span>
            <button
              v-for="d in diagnostics.slice(0, 4)"
              :key="`${d.line}-${d.message}`"
              class="diag-pill"
              :class="d.type"
              @click="codeTaEl?.focus()"
            >
              L{{ d.line }} {{ d.message }}
            </button>
          </div>

          <div class="console-panel">
            <div class="test-tabs">
              <span class="active">Testlar</span>
              <span>Natijalar</span>
              <span>Console</span>
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
                <pre v-if="sandboxResult?.stderr" class="code-tests error-out">{{ sandboxResult.stderr }}</pre>
                <pre v-if="sandboxResult?.stdout" class="code-tests">{{ sandboxResult.stdout }}</pre>
              </div>
            </div>
          </div>

          <details v-if="scenario.code_challenge.buggy_code" class="buggy-block">
            <summary>Broken code</summary>
            <pre class="code-tests">{{ scenario.code_challenge.buggy_code }}</pre>
          </details>
          <div v-if="scenario.code_challenge.hint" class="code-hint">{{ scenario.code_challenge.hint }}</div>
        </template>
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
}
.kicker {
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: .08em;
  color: var(--accent);
  font-weight: 900;
}
h3 {
  margin: 4px 0 0;
  font-size: 20px;
  line-height: 1.15;
}
.split {
  flex: 1;
  min-height: 0;
  display: grid;
  grid-template-columns: minmax(360px, var(--chat-width)) 8px minmax(420px, 1fr);
  border: 1px solid var(--border-strong);
  background: var(--panel);
  border-radius: var(--radius);
  overflow: hidden;
}
.sim-shell:not(.code-open) .split {
  grid-template-columns: minmax(0, 1fr);
}
.sim-shell:not(.code-open) .split-resizer,
.sim-shell:not(.code-open) .ide-pane {
  display: none;
}
.chat-pane,
.ide-pane {
  min-height: 0;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  position: relative;
}
.chat-head,
.ide-head {
  min-height: 74px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  padding: 14px 16px;
  border-bottom: 1px solid var(--border-strong);
  background: linear-gradient(180deg, var(--panel-2), var(--panel));
}
.chat-tools,
.ide-status {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
  justify-content: flex-end;
}
.chat-tools select {
  min-width: 170px;
  padding: 9px 10px;
  border-radius: 8px;
  border: 1px solid var(--border);
  background: var(--panel-2);
  color: var(--text);
}
.meta-pill,
.ide-status {
  padding: 8px 10px;
  border-radius: 8px;
  background: var(--panel-2);
  border: 1px solid var(--border);
  color: var(--text-dim);
  font-size: 12px;
  white-space: nowrap;
}
.status-dot {
  width: 9px;
  height: 9px;
  border-radius: 50%;
  background: var(--good);
  box-shadow: 0 0 16px rgba(124, 207, 150, .8);
}
.status-dot.bad {
  background: var(--bad);
  box-shadow: 0 0 16px rgba(227, 138, 124, .75);
}
.status-dot.warn {
  background: var(--warn);
  box-shadow: 0 0 16px rgba(224, 181, 108, .7);
}
.split-resizer {
  width: 8px;
  border-radius: 0;
  background:
    linear-gradient(180deg, transparent, var(--border-strong), transparent),
    var(--bg-soft);
  border-left: 1px solid var(--border);
  border-right: 1px solid var(--border);
  cursor: col-resize;
}
.split-resizer:hover {
  background:
    linear-gradient(180deg, transparent, var(--accent), transparent),
    var(--panel-2);
}
.terminal-title {
  font-weight: 900;
  font-size: 15px;
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
.msg {
  display: flex;
  gap: 10px;
  max-width: 90%;
}
.msg.user {
  align-self: flex-end;
  flex-direction: row-reverse;
}
.msg-body {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}
.msg.user .msg-body {
  align-items: flex-end;
}
.avatar {
  flex: none;
  width: 34px;
  height: 34px;
  border-radius: 8px;
  display: grid;
  place-items: center;
  font-size: 11px;
  font-weight: 900;
  background: var(--panel-2);
  border: 1px solid var(--border);
  color: var(--text-dim);
}
.msg.user .avatar {
  background: var(--ink);
  color: var(--paper);
  border-color: var(--ink);
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
  background: var(--bg-soft);
  border: 1px solid var(--border);
  padding: 1px 6px;
  border-radius: 5px;
  font-size: 13px;
  font-family: var(--mono);
}
.bubble :deep(strong) {
  color: var(--accent);
}
.time {
  font-size: 11px;
  color: var(--text-dim);
  padding: 0 4px;
}
.typing {
  display: flex;
  gap: 5px;
  align-items: center;
  padding: 16px 15px;
}
.typing span {
  width: 7px;
  height: 7px;
  border-radius: 50%;
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
  width: 38px;
  height: 38px;
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
.chat-error {
  color: var(--bad);
  font-size: 13px;
  padding: 8px 18px 0;
  margin: 0;
}
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
.send-btn {
  padding: 12px 20px;
  height: 46px;
}
.finish-zone {
  padding: 0 16px 14px;
  background: var(--bg-soft);
}
.finish-btns {
  display: flex;
  gap: 10px;
}
.finish-btn {
  flex: 1;
}
.ide-pane {
  background: var(--bg);
}
.code-locked {
  margin: 16px;
  padding: 18px;
  border-radius: 8px;
  background: var(--panel);
  border: 1px dashed var(--border-strong);
  color: var(--text-dim);
  font-size: 14px;
}
.editor-toolbar {
  display: grid;
  grid-template-columns: auto auto 1fr auto auto;
  gap: 10px;
  padding: 10px 12px;
  border-bottom: 1px solid var(--border);
  background: var(--panel);
  align-items: center;
}
.select-like {
  height: 40px;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 0 12px;
  border-radius: 8px;
  border: 1px solid var(--border);
  background: var(--panel-2);
  color: var(--text);
  font-weight: 800;
  font-size: 13px;
}
.lang-mark {
  display: grid;
  place-items: center;
  width: 24px;
  height: 24px;
  border-radius: 6px;
  color: #101210;
  background: var(--accent-2);
  font-size: 11px;
}
.small {
  padding: 9px 12px;
  font-size: 13px;
  min-height: 40px;
}
.run,
.submit {
  min-width: 104px;
}
.submit {
  background: #08351f;
  border-color: rgba(124, 207, 150, .5);
  color: #98f2b4;
}
.editor-shell {
  flex: 1;
  min-height: 250px;
  display: grid;
  grid-template-columns: 54px minmax(0, 1fr);
  background: #0c0f0d;
  border-bottom: 1px solid var(--border);
}
.gutter {
  overflow: hidden;
  padding: 14px 0;
  background: #111511;
  border-right: 1px solid #263027;
  color: #7c867c;
  font-family: var(--mono);
  font-size: 14px;
  line-height: 1.65;
  text-align: right;
}
.line-no {
  height: 23.1px;
  padding-right: 12px;
  position: relative;
}
.line-no.has-error {
  color: var(--bad);
}
.line-no.has-warning {
  color: var(--warn);
}
.line-no.has-error::after,
.line-no.has-warning::after {
  content: '';
  position: absolute;
  right: 4px;
  top: 9px;
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: currentColor;
}
.code-layer {
  position: relative;
  min-width: 0;
  min-height: 0;
}
.code-highlight,
.code-input {
  position: absolute;
  inset: 0;
  margin: 0;
  padding: 14px 16px;
  border: 0;
  border-radius: 0;
  overflow: auto;
  white-space: pre;
  font-family: var(--mono);
  font-size: 14px;
  line-height: 1.65;
  tab-size: 4;
}
.code-highlight {
  pointer-events: none;
  color: #dbe5d6;
  background:
    linear-gradient(90deg, rgba(125, 184, 163, .06) 1px, transparent 1px),
    #0c0f0d;
  background-size: 80px 100%;
}
.code-input {
  resize: none;
  background: transparent;
  color: transparent;
  caret-color: #f2eee4;
  -webkit-text-fill-color: transparent;
}
.code-input::selection {
  background: rgba(125, 184, 163, .35);
}
.code-highlight :deep(.tok-keyword) { color: #7db8ff; font-weight: 800; }
.code-highlight :deep(.tok-string) { color: #e0b56c; }
.code-highlight :deep(.tok-number) { color: #cb855f; }
.code-highlight :deep(.tok-comment) { color: #75a36f; }
.diagnostics-strip {
  min-height: 42px;
  display: flex;
  gap: 8px;
  align-items: center;
  overflow-x: auto;
  padding: 8px 12px;
  border-bottom: 1px solid var(--border);
  background: var(--panel);
}
.diag-ok,
.diag-pill {
  flex: none;
  font-size: 12px;
  border-radius: 999px;
  padding: 7px 10px;
}
.diag-ok {
  color: var(--good);
  background: var(--success-bg);
  border: 1px solid var(--success-border);
}
.diag-pill {
  border: 1px solid var(--border);
  background: var(--panel-2);
  color: var(--text);
}
.diag-pill.error {
  color: var(--danger-text);
  background: var(--danger-bg);
  border-color: var(--danger-border);
}
.diag-pill.warning {
  color: var(--warn);
  border-color: rgba(224, 181, 108, .35);
}
.console-panel {
  max-height: 250px;
  overflow: auto;
  background: var(--bg-soft);
  border-bottom: 1px solid var(--border);
}
.test-tabs {
  display: flex;
  gap: 6px;
  padding: 10px 12px 0;
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
  padding: 12px;
}
.test-title {
  font-size: 12px;
  text-transform: uppercase;
  letter-spacing: .06em;
  color: var(--text-dim);
  font-weight: 800;
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
.sandbox-meta .good { color: var(--good); font-weight: 800; }
.sandbox-meta .bad { color: var(--bad); font-weight: 800; }
.error-out {
  color: var(--danger-text);
}
.buggy-block {
  margin: 10px 12px;
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
  margin: 0 12px 12px;
  font-size: 13px;
  color: var(--accent);
  line-height: 1.5;
}

@media (max-width: 1100px) {
  .split {
    grid-template-columns: minmax(330px, var(--chat-width)) 8px minmax(360px, 1fr);
  }
  .console-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 900px) {
  .split {
    display: flex;
    border: 0;
    background: transparent;
  }
  .chat-pane,
  .ide-pane {
    width: 100%;
    border: 1px solid var(--border-strong);
    border-radius: var(--radius);
    background: var(--panel);
  }
  .split-resizer {
    display: none;
  }
  .ide-pane {
    display: none;
  }
  .code-open .chat-pane {
    display: none;
  }
  .code-open .ide-pane {
    display: flex;
  }
  .msg {
    max-width: 94%;
  }
}

@media (max-width: 560px) {
  .chat-head,
  .ide-head {
    align-items: stretch;
    flex-direction: column;
  }
  .chat-tools {
    justify-content: space-between;
  }
  .chat-tools select,
  .code-toggle {
    min-width: 0;
    flex: 1 1 140px;
  }
  .composer {
    flex-direction: column;
    align-items: stretch;
  }
  .send-btn {
    width: 100%;
  }
  .editor-toolbar {
    grid-template-columns: 1fr 1fr;
  }
  .editor-toolbar .small {
    width: 100%;
  }
  .editor-shell {
    min-height: 330px;
  }
  .run,
  .submit {
    min-width: 0;
  }
}
</style>
