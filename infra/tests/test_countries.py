"""countries.py (A-20): the only place a region, account, web address or country default is
written. Its entries are checked when they're made, so a bad one stops `cdk synth`."""

import dataclasses
import pathlib
import re

import pytest

import countries
from countries import Environment

GOOD = dict(
    name="ca-qa",
    stage="qa",
    account="111111111111",
    region="ca-central-1",
    time_zone="America/Toronto",
    locale="en-CA",
    phone_country="CA",
    national_id_type="nhs",
)


def test_uk_qa_is_the_only_entry_for_now():
    assert [e.name for e in countries.ENVIRONMENTS] == ["uk-qa"]
    uk = countries.get("uk-qa")
    assert uk.stage == "qa" and uk.dev_login
    assert uk.param_prefix == "/sj/qa"


def test_get_an_unknown_entry():
    with pytest.raises(KeyError, match="no-such"):
        countries.get("no-such")


@pytest.mark.parametrize(
    "field,bad",
    [
        ("name", "CA QA"),
        ("stage", "staging"),
        ("account", "12345"),
        ("region", "canada"),
        ("time_zone", "Mars/Olympus"),
        ("time_zone", "Local"),
        # Well-shaped tags that make the browser's Intl.DateTimeFormat throw (pending.md, S05).
        ("locale", "en-12"),
        ("locale", "en-gb-gb"),
        ("locale", "english"),
        ("phone_country", "can"),
        ("national_id_type", ""),
        ("app_url", "http://ca.example.test"),
        ("app_url", "https://ca.example.test/"),
        ("budget_usd", 0),
    ],
)
def test_bad_values_are_refused(field, bad):
    with pytest.raises(ValueError, match=field.upper() if field == "dev_login" else field):
        Environment(**{**GOOD, field: bad})


def test_dev_login_only_in_qa():
    with pytest.raises(ValueError, match="DEV_LOGIN"):
        Environment(**{**GOOD, "name": "ca-production", "stage": "production", "dev_login": True})
    assert not Environment(**{**GOOD, "stage": "production"}).dev_login


def test_names_are_unique():
    names = [e.name for e in countries.ENVIRONMENTS]
    assert len(names) == len(set(names))


# A-20: no region, time zone or locale literal in the CDK code outside countries.py (as the Go
# guard test does for apps/api, apps/web and packages/clinical).
LITERAL = re.compile(
    r"""["']([a-z]{2}-[A-Z]{2}"""
    r"""|(Africa|America|Antarctica|Asia|Atlantic|Australia|Europe|Indian|Pacific)/[A-Za-z_]+"""
    r"""|[a-z]{2}-(north|south|east|west|central|northeast|southeast)-\d)["']"""
)


def test_no_country_literals_outside_countries_py():
    infra = pathlib.Path(__file__).resolve().parent.parent
    scanned = 0
    for path in infra.glob("*.py"):
        if path.name == "countries.py":
            continue
        scanned += 1
        for i, line in enumerate(path.read_text().splitlines(), 1):
            code = line.split("#", 1)[0]
            assert not LITERAL.search(code), f"{path.name}:{i} has a per-country literal: {line.strip()}"
    assert scanned >= 4
