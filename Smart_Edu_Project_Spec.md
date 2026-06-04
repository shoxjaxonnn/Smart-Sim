# Smart Edu — AI Simulation Platform

> **Versiya:** MVP v2 (Hackathon Scope)
> **Fokus:** IT yo'nalishi (lekin arxitektura har qanday fan uchun ochiq)
> **Stack:** Go · Vue.js 3 · PostgreSQL + pgvector · Gemini API

---

## 1. Loyiha haqida

**Smart Edu** — an'anaviy testlarni *situatsion AI simulyatsiyalar* bilan almashtiradigan interaktiv ta'lim platformasi. Talaba statik savollarga javob bermaydi — u boshqariladigan, AI yuritadigan muhitda real muammoni hal qiladi.

### Asosiy g'oya: "Fan — bu ma'lumot, kod emas"

MVP **IT yo'nalishiga** qaratilgan (SQL injection, server xatolari, debugging va h.k.), lekin platforma shunday quriladiki, **istalgan fan** uni ishlatishi mumkin. Kimyo, tarix, iqtisod — farqi yo'q. Yangi fan qo'shish uchun kod yozilmaydi, faqat yangi **senariy fayli** qo'shiladi.

### Goals (MVP)
- Educator Markdown'da senariy yozadi → platforma uni ishga tushiradi.
- Talaba AI bilan real vaqtda muloqot qilib muammoni yechadi.
- Tizim talaba javobini rubrika bo'yicha avtomatik baholaydi.
- AI **hech qachon ma'lumot to'qimaydi** (anti-hallucination kafolatlangan).

