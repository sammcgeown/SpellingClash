package handlers

import (
	"bytes"
	"errors"
	"log"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRespondWithErrorWritesStatusAndBody(t *testing.T) {
	recorder := httptest.NewRecorder()

	respondWithError(recorder, 418, "Teapot", "", nil)

	if recorder.Code != 418 {
		t.Fatalf("expected status 418, got %d", recorder.Code)
	}

	body := strings.TrimSpace(recorder.Body.String())
	if body != "Teapot" {
		t.Fatalf("expected body 'Teapot', got %q", body)
	}
}

func TestRespondWithErrorLogsMessage(t *testing.T) {
	var buf bytes.Buffer
	logger := log.Default()
	originalOutput := logger.Writer()
	logger.SetOutput(&buf)
	defer logger.SetOutput(originalOutput)

	recorder := httptest.NewRecorder()
	err := errors.New("boom")

	respondWithError(recorder, 500, "Internal server error", "", err)

	logOutput := buf.String()
	if !strings.Contains(logOutput, "Internal server error") {
		t.Fatalf("expected log to include user message, got %q", logOutput)
	}
	if !strings.Contains(logOutput, "boom") {
		t.Fatalf("expected log to include error, got %q", logOutput)
	}
}
