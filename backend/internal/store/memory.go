package store

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"smartedu/internal/scenario"
)

type Memory struct {
	mu        sync.RWMutex
	scenarios map[string]*scenario.Scenario
	documents map[string]*Document
	sessions  map[string]*Session
	messages  map[string][]Message
	msgSeq    int64
}

func NewMemory() *Memory {
	return &Memory{
		scenarios: map[string]*scenario.Scenario{},
		documents: map[string]*Document{},
		sessions:  map[string]*Session{},
		messages:  map[string][]Message{},
	}
}

func (m *Memory) Init(ctx context.Context) error { return nil }

func (m *Memory) SeedScenarios(ctx context.Context, dir string) error {
	matches, err := filepath.Glob(filepath.Join(dir, "*.md"))
	if err != nil {
		return err
	}
	for _, p := range matches {
		sc, err := scenario.Load(p)
		if err != nil {
			continue
		}
		_ = m.UpsertScenario(ctx, sc)
	}
	return nil
}

func (m *Memory) ListScenarios(ctx context.Context, includeDrafts bool) ([]scenario.Scenario, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]scenario.Scenario, 0, len(m.scenarios))
	for _, sc := range m.scenarios {
		if !includeDrafts && !isApprovedScenario(sc) {
			continue
		}
		out = append(out, cloneScenario(sc))
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Status != out[j].Status {
			return out[i].Status < out[j].Status
		}
		return out[i].Title < out[j].Title
	})
	return out, nil
}

func (m *Memory) GetScenario(ctx context.Context, id string) (*scenario.Scenario, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	sc, ok := m.scenarios[id]
	if !ok {
		return nil, fmt.Errorf("scenario not found")
	}
	out := cloneScenario(sc)
	return &out, nil
}

func (m *Memory) UpsertScenario(ctx context.Context, sc *scenario.Scenario) error {
	if sc == nil {
		return fmt.Errorf("scenario required")
	}
	if strings.TrimSpace(sc.ID) == "" {
		sc.ID = seedScenarioID(sc.Title)
	}
	copy := cloneScenario(sc)
	if copy.Status == "" {
		copy.Status = "approved"
	}
	m.mu.Lock()
	m.scenarios[copy.ID] = &copy
	m.mu.Unlock()
	return nil
}

func (m *Memory) ApproveScenario(ctx context.Context, id string) (*scenario.Scenario, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	sc, ok := m.scenarios[id]
	if !ok {
		return nil, fmt.Errorf("scenario not found")
	}
	sc.Status = "approved"
	out := cloneScenario(sc)
	return &out, nil
}

func (m *Memory) ListDocuments(ctx context.Context) ([]Document, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Document, 0, len(m.documents))
	for _, d := range m.documents {
		out = append(out, *d)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}

func (m *Memory) CreateDocument(ctx context.Context, doc *Document) error {
	if doc == nil {
		return fmt.Errorf("document required")
	}
	if strings.TrimSpace(doc.ID) == "" {
		doc.ID = "doc-" + strings.ReplaceAll(time.Now().UTC().Format("20060102150405.000000000"), ".", "")
	}
	now := time.Now().UTC()
	if doc.CreatedAt.IsZero() {
		doc.CreatedAt = now
	}
	doc.UpdatedAt = now
	copy := *doc
	m.mu.Lock()
	m.documents[copy.ID] = &copy
	m.mu.Unlock()
	return nil
}

func (m *Memory) GetDocument(ctx context.Context, id string) (*Document, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	doc, ok := m.documents[id]
	if !ok {
		return nil, fmt.Errorf("document not found")
	}
	copy := *doc
	return &copy, nil
}

func (m *Memory) UpdateDocumentScenario(ctx context.Context, docID, scenarioID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	doc, ok := m.documents[docID]
	if !ok {
		return fmt.Errorf("document not found")
	}
	doc.ScenarioID = scenarioID
	doc.UpdatedAt = time.Now().UTC()
	return nil
}

func (m *Memory) CreateSession(ctx context.Context, sess *Session) error {
	if sess == nil {
		return fmt.Errorf("session required")
	}
	if strings.TrimSpace(sess.ID) == "" {
		sess.ID = fmt.Sprintf("sess-%d", time.Now().UnixNano())
	}
	if sess.CreatedAt.IsZero() {
		sess.CreatedAt = time.Now().UTC()
	}
	copy := *sess
	m.mu.Lock()
	m.sessions[copy.ID] = &copy
	m.mu.Unlock()
	return nil
}

func (m *Memory) GetSession(ctx context.Context, id string) (*Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	sess, ok := m.sessions[id]
	if !ok {
		return nil, fmt.Errorf("session not found")
	}
	copy := *sess
	return &copy, nil
}

func (m *Memory) ListMessages(ctx context.Context, sessionID string) ([]Message, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	msgs := append([]Message(nil), m.messages[sessionID]...)
	sort.Slice(msgs, func(i, j int) bool {
		if msgs[i].CreatedAt.Equal(msgs[j].CreatedAt) {
			return msgs[i].ID < msgs[j].ID
		}
		return msgs[i].CreatedAt.Before(msgs[j].CreatedAt)
	})
	return msgs, nil
}

func (m *Memory) AppendMessage(ctx context.Context, msg *Message) error {
	if msg == nil {
		return fmt.Errorf("message required")
	}
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = time.Now().UTC()
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.msgSeq++
	msg.ID = m.msgSeq
	copy := *msg
	m.messages[msg.SessionID] = append(m.messages[msg.SessionID], copy)
	return nil
}
