package selfcheck_malformed_response_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"seed-germination-workbench/internal/selfcheck"
	"testing"
)

func TestSelfcheckReturnsErrorForMalformedSuccessResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		_, _ = rw.Write([]byte("{}"))
	}))
	defer server.Close()
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("Runner.Run panicked instead of returning a response-contract error: %v", recovered)
		}
	}()

	err := (selfcheck.Runner{Base: server.URL, Client: server.Client()}).Run(context.Background())
	if err == nil {
		t.Fatal("Runner.Run accepted a malformed successful response")
	}
}
