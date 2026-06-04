<script setup>
import { ref, nextTick, onMounted, computed } from 'vue'
import { api } from '../api'

const props = defineProps({
  scenario: { type: Object, required: true },
  sessionId: { type: String, required: true },
})
const emit = defineEmits(['finished'])

const messages = ref([])
const input = ref('')
const sending = ref(false)
const grading = ref(false)
const confirming = ref(false)
const errorMsg = ref('')
const atBottom = ref(true)

const streamEl = ref(null)
const taEl = ref(null)
let nextId = 0

const starters = [
  'Server error logida nima yozilgan?',
  'Qaysi jadval hujumga uchragan?',
  'Bu qanday hujum turi deb o\'ylaysan?',
]

const userTurns = computed(() => messages.value.filter((m) => m.role === 'user').length)
const showStarters = computed(() => userTurns.value === 0 && !sending.value)

onMounted(() => {
  pushMsg('assistant',
    `Salom! "${props.scenario.title}" vaziyatini o'qib chiqding. ` +
    `Tayyor bo'lsang — muammoni qanday tekshira boshlaysan?`)
})

function now() {
  return new Date().toLocaleTimeString('uz', { hour: '2-digit', minute: '2-digit' })
}

function pushMsg(role, content) {
  messages.value.push({ id: nextId++, role, content, time: now() })
}

// Light, safe formatter: escape HTML, then re-enable `code` and **bold**.
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
    pushMsg('assistant', '⚠️ Xatolik: ' + e.message)
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

function onKeydown(e) {
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    send()
  }
}
</script>

<template>
  <div class="chat-layout">
    <aside class="situation">
      <div class="sit-head">
        <span class="sit-tag">Vaziyat</span>
        <h3>{{ scenario.title }}</h3>
      </div>
      <p class="sit-body">{{ scenario.situation }}</p>

      <div class="sit-meta">
        <div class="meta-row">
          <span>Savol-javob</span>
          <span class="meta-val">{{ userTurns }}</span>
        </div>
        <div class="meta-row">
          <span>Faktlar himoyasi</span>
          <span class="meta-badge">🛡️ Faol</span>
        </div>
      </div>

      <div class="sit-hint">
        💡 AI yo'naltiradi, lekin yechimni o'zing topasan. Aniq raqam kerak bo'lsa — so'ra,
        AI faqat tasdiqlangan faktni beradi (to'qima yo'q).
      </div>

      <div class="finish-zone">
        <p v-if="confirming" class="confirm-text">
          Yakunlasang, suhbat baholanadi. Davom etamizmi?
        </p>
        <div class="finish-btns">
          <button
            v-if="confirming"
            class="btn-ghost"
            :disabled="grading"
            @click="confirming = false"
          >Bekor</button>
          <button
            class="btn-primary finish-btn"
            :disabled="grading || userTurns === 0"
            @click="finishClick"
          >
            {{ grading ? 'Baholanmoqda…' : confirming ? 'Ha, bahola' : 'Yakunlash va baho olish' }}
          </button>
        </div>
        <p v-if="userTurns === 0" class="finish-note">Avval kamida bitta javob yoz.</p>
      </div>
    </aside>

    <section class="chat">
      <div class="chat-head">
        <span class="dot"></span>
        <span class="chat-head-title">AI Tutor</span>
        <span class="chat-head-sub">{{ scenario.subject }}</span>
      </div>

      <div class="stream" ref="streamEl" @scroll="onScroll">
        <TransitionGroup name="msg">
          <div
            v-for="m in messages"
            :key="m.id"
            class="msg"
            :class="m.role"
          >
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
          <span class="starters-label">Boshlash uchun:</span>
          <button
            v-for="s in starters"
            :key="s"
            class="chip"
            @click="send(s)"
          >{{ s }}</button>
        </div>
      </Transition>

      <p v-if="errorMsg" class="chat-error">{{ errorMsg }}</p>

      <div class="composer">
        <textarea
          ref="taEl"
          v-model="input"
          rows="1"
          placeholder="Javobingni yoz…  (Enter — yuborish · Shift+Enter — yangi qator)"
          @keydown="onKeydown"
          @input="autoGrow"
        ></textarea>
        <button class="btn-primary send-btn" :disabled="sending || !input.trim()" @click="send()">
          <span>Yuborish</span>
        </button>
      </div>
    </section>
  </div>
</template>

<style scoped>
.chat-layout {
  flex: 1;
  min-height: 0;
  display: grid;
  grid-template-columns: 320px 1fr;
  gap: 18px;
}

/* ---- situation sidebar ---- */
.situation {
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  padding: 22px;
  display: flex;
  flex-direction: column;
  overflow-y: auto;
}
.sit-tag {
  font-size: 11px; text-transform: uppercase; letter-spacing: .7px;
  color: var(--accent); font-weight: 700;
}
.sit-head h3 { margin: 8px 0 14px; font-size: 18px; }
.sit-body { color: var(--text); line-height: 1.65; font-size: 14.5px; white-space: pre-line; margin: 0; }

