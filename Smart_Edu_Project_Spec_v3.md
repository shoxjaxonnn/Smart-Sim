# Smart Edu — AI Simulation Platform

> **Versiya:** MVP v3 (Hackathon Scope)
> **Fokus:** IT yo'nalishi (lekin arxitektura har qanday fan uchun ochiq)
> **Stack:** Go · Vue.js 3 · PostgreSQL + pgvector · Gemini Flash API · Docker

---

## 1. Loyiha haqida

**Smart Edu** — an'anaviy testlarni *situatsion AI simulyatsiyalar* bilan almashtiradigan interaktiv ta'lim platformasi. Talaba statik savollarga javob bermaydi — u boshqariladigan, AI yuritadigan muhitda real muammoni hal qiladi, xatoli kodni tuzatadi, va deterministik testlar orqali baholanadi.

### Asosiy g'oya: "Fan — bu ma'lumot, kod emas"

MVP **IT yo'nalishiga** qaratilgan (SQL injection, server xatolari, debugging va h.k.), lekin platforma shunday quriladiki, **istalgan fan** uni ishlatishi mumkin. Yangi fan qo'shish uchun kod yozilmaydi, faqat yangi **senariy** qo'shiladi.

### Goals (MVP)
- Teacher dars kontekstini (DOCX) yuklaydi → platforma markdown'ga aylantiradi.
- AI dars kontekstidan senariy draft yaratadi → Teacher tasdiqlaydi.
- Talaba AI bilan real vaqtda muloqot qilib muammoni yechadi.
- AI simulyatsiya davomida ataylab xatoli kod beradi → Talaba tuzatadi → Docker sandbox testlar tekshiradi.
- Tizim talaba javobini rubrika + test natijalari bo'yicha baholaydi.
- AI **hech qachon ma'lumot to'qimaydi** (get_fact mexanizmi bilan kafolatlangan).

### Non-Goals (MVP'da YO'Q)
- Medicina yo'nalishi (xavf yuqori — keyingi bosqichga qoldirildi).
- Foydalanuvchi autentifikatsiyasi (demo uchun soddalashtirilgan).
- PPT, PDF qo'llab-quvvatlash (faqat DOCX).
- Mobil ilova.
- Bir nechta til (Code Sandbox faqat Python).

---

## 2. Arxitektura

```
                    ┌──────────────────────┐
                    │     Vue.js 3 UI      │
                    │  Teacher  │  Talaba  │
                    └─────┬────┴────┬─────┘
                          │  HTTP   │
                    ┌─────▼─────────▼─────────────────────┐
                    │           Go Backend                  │
                    │                                       │
                    │  ┌─────────────────────────────────┐  │
                    │  │ 3 ta AI qatlam (alohida kalitlar)│  │
                    │  │                                   │  │
                    │  │ Q1: Hujjat tahlili               │  │
                    │  │     DOCX → Markdown (koddda)     │  │
                    │  │     + LLM tozalash                │  │
                    │  │                                   │  │
                    │  │ Q2: Talaba simulyatsiyasi         │  │
                    │  │     RAG + get_fact + chat         │  │
                    │  │                                   │  │
                    │  │ Q3: Teacher kontent generatsiya   │  │
                    │  │     Kontekst → senariy draft      │  │
                    │  └─────────────────────────────────┘  │
                    │                                       │
                    │  ┌──────────────┐  ┌──────────────┐  │
                    │  │ RAG Retriever│  │ Code Sandbox │  │
                    │  │ (pgvector)   │  │ (Docker)     │  │
                    │  └──────────────┘  └──────────────┘  │
                    └───────────┬───────────────────────────┘
                                │
              ┌─────────────────┼─────────────────┐
              ▼                 ▼                  ▼
      ┌──────────────┐  ┌─────────────┐  ┌────────────────┐
      │  Gemini API  │  │ PostgreSQL  │  │ Docker Engine  │
      │  (3 kalit)   │  │ + pgvector  │  │ (sandbox)      │
      └──────────────┘  └─────────────┘  └────────────────┘
```

