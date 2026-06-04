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

const defaultGeminiModel = "gemini-3.5-flash"

// ModelName returns the effective model name (default if empty).
func ModelName(model string) string {
	if model == "" {
		return defaultGeminiModel
	}
	return model
}

// Gemini implements Provider via the Google Generative Language REST API.
type Gemini struct {
	apiKey string
	model  string
	http   *http.Client
}

// NewGemini builds a Gemini provider. Empty model => defaultGeminiModel.
func NewGemini(apiKey, model string) *Gemini {
	return &Gemini{
		apiKey: apiKey,
		model:  ModelName(model),
		http:   &http.Client{Timeout: 60 * time.Second},
	}
}

// ---- REST payload shapes (minimal subset) ----

type gContent struct {
	Role  string  `json:"role,omitempty"`
	Parts []gPart `json:"parts"`
}

type gPart struct {
	Text             string         `json:"text,omitempty"`
	FunctionCall     *gFunctionCall `json:"functionCall,omitempty"`
	FunctionResponse *gFunctionResp `json:"functionResponse,omitempty"`
	// ThoughtSignature — Gemini 3.x returns this on function-call parts; it MUST
	// be echoed back unchanged in the next request or tool calls are rejected.
	// https://ai.google.dev/gemini-api/docs/thought-signatures
	ThoughtSignature string `json:"thoughtSignature,omitempty"`
}

type gFunctionCall struct {
	Name string                 `json:"name"`
	Args map[string]interface{} `json:"args"`
}

type gFunctionResp struct {
	Name     string                 `json:"name"`
	Response map[string]interface{} `json:"response"`
}

type gTool struct {
	FunctionDeclarations []gFuncDecl `json:"functionDeclarations"`
}

type gFuncDecl struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters"`
}

type gGenConfig struct {
	ResponseMimeType string      `json:"responseMimeType,omitempty"`
	ResponseSchema   interface{} `json:"responseSchema,omitempty"`
	Temperature      float64     `json:"temperature,omitempty"`
}

type gRequest struct {
	SystemInstruction *gContent   `json:"systemInstruction,omitempty"`
	Contents          []gContent  `json:"contents"`
	Tools             []gTool     `json:"tools,omitempty"`
	GenerationConfig  *gGenConfig `json:"generationConfig,omitempty"`
}

type gResponse struct {
	Candidates []struct {
		Content gContent `json:"content"`
	} `json:"candidates"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (g *Gemini) call(ctx context.Context, body gRequest) (*gResponse, error) {
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent", g.model)
	buf, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(buf))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", g.apiKey) // docs-preferred auth (vs ?key= query param)
	resp, err := g.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var out gResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("decode gemini response: %w (raw: %s)", err, string(raw))
	}
	if out.Error != nil {
		return nil, fmt.Errorf("gemini error: %s", out.Error.Message)
	}
	return &out, nil
}

// Chat runs the conversation with the get_fact tool wired in. The model can
// only obtain scenario numbers by calling get_fact; code answers it from the
// FactsStore so the model can never fabricate a value (spec 3.1).
func (g *Gemini) Chat(ctx context.Context, req ChatRequest) (string, error) {
	tools := []gTool{{
		FunctionDeclarations: []gFuncDecl{{
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
		}},
	}}

	contents := make([]gContent, 0, len(req.History)+1)
	for _, m := range req.History {
		role := "user"
		if m.Role == "assistant" {
			role = "model"
		}
		contents = append(contents, gContent{Role: role, Parts: []gPart{{Text: m.Content}}})
	}
	contents = append(contents, gContent{Role: "user", Parts: []gPart{{Text: req.UserMessage}}})

	sys := &gContent{Parts: []gPart{{Text: req.SystemPrompt}}}

	// Tool loop: resolve any get_fact calls until the model returns plain text.
	for i := 0; i < 5; i++ {
		resp, err := g.call(ctx, gRequest{
			SystemInstruction: sys,
			Contents:          contents,
			Tools:             tools,
			GenerationConfig:  &gGenConfig{Temperature: 0.7},
		})
		if err != nil {
			return "", err
		}
		if len(resp.Candidates) == 0 {
			return "", fmt.Errorf("gemini returned no candidates")
		}
		parts := resp.Candidates[0].Content.Parts

		var calls []*gFunctionCall
		var text strings.Builder
		for _, p := range parts {
			if p.FunctionCall != nil {
				calls = append(calls, p.FunctionCall)
			}
			if p.Text != "" {
				text.WriteString(p.Text)
			}
		}

		if len(calls) == 0 {
			return strings.TrimSpace(text.String()), nil
		}

		// Echo the model's function-call turn, then append our tool results.
		contents = append(contents, gContent{Role: "model", Parts: parts})
		respParts := make([]gPart, 0, len(calls))
		for _, c := range calls {
			key, _ := c.Args["key"].(string)
			fact := req.Facts.Get(key)
			respParts = append(respParts, gPart{FunctionResponse: &gFunctionResp{
				Name:     "get_fact",
				Response: map[string]interface{}{"value": fact.Value, "found": fact.Found},
			}})
		}
		contents = append(contents, gContent{Role: "user", Parts: respParts})
	}
	return "", fmt.Errorf("get_fact tool loop exceeded limit")
}

// Grade returns structured rubric scoring. Uses Gemini structured output
// (responseSchema) to force valid JSON (spec 3.2 + demo risk 10).
func (g *Gemini) Grade(ctx context.Context, modelAnswer, studentAnswer string, rubric []Criterion) (GradeResult, error) {
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

Return one criterion object per rubric line, with an honest justification.`, maxTotal, rb.String(), modelAnswer, studentAnswer)

	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"total_score": map[string]interface{}{"type": "integer"},
			"max_score":   map[string]interface{}{"type": "integer"},
			"criteria": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"name":          map[string]interface{}{"type": "string"},
						"score":         map[string]interface{}{"type": "integer"},
						"max":           map[string]interface{}{"type": "integer"},
						"justification": map[string]interface{}{"type": "string"},
					},
					"required": []string{"name", "score", "max", "justification"},
				},
			},
		},
		"required": []string{"total_score", "max_score", "criteria"},
	}

	resp, err := g.call(ctx, gRequest{
		Contents: []gContent{{Role: "user", Parts: []gPart{{Text: prompt}}}},
		GenerationConfig: &gGenConfig{
			ResponseMimeType: "application/json",
			ResponseSchema:   schema,
			Temperature:      0.2,
		},
	})
	if err != nil {
		return out, err
	}
	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return out, fmt.Errorf("gemini grade: empty response")
	}
	jsonText := resp.Candidates[0].Content.Parts[0].Text
	if err := json.Unmarshal([]byte(jsonText), &out); err != nil {
		return out, fmt.Errorf("parse grade json: %w (raw: %s)", err, jsonText)
	}
	if out.MaxScore == 0 {
		out.MaxScore = maxTotal
	}
	return out, nil
}
