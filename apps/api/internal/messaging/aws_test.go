package messaging_test

import (
	"bytes"
	"context"
	"errors"
	"log"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/pinpointsmsvoicev2"
	smstypes "github.com/aws/aws-sdk-go-v2/service/pinpointsmsvoicev2/types"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/smithy-go"

	"github.com/rlamba89/poised/apps/api/internal/messaging"
)

type fakeSES struct {
	in  *sesv2.SendEmailInput
	err error
}

func (f *fakeSES) SendEmail(_ context.Context, in *sesv2.SendEmailInput, _ ...func(*sesv2.Options)) (*sesv2.SendEmailOutput, error) {
	f.in = in
	if f.err != nil {
		return nil, f.err
	}
	return &sesv2.SendEmailOutput{MessageId: aws.String("ses-123")}, nil
}

type fakeSMS struct {
	in  *pinpointsmsvoicev2.SendTextMessageInput
	err error
}

func (f *fakeSMS) SendTextMessage(_ context.Context, in *pinpointsmsvoicev2.SendTextMessageInput, _ ...func(*pinpointsmsvoicev2.Options)) (*pinpointsmsvoicev2.SendTextMessageOutput, error) {
	f.in = in
	if f.err != nil {
		return nil, f.err
	}
	return &pinpointsmsvoicev2.SendTextMessageOutput{MessageId: aws.String("sms-456")}, nil
}

// captureLog collects what the package logs during a test.
func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	old, flags := log.Writer(), log.Flags()
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(old); log.SetFlags(flags) })
	return &buf
}

func TestSESSendsAPlainTextEmailThroughTheConfigurationSet(t *testing.T) {
	c := &fakeSES{}
	s := messaging.SES{Client: c, From: "sender@example.test", ConfigurationSet: "poised-xx-qa"}
	id, err := s.SendEmail(context.Background(), "patient@example.test", "St Mary's: your questionnaire", "Hello\nThe link")
	if err != nil || id != "ses-123" {
		t.Fatalf("id %q, err %v", id, err)
	}
	in := c.in
	if got := aws.ToString(in.FromEmailAddress); got != `"Poised" <sender@example.test>` {
		t.Errorf("from %q", got)
	}
	if len(in.Destination.ToAddresses) != 1 || in.Destination.ToAddresses[0] != "patient@example.test" {
		t.Errorf("to %v", in.Destination.ToAddresses)
	}
	if aws.ToString(in.ConfigurationSetName) != "poised-xx-qa" {
		t.Errorf("configuration set %q: the delivery events come through it", aws.ToString(in.ConfigurationSetName))
	}
	msg := in.Content.Simple
	if aws.ToString(msg.Subject.Data) != "St Mary's: your questionnaire" || aws.ToString(msg.Body.Text.Data) != "Hello\nThe link" || msg.Body.Html != nil {
		t.Errorf("content %+v", msg)
	}
}

func TestSMSSendsATransactionalTextThroughTheConfigurationSet(t *testing.T) {
	c := &fakeSMS{}
	s := messaging.SMSAWS{Client: c, ConfigurationSet: "poised-xx-qa"}
	id, err := s.SendSMS(context.Background(), "+447700900123", "Your link")
	if err != nil || id != "sms-456" {
		t.Fatalf("id %q, err %v", id, err)
	}
	in := c.in
	if aws.ToString(in.DestinationPhoneNumber) != "+447700900123" || aws.ToString(in.MessageBody) != "Your link" {
		t.Errorf("sent %+v", in)
	}
	if in.MessageType != smstypes.MessageTypeTransactional {
		t.Errorf("type %q", in.MessageType)
	}
	if aws.ToString(in.ConfigurationSetName) != "poised-xx-qa" {
		t.Errorf("configuration set %q: its protect configuration allows only the country's numbers", aws.ToString(in.ConfigurationSetName))
	}
	if in.OriginationIdentity != nil {
		t.Errorf("origination identity %q without a sender ID", aws.ToString(in.OriginationIdentity))
	}

	s.SenderID = "Poised"
	if _, err := s.SendSMS(context.Background(), "+447700900123", "x"); err != nil || aws.ToString(c.in.OriginationIdentity) != "Poised" {
		t.Errorf("sender ID: %v, %v", err, c.in.OriginationIdentity)
	}
}