### Texnologiya tanlovi
| Qatlam | Texnologiya | Sabab |
|---|---|---|
| Backend | **Go (Golang)** | Yuqori concurrency, tez API, oddiy deploy |
| Frontend | **Vue.js 3** | Reaktiv, tez, real-vaqt chat uchun qulay |
| Ma'lumotlar bazasi | **PostgreSQL + pgvector** | Semantik qidiruv + oddiy SQL bir joyda |
| LLM | **Gemini Flash API** | Bepul tier, arzon, hackathon uchun ideal |
| Embedding | **`all-MiniLM-L6-v2` (384 dim)** | Lokal, internetsiz ishlaydi |
| Code Sandbox | **Docker** | Izolyatsiyalangan kod ishga tushirish |
| Hujjat parser | **Go DOCX parser** | Determistik, LLM siz konvertatsiya |

---

## 3. Uchta AI qatlami

Har bir qatlam alohida Gemini API kaliti ishlatadi. Sabablari: rate limit izolyatsiyasi, xarajat kuzatuvi, xavfsizlik.

> ⚠️ **Barcha API kalitlar faqat Go backend'da saqlanadi (env variable). Vue frontend hech qachon API kalitni ko'rmaydi.**

### Q1: Hujjat tahlili (DOCX → Markdown → DB)

**Qachon ishlaydi:** Teacher DOCX fayl yuklaganda (bir martalik operatsiya).
**Yondashuv:** Determistik parser — Go kutubxonasi DOCX'ni parse qiladi (sarlavhalar, paragraflar, jadvallar, kod bloklari). LLM faqat natijani tozalash/strukturalash uchun ishlatiladi (ixtiyoriy).
**Nega LLM'ni asosiy parser sifatida ishlatmaymiz:** LLM ma'lumot tushirib qoldirishi yoki qo'shib qo'yishi mumkin — ta'lim kontekstida bu qabul qilinmas.
**Faqat DOCX:** Bitta format = bitta parser = ishonchli natija. PDF va PPT keyingi versiyaga qoldirildi.
**Token sarfi:** ~15,000-20,000 token (50 sahifa uchun), narxi ~$0.02. Bir martalik.

```
Teacher DOCX yuklaydi
  → Go DOCX parser: matnni ajratadi (heading, paragraph, table, code)
  → Markdown formatga aylantiradi
  → (Ixtiyoriy) LLM: markdownni tozalaydi/strukturalaydi
  → Markdown DB'ga yoziladi (xom dars materiali)
```

### Q2: Talaba simulyatsiyasi (Chat + RAG + get_fact)

**Qachon ishlaydi:** Talaba simulyatsiya qilganda (eng ko'p ishlatiladi).
**Xarakteristika:** Barqarorlik eng muhim. Tez javob (< 2 soniya). Implicit kesh avtomatik ishlaydi.
**Token sarfi:** 8 raundli sessiya ~$0.018 (keshsiz), ~$0.008 (kesh bilan).
**Function calling:** `get_fact` vositasi har raundda 20-30% qo'shimcha token sarflaydi — hisob-kitobda buni hisobga oling.
**Sessiya chegarasi:** Raund soni emas, token budjet bilan cheklangan (maks 15,000 jami input token).

```
Talaba savol yozadi
  → Savol embed qilinadi → pgvector'da o'xshash chunk qidiriladi (RAG)
  → System prompt + RAG kontekst + tarix + savol → LLM'ga yuboriladi
  → LLM kerak bo'lsa get_fact(key) chaqiradi → faqat Facts Store'dan o'qiydi
  → Javob talabaga qaytadi, tarixga yoziladi
  → Belgilangan raundda buggy kod ko'rsatiladi → Code Sandbox'ga o'tiladi
```

### Q3: Teacher kontent generatsiya (Kontekst → Senariy draft)

**Qachon ishlaydi:** Teacher "Senariy yaratish" tugmasini bosganda (kamdan-kam).
**Xarakteristika:** Sifat muhim, tezlik emas. Teacher 10-15 soniya kutishi normal.
**Nima generatsiya qiladi:** Vaziyatli masala (senariy prose), faktlar to'plami (kalit/qiymat), rubrika (mezonlar + og'irliklar), buggy kod, va test kodlari.
**Muhim:** AI yaratgan hamma narsa `draft` statusida saqlanadi. Teacher ko'rib chiqadi, tahrirlaydi, va **faqat tasdiqlangandan keyin** talabalar uchun ochiladi.
**Token sarfi:** ~2,000-3,000 token per generatsiya, narxi ~$0.01. Bir martalik.

