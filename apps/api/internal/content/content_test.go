package content

import (
	"encoding/json"
	"os"
	"slices"
	"strings"
	"testing"
)

// The video link cases are a file, so packages/clinical's check (S34) runs the same ones.
func TestCheckVideo(t *testing.T) {
	raw, err := os.ReadFile("testdata/video_links.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases struct {
		Accepted []struct{ URL, Provider, ID, Hash string }
		Refused  []string
	}
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatal(err)
	}
	if len(cases.Accepted) == 0 || len(cases.Refused) == 0 {
		t.Fatal("no cases")
	}
	for _, c := range cases.Accepted {
		got, ok := CheckVideo(c.URL)
		if !ok || got.Provider != c.Provider || got.ID != c.ID || got.Hash != c.Hash {
			t.Errorf("CheckVideo(%q) = %+v, %v; want %s %s %q", c.URL, got, ok, c.Provider, c.ID, c.Hash)
		}
	}
	for _, u := range cases.Refused {
		if got, ok := CheckVideo(u); ok {
			t.Errorf("CheckVideo(%q) = %+v, want it refused", u, got)
		}
	}
}

func TestCheckLeaflet(t *testing.T) {
	for _, u := range []string{
		"https://www.trust-a.example/leaflets/fasting.pdf",
		"https://example.org/download?id=12",
		"  https://example.org/a.pdf ",
		"HTTPS://example.org/a.pdf", // a phone keyboard's capital (verification 1)
		"Https://Example.org/a.pdf",
	} {
		if !CheckLeaflet(u) {
			t.Errorf("CheckLeaflet(%q) = false, want true", u)
		}
	}
	for _, u := range []string{
		"", "example.org/a.pdf", "http://example.org/a.pdf", "https://", "https:///a.pdf", "javascript:alert(1)",
		"https://user:pw@example.org/a.pdf", "ftp://example.org/a.pdf", "https://exa mple.org/a.pdf",
		"https://example.org/" + strings.Repeat("a", MaxURL),
	} {
		if CheckLeaflet(u) {
			t.Errorf("CheckLeaflet(%q) = true, want false", u)
		}
	}
}

// PX-13's checklist: the topics are offered as ready-made tags, and a topic is covered once an
// item has its tag, in any case.
func TestChecklist(t *testing.T) {
	if len(Checklist) < 10 {
		t.Fatalf("Checklist has %d topics, want PX-13's ten or more", len(Checklist))
	}
	seen := map[string]bool{}
	for _, topic := range Checklist {
		if topic == "" || seen[strings.ToLower(topic)] {
			t.Errorf("topic %q is empty or repeated", topic)
		}
		seen[strings.ToLower(topic)] = true
	}
	got := Covered([][]string{{"fasting", "Knee"}, {"Who to call"}, {}})
	var covered []string
	for _, c := range got {
		if c.Covered {
			covered = append(covered, c.Topic)
		}
	}
	if !slices.Equal(covered, []string{"Fasting", "Who to call"}) || len(got) != len(Checklist) {
		t.Errorf("covered = %v of %d, want Fasting and Who to call", covered, len(got))
	}
}

func TestTags(t *testing.T) {
	got, msg := Tags([]string{" Fasting ", "fasting", "", "Knee"})
	if msg != "" || !slices.Equal(got, []string{"Fasting", "Knee"}) {
		t.Errorf("Tags = %v %q", got, msg)
	}
	if got, msg := Tags(nil); msg != "" || got == nil || len(got) != 0 {
		t.Errorf("Tags(nil) = %#v %q, want an empty list", got, msg)
	}
	if _, msg := Tags([]string{strings.Repeat("x", MaxTag+1)}); msg == "" {
		t.Error("a long tag was accepted")
	}
	many := make([]string, MaxTags+1)
	for i := range many {
		many[i] = strings.Repeat("t", i+1)
	}
	if _, msg := Tags(many); msg == "" {
		t.Error("too many tags were accepted")
	}
}

// Link is the link as stored: trimmed, with the scheme and host in lower case, so a phone
// keyboard's "Https://" is kept as "https://" (verification 1: it broke the table's CHECK).
func TestLink(t *testing.T) {
	for in, want := range map[string]string{
		"  HTTPS://WWW.YouTube.com/watch?v=aqz-KE-bpKQ ": "https://www.youtube.com/watch?v=aqz-KE-bpKQ",
		"Https://Example.org/Leaflets/A.pdf?X=1#Top":     "https://example.org/Leaflets/A.pdf?X=1#Top",
		"https://vimeo.com/76979871/8272103f6e":          "https://vimeo.com/76979871/8272103f6e",
	} {
		if got, ok := Link(in); !ok || got != want {
			t.Errorf("Link(%q) = %q, %v; want %q", in, got, ok, want)
		}
	}
	for _, in := range []string{"", "http://example.org/a.pdf", "HTTP://example.org", "javascript:alert(1)", "https://user@example.org/"} {
		if got, ok := Link(in); ok {
			t.Errorf("Link(%q) = %q, want it refused", in, got)
		}
	}
}
