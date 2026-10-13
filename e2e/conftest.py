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

import os
import urllib.request

import pytest
from playwright.sync_api import expect

from support import Api, Patient

expect.set_options(timeout=15_000)


BASE_URL = os.environ.get("E2E_BASE_URL", "http://localhost:3000")


def pytest_sessionstart(session):
    """Stops the run before any journey, with a clear message, when the app isn't up."""
    try:
        with urllib.request.urlopen(f"{BASE_URL}/api/health", timeout=5) as res:
            ok = res.status == 200
    except OSError:
        ok = False
    if not ok:
        raise pytest.UsageError(f"The app isn't answering at {BASE_URL}/api/health. Start it first: `make dev`.")


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
