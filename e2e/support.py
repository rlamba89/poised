"""Test data set up through the real API (plan F1 Step 3), never through the database.

Each journey signs in as a seeded user and builds what it needs here, so the browser is used only
for the journey being tested. Every name gets a unique suffix, so runs never collide and nothing
needs cleaning up.

How a seeded user signs in depends on the deployment (GET /api/config, S12):
  - locally, the dev stub (POST /api/dev/login);
  - on QA, Cognito's managed login, in a browser, with the test user's password and a code from
    their authenticator secret (pyotp). Those come from e2e/.qa-users.json (`make e2e-users`, or
    the pipeline's E2E step), or the file named by E2E_TEST_USERS. Each user signs in once per run;
    the session cookie is then reused (Cognito takes each authenticator code only once).
The testers' tools (captured messages, scheduled jobs) need a platform admin on QA, so they're
used as Sam Superadmin (qa_request, qa_page); locally they're open, and Sam works there too.
"""

import json
import os
import re
import secrets
import time
from datetime import datetime
from pathlib import Path

import pyotp
from playwright.sync_api import APIRequestContext, BrowserContext, Locator, Page, Playwright, expect
from playwright.sync_api import TimeoutError as PlaywrightTimeoutError

DATA = Path(__file__).parent / "data"
USERS_FILE = Path(os.environ.get("E2E_TEST_USERS", Path(__file__).parent / ".qa-users.json"))
PLATFORM_ADMIN = "Sam Superadmin"
COOKIE = "sj_token"  # the staff session cookie (apps/api/internal/auth)


class _Run:
    """What the whole run shares: set once by conftest (setup), closed at the end (teardown)."""

    playwright: Playwright | None = None
    base_url = ""
    staff_sign_in: dict = {"devLogin": True, "cognito": None}
    cookies: dict[str, dict] = {}  # Cognito: each user's session cookie, from their one sign-in
    last_code: dict[str, str] = {}  # the authenticator code each secret last used
    qa: APIRequestContext | None = None
    contexts: list[BrowserContext] = []


def setup(playwright: Playwright, base_url: str, config: dict) -> None:
    _Run.playwright, _Run.base_url = playwright, base_url
    _Run.staff_sign_in = config.get("staffSignIn") or {"devLogin": True, "cognito": None}


def teardown() -> None:
    for ctx in _Run.contexts:
        ctx.close()
    if _Run.qa:
        _Run.qa.dispose()


def cognito() -> bool:
    """Whether staff sign in through Cognito here (QA), not the dev login (local)."""
    return not _Run.staff_sign_in.get("devLogin") and bool(_Run.staff_sign_in.get("cognito"))


def qa_users() -> dict[str, dict]:
    """The QA test users' sign-ins, by name: {name, email, password, totpSecret}."""
    return {u["name"]: u for u in json.loads(USERS_FILE.read_text())}


def authenticator_code(secret: str) -> str:
    """The current code, but never the one this secret last used: Cognito takes a code once, so
    a second sign-in in the same 30 seconds waits for the next one."""
    totp = pyotp.TOTP(secret)
    code = totp.now()
    if _Run.last_code.get(secret) == code:
        time.sleep(30 - time.time() % 30 + 1)
        code = totp.now()
    _Run.last_code[secret] = code
    return code


def managed_login(page: Page, email: str, password: str, secret: str) -> None:
    """Signs in as a person does on QA: Sign in, then Cognito's managed login (email, password,
    the authenticator's code), back to the app (S12)."""
    page.goto("/login")
    page.get_by_role("button", name="Sign in", exact=True).click()
    managed_login_form(page, email, password)
    expect(page.get_by_role("heading", name="Authenticator app MFA")).to_be_visible()
    back = re.compile(r"^" + re.escape(_Run.base_url) + r"/(?!auth/callback|login)")
    for attempt in range(2):
        page.get_by_label("Code").fill(authenticator_code(secret))
        page.get_by_role("button", name="Sign in").click()
        refused = page.get_by_text("Invalid or expired code entered.").first
        try:
            page.wait_for_url(back, timeout=10_000)
            return
        except PlaywrightTimeoutError:
            # The code was used moments ago (e.g. by another run): Cognito takes each once.
            if attempt == 1 or not refused.is_visible():
                raise
            _Run.last_code[secret] = pyotp.TOTP(secret).now()  # wait for the next one


