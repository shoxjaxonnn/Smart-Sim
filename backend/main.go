package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"smartedu/internal/docx"
	"smartedu/internal/llm"
	"smartedu/internal/sandbox"
	"smartedu/internal/scenario"
	"smartedu/internal/store"
)

type server struct {
	repo             store.Repository
	teacherProvider  llm.Provider
	studentProvider  llm.Provider
	documentProvider llm.Provider
}

func main() {
	loadDotEnv(".env")
	ctx := context.Background()

	repo, err := store.Open(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Printf("postgres unavailable, fallback memory: %v", err)
		repo = store.NewMemory()
	}

	if err := repo.Init(ctx); err != nil {
		log.Fatalf("db init: %v", err)
	}
	if err := repo.SeedScenarios(ctx, envOr("SCENARIOS_DIR", "scenarios")); err != nil {
		log.Printf("warning: seed scenarios: %v", err)
	}

	teacherProvider := providerFromEnv("TEACHER")
	studentProvider := providerFromEnv("STUDENT")
	documentProvider := providerFromEnv("DOCUMENT")

	if teacherProvider == nil && documentProvider != nil {
		teacherProvider = documentProvider
	}
	if documentProvider == nil && teacherProvider != nil {
		documentProvider = teacherProvider
	}
	if studentProvider == nil && teacherProvider != nil {
		studentProvider = teacherProvider
	}
	if teacherProvider == nil && studentProvider == nil && documentProvider == nil {
		mock := llm.NewMock()
		teacherProvider = mock
		studentProvider = mock
		documentProvider = mock
		log.Println("LLM providers: Mock")
	}
	if teacherProvider == nil {
		teacherProvider = studentProvider
	}
	if documentProvider == nil {
		documentProvider = teacherProvider
	}
	if studentProvider == nil {
		studentProvider = teacherProvider
	}

	s := &server{
		repo:             repo,
		teacherProvider:  teacherProvider,
		studentProvider:  studentProvider,
		documentProvider: documentProvider,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/scenarios", s.handleScenarios)
	mux.HandleFunc("/api/scenarios/", s.handleScenarioByID)
	mux.HandleFunc("/api/session", s.handleStartSession)
	mux.HandleFunc("/api/chat", s.handleChat)
	mux.HandleFunc("/api/grade", s.handleGrade)
	mux.HandleFunc("/api/sandbox/submit", s.handleSandboxSubmit)
	mux.HandleFunc("/api/teacher/scenarios", s.handleTeacherScenarios)
	mux.HandleFunc("/api/teacher/scenarios/", s.handleTeacherScenarioByID)
	mux.HandleFunc("/api/teacher/documents", s.handleTeacherDocuments)
	mux.HandleFunc("/api/teacher/documents/", s.handleTeacherDocumentByID)

	port := envOr("PORT", "8080")
	addr := ":" + port
	log.Printf("Smart Edu backend on http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, cors(mux)))
}

// ---- env / provider ----

func loadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.Trim(strings.TrimSpace(v), `"'`)
		if _, exists := os.LookupEnv(k); !exists {
			_ = os.Setenv(k, v)
		}
	}
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func providerFromEnv(role string) llm.Provider {
	role = strings.ToUpper(strings.TrimSpace(role))

	if key := os.Getenv("GEMINI_" + role + "_API_KEY"); key != "" {
		model := os.Getenv("GEMINI_" + role + "_MODEL")
		return llm.NewGemini(key, model)
	}
	if key := os.Getenv("OPENROUTER_" + role + "_API_KEY"); key != "" {
		model := os.Getenv("OPENROUTER_" + role + "_MODEL")
		return llm.NewOpenRouter(key, model)
	}
	if key := os.Getenv("GEMINI_API_KEY"); key != "" {
		model := os.Getenv("GEMINI_MODEL")
		return llm.NewGemini(key, model)
	}
	if key := os.Getenv("OPENROUTER_API_KEY"); key != "" {
		model := os.Getenv("OPENROUTER_MODEL")
		return llm.NewOpenRouter(key, model)
	}
	return nil
}

// ---- routes ----

func (s *server) handleHealth(w http.ResponseWriter, r *http.Request) {
	scs, err := s.repo.ListScenarios(r.Context(), true)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	docs, err := s.repo.ListDocuments(r.Context())
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	approved := 0
	for i := range scs {
		if isApproved(&scs[i]) {
			approved++
		}
	}
	currentSubject := ""
	currentDocumentTitle := ""
	if len(docs) > 0 {
		currentSubject = strings.TrimSpace(docs[0].Subject)
		currentDocumentTitle = strings.TrimSpace(docs[0].Title)
		if currentDocumentTitle == "" {
			currentDocumentTitle = strings.TrimSpace(docs[0].FileName)
		}
	}
	writeJSON(w, 200, map[string]any{
		"ok":               true,
		"total":            len(scs),
		"approved":         approved,
		"current_subject":  currentSubject,
		"current_document": currentDocumentTitle,
		"current_document_id": func() string {
			if len(docs) > 0 {
				return docs[0].ID
			}
			return ""
		}(),
	})
}

func (s *server) handleScenarios(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, 405, map[string]string{"error": "method not allowed"})
		return
	}
	list, err := s.repo.ListScenarios(r.Context(), false)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, 200, briefScenarios(list))
}

