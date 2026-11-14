package httpapi

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"unicode/utf16"

	"github.com/google/uuid"
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

// S49a: a CSV cell holds 32,767 characters, so 862 ids ("; " between them) fit one and 863 don't.
func TestCellParts(t *testing.T) {
	ids := func(n int) []string {
		out := make([]string, n)
		for i := range out {
			out[i] = uuid.NewString()
		}
		return out
	}
	if got := cellParts(ids(0)); len(got) != 0 {
		t.Errorf("no ids: %v", got)
	}
	if got := cellParts(ids(862)); len(got) != 1 || got[0] != [2]int{0, 862} {
		t.Errorf("862 ids: %v", got)
	}
	if got := cellParts(ids(863)); len(got) != 2 || got[0] != [2]int{0, 862} || got[1] != [2]int{862, 863} {
		t.Errorf("863 ids: %v", got)
	}
	// Two columns side by side (names and ids): a run ends when either would pass the limit.
	names := make([]string, 100)
	for i := range names {
		names[i] = strings.Repeat("é", 1000) // characters, not bytes
	}
	got := cellParts(names, ids(100))
	if len(got) != 4 || got[0] != [2]int{0, 32} || got[3] != [2]int{96, 100} {
		t.Errorf("long names: %v", got)
	}
	for _, p := range got {
		if n := len([]rune(strings.Join(names[p[0]:p[1]], "; "))); n > csvCellMax-1 {
			t.Errorf("part %v is %d characters", p, n)
		}
	}
	// A spreadsheet counts UTF-16 units: a character outside the Basic Multilingual Plane (an emoji,
	// some CJK) is two (S49a verification 1, finding 3). 2,000 units a name: 16 names a part.
	wide := make([]string, 100)
	for i := range wide {
		wide[i] = strings.Repeat("😀", 1000)
	}
	got = cellParts(wide)
	if len(got) != 7 || got[0] != [2]int{0, 16} || got[6] != [2]int{96, 100} {
		t.Errorf("wide names: %v", got)
	}
	for _, p := range got {
		if n := len(utf16.Encode([]rune(strings.Join(wide[p[0]:p[1]], "; ")))); n > csvCellMax-1 {
			t.Errorf("part %v is %d UTF-16 units", p, n)
		}
	}
}
