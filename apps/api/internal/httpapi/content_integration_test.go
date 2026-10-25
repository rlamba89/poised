//go:build integration

package httpapi_test

import (
	"net/http"
	"net/url"
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/rlamba89/poised/apps/api/internal/apitest"
)

// The content library (plan C5, session S33; INF-05, PX-13, A-6):
//   - /api/admin/content: Preopsis's videos and leaflets (org_id NULL), platform admins only;
//   - /api/o/{oid}/content: the trust's own, changed by its whole-trust admins, read by them and its
//     whole-trust super clinicians, with Preopsis's active items shown alongside;
//   - /api/o/{oid}/content/search: what an author may pick (S34): active items, Preopsis's and the
//     trust's own;
//   - /api/admin/library-questions: the platform's starter questions, which a platform admin
//     retires or offers again (S33 answers 4).
// No patient data, so none of it is audited.

type contentItem struct {
	ID            uuid.UUID
	Platform      bool
	Kind          string
	Title         string
	URL           string `json:"url"`
	LengthSeconds *int
	Tags          []string
	Status        string
	UpdatedByName string
}

type contentList struct {
	Items     []contentItem
	Checklist []struct {
		Topic   string
		Covered bool
	}
}

// contentWorld is trust A (Hospital A1) and trust B, a platform admin, and trust A's people.
type contentWorld struct {
	c                    *apitest.Client // not signed in
	fx                   *apitest.Fixtures
	a1, b1               apitest.Hospital
	sam, ada, alex       *apitest.Client // the platform admin; A's admin; A's whole-trust super clinician
	cara, sue, trustClin *apitest.Client // a clinician at A1; a super clinician of A1 only; a whole-trust clinician of A
	bAdmin, bAuthor      *apitest.Client // trust B's admin and whole-trust super clinician
	samUser              apitest.User
}

func newContentWorld(t *testing.T) *contentWorld {
	t.Helper()
	c, fx := apitest.New(t)
	w := &contentWorld{c: c, fx: fx, a1: fx.Hospital(t), b1: fx.Hospital(t)}
	w.samUser = fx.PlatformAdmin(t)
	w.sam = c.LoginAs(t, w.samUser)
	w.ada = c.LoginAs(t, fx.TrustUser(t, w.a1.Org, "admin"))
	w.alex = c.LoginAs(t, fx.TrustUser(t, w.a1.Org, "super_clinician"))
	w.cara = c.LoginAs(t, fx.User(t, w.a1, "clinician"))
	w.sue = c.LoginAs(t, fx.User(t, w.a1, "super_clinician"))
	w.trustClin = c.LoginAs(t, fx.TrustUser(t, w.a1.Org, "clinician"))
	w.bAdmin = c.LoginAs(t, fx.TrustUser(t, w.b1.Org, "admin"))
	w.bAuthor = c.LoginAs(t, fx.TrustUser(t, w.b1.Org, "super_clinician"))
	return w
}

const (
	adminContent = "/api/admin/content"
	ytLink       = "https://www.youtube.com/watch?v=aqz-KE-bpKQ"
	vimeoLink    = "https://vimeo.com/76979871"
	leafletLink  = "https://www.trust-a.example/leaflets/fasting.pdf"
)

func contentPath(org uuid.UUID, p string) string { return orgPath(org, "/content"+p) }

func video(title, link string, seconds int, tags ...string) map[string]any {
	if tags == nil {
		tags = []string{}
	}
	return map[string]any{"kind": "video", "title": title, "url": link, "lengthSeconds": seconds, "tags": tags}
}

func leaflet(title, link string, tags ...string) map[string]any {
	if tags == nil {
		tags = []string{}
	}
	return map[string]any{"kind": "leaflet", "title": title, "url": link, "tags": tags}
}

// addContent posts an item and returns its id.
func addContent(t *testing.T, c *apitest.Client, path string, body map[string]any) uuid.UUID {
	t.Helper()
	r := c.Do("POST", path, body)
	r.Want(t, http.StatusCreated)
	var got struct{ ID uuid.UUID }
	r.Decode(t, &got)
	return got.ID
}

