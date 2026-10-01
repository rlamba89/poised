package httpapi

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/rlamba89/poised/apps/api/internal/db"
	"github.com/rlamba89/poised/apps/api/internal/episode"
)

// reviewFake adds the validation queries to episodeFake. Writes succeed only while the
// episode is Ready for review, as the SQL's WHERE does.
type reviewFake struct {
	*episodeFake
	saved     []db.SaveClinicianAnswersParams
	completed int
}

func (f *reviewFake) GetVersionChapter(_ context.Context, arg db.GetVersionChapterParams) (db.GetVersionChapterRow, error) {
	if arg.ID != chapterID {
		return db.GetVersionChapterRow{}, pgx.ErrNoRows
	}
	return db.GetVersionChapterRow{ID: chapterID, Audience: "patient"}, nil
}

func (f *reviewFake) SaveClinicianAnswers(_ context.Context, arg db.SaveClinicianAnswersParams) (int64, error) {
	if f.episode.Status != episode.ReadyForReview {
		return 0, nil
	}
	f.saved = append(f.saved, arg)
	return 1, nil
}

func (f *reviewFake) CompleteReview(context.Context, db.CompleteReviewParams) (int64, error) {
	if f.episode.Status != episode.ReadyForReview {
		return 0, nil
	}
	f.completed++
	return 1, nil
}

func TestSaveClinicianAnswers(t *testing.T) {
	path := hospitalPath("/episodes/" + episodeID.String() + "/answers/")
	tests := []struct {
		name, status, chapter, body string
		want                        int
	}{
		{"validate during review", episode.ReadyForReview, chapterID.String(), `{"data":{"q_smoke":"o_n"},"validated":true}`, http.StatusNoContent},
		{"before the patient sends", episode.HQNotComplete, chapterID.String(), `{"data":{}}`, http.StatusConflict},
		{"after the review", episode.ReadyForPOA, chapterID.String(), `{"data":{}}`, http.StatusConflict},
		{"Question Set of another version", episode.ReadyForReview, uuid.NewString(), `{"data":{}}`, http.StatusNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &reviewFake{episodeFake: newEpisodeFake(tt.status, "clinician")}
			rec := call(t, f, "PUT", path+tt.chapter, tt.body)
			if rec.Code != tt.want {
				t.Fatalf("got %d (%s), want %d", rec.Code, rec.Body.String(), tt.want)
			}
			if tt.want == http.StatusNoContent && (!f.saved[0].Validated || f.saved[0].UpdatedBy.UUID != userA.ID) {
				t.Errorf("saved %+v: want validated by %s", f.saved[0], userA.ID)
			}
		})
	}
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