func (s *server) handleScenarioByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, 405, map[string]string{"error": "method not allowed"})
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/scenarios/")
	sc, err := s.repo.GetScenario(r.Context(), id)
	if err != nil || !isApproved(sc) {
		writeJSON(w, 404, map[string]string{"error": "scenario not found"})
		return
	}
	writeJSON(w, 200, scenarioResponse(sc))
}

func (s *server) handleStartSession(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ScenarioID string `json:"scenario_id"`
	}
	if !decode(w, r, &in) {
		return
	}
	sc, err := s.repo.GetScenario(r.Context(), in.ScenarioID)
	if err != nil || !isApproved(sc) {
		writeJSON(w, 404, map[string]string{"error": "scenario not found"})
		return
	}
	sess := &store.Session{ScenarioID: in.ScenarioID}
	if err := s.repo.CreateSession(r.Context(), sess); err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, 200, map[string]string{"session_id": sess.ID})
}

func (s *server) handleChat(w http.ResponseWriter, r *http.Request) {
	var in struct {
		SessionID string `json:"session_id"`
		Message   string `json:"message"`
	}
	if !decode(w, r, &in) {
		return
	}
	sess, sc, history, ok, err := s.sessionContext(r.Context(), in.SessionID)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	if !ok {
		writeJSON(w, 404, map[string]string{"error": "session not found"})
		return
	}

	req := llm.ChatRequest{
		SystemPrompt: personaPrompt(sc),
		History:      history,
		UserMessage:  in.Message,
		Facts:        sc.Facts,
	}
	ctx, cancel := context.WithTimeout(r.Context(), 70*time.Second)
	defer cancel()
	reply, err := s.studentProvider.Chat(ctx, req)
	if err != nil {
		writeJSON(w, 502, map[string]string{"error": "LLM xatosi: " + err.Error()})
		return
	}
	if err := s.repo.AppendMessage(r.Context(), &store.Message{SessionID: sess.ID, Role: "user", Content: in.Message}); err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	if err := s.repo.AppendMessage(r.Context(), &store.Message{SessionID: sess.ID, Role: "assistant", Content: reply}); err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, 200, map[string]string{"reply": reply})
}

