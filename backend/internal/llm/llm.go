package llm

import "context"

// Fact — result returned by the get_fact tool.
type Fact struct {
	Key   string
	Value string
	Found bool
}

// FactsStore — per-scenario facts store (spec 3.1).
// The LLM may ONLY read values through Get; it never invents numbers.
type FactsStore map[string]string

// Get returns the fact for key. Missing key => hard-coded "unavailable"
// guarantee produced by code, not by the model.
func (fs FactsStore) Get(key string) Fact {
	v, ok := fs[key]
	if !ok {
		return Fact{Key: key, Value: "That information is currently unavailable.", Found: false}
	}
	return Fact{Key: key, Value: v, Found: true}
}

// Message — one chat turn.
type Message struct {
	Role    string `json:"role"` // "user" | "assistant"
	Content string `json:"content"`
}

// ChatRequest — one simulation step.
type ChatRequest struct {
	SystemPrompt string
	History      []Message
	UserMessage  string
	Facts        FactsStore
}

// ScenarioDraftRequest — input used to generate a teacher draft.
type ScenarioDraftRequest struct {
	Title              string `json:"title"`
	Subject            string `json:"subject"`
	Language           string `json:"language"`
	LessonContext      string `json:"lesson_context"`
	ProblemFocus       string `json:"problem_focus"`
	CodeLanguage       string `json:"code_language"`
	SourceDocumentName string `json:"source_document_name"`
	TeacherInstruction string `json:"teacher_instruction"`
	DocumentText       string `json:"document_text"`
}

// Criterion — one rubric line.
type Criterion struct {
	Name     string   `json:"name"`
	Max      int      `json:"max"`
	Keywords []string `json:"keywords"`
}

// CriterionScore — graded criterion.
type CriterionScore struct {
	Name          string `json:"name"`
	Score         int    `json:"score"`
	Max           int    `json:"max"`
	Justification string `json:"justification"`
}

// GradeResult — rubric grading output (spec 3.2).
type GradeResult struct {
	TotalScore int              `json:"total_score"`
	MaxScore   int              `json:"max_score"`
	Criteria   []CriterionScore `json:"criteria"`
}

// ScenarioDraft — structured output for teacher scenario generation.
type ScenarioDraft struct {
	ID                      string            `json:"id"`
	Title                   string            `json:"title"`
	Subject                 string            `json:"subject"`
	Language                string            `json:"language"`
	Situation               string            `json:"situation"`
	Facts                   map[string]string `json:"facts"`
	Rubric                  []Criterion       `json:"rubric"`
	ModelAnswer             string            `json:"model_answer"`
	BuggyCode               string            `json:"buggy_code"`
	Hint                    string            `json:"hint"`
	Tests                   string            `json:"tests"`
	CodeChallengeAfterRound int               `json:"code_challenge_after_round"`
	CodeLanguage            string            `json:"code_language"`
}

// Provider — every LLM (Gemini, OpenAI...) implements this.
type Provider interface {
	// Chat — converse with student. Wires up the get_fact tool automatically.
	Chat(ctx context.Context, req ChatRequest) (string, error)

	// Grade — grade student answer against rubric, return structured JSON.
	Grade(ctx context.Context, modelAnswer, studentAnswer string, rubric []Criterion) (GradeResult, error)

	// GenerateScenario — produce a teacher draft from lesson context.
	GenerateScenario(ctx context.Context, req ScenarioDraftRequest) (ScenarioDraft, error)
}
