"""Fixtures shared by the journeys in journeys/.

From pytest-playwright: `page` (a desktop Chromium tab, new for each test) and its `context`.
Added here:
  - `base_url`: E2E_BASE_URL, default http://localhost:3000 (`make dev`);
  - `as_user("Alex Author")`: that seeded user's API session at Hospital A1 (or the hospital
    named), for setting up data (support.Api);
  - `patient_link(episode)`: the patient's API session, through the link in their captured invite
    and their date of birth (support.Patient, S14);
  - `phone`: a browser context on an emulated iPhone 13 (WebKit), for the patient's journeys.
"""

import json
import os
import urllib.request

import pytest
from playwright.sync_api import expect

import support
from support import Api, Patient

expect.set_options(timeout=15_000)


BASE_URL = os.environ.get("E2E_BASE_URL", "http://localhost:3000")


CONFIG: dict = {}


def pytest_sessionstart(session):
    """Stops the run before any journey, with a clear message, when the app isn't up, or when it
    signs staff in with Cognito (QA, S12) and the test users' sign-ins aren't here."""
    try:
        with urllib.request.urlopen(f"{BASE_URL}/api/health", timeout=5) as res:
            ok = res.status == 200
        with urllib.request.urlopen(f"{BASE_URL}/api/config", timeout=5) as res:
            CONFIG.update(json.load(res))
    except OSError:
        ok = False
    if not ok:
        raise pytest.UsageError(f"The app isn't answering at {BASE_URL}/api/health. Start it first: `make dev`.")
    signin = CONFIG.get("staffSignIn") or {}
    if not signin.get("devLogin") and signin.get("cognito") and not support.USERS_FILE.exists():
        raise pytest.UsageError(
            f"{BASE_URL} signs staff in with Cognito, and the test users' sign-ins aren't in {support.USERS_FILE}. "
            "Fetch them first: `make e2e-users PROFILE=<profile>`."
        )


@pytest.fixture(scope="session", autouse=True)
def _run(playwright, base_url):
    """Tells support.py how staff sign in here, and closes what it opened at the end."""
    support.setup(playwright, base_url, CONFIG)
    yield
    support.teardown()


def pytest_collection_modifyitems(config, items):
    """Journeys marked dev_login need the dev stub: they sign in as people they've just invited,
    whose temporary password Cognito emails (QA's Cognito sends about 50 emails a day, and the
    journeys can't read them). On QA they're skipped; the API tests and the local run cover them,
    and a real invite is checked by hand (S12)."""
    signin = CONFIG.get("staffSignIn") or {}
    if signin.get("devLogin", True):
        skip = pytest.mark.skip(reason="Cognito only: this deployment uses the dev login")
        for item in items:
            if "cognito_only" in item.keywords:
                item.add_marker(skip)
        return
    skip = pytest.mark.skip(reason="needs the dev login (local only): it signs in as people it invites")
    for item in items:
        if "dev_login" in item.keywords:
            item.add_marker(skip)


@pytest.fixture(scope="session")
def base_url() -> str:
    return BASE_URL


@pytest.fixture(scope="session")
def browser_context_args(browser_context_args):
    """Staff journeys run on a desktop-sized window."""
    return {**browser_context_args, "viewport": {"width": 1440, "height": 900}}


@pytest.fixture
def as_user(playwright, base_url):
    sessions = []

    def make(name: str, hospital: str | None = "Hospital A1") -> Api:
        api = Api(playwright, base_url, name, hospital)
        sessions.append(api)
        return api

    yield make
    for api in sessions:
        api.dispose()


@pytest.fixture
def patient_link(playwright, base_url):
    sessions = []

    def make(episode: dict) -> Patient:
        p = Patient(playwright, base_url, episode)
        sessions.append(p)
        return p

    yield make
    for p in sessions:
        p.dispose()


@pytest.fixture
def phone(playwright, base_url, browser_type_launch_args, _artifacts_recorder):
    """An iPhone 13 (Safari's engine, WebKit), as a patient would use.

    It's registered with pytest-playwright's recorder (a private fixture, so check it when
    upgrading pytest-playwright), so a failing test keeps this context's trace too.
    """
    browser = playwright.webkit.launch(**browser_type_launch_args)
    context = browser.new_context(**playwright.devices["iPhone 13"], base_url=base_url)
    _artifacts_recorder.on_did_create_browser_context(context)
    yield context
    _artifacts_recorder.on_will_close_browser_context(context)
    context.close()
    browser.close()