func (s *server) handleGrade(w http.ResponseWriter, r *http.Request) {
	var in struct {
		SessionID string `json:"session_id"`
		Answer    string `json:"answer"`
	}
	if !decode(w, r, &in) {
		return
	}
	_, sc, history, ok, err := s.sessionContext(r.Context(), in.SessionID)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	if !ok {
		writeJSON(w, 404, map[string]string{"error": "session not found"})
		return
	}
	answer := strings.TrimSpace(in.Answer)
	if answer == "" {
		var parts []string
		for _, m := range history {
			if m.Role == "user" {
				parts = append(parts, m.Content)
			}
		}
		answer = strings.Join(parts, "\n")
	}
	ctx, cancel := context.WithTimeout(r.Context(), 70*time.Second)
	defer cancel()
	res, err := s.studentProvider.Grade(ctx, sc.ModelAnswer, answer, sc.Rubric)
	if err != nil {
		writeJSON(w, 502, map[string]string{"error": "Baholash xatosi: " + err.Error()})
		return
	}
	writeJSON(w, 200, res)
}

func (s *server) handleSandboxSubmit(w http.ResponseWriter, r *http.Request) {
	var in struct {
		SessionID string `json:"session_id"`
		Code      string `json:"code"`
	}
	if !decode(w, r, &in) {
		return
	}
	_, sc, _, ok, err := s.sessionContext(r.Context(), in.SessionID)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	if !ok {
		writeJSON(w, 404, map[string]string{"error": "session not found"})
		return
	}
	if strings.TrimSpace(sc.CodeChallenge.Tests) == "" {
		writeJSON(w, 400, map[string]string{"error": "scenario has no code challenge"})
		return
	}
	if strings.TrimSpace(in.Code) == "" {
		writeJSON(w, 400, map[string]string{"error": "code is required"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 7*time.Second)
	defer cancel()
	res, err := sandbox.RunPython(ctx, in.Code, sc.CodeChallenge.Tests)
	if err != nil {
		writeJSON(w, 502, map[string]string{"error": "sandbox xatosi: " + err.Error()})
		return
	}
	writeJSON(w, 200, map[string]any{
		"passed":      res.Passed,
		"timed_out":   res.TimedOut,
		"exit_code":   res.ExitCode,
		"stdout":      res.Stdout,
		"stderr":      res.Stderr,
		"duration_ms": res.DurationMs,
		"error":       res.Error,
	})
}

func (s *server) handleTeacherScenarios(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		list, err := s.repo.ListScenarios(r.Context(), true)
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, 200, briefScenarios(list))
	case http.MethodPost:
		s.handleTeacherGenerateScenario(w, r)
	default:
		writeJSON(w, 405, map[string]string{"error": "method not allowed"})
	}
}

