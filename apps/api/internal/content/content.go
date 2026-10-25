// Package content holds the content library's rules (plan C5, session S33; INF-05, PX-13, A-6):
// which links a video or a leaflet may have, the tags, and the PX-13 checklist.
//
// Videos live on YouTube or Vimeo only, never on our servers (A-6), so a video's link must be one
// of their forms (CheckVideo). A leaflet is a link to a PDF on the web until plan C8 brings
// uploads (S33 answers 2). packages/clinical gets the same video check in S34; both run the cases
// in testdata/video_links.json.
package content

import (
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strings"
)

const (
	MaxURL   = 2000 // characters in a link
	MaxTitle = 200
	MaxTags  = 10
	MaxTag   = 40
	// MaxLength is a video's longest length, in seconds (4 hours): longer is surely a typing mistake.
	MaxLength = 4 * 60 * 60
)

// Checklist is PX-13's list of what a patient should be told before an operation. Its topics are
// offered as ready-made tags, and the library pages show which have no item yet (S33 answers 3).
var Checklist = []string{
	"Appointment details",
	"Fasting",
	"Medicines",
	"What to bring",
	"Screening",
	"Prehab",
	"Stopping smoking",
	"Consent",
	"The anaesthetic",
	"Recovery",
	"Who to call",
}

// Topic is one checklist topic and whether any item has its tag.
type Topic struct {
	Topic   string `json:"topic"`
	Covered bool   `json:"covered"`
}

// Covered is the checklist against the tags of the items that count (each item's tags), in any case.
func Covered(itemTags [][]string) []Topic {
	out := make([]Topic, len(Checklist))
	for i, topic := range Checklist {
		out[i] = Topic{Topic: topic}
		for _, tags := range itemTags {
			if slices.ContainsFunc(tags, func(t string) bool { return strings.EqualFold(t, topic) }) {
				out[i].Covered = true
				break
			}
		}
	}
	return out
}

// Tags trims the tags and drops empty and repeated ones (in any case); or a plain-language problem.
func Tags(in []string) ([]string, string) {
	out := []string{}
	for _, t := range in {
		t = strings.TrimSpace(t)
		if t == "" || slices.ContainsFunc(out, func(o string) bool { return strings.EqualFold(o, t) }) {
			continue
		}
		if len([]rune(t)) > MaxTag {
			return nil, fmt.Sprintf("A tag is too long (%d characters at most).", MaxTag)
		}
		out = append(out, t)
	}
	if len(out) > MaxTags {
		return nil, fmt.Sprintf("Too many tags (%d at most).", MaxTags)
	}
	return out, ""
}

// Video is what a video link plays: its provider ("youtube" or "vimeo"), the video's id, and
// for a private Vimeo link its hash.
type Video struct {
	Provider string `json:"provider"`
	ID       string `json:"id"`
	Hash     string `json:"hash,omitempty"`
}

var (
	youtubeID = regexp.MustCompile(`^[A-Za-z0-9_-]{11}$`)
	vimeoID   = regexp.MustCompile(`^[0-9]{1,12}$`)
	vimeoHash = regexp.MustCompile(`^[0-9a-f]{6,20}$`)
)

// CheckVideo reads a YouTube or Vimeo link, in one of the forms people copy:
//   - YouTube: youtube.com/watch?v=<id> (www., m. or none), youtu.be/<id>, youtube.com/embed/<id>
//     and youtube-nocookie.com/embed/<id>;
//   - Vimeo: vimeo.com/<id>, the private link vimeo.com/<id>/<hash>, and its player's
//     player.vimeo.com/video/<id> (with ?h=<hash> when private).
//
// Only https, with no user name or port. ok is false for anything else.
func CheckVideo(raw string) (v Video, ok bool) {
	u, ok := httpsURL(raw)
	if !ok {
		return Video{}, false
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if u.Path == "" || u.Path == "/" {
		parts = nil
	}
	switch strings.TrimPrefix(u.Host, "www.") {
	case "youtube.com", "m.youtube.com":
		if u.Path == "/watch" {
			v = Video{Provider: "youtube", ID: u.Query().Get("v")}
		} else if len(parts) == 2 && parts[0] == "embed" {
			v = Video{Provider: "youtube", ID: parts[1]}
		}
	case "youtube-nocookie.com":
		if len(parts) == 2 && parts[0] == "embed" {
			v = Video{Provider: "youtube", ID: parts[1]}
		}
	case "youtu.be":
		if len(parts) == 1 {
			v = Video{Provider: "youtube", ID: parts[0]}
		}
	case "vimeo.com":
		if len(parts) == 1 || len(parts) == 2 {
			v = Video{Provider: "vimeo", ID: parts[0]}
		}
		if len(parts) == 2 {
			v.Hash = parts[1]
			if !vimeoHash.MatchString(v.Hash) {
				return Video{}, false
			}
		}
	case "player.vimeo.com":
		if len(parts) == 2 && parts[0] == "video" {
			v = Video{Provider: "vimeo", ID: parts[1], Hash: u.Query().Get("h")}
			if v.Hash != "" && !vimeoHash.MatchString(v.Hash) {
				return Video{}, false
			}
		}
	}
	switch v.Provider {
	case "youtube":
		return v, youtubeID.MatchString(v.ID)
	case "vimeo":
		return v, vimeoID.MatchString(v.ID)
	}
	return Video{}, false
}

// CheckLeaflet says whether a leaflet's link is a web address we may send a patient to: https,
// with a host, and no user name or password.
func CheckLeaflet(raw string) bool {
	_, ok := httpsURL(raw)
	return ok
}

// Link is a link as it's stored: trimmed, with its scheme and host in lower case (a phone keyboard
// types "Https://"), or ok false if it isn't an https link CheckLeaflet would accept.
func Link(raw string) (string, bool) {
	u, ok := httpsURL(raw)
	if !ok {
		return "", false
	}
	return u.String(), true
}

// httpsURL parses an https link with a host and no user name, password or port, lowercasing the
// host. The scheme is matched in any case (url.Parse lowercases it).
func httpsURL(raw string) (*url.URL, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" || len(raw) > MaxURL || strings.ContainsAny(raw, " \t\r\n") {
		return nil, false
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.Port() != "" || u.Opaque != "" {
		return nil, false
	}
	u.Host = strings.ToLower(u.Host)
	return u, true
}
