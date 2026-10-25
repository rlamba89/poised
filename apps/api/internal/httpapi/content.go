package httpapi

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/rlamba89/poised/apps/api/internal/content"
	"github.com/rlamba89/poised/apps/api/internal/db"
	"github.com/rlamba89/poised/apps/api/internal/role"
)

// The content library (plan C5, session S33; INF-05, PX-13, A-6): videos (YouTube or Vimeo links)
// and leaflets (links to PDFs on the web, until C8) that questionnaires and the patient's
// Information point to.
//   - /api/admin/content: Preopsis's own items (org_id NULL), for platform admins only. Every trust
//     sees the active ones.
//   - /api/o/{oid}/content: the trust's own items, which its whole-trust admins add, edit and
//     retire; whole-trust super clinicians may read them. Preopsis's active items come with them,
//     read-only, and the PX-13 checklist says which topics have nothing yet (S33 answers 3).
//   - /api/o/{oid}/content/search: what an author may pick (S34): active items only.
// An item is never deleted, only retired (and may be brought back). No patient data, so not audited.

const (
	contentNotFound   = "Content item not found."
	contentReadRole   = "Only super clinicians and admins of the whole trust can see the content library."
	contentChangeRole = "Only admins of the whole trust can change its content library."
	contentPlatform   = "Preopsis's items can't be changed by a trust."
	contentVideoLink  = "Only YouTube or Vimeo links can be used for a video, for example https://www.youtube.com/watch?v=… or https://vimeo.com/…"
	contentLeaflet    = "Enter the leaflet's web address, starting https://"
	maxContentSearch  = 200
)

// contentItem is one item as the library pages list it.
type contentItem struct {
	ID            uuid.UUID `json:"id"`
	Platform      bool      `json:"platform"`
	Kind          string    `json:"kind"`
	Title         string    `json:"title"`
	URL           string    `json:"url"`
	LengthSeconds *int32    `json:"lengthSeconds"`
	Tags          []string  `json:"tags"`
	Status        string    `json:"status"`
	UpdatedAt     time.Time `json:"updatedAt"`
	UpdatedByName string    `json:"updatedByName"`
}

// contentList is a library page: its items and the checklist against the active ones.
type contentList struct {
	Items     []contentItem   `json:"items"`
	Checklist []content.Topic `json:"checklist"`
}

func newContentList(items []contentItem) contentList {
	var tags [][]string
	for _, i := range items {
		if i.Status == "active" {
			tags = append(tags, i.Tags)
		}
	}
	return contentList{Items: items, Checklist: content.Covered(tags)}
}

// adminListContent is the platform admin's page: Preopsis's items, retired ones included.
func (s *server) adminListContent(w http.ResponseWriter, r *http.Request) {
	rows, err := s.q.ListPlatformContent(r.Context())
	if err != nil {
		serverError(w, "list platform content", err)
		return
	}
	items := []contentItem{}
	for _, c := range rows {
		items = append(items, contentItem{ID: c.ID, Platform: true, Kind: c.Kind, Title: c.Title, URL: c.Url, LengthSeconds: c.LengthSeconds,
			Tags: c.Tags, Status: c.Status, UpdatedAt: c.UpdatedAt, UpdatedByName: c.UpdatedByName})
	}
	writeJSON(w, http.StatusOK, newContentList(items))
}

// listContent is the trust's page: its own items (retired ones included) and Preopsis's active ones.
func (s *server) listContent(w http.ResponseWriter, r *http.Request) {
	if !here(r).TrustRole.AtLeast(role.SuperClinician) {
		writeError(w, http.StatusForbidden, contentReadRole)
		return
	}
	rows, err := s.q.ListTrustContent(r.Context(), uuid.NullUUID{UUID: orgID(r), Valid: true})
	if err != nil {
		serverError(w, "list content", err)
		return
	}
	items := []contentItem{}
	for _, c := range rows {
		items = append(items, contentItem{ID: c.ID, Platform: c.Platform, Kind: c.Kind, Title: c.Title, URL: c.Url, LengthSeconds: c.LengthSeconds,
			Tags: c.Tags, Status: c.Status, UpdatedAt: c.UpdatedAt, UpdatedByName: c.UpdatedByName})
	}
	writeJSON(w, http.StatusOK, newContentList(items))
}