// A refused send is logged for the delivery-failure alarm and returned, without the address or
// number, which AWS's error messages can hold.
func TestAFailedSendIsLoggedForTheAlarmWithoutTheAddress(t *testing.T) {
	apiErr := &smithy.GenericAPIError{Code: "MessageRejected", Message: "Email address is not verified: patient@example.test"}
	buf := captureLog(t)
	_, err := messaging.SES{Client: &fakeSES{err: apiErr}, From: "s@example.test"}.SendEmail(context.Background(), "patient@example.test", "s", "b")
	if err == nil || strings.Contains(err.Error(), "patient@example.test") || !strings.Contains(err.Error(), "MessageRejected") {
		t.Errorf("error %v: want the code and no address", err)
	}
	smsErr := &smithy.GenericAPIError{Code: "ValidationException", Message: "+447700900123 is not verified"}
	_, err = messaging.SMSAWS{Client: &fakeSMS{err: smsErr}}.SendSMS(context.Background(), "+447700900123", "b")
	if err == nil || strings.Contains(err.Error(), "+447700900123") || !strings.Contains(err.Error(), "ValidationException") {
		t.Errorf("error %v: want the code and no number", err)
	}
	_, err = messaging.SMSAWS{Client: &fakeSMS{err: errors.New("dial tcp: +447700900123")}}.SendSMS(context.Background(), "+447700900123", "b")
	if err == nil || strings.Contains(err.Error(), "+447700900123") {
		t.Errorf("error %v: a non-API error mustn't leak the number either", err)
	}

	logged := buf.String()
	if strings.Count(logged, messaging.DeliveryFailedMarker) != 3 {
		t.Errorf("want 3 %s lines, got:\n%s", messaging.DeliveryFailedMarker, logged)
	}
	if strings.Contains(logged, "patient@example.test") || strings.Contains(logged, "+447700900123") {
		t.Errorf("the log holds a contact:\n%s", logged)
	}
}

func TestTestRecipientsCaptureEverythingAndReallySendOnlyToThem(t *testing.T) {
	captured, real := &messaging.Recorder{}, &messaging.Recorder{}
	tr := messaging.TestRecipients{Capture: captured, SMS: real, Email: real,
		Allowed: []string{"+447700900001", " Rahul@Example.test "}}
	ctx := context.Background()

	id, err := tr.SendSMS(ctx, "+447700900001", "to Rahul")
	if err != nil || id != "rec-1" {
		t.Errorf("a test recipient's text: id %q (want the real sender's), err %v", id, err)
	}
	if id, _ := tr.SendSMS(ctx, "+447700900999", "to someone else"); id != "rec-2" {
		t.Errorf("anyone else's text: id %q, want the captured one's", id)
	}
	tr.SendEmail(ctx, "rahul@example.TEST", "s", "to Rahul")                      // any case
	tr.SendEmail(ctx, "bounce@simulator.amazonses.com", "s", "AWS's own address") // SES's simulator
	tr.SendEmail(ctx, "someone@example.test", "s", "to someone else")

	if n := len(captured.Sent()); n != 5 {
		t.Errorf("captured %d messages, want every one (5)", n)
	}
	var to []string
	for _, m := range real.Sent() {
		to = append(to, m.To)
	}
	want := "+447700900001 rahul@example.TEST bounce@simulator.amazonses.com"
	if strings.Join(to, " ") != want {
		t.Errorf("really sent to %v, want only %s", to, want)
	}

	real.FailWith(errors.New("refused"))
	if _, err := tr.SendSMS(ctx, "+447700900001", "x"); err == nil {
		t.Error("a failed real send must be reported, though the message was captured")
	}
}
