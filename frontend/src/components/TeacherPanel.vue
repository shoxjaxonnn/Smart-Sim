<script setup>
import { computed, onMounted, ref } from 'vue'
import { api } from '../api'

const busy = ref(false)
const error = ref('')
const documents = ref([])
const scenarios = ref([])
const selectedDocumentId = ref('')
const selectedScenarioId = ref('')
const draft = ref(newDraft())
const factsText = ref('{}')
const rubricText = ref('[]')

const uploadForm = ref({
  file: null,
  instruction: '',
  title: '',
  subject: 'IT / Web Security',
  language: 'uz',
  codeLanguage: 'python',
  problemFocus: '',
})

const sortedDocuments = computed(() => {
  return [...documents.value].sort((a, b) => (b.created_at || '').localeCompare(a.created_at || ''))
})

const sortedScenarios = computed(() => {
  return [...scenarios.value].sort((a, b) => {
    const aKey = `${a.status || ''}-${a.title || ''}`
    const bKey = `${b.status || ''}-${b.title || ''}`
    return aKey.localeCompare(bKey)
  })
})

const selectedDocument = computed(() => {
  return documents.value.find((d) => d.id === selectedDocumentId.value) || null
})

onMounted(loadAll)

function newDraft() {
  return {
    id: '',
    title: '',
    subject: 'IT / Web Security',
    language: 'uz',
    status: 'draft',
    code_challenge_after_round: 3,
    code_language: 'python',
    situation: '',
    facts: {},
    rubric: [],
    model_answer: '',
    code_challenge: {
      buggy_code: '',
      hint: '',
      tests: '',
    },
  }
}

function cloneScenario(s) {
  const base = JSON.parse(JSON.stringify(s ?? newDraft()))
  base.code_challenge = {
    buggy_code: '',
    hint: '',
    tests: '',
    ...(base.code_challenge || {}),
  }
  base.facts = base.facts || {}
  base.rubric = base.rubric || []
  return base
}

function syncSelection(s) {
  draft.value = cloneScenario(s)
  factsText.value = JSON.stringify(draft.value.facts || {}, null, 2)
  rubricText.value = JSON.stringify(draft.value.rubric || [], null, 2)
  selectedScenarioId.value = draft.value.id
}

async function loadAll() {
  try {
    const [docs, list] = await Promise.all([api.teacherDocuments(), api.teacherScenarios()])
    documents.value = docs
    scenarios.value = list
    if (!selectedDocumentId.value && docs.length) {
      selectedDocumentId.value = docs[0].id
    }
    const doc = docs.find((d) => d.id === selectedDocumentId.value) || docs[0]
    if (doc && doc.scenario_id) {
      await selectScenarioById(doc.scenario_id)
    } else if (!selectedScenarioId.value && list.length) {
      await selectScenarioById(list[0].id)
    }
  } catch (e) {
    error.value = friendlyError(e)
  }
}

async function selectScenarioById(id) {
  if (!id) return
  selectedScenarioId.value = id
  try {
    const full = await api.teacherScenario(id)
    syncSelection(full)
  } catch (e) {
    error.value = friendlyError(e)
  }
}

function selectDocument(doc) {
  selectedDocumentId.value = doc.id
  if (doc.scenario_id) {
    selectScenarioById(doc.scenario_id)
  }
}

function onFileChange(e) {
  uploadForm.value.file = e.target.files?.[0] || null
}

async function uploadDocument() {
  if (!uploadForm.value.file) {
    error.value = 'DOCX fayl tanla.'
    return
  }
  busy.value = true
  error.value = ''
  try {
    const form = new FormData()
    form.append('file', uploadForm.value.file)
    form.append('instruction', uploadForm.value.instruction || '')
    form.append('title', uploadForm.value.title || '')
    form.append('subject', uploadForm.value.subject || '')
    form.append('language', uploadForm.value.language || '')
    form.append('code_language', uploadForm.value.codeLanguage || '')
    form.append('problem_focus', uploadForm.value.problemFocus || '')
    const created = await api.uploadTeacherDocument(form)
    documents.value = [created, ...documents.value.filter((d) => d.id !== created.id)]
    selectedDocumentId.value = created.id
    uploadForm.value.file = null
    await loadAll()
  } catch (e) {
    error.value = friendlyError(e)
  } finally {
    busy.value = false
  }
}