// searchContent is the picker's search (S34): ?tag= a whole tag, ?q= words in the title or a tag.
func (s *server) searchContent(w http.ResponseWriter, r *http.Request) {
	if !here(r).TrustRole.AtLeast(role.SuperClinician) {
		writeError(w, http.StatusForbidden, contentReadRole)
		return
	}
	tag, q := strings.TrimSpace(r.URL.Query().Get("tag")), strings.TrimSpace(r.URL.Query().Get("q"))
	if len([]rune(tag)) > content.MaxTag || len([]rune(q)) > maxContentSearch {
		writeError(w, http.StatusBadRequest, "The search is too long.")
		return
	}
	rows, err := s.q.SearchContent(r.Context(), db.SearchContentParams{OrgID: uuid.NullUUID{UUID: orgID(r), Valid: true}, Tag: tag, Search: q})
	if err != nil {
		serverError(w, "search content", err)
		return
	}
	type found struct {
		ID            uuid.UUID `json:"id"`
		Platform      bool      `json:"platform"`
		Kind          string    `json:"kind"`
		Title         string    `json:"title"`
		URL           string    `json:"url"`
		LengthSeconds *int32    `json:"lengthSeconds"`
		Tags          []string  `json:"tags"`
	}
	out := []found{}
	for _, c := range rows {
		out = append(out, found{ID: c.ID, Platform: c.Platform, Kind: c.Kind, Title: c.Title, URL: c.Url, LengthSeconds: c.LengthSeconds, Tags: c.Tags})
	}
	writeJSON(w, http.StatusOK, out)
}

// contentFields are an item's changeable fields (its kind is set when it's added).
type contentFields struct {
	Kind, Title, URL string
	LengthSeconds    *int32
	Tags             []string
	Status           string
}

// contentChange is a POST or PATCH body, decoded over the current fields: a field left out keeps
// its value; one sent as null is refused (a leaflet's length aside, which it never has).
type contentChange struct {
	Kind          *string   `json:"kind"` // when adding only: an item's kind never changes
	Title         *string   `json:"title"`
	URL           *string   `json:"url"`
	LengthSeconds *int64    `json:"lengthSeconds"`
	Tags          *[]string `json:"tags"`
	Status        *string   `json:"status"`
}

// readContentChange reads the body over current and checks the result; it answers any problem
// itself. Only a new item (adding) takes its kind from the body.
func readContentChange(w http.ResponseWriter, r *http.Request, current contentFields, adding bool) (contentFields, bool) {
	next := current
	var length *int64
	if current.LengthSeconds != nil {
		l := int64(*current.LengthSeconds)
		length = &l
	}
	tags := current.Tags
	in := contentChange{Title: &next.Title, URL: &next.URL, LengthSeconds: length, Tags: &tags, Status: &next.Status}
	if adding {
		in.Kind = &next.Kind
	}
	if !readJSON(w, r, 16<<10, &in) {
		return current, false
	}
	if !adding || in.Kind == nil {
		next.Kind = current.Kind
	}
	if msg := in.apply(&next); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return current, false
	}
	return next, true
}

// apply checks the decoded change and puts it into c; it returns a plain-language problem, or "".
func (in contentChange) apply(c *contentFields) string {
	if in.Title == nil || in.URL == nil || in.Tags == nil || in.Status == nil {
		return "The request could not be read."
	}
	c.Title, c.URL, c.Status = strings.TrimSpace(*in.Title), strings.TrimSpace(*in.URL), *in.Status
	var msg string
	if c.Tags, msg = content.Tags(*in.Tags); msg != "" {
		return msg
	}
	switch {
	case c.Kind != "video" && c.Kind != "leaflet":
		return "Choose a video or a leaflet."
	case c.Title == "":
		return "Enter a title."
	case len([]rune(c.Title)) > content.MaxTitle:
		return fmt.Sprintf("The title is too long (%d characters at most).", content.MaxTitle)
	case c.Status != "active" && c.Status != "retired":
		return "The status must be active or retired."
	}
	if link, ok := content.Link(c.URL); ok {
		c.URL = link // stored with "https://" in lower case, as the table requires
	}
	if c.Kind == "leaflet" {
		c.LengthSeconds = nil // a leaflet has no length
		if !content.CheckLeaflet(c.URL) {
			return contentLeaflet
		}
		return ""
	}
	if _, ok := content.CheckVideo(c.URL); !ok {
		return contentVideoLink
	}
	if in.LengthSeconds == nil {
		return "Enter the video's length."
	}
	if *in.LengthSeconds < 1 || *in.LengthSeconds > content.MaxLength {
		return "The video's length must be between 1 second and 4 hours."
	}
	l := int32(*in.LengthSeconds)
	c.LengthSeconds = &l
	return ""
}