```
Teacher "Senariy yaratish" bosadi
  → Dars konteksti (markdown) + maxsus prompt → LLM
  → LLM strukturali JSON qaytaradi:
    { prose, facts, rubric, buggy_code, tests }
  → Backend parse qiladi → DB'ga yozadi (status: draft)
  → Teacher ko'radi → tahrirlaydi → "Tasdiqlash" bosadi
  → Status: approved
  → RAG pipeline ishga tushadi (chunkla → embed → pgvector)
  → Senariy talabalar uchun ochiladi
```

---

## 4. Code Sandbox (Docker izolyatsiya)

### Nima uchun kerak?
Talaba kodni yozadi va yuboradi. Bu kodni serverda ishga tushirish **xavfli** — cheksiz sikl, fayl o'chirish, tarmoq hujumi. Docker konteyner bu xavflarni butunlay bartaraf qiladi.

### Konteyner cheklovlari
| Cheklov | Qiymat | Sabab |
|---|---|---|
| Vaqt (timeout) | **5 soniya** | Cheksiz sikl oldini olish |
| Xotira | **64 MB** | Xotira to'ldirish oldini olish |
| Tarmoq | **O'chirilgan** (`--network=none`) | Tashqariga chiqish taqiqlangan |
| Fayl tizimi | **Faqat o'qish** (`--read-only`) | Tizim fayllarini himoya |
| Yozish | **Faqat /tmp, 10MB** | Minimal yozish imkoniyati |
| Protsesslar | **Maks 32** (`--pids-limit=32`) | Fork bomb oldini olish |
| Foydalanuvchi | **nobody** (root emas) | Root huquqlari yo'q |

### Qo'llab-quvvatlanadigan til
MVP'da faqat **Python 3.11**. Sabablari: IT talabalarining ko'pchiligi biladi, test yozish oson (`assert`), Docker image kichik (~50MB alpine).

### Docker image
Oldindan qurilgan base image: `smartedu-sandbox:python`. Ichida: Python 3.11, standart kutubxona. Boshqa hech narsa — `import requests` ishlamaydi. Har safar yangi image qurilmaydi — faqat talaba kodi mount qilinadi.

### Test ishga tushirish modeli
Sodda yondashuv: talaba kodi va testlar bitta faylga birlashtirilib ishga tushiriladi.

```
# === TALABA KODI ===
def login(username, password):
    query = f"SELECT * FROM users WHERE name='{username}' AND pass='{password}'"
    return db.execute(query)

# === TESTLAR (teacher/AI yaratgan, talaba ko'rmaydi) ===
assert login("admin", "pass") != None, "Oddiy login ishlashi kerak"
assert login("admin' OR '1'='1", "x") == None, "SQL injection oldini olinmagan!"
assert login("'; DROP TABLE users;--", "x") == None, "Xavfli kiritma qabul qilindi!"
```

### Natija holatlari
| Holat | Rang | Talabaga xabar |
|---|---|---|
| Barcha testlar o'tdi | 🟢 Yashil | "Barcha testlar muvaffaqiyatli!" + ball |
| Ba'zi testlar sindi | 🔴 Qizil | Qaysi test singani + xato xabari (javobsiz) |
| Sintaksis xatosi | 🟡 Sariq | "SyntaxError: line 3" |
| Vaqt tugadi | ⚪ Kulrang | "Kod 5 soniyadan oshdi. Cheksiz sikl?" |

### Simulyatsiya ichida qachon paydo bo'ladi?
**Oldindan belgilangan:** Senariy faylida `code_challenge_after_round: 4` yoziladi. 4-raunddan keyin avtomatik buggy kod ko'rsatiladi. AI qaror qilmaydi — deterministik va demo uchun ishonchli.