func (s *server) handleTeacherGenerateScenario(w http.ResponseWriter, r *http.Request) {
	var in llm.ScenarioDraftRequest
	if !decode(w, r, &in) {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
	defer cancel()
	draft, err := s.teacherProvider.GenerateScenario(ctx, in)
	if err != nil {
		writeJSON(w, 502, map[string]string{"error": "senariy yaratish xatosi: " + err.Error()})
		return
	}
	if draft.Title == "" {
		draft.Title = fallback(in.Title, "Yangi senariy")
	}
	if draft.Subject == "" {
		draft.Subject = fallback(in.Subject, "IT / Web Security")
	}
	if draft.Language == "" {
		draft.Language = fallback(in.Language, "uz")
	}
	if draft.CodeLanguage == "" {
		draft.CodeLanguage = fallback(in.CodeLanguage, "python")
	}
	sc := scenarioFromDraft(draft)
	sc.ID = normalizeID(fallback(in.Title, draft.Title))
	sc.Status = "draft"
	if err := s.repo.UpsertScenario(r.Context(), sc); err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	updated, _ := s.repo.GetScenario(r.Context(), sc.ID)
	writeJSON(w, 201, scenarioResponse(updated))
}

func (s *server) handleTeacherScenarioByID(w http.ResponseWriter, r *http.Request) {
	raw := strings.TrimPrefix(r.URL.Path, "/api/teacher/scenarios/")
	approve := strings.HasSuffix(raw, "/approve")
	id := strings.TrimSuffix(raw, "/approve")
	id = strings.TrimSuffix(id, "/")
	if id == "" {
		writeJSON(w, 404, map[string]string{"error": "scenario not found"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		sc, err := s.repo.GetScenario(r.Context(), id)
		if err != nil {
			writeJSON(w, 404, map[string]string{"error": "scenario not found"})
			return
		}
		writeJSON(w, 200, scenarioResponse(sc))
	case http.MethodPut:
		var incoming scenario.Scenario
		if !decode(w, r, &incoming) {
			return
		}
		current, err := s.repo.GetScenario(r.Context(), id)
		if err != nil {
			writeJSON(w, 404, map[string]string{"error": "scenario not found"})
			return
		}
		if incoming.Status == "" {
			incoming.Status = current.Status
		}
		if incoming.CodeLanguage == "" {
			incoming.CodeLanguage = current.CodeLanguage
		}
		if incoming.Language == "" {
			incoming.Language = current.Language
		}
		incoming.ID = id
		if err := s.repo.UpsertScenario(r.Context(), &incoming); err != nil {
			writeJSON(w, 500, map[string]string{"error": err.Error()})
			return
		}
		updated, _ := s.repo.GetScenario(r.Context(), id)
		writeJSON(w, 200, scenarioResponse(updated))
	case http.MethodPatch:
		if !approve {
			writeJSON(w, 404, map[string]string{"error": "scenario not found"})
			return
		}
		updated, err := s.repo.ApproveScenario(r.Context(), id)
		if err != nil {
			writeJSON(w, 404, map[string]string{"error": "scenario not found"})
			return
		}
		writeJSON(w, 200, scenarioResponse(updated))
	default:
		writeJSON(w, 405, map[string]string{"error": "method not allowed"})
	}
}

func (s *server) handleTeacherDocuments(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		docs, err := s.repo.ListDocuments(r.Context())
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, 200, docs)
	case http.MethodPost:
		s.handleUploadDocument(w, r)
	default:
		writeJSON(w, 405, map[string]string{"error": "method not allowed"})
	}
}

func (s *server) handleTeacherDocumentByID(w http.ResponseWriter, r *http.Request) {
	raw := strings.TrimPrefix(r.URL.Path, "/api/teacher/documents/")
	if strings.HasSuffix(raw, "/generate-scenario") {
		s.handleGenerateScenarioFromDoc(w, r)
		return
	}
	id := strings.TrimSuffix(raw, "/")
	if id == "" {
		writeJSON(w, 404, map[string]string{"error": "document not found"})
		return
	}
	switch r.Method {
	case http.MethodGet:
		doc, err := s.repo.GetDocument(r.Context(), id)
		if err != nil {
			writeJSON(w, 404, map[string]string{"error": "document not found"})
			return
		}
		writeJSON(w, 200, doc)
	default:
		writeJSON(w, 405, map[string]string{"error": "method not allowed"})
	}
}

func (s *server) handleUploadDocument(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(12 << 20); err != nil {
		writeJSON(w, 400, map[string]string{"error": "bad multipart: " + err.Error()})
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, 400, map[string]string{"error": "file required"})
		return
	}
	defer file.Close()

	filename := header.Filename
	if !strings.HasSuffix(strings.ToLower(filename), ".docx") {
		writeJSON(w, 400, map[string]string{"error": "only .docx allowed"})
		return
	}
	raw, err := ioReadAllLimit(file, 12<<20)
	if err != nil {
		writeJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}
	parsed, err := docx.ExtractText(raw)
	if err != nil {
		writeJSON(w, 400, map[string]string{"error": "docx parse xatosi: " + err.Error()})
		return
	}

	doc := &store.Document{
		FileName:           filename,
		ContentType:        header.Header.Get("Content-Type"),
		ParsedText:         parsed,
		TeacherInstruction: strings.TrimSpace(r.FormValue("instruction")),
		Title:              strings.TrimSpace(r.FormValue("title")),
		Subject:            strings.TrimSpace(r.FormValue("subject")),
		Language:           strings.TrimSpace(r.FormValue("language")),
		CodeLanguage:       fallback(strings.TrimSpace(r.FormValue("code_language")), "python"),
		ProblemFocus:       strings.TrimSpace(r.FormValue("problem_focus")),
	}
	if doc.ContentType == "" {
		doc.ContentType = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	}
	if err := s.repo.CreateDocument(r.Context(), doc); err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, 201, doc)
}