// addContent adds an item for owner (NULL: Preopsis's). The caller has checked the role.
func (s *server) addContent(w http.ResponseWriter, r *http.Request, owner uuid.NullUUID) {
	in, ok := readContentChange(w, r, contentFields{Tags: []string{}, Status: "active"}, true)
	if !ok {
		return
	}
	if in.Status != "active" { // a new item is always offered; retire it afterwards
		writeError(w, http.StatusBadRequest, "A new item is always active.")
		return
	}
	id, err := s.q.CreateContentItem(r.Context(), db.CreateContentItemParams{
		OrgID: owner, Kind: in.Kind, Title: in.Title, Url: in.URL, LengthSeconds: in.LengthSeconds, Tags: in.Tags,
		By: uuid.NullUUID{UUID: currentUser(r).ID, Valid: true},
	})
	if err != nil {
		serverError(w, "add content", err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]uuid.UUID{"id": id})
}

// changeContent changes the fields sent of one of owner's items (NULL: Preopsis's): an edit,
// retiring it or bringing it back. A trust finds Preopsis's items too, but may not change them.
func (s *server) changeContent(w http.ResponseWriter, r *http.Request, owner uuid.NullUUID) {
	id, ok := pathID(w, r, "cid", contentNotFound)
	if !ok {
		return
	}
	ok = s.inTx(w, r, func(q *db.Queries) bool {
		row, err := q.GetContentItem(r.Context(), db.GetContentItemParams{ID: id, Owner: owner})
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, contentNotFound)
			return false
		}
		if err != nil {
			serverError(w, "get content", err)
			return false
		}
		if row.OrgID != owner {
			writeError(w, http.StatusForbidden, contentPlatform)
			return false
		}
		next, ok := readContentChange(w, r, contentFields{Kind: row.Kind, Title: row.Title, URL: row.Url, LengthSeconds: row.LengthSeconds, Tags: row.Tags, Status: row.Status}, false)
		if !ok {
			return false
		}
		if err := q.UpdateContentItem(r.Context(), db.UpdateContentItemParams{
			ID: id, Title: next.Title, Url: next.URL, LengthSeconds: next.LengthSeconds, Tags: next.Tags, Status: next.Status,
			By: uuid.NullUUID{UUID: currentUser(r).ID, Valid: true},
		}); err != nil {
			serverError(w, "change content", err)
			return false
		}
		return true
	})
	if ok {
		w.WriteHeader(http.StatusNoContent)
	}
}

func (s *server) adminAddContent(w http.ResponseWriter, r *http.Request) {
	s.addContent(w, r, uuid.NullUUID{})
}

func (s *server) adminChangeContent(w http.ResponseWriter, r *http.Request) {
	s.changeContent(w, r, uuid.NullUUID{})
}

func (s *server) addTrustContent(w http.ResponseWriter, r *http.Request) {
	if !here(r).TrustRole.AtLeast(role.Admin) {
		writeError(w, http.StatusForbidden, contentChangeRole)
		return
	}
	s.addContent(w, r, uuid.NullUUID{UUID: orgID(r), Valid: true})
}

func (s *server) changeTrustContent(w http.ResponseWriter, r *http.Request) {
	if !here(r).TrustRole.AtLeast(role.Admin) {
		writeError(w, http.StatusForbidden, contentChangeRole)
		return
	}
	s.changeContent(w, r, uuid.NullUUID{UUID: orgID(r), Valid: true})
}

// The platform's starter questions (S33 answers 4): GET /api/admin/library-questions lists them,
// PATCH …/{lid} {"retired": true|false} retires one or offers it again. New ones still come from
// the seed (cmd/seed/library.json), which leaves a retirement alone.

func (s *server) adminListStarterQuestions(w http.ResponseWriter, r *http.Request) {
	rows, err := s.q.ListPlatformLibrary(r.Context())
	if err != nil {
		serverError(w, "list starter questions", err)
		return
	}
	if rows == nil {
		rows = []db.ListPlatformLibraryRow{}
	}
	writeJSON(w, http.StatusOK, rows)
}

func (s *server) adminRetireStarterQuestion(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "lid", libraryNotFound)
	if !ok {
		return
	}
	var in struct {
		Retired *bool `json:"retired"`
	}
	if !readJSON(w, r, 1<<10, &in) {
		return
	}
	if in.Retired == nil {
		writeError(w, http.StatusBadRequest, "Say whether the question is retired.")
		return
	}
	n, err := s.q.SetPlatformLibraryRetired(r.Context(), db.SetPlatformLibraryRetiredParams{ID: id, Retired: *in.Retired})
	if err != nil {
		serverError(w, "retire starter question", err)
		return
	}
	if n == 0 {
		writeError(w, http.StatusNotFound, libraryNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