func getContent(t *testing.T, c *apitest.Client, path string) contentList {
	t.Helper()
	r := c.Do("GET", path, nil)
	r.Want(t, http.StatusOK)
	var got contentList
	r.Decode(t, &got)
	return got
}

func searchContent(t *testing.T, c *apitest.Client, org uuid.UUID, tag, q string) []contentItem {
	t.Helper()
	r := c.Do("GET", contentPath(org, "/search?tag="+url.QueryEscape(tag)+"&q="+url.QueryEscape(q)), nil)
	r.Want(t, http.StatusOK)
	var got []contentItem
	r.Decode(t, &got)
	return got
}

func contentTitles(items []contentItem) []string {
	out := []string{}
	for _, i := range items {
		out = append(out, i.Title)
	}
	return out
}

func findContent(items []contentItem, title string) *contentItem {
	for i := range items {
		if items[i].Title == title {
			return &items[i]
		}
	}
	return nil
}

func covered(l contentList, topic string) bool {
	for _, c := range l.Checklist {
		if c.Topic == topic {
			return c.Covered
		}
	}
	return false
}

func TestAdminContent(t *testing.T) {
	t.Parallel()
	w := newContentWorld(t)

	t.Run("happy path: a platform admin adds a video and a leaflet, edits one and retires one", func(t *testing.T) {
		id := addContent(t, w.sam, adminContent, video("Your anaesthetic", ytLink, 200, "The anaesthetic", " anaesthetic ", "The ANAESTHETIC"))
		leaf := addContent(t, w.sam, adminContent, leaflet("Fasting before surgery", leafletLink, "Fasting"))
		var orgSet bool
		var kind, status string
		var length *int
		var tags []string
		var createdBy uuid.UUID
		if err := w.c.DB.QueryRow(t.Context(), `SELECT org_id IS NOT NULL, kind, status, length_seconds, tags, created_by FROM content_items WHERE id = $1`, id).
			Scan(&orgSet, &kind, &status, &length, &tags, &createdBy); err != nil {
			t.Fatal(err)
		}
		if orgSet || kind != "video" || status != "active" || length == nil || *length != 200 || !slices.Equal(tags, []string{"The anaesthetic", "anaesthetic"}) || createdBy != w.samUser.ID {
			t.Errorf("row = org %v %s %s %v %q by %s", orgSet, kind, status, length, tags, createdBy)
		}
		got := getContent(t, w.sam, adminContent)
		if titles := contentTitles(got.Items); !slices.Equal(titles, []string{"Fasting before surgery", "Your anaesthetic"}) {
			t.Errorf("items = %q", titles)
		}
		if l := findContent(got.Items, "Fasting before surgery"); l == nil || !l.Platform || l.Kind != "leaflet" || l.LengthSeconds != nil || l.URL != leafletLink || l.UpdatedByName == "" {
			t.Errorf("leaflet = %+v", l)
		}
		if !covered(got, "Fasting") || !covered(got, "The anaesthetic") || covered(got, "Who to call") {
			t.Errorf("checklist = %+v", got.Checklist)
		}

		w.sam.Do("PATCH", adminContent+"/"+id.String(), map[string]any{"title": "Your anaesthetic explained", "url": vimeoLink, "lengthSeconds": 95, "tags": []string{"Recovery"}}).
			Want(t, http.StatusNoContent)
		w.sam.Do("PATCH", adminContent+"/"+leaf.String(), map[string]any{"status": "retired"}).Want(t, http.StatusNoContent)
		got = getContent(t, w.sam, adminContent)
		v := findContent(got.Items, "Your anaesthetic explained")
		if v == nil || v.URL != vimeoLink || v.LengthSeconds == nil || *v.LengthSeconds != 95 || !slices.Equal(v.Tags, []string{"Recovery"}) || v.Status != "active" {
			t.Errorf("edited video = %+v", v)
		}
		if titles := contentTitles(got.Items); !slices.Equal(titles, []string{"Your anaesthetic explained", "Fasting before surgery"}) {
			t.Errorf("items = %q, want the retired one last", titles)
		}
		if covered(got, "Fasting") {
			t.Error("a retired leaflet still covers Fasting")
		}
		// …and brings it back.
		w.sam.Do("PATCH", adminContent+"/"+leaf.String(), map[string]any{"status": "active"}).Want(t, http.StatusNoContent)
		if !covered(getContent(t, w.sam, adminContent), "Fasting") {
			t.Error("Fasting isn't covered after bringing the leaflet back")
		}
	})
	t.Run("happy path: a capital in the link's scheme or host (a phone keyboard's) is saved in lower case (verification 1)", func(t *testing.T) {
		leaf := addContent(t, w.sam, adminContent, leaflet("Capital leaflet", "HTTPS://Example.ORG/Leaflets/A.pdf"))
		vid := addContent(t, w.sam, adminContent, video("Capital video", "Https://youtu.be/aqz-KE-bpKQ", 30))
		w.sam.Do("PATCH", adminContent+"/"+vid.String(), map[string]any{"url": "Https://Vimeo.com/76979871"}).Want(t, http.StatusNoContent)
		for id, want := range map[uuid.UUID]string{leaf: "https://example.org/Leaflets/A.pdf", vid: "https://vimeo.com/76979871"} {
			var got string
			if err := w.c.DB.QueryRow(t.Context(), `SELECT url FROM content_items WHERE id = $1`, id).Scan(&got); err != nil || got != want {
				t.Errorf("stored url = %q, %v; want %q", got, err, want)
			}
		}
	})
	t.Run("no sign-in", func(t *testing.T) {
		refused(t, w.c.Do("GET", adminContent, nil), http.StatusUnauthorized)
		refused(t, w.c.Do("POST", adminContent, video("X", ytLink, 10)), http.StatusUnauthorized)
		refused(t, w.c.Do("PATCH", adminContent+"/"+uuid.NewString(), map[string]any{"status": "retired"}), http.StatusUnauthorized)
	})
	t.Run("wrong role: a trust admin isn't a platform admin", func(t *testing.T) {
		id := addContent(t, w.sam, adminContent, video("Platform only", ytLink, 10))
		refused(t, w.ada.Do("GET", adminContent, nil), http.StatusForbidden)
		refused(t, w.ada.Do("POST", adminContent, video("X", ytLink, 10)), http.StatusForbidden)
		refused(t, w.ada.Do("PATCH", adminContent+"/"+id.String(), map[string]any{"status": "retired"}), http.StatusForbidden)
		if n := count(t, w.c.DB, `SELECT count(*) FROM content_items WHERE title = 'X' OR (id = $1 AND status <> 'active')`, id); n != 0 {
			t.Errorf("%d rows changed by a refused request", n)
		}
	})
	t.Run("another tenant's data: a trust's own item isn't the platform's to change", func(t *testing.T) {
		own := addContent(t, w.ada, contentPath(w.a1.Org, ""), video("Trust A's own", ytLink, 10))
		refused(t, w.sam.Do("PATCH", adminContent+"/"+own.String(), map[string]any{"status": "retired"}), http.StatusNotFound)
		if titles := contentTitles(getContent(t, w.sam, adminContent).Items); slices.Contains(titles, "Trust A's own") {
			t.Errorf("the platform admin's list has a trust's item: %q", titles)
		}
		if n := count(t, w.c.DB, `SELECT count(*) FROM content_items WHERE id = $1 AND status = 'active'`, own); n != 1 {
			t.Error("the trust's item was changed")
		}
	})
	t.Run("invalid input", func(t *testing.T) {
		id := addContent(t, w.sam, adminContent, video("Valid", ytLink, 10))
		for name, body := range map[string]map[string]any{
			"a link that isn't YouTube or Vimeo": video("Bad", "https://www.dailymotion.com/video/x7tgad0", 10),
			"a YouTube channel":                  video("Bad", "https://www.youtube.com/channel/UC123", 10),
			"http":                               video("Bad", "http://www.youtube.com/watch?v=aqz-KE-bpKQ", 10),
			"no length":                          {"kind": "video", "title": "Bad", "url": ytLink},
			"a length of 0":                      video("Bad", ytLink, 0),
			"over 4 hours":                       video("Bad", ytLink, 4*3600+1),
			"no title":                           video("  ", ytLink, 10),
			"another kind":                       {"kind": "image", "title": "Bad", "url": ytLink},
			"a leaflet without https":            leaflet("Bad", "http://example.org/a.pdf"),
			"a leaflet that isn't a link":        leaflet("Bad", "javascript:alert(1)"),
			"too many tags":                      video("Bad", ytLink, 10, "1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11"),
		} {
			r := w.sam.Do("POST", adminContent, body)
			refused(t, r, http.StatusBadRequest)
			if name == "a link that isn't YouTube or Vimeo" && r.Message() != "Only YouTube or Vimeo links can be used for a video, for example https://www.youtube.com/watch?v=… or https://vimeo.com/…" {
				t.Errorf("%s: %q", name, r.Message())
			}
		}
		refused(t, w.sam.Do("PATCH", adminContent+"/"+id.String(), map[string]any{"url": "https://example.org/v.mp4"}), http.StatusBadRequest)
		refused(t, w.sam.Do("PATCH", adminContent+"/"+id.String(), map[string]any{"status": "deleted"}), http.StatusBadRequest)
		// A new item is always active (verification 1, finding 3).
		refused(t, w.sam.Do("POST", adminContent, map[string]any{"kind": "video", "title": "Bad", "url": ytLink, "lengthSeconds": 10, "tags": []string{}, "status": "retired"}), http.StatusBadRequest)
		refused(t, w.sam.Do("PATCH", adminContent+"/"+id.String(), map[string]any{"title": nil}), http.StatusBadRequest)
		refused(t, w.sam.Do("PATCH", adminContent+"/abc", map[string]any{"status": "retired"}), http.StatusNotFound)
		refused(t, w.sam.Do("PATCH", adminContent+"/"+uuid.NewString(), map[string]any{"status": "retired"}), http.StatusNotFound)
		if n := count(t, w.c.DB, `SELECT count(*) FROM content_items WHERE title = 'Bad' OR (id = $1 AND (url <> $2 OR status <> 'active'))`, id, ytLink); n != 0 {
			t.Errorf("%d rows saved or changed by a refused request", n)
		}
		// A leaflet's length is ignored: it has none.
		leaf := addContent(t, w.sam, adminContent, map[string]any{"kind": "leaflet", "title": "Leaflet", "url": leafletLink, "lengthSeconds": 30})
		if n := count(t, w.c.DB, `SELECT count(*) FROM content_items WHERE id = $1 AND length_seconds IS NULL`, leaf); n != 1 {
			t.Error("a leaflet kept a length")
		}
	})
}

