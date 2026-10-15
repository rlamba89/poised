"""Every deployment of Poised: one entry per country and environment (A-20).

This is the only file that may name a region, an account, a web address or a country default
(time zone, locale, phone country, patient identifier). There's no dev login on AWS: staff sign in
with Cognito everywhere, QA included (S12), so an entry can't ask for it. The stacks take an entry and pass its
values to the Lambdas as environment variables; the Go API reads them in internal/config.

Adding a country is a new entry here plus the runbook in README.md, never a code change.
Moving to another AWS account is a change to `account` here plus the same runbook.

Run `python countries.py <name> <field>` to print one value (the Makefile's scripts do).
"""

from __future__ import annotations

import dataclasses
import re
import sys
import zoneinfo

STAGES = ("qa", "training", "production")


@dataclasses.dataclass(frozen=True)
class Environment:
    name: str  # "uk-qa": in every stack and resource name, and the poised-env cost tag
    stage: str  # one of STAGES
    account: str  # the AWS account it's deployed to
    region: str  # the AWS region: the country's own (A-20)
    time_zone: str  # DEFAULT_TIME_ZONE: an IANA zone
    locale: str  # DEFAULT_LOCALE: language-REGION, for formatting dates
    phone_country: str  # PHONE_COUNTRY: the only country SMS goes to (PAT-11)
    national_id_type: str  # NATIONAL_ID_TYPE: a scheme in apps/api/internal/nationalid
    # The web address. None until there's a domain: then it's Amplify's own address,
    # https://main.<app id>.amplifyapp.com, which is known only once the app exists.
    app_url: str | None = None
    # MESSAGING (S17): "capture" stores texts and emails (QA only: the journeys and testers read
    # them); "send" sends them with SES and End User Messaging SMS (Training and Production).
    messaging: str = "capture"
    # QA only: real sending on QA must be named here. Messages are still captured, and the ones to
    # the test recipients in SSM (<prefix>/messaging-test-recipients, put there by hand; none: no
    # one) are also really sent. In SES's and SMS's sandbox those must be verified (README.md).
    qa_test_recipients: bool = False
    # SMS_SENDER_ID: the texts' origination identity (a registered sender ID); None lets AWS choose.
    sms_sender_id: str | None = None
    jobs_every_minutes: int = 5  # the jobs Lambda's schedule (reminders, "not responding")
    budget_usd: int = 6  # the monthly budget alarm, about £5 (AWS Budgets bills in USD)

    @property
    def sends(self) -> bool:
        """Whether any message is really sent here, so the stage needs a MessagingStack."""
        return self.messaging == "send" or self.qa_test_recipients

    @property
    def param_prefix(self) -> str:
        """Where the environment's SSM parameters live, e.g. /sj/qa/database-url."""
        return f"/sj/{self.stage}"

    def __post_init__(self) -> None:
        errors = []

        def check(ok: bool, field: str, want: str) -> None:
            if not ok:
                errors.append(f"{field} must be {want}, not {getattr(self, field)!r}")

        check(bool(re.fullmatch(r"[a-z]{2}-[a-z]+", self.name)), "name", "country-stage, e.g. uk-qa")
        check(self.stage in STAGES, "stage", f"one of {STAGES}")
        check(bool(re.fullmatch(r"\d{12}", self.account)), "account", "a 12-digit AWS account ID")
        check(bool(re.fullmatch(r"[a-z]{2}(-[a-z]+)+-\d", self.region)), "region", "an AWS region")
        check(_is_zone(self.time_zone), "time_zone", "an IANA time zone")
        # Stricter than the Go API's check: a well-shaped tag such as en-12 or en-gb-gb makes the
        # browser's Intl.DateTimeFormat throw, which would break every date (pending.md, S05).
        check(bool(re.fullmatch(r"[a-z]{2,3}-[A-Z]{2}", self.locale)), "locale", "language-REGION, e.g. en-CA")
        check(bool(re.fullmatch(r"[A-Z]{2}", self.phone_country)), "phone_country", "a two-letter country code")
        check(bool(re.fullmatch(r"[a-z]+", self.national_id_type)), "national_id_type", "a scheme name, e.g. nhs")
        check(
            self.app_url is None or bool(re.fullmatch(r"https://[a-z0-9.-]+", self.app_url)),
            "app_url",
            "None or an https address with no path or trailing slash",
        )
        check(self.budget_usd > 0, "budget_usd", "more than 0")
        check(self.messaging in ("capture", "send"), "messaging", '"capture" or "send"')
        if self.messaging == "capture" and self.stage != "qa":
            errors.append(f'messaging must be "send" on {self.stage}: only QA captures messages instead of sending them')
        if self.messaging == "send" and self.stage == "qa":
            errors.append('messaging "send" isn\'t allowed on QA: keep "capture" and set qa_test_recipients=True to send to the test recipients only')
        if self.qa_test_recipients and (self.stage != "qa" or self.messaging != "capture"):
            errors.append("qa_test_recipients is for QA in capture mode only")
        check(
            # CloudFormation's AWS::SMSVOICE::SenderId takes capitals only (found deploying S17).
            self.sms_sender_id is None or bool(re.fullmatch(r"[A-Z0-9-]{1,11}", self.sms_sender_id)),
            "sms_sender_id",
            "None or 1 to 11 capital letters, digits or hyphens",
        )
        check(1 <= self.jobs_every_minutes <= 1440, "jobs_every_minutes", "from 1 to 1440")
        if errors:
            raise ValueError(f"countries.py, {self.name}: " + "; ".join(errors))


