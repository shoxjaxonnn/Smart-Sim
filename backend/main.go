package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"smartedu/internal/llm"
	"smartedu/internal/scenario"
)

// loadDotEnv reads KEY=VALUE lines from .env into the process env if not
// already set. Zero-dependency; ignores blanks and # comments.
func loadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return // no .env is fine — env vars may be set another way
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
			os.Setenv(k, v)
		}
	}
}

// session — in-memory student session (no DB needed for the MVP demo).
type session struct {
	ScenarioID string
	History    []llm.Message
}

type server struct {
	provider  llm.Provider
	scenarios map[string]*scenario.Scenario
	mu        sync.Mutex
	sessions  map[string]*session
	seq       int
}

func main() {
	loadDotEnv(".env")
	port := envOr("PORT", "8080")

	var provider llm.Provider
	if key := os.Getenv("OPENROUTER_API_KEY"); key != "" {
		model := envOr("OPENROUTER_MODEL", "")
		provider = llm.NewOpenRouter(key, model)
		log.Printf("LLM provider: OpenRouter (model=%s)", llm.OpenRouterModelName(model))
	} else if key := os.Getenv("GEMINI_API_KEY"); key != "" {
		model := envOr("GEMINI_MODEL", "")
		provider = llm.NewGemini(key, model)
		log.Printf("LLM provider: Gemini (model=%s)", llm.ModelName(model))
	} else {
		provider = llm.NewMock()
		log.Println("LLM provider: Mock (set GEMINI_API_KEY to use Gemini)")
	}

	s := &server{
		provider:  provider,
		scenarios: map[string]*scenario.Scenario{},
		sessions:  map[string]*session{},
	}

	dir := envOr("SCENARIOS_DIR", "scenarios")
	if err := s.loadScenarios(dir); err != nil {
		log.Printf("warning: load scenarios: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/scenarios", s.handleScenarios) // GET list
	mux.HandleFunc("/api/scenarios/", s.handleScenarioByID)
	mux.HandleFunc("/api/session", s.handleStartSession) // POST {scenario_id}
	mux.HandleFunc("/api/chat", s.handleChat)            // POST {session_id, message}
	mux.HandleFunc("/api/grade", s.handleGrade)          // POST {session_id, answer}

	addr := ":" + port
	log.Printf("Smart Edu backend on http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, cors(mux)))
}

func (s *server) loadScenarios(dir string) error {
	matches, err := filepath.Glob(filepath.Join(dir, "*.md"))
	if err != nil {
		return err
	}
	for _, p := range matches {
		sc, err := scenario.Load(p)
		if err != nil {
			log.Printf("skip %s: %v", p, err)
			continue
		}
		s.scenarios[sc.ID] = sc
		log.Printf("loaded scenario %q (%s)", sc.ID, sc.Title)
	}
	if len(s.scenarios) == 0 {
		return fmt.Errorf("no scenarios loaded from %s", dir)
	}
	return nil
}

// ---- handlers ----

func (s *server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"ok": true, "scenarios": len(s.scenarios)})
}

type scenarioBrief struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Subject string `json:"subject"`
}

func (s *server) handleScenarios(w http.ResponseWriter, r *http.Request) {
	list := []scenarioBrief{}
	for _, sc := range s.scenarios {
		list = append(list, scenarioBrief{ID: sc.ID, Title: sc.Title, Subject: sc.Subject})
	}
	writeJSON(w, 200, list)
}

func (s *server) handleScenarioByID(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/api/scenarios/"):]
	sc, ok := s.scenarios[id]
	if !ok {
		writeJSON(w, 404, map[string]string{"error": "scenario not found"})
		return
	}
	writeJSON(w, 200, map[string]any{
		"id": sc.ID, "title": sc.Title, "subject": sc.Subject, "situation": sc.Situation,
	})
}

