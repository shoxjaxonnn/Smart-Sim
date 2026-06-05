package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"smartedu/internal/llm"
	"smartedu/internal/scenario"
)

type Document struct {
	ID                 string    `json:"id"`
	FileName           string    `json:"file_name"`
	ContentType        string    `json:"content_type"`
	ParsedText         string    `json:"parsed_text"`
	TeacherInstruction string    `json:"teacher_instruction"`
	Title              string    `json:"title"`
	Subject            string    `json:"subject"`
	Language           string    `json:"language"`
	CodeLanguage       string    `json:"code_language"`
	ProblemFocus       string    `json:"problem_focus"`
	ScenarioID         string    `json:"scenario_id"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type Session struct {
	ID         string    `json:"id"`
	ScenarioID string    `json:"scenario_id"`
	CreatedAt  time.Time `json:"created_at"`
}

type Message struct {
	ID        int64     `json:"id"`
	SessionID string    `json:"session_id"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type Repository interface {
	Init(ctx context.Context) error
	SeedScenarios(ctx context.Context, dir string) error

	ListScenarios(ctx context.Context, includeDrafts bool) ([]scenario.Scenario, error)
	GetScenario(ctx context.Context, id string) (*scenario.Scenario, error)
	UpsertScenario(ctx context.Context, sc *scenario.Scenario) error
	ApproveScenario(ctx context.Context, id string) (*scenario.Scenario, error)

	ListDocuments(ctx context.Context) ([]Document, error)
	CreateDocument(ctx context.Context, doc *Document) error
	GetDocument(ctx context.Context, id string) (*Document, error)
	UpdateDocumentScenario(ctx context.Context, docID, scenarioID string) error

	CreateSession(ctx context.Context, sess *Session) error
	GetSession(ctx context.Context, id string) (*Session, error)
	ListMessages(ctx context.Context, sessionID string) ([]Message, error)
	AppendMessage(ctx context.Context, msg *Message) error
}

func Open(ctx context.Context, dsn string) (Repository, error) {
	if strings.TrimSpace(dsn) == "" {
		return NewMemory(), nil
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &Postgres{db: db}, nil
}

func seedScenarioID(path string) string {
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	base = strings.ToLower(strings.TrimSpace(base))
	base = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return r
		case r >= '0' && r <= '9':
			return r
		case r == '-' || r == '_':
			return r
		default:
			return '-'
		}
	}, base)
	base = strings.Trim(base, "-")
	if base == "" {
		return "scenario"
	}
	return base
}

func cloneScenario(sc *scenario.Scenario) scenario.Scenario {
	if sc == nil {
		return scenario.Scenario{}
	}
	out := *sc
	if sc.Facts != nil {
		out.Facts = make(llm.FactsStore, len(sc.Facts))
		for k, v := range sc.Facts {
			out.Facts[k] = v
		}
	}
	if sc.Rubric != nil {
		out.Rubric = append([]llm.Criterion(nil), sc.Rubric...)
	}
	return out
}

func encodeJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

type scenarioRow struct {
	ID                      string
	Title                   string
	Subject                 string
	Language                string
	Status                  string
	CodeChallengeAfterRound int
	CodeLanguage            string
	Facts                   []byte
	Rubric                  []byte
	ModelAnswer             string
	CodeChallenge           []byte
	Situation               string
}

func decodeScenarioRow(row scenarioRow) (*scenario.Scenario, error) {
	out := &scenario.Scenario{
		ID:                      row.ID,
		Title:                   row.Title,
		Subject:                 row.Subject,
		Language:                row.Language,
		Status:                  row.Status,
		CodeChallengeAfterRound: row.CodeChallengeAfterRound,
		CodeLanguage:            row.CodeLanguage,
		ModelAnswer:             row.ModelAnswer,
		Situation:               row.Situation,
	}
	if len(row.Facts) > 0 {
		if err := json.Unmarshal(row.Facts, &out.Facts); err != nil {
			return nil, err
		}
	}
	if len(row.Rubric) > 0 {
		if err := json.Unmarshal(row.Rubric, &out.Rubric); err != nil {
			return nil, err
		}
	}
	if len(row.CodeChallenge) > 0 {
		if err := json.Unmarshal(row.CodeChallenge, &out.CodeChallenge); err != nil {
			return nil, err
		}
	}
	if out.Status == "" {
		out.Status = "approved"
	}
	return out, nil
}

func scenarioToRow(sc *scenario.Scenario) map[string]any {
	return map[string]any{
		"id":                         sc.ID,
		"title":                      sc.Title,
		"subject":                    sc.Subject,
		"language":                   sc.Language,
		"status":                     sc.Status,
		"code_challenge_after_round": sc.CodeChallengeAfterRound,
		"code_language":              sc.CodeLanguage,
		"facts":                      encodeJSON(sc.Facts),
		"rubric":                     encodeJSON(sc.Rubric),
		"model_answer":               sc.ModelAnswer,
		"code_challenge":             encodeJSON(sc.CodeChallenge),
		"situation":                  sc.Situation,
	}
}

func ensureAbs(dsn string) string {
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(dsn)), "postgres://") ||
		strings.HasPrefix(strings.ToLower(strings.TrimSpace(dsn)), "postgresql://") {
		return dsn
	}
	return dsn
}

func isApprovedScenario(sc *scenario.Scenario) bool {
	if sc == nil {
		return false
	}
	status := strings.ToLower(strings.TrimSpace(sc.Status))
	return status == "" || status == "approved"
}
