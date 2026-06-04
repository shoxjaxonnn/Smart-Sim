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
