package httpapi

import (
	"context"
	"net/http"
	"testing"

	"github.com/rlamba89/poised/apps/api/internal/db"
	"github.com/rlamba89/poised/apps/api/internal/episode"
)

// reviewFake adds CompleteReview to episodeFake. It succeeds only while the episode is Ready
// for review, as the SQL's WHERE does. (Answer saves are tested against the database:
// validation_integration_test.go.)
type reviewFake struct {
	*episodeFake
	completed int
}

func (f *reviewFake) CompleteReview(context.Context, db.CompleteReviewParams) (int64, error) {
	if f.episode.Status != episode.ReadyForReview {
		return 0, nil
	}
	f.completed++
	return 1, nil
}

func TestCompleteReview(t *testing.T) {
	path := hospitalPath("/episodes/" + episodeID.String() + "/complete-review")
	f := &reviewFake{episodeFake: newEpisodeFake(episode.ReadyForReview, "clinician")}
	if rec := call(t, f, "POST", path, ""); rec.Code != http.StatusNoContent {
		t.Fatalf("got %d (%s)", rec.Code, rec.Body.String())
	}
	if len(f.events) != 1 || f.events[0].Text != "HQ review completed by Alex Author" {
		t.Errorf("events = %+v", f.events)
	}
	f = &reviewFake{episodeFake: newEpisodeFake(episode.HQNotComplete, "clinician")}
	if rec := call(t, f, "POST", path, ""); rec.Code != http.StatusConflict || errorOf(t, rec) != "The patient hasn't sent their answers yet." {
		t.Errorf("before the patient sends: got %d %s", rec.Code, rec.Body.String())
	}
}