### Baholash ulanishi
Rubrikada alohida mezon: `"Kodni to'g'ri tuzatdi"` — maks 3 ball. Bu mezonni LLM baholamaydi — backend kodi deterministik ball beradi:
- Barcha testlar o'tdi → 3 ball
- Qisman o'tdi → 1-2 ball (o'tgan testlar soniga qarab)
- Hech biri o'tmadi → 0 ball

---

## 5. Senariy fayl formati

Bu fayl barcha qatlamlarni bog'laydi. Teacher panel orqali yaratiladi (AI yordamida) yoki qo'lda yoziladi.

```yaml
---
id: "sql-injection-001"
title: "Shubhali login formasi"
subject: "IT / Web Security"
language: "uz"
status: "draft"                    # draft → approved

# Code challenge sozlamalari
code_challenge_after_round: 4      # 4-raunddan keyin buggy kod chiqadi
code_language: "python"

# Anti-hallucination: Facts Store (Q2 — get_fact faqat shu yerdan o'qiydi)
facts:
  server.error_log: "Error: unexpected token near '--' in query"
  db.table: "users"
  login.field: "username"
  server.cpu: "Bu ma'lumot mavjud emas"

# Baholash rubrikasi
rubric:
  - name: "Hujum turini aniqlash"
    max: 3
    keywords: ["SQL injection", "injeksiya"]
  - name: "Sababni tushuntirish"
    max: 4
    keywords: ["validatsiya", "sanitatsiya", "user input"]
  - name: "Kodni to'g'ri tuzatdi"
    max: 3
    type: "code_test"              # LLM emas, test natijasi belgilaydi
    keywords: []

model_answer: >
  Bu SQL injection hujumi. Login formasi foydalanuvchi kiritmasini
  to'g'ridan-to'g'ri so'rovga qo'shgani uchun yuzaga keladi. Yechim —
  parametrlangan so'rovlar (prepared statements) ishlatish.

# Code Challenge
code_challenge:
  buggy_code: |
    def login(username, password):
        query = "SELECT * FROM users WHERE name='" + username + "' AND pass='" + password + "'"
        return db.execute(query)
  hint: "Foydalanuvchi kiritmasini tekshiring"
  tests: |
    assert login("admin", "pass") != None, "Oddiy login ishlashi kerak"
    assert login("admin' OR '1'='1", "x") == None, "SQL injection oldini olinmagan!"
    assert login("'; DROP TABLE users;--", "x") == None, "Xavfli kiritma qabul qilindi!"
---

## Vaziyat

Sen DevOps muhandisisan. Tungi 02:00 da `users` jadvaliga g'alati
so'rovlar tushayotgani haqida ogohlantirish keldi. Login sahifasida
nimadir noto'g'ri. Muammoni aniqla va hal qil.
```

---

## 6. LLM Service — Go interfeys skeleti

Barcha provayderlar shu interfeysni implement qiladi. Almashtirish = bitta qatorni o'zgartirish.

```go
package llm

import "context"

// FactsStore — senariy faktlar ombori
type FactsStore map[string]string

func (fs FactsStore) Get(key string) Fact {
    v, ok := fs[key]
    if !ok {
        return Fact{Key: key, Value: "That information is currently unavailable.", Found: false}
    }
    return Fact{Key: key, Value: v, Found: true}
}

type Fact struct {
    Key   string
    Value string
    Found bool
}

// Provider — har bir LLM (Gemini, OpenAI...) shuni implement qiladi
type Provider interface {
    Chat(ctx context.Context, req ChatRequest) (string, error)
    Grade(ctx context.Context, modelAnswer, studentAnswer string, rubric []Criterion) (GradeResult, error)
    GenerateScenario(ctx context.Context, lessonContext string) (ScenarioDraft, error)
}

type ChatRequest struct {
    SystemPrompt string
    History      []Message
    UserMessage  string
    Facts        FactsStore
}

type Message struct {
    Role    string // "user" yoki "assistant"
    Content string
}

// Baholash
type GradeResult struct {
    TotalScore int              `json:"total_score"`
    MaxScore   int              `json:"max_score"`
    Criteria   []CriterionScore `json:"criteria"`
}

type CriterionScore struct {
    Name          string `json:"name"`
    Score         int    `json:"score"`
    Max           int    `json:"max"`
    Justification string `json:"justification"`
}

type Criterion struct {
    Name     string
    Max      int
    Keywords []string
    Type     string // "" (default: LLM) yoki "code_test" (deterministik)
}

// Senariy generatsiya (Q3 — teacher uchun)
type ScenarioDraft struct {
    Title         string            `json:"title"`
    Prose         string            `json:"prose"`
    Facts         map[string]string `json:"facts"`
    Rubric        []Criterion       `json:"rubric"`
    BuggyCode     string            `json:"buggy_code"`
    Hint          string            `json:"hint"`
    Tests         string            `json:"tests"`
    ModelAnswer   string            `json:"model_answer"`
}
```

