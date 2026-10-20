package messaging

import (
	"context"
	"errors"
	"log"
	"net/mail"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/pinpointsmsvoicev2"
	smstypes "github.com/aws/aws-sdk-go-v2/service/pinpointsmsvoicev2/types"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	sestypes "github.com/aws/aws-sdk-go-v2/service/sesv2/types"
	"github.com/aws/smithy-go"
)

// DeliveryFailedMarker starts every log line about a message that couldn't be sent or wasn't
// delivered. infra's metric filter counts it for the delivery-failure alarm (OPS-05), so keep the
// two the same.
const DeliveryFailedMarker = "DELIVERY_FAILED"

// LogDeliveryFailed logs one failed message for the alarm. what must never hold a contact.
func LogDeliveryFailed(what string) { log.Printf("%s %s", DeliveryFailedMarker, what) }

// SESClient is the part of the SES v2 client SES uses; tests pass a fake.
type SESClient interface {
	SendEmail(ctx context.Context, in *sesv2.SendEmailInput, opts ...func(*sesv2.Options)) (*sesv2.SendEmailOutput, error)
}

// SES sends email with Amazon SES v2 (MESSAGING=send, S17), from the deployment's verified
// sender, through its configuration set, whose event destination reports bounces
// (internal/delivery).
type SES struct {
	Client           SESClient
	From             string // EMAIL_FROM: a verified address (or one at a verified domain)
	ConfigurationSet string // SES_CONFIGURATION_SET
}

func (s SES) SendEmail(ctx context.Context, to, subject, body string) (string, error) {
	text := func(v string) *sestypes.Content {
		return &sestypes.Content{Data: aws.String(v), Charset: aws.String("UTF-8")}
	}
	out, err := s.Client.SendEmail(ctx, &sesv2.SendEmailInput{
		FromEmailAddress:     aws.String((&mail.Address{Name: "Preopsis", Address: s.From}).String()),
		Destination:          &sestypes.Destination{ToAddresses: []string{to}},
		Content:              &sestypes.EmailContent{Simple: &sestypes.Message{Subject: text(subject), Body: &sestypes.Body{Text: text(body)}}},
		ConfigurationSetName: optional(s.ConfigurationSet),
	})
	if err != nil {
		return "", refused("email", err)
	}
	return aws.ToString(out.MessageId), nil
}

// SMSClient is the part of the End User Messaging SMS (v2) client SMSAWS uses; tests pass a fake.
type SMSClient interface {
	SendTextMessage(ctx context.Context, in *pinpointsmsvoicev2.SendTextMessageInput, opts ...func(*pinpointsmsvoicev2.Options)) (*pinpointsmsvoicev2.SendTextMessageOutput, error)
}

// SMSAWS sends texts with AWS End User Messaging SMS (MESSAGING=send, S17), as transactional
// messages through the deployment's configuration set: its protect configuration lets texts go
// only to the deployment's country (PAT-11, as patient.SMSAllowed does), and its event destination
// reports undelivered ones (internal/delivery).
type SMSAWS struct {
	Client           SMSClient
	ConfigurationSet string // SMS_CONFIGURATION_SET
	SenderID         string // SMS_SENDER_ID: the origination identity; "" lets AWS choose
}

func (s SMSAWS) SendSMS(ctx context.Context, to, body string) (string, error) {
	out, err := s.Client.SendTextMessage(ctx, &pinpointsmsvoicev2.SendTextMessageInput{
		DestinationPhoneNumber: aws.String(to),
		MessageBody:            aws.String(body),
		MessageType:            smstypes.MessageTypeTransactional,
		ConfigurationSetName:   optional(s.ConfigurationSet),
		OriginationIdentity:    optional(s.SenderID),
	})
	if err != nil {
		return "", refused("text message", err)
	}
	return aws.ToString(out.MessageId), nil
}

// refused logs a send AWS refused, for the alarm, and returns an error naming only AWS's error
// code: AWS's messages can hold the address or number, which mustn't reach the logs.
func refused(what string, err error) error {
	code := "no response"
	if apiErr, ok := errors.AsType[smithy.APIError](err); ok {
		code = apiErr.ErrorCode()
	} else if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		code = "timed out"
	}
	LogDeliveryFailed(what + " refused by AWS: " + code)
	return errors.New(what + " refused by AWS: " + code)
}

func optional(v string) *string {
	if v == "" {
		return nil
	}
	return aws.String(v)
}