def managed_login_form(page: Page, email: str, password: str) -> None:
    """The managed login's first page: email and password (Cognito's page, S12)."""
    page.wait_for_url(re.compile(r"amazoncognito\.com/"))
    page.get_by_label("Email address").fill(email)
    page.get_by_label("Password", exact=True).fill(password)
    page.get_by_role("button", name="Sign in").click()


def session_cookie(name: str) -> dict:
    """Cognito: the user's session cookie, signing them in through the managed login the first time."""
    if name not in _Run.cookies:
        users = qa_users()
        assert name in users, f"No QA test user called {name!r}; there are {sorted(users)}. Run `make qa-users`."
        u = users[name]
        browser = _Run.playwright.chromium.launch()
        try:
            ctx = browser.new_context(base_url=_Run.base_url)
            managed_login(ctx.new_page(), u["email"], u["password"], u["totpSecret"])
            [cookie] = [c for c in ctx.cookies() if c["name"] == COOKIE]
            _Run.cookies[name] = cookie
        finally:
            browser.close()
    return _Run.cookies[name]


def qa_request() -> APIRequestContext:
    """An API session that may use the testers' tools: the platform admin's."""
    if _Run.qa is None:
        _Run.qa = _signed_in_request(PLATFORM_ADMIN)
    return _Run.qa


def qa_page(page: Page) -> Page:
    """A tab, in its own browser context, signed in as the platform admin, for the testers' pages
    (/qa/messages, /qa/jobs), which need one on QA (S12). The journey's own `page` stays as it is."""
    ctx = page.context.browser.new_context(base_url=_Run.base_url, viewport={"width": 1440, "height": 900})
    _Run.contexts.append(ctx)
    sign_in(ctx, PLATFORM_ADMIN, None)
    return ctx.new_page()


def _signed_in_request(name: str) -> APIRequestContext:
    if cognito():
        return _Run.playwright.request.new_context(
            base_url=_Run.base_url, storage_state={"cookies": [session_cookie(name)], "origins": []}
        )
    request = _Run.playwright.request.new_context(base_url=_Run.base_url)
    _dev_sign_in(request, name)
    return request


def unique() -> str:
    """A short suffix that differs on every call, e.g. "1012 154233 7f3a"."""
    return f"{datetime.now():%d%m %H%M%S} {secrets.token_hex(2)}"


def nhs_number() -> str:
    """A made-up NHS number with a valid check digit (Modulus 11), in the 999 range, different
    almost every call, so each run's patients are new to the trust."""
    while True:
        body = "999" + "".join(secrets.choice("0123456789") for _ in range(6))
        check = 11 - sum(int(d) * w for d, w in zip(body, range(10, 1, -1))) % 11
        if check == 11:
            check = 0
        if check != 10:
            return body + str(check)


def sign_in(context: BrowserContext, name: str, hospital: str | None = "Hospital A1") -> str:
    """Signs a browser context in as a seeded user (the dev stub locally, Cognito on QA); returns
    the start of the named hospital's URLs, /o/<trust>/h/<hospital> (S08), or "" for None.

    The cookie lands in the context, so every page opened from it is signed in.
    """
    if cognito():
        context.add_cookies([session_cookie(name)])
    else:
        _dev_sign_in(context.request, name)
    return _place(context.request, name, hospital)


def _dev_sign_in(request: APIRequestContext, name: str) -> None:
    users = _ok(request.get("/api/dev/users"), "GET /api/dev/users").json()
    match = [u for u in users if u["name"] == name]
    assert match, f"No seeded user called {name!r}. Run `make seed`."
    _ok(request.post("/api/dev/login", data={"userId": match[0]["id"]}), "POST /api/dev/login")


def _place(request: APIRequestContext, name: str, hospital: str | None) -> str:
    if hospital is None:  # e.g. the platform admin, who works at no hospital
        return ""
    me = _ok(request.get("/api/me"), "GET /api/me").json()
    places = [f"/o/{t['id']}/h/{h['id']}" for t in me["trusts"] for h in t["hospitals"] if h["name"] == hospital]
    assert places, f"{name} can't work at {hospital!r}. Run `make seed`."
    return places[0]


