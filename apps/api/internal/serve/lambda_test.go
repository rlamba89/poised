package serve_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/rlamba89/poised/apps/api/internal/cognito"
	"github.com/rlamba89/poised/apps/api/internal/config"
	"github.com/rlamba89/poised/apps/api/internal/httpapi"
	"github.com/rlamba89/poised/apps/api/internal/messaging"
	"github.com/rlamba89/poised/apps/api/internal/serve"
)

// event is an API Gateway HTTP API (payload v2) request, as API Gateway sends it to the
// Lambda: the browser's cookies arrive in their own list, not as a Cookie header.
func event(method, path, sourceIP, forwardedFor string, cookies ...string) []byte {
	e := map[string]any{
		"version":        "2.0",
		"routeKey":       "ANY /api/{proxy+}",
		"rawPath":        path,
		"rawQueryString": "a=1&b=2",
		"cookies":        cookies,
		"headers": map[string]string{
			"host":            "abc123.execute-api.example.test",
			"x-forwarded-for": forwardedFor,
			"content-type":    "application/json",
		},
		"requestContext": map[string]any{
			"http": map[string]any{"method": method, "path": path, "sourceIp": sourceIP},
		},
		"body":            `{"hello":"world"}`,
		"isBase64Encoded": false,
	}
	if forwardedFor == "" {
		delete(e["headers"].(map[string]string), "x-forwarded-for")
	}
	b, _ := json.Marshal(e)
	return b
}

// response is the payload v2 response the Lambda returns to API Gateway.
type response struct {
	StatusCode int               `json:"statusCode"`
	Headers    map[string]string `json:"headers"`
	Cookies    []string          `json:"cookies"`
	Body       string            `json:"body"`
}

func invoke(t *testing.T, h http.Handler, hops int, payload []byte) response {
	t.Helper()
	out, err := serve.Lambda(h, hops).Invoke(context.Background(), payload)
	if err != nil {
		t.Fatal(err)
	}
	var r response
	if err := json.Unmarshal(out, &r); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	return r
}

// echo reports what the API's handlers would see.
var echo = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	cookie := func(name string) string {
		if c, err := r.Cookie(name); err == nil {
			return c.Value
		}
		return ""
	}
	body, _ := io.ReadAll(r.Body)
	http.SetCookie(w, &http.Cookie{Name: "one", Value: "1", Path: "/", HttpOnly: true})
	http.SetCookie(w, &http.Cookie{Name: "two", Value: "2", Path: "/api/p", HttpOnly: true})
	fmt.Fprintf(w, "%s %s?%s remote=%s a=%s b=%s body=%s", r.Method, r.URL.Path, r.URL.RawQuery, r.RemoteAddr, cookie("a"), cookie("b"), body)
})

func TestLambdaPassesTheRequestAndBothWaysOfCookies(t *testing.T) {
	r := invoke(t, echo, 0, event("POST", "/api/p/signin", "198.51.100.7", "", "a=apple", "b=banana"))
	if r.StatusCode != 200 {
		t.Fatalf("status %d: %s", r.StatusCode, r.Body)
	}
	want := `POST /api/p/signin?a=1&b=2 remote=198.51.100.7 a=apple b=banana body={"hello":"world"}`
	if r.Body != want {
		t.Errorf("handler saw\n %s\nwant\n %s", r.Body, want)
	}
	// Two Set-Cookie headers can't share one header value: payload v2 returns them as a list.
	if len(r.Cookies) != 2 || !strings.HasPrefix(r.Cookies[0], "one=1") || !strings.HasPrefix(r.Cookies[1], "two=2") {
		t.Errorf("cookies = %q", r.Cookies)
	}
}

func TestLambdaClientAddress(t *testing.T) {
	// Behind Amplify's rewrite and API Gateway: the browser's address, then each proxy's.
	const viaProxies = "203.0.113.9, 130.176.1.1"
	cases := []struct {
		name, xff string
		hops      int
		want      string
	}{
		{"no trusted proxies: the connection's address", viaProxies, 0, "198.51.100.7"},
		{"one hop: the last address added", viaProxies, 1, "130.176.1.1"},
		{"two hops: the browser behind both proxies", viaProxies, 2, "203.0.113.9"},
		{"a made-up address the browser sent first is ignored", "6.6.6.6, " + viaProxies, 2, "203.0.113.9"},
		{"too few addresses (not through the proxies): the connection's", "203.0.113.9", 2, "198.51.100.7"},
		{"no header: the connection's", "", 2, "198.51.100.7"},
		{"junk in the chosen place: the connection's", "not-an-ip, 130.176.1.1", 2, "198.51.100.7"},
		{"IPv6", "2001:db8::5, 130.176.1.1", 2, "2001:db8::5"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := invoke(t, echo, c.hops, event("GET", "/api/health", "198.51.100.7", c.xff))
			if !strings.Contains(r.Body, " remote="+c.want+" ") {
				t.Errorf("got %s; want remote=%s", r.Body, c.want)
			}
		})
	}
}

// The real patient router runs through the adapter: GET /api/config needs no database.
func TestLambdaServesTheRouter(t *testing.T) {
	cfg, err := config.Load(func(k string) string {
		return map[string]string{"DATABASE_URL": "postgres://unused", "TOKEN_SECRET": "x"}[k]
	})
	if err != nil {
		t.Fatal(err)
	}
	capture := messaging.Capture{} // never used: no route here sends anything
	h := httpapi.NewPatientRouter(httpapi.Deps{Config: cfg, Cognito: cognito.Local{}, SMS: capture, Email: capture})
	r := invoke(t, h, 0, event("GET", "/api/config", "198.51.100.7", ""))
	if r.StatusCode != 200 || !strings.Contains(r.Body, `"locale"`) {
		t.Errorf("status %d: %s", r.StatusCode, r.Body)
	}
	if r := invoke(t, h, 0, event("GET", "/api/me", "198.51.100.7", "")); r.StatusCode != 404 {
		t.Errorf("a staff route on the patient function: %d, want 404", r.StatusCode)
	}
}