func (s *server) handleGenerateScenarioFromDoc(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, 405, map[string]string{"error": "method not allowed"})
		return
	}
	docID := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/teacher/documents/"), "/generate-scenario")
	docID = strings.Trim(docID, "/")
	if docID == "" {
		writeJSON(w, 404, map[string]string{"error": "document not found"})
		return
	}

	var in struct {
		Title              string `json:"title"`
		Subject            string `json:"subject"`
		Language           string `json:"language"`
		CodeLanguage       string `json:"code_language"`
		ProblemFocus       string `json:"problem_focus"`
		TeacherInstruction string `json:"teacher_instruction"`
	}
	if r.Body != nil && r.Method == http.MethodPost {
		_ = json.NewDecoder(r.Body).Decode(&in)
	}

	doc, err := s.repo.GetDocument(r.Context(), docID)
	if err != nil {
		writeJSON(w, 404, map[string]string{"error": "document not found"})
		return
	}

	prompt := llm.ScenarioDraftRequest{
		Title:              fallback(strings.TrimSpace(in.Title), doc.Title),
		Subject:            fallback(strings.TrimSpace(in.Subject), doc.Subject),
		Language:           fallback(strings.TrimSpace(in.Language), doc.Language),
		CodeLanguage:       fallback(strings.TrimSpace(in.CodeLanguage), doc.CodeLanguage),
		ProblemFocus:       fallback(strings.TrimSpace(in.ProblemFocus), doc.ProblemFocus),
		SourceDocumentName: doc.FileName,
		TeacherInstruction: fallback(strings.TrimSpace(in.TeacherInstruction), doc.TeacherInstruction),
		DocumentText:       doc.ParsedText,
		LessonContext:      doc.ParsedText,
	}

	ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
	defer cancel()
	draft, err := s.documentProvider.GenerateScenario(ctx, prompt)
	if err != nil {
		writeJSON(w, 502, map[string]string{"error": "senariy yaratish xatosi: " + err.Error()})
		return
	}
	if draft.Title == "" {
		draft.Title = fallback(prompt.Title, "Yangi senariy")
	}
	if draft.Subject == "" {
		draft.Subject = fallback(prompt.Subject, "IT / Web Security")
	}
	if draft.Language == "" {
		draft.Language = fallback(prompt.Language, "uz")
	}
	if draft.CodeLanguage == "" {
		draft.CodeLanguage = fallback(prompt.CodeLanguage, "python")
	}
	sc := scenarioFromDraft(draft)
	sc.Status = "draft"
	sc.ID = fallback(doc.ScenarioID, normalizeID(doc.ID+"-"+draft.Title))
	if err := s.repo.UpsertScenario(r.Context(), sc); err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	if err := s.repo.UpdateDocumentScenario(r.Context(), doc.ID, sc.ID); err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	updated, _ := s.repo.GetScenario(r.Context(), sc.ID)
	writeJSON(w, 201, map[string]any{
		"document": doc,
		"scenario": scenarioResponse(updated),
	})
}

// ---- helpers ----

func (s *server) sessionContext(ctx context.Context, sessionID string) (*store.Session, *scenario.Scenario, []llm.Message, bool, error) {
	sess, err := s.repo.GetSession(ctx, sessionID)
	if err != nil {
		return nil, nil, nil, false, nil
	}
	sc, err := s.repo.GetScenario(ctx, sess.ScenarioID)
	if err != nil || !isApproved(sc) {
		return nil, nil, nil, false, nil
	}
	msgs, err := s.repo.ListMessages(ctx, sessionID)
	if err != nil {
		return nil, nil, nil, false, err
	}
	history := make([]llm.Message, 0, len(msgs))
	for _, m := range msgs {
		history = append(history, llm.Message{Role: m.Role, Content: m.Content})
	}
	return sess, sc, history, true, nil
}