def choose(scope: Page | Locator, question: str, option: str) -> None:
    """Picks a SurveyJS option as a person does, by tapping its text (the real radio is hidden)."""
    group = scope.get_by_role("radiogroup", name=question)
    group.get_by_text(option, exact=True).click()
    expect(group.get_by_role("radio", name=option)).to_be_checked()


def pick(page: Page, label: str, option: str) -> None:
    """Picks an option from a dropdown (Mantine Select) by its label."""
    page.get_by_role("combobox", name=label).click()
    page.get_by_role("option", name=option, exact=True).click()
    expect(page.get_by_role("combobox", name=label)).to_have_value(option)


INVITE = re.compile(r"/i/([A-Za-z0-9_-]{43})")


def captured(to: str) -> list[dict]:
    """The captured texts and emails to `to`, newest first (MESSAGING=capture, S14), read as the
    platform admin (QA's tools need one, S12)."""
    return _ok(qa_request().get("/api/qa/messages", params={"to": to}), "GET /api/qa/messages").json()


def invite_path(to: str) -> str:
    """The path of the link in the newest invite sent to `to` (a mobile in E.164, or an email):
    /i/<token>. The app's address in it is APP_URL; the journeys open the path on their own."""
    for m in captured(to):
        found = INVITE.search(m["body"])
        if found:
            return found.group(0)
    raise AssertionError(f"No invite has been captured for {to}")


SIGNIN = re.compile(r"/s/([A-Za-z0-9_-]{43})")
VERIFY = re.compile(r"/v/([A-Za-z0-9_-]{43})")


def link_path(to: str, pattern: re.Pattern = SIGNIN) -> str:
    """The path of the newest captured link matching `pattern` sent to `to`: a sign-in link
    (/s/<token>, S15) unless another pattern is given (VERIFY: a new contact's /v/<token>)."""
    for m in captured(to):
        found = pattern.search(m["body"])
        if found:
            return found.group(0)
    raise AssertionError(f"No {pattern.pattern} link has been captured for {to}")


def _ok(res, what: str):
    assert res.ok, f"{what} → {res.status}: {res.text()}"
    return res


