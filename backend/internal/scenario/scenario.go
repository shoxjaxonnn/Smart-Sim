package scenario

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"smartedu/internal/llm"
)

// Scenario — parsed scenario file (spec section 4).
type Scenario struct {
	ID                      string          `json:"id"`
	Title                   string          `json:"title"`
	Subject                 string          `json:"subject"`
	Language                string          `json:"language"`
	Status                  string          `json:"status"`
	CodeChallengeAfterRound int             `json:"code_challenge_after_round"`
	CodeLanguage            string          `json:"code_language"`
	Facts                   llm.FactsStore  `json:"facts"`
	Rubric                  []llm.Criterion `json:"rubric"`
	ModelAnswer             string          `json:"model_answer"`
	CodeChallenge           CodeChallenge   `json:"code_challenge"`

	// Situation — the prose under "## Vaziyat" shown to the student.
	Situation string `json:"situation"`
}

// CodeChallenge — optional code repair task attached to a scenario.
type CodeChallenge struct {
	BuggyCode string `json:"buggy_code"`
	Hint      string `json:"hint"`
	Tests     string `json:"tests"`
}

// Load reads and parses a scenario markdown file: YAML-like frontmatter between
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
	s, err := parseFrontmatter(front)
	if err != nil {
		return nil, err
	}
	s.Situation = extractSituation(body)
	if s.ID == "" {
		return nil, fmt.Errorf("scenario missing id")
	}
	return s, nil
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

func parseFrontmatter(front string) (*Scenario, error) {
	lines := strings.Split(strings.ReplaceAll(front, "\r\n", "\n"), "\n")
	s := &Scenario{Facts: llm.FactsStore{}}

	for i := 0; i < len(lines); {
		line := strings.TrimRight(lines[i], " \t")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			i++
			continue
		}

		switch {
		case strings.HasPrefix(trimmed, "facts:"):
			i++
			for i < len(lines) {
				l := lines[i]
				if !isIndented(l) {
					break
				}
				t := strings.TrimSpace(l)
				if t == "" {
					i++
					continue
				}
				k, v, ok := strings.Cut(t, ":")
				if ok {
					s.Facts[strings.TrimSpace(k)] = stripQuotes(strings.TrimSpace(v))
				}
				i++
			}

		case strings.HasPrefix(trimmed, "rubric:"):
			i++
			for i < len(lines) {
				l := lines[i]
				if !isIndented(l) {
					break
				}
				t := strings.TrimSpace(l)
				if t == "" {
					i++
					continue
				}
				if strings.HasPrefix(t, "- ") {
					item := llm.Criterion{}
					t = strings.TrimSpace(strings.TrimPrefix(t, "- "))
					if key, val, ok := strings.Cut(t, ":"); ok {
						assignCriterionField(&item, key, val)
					}
					i++
					for i < len(lines) && isIndented(lines[i]) && !strings.HasPrefix(strings.TrimSpace(lines[i]), "- ") {
						sub := strings.TrimSpace(lines[i])
						if sub != "" {
							key, val, ok := strings.Cut(sub, ":")
							if ok {
								assignCriterionField(&item, key, val)
							}
						}
						i++
					}
					s.Rubric = append(s.Rubric, item)
					continue
				}
				i++
			}

		case strings.HasPrefix(trimmed, "code_challenge:"):
			i++
			for i < len(lines) {
				l := lines[i]
				if !isIndented(l) {
					break
				}
				t := strings.TrimSpace(l)
				if t == "" {
					i++
					continue
				}
				key, val, ok := strings.Cut(t, ":")
				if !ok {
					i++
					continue
				}
				key = strings.TrimSpace(key)
				val = strings.TrimSpace(val)
				switch key {
				case "buggy_code":
					if val == "|" || val == ">" {
						block, next := consumeBlock(lines, i+1, indentWidth(lines[i]))
						s.CodeChallenge.BuggyCode = strings.TrimSpace(strings.Join(block, "\n"))
						i = next
						continue
					}
					s.CodeChallenge.BuggyCode = stripQuotes(val)
				case "hint":
					if val == "|" || val == ">" {
						block, next := consumeBlock(lines, i+1, indentWidth(lines[i]))
						s.CodeChallenge.Hint = strings.TrimSpace(strings.Join(block, "\n"))
						i = next
						continue
					}
					s.CodeChallenge.Hint = stripQuotes(val)
				case "tests":
					if val == "|" || val == ">" {
						block, next := consumeBlock(lines, i+1, indentWidth(lines[i]))
						s.CodeChallenge.Tests = strings.TrimSpace(strings.Join(block, "\n"))
						i = next
						continue
					}
					s.CodeChallenge.Tests = stripQuotes(val)
				}
				i++
			}

		default:
			key, val, ok := strings.Cut(trimmed, ":")
			if !ok {
				i++
				continue
			}
			key = strings.TrimSpace(key)
			val = strings.TrimSpace(val)
			switch key {
			case "id":
				s.ID = stripQuotes(val)
			case "title":
				s.Title = stripQuotes(val)
			case "subject":
				s.Subject = stripQuotes(val)
			case "language":
				s.Language = stripQuotes(val)
			case "status":
				s.Status = stripQuotes(val)
			case "code_challenge_after_round":
				n, _ := strconv.Atoi(stripQuotes(val))
				s.CodeChallengeAfterRound = n
			case "code_language":
				s.CodeLanguage = stripQuotes(val)
			case "model_answer":
				if val == "|" || val == ">" {
					block, next := consumeBlock(lines, i+1, indentWidth(lines[i]))
					s.ModelAnswer = strings.TrimSpace(strings.Join(block, "\n"))
					i = next
					continue
				}
				s.ModelAnswer = stripQuotes(val)
			}
			i++
		}
	}

	if s.Status == "" {
		s.Status = "approved"
	}
	return s, nil
}

func assignCriterionField(c *llm.Criterion, key, raw string) {
	key = strings.TrimSpace(key)
	raw = strings.TrimSpace(raw)
	switch key {
	case "name":
		c.Name = stripQuotes(raw)
	case "max":
		n, _ := strconv.Atoi(stripQuotes(raw))
		c.Max = n
	case "keywords":
		c.Keywords = parseStringList(raw)
	}
}

func parseStringList(raw string) []string {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "[")
	raw = strings.TrimSuffix(raw, "]")
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = stripQuotes(strings.TrimSpace(p))
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func consumeBlock(lines []string, start int, parentIndent int) ([]string, int) {
	out := []string{}
	i := start
	for i < len(lines) {
		l := lines[i]
		if strings.TrimSpace(l) == "" {
			out = append(out, "")
			i++
			continue
		}
		if indentWidth(l) <= parentIndent {
			break
		}
		out = append(out, strings.TrimLeft(l, " \t"))
		i++
	}
	return out, i
}

func isIndented(line string) bool {
	return strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t")
}

func indentWidth(line string) int {
	n := 0
	for _, r := range line {
		if r == ' ' {
			n++
			continue
		}
		if r == '\t' {
			n += 4
			continue
		}
		break
	}
	return n
}

func stripQuotes(v string) string {
	v = strings.TrimSpace(v)
	v = strings.Trim(v, `"'`)
	return v
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