func TestTrustContent(t *testing.T) {
	t.Parallel()
	w := newContentWorld(t)
	platform := addContent(t, w.sam, adminContent, video("Platform knee video", ytLink, 120, "Knee"))
	retiredPlatform := addContent(t, w.sam, adminContent, video("Platform retired", ytLink, 60, "Knee"))
	w.sam.Do("PATCH", adminContent+"/"+retiredPlatform.String(), map[string]any{"status": "retired"}).Want(t, http.StatusNoContent)

	t.Run("happy path: a trust admin adds a video and a leaflet, edits and retires them", func(t *testing.T) {
		v := addContent(t, w.ada, contentPath(w.a1.Org, ""), video("A's hip video", vimeoLink+"/8272103f6e", 300, "Hip", "Recovery"))
		l := addContent(t, w.ada, contentPath(w.a1.Org, ""), leaflet("A's fasting leaflet", leafletLink, "fasting"))
		var org uuid.UUID
		if err := w.c.DB.QueryRow(t.Context(), `SELECT org_id FROM content_items WHERE id = $1`, v).Scan(&org); err != nil || org != w.a1.Org {
			t.Errorf("org_id = %s, %v; want trust A", org, err)
		}
		got := getContent(t, w.ada, contentPath(w.a1.Org, ""))
		// The trust's own first, then Preopsis's active ones; never a retired platform item.
		if titles := contentTitles(got.Items); !slices.Equal(titles, []string{"A's fasting leaflet", "A's hip video", "Platform knee video"}) {
			t.Errorf("items = %q", titles)
		}
		if p := findContent(got.Items, "Platform knee video"); p == nil || !p.Platform || p.ID != platform {
			t.Errorf("platform item = %+v", p)
		}
		if !covered(got, "Fasting") || !covered(got, "Recovery") || covered(got, "Consent") {
			t.Errorf("checklist = %+v", got.Checklist)
		}
		w.ada.Do("PATCH", contentPath(w.a1.Org, "/"+v.String()), map[string]any{"title": "A's hip replacement video", "tags": []string{"Hip"}}).Want(t, http.StatusNoContent)
		w.ada.Do("PATCH", contentPath(w.a1.Org, "/"+l.String()), map[string]any{"status": "retired"}).Want(t, http.StatusNoContent)
		got = getContent(t, w.ada, contentPath(w.a1.Org, ""))
		if titles := contentTitles(got.Items); !slices.Equal(titles, []string{"A's hip replacement video", "A's fasting leaflet", "Platform knee video"}) {
			t.Errorf("after the changes, items = %q (retired own items last of the trust's)", titles)
		}
		if covered(got, "Fasting") || covered(got, "Recovery") {
			t.Errorf("checklist after retiring and untagging = %+v", got.Checklist)
		}
	})
	t.Run("happy path: a whole-trust super clinician reads the list", func(t *testing.T) {
		got := getContent(t, w.alex, contentPath(w.a1.Org, ""))
		if !slices.Contains(contentTitles(got.Items), "Platform knee video") {
			t.Errorf("Alex sees %q", contentTitles(got.Items))
		}
	})
	t.Run("no sign-in", func(t *testing.T) {
		refused(t, w.c.Do("GET", contentPath(w.a1.Org, ""), nil), http.StatusUnauthorized)
		refused(t, w.c.Do("POST", contentPath(w.a1.Org, ""), video("X", ytLink, 10)), http.StatusUnauthorized)
		refused(t, w.c.Do("PATCH", contentPath(w.a1.Org, "/"+uuid.NewString()), map[string]any{"status": "retired"}), http.StatusUnauthorized)
	})
	t.Run("wrong role", func(t *testing.T) {
		own := addContent(t, w.ada, contentPath(w.a1.Org, ""), video("A's own", ytLink, 10))
		// Reading: clinicians and a super clinician of one hospital only.
		for _, c := range []*apitest.Client{w.cara, w.sue, w.trustClin} {
			refused(t, c.Do("GET", contentPath(w.a1.Org, ""), nil), http.StatusForbidden)
		}
		// Changing: everyone but whole-trust admins, Alex included.
		for _, c := range []*apitest.Client{w.alex, w.cara, w.sue, w.trustClin} {
			refused(t, c.Do("POST", contentPath(w.a1.Org, ""), video("X", ytLink, 10)), http.StatusForbidden)
			refused(t, c.Do("PATCH", contentPath(w.a1.Org, "/"+own.String()), map[string]any{"status": "retired"}), http.StatusForbidden)
		}
		// A platform admin has no membership of the trust (OPS-06).
		refused(t, w.sam.Do("POST", contentPath(w.a1.Org, ""), video("X", ytLink, 10)), http.StatusNotFound)
		// A trust admin can't change Preopsis's items.
		refused(t, w.ada.Do("PATCH", contentPath(w.a1.Org, "/"+platform.String()), map[string]any{"status": "retired"}), http.StatusForbidden)
		if n := count(t, w.c.DB, `SELECT count(*) FROM content_items WHERE title = 'X' OR (id IN ($1, $2) AND status <> 'active')`, own, platform); n != 0 {
			t.Errorf("%d rows changed by a refused request", n)
		}
	})
	t.Run("another trust's data", func(t *testing.T) {
		own := addContent(t, w.ada, contentPath(w.a1.Org, ""), video("A's secret", ytLink, 10))
		bOwn := addContent(t, w.bAdmin, contentPath(w.b1.Org, ""), video("B's own", ytLink, 10))
		// Trust B's admin can't read, add to or change trust A's library through A's path…
		refused(t, w.bAdmin.Do("GET", contentPath(w.a1.Org, ""), nil), http.StatusNotFound)
		refused(t, w.bAdmin.Do("POST", contentPath(w.a1.Org, ""), video("X", ytLink, 10)), http.StatusNotFound)
		refused(t, w.bAdmin.Do("PATCH", contentPath(w.a1.Org, "/"+own.String()), map[string]any{"status": "retired"}), http.StatusNotFound)
		// …nor change A's item through B's own path.
		refused(t, w.bAdmin.Do("PATCH", contentPath(w.b1.Org, "/"+own.String()), map[string]any{"status": "retired"}), http.StatusNotFound)
		// Each trust's list has its own items and Preopsis's, never the other trust's.
		if titles := contentTitles(getContent(t, w.bAdmin, contentPath(w.b1.Org, "")).Items); slices.Contains(titles, "A's secret") || !slices.Contains(titles, "B's own") || !slices.Contains(titles, "Platform knee video") {
			t.Errorf("trust B's list = %q", titles)
		}
		if titles := contentTitles(getContent(t, w.ada, contentPath(w.a1.Org, "")).Items); slices.Contains(titles, "B's own") {
			t.Errorf("trust A's list = %q", titles)
		}
		if n := count(t, w.c.DB, `SELECT count(*) FROM content_items WHERE title = 'X' OR (id IN ($1, $2) AND status <> 'active')`, own, bOwn); n != 0 {
			t.Errorf("%d rows changed by a refused request", n)
		}
	})
	t.Run("invalid input", func(t *testing.T) {
		own := addContent(t, w.ada, contentPath(w.a1.Org, ""), leaflet("A's leaflet", leafletLink))
		refused(t, w.ada.Do("POST", contentPath(w.a1.Org, ""), video("Bad", "https://example.org/v.mp4", 10)), http.StatusBadRequest)
		refused(t, w.ada.Do("POST", contentPath(w.a1.Org, ""), leaflet("Bad", "leaflet.pdf")), http.StatusBadRequest)
		refused(t, w.ada.Do("PATCH", contentPath(w.a1.Org, "/"+own.String()), map[string]any{"url": "http://example.org/a.pdf"}), http.StatusBadRequest)
		refused(t, w.ada.Do("PATCH", contentPath(w.a1.Org, "/abc"), map[string]any{"status": "retired"}), http.StatusNotFound)
		refused(t, w.ada.Do("GET", "/api/o/abc/content", nil), http.StatusNotFound)
		if n := count(t, w.c.DB, `SELECT count(*) FROM content_items WHERE title = 'Bad' OR (id = $1 AND url <> $2)`, own, leafletLink); n != 0 {
			t.Errorf("%d rows saved or changed by a refused request", n)
		}
	})
}