func (s *server) handleStartSession(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ScenarioID string `json:"scenario_id"`
	}
	if !decode(w, r, &in) {
		return
	}
	if _, ok := s.scenarios[in.ScenarioID]; !ok {
		writeJSON(w, 404, map[string]string{"error": "scenario not found"})
		return
	}
	s.mu.Lock()
	s.seq++
	id := fmt.Sprintf("sess-%d-%d", time.Now().Unix(), s.seq)
	s.sessions[id] = &session{ScenarioID: in.ScenarioID}
	s.mu.Unlock()
	writeJSON(w, 200, map[string]string{"session_id": id})
}

func (s *server) handleChat(w http.ResponseWriter, r *http.Request) {
	var in struct {
		SessionID string `json:"session_id"`
		Message   string `json:"message"`
	}
	if !decode(w, r, &in) {
		return
	}
	sess, sc, ok := s.lookup(in.SessionID)
	if !ok {
		writeJSON(w, 404, map[string]string{"error": "session not found"})
		return
	}

	req := llm.ChatRequest{
		SystemPrompt: personaPrompt(sc),
		History:      sess.History,
		UserMessage:  in.Message,
		Facts:        sc.Facts,
	}
	ctx, cancel := context.WithTimeout(r.Context(), 70*time.Second)
	defer cancel()
	reply, err := s.provider.Chat(ctx, req)
	if err != nil {
		log.Printf("chat error: %v", err)
		writeJSON(w, 502, map[string]string{"error": "LLM xatosi: " + err.Error()})
		return
	}

	s.mu.Lock()
	sess.History = append(sess.History,
		llm.Message{Role: "user", Content: in.Message},
		llm.Message{Role: "assistant", Content: reply},
	)
	s.mu.Unlock()

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
	sess, sc, ok := s.lookup(in.SessionID)
	if !ok {
		writeJSON(w, 404, map[string]string{"error": "session not found"})
		return
	}
	answer := in.Answer
	if answer == "" {
		// Grade the student's side of the conversation if no explicit answer.
		for _, m := range sess.History {
			if m.Role == "user" {
				answer += m.Content + "\n"
			}
		}
	}
	ctx, cancel := context.WithTimeout(r.Context(), 70*time.Second)
	defer cancel()
	res, err := s.provider.Grade(ctx, sc.ModelAnswer, answer, sc.Rubric)
	if err != nil {
		log.Printf("grade error: %v", err)
		writeJSON(w, 502, map[string]string{"error": "Baholash xatosi: " + err.Error()})
		return
	}
	writeJSON(w, 200, res)
}

// ---- helpers ----

func (s *server) lookup(sessionID string) (*session, *scenario.Scenario, bool) {
	s.mu.Lock()
	sess, ok := s.sessions[sessionID]
	s.mu.Unlock()
	if !ok {
		return nil, nil, false
	}
	sc, ok := s.scenarios[sess.ScenarioID]
	if !ok {
		return nil, nil, false
	}
	return sess, sc, true
}

func personaPrompt(sc *scenario.Scenario) string {
	keys := make([]string, 0, len(sc.Facts))
	for k := range sc.Facts {
		keys = append(keys, k)
	}
	keyList := strings.Join(keys, ", ")
	return fmt.Sprintf(`You are an interactive tutor running a situational simulation.
Subject: %s. Title: %s.

SITUATION (this is the world; never contradict or go beyond it):
%s

RULES:
- Guide the student to solve the problem themselves. Ask probing questions; do not hand them the full answer.
- You do NOT know any concrete numbers, logs, IPs, file contents, table names, or other specifics from memory. For ANY specific value the student asks about, you MUST call the get_fact tool BEFORE replying. Do NOT say "unavailable" without first calling get_fact and seeing found=false.
- Known fact keys for this scenario (call get_fact with one of these whenever relevant): %s.
- If get_fact returns found=true, share the value with the student. If found=false, tell the student that information is unavailable.
- Stay in character. Reply in the student's language (Uzbek by default). Keep replies concise.`,
		sc.Subject, sc.Title, sc.Situation, keyList)
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	if r.Method != http.MethodPost {
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
	json.NewEncoder(w).Encode(v)
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(204)
			return
		}
		next.ServeHTTP(w, r)
	})
}
