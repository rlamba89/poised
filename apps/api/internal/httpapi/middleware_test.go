package httpapi

import (
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// A handler that panics gets the plain-language 500, not a dropped connection (S02).
func TestRecoverPanics(t *testing.T) {
	h := recoverPanics(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		var m map[string]int
		m["boom"]++ // a nil map write panics
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/api/anything", nil))

	if rec.Code != http.StatusInternalServerError || !strings.Contains(rec.Body.String(), "Something went wrong") {
		t.Errorf("got %d %s, want 500 with the plain-language message", rec.Code, rec.Body)
	}
}

// http.ErrAbortHandler is the standard way to abort a response on purpose, so it is passed on.
func TestRecoverPanicsPassesOnAbort(t *testing.T) {
	h := recoverPanics(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic(http.ErrAbortHandler) }))
	defer func() {
		if recover() != http.ErrAbortHandler {
			t.Error("http.ErrAbortHandler was swallowed")
		}
	}()
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
}

// The log names the route, not the path: a patient's path holds their link token, which is
// their only key to the HQ (code review, S02).
func TestRecoverPanicsKeepsTokensOutOfTheLog(t *testing.T) {
	var logged strings.Builder
	log.SetOutput(&logged)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/p/{token}", func(http.ResponseWriter, *http.Request) { panic("boom") })
	const token = "SECRETtokenSECRETtokenSECRETtokenSECRETtok"

	rec := httptest.NewRecorder()
	recoverPanics(mux).ServeHTTP(rec, httptest.NewRequest("GET", "/api/p/"+token, nil))

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("got %d, want 500", rec.Code)
	}
	if strings.Contains(logged.String(), "SECRETtoken") || !strings.Contains(logged.String(), "GET /api/p/{token}") {
		t.Errorf("log = %q, want the route pattern and no token", logged.String())
	}
}

// A handler that panics after it has started its response can't be given a 500 any more, so
// the response is aborted, not sent with an error appended (code review, S02).
func TestRecoverPanicsAbortsAStartedResponse(t *testing.T) {
	log.SetOutput(io.Discard)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
	h := recoverPanics(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
		panic("boom")
	}))
	rec := httptest.NewRecorder()
	defer func() {
		if recover() != http.ErrAbortHandler {
			t.Error("the started response wasn't aborted")
		}
		if rec.Code != http.StatusNoContent || rec.Body.Len() != 0 {
			t.Errorf("got %d %q: an error was written on top of the response", rec.Code, rec.Body)
		}
	}()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
}