### Uchta provider instansiyasi

```go
// main.go — uchta alohida kalit
var (
    docProvider      = llm.NewGemini(os.Getenv("GEMINI_KEY_DOC"))       // Q1
    studentProvider  = llm.NewGemini(os.Getenv("GEMINI_KEY_STUDENT"))   // Q2
    teacherProvider  = llm.NewGemini(os.Getenv("GEMINI_KEY_TEACHER"))   // Q3
)
```

---

## 7. Ma'lumotlar bazasi sxemasi

```sql
CREATE EXTENSION IF NOT EXISTS vector;

-- O'qituvchilar
CREATE TABLE teachers (
    id          SERIAL PRIMARY KEY,
    name        TEXT NOT NULL,
    created_at  TIMESTAMPTZ DEFAULT now()
);

-- Dars materiallari (DOCX → Markdown)
CREATE TABLE lesson_materials (
    id          SERIAL PRIMARY KEY,
    teacher_id  INTEGER REFERENCES teachers(id),
    title       TEXT NOT NULL,
    markdown    TEXT NOT NULL,           -- DOCX'dan parse qilingan markdown
    created_at  TIMESTAMPTZ DEFAULT now()
);

-- Senariylar
CREATE TABLE scenarios (
    id                       TEXT PRIMARY KEY,
    teacher_id               INTEGER REFERENCES teachers(id),
    lesson_material_id       INTEGER REFERENCES lesson_materials(id),
    title                    TEXT NOT NULL,
    subject                  TEXT NOT NULL,
    status                   TEXT NOT NULL DEFAULT 'draft',  -- draft | approved
    prose                    TEXT NOT NULL,
    model_answer             TEXT NOT NULL,
    facts                    JSONB NOT NULL,
    rubric                   JSONB NOT NULL,
    code_challenge           JSONB,           -- buggy_code, hint, tests
    code_challenge_after_round INTEGER DEFAULT 4,
    code_language            TEXT DEFAULT 'python',
    created_at               TIMESTAMPTZ DEFAULT now()
);

-- RAG uchun chunklar (faqat approved senariylar uchun)
CREATE TABLE scenario_chunks (
    id          SERIAL PRIMARY KEY,
    scenario_id TEXT REFERENCES scenarios(id) ON DELETE CASCADE,
    content     TEXT NOT NULL,
    embedding   vector(384)
);

CREATE INDEX ON scenario_chunks USING ivfflat (embedding vector_cosine_ops);

-- Talaba sessiyalari
CREATE TABLE sessions (
    id               SERIAL PRIMARY KEY,
    scenario_id      TEXT REFERENCES scenarios(id),
    history          JSONB,
    total_tokens     INTEGER DEFAULT 0,     -- token budjet kuzatuvi
    code_submitted   TEXT,                  -- talaba yuborgan kod
    code_result      JSONB,                 -- {passed: true/false, details: [...]}
    grade            JSONB,                 -- GradeResult (tugagach yoziladi)
    created_at       TIMESTAMPTZ DEFAULT now()
);
```

---

## 8. API endpointlari

