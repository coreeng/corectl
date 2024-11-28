package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHandleHello(t *testing.T) {
	recorder := httptest.NewRecorder()
	ctx, r := gin.CreateTestContext(recorder)
	setupHelloRoutes(r)
	req, err := http.NewRequestWithContext(ctx, "GET", "/hello", nil)
	if err != nil {
		t.Errorf("got error: %s", err)
	}
	r.ServeHTTP(recorder, req)
	if http.StatusOK != recorder.Code {
		t.Fatalf("expected response code %d, got %d", http.StatusOK, recorder.Code)
	}
	body := recorder.Body.String()
	if "Hello world" != body {
		t.Fatalf("expected body '%s', got %s", "Hello world", body)
	}
}