package main

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"smartedu/internal/docx"
	"smartedu/internal/llm"
	"smartedu/internal/store"
)

func TestUploadDocumentPersistsParsedDocxText(t *testing.T) {
	srv := &server{repo: store.NewMemory()}
	docxPath := filepath.Join("..", "HTML_Asoslari_Dars_Materiali.docx")
	raw, err := os.ReadFile(docxPath)
	if err != nil {
		t.Fatalf("read docx: %v", err)
	}
	expectedText, err := docx.ExtractText(raw)
	if err != nil {
		t.Fatalf("extract docx text: %v", err)
	}
	if strings.TrimSpace(expectedText) == "" {
		t.Fatalf("expected parsed docx text to be non-empty")
	}

	req := newMultipartUploadRequest(t, raw, "HTML_Asoslari_Dars_Materiali.docx")
	rr := httptest.NewRecorder()

	srv.handleUploadDocument(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("unexpected status: got %d body=%s", rr.Code, rr.Body.String())
	}

	var created store.Document
	if err := json.Unmarshal(rr.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if created.ID == "" {
		t.Fatalf("expected created document id")
	}
	if created.FileName != "HTML_Asoslari_Dars_Materiali.docx" {
		t.Fatalf("unexpected filename: %q", created.FileName)
	}
	if created.ParsedText != expectedText {
		t.Fatalf("parsed text mismatch\nexpected:\n%s\n\ngot:\n%s", expectedText, created.ParsedText)
	}

	stored, err := srv.repo.GetDocument(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("repo get document: %v", err)
	}
	if stored.ParsedText != expectedText {
		t.Fatalf("stored parsed text mismatch")
	}
	if stored.Title != "HTML asoslari" {
		t.Fatalf("expected uploaded title to persist, got %q", stored.Title)
	}
}

func TestUploadThenGenerateScenarioLinksDocument(t *testing.T) {
	repo := store.NewMemory()
	mock := llm.NewMock()
	srv := &server{
		repo:             repo,
		teacherProvider:  mock,
		studentProvider:  mock,
		documentProvider: mock,
	}

	raw, err := os.ReadFile(filepath.Join("..", "HTML_Asoslari_Dars_Materiali.docx"))
	if err != nil {
		t.Fatalf("read docx: %v", err)
	}

	uploadReq := newMultipartUploadRequest(t, raw, "HTML_Asoslari_Dars_Materiali.docx")
	uploadRR := httptest.NewRecorder()
	srv.handleUploadDocument(uploadRR, uploadReq)
	if uploadRR.Code != http.StatusCreated {
		t.Fatalf("upload failed: status=%d body=%s", uploadRR.Code, uploadRR.Body.String())
	}

	var created store.Document
	if err := json.Unmarshal(uploadRR.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode upload response: %v", err)
	}

	genReq := httptest.NewRequest(http.MethodPost, "/api/teacher/documents/"+created.ID+"/generate-scenario", bytes.NewReader([]byte(`{}`)))
	genRR := httptest.NewRecorder()
	srv.handleGenerateScenarioFromDoc(genRR, genReq)

	if genRR.Code != http.StatusCreated {
		t.Fatalf("generate scenario failed: status=%d body=%s", genRR.Code, genRR.Body.String())
	}

	var payload struct {
		Document store.Document `json:"document"`
		Scenario map[string]any `json:"scenario"`
	}
	if err := json.Unmarshal(genRR.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode generate response: %v", err)
	}
	if payload.Scenario["id"] == "" {
		t.Fatalf("expected generated scenario id")
	}

	stored, err := repo.GetDocument(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("repo get document: %v", err)
	}
	if stored.ScenarioID == "" {
		t.Fatalf("expected repo document scenario_id to be persisted")
	}
	if payload.Document.ScenarioID != "" {
		t.Fatalf("expected response document snapshot to remain original, got scenario_id=%q", payload.Document.ScenarioID)
	}

	sc, err := repo.GetScenario(context.Background(), stored.ScenarioID)
	if err != nil {
		t.Fatalf("repo get scenario: %v", err)
	}
	if sc == nil || sc.Title == "" {
		t.Fatalf("expected generated scenario to exist")
	}
}

func newMultipartUploadRequest(t *testing.T, fileBytes []byte, filename string) *http.Request {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	fileWriter, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := fileWriter.Write(fileBytes); err != nil {
		t.Fatalf("write form file: %v", err)
	}

	for k, v := range map[string]string{
		"instruction":   "Create a scenario from this material.",
		"title":         "HTML asoslari",
		"subject":       "Web Development",
		"language":      "uz",
		"code_language": "python",
		"problem_focus": "HTML semantics",
	} {
		if err := writer.WriteField(k, v); err != nil {
			t.Fatalf("write form field %s: %v", k, err)
		}
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/teacher/documents", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}
