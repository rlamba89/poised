"""S17: the jobs Lambda on its schedule, real sending (MessagingStack), undelivered messages and
their alarm. Written first; no AWS account is needed. The made-up second country (ca-*) keeps the
UK out of it (A-20)."""

import dataclasses
import json

import aws_cdk as cdk
import pytest
from aws_cdk.assertions import Match, Template

import countries
from countries import Environment
from sms_countries import SMS_COUNTRY_CODES
from stage import PoisedStage
from test_stacks import CA_QA, lambda_env, policy_statements

# QA with real sending to its test recipients only, and a training stage that really sends.
CA_QA_TEST = dataclasses.replace(CA_QA, qa_test_recipients=True)
CA_TRAINING = dataclasses.replace(CA_QA, name="ca-training", stage="training", messaging="send", sms_sender_id="PREOPSIS")

# End User Messaging SMS's failure events: the Go API's delivery.SMSFailureEvents (a Go test
# checks the two lists agree).
SMS_FAILURES = {
    "TEXT_INVALID", "TEXT_UNREACHABLE", "TEXT_CARRIER_UNREACHABLE", "TEXT_TTL_EXPIRED", "TEXT_UNKNOWN",
    "TEXT_BLOCKED", "TEXT_CARRIER_BLOCKED", "TEXT_SPAM", "TEXT_INVALID_MESSAGE", "TEXT_PROTECT_BLOCKED",
}


@pytest.fixture
def lambda_code(tmp_path):
    for fn in ("staff-api", "patient-api", "jobs"):
        (tmp_path / fn).mkdir()
        (tmp_path / fn / "bootstrap").write_text("not a real binary")
    return tmp_path


def stage_of(env: Environment, lambda_code) -> PoisedStage:
    return PoisedStage(cdk.App(), env, lambda_code=str(lambda_code))


def api(env, lambda_code) -> Template:
    return Template.from_stack(stage_of(env, lambda_code).api)


def messaging(env, lambda_code) -> Template:
    stage = stage_of(env, lambda_code)
    assert stage.messaging is not None, f"{env.name} has no MessagingStack"
    return Template.from_stack(stage.messaging)


# --- countries.py ---------------------------------------------------------------------------------

GOOD = {f.name: getattr(CA_QA, f.name) for f in dataclasses.fields(Environment)}


@pytest.mark.parametrize("stage", ["training", "production"])
def test_capture_is_refused_outside_qa(stage):
    with pytest.raises(ValueError, match="messaging"):
        Environment(**{**GOOD, "name": f"ca-{stage}", "stage": stage, "messaging": "capture"})


def test_qa_never_sends_to_everyone():
    """Real sending on QA must be named in the entry, and then goes to its test recipients only."""
    with pytest.raises(ValueError, match="qa_test_recipients"):
        Environment(**{**GOOD, "messaging": "send"})


def test_test_recipients_only_on_qa():
    with pytest.raises(ValueError, match="qa_test_recipients"):
        Environment(**{**GOOD, "name": "ca-training", "stage": "training", "messaging": "send", "qa_test_recipients": True})


@pytest.mark.parametrize(
    "field,bad",
    [("messaging", "aws"), ("sms_sender_id", "PREOPSIS HQ"), ("sms_sender_id", "Preopsis"), ("sms_sender_id", ""), ("jobs_every_minutes", 0)],
)
def test_bad_messaging_values_are_refused(field, bad):
    with pytest.raises(ValueError, match=field):
        Environment(**{**GOOD, field: bad})


def test_sends():
    assert not CA_QA.sends and CA_QA_TEST.sends and CA_TRAINING.sends


def test_uk_qa_captures_and_sends_to_its_test_recipients():
    """S17: QA keeps capturing (the journeys and testers read the captured messages), and also
    really sends to the test recipients in SSM (Rahul's own, in the SES and SMS sandbox)."""
    uk = countries.get("uk-qa")
    assert uk.messaging == "capture" and uk.qa_test_recipients
    # S17 answers 7: every 15 minutes on QA, so its Neon database can sleep between runs.
    assert uk.jobs_every_minutes == 15


# --- The jobs Lambda and its schedule ---------------------------------------------------------------


def test_the_jobs_function(lambda_code):
    props = lambda_env({"api": api(CA_QA, lambda_code)}, "poised-ca-qa-jobs")
    assert props["Runtime"] == "provided.al2023" and props["Architectures"] == ["arm64"]
    env = props["Environment"]["Variables"]
    assert env["CONFIG_STRICT"] == "true" and env["MESSAGING"] == "capture" and env["DEFAULT_TIME_ZONE"] == "America/Toronto"
    assert props["Timeout"] <= 300


