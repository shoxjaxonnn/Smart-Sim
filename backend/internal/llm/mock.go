package llm

import (
	"context"
	"fmt"
	"strings"
)

// Mock — offline provider. No API key needed. Used as a fallback so the
// demo always works (spec demo risk: "Gemini rate-limit / wifi dies").
type Mock struct{}

func NewMock() *Mock { return &Mock{} }

func (m *Mock) Chat(ctx context.Context, req ChatRequest) (string, error) {
	msg := strings.ToLower(req.UserMessage)

	// Crude fact lookup: if the student asks about a known key, answer from
	// the store — never fabricate (mirrors the get_fact guarantee).
	for key, val := range req.Facts {
		short := key
		if i := strings.LastIndex(key, "."); i >= 0 {
			short = key[i+1:]
		}
		if strings.Contains(msg, short) || strings.Contains(msg, strings.ReplaceAll(key, ".", " ")) {
			return fmt.Sprintf("Tekshirdim — %s: %s", key, val), nil
		}
	}

	switch {
	case strings.Contains(msg, "salom") || strings.Contains(msg, "hello"):
		return "Salom! Men tizim yordamchisiman. Login sahifasidagi muammoni birga aniqlaymiz. Nimadan boshlaymiz?", nil
	case strings.Contains(msg, "log") || strings.Contains(msg, "error") || strings.Contains(msg, "xato"):
		return "Server xato logiga qarang. `get_fact(\"server.error_log\")` orqali aniq qatorni so'rashingiz mumkin. Nima ko'rdingiz?", nil
	case strings.Contains(msg, "sql") || strings.Contains(msg, "injection") || strings.Contains(msg, "injeksiya"):
		return "Yaxshi yo'nalish. Nega aynan SQL injection deb o'ylaysiz? Qaysi belgi shunga ishora qiladi?", nil
	default:
		return "[mock] Tushunarli. Davom et — muammoni qanday tekshirasan? (Aniq raqam kerak bo'lsa get_fact ishlat.)", nil
	}
}

func (m *Mock) Grade(ctx context.Context, modelAnswer, studentAnswer string, rubric []Criterion) (GradeResult, error) {
	res := GradeResult{}
	ans := strings.ToLower(studentAnswer)
	for _, c := range rubric {
		hits := 0
		for _, kw := range c.Keywords {
			if strings.Contains(ans, strings.ToLower(kw)) {
				hits++
			}
		}
		score := 0
		if len(c.Keywords) > 0 {
			score = c.Max * hits / len(c.Keywords)
		}
		just := "[mock] Kalit so'zlar topildi: " + fmt.Sprint(hits) + "/" + fmt.Sprint(len(c.Keywords))
		if hits == 0 {
			just = "[mock] Bu mezon bo'yicha kalit atamalar topilmadi."
		}
		res.Criteria = append(res.Criteria, CriterionScore{Name: c.Name, Score: score, Max: c.Max, Justification: just})
		res.TotalScore += score
		res.MaxScore += c.Max
	}
	return res, nil
}

func (m *Mock) GenerateScenario(ctx context.Context, req ScenarioDraftRequest) (ScenarioDraft, error) {
	title := req.Title
	if title == "" {
		title = "Shubhali tizim xatosi"
	}
	subject := req.Subject
	if subject == "" {
		subject = "IT / Web Security"
	}
	codeLang := req.CodeLanguage
	if codeLang == "" {
		codeLang = "python"
	}
	lc := strings.ToLower(req.LessonContext + " " + req.ProblemFocus + " " + req.TeacherInstruction + " " + req.DocumentText)
	draft := ScenarioDraft{
		Title:     title,
		Subject:   subject,
		Language:  req.Language,
		Situation: "Tizimda nosozlik kuzatildi. Talaba muammoni aniqlab, kodni tuzatishi kerak.",
		Facts: map[string]string{
			"server.error_log": "Error: unexpected token near '--' in query",
			"db.table":         "users",
			"login.field":      "username",
			"server.cpu":       "94%",
		},
		Rubric: []Criterion{
			{Name: "Muammoni aniqladi", Max: 3, Keywords: []string{"sql injection", "injeksiya"}},
			{Name: "Sababni tushuntirdi", Max: 4, Keywords: []string{"validatsiya", "sanitatsiya", "user input"}},
			{Name: "Yechim berdi", Max: 3, Keywords: []string{"prepared statement", "parametrlangan", "orm"}},
		},
		ModelAnswer:             "Bu SQL injection. Foydalanuvchi inputi so'rovga to'g'ridan-to'g'ri qo'shilgan. Yechim: parametrlangan so'rovlar ishlatish.",
		CodeLanguage:            codeLang,
		CodeChallengeAfterRound: 3,
		Hint:                    "Foydalanuvchi kiritmasini tekshiring va query string yig'ishni to'xtating.",
	}

	if strings.TrimSpace(req.DocumentText) != "" {
		draft.Situation = "Yuklangan dars materiali asosida muammo qayta tuzildi. Talaba endi uni chat orqali tahlil qiladi."
	}
	if strings.Contains(lc, "auth") || strings.Contains(lc, "login") || strings.Contains(lc, "sql") {
		draft.Situation = "Login sahifasida g'alati xatti-harakat kuzatildi. Foydalanuvchi kiritmasi noto'g'ri ishlov berilmoqda."
		draft.BuggyCode = "def login(username, password):\n    query = \"SELECT * FROM users WHERE name='\" + username + \"' AND pass='\" + password + \"'\"\n    return db.execute(query)\n"
		draft.Tests = "assert login('admin', 'pass') is not None\nassert login(\"admin' OR '1'='1\", 'x') is None\nassert login(\"'; DROP TABLE users;--\", 'x') is None\n"
	} else {
		draft.BuggyCode = "def solve(x):\n    return x + 1\n"
		draft.Tests = "assert solve(1) == 2\nassert solve(3) == 4\n"
	}
	return draft, nil
}
