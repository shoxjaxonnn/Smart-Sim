package scenario

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
	"smartedu/internal/llm"
)

// Scenario — parsed scenario file (spec section 4).
type Scenario struct {
	ID          string          `yaml:"id"`
	Title       string          `yaml:"title"`
	Subject     string          `yaml:"subject"`
	Language    string          `yaml:"language"`
	Facts       llm.FactsStore  `yaml:"facts"`
	Rubric      []llm.Criterion `yaml:"rubric"`
	ModelAnswer string          `yaml:"model_answer"`

	// Situation — the prose under "## Vaziyat" shown to the student.
	Situation string `yaml:"-"`
}

// Load reads and parses a scenario markdown file: YAML frontmatter between
// the first two "---" fences, then the markdown body (the situation prose).
func Load(path string) (*Scenario, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	front, body, err := splitFrontmatter(string(raw))
	if err != nil {
		return nil, err
	}
	var s Scenario
	if err := yaml.Unmarshal([]byte(front), &s); err != nil {
		return nil, fmt.Errorf("parse frontmatter: %w", err)
	}
	s.Situation = extractSituation(body)
	if s.ID == "" {
		return nil, fmt.Errorf("scenario missing id")
	}
	return &s, nil
}

func splitFrontmatter(text string) (front, body string, err error) {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	t := strings.TrimLeft(text, "\n")
	if !strings.HasPrefix(t, "---") {
		return "", "", fmt.Errorf("file does not start with --- frontmatter fence")
	}
	t = t[len("---"):]
	idx := strings.Index(t, "\n---")
	if idx < 0 {
		return "", "", fmt.Errorf("closing --- fence not found")
	}
	front = t[:idx]
	body = t[idx+len("\n---"):]
	return front, body, nil
}

// extractSituation returns text under the "## Vaziyat" heading (stops at the
// next heading). Falls back to the whole body if the heading is absent.
func extractSituation(body string) string {
	lines := strings.Split(body, "\n")
	var out []string
	capturing := false
	for _, ln := range lines {
		trimmed := strings.TrimSpace(ln)
		if strings.HasPrefix(trimmed, "## ") {
			if capturing {
				break
			}
			if strings.Contains(strings.ToLower(trimmed), "vaziyat") {
				capturing = true
			}
			continue
		}
		if capturing {
			out = append(out, ln)
		}
	}
	res := strings.TrimSpace(strings.Join(out, "\n"))
	if res == "" {
		return strings.TrimSpace(body)
	}
	return res
}