def test_the_schedule_runs_the_jobs_every_five_minutes(lambda_code):
    t = api(CA_QA, lambda_code)
    t.has_resource_properties(
        "AWS::Scheduler::Schedule",
        {
            "Name": "poised-ca-qa-jobs",
            "ScheduleExpression": "rate(5 minutes)",
            "State": "ENABLED",
            "FlexibleTimeWindow": {"Mode": "OFF"},
            # The next run comes in minutes: a failed one isn't retried for a day.
            "Target": Match.object_like({"RetryPolicy": Match.object_like({"MaximumRetryAttempts": 0})}),
        },
    )
    target = next(iter(t.find_resources("AWS::Scheduler::Schedule").values()))["Properties"]["Target"]
    assert "jobs" in json.dumps(target["Arn"])


def test_the_schedule_follows_the_entry(lambda_code):
    t = api(dataclasses.replace(CA_QA, jobs_every_minutes=15), lambda_code)
    t.has_resource_properties("AWS::Scheduler::Schedule", {"ScheduleExpression": "rate(15 minutes)"})


def test_the_functions_know_the_schedule_and_run_no_timer(lambda_code):
    """Only the schedule (and the testers' "run jobs now") runs the jobs on AWS; the jobs page says how often."""
    t = api(dataclasses.replace(CA_QA, jobs_every_minutes=15), lambda_code)
    for fn in ("staff-api", "patient-api", "jobs"):
        env = lambda_env({"api": t}, f"poised-ca-qa-{fn}")["Environment"]["Variables"]
        assert env["JOBS_TIMER"] == "false" and env["JOBS_SCHEDULE_MINUTES"] == "15"


# --- Capture only: nothing can be sent ---------------------------------------------------------------


def test_capture_only_has_no_messaging_stack_and_no_sending_rights(lambda_code):
    stage = stage_of(CA_QA, lambda_code)
    assert stage.messaging is None
    t = Template.from_stack(stage.api)
    actions = json.dumps([s["Action"] for s in policy_statements(t)])
    assert "ses:" not in actions and "sms-voice:" not in actions
    for fn in ("staff-api", "patient-api", "jobs"):
        env = lambda_env({"api": t}, f"poised-ca-qa-{fn}")["Environment"]["Variables"]
        assert "EMAIL_FROM_PARAM" not in env and "MESSAGING_TEST_RECIPIENTS_PARAM" not in env


# --- MessagingStack ---------------------------------------------------------------------------------


def test_the_ses_sender_comes_from_ssm(lambda_code):
    t = messaging(CA_QA_TEST, lambda_code)
    identity = next(iter(t.find_resources("AWS::SES::EmailIdentity").values()))["Properties"]
    assert "Ref" in identity["EmailIdentity"]  # a deploy-time parameter: no address in the repo
    params = t.to_json()["Parameters"]
    assert any(p.get("Default") == "/sj/qa/email-from" for p in params.values())


def test_ses_reports_failures_to_the_topic(lambda_code):
    t = messaging(CA_QA_TEST, lambda_code)
    t.has_resource_properties("AWS::SES::ConfigurationSet", {"Name": "poised-ca-qa"})
    dest = next(iter(t.find_resources("AWS::SES::ConfigurationSetEventDestination").values()))["Properties"]["EventDestination"]
    assert dest["Enabled"] is True
    assert {e.lower() for e in dest["MatchingEventTypes"]} == {"bounce", "reject", "renderingfailure"}
    assert "SnsDestination" in dest


def test_texts_go_only_to_the_countrys_numbers(lambda_code):
    t = messaging(CA_QA_TEST, lambda_code)
    pc = next(iter(t.find_resources("AWS::SMSVOICE::ProtectConfiguration").values()))["Properties"]
    rules = {r["CountryCode"]: r["ProtectStatus"] for r in pc["CountryRuleSet"]["SMS"]}
    assert set(rules) == set(SMS_COUNTRY_CODES)
    assert rules.pop("CA") == "ALLOW"
    assert set(rules.values()) == {"BLOCK"}
    cs = next(iter(t.find_resources("AWS::SMSVOICE::ConfigurationSet").values()))["Properties"]
    assert cs["ConfigurationSetName"] == "poised-ca-qa"
    assert "Ref" in json.dumps(cs["ProtectConfigurationId"]) or "Fn::GetAtt" in json.dumps(cs["ProtectConfigurationId"])
    (dest,) = cs["EventDestinations"]
    assert set(dest["MatchingEventTypes"]) == SMS_FAILURES
    assert "SnsDestination" in dest and dest["Enabled"] is True