async function generateFromDocument(doc) {
  if (!doc) return
  busy.value = true
  error.value = ''
  try {
    const res = await api.generateScenarioFromDocument(doc.id, {})
    if (res?.scenario?.id) {
      selectedScenarioId.value = res.scenario.id
      syncSelection(await api.teacherScenario(res.scenario.id))
    }
    await loadAll()
  } catch (e) {
    error.value = friendlyError(e)
  } finally {
    busy.value = false
  }
}

function toPayload() {
  let facts = {}
  let rubric = []
  try {
    facts = JSON.parse(factsText.value || '{}')
  } catch {
    throw new Error('Facts JSON notogri. JSON formatida yoz.')
  }
  try {
    rubric = JSON.parse(rubricText.value || '[]')
  } catch {
    throw new Error('Rubric JSON notogri. JSON formatida yoz.')
  }

  return {
    id: draft.value.id,
    title: draft.value.title,
    subject: draft.value.subject,
    language: draft.value.language,
    status: draft.value.status,
    code_challenge_after_round: Number(draft.value.code_challenge_after_round || 0),
    code_language: draft.value.code_language || 'python',
    situation: draft.value.situation,
    facts,
    rubric,
    model_answer: draft.value.model_answer,
    code_challenge: {
      buggy_code: draft.value.code_challenge?.buggy_code || '',
      hint: draft.value.code_challenge?.hint || '',
      tests: draft.value.code_challenge?.tests || '',
    },
  }
}

async function saveDraft() {
  if (!draft.value.id) return
  busy.value = true
  error.value = ''
  try {
    const saved = await api.updateTeacherScenario(draft.value.id, toPayload())
    syncSelection(saved)
    await loadAll()
  } catch (e) {
    error.value = friendlyError(e)
  } finally {
    busy.value = false
  }
}

async function approveDraft() {
  if (!draft.value.id) return
  busy.value = true
  error.value = ''
  try {
    const saved = await api.updateTeacherScenario(draft.value.id, toPayload())
    syncSelection(saved)
    const approved = await api.approveTeacherScenario(draft.value.id)
    syncSelection(approved)
    await loadAll()
  } catch (e) {
    error.value = friendlyError(e)
  } finally {
    busy.value = false
  }
}

function friendlyError(e) {
  const message = e?.message || 'Unknown error'
  if (message.includes('404')) {
    return 'Teacher data route not ready yet. Check backend endpoints or seed data.'
  }
  return message
}

function badgeClass(status) {
  if (status === 'approved') return 'good'
  if (status === 'draft') return 'warn'
  return 'neutral'
}
</script>