### Teacher API
| Method | Endpoint | Vazifa |
|---|---|---|
| `POST` | `/materials/upload` | DOCX yuklash → markdown'ga aylantirish |
| `GET` | `/materials` | Barcha dars materiallari |
| `POST` | `/scenarios/generate` | AI senariy draft yaratadi (Q3) |
| `GET` | `/scenarios` | Barcha senariylar (draft + approved) |
| `PUT` | `/scenarios/:id` | Senariyni tahrirlash |
| `PATCH` | `/scenarios/:id/approve` | Tasdiqlash (draft → approved + RAG) |
| `DELETE` | `/scenarios/:id` | Senariyni o'chirish |
| `GET` | `/scenarios/:id/results` | Talaba natijalari |

### Talaba API
| Method | Endpoint | Vazifa |
|---|---|---|
| `GET` | `/scenarios/available` | Faqat approved senariylar |
| `POST` | `/chat` | Simulyatsiya muloqoti (Q2) |
| `POST` | `/sandbox/submit` | Tuzatilgan kodni yuborish |
| `POST` | `/grade` | Sessiyani yakunlab baholash |

### Ichki (frontend chaqirmaydi)
| Method | Endpoint | Vazifa |
|---|---|---|
| — | `sandbox.Run()` | Docker'da kod ishga tushirish (ichki funksiya) |

---

## 9. Asosiy ish oqimlari

### Oqim A: Teacher kontent yaratadi
```
1. Teacher DOCX yuklaydi → POST /materials/upload
2. Go DOCX parser → markdown → DB'ga yoziladi
3. Teacher "Senariy yaratish" bosadi → POST /scenarios/generate
4. Q3 (AI): dars konteksti → senariy draft (prose + facts + rubric + code)
5. Draft DB'ga yoziladi (status: draft)
6. Teacher ko'radi, tahrirlaydi → PUT /scenarios/:id
7. Teacher "Tasdiqlash" bosadi → PATCH /scenarios/:id/approve
8. Status: approved → RAG pipeline: prose chunklanadi → embed → pgvector
```

### Oqim B: Talaba simulyatsiya qiladi
```
1. Talaba senariy tanlaydi (faqat approved) → GET /scenarios/available
2. Sessiya boshlanadi → POST /chat (1-raund)
3. Har raund:
   a. Talaba savol → embed → pgvector qidiruv (RAG)
   b. System prompt + RAG + tarix + savol → Q2 (Gemini)
   c. LLM kerak bo'lsa get_fact() chaqiradi → Facts Store'dan
   d. Javob → talabaga qaytadi
   e. Token counter yangilanadi
4. [code_challenge_after_round] raundda:
   a. Buggy kod talabaga ko'rsatiladi
   b. Talaba tuzatadi → POST /sandbox/submit
   c. Docker konteyner: talaba kodi + testlar → ishga tushadi
   d. Natija: pass/fail → talabaga ko'rsatiladi
   e. Agar fail → talaba qayta tuzatishi mumkin (1 marta)
5. Talaba "Yakunlash" bosadi → POST /grade
   a. Q2 (AI): rubrika bo'yicha baholash (code_test mezonidan tashqari)
   b. Code test mezoni: Docker natijasidan deterministik ball
   c. Umumiy ball → talabaga ko'rsatiladi
```

---

## 10. Token hisob-kitobi va narx modeli