func personaPrompt(sc *scenario.Scenario) string {
	keys := make([]string, 0, len(sc.Facts))
	for k := range sc.Facts {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	keyList := strings.Join(keys, ", ")

	codeSection := "No code challenge attached."
	if sc.CodeChallenge.BuggyCode != "" {
		codeSection = fmt.Sprintf("Buggy code:\n%s\n\nHint: %s\n\nTests:\n%s", sc.CodeChallenge.BuggyCode, sc.CodeChallenge.Hint, sc.CodeChallenge.Tests)
	}

	return fmt.Sprintf(`You are an interactive tutor running a situational simulation.
Subject: %s. Title: %s.

SITUATION:
%s

CODE CHALLENGE:
%s

RULES:
- Guide the student to solve the problem themselves. Ask probing questions; do not hand them the full answer.
- You do NOT know any concrete numbers, logs, IPs, file contents, table names, or other specifics from memory. For ANY specific value the student asks about, you MUST call the get_fact tool BEFORE replying.
- Known fact keys for this scenario: %s.
- If get_fact returns found=true, share the value with the student. If found=false, tell the student that information is unavailable.
- Stay in character. Reply in the student's language (Uzbek by default). Keep replies concise.`,
		sc.Subject, sc.Title, sc.Situation, codeSection, keyList)
}

func isApproved(sc *scenario.Scenario) bool {
	if sc == nil {
		return false
	}
	status := strings.ToLower(strings.TrimSpace(sc.Status))
	return status == "" || status == "approved"
}

func scenarioFromDraft(d llm.ScenarioDraft) *scenario.Scenario {
	return &scenario.Scenario{
		Title:                   d.Title,
		Subject:                 d.Subject,
		Language:                d.Language,
		Status:                  "draft",
		CodeChallengeAfterRound: d.CodeChallengeAfterRound,
		CodeLanguage:            d.CodeLanguage,
		Facts:                   llm.FactsStore(d.Facts),
		Rubric:                  d.Rubric,
		ModelAnswer:             d.ModelAnswer,
		CodeChallenge: scenario.CodeChallenge{
			BuggyCode: d.BuggyCode,
			Hint:      d.Hint,
			Tests:     d.Tests,
		},
		Situation: d.Situation,
	}
}

func briefScenarios(list []scenario.Scenario) []map[string]any {
	out := make([]map[string]any, 0, len(list))
	for _, sc := range list {
		out = append(out, map[string]any{
			"id":      sc.ID,
			"title":   sc.Title,
			"subject": sc.Subject,
			"status":  sc.Status,
		})
	}
	return out
}

func scenarioResponse(sc *scenario.Scenario) map[string]any {
	return map[string]any{
		"id":                         sc.ID,
		"title":                      sc.Title,
		"subject":                    sc.Subject,
		"language":                   sc.Language,
		"status":                     sc.Status,
		"code_challenge_after_round": sc.CodeChallengeAfterRound,
		"code_language":              sc.CodeLanguage,
		"facts":                      sc.Facts,
		"rubric":                     sc.Rubric,
		"model_answer":               sc.ModelAnswer,
		"code_challenge":             sc.CodeChallenge,
		"situation":                  sc.Situation,
	}
}

func fallback(v, def string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return def
	}
	return v
}

func normalizeID(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	v = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return r
		case r >= '0' && r <= '9':
			return r
		case r == '-' || r == '_':
			return r
		case r == ' ':
			return '-'
		default:
			return -1
		}
	}, v)
	v = strings.Trim(v, "-")
	if v == "" {
		return "scenario"
	}
	return v
}

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	switch r.Method {
	case http.MethodPost, http.MethodPut, http.MethodPatch:
	default:
		writeJSON(w, 405, map[string]string{"error": "method not allowed"})
		return false
	}
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeJSON(w, 400, map[string]string{"error": "bad json: " + err.Error()})
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(204)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func ioReadAllLimit(r io.Reader, limit int64) ([]byte, error) {
	return io.ReadAll(io.LimitReader(r, limit))
}