.sit-meta {
  margin-top: 16px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.meta-row {
  display: flex; justify-content: space-between; align-items: center;
  font-size: 13px; color: var(--text-dim);
  padding: 9px 12px;
  background: var(--bg-soft);
  border: 1px solid var(--border);
  border-radius: 9px;
}
.meta-val { color: var(--text); font-weight: 700; font-variant-numeric: tabular-nums; }
.meta-badge { color: var(--good); font-weight: 600; font-size: 12px; }

.sit-hint {
  margin-top: 14px;
  background: var(--bg-soft);
  border: 1px solid var(--border);
  border-radius: 10px;
  padding: 12px;
  font-size: 13px;
  color: var(--text-dim);
  line-height: 1.5;
}

.finish-zone { margin-top: auto; padding-top: 16px; }
.confirm-text { font-size: 13px; color: var(--warn); margin: 0 0 10px; line-height: 1.45; }
.finish-btns { display: flex; gap: 10px; }
.finish-btn { flex: 1; }
.finish-note { font-size: 12px; color: var(--text-dim); margin: 8px 0 0; text-align: center; }

/* ---- chat column ---- */
.chat {
  position: relative;
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  display: flex;
  flex-direction: column;
  min-height: 0;
  overflow: hidden;
}
.chat-head {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 14px 20px;
  border-bottom: 1px solid var(--border);
  background: var(--bg-soft);
}
.dot {
  width: 9px; height: 9px; border-radius: 50%;
  background: var(--good);
  box-shadow: 0 0 0 4px rgba(52, 211, 153, .15);
}
.chat-head-title { font-weight: 650; font-size: 15px; }
.chat-head-sub { margin-left: auto; font-size: 12px; color: var(--text-dim); }

.stream {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 22px;
  display: flex;
  flex-direction: column;
  gap: 16px;
  scroll-behavior: smooth;
}
.msg { display: flex; gap: 12px; max-width: 82%; }
.msg.user { align-self: flex-end; flex-direction: row-reverse; }
.msg-body { display: flex; flex-direction: column; gap: 4px; min-width: 0; }
.msg.user .msg-body { align-items: flex-end; }

.avatar {
  flex: none;
  width: 38px; height: 38px;
  border-radius: 10px;
  display: grid; place-items: center;
  font-size: 12px; font-weight: 700;
  background: var(--panel-2);
  border: 1px solid var(--border);
  color: var(--text-dim);
}
.msg.user .avatar {
  background: linear-gradient(135deg, var(--accent), var(--accent-2));
  color: #fff; border: none;
}
.bubble {
  background: var(--bg-soft);
  border: 1px solid var(--border);
  padding: 12px 15px;
  border-radius: 12px;
  line-height: 1.55;
  font-size: 14.5px;
  white-space: pre-wrap;
  word-break: break-word;
}
.msg.assistant .bubble { border-top-left-radius: 4px; }
.msg.user .bubble {
  border-top-right-radius: 4px;
  background: linear-gradient(135deg, rgba(109,139,255,.22), rgba(139,92,246,.22));
  border-color: rgba(109,139,255,.4);
}
.bubble :deep(code) {
  background: rgba(139,92,246,.18);
  border: 1px solid rgba(139,92,246,.3);
  padding: 1px 6px;
  border-radius: 6px;
  font-size: 13px;
  font-family: 'Cascadia Code', 'Consolas', monospace;
}
.bubble :deep(strong) { color: #fff; }
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

/* scroll-to-bottom button */
.scroll-fab {
  position: absolute;
  right: 20px;
  bottom: 150px;
  width: 38px; height: 38px;
  border-radius: 50%;
  background: var(--panel-2);
  border: 1px solid var(--border);
  color: var(--text);
  font-size: 18px;
  box-shadow: var(--shadow);
}
.scroll-fab:hover { background: #243056; }

/* starter chips */
.starters {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
  padding: 0 16px 4px;
}
.starters-label { font-size: 12px; color: var(--text-dim); }
.chip {
  background: var(--panel-2);
  border: 1px solid var(--border);
  color: var(--text);
  font-size: 13px;
  padding: 8px 13px;
  border-radius: 999px;
}
.chip:hover { border-color: var(--accent); background: #243056; }

.chat-error { color: var(--bad); font-size: 13px; padding: 8px 20px 0; margin: 0; }

.composer {
  display: flex;
  gap: 12px;
  padding: 16px;
  border-top: 1px solid var(--border);
  background: var(--bg-soft);
  align-items: flex-end;
}
.composer textarea {
  flex: 1;
  resize: none;
  padding: 12px 14px;
  font-size: 14.5px;
  line-height: 1.5;
  max-height: 140px;
  overflow-y: auto;
}
.send-btn { padding: 12px 20px; height: 46px; }

/* transitions */
.msg-enter-active { transition: all .28s cubic-bezier(.2,.8,.2,1); }
.msg-enter-from { opacity: 0; transform: translateY(10px); }

@media (max-width: 860px) {
  .chat-layout { grid-template-columns: 1fr; }
  .situation { max-height: 240px; }
  .msg { max-width: 92%; }
  .scroll-fab { bottom: 140px; }
}
</style>