func TestSearchContent(t *testing.T) {
	t.Parallel()
	w := newContentWorld(t)
	addContent(t, w.sam, adminContent, video("Knee replacement explained", ytLink, 120, "Knee replacement", "The anaesthetic"))
	gone := addContent(t, w.sam, adminContent, video("Old knee video", ytLink, 60, "Knee replacement"))
	w.sam.Do("PATCH", adminContent+"/"+gone.String(), map[string]any{"status": "retired"}).Want(t, http.StatusNoContent)
	addContent(t, w.ada, contentPath(w.a1.Org, ""), video("A's knee class", vimeoLink, 900, "knee REPLACEMENT"))
	aGone := addContent(t, w.ada, contentPath(w.a1.Org, ""), leaflet("A's old knee leaflet", leafletLink, "Knee replacement"))
	w.ada.Do("PATCH", contentPath(w.a1.Org, "/"+aGone.String()), map[string]any{"status": "retired"}).Want(t, http.StatusNoContent)
	addContent(t, w.ada, contentPath(w.a1.Org, ""), leaflet("A's parking map", leafletLink, "Appointment details"))
	addContent(t, w.bAdmin, contentPath(w.b1.Org, ""), video("B's knee video", ytLink, 30, "Knee replacement"))

	t.Run("happy path: by tag, Preopsis's and the trust's own, never retired ones or another trust's", func(t *testing.T) {
		got := searchContent(t, w.alex, w.a1.Org, "Knee replacement", "")
		if titles := contentTitles(got); !slices.Equal(titles, []string{"A's knee class", "Knee replacement explained"}) {
			t.Errorf("by tag = %q", titles)
		}
		if k := findContent(got, "A's knee class"); k == nil || k.Platform || k.Kind != "video" || k.LengthSeconds == nil || *k.LengthSeconds != 900 || k.URL != vimeoLink {
			t.Errorf("A's knee class = %+v", k)
		}
		if titles := contentTitles(searchContent(t, w.bAuthor, w.b1.Org, "knee replacement", "")); !slices.Equal(titles, []string{"B's knee video", "Knee replacement explained"}) {
			t.Errorf("trust B by tag = %q", titles)
		}
	})
	t.Run("happy path: by words in the title or a tag; with no tag or words, everything active", func(t *testing.T) {
		if titles := contentTitles(searchContent(t, w.alex, w.a1.Org, "", "PARKING")); !slices.Equal(titles, []string{"A's parking map"}) {
			t.Errorf("by title = %q", titles)
		}
		if titles := contentTitles(searchContent(t, w.alex, w.a1.Org, "", "anaesth")); !slices.Equal(titles, []string{"Knee replacement explained"}) {
			t.Errorf("by tag words = %q", titles)
		}
		if titles := contentTitles(searchContent(t, w.ada, w.a1.Org, "", "")); !slices.Equal(titles, []string{"A's knee class", "A's parking map", "Knee replacement explained"}) {
			t.Errorf("everything = %q", titles)
		}
		if titles := contentTitles(searchContent(t, w.alex, w.a1.Org, "Knee", "")); len(titles) != 0 {
			t.Errorf("a tag matches whole tags only, got %q", titles)
		}
		if titles := contentTitles(searchContent(t, w.alex, w.a1.Org, "", "%")); len(titles) != 0 {
			t.Errorf("a %% matched %q: it isn't a wildcard", titles)
		}
	})
	t.Run("no sign-in", func(t *testing.T) {
		refused(t, w.c.Do("GET", contentPath(w.a1.Org, "/search?tag=Knee"), nil), http.StatusUnauthorized)
	})
	t.Run("wrong role: clinicians and a super clinician of one hospital don't author the trust's questionnaires", func(t *testing.T) {
		for _, c := range []*apitest.Client{w.cara, w.sue, w.trustClin} {
			refused(t, c.Do("GET", contentPath(w.a1.Org, "/search?tag=Knee"), nil), http.StatusForbidden)
		}
	})
	t.Run("another trust", func(t *testing.T) {
		refused(t, w.bAuthor.Do("GET", contentPath(w.a1.Org, "/search?tag=Knee+replacement"), nil), http.StatusNotFound)
	})
	t.Run("invalid input", func(t *testing.T) {
		refused(t, w.alex.Do("GET", "/api/o/abc/content/search", nil), http.StatusNotFound)
		refused(t, w.alex.Do("GET", contentPath(w.a1.Org, "/search?q="+longText(201)), nil), http.StatusBadRequest)
	})
}

