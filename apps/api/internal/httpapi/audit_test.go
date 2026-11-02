package httpapi

import (
	"context"
	"net/http"
	"testing"
)

func TestAuditStatus(t *testing.T) {
	gone, cancel := context.WithCancel(context.Background())
	cancel()
	timedOut, stop := context.WithTimeout(context.Background(), 0)
	defer stop()
	for _, c := range []struct {
		name string
		ctx  context.Context
		code int
		want int
	}{
		{"answered", context.Background(), http.StatusOK, http.StatusOK},
		{"a real failure", context.Background(), http.StatusInternalServerError, http.StatusInternalServerError},
		{"the caller left mid-request", gone, http.StatusInternalServerError, statusCallerLeft},
		{"the caller left after it was done: still done", gone, http.StatusOK, http.StatusOK},
		{"the caller left after a refusal: still refused", gone, http.StatusConflict, http.StatusConflict},
		{"a timeout is our failure, not the caller leaving", timedOut, http.StatusInternalServerError, http.StatusInternalServerError},
	} {
		if got := auditStatus(c.ctx, c.code); got != c.want {
			t.Errorf("%s: %d, want %d", c.name, got, c.want)
		}
	}
}
