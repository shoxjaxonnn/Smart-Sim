package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultOpenRouterModel = "openai/gpt-oss-120b:free"

// OpenRouter implements Provider via the OpenRouter REST API (OpenAI-compatible
// chat completions). Works in regions where Google AI Studio is blocked.
type OpenRouter struct {
	apiKey string
	model  string
	http   *http.Client
}

func OpenRouterModelName(m string) string {
	if m == "" {
		return defaultOpenRouterModel
	}
	return m
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func NewOpenRouter(apiKey, model string) *OpenRouter {
	return &OpenRouter{
		apiKey: apiKey,
		model:  OpenRouterModelName(model),
		http:   &http.Client{Timeout: 60 * time.Second},
	}
}

// ---- OpenAI-compatible payload shapes ----

type orMessage struct {
	Role       string           `json:"role"`
	Content    string           `json:"content,omitempty"`
	ToolCalls  []orToolCall     `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
	Name       string           `json:"name,omitempty"`
}

type orToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type orTool struct {
	Type     string `json:"type"`
	Function struct {
		Name        string      `json:"name"`
		Description string      `json:"description"`
		Parameters  interface{} `json:"parameters"`
	} `json:"function"`
}

type orRequest struct {
	Model          string      `json:"model"`
	Messages       []orMessage `json:"messages"`
	Tools          []orTool    `json:"tools,omitempty"`
	ResponseFormat interface{} `json:"response_format,omitempty"`
	Temperature    float64     `json:"temperature,omitempty"`
}

type orResponse struct {
	Choices []struct {
		Message orMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Code    any    `json:"code"`
	} `json:"error"`
}

func (o *OpenRouter) call(ctx context.Context, body orRequest) (*orResponse, error) {
	buf, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://openrouter.ai/api/v1/chat/completions", bytes.NewReader(buf))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+o.apiKey)
	req.Header.Set("HTTP-Referer", "https://smart-edu.local")
	req.Header.Set("X-Title", "Smart Edu")
	resp, err := o.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var out orResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("decode openrouter response: %w (raw: %s)", err, string(raw))
	}
	if out.Error != nil {
		return nil, fmt.Errorf("openrouter error: %s | raw: %s", out.Error.Message, truncate(string(raw), 400))
	}
	return &out, nil
}

// Chat runs the chat loop with get_fact wired as a tool (OpenAI tool-calling
// format). Same anti-hallucination guarantee: the model can ONLY get scenario
// numbers via get_fact, never invented (spec 3.1).
func (o *OpenRouter) Chat(ctx context.Context, req ChatRequest) (string, error) {
	tools := []orTool{{
		Type: "function",
		Function: struct {
			Name        string      `json:"name"`
			Description string      `json:"description"`
			Parameters  interface{} `json:"parameters"`
		}{
			Name:        "get_fact",
			Description: "Retrieve a verified scenario fact by key. You MUST use this for any concrete number, log line, or detail. Never invent values.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"key": map[string]interface{}{
						"type":        "string",
						"description": "Fact key, e.g. server.cpu",
					},
				},
				"required": []string{"key"},
			},
		},
	}}

	msgs := []orMessage{{Role: "system", Content: req.SystemPrompt}}
	for _, m := range req.History {
		role := m.Role
		if role == "assistant" || role == "user" {
			msgs = append(msgs, orMessage{Role: role, Content: m.Content})
		}
	}
	msgs = append(msgs, orMessage{Role: "user", Content: req.UserMessage})

	for i := 0; i < 5; i++ {
		resp, err := o.call(ctx, orRequest{
			Model:       o.model,
			Messages:    msgs,
			Tools:       tools,
			Temperature: 0.7,
		})
		if err != nil {
			return "", err
		}
		if len(resp.Choices) == 0 {
			return "", fmt.Errorf("openrouter returned no choices")
		}
		choice := resp.Choices[0].Message

		if len(choice.ToolCalls) == 0 {
			return strings.TrimSpace(choice.Content), nil
		}

		// Echo assistant tool-call turn, then append each tool result.
		msgs = append(msgs, choice)
		for _, tc := range choice.ToolCalls {
			var args struct {
				Key string `json:"key"`
			}
			_ = json.Unmarshal([]byte(tc.Function.Arguments), &args)
			fact := req.Facts.Get(args.Key)
			result, _ := json.Marshal(map[string]any{"value": fact.Value, "found": fact.Found})
			msgs = append(msgs, orMessage{
				Role:       "tool",
				ToolCallID: tc.ID,
				Name:       "get_fact",
				Content:    string(result),
			})
		}
	}
	return "", fmt.Errorf("get_fact tool loop exceeded limit")
}

// Grade — JSON-mode grading. Uses response_format=json_object and a clear
// schema in the prompt (some models on OpenRouter don't support json_schema).
func (o *OpenRouter) Grade(ctx context.Context, modelAnswer, studentAnswer string, rubric []Criterion) (GradeResult, error) {
	var out GradeResult

	var rb strings.Builder
	maxTotal := 0
	for _, c := range rubric {
		maxTotal += c.Max
		fmt.Fprintf(&rb, "- %s (max %d). Keywords (credit only when used correctly, not merely mentioned): %s\n",
			c.Name, c.Max, strings.Join(c.Keywords, ", "))
	}

	prompt := fmt.Sprintf(`You are a strict grader. Grade the student answer against the rubric.
A keyword earns points ONLY if used correctly and with understanding — never for merely naming it.

RUBRIC (total max %d):
%s
MODEL ANSWER:
%s

STUDENT ANSWER:
%s

Return ONLY a JSON object with this exact shape:
{
  "total_score": <int>,
  "max_score": <int>,
  "criteria": [
    {"name": "<criterion name>", "score": <int>, "max": <int>, "justification": "<short reason>"}
  ]
}
One criterion object per rubric line. No prose outside the JSON.`, maxTotal, rb.String(), modelAnswer, studentAnswer)

	resp, err := o.call(ctx, orRequest{
		Model:          o.model,
		Messages:       []orMessage{{Role: "user", Content: prompt}},
		ResponseFormat: map[string]string{"type": "json_object"},
		Temperature:    0.2,
	})
	if err != nil {
		return out, err
	}
	if len(resp.Choices) == 0 {
		return out, fmt.Errorf("openrouter grade: empty response")
	}
	text := resp.Choices[0].Message.Content
	// Trim leading prose if the model added any.
	if i := strings.Index(text, "{"); i > 0 {
		text = text[i:]
	}
	if j := strings.LastIndex(text, "}"); j >= 0 {
		text = text[:j+1]
	}
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		return out, fmt.Errorf("parse grade json: %w (raw: %s)", err, text)
	}
	if out.MaxScore == 0 {
		out.MaxScore = maxTotal
	}
	return out, nil
}