class Api:
    """One seeded user's API session, at one of their hospitals (Hospital A1 unless named; None for
    the platform admin)."""

    def __init__(self, playwright: Playwright, base_url: str, name: str, hospital: str | None = "Hospital A1"):
        self.request = _signed_in_request(name)
        self.base = _place(self.request, name, hospital)

    def call(self, method: str, path: str, body=None):
        """Sends a request to /api<path> and returns the decoded JSON (None for 204)."""
        res = _ok(self.request.fetch(f"/api{path}", method=method, data=body), f"{method} {path}")
        return None if res.status == 204 else res.json()

    def h(self, path: str) -> str:
        """A path of the hospital: /o/<trust>/h/<hospital><path>."""
        return f"{self.base}{path}"

    def o(self, path: str) -> str:
        """A path of the hospital's trust: /o/<trust><path>."""
        return f"{self.base.split('/h/')[0]}{path}"

    # Staff (S09)

    def invite(self, email: str, name: str, hospital_id: str | None, role: str) -> dict:
        """Invites someone to this trust: hospital_id None is the whole trust. Needs a whole-trust admin."""
        return self.call("POST", self.o("/staff/invites"), {"email": email, "name": name, "hospitalId": hospital_id, "role": role})

    # The platform admin (S10)

    def new_trust(self, name: str, code: str, admin_email: str, admin_name: str) -> dict:
        """Creates a trust and invites its first admin. Needs the platform admin (Sam Superadmin)."""
        return self.call("POST", "/admin/orgs", {
            "name": name, "code": code, "contactName": "E2E Contact", "contactEmail": "contact@e2e.example",
            "adminEmail": admin_email, "adminName": admin_name, "optIns": {},
        })

    def new_hospital(self, trust_id: str, name: str) -> dict:
        """Adds a hospital to a trust and returns the trust. Needs the platform admin."""
        return self.call("POST", f"/admin/orgs/{trust_id}/hospitals", {"name": name, "address": "1 E2E Road"})

    # Questionnaires

    def draft_hq(self, data_file: str, name: str) -> dict:
        """Creates a draft from e2e/data/<data_file>: its Question Sets, audiences and content."""
        spec = json.loads((DATA / data_file).read_text())
        qid = self.call("POST", self.h("/questionnaires"), {"name": name, "description": spec["description"]})["id"]
        chapters = {}
        for c in spec["chapters"]:
            cid = self.call("POST", self.h(f"/questionnaires/{qid}/chapters"), {"name": c["name"]})["id"]
            self.call("PATCH", self.h(f"/chapters/{cid}"), {"audience": c["audience"]})
            revision = self.call("GET", self.h(f"/chapters/{cid}"))["revision"]
            self.call("PUT", self.h(f"/chapters/{cid}/content"), {"content": c["content"], "revision": revision})
            chapters[c["name"]] = cid
        version_id = self.call("GET", self.h(f"/questionnaires/{qid}"))["questionnaire"]["versionId"]
        return {"id": qid, "name": name, "versionId": version_id, "chapters": chapters}

    def published_hq(self, data_file: str, name: str) -> dict:
        hq = self.draft_hq(data_file, name)
        self.call("POST", self.h(f"/questionnaires/{hq['id']}/publish"), {"versionId": hq["versionId"]})
        return hq

    # Episodes

    def episode(self, hq: dict, last_name: str, hospital_number: str = "E2E", **patient) -> dict:
        """An episode on the published HQ for a made-up patient, typed as on the New episode page
        (S13): Sam <last_name>, born 1980-05-17, with a new made-up email (so their invite can be
        found, S14), unless `patient` says otherwise (e.g. nationalId=…, mobile=…). The trust's
        record is linked to silently if it matches one."""
        email = f"e2e.{secrets.token_hex(5)}@e2e.example"
        details = {"firstName": "Sam", "lastName": last_name, "dateOfBirth": "1980-05-17", "sex": "female", "email": email, **patient}
        eid = self.call("POST", self.h("/episodes"), {
            "patient": details, "versionId": hq["versionId"], "hospitalNumber": hospital_number,
            "procedure": "Knee replacement", "anaesthetic": "General", "consultant": "Dr Example",
        })["id"]
        e = self.call("GET", self.h(f"/episodes/{eid}"))["episode"]
        return {"id": eid, "name": f"{e['firstName']} {e['lastName']}", "hq": hq, "patientId": e["patientId"],
                "email": e["email"], "mobile": e["mobile"], "dob": details["dateOfBirth"]}

    def validate(self, episode: dict, set_name: str, answers: dict) -> None:
        """Saves the clinician's answers for one Question Set and validates it."""
        cid = episode["hq"]["chapters"][set_name]
        rows = self.call("GET", self.h(f"/episodes/{episode['id']}/hq"))["answers"]
        revision = next((a["revision"] for a in rows if a["chapterId"] == cid and a["actor"] == "clinician"), 0)
        self.call("PUT", self.h(f"/episodes/{episode['id']}/answers/{cid}"),
                  {"data": answers, "validated": True, "revision": revision})

    def complete_review(self, episode: dict) -> None:
        self.call("POST", self.h(f"/episodes/{episode['id']}/complete-review"))

    def dispose(self) -> None:
        self.request.dispose()


class Patient:
    """The patient, as their phone would be (S14): the link from their captured invite email,
    Continue and their date of birth, then their session (a cookie this context keeps)."""

    def __init__(self, playwright: Playwright, base_url: str, episode: dict):
        self.request = playwright.request.new_context(base_url=base_url)
        self.episode = episode
        token = invite_path(episode["email"]).removeprefix("/i/")
        _ok(self.request.post(f"/api/p/links/{token}/continue"), "POST continue")
        _ok(self.request.post(f"/api/p/links/{token}/dob", data={"dateOfBirth": episode["dob"]}), "POST date of birth")
        self.path = f"/api/p/episodes/{episode['id']}"

    def answer(self, set_name: str, answers: dict) -> None:
        cid = self.episode["hq"]["chapters"][set_name]
        revision = _ok(self.request.get(self.path), "GET patient HQ").json()["revisions"].get(cid, 0)
        _ok(self.request.put(f"{self.path}/answers/{cid}", data={"data": answers, "revision": revision}), "PUT patient answers")

    def submit(self) -> None:
        _ok(self.request.post(f"{self.path}/submit"), "POST submit")

    def dispose(self) -> None:
        self.request.dispose()


# The answers the made-up patient gives in J3, and the API sets up for J4 and J5.
PATIENT_ANSWERS = {
    "About you": {"q_smokes": "o_yes", "q_alcohol": "o_some"},
    "Your health": {"q_diabetes": "o_no"},
}