### Non-Goals (MVP'da YO'Q)
- Medicina yo'nalishi (xavf yuqori — keyingi bosqichga qoldirildi).
- Foydalanuvchi autentifikatsiyasi/rollar tizimi (demo uchun soddalashtirilgan).
- Mobil ilova.
- Bir nechta LLM provayderni bir vaqtda ishlatish (lekin almashtirish oson — 6-bo'limga qarang).

---

## 2. Arxitektura

```
┌─────────────────┐         ┌──────────────────────────────────┐
│   Vue.js 3 UI   │  HTTP   │            Go Backend              │
│  (Student Chat) │ ──────► │                                    │
└─────────────────┘         │  ┌──────────────────────────────┐  │
                            │  │   POST /chat  (Simulation)    │  │
                            │  └───────────┬──────────────────┘  │
                            │              │                     │
                            │  ┌───────────▼──────────┐          │
                            │  │   RAG Retriever       │          │
                            │  │   (pgvector search)   │          │
                            │  └───────────┬──────────┘          │
                            │              │                     │
                            │  ┌───────────▼──────────┐          │
                            │  │   LLM Service         │  tool:   │
                            │  │   (interface)         │◄─ get_fact│
                            │  └───────────┬──────────┘          │
                            └──────────────┼─────────────────────┘
                                           │
                          ┌────────────────┼────────────────┐
                          ▼                ▼                ▼
                  ┌──────────────┐  ┌─────────────┐  ┌────────────┐
                  │  Gemini API  │  │ PostgreSQL  │  │ Facts Store│
                  │  (LLM)       │  │ + pgvector  │  │ (per-scen.)│
                  └──────────────┘  └─────────────┘  └────────────┘
```

### Texnologiya tanlovi
| Qatlam | Texnologiya | Sabab |
|---|---|---|
| Backend | **Go (Golang)** | Yuqori concurrency, tez API, oddiy deploy |
| Frontend | **Vue.js 3** | Reaktiv, tez, real-vaqt chat uchun qulay |
| Ma'lumotlar bazasi | **PostgreSQL + pgvector** | Semantik qidiruv + oddiy SQL bir joyda |
| LLM | **Gemini API (Flash)** | Bepul tier, arzon, hackathon uchun ideal |
| Embedding | **`all-MiniLM-L6-v2` (384 dim)** yoki hosted | Quyida — 5-bo'lim |

---

## 3. Uchta asosiy yechim (Zaifliklarni hal qilish)

Bu uchta qaror butun loyihaning yadrosi. Ularning hammasi "har qanday fan uchun ishlaydi" g'oyasini kuchaytiradi.

### 3.1. Anti-Hallucination — *kafolatlangan*, umid emas

**Muammo:** "System prompt LLM'ga to'qima deb aytadi" — bu umid. LLM aqlli savol oldida ma'lumot oqib chiqaradi.

**Yechim:** LLM raqamlarni **eslab qolmaydi** — u ularni **vosita orqali qidiradi**.

Har bir senariyda alohida **Facts Store** (kalit/qiymat jadvali) bo'ladi. LLM'ga faqat bitta vosita beriladi: `get_fact(key)`.

```
get_fact("server.cpu")  → "94%"          (faktlar omborida bor)
get_fact("server.ram")  → "unavailable"  (yo'q → qattiq qoida)
```

LLM raqamni **o'zi hech qachon yaratmaydi** — u faqat `get_fact` chaqiradi. Agar kalit topilmasa, kod (LLM emas) `"unavailable"` qaytaradi. Shunda `"That information is currently unavailable."` javobi **mexanik kafolatlangan**.

Aynan shu mexanizm IT uchun (`log.line_42`), kimyo uchun (`reaction.temp`), tarix uchun (`treaty.year`) — bir xil ishlaydi.

### 3.2. Baholash — Rubrika asosiy, kalit-so'z yordamchi

**Muammo:** Sof kalit-so'z baholash aldash oson. Talaba so'zlarni shunchaki sanab ketsa, yuqori ball oladi.

**Yechim:** Educator kichik **rubrika** (mezon + og'irlik) beradi. LLM har bir mezonni baholab, **strukturali JSON** qaytaradi:

```json
{
  "total_score": 7,
  "max_score": 10,
  "criteria": [
    { "name": "Muammoni to'g'ri aniqladi", "score": 3, "max": 3,
      "justification": "Talaba SQL injection ekanini aniq topdi." },
    { "name": "Yechim taklif qildi",        "score": 3, "max": 4,
      "justification": "Parametrlangan so'rovni aytdi, lekin validatsiyani tushirib qoldirdi." },
    { "name": "Kalit atamalar (to'g'ri ishlatilgan)", "score": 1, "max": 3,
      "justification": "'prepared statement' atamasini to'g'ri kontekstda ishlatdi." }
  ]
}
```

**Qoida:** kalit-so'z balli faqat atama **to'g'ri va tushunib** ishlatilganda beriladi — shunchaki eslatilganda emas.

### 3.3. LLM Abstraktsiya — bitta uyga yig'

**Muammo:** Demo paytida Gemini ishlamay qolsa, butun kodni qayta yozib bo'lmaydi.

**Yechim:** Barcha LLM chaqiruvlari bitta Go interfeysi ortida. Provayderni almashtirish = bitta faylni o'zgartirish (6-bo'limga qarang).

---

## 4. Senariy fayl formati (Eng muhim artefakt)

Bu fayl uchchala yechimni bog'laydi. Educator faqat shuni yozadi — prose + faktlar + rubrika bir joyda. Frontmatter (YAML) + Markdown.

```markdown
---
id: "sql-injection-001"
title: "Shubhali login formasi"
subject: "IT / Web Security"     # istalgan fan shu yerda
language: "uz"

# 3.1 — Facts Store (LLM faqat shu yerdan ma'lumot oladi)
facts:
  server.error_log: "Error: unexpected token near '--' in query"
  db.table: "users"
  login.field: "username"
  server.cpu: "Bu ma'lumot mavjud emas"   # ataylab cheklangan ham bo'lishi mumkin

# 3.2 — Grading Rubric
rubric:
  - name: "Hujum turini aniqlash"
    max: 3
    keywords: ["SQL injection", "injeksiya"]
  - name: "Sababni tushuntirish"
    max: 4
    keywords: ["validatsiya", "sanitatsiya", "user input"]
  - name: "Yechim taklif qilish"
    max: 3
    keywords: ["prepared statement", "parametrlangan", "ORM"]

model_answer: >
  Bu SQL injection hujumi. Login formasi foydalanuvchi kiritmasini
  to'g'ridan-to'g'ri so'rovga qo'shgani uchun yuzaga keladi. Yechim —
  parametrlangan so'rovlar (prepared statements) ishlatish.
---

## Vaziyat (talabaga ko'rsatiladigan matn)

Sen DevOps muhandisisan. Tungi 02:00 da `users` jadvaliga g'alati
so'rovlar tushayotgani haqida ogohlantirish keldi. Login sahifasida
nimadir noto'g'ri. Muammoni aniqla va hal qil.

(LLM bu matnni yetkazadi, lekin undan chetga chiqmaydi. Aniq raqam
so'ralsa — faqat `facts` ichidan get_fact orqali beradi.)
```

**Ishlash tartibi:**
1. `## Vaziyat` ostidagi prose → chunklanadi, embed qilinadi, `pgvector`ga yoziladi (RAG).
2. `facts:` → Facts Store sifatida saqlanadi (`get_fact` faqat shundan o'qiydi).
3. `rubric:` + `model_answer` → baholash vaqtida ishlatiladi.

---

## 5. Embedding modeli

Buni **hozir** tanlash kerak — u `pgvector` ustun o'lchamini belgilaydi va keyin o'zgartirish og'riqli (qayta embed qilish kerak bo'ladi).

| Variant | O'lcham | Qachon | Eslatma |
|---|---|---|---|
| **`all-MiniLM-L6-v2` (lokal)** | 384 | Wifi ishonchsiz bo'lsa | Demo internetsiz ham ishlaydi ✅ |
| **Gemini `text-embedding-004`** | 768 | Wifi mustahkam bo'lsa | Bitta xizmat kam — qulay |

**Tavsiya:** Demo xavfsizligi uchun **lokal model (384 dim)**. Go'da embedding ekotizimi kuchsiz, shuning uchun kichik **Python sidecar** (sentence-transformers) ko'taring, Go undan HTTP orqali so'raydi.

> ⚠️ **Qoida:** `pgvector` ustun o'lchamini birinchi kuni o'rnat va tegma:
> `embedding vector(384)`

---

## 6. LLM Service — Go interfeys skeleti

Barcha provayderlar shu interfeysni implement qiladi. Almashtirish = `main.go`'da bitta qatorni o'zgartirish.

```go
package llm

import "context"

// Fact — get_fact vositasi qaytaradigan natija
type Fact struct {
	Key   string
	Value string
	Found bool
}

// FactsStore — senariy faktlar ombori (3.1)
type FactsStore map[string]string

func (fs FactsStore) Get(key string) Fact {
	v, ok := fs[key]
	if !ok {
		return Fact{Key: key, Value: "That information is currently unavailable.", Found: false}
	}
	return Fact{Key: key, Value: v, Found: true}
}

// ChatRequest — bitta muloqot qadami
type ChatRequest struct {
	SystemPrompt string
	History      []Message
	UserMessage  string
	Facts        FactsStore // get_fact shu yerdan o'qiydi
}

type Message struct {
	Role    string // "user" yoki "assistant"
	Content string
}

// GradeResult — rubrika baholash natijasi (3.2)
type GradeResult struct {
	TotalScore int             `json:"total_score"`
	MaxScore   int             `json:"max_score"`
	Criteria   []CriterionScore `json:"criteria"`
}

type CriterionScore struct {
	Name          string `json:"name"`
	Score         int    `json:"score"`
	Max           int    `json:"max"`
	Justification string `json:"justification"`
}

// Provider — har bir LLM (Gemini, OpenAI...) shuni implement qiladi
type Provider interface {
	// Chat — talaba bilan muloqot. get_fact vositasini avtomatik ulaydi.
	Chat(ctx context.Context, req ChatRequest) (string, error)

	// Grade — talaba javobini rubrika bo'yicha baholaydi, JSON qaytaradi.
	Grade(ctx context.Context, modelAnswer, studentAnswer string, rubric []Criterion) (GradeResult, error)
}

type Criterion struct {
	Name     string
	Max      int
	Keywords []string
}
```

**Gemini implementatsiyasi** (`llm/gemini.go`) shu interfeysni to'ldiradi. Keyin OpenAI kerak bo'lsa — `llm/openai.go` yoziladi, qolgan kod **tegmaydi**.

```go
// main.go — almashtirish shu yerda, bitta qator:
var provider llm.Provider = llm.NewGemini(apiKey)
// var provider llm.Provider = llm.NewOpenAI(apiKey)  // ← zaxira
```

---

## 7. Ma'lumotlar bazasi sxemasi

```sql
CREATE EXTENSION IF NOT EXISTS vector;

-- Senariylar
CREATE TABLE scenarios (
    id           TEXT PRIMARY KEY,
    title        TEXT NOT NULL,
    subject      TEXT NOT NULL,
    model_answer TEXT NOT NULL,
    facts        JSONB NOT NULL,   -- Facts Store (3.1)
    rubric       JSONB NOT NULL,   -- Grading rubric (3.2)
    created_at   TIMESTAMPTZ DEFAULT now()
);

-- RAG uchun chunklar
CREATE TABLE scenario_chunks (
    id          SERIAL PRIMARY KEY,
    scenario_id TEXT REFERENCES scenarios(id) ON DELETE CASCADE,
    content     TEXT NOT NULL,
    embedding   vector(384)       -- embedding model o'lchamiga MOS
);

-- Semantik qidiruv indeksi
CREATE INDEX ON scenario_chunks USING ivfflat (embedding vector_cosine_ops);

-- Talaba sessiyalari (baholash uchun)
CREATE TABLE sessions (
    id          SERIAL PRIMARY KEY,
    scenario_id TEXT REFERENCES scenarios(id),
    history     JSONB,            -- muloqot tarixi
    grade       JSONB,            -- GradeResult (tugagach yoziladi)
    created_at  TIMESTAMPTZ DEFAULT now()
);
```

---

## 8. API endpointlari

| Method | Endpoint | Vazifa |
|---|---|---|
| `POST` | `/scenarios` | Educator senariy yuklaydi (ingestion) |
| `POST` | `/chat` | Talaba muloqoti (RAG + get_fact + persona) |
| `POST` | `/grade` | Sessiyani yakunlab, rubrika bo'yicha baholaydi |
| `GET`  | `/scenarios/:id` | Senariy ma'lumotini olish |

### `POST /chat` ish oqimi (eng muhim)
```
1. session_id + user_message qabul qil
2. user_message'ni embed qil → pgvector'da o'xshash chunk qidir (RAG)
3. System prompt + topilgan kontekst + tarixni yig'
4. LLM.Chat() chaqir → LLM kerak bo'lsa get_fact(key) chaqiradi
5. get_fact → faqat Facts Store'dan o'qiydi (to'qima YO'Q)
6. Javobni talabaga qaytar, tarixga yoz
```

---

## 9. Implementatsiya bosqichlari (Hackathon)

> Tartib muhim — har bosqich keyingisini ochadi.

### Faza 0 — Setup (eng avval)
- [ ] Embedding model tanla va o'lchamni qotirib qo'y (384).
- [ ] PostgreSQL + pgvector ko'tar, sxemani yarat.
- [ ] Gemini API kalitini Google AI Studio'dan ol (bepul tier).

### Faza 1 — Ingestion Pipeline
- [ ] Go utility: Markdown senariyni pars qil (frontmatter + prose).
- [ ] Prose'ni chunkla → embed qil → `scenario_chunks`ga yoz.
- [ ] `facts`, `rubric`, `model_answer`ni `scenarios`ga yoz.

### Faza 2 — Simulation API (`POST /chat`)
- [ ] `llm.Provider` interfeysi + Gemini implementatsiyasi.
- [ ] `get_fact` vositasini ulang (3.1 — kafolatlangan).
- [ ] RAG retrieval + persona system prompt.

### Faza 3 — Student UI (Vue 3)
- [ ] Chat interfeysi (real-vaqt, xabarlar oqimi).
- [ ] Senariy boshlash + "Yakunlash" tugmasi → `/grade`.
- [ ] Baholash natijasini rubrika ko'rinishida ko'rsat.

### Faza 4 (Bonus, vaqt qolsa)
- [ ] Grading'ni `/grade` orqali ulash (3.2 — JSON rubrika).
- [ ] Educator uchun oddiy senariy yuklash sahifasi.

---

## 10. Demo'da xavflar va ularning oldini olish

| Xavf | Oldini olish |
|---|---|
| Gemini JSON sintaksisni buzadi | Parse'dan oldin `try/catch` + sxema validatsiya, sinsa qayta so'ra |
| Demo wifi o'ladi | Lokal embedding model + senariylarni oldindan ingest qilib qo'y |
| LLM ma'lumot to'qiydi | `get_fact` qattiq cheklov — LLM raqam yaratolmaydi (3.1) |
| Gemini rate-limit | Demo'dan oldin javoblarni keshla yoki zaxira kalit tayyorla |
| Provider ishlamay qoldi | `llm.Provider` interfeysi — bitta qator bilan OpenAI'ga sakra |

---

## 11. Keyingi qadam

Birinchi kod — **Faza 1, Ingestion Pipeline**. U Markdown senariyni o'qib, faktlar/rubrikani ajratib, prose'ni embed qiladi. Bu hamma narsaning poydevori.

Tayyor bo'lsang, shu pipeline'ning Go kodini yozishdan boshlaymiz. 🪨
