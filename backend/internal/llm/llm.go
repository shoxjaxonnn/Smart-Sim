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
	Role    string `json:"role"`    // "user" | "assistant"
	Content string `json:"content"`
}

// ChatRequest — one simulation step.
type ChatRequest struct {
	SystemPrompt string
	History      []Message
	UserMessage  string
	Facts        FactsStore
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

// Provider — every LLM (Gemini, OpenAI...) implements this.
type Provider interface {
	// Chat — converse with student. Wires up the get_fact tool automatically.
	Chat(ctx context.Context, req ChatRequest) (string, error)

	// Grade — grade student answer against rubric, return structured JSON.
	Grade(ctx context.Context, modelAnswer, studentAnswer string, rubric []Criterion) (GradeResult, error)
}