<template>
  <section class="panel teacher">
    <div class="panel-head">
      <div>
        <div class="kicker">Teacher panel</div>
        <h2>DOCX upload → AI draft → approve</h2>
      </div>
      <button class="btn-ghost" :disabled="busy" @click="loadAll">Refresh</button>
    </div>

    <p v-if="error" class="error">{{ error }}</p>

    <div class="upload-card">
      <div class="section-label">Upload document</div>
      <div class="grid-2">
        <label>
          <span>DOCX file</span>
          <input type="file" accept=".docx" @change="onFileChange" />
        </label>
        <label>
          <span>Title</span>
          <input v-model="uploadForm.title" placeholder="Ko'paytma array" />
        </label>
        <label>
          <span>Subject</span>
          <input v-model="uploadForm.subject" placeholder="IT / Web Security" />
        </label>
        <label>
          <span>Language</span>
          <input v-model="uploadForm.language" placeholder="uz" />
        </label>
        <label>
          <span>Code language</span>
          <input v-model="uploadForm.codeLanguage" placeholder="python" />
        </label>
        <label>
          <span>Problem focus</span>
          <input v-model="uploadForm.problemFocus" placeholder="Broken login flow" />
        </label>
      </div>
      <label>
        <span>Teacher instruction</span>
        <textarea
          v-model="uploadForm.instruction"
          rows="5"
          placeholder="AIga aniq nima qilishni yoz. Masalan: shu DOCXdagi mavzudan SQL injection senariysi yarat."
        ></textarea>
      </label>
      <div class="actions">
        <button class="btn-primary" :disabled="busy" @click="uploadDocument">
          {{ busy ? 'Working…' : 'Upload DOCX' }}
        </button>
      </div>
    </div>

    <div class="teacher-grid">
      <div class="teacher-list">
        <div class="section-label">Documents</div>
        <button
          v-for="d in sortedDocuments"
          :key="d.id"
          class="scenario-card"
          :class="{ active: d.id === selectedDocumentId }"
          @click="selectDocument(d)"
        >
          <div class="scenario-top">
            <span class="scenario-title">{{ d.title || d.file_name }}</span>
            <span class="status" :class="badgeClass(d.scenario_id ? 'approved' : 'draft')">
              {{ d.scenario_id ? 'linked' : 'new' }}
            </span>
          </div>
          <div class="scenario-sub">{{ d.file_name }}</div>
          <div class="scenario-id">{{ (d.parsed_text || '').slice(0, 110) }}</div>
          <div class="mini-actions">
            <button class="mini-btn" :disabled="busy" @click.stop="generateFromDocument(d)">
              Generate
            </button>
          </div>
        </button>
        <div v-if="sortedDocuments.length === 0" class="empty">
          Hali document yo'q. DOCX yukla.
        </div>
      </div>

      <div class="teacher-editor">
        <div class="section-label">Scenario editor</div>
        <div v-if="selectedDocument" class="doc-preview">
          <div class="preview-head">
            <strong>{{ selectedDocument.title || selectedDocument.file_name }}</strong>
            <span class="preview-meta">{{ selectedDocument.file_name }}</span>
          </div>
          <pre class="preview-text">{{ selectedDocument.parsed_text }}</pre>
          <div class="actions">
            <button class="btn-ghost" :disabled="busy" @click="generateFromDocument(selectedDocument)">
              Generate draft from doc
            </button>
          </div>
        </div>

        <div v-if="draft.id" class="editor-block">
          <div class="grid-2">
            <label>
              <span>ID</span>
              <input v-model="draft.id" disabled />
            </label>
            <label>
              <span>Status</span>
              <input v-model="draft.status" />
            </label>
            <label>
              <span>Title</span>
              <input v-model="draft.title" />
            </label>
            <label>
              <span>Subject</span>
              <input v-model="draft.subject" />
            </label>
            <label>
              <span>Language</span>
              <input v-model="draft.language" />
            </label>
            <label>
              <span>Code language</span>
              <input v-model="draft.code_language" />
            </label>
            <label>
              <span>Reveal round</span>
              <input v-model="draft.code_challenge_after_round" type="number" min="1" />
            </label>
          </div>

          <label>
            <span>Situation</span>
            <textarea v-model="draft.situation" rows="5"></textarea>
          </label>

          <div class="grid-2">
            <label>
              <span>Model answer</span>
              <textarea v-model="draft.model_answer" rows="8"></textarea>
            </label>
            <label>
              <span>Broken code</span>
              <textarea v-model="draft.code_challenge.buggy_code" rows="8"></textarea>
            </label>
          </div>

          <div class="grid-2">
            <label>
              <span>Hint</span>
              <textarea v-model="draft.code_challenge.hint" rows="5"></textarea>
            </label>
            <label>
              <span>Tests</span>
              <textarea v-model="draft.code_challenge.tests" rows="5"></textarea>
            </label>
          </div>

          <div class="grid-2">
            <label>
              <span>Facts JSON</span>
              <textarea v-model="factsText" rows="10" class="mono"></textarea>
            </label>
            <label>
              <span>Rubric JSON</span>
              <textarea v-model="rubricText" rows="10" class="mono"></textarea>
            </label>
          </div>

          <div class="actions">
            <button class="btn-ghost" :disabled="busy" @click="saveDraft">Save draft</button>
            <button class="btn-primary" :disabled="busy" @click="approveDraft">Approve</button>
          </div>
        </div>

        <div v-else class="empty editor-empty">
          Document dan draft chiqardi. Yoki old draft ni tanla.
        </div>
      </div>
    </div>

    <div class="teacher-grid bottom-grid">
      <div class="teacher-list">
        <div class="section-label">Scenarios</div>
        <button
          v-for="s in sortedScenarios"
          :key="s.id"
          class="scenario-card"
          :class="{ active: s.id === selectedScenarioId }"
          @click="selectScenarioById(s.id)"
        >
          <div class="scenario-top">
            <span class="scenario-title">{{ s.title }}</span>
            <span class="status" :class="badgeClass(s.status)">{{ s.status || 'draft' }}</span>
          </div>
          <div class="scenario-sub">{{ s.subject }}</div>
          <div class="scenario-id">{{ s.id }}</div>
        </button>
        <div v-if="sortedScenarios.length === 0" class="empty">
          Hali scenario yo'q.
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.panel {
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  padding: 20px;
  box-shadow: var(--shadow);
  min-height: 0;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.panel-head {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  align-items: center;
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
.error {
  background: var(--danger-bg);
  border: 1px solid var(--danger-border);
  color: var(--danger-text);
  padding: 10px 12px;
  border-radius: 10px;
  margin: 0;
}
.upload-card {
  background: linear-gradient(160deg, var(--panel-2), var(--bg-soft));
  border: 1px solid var(--border);
  border-radius: 14px;
  padding: 16px;
  box-shadow: var(--shadow);
}
.teacher-grid {
  min-height: 0;
  display: grid;
  grid-template-columns: 320px 1fr;
  gap: 16px;
}
.bottom-grid {
  grid-template-columns: 1fr;
}
.teacher-list,
.teacher-editor {
  min-height: 0;
  overflow: auto;
}
.teacher-list {
  padding-right: 4px;
}
.section-label {
  font-size: 12px;
  text-transform: uppercase;
  letter-spacing: .7px;
  color: var(--text-dim);
  margin-bottom: 10px;
  font-weight: 700;
}
.scenario-card {
  width: 100%;
  text-align: left;
  background: linear-gradient(180deg, var(--panel-2), var(--panel));
  border: 1px solid var(--border);
  color: var(--text);
  padding: 12px;
  border-radius: 12px;
  margin-bottom: 10px;
}
.scenario-card.active {
  border-color: var(--accent);
  box-shadow: 0 0 0 1px var(--focus-ring) inset;
}
.scenario-top {
  display: flex;
  justify-content: space-between;
  gap: 10px;
  align-items: start;
}
.scenario-title { font-weight: 700; font-size: 14px; line-height: 1.3; }
.scenario-sub { margin-top: 6px; font-size: 12px; color: var(--text-dim); }
.scenario-id { margin-top: 4px; font-size: 11px; color: var(--text-dim); opacity: .8; word-break: break-word; }
.status {
  font-size: 11px;
  padding: 4px 8px;
  border-radius: 999px;
  border: 1px solid var(--border);
  flex: none;
  text-transform: uppercase;
  letter-spacing: .5px;
}
.status.good { color: var(--good); border-color: rgba(52, 211, 153, .35); }
.status.warn { color: var(--warn); border-color: rgba(251, 191, 36, .35); }
.status.neutral { color: var(--text-dim); }
.empty {
  color: var(--text-dim);
  font-size: 14px;
  line-height: 1.5;
  border: 1px dashed var(--border);
  border-radius: 12px;
  padding: 14px;
}
.editor-empty { min-height: 160px; display: grid; place-items: center; }
label {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-bottom: 12px;
}
label span {
  font-size: 12px;
  color: var(--text-dim);
  font-weight: 600;
}
input, textarea {
  width: 100%;
  padding: 11px 12px;
  font-size: 14px;
}
.grid-2 {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}
.actions {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
  margin-top: 2px;
}
.editor-block {
  padding-top: 8px;
  border-top: 1px solid var(--border);
}
.mono {
  font-family: 'Cascadia Code', 'Consolas', monospace;
  font-size: 12.5px;
}
.doc-preview {
  margin-bottom: 14px;
  background: linear-gradient(160deg, var(--panel-2), var(--bg-soft));
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 14px;
}
.preview-head {
  display: flex;
  justify-content: space-between;
  gap: 8px;
  align-items: center;
  margin-bottom: 10px;
}
.preview-meta {
  font-size: 12px;
  color: var(--text-dim);
}
.preview-text {
  margin: 0;
  white-space: pre-wrap;
  max-height: 240px;
  overflow: auto;
  color: var(--text);
  line-height: 1.55;
}
.mini-actions {
  margin-top: 10px;
  display: flex;
  justify-content: flex-end;
}
.mini-btn {
  background: var(--panel-2);
  color: var(--text);
  border: 1px solid var(--border);
  padding: 7px 10px;
  border-radius: 999px;
}

@media (max-width: 1180px) {
  .teacher-grid { grid-template-columns: 1fr; }
}

@media (max-width: 760px) {
  .grid-2 { grid-template-columns: 1fr; }
}
</style>