def test_the_services_may_publish_to_the_topic(lambda_code):
    t = messaging(CA_QA_TEST, lambda_code)
    policy = json.dumps(next(iter(t.find_resources("AWS::SNS::TopicPolicy").values())))
    assert "sms-voice.amazonaws.com" in policy and "ses.amazonaws.com" in policy
    assert "aws:SourceAccount" in policy


def test_a_sender_id_is_made_when_the_entry_names_one(lambda_code):
    t = messaging(CA_TRAINING, lambda_code)
    t.has_resource_properties("AWS::SMSVOICE::SenderId", {"SenderId": "PREOPSIS", "IsoCountryCode": "CA"})
    assert len(messaging(CA_QA_TEST, lambda_code).find_resources("AWS::SMSVOICE::SenderId")) == 0


def test_the_jobs_function_hears_the_delivery_events(lambda_code):
    t = api(CA_QA_TEST, lambda_code)
    sub = next(iter(t.find_resources("AWS::SNS::Subscription").values()))["Properties"]
    assert sub["Protocol"] == "lambda" and "jobs" in json.dumps(sub["Endpoint"])


# --- The functions' sending settings -----------------------------------------------------------------


def test_qa_with_test_recipients(lambda_code):
    t = api(CA_QA_TEST, lambda_code)
    for fn in ("staff-api", "patient-api", "jobs"):
        env = lambda_env({"api": t}, f"poised-ca-qa-{fn}")["Environment"]["Variables"]
        assert env["MESSAGING"] == "capture"
        assert env["MESSAGING_TEST_RECIPIENTS_PARAM"] == "/sj/qa/messaging-test-recipients"
        assert env["EMAIL_FROM_PARAM"] == "/sj/qa/email-from"
        assert env["SES_CONFIGURATION_SET"] == "poised-ca-qa" and env["SMS_CONFIGURATION_SET"] == "poised-ca-qa"
        assert "SMS_SENDER_ID" not in env
    ssm = json.dumps([s["Resource"] for s in policy_statements(t) if "ssm:GetParameter" in json.dumps(s["Action"])])
    assert "/sj/qa/email-from" in ssm and "/sj/qa/messaging-test-recipients" in ssm


def test_training_really_sends(lambda_code):
    t = api(CA_TRAINING, lambda_code)
    env = lambda_env({"api": t}, "poised-ca-training-staff-api")["Environment"]["Variables"]
    assert env["MESSAGING"] == "send" and env["SMS_SENDER_ID"] == "PREOPSIS"
    assert "MESSAGING_TEST_RECIPIENTS_PARAM" not in env


def test_the_functions_may_send(lambda_code):
    t = api(CA_QA_TEST, lambda_code)
    statements = policy_statements(t)
    ses = [s for s in statements if "ses:SendEmail" in json.dumps(s["Action"])]
    sms = [s for s in statements if "sms-voice:SendTextMessage" in json.dumps(s["Action"])]
    assert ses and sms
    assert "configuration-set/poised-ca-qa" in json.dumps(ses)


# --- The alarms (OPS-05) -------------------------------------------------------------------------------


def test_failed_deliveries_and_failed_jobs_raise_an_alarm(lambda_code):
    t = api(CA_QA_TEST, lambda_code)
    filters = t.find_resources("AWS::Logs::MetricFilter")
    assert len(filters) == 3  # every function can send
    for f in filters.values():
        p = f["Properties"]
        assert "DELIVERY_FAILED" in p["FilterPattern"]
        assert p["MetricTransformations"][0]["MetricName"] == "DeliveryFailures"
    alarms = [a["Properties"] for a in t.find_resources("AWS::CloudWatch::Alarm").values()]
    names = {a.get("MetricName") for a in alarms}
    assert names == {"DeliveryFailures", "Errors"}
    for a in alarms:
        assert a["ComparisonOperator"] == "GreaterThanOrEqualToThreshold" and a["Threshold"] == 1
        assert a["AlarmActions"]
    # The alert address is the budget's, read from SSM at deploy time.
    t.has_resource_properties("AWS::SNS::Subscription", {"Protocol": "email", "Endpoint": Match.object_like({"Ref": Match.any_value()})})
