"""MessagingStack: real texts and emails (plan C2, session S17), for an entry that sends
(countries.py: messaging "send", or QA's test recipients).

- **SES:** the sender's identity, and a configuration set whose event destination reports bounces,
  rejects and rendering failures. Until there's a domain the identity is an email address, read
  from SSM (<prefix>/email-from, stored by `make aws-setup`) so no address is in the repo; AWS
  emails it a link to click once.
- **SMS (AWS End User Messaging):** a configuration set whose protect configuration lets texts go
  only to the entry's phone country (PAT-11, as the API's own check does), and whose event
  destination reports undelivered texts; and the entry's sender ID, if it names one.
- **The delivery topic (SNS):** both services publish their failure events to it; ApiStack
  subscribes the jobs Lambda, which records them on the episode (OPS-05).

Sandbox, production access and a UK sender ID's registration are account paperwork, not
resources (README.md).
"""

from __future__ import annotations

import aws_cdk as cdk
from aws_cdk import aws_iam as iam
from aws_cdk import aws_ses as ses
from aws_cdk import aws_smsvoice as smsvoice
from aws_cdk import aws_sns as sns
from aws_cdk import aws_ssm as ssm
from constructs import Construct

from countries import Environment
from sms_countries import SMS_COUNTRY_CODES

# End User Messaging SMS's final failure events. Keep in step with the Go API's
# delivery.SMSFailureEvents (a Go test compares them).
SMS_FAILURE_EVENTS = [
    "TEXT_INVALID",
    "TEXT_UNREACHABLE",
    "TEXT_CARRIER_UNREACHABLE",
    "TEXT_TTL_EXPIRED",
    "TEXT_UNKNOWN",
    "TEXT_BLOCKED",
    "TEXT_CARRIER_BLOCKED",
    "TEXT_SPAM",
    "TEXT_INVALID_MESSAGE",
    "TEXT_PROTECT_BLOCKED",
]


def email_from_param(env: Environment) -> str:
    return f"{env.param_prefix}/email-from"


def test_recipients_param(env: Environment) -> str:
    return f"{env.param_prefix}/messaging-test-recipients"


def configuration_set_name(env: Environment) -> str:
    """The SES and the SMS configuration set share it."""
    return f"poised-{env.name}"


class MessagingStack(cdk.Stack):
    def __init__(self, scope: Construct, id: str, *, env: Environment, **kwargs) -> None:
        super().__init__(scope, id, **kwargs)
        name = configuration_set_name(env)

        self.deliveries = sns.Topic(self, "Deliveries", topic_name=f"poised-{env.name}-deliveries")
        for service in ("ses.amazonaws.com", "sms-voice.amazonaws.com"):
            self.deliveries.add_to_resource_policy(
                iam.PolicyStatement(
                    principals=[iam.ServicePrincipal(service)],
                    actions=["sns:Publish"],
                    resources=[self.deliveries.topic_arn],
                    conditions={"StringEquals": {"aws:SourceAccount": self.account}},
                )
            )

        # --- SES ---
        sender = ssm.StringParameter.value_for_string_parameter(self, email_from_param(env))
        ses_set = ses.CfnConfigurationSet(self, "EmailSet", name=name)
        ses.CfnConfigurationSetEventDestination(
            self,
            "EmailFailures",
            configuration_set_name=ses_set.ref,
            event_destination=ses.CfnConfigurationSetEventDestination.EventDestinationProperty(
                enabled=True,
                matching_event_types=["bounce", "reject", "renderingFailure"],
                sns_destination=ses.CfnConfigurationSetEventDestination.SnsDestinationProperty(
                    topic_arn=self.deliveries.topic_arn
                ),
            ),
        )
        ses.CfnEmailIdentity(self, "Sender", email_identity=sender)

        # --- SMS ---
        protect = smsvoice.CfnProtectConfiguration(
            self,
            "OnlyThisCountry",
            country_rule_set=smsvoice.CfnProtectConfiguration.CountryRuleSetProperty(
                sms=[
                    smsvoice.CfnProtectConfiguration.CountryRuleProperty(
                        country_code=code, protect_status="ALLOW" if code == env.phone_country else "BLOCK"
                    )
                    for code in SMS_COUNTRY_CODES
                ]
            ),
        )
        smsvoice.CfnConfigurationSet(
            self,
            "TextSet",
            configuration_set_name=name,
            protect_configuration_id=protect.attr_protect_configuration_id,
            event_destinations=[
                smsvoice.CfnConfigurationSet.EventDestinationProperty(
                    event_destination_name="failures",
                    enabled=True,
                    matching_event_types=SMS_FAILURE_EVENTS,
                    sns_destination=smsvoice.CfnConfigurationSet.SnsDestinationProperty(
                        topic_arn=self.deliveries.topic_arn
                    ),
                )
            ],
        )
        if env.sms_sender_id:
            # Registration (where the country needs it) is paperwork in the console (README.md).
            smsvoice.CfnSenderId(self, "SenderId", iso_country_code=env.phone_country, sender_id=env.sms_sender_id)