func longText(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = 'a'
	}
	return string(b)
}

// The platform's starter questions (S33 answers 4): a platform admin lists them and retires one;
// Insert from library and "Same as…" stop offering it; questionnaires it was inserted into keep it.
func TestAdminStarterQuestions(t *testing.T) {
	t.Parallel()
	w := newContentWorld(t)
	cw := &cast{fx: w.fx, db: w.c.DB}
	smoker := platformQuestion(t, cw, "smoker", "Do you smoke?", smokerElement, "Lifestyle")
	platformQuestion(t, cw, "snoring", "Do you snore?", `{"type":"radiogroup","name":"snoring","title":"Do you snore?","choices":[{"value":"yes","text":"Yes"},{"value":"no","text":"No"}]}`, "Sleep")
	w.alex.Do("POST", libraryPath(w.a1.Org, ""), map[string]any{
		"element": map[string]any{"type": "text", "name": "q_job", "title": "What is your job?"},
	}).Want(t, http.StatusCreated)
	type starter struct {
		ID         uuid.UUID
		Key, Title string
		Tags       []string
		Retired    bool
	}
	list := func(t *testing.T) []starter {
		t.Helper()
		r := w.sam.Do("GET", "/api/admin/library-questions", nil)
		r.Want(t, http.StatusOK)
		var got []starter
		r.Decode(t, &got)
		return got
	}

	t.Run("happy path: retire one, then offer it again", func(t *testing.T) {
		got := list(t)
		if len(got) != 2 || got[0].Key != "smoker" || got[0].Title != "Do you smoke?" || !slices.Equal(got[0].Tags, []string{"Lifestyle"}) || got[0].Retired || got[1].Key != "snoring" {
			t.Fatalf("starter questions = %+v (the trust's own never listed)", got)
		}
		w.sam.Do("PATCH", "/api/admin/library-questions/"+smoker.String(), map[string]any{"retired": true}).Want(t, http.StatusNoContent)
		if got := list(t); got[0].Key != "snoring" || got[1].Key != "smoker" || !got[1].Retired {
			t.Errorf("after retiring, list = %+v (retired last)", got)
		}
		// Insert from library no longer offers it; the trust's own and the other starter stay.
		if k := libraryKeys(listLibrary(t, w.alex, w.a1.Org, "")); !slices.Equal(k, []string{"snoring", "q_job"}) {
			t.Errorf("the editor's library = %v, want no retired smoker", k)
		}
		if n := count(t, w.c.DB, `SELECT count(*) FROM library_questions WHERE id = $1 AND retired`, smoker); n != 1 {
			t.Error("the row isn't retired")
		}
		w.sam.Do("PATCH", "/api/admin/library-questions/"+smoker.String(), map[string]any{"retired": false}).Want(t, http.StatusNoContent)
		if k := libraryKeys(listLibrary(t, w.alex, w.a1.Org, "")); !slices.Equal(k, []string{"smoker", "snoring", "q_job"}) {
			t.Errorf("after offering it again, the library = %v", k)
		}
	})
	t.Run("no sign-in", func(t *testing.T) {
		refused(t, w.c.Do("GET", "/api/admin/library-questions", nil), http.StatusUnauthorized)
		refused(t, w.c.Do("PATCH", "/api/admin/library-questions/"+smoker.String(), map[string]any{"retired": true}), http.StatusUnauthorized)
	})
	t.Run("wrong role: a trust admin and an author", func(t *testing.T) {
		for _, c := range []*apitest.Client{w.ada, w.alex} {
			refused(t, c.Do("GET", "/api/admin/library-questions", nil), http.StatusForbidden)
			refused(t, c.Do("PATCH", "/api/admin/library-questions/"+smoker.String(), map[string]any{"retired": true}), http.StatusForbidden)
		}
		if n := count(t, w.c.DB, `SELECT count(*) FROM library_questions WHERE retired`); n != 0 {
			t.Errorf("%d retired by a refused request", n)
		}
	})
	t.Run("another tenant's data: a trust's own library question isn't a starter", func(t *testing.T) {
		var own uuid.UUID
		if err := w.c.DB.QueryRow(t.Context(), `SELECT id FROM library_questions WHERE key = 'q_job'`).Scan(&own); err != nil {
			t.Fatal(err)
		}
		refused(t, w.sam.Do("PATCH", "/api/admin/library-questions/"+own.String(), map[string]any{"retired": true}), http.StatusNotFound)
		if n := count(t, w.c.DB, `SELECT count(*) FROM library_questions WHERE retired`); n != 0 {
			t.Errorf("%d retired", n)
		}
	})
	t.Run("invalid input", func(t *testing.T) {
		refused(t, w.sam.Do("PATCH", "/api/admin/library-questions/"+smoker.String(), map[string]any{}), http.StatusBadRequest)
		refused(t, w.sam.Do("PATCH", "/api/admin/library-questions/"+smoker.String(), map[string]any{"retired": "yes"}), http.StatusBadRequest)
		refused(t, w.sam.Do("PATCH", "/api/admin/library-questions/abc", map[string]any{"retired": true}), http.StatusNotFound)
		refused(t, w.sam.Do("PATCH", "/api/admin/library-questions/"+uuid.NewString(), map[string]any{"retired": true}), http.StatusNotFound)
	})
}