def _is_zone(name: str) -> bool:
    if name in ("", "Local") or "/" not in name:
        return False
    try:
        zoneinfo.ZoneInfo(name)
    except (zoneinfo.ZoneInfoNotFoundError, ValueError):
        return False
    return True


ENVIRONMENTS: list[Environment] = [
    Environment(
        name="uk-qa",
        stage="qa",
        # A temporary account (14 Oct 2026): QA moves to its own account before sales.
        account="026090536258",
        region="eu-west-2",
        time_zone="Europe/London",
        locale="en-GB",
        phone_country="GB",
        national_id_type="nhs",
        app_url=None,  # no domain yet: Amplify's own address
        # S17: still captured, and also really sent to the test recipients in SSM (Rahul's own, in
        # SES's and SMS's sandbox). Remove the parameter to send to no one.
        qa_test_recipients=True,
        # Every 15 minutes, not 5 (Rahul, 14 Oct, S17 answers 7): each run wakes QA's Neon database,
        # which sleeps after 5 idle minutes, and 5 would keep it awake past the free plan's hours.
        jobs_every_minutes=15,
        # S17 answers 9: try a UK alphanumeric sender ID (the account has no origination identity).
        sms_sender_id="POISED",
    ),
]


@dataclasses.dataclass(frozen=True)
class PipelineHome:
    """Where the pipeline runs (one for every environment) and what it watches (plan F4 Step 2)."""

    account: str
    region: str
    repo: str  # GitHub owner/name
    branch: str  # a push to it runs the pipeline

    def __post_init__(self) -> None:
        if not re.fullmatch(r"\d{12}", self.account):
            raise ValueError(f"countries.py, PIPELINE: account must be a 12-digit AWS account ID, not {self.account!r}")
        if not re.fullmatch(r"[a-z]{2}(-[a-z]+)+-\d", self.region):
            raise ValueError(f"countries.py, PIPELINE: region must be an AWS region, not {self.region!r}")
        if not re.fullmatch(r"[\w.-]+/[\w.-]+", self.repo):
            raise ValueError(f"countries.py, PIPELINE: repo must be owner/name, not {self.repo!r}")


# The pipeline lives with the first country's QA. It follows `master`, the GitHub repo's default
# branch (S07, 14 Oct 2026: the plan said `main`; no branch was renamed).
PIPELINE = PipelineHome(account="026090536258", region="eu-west-2", repo="rlamba89/poised", branch="master")


def get(name: str) -> Environment:
    for env in ENVIRONMENTS:
        if env.name == name:
            return env
    raise KeyError(f"no environment {name!r} in countries.py; there are {[e.name for e in ENVIRONMENTS]}")


if __name__ == "__main__":
    if len(sys.argv) != 3:
        sys.exit("usage: python countries.py <environment or pipeline> <field>, e.g. uk-qa region")
    entry = PIPELINE if sys.argv[1] == "pipeline" else get(sys.argv[1])
    print(getattr(entry, sys.argv[2]))
