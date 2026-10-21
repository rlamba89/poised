package delivery_test

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/rlamba89/poised/apps/api/internal/delivery"
	"github.com/rlamba89/poised/apps/api/internal/messaging"
)

// The events as SES and End User Messaging SMS publish them to the SNS topic (the SNS message's
// body), trimmed to the fields used (AWS docs: "Examples of event data that Amazon SES publishes
// to Amazon SNS"; "Example AWS End User Messaging SMS event data").
func TestParse(t *testing.T) {
	cases := []struct {
		name, msg string
		want      delivery.Failure
		failed    bool
	}{
		{"an SES permanent bounce", `{"eventType":"Bounce","bounce":{"bounceType":"Permanent","bounceSubType":"General"},"mail":{"messageId":"ses-1"}}`,
			delivery.Failure{ProviderID: "ses-1", Channel: "email", Reason: "the email address doesn't exist or doesn't accept email"}, true},
		{"an SES transient bounce", `{"eventType":"Bounce","bounce":{"bounceType":"Transient","bounceSubType":"MailboxFull"},"mail":{"messageId":"ses-2"}}`,
			delivery.Failure{ProviderID: "ses-2", Channel: "email", Reason: "the patient's mailbox didn't accept it (it may be full)"}, true},
		{"an SES undetermined bounce", `{"eventType":"Bounce","bounce":{"bounceType":"Undetermined"},"mail":{"messageId":"ses-3"}}`,
			delivery.Failure{ProviderID: "ses-3", Channel: "email", Reason: "it bounced"}, true},
		{"an SES reject", `{"eventType":"Reject","reject":{"reason":"Bad content"},"mail":{"messageId":"ses-4"}}`,
			delivery.Failure{ProviderID: "ses-4", Channel: "email", Reason: "the email service refused it"}, true},
		{"an SES delivery is no failure", `{"eventType":"Delivery","mail":{"messageId":"ses-5"}}`, delivery.Failure{}, false},
		{"an SES delay is no failure (yet)", `{"eventType":"DeliveryDelay","mail":{"messageId":"ses-6"}}`, delivery.Failure{}, false},
		{"an invalid mobile", `{"eventType":"TEXT_INVALID","isFinal":true,"messageId":"sms-1","messageStatus":"INVALID"}`,
			delivery.Failure{ProviderID: "sms-1", Channel: "sms", Reason: "the mobile number isn't valid"}, true},
		{"an unreachable phone", `{"eventType":"TEXT_UNREACHABLE","isFinal":true,"messageId":"sms-2"}`,
			delivery.Failure{ProviderID: "sms-2", Channel: "sms", Reason: "the phone couldn't be reached (it may be switched off)"}, true},
		{"the network couldn't", `{"eventType":"TEXT_CARRIER_UNREACHABLE","isFinal":true,"messageId":"sms-3"}`,
			delivery.Failure{ProviderID: "sms-3", Channel: "sms", Reason: "the mobile network couldn't deliver it"}, true},
		{"no final word in 72 hours", `{"eventType":"TEXT_UNKNOWN","isFinal":true,"messageId":"sms-4"}`,
			delivery.Failure{ProviderID: "sms-4", Channel: "sms", Reason: "the mobile network couldn't deliver it"}, true},
		{"blocked by the protect configuration", `{"eventType":"TEXT_PROTECT_BLOCKED","isFinal":true,"messageId":"sms-5"}`,
			delivery.Failure{ProviderID: "sms-5", Channel: "sms", Reason: "texts go only to mobile numbers in this country"}, true},
		{"the network called it spam", `{"eventType":"TEXT_SPAM","isFinal":true,"messageId":"sms-6"}`,
			delivery.Failure{ProviderID: "sms-6", Channel: "sms", Reason: "the mobile network blocked it"}, true},
		{"a delivered text is no failure", `{"eventType":"TEXT_DELIVERED","isFinal":true,"messageId":"sms-7"}`, delivery.Failure{}, false},
		{"a pending text is no failure", `{"eventType":"TEXT_PENDING","isFinal":false,"messageId":"sms-8"}`, delivery.Failure{}, false},
		{"SNS's subscription message is ignored", `{"Type":"SubscriptionConfirmation"}`, delivery.Failure{}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, failed, err := delivery.Parse(c.msg)
			if err != nil || failed != c.failed || got != c.want {
				t.Errorf("got %+v, %v, %v; want %+v, %v", got, failed, err, c.want, c.failed)
			}
		})
	}
	for _, bad := range []string{`not json`, `{"eventType":"Bounce","mail":{}}`, `{"eventType":"TEXT_INVALID"}`} {
		if _, _, err := delivery.Parse(bad); err == nil {
			t.Errorf("Parse(%s): no error", bad)
		}
	}
}

func TestNote(t *testing.T) {
	cases := []struct{ kind, status, channel, reason, want string }{
		{"invite", "", "email", "it bounced",
			"The invite's email wasn't delivered: it bounced. Check the patient's email address, then resend the invite."},
		{"reminder", "", "sms", "the mobile number isn't valid",
			"A reminder's text message wasn't delivered: the mobile number isn't valid. Check the patient's mobile number, then resend the invite."},
		// S21: a status message has no invite to resend.
		{"status", "ready_for_admission", "sms", "the phone couldn't be reached",
			"The Ready for admission text message wasn't delivered: the phone couldn't be reached. Check the patient's mobile number, and tell them another way."},
		{"status", "cancelled", "email", "it bounced",
			"The Cancelled email wasn't delivered: it bounced. Check the patient's email address, and tell them another way."},
	}
	for _, c := range cases {
		if got := delivery.Note(c.kind, c.status, c.channel, c.reason); got != c.want {
			t.Errorf("Note(%s, %s) =\n %s\nwant\n %s", c.kind, c.channel, got, c.want)
		}
	}
}

// infra subscribes the SNS topic to exactly these SMS events, and its alarm counts the senders'
// DELIVERY_FAILED lines: the two sides are in different languages, so compare them here.
func TestInfraAgrees(t *testing.T) {
	read := func(name string) string {
		b, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "infra", name))
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	stack := read("messaging_stack.py")
	list := stack[strings.Index(stack, "SMS_FAILURE_EVENTS = ["):]
	list = list[:strings.Index(list, "]")]
	got := regexp.MustCompile(`"(TEXT_[A-Z_]+)"`).FindAllStringSubmatch(list, -1)
	var infra []string
	for _, m := range got {
		infra = append(infra, m[1])
	}
	want := delivery.SMSFailureEvents()
	slices.Sort(infra)
	slices.Sort(want)
	if !slices.Equal(infra, want) {
		t.Errorf("infra's SMS_FAILURE_EVENTS %v, Go's %v", infra, want)
	}
	if !strings.Contains(read("api_stack.py"), `DELIVERY_FAILED_MARKER = "`+messaging.DeliveryFailedMarker+`"`) {
		t.Errorf("infra/api_stack.py's DELIVERY_FAILED_MARKER isn't %q", messaging.DeliveryFailedMarker)
	}
}