### Bitta sessiya (8 raund + grading + code sandbox)
| Komponent | Input token | Output token |
|---|---|---|
| 8 raund chat (to'planuvchi) | ~19,000 | ~2,000 |
| get_fact qo'shimchasi (+25%) | ~4,750 | — |
| Baholash (grading) | ~800 | ~150 |
| **Jami (keshsiz)** | **~24,550** | **~2,150** |
| **Jami (implicit kesh bilan)** | **~12,000** | **~2,150** |

> Code Sandbox token sarflamaydi — u Docker'da ishlaydi, LLM emas.

### Narx (Gemini Flash)
| Stsenariy | Sessiya narxi | 1000 talaba/yil |
|---|---|---|
| Keshsiz | ~$0.047 | ~$17,155 |
| Implicit kesh bilan (~50% tejash) | ~$0.031 | ~$11,315 |

### Uchta qatlamning yillik narxi (1000 talaba)
| Qatlam | Narx/yil | Izoh |
|---|---|---|
| Q1: Hujjat tahlili | ~$50 | Bir martalik, 100 ta dars |
| Q2: Talaba simulyatsiya | ~$11,000 | Eng katta xarajat |
| Q3: Teacher generatsiya | ~$30 | 300 ta senariy |
| Docker infra | ~$500 | Server resurslari |
| **Jami** | **~$11,580** | |

Litsenziya narxi $40,000 bo'lsa → **marja 71%**.

---

## 11. Embedding modeli

| Variant | O'lcham | Tavsiya |
|---|---|---|
| **`all-MiniLM-L6-v2` (lokal)** | 384 | ✅ Demo xavfsizligi uchun |
| Gemini `text-embedding-004` | 768 | Wifi mustahkam bo'lsa |

> ⚠️ `pgvector` ustun o'lchamini birinchi kuni o'rnat va tegma: `embedding vector(384)`

Python sidecar (sentence-transformers) — Go'dan HTTP orqali chaqiriladi.

---

## 12. Demo xavflari va oldini olish

| Xavf | Oldini olish |
|---|---|
| Gemini JSON buziladi | `try/catch` + sxema validatsiya, sinsa qayta so'ra |
| Demo wifi o'ladi | Lokal embedding + senariylarni oldindan ingest |
| LLM ma'lumot to'qiydi | `get_fact` qattiq cheklov — LLM raqam yaratOLMAYDI |
| Gemini rate-limit | Alohida kalitlar + zaxira kalit |
| Provider ishlamay qoldi | `llm.Provider` interfeysi — bitta qator bilan almashtirish |
| Docker konteyner qotib qoldi | 5 soniya timeout + `--rm` (avtomatik o'chirish) |
| Talaba xavfli kod yozdi | Docker: no-network, read-only, 64MB, nobody user |
| DOCX parse xatosi | Determistik parser, LLM'ga bog'liq emas |

---

## 13. Implementatsiya bosqichlari (Hackathon)

### Faza 0 — Setup
- [ ] PostgreSQL + pgvector ko'tarish, sxemani yaratish
- [ ] Gemini API kalitlarini olish (3 ta — doc, student, teacher)
- [ ] Docker o'rnatish + sandbox base image qurish
- [ ] Embedding model (Python sidecar) ko'tarish

### Faza 1 — DOCX Pipeline + Teacher API
- [ ] Go DOCX parser: DOCX → Markdown konvertatsiya
- [ ] `POST /materials/upload` endpoint
- [ ] `POST /scenarios/generate` — Q3 orqali AI draft generatsiya
- [ ] `PATCH /scenarios/:id/approve` — tasdiqlash + RAG pipeline

### Faza 2 — Simulation API
- [ ] `llm.Provider` interfeysi + Gemini implementatsiyasi
- [ ] `get_fact` function calling integratsiyasi
- [ ] `POST /chat` — RAG + persona + token tracking
- [ ] Sessiya token budjet chegarasi (15,000)

### Faza 3 — Code Sandbox
- [ ] Docker sandbox: konteyner yaratish, cheklovlar, timeout
- [ ] `POST /sandbox/submit` — talaba kodi + testlar → Docker → natija
- [ ] Natijani umumiy ballga ulash (code_test mezoni)

### Faza 4 — Frontend (Vue 3)
- [ ] Talaba: Chat interfeysi + kod muharriri + natija ko'rsatish
- [ ] Teacher: Minimal panel (DOCX yuklash + senariy generatsiya + tasdiqlash)
- [ ] Baholash: rubrika + test natijalari ko'rinishi

### Faza 5 — Demo tayyorgarlik
- [ ] Senariylarni oldindan yaratib, ingest qilib qo'yish
- [ ] Demo skriptini mashq qilish (80 soniya ichida)
- [ ] Zaxira rejani tekshirish (wifi o'lsa, Gemini ishlamasa)
