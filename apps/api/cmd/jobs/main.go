// Command jobs is the jobs Lambda (plan C2, session S17). It's called two ways:
//   - by its EventBridge Scheduler schedule, every few minutes: it runs the due jobs (reminders
//     and "not responding"), the same jobs.Runner the local timer in cmd/api runs (S16). Where
//     messages are captured (QA), the testers' clock offset applies, as it does locally;
//   - by the deployment's SNS topic, with SES's and End User Messaging SMS's reports of
//     undelivered messages: it records them on the episode (internal/delivery, OPS-05).
//
// Run locally (`go run ./cmd/jobs`), it runs the due jobs once and stops.
package main

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"github.com/rlamba89/poised/apps/api/internal/config"
	"github.com/rlamba89/poised/apps/api/internal/delivery"
	"github.com/rlamba89/poised/apps/api/internal/httpapi"
	"github.com/rlamba89/poised/apps/api/internal/jobs"
	"github.com/rlamba89/poised/apps/api/internal/serve"
)

func main() {
	ctx := context.Background()
	deps := serve.Setup(ctx, "jobs")
	defer deps.Pool.Close()
	handle := handler(deps)
	if serve.OnLambda() {
		log.Printf("jobs on Lambda (messaging %s, time zone %s)", deps.Config.Messaging, deps.Config.Country.TimeZone)
		lambda.Start(handle)
		return
	}
	if _, err := handle(ctx, json.RawMessage(`{}`)); err != nil {
		log.Fatalf("jobs: %v", err)
	}
}

// summary is what one call did, for the Lambda's log and result.
type summary struct {
	Jobs       jobs.Summary `json:"jobs"`
	Deliveries int          `json:"deliveries,omitempty"` // undelivered messages reported
}

// handler answers both callers: an SNS notification (it has Records) holds delivery events;
// anything else is the schedule.
func handler(deps httpapi.Deps) func(context.Context, json.RawMessage) (any, error) {
	runner := httpapi.NewJobsRunner(deps)
	qaClock := deps.Config.Messaging == config.MessagingCapture // the testers' clock: local and QA only
	return func(ctx context.Context, raw json.RawMessage) (any, error) {
		var sns events.SNSEvent
		if json.Unmarshal(raw, &sns) == nil && len(sns.Records) > 0 {
			err := delivery.HandleSNS(ctx, deps.Pool, sns, time.Now())
			return summary{Deliveries: len(sns.Records)}, err
		}
		sum, err := runner.Tick(ctx, time.Now(), qaClock)
		if err != nil {
			log.Printf("jobs: %v", err)
		} else if sum.Jobs > 0 {
			log.Printf("jobs: ran %d as of %s: %d reminders, %d not responding, %d failed",
				sum.Jobs, sum.AsOf.Format(time.RFC3339), sum.Reminders, sum.NotResponding, sum.Failed)
		}
		return summary{Jobs: sum}, err
	}
}
