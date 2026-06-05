package store

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"smartedu/internal/scenario"
)

type Postgres struct {
	db *sql.DB
}

func (p *Postgres) Init(ctx context.Context) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS scenarios (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			subject TEXT NOT NULL,
			language TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'approved',
			code_challenge_after_round INTEGER NOT NULL DEFAULT 0,
			code_language TEXT NOT NULL DEFAULT 'python',
			facts JSONB NOT NULL DEFAULT '{}'::jsonb,
			rubric JSONB NOT NULL DEFAULT '[]'::jsonb,
			model_answer TEXT NOT NULL DEFAULT '',
			code_challenge JSONB NOT NULL DEFAULT '{}'::jsonb,
			situation TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE TABLE IF NOT EXISTS documents (
			id TEXT PRIMARY KEY,
			file_name TEXT NOT NULL,
			content_type TEXT NOT NULL,
			parsed_text TEXT NOT NULL DEFAULT '',
			teacher_instruction TEXT NOT NULL DEFAULT '',
			title TEXT NOT NULL DEFAULT '',
			subject TEXT NOT NULL DEFAULT '',
			language TEXT NOT NULL DEFAULT '',
			code_language TEXT NOT NULL DEFAULT 'python',
			problem_focus TEXT NOT NULL DEFAULT '',
			scenario_id TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS sessions (
			id TEXT PRIMARY KEY,
			scenario_id TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS messages (
			id BIGSERIAL PRIMARY KEY,
			session_id TEXT NOT NULL,
			role TEXT NOT NULL,
			content TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
	}
	for _, stmt := range stmts {
		if _, err := p.db.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}

func (p *Postgres) SeedScenarios(ctx context.Context, dir string) error {
	var count int
	if err := p.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM scenarios`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	matches, err := filepath.Glob(filepath.Join(dir, "*.md"))
	if err != nil {
		return err
	}
	for _, path := range matches {
		sc, err := scenario.Load(path)
		if err != nil {
			continue
		}
		if strings.TrimSpace(sc.ID) == "" {
			sc.ID = seedScenarioID(path)
		}
		if err := p.UpsertScenario(ctx, sc); err != nil {
			return err
		}
	}
	return nil
}

func (p *Postgres) ListScenarios(ctx context.Context, includeDrafts bool) ([]scenario.Scenario, error) {
	q := `SELECT id, title, subject, language, status, code_challenge_after_round, code_language, facts, rubric, model_answer, code_challenge, situation
	      FROM scenarios`
	if !includeDrafts {
		q += ` WHERE COALESCE(NULLIF(TRIM(status), ''), 'approved') = 'approved'`
	}
	q += ` ORDER BY status, title`
	rows, err := p.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []scenario.Scenario{}
	for rows.Next() {
		var row scenarioRow
		if err := rows.Scan(&row.ID, &row.Title, &row.Subject, &row.Language, &row.Status, &row.CodeChallengeAfterRound, &row.CodeLanguage, &row.Facts, &row.Rubric, &row.ModelAnswer, &row.CodeChallenge, &row.Situation); err != nil {
			return nil, err
		}
		sc, err := decodeScenarioRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, *sc)
	}
	return out, rows.Err()
}

func (p *Postgres) GetScenario(ctx context.Context, id string) (*scenario.Scenario, error) {
	var row scenarioRow
	err := p.db.QueryRowContext(ctx, `SELECT id, title, subject, language, status, code_challenge_after_round, code_language, facts, rubric, model_answer, code_challenge, situation
		FROM scenarios WHERE id = $1`, id).Scan(&row.ID, &row.Title, &row.Subject, &row.Language, &row.Status, &row.CodeChallengeAfterRound, &row.CodeLanguage, &row.Facts, &row.Rubric, &row.ModelAnswer, &row.CodeChallenge, &row.Situation)
	if err != nil {
		return nil, err
	}
	return decodeScenarioRow(row)
}

func (p *Postgres) UpsertScenario(ctx context.Context, sc *scenario.Scenario) error {
	if sc == nil {
		return fmt.Errorf("scenario required")
	}
	if strings.TrimSpace(sc.ID) == "" {
		sc.ID = seedScenarioID(sc.Title)
	}
	if sc.Status == "" {
		sc.Status = "approved"
	}
	row := scenarioToRow(sc)
	_, err := p.db.ExecContext(ctx, `
		INSERT INTO scenarios (id, title, subject, language, status, code_challenge_after_round, code_language, facts, rubric, model_answer, code_challenge, situation)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		ON CONFLICT (id) DO UPDATE SET
			title = EXCLUDED.title,
			subject = EXCLUDED.subject,
			language = EXCLUDED.language,
			status = EXCLUDED.status,
			code_challenge_after_round = EXCLUDED.code_challenge_after_round,
			code_language = EXCLUDED.code_language,
			facts = EXCLUDED.facts,
			rubric = EXCLUDED.rubric,
			model_answer = EXCLUDED.model_answer,
			code_challenge = EXCLUDED.code_challenge,
			situation = EXCLUDED.situation
	`,
		row["id"], row["title"], row["subject"], row["language"], row["status"], row["code_challenge_after_round"], row["code_language"], row["facts"], row["rubric"], row["model_answer"], row["code_challenge"], row["situation"],
	)
	return err
}

func (p *Postgres) ApproveScenario(ctx context.Context, id string) (*scenario.Scenario, error) {
	if _, err := p.db.ExecContext(ctx, `UPDATE scenarios SET status = 'approved' WHERE id = $1`, id); err != nil {
		return nil, err
	}
	return p.GetScenario(ctx, id)
}

func (p *Postgres) ListDocuments(ctx context.Context) ([]Document, error) {
	rows, err := p.db.QueryContext(ctx, `SELECT id, file_name, content_type, parsed_text, teacher_instruction, title, subject, language, code_language, problem_focus, scenario_id, created_at, updated_at FROM documents ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Document{}
	for rows.Next() {
		var d Document
		if err := rows.Scan(&d.ID, &d.FileName, &d.ContentType, &d.ParsedText, &d.TeacherInstruction, &d.Title, &d.Subject, &d.Language, &d.CodeLanguage, &d.ProblemFocus, &d.ScenarioID, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (p *Postgres) CreateDocument(ctx context.Context, doc *Document) error {
	if doc == nil {
		return fmt.Errorf("document required")
	}
	if strings.TrimSpace(doc.ID) == "" {
		doc.ID = "doc-" + strings.ReplaceAll(time.Now().UTC().Format("20060102150405.000000000"), ".", "")
	}
	if doc.CodeLanguage == "" {
		doc.CodeLanguage = "python"
	}
	now := time.Now().UTC()
	if doc.CreatedAt.IsZero() {
		doc.CreatedAt = now
	}
	doc.UpdatedAt = now
	_, err := p.db.ExecContext(ctx, `
		INSERT INTO documents (id, file_name, content_type, parsed_text, teacher_instruction, title, subject, language, code_language, problem_focus, scenario_id, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
	`,
		doc.ID, doc.FileName, doc.ContentType, doc.ParsedText, doc.TeacherInstruction, doc.Title, doc.Subject, doc.Language, doc.CodeLanguage, doc.ProblemFocus, doc.ScenarioID, doc.CreatedAt, doc.UpdatedAt,
	)
	return err
}

func (p *Postgres) GetDocument(ctx context.Context, id string) (*Document, error) {
	var d Document
	err := p.db.QueryRowContext(ctx, `SELECT id, file_name, content_type, parsed_text, teacher_instruction, title, subject, language, code_language, problem_focus, scenario_id, created_at, updated_at FROM documents WHERE id = $1`, id).
		Scan(&d.ID, &d.FileName, &d.ContentType, &d.ParsedText, &d.TeacherInstruction, &d.Title, &d.Subject, &d.Language, &d.CodeLanguage, &d.ProblemFocus, &d.ScenarioID, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (p *Postgres) UpdateDocumentScenario(ctx context.Context, docID, scenarioID string) error {
	_, err := p.db.ExecContext(ctx, `UPDATE documents SET scenario_id = $2, updated_at = NOW() WHERE id = $1`, docID, scenarioID)
	return err
}

func (p *Postgres) CreateSession(ctx context.Context, sess *Session) error {
	if sess == nil {
		return fmt.Errorf("session required")
	}
	if strings.TrimSpace(sess.ID) == "" {
		sess.ID = fmt.Sprintf("sess-%d", time.Now().UnixNano())
	}
	if sess.CreatedAt.IsZero() {
		sess.CreatedAt = time.Now().UTC()
	}
	_, err := p.db.ExecContext(ctx, `INSERT INTO sessions (id, scenario_id, created_at) VALUES ($1,$2,$3)`, sess.ID, sess.ScenarioID, sess.CreatedAt)
	return err
}

func (p *Postgres) GetSession(ctx context.Context, id string) (*Session, error) {
	var s Session
	err := p.db.QueryRowContext(ctx, `SELECT id, scenario_id, created_at FROM sessions WHERE id = $1`, id).Scan(&s.ID, &s.ScenarioID, &s.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (p *Postgres) ListMessages(ctx context.Context, sessionID string) ([]Message, error) {
	rows, err := p.db.QueryContext(ctx, `SELECT id, session_id, role, content, created_at FROM messages WHERE session_id = $1 ORDER BY created_at, id`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Message{}
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.SessionID, &m.Role, &m.Content, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (p *Postgres) AppendMessage(ctx context.Context, msg *Message) error {
	if msg == nil {
		return fmt.Errorf("message required")
	}
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = time.Now().UTC()
	}
	err := p.db.QueryRowContext(ctx, `INSERT INTO messages (session_id, role, content, created_at) VALUES ($1,$2,$3,$4) RETURNING id`, msg.SessionID, msg.Role, msg.Content, msg.CreatedAt).Scan(&msg.ID)
	return err
}

var _ Repository = (*Postgres)(nil)
