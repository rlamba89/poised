"""Test data set up through the real API (plan F1 Step 3), never through the database.

Each journey signs in as a seeded user with the dev stub and builds what it needs here, so the
browser is used only for the journey being tested. Every name gets a unique suffix, so runs
never collide and nothing needs cleaning up.
"""

import json
import secrets
from datetime import datetime
from pathlib import Path

from playwright.sync_api import APIRequestContext, BrowserContext, Locator, Page, Playwright, expect

DATA = Path(__file__).parent / "data"


def unique() -> str:
    """A short suffix that differs on every call, e.g. "1012 154233 7f3a"."""
    return f"{datetime.now():%d%m %H%M%S} {secrets.token_hex(2)}"


def sign_in(context: BrowserContext, name: str, hospital: str = "Hospital A1") -> str:
    """Signs a browser context in as a seeded user (dev stub); returns the start of the named
    hospital's URLs, /o/<trust>/h/<hospital> (S08).

    The cookie lands in the context, so every page opened from it is signed in.
    """
    return _sign_in(context.request, name, hospital)


def _sign_in(request: APIRequestContext, name: str, hospital: str) -> str:
    users = _ok(request.get("/api/dev/users"), "GET /api/dev/users").json()
    match = [u for u in users if u["name"] == name]
    assert match, f"No seeded user called {name!r}. Run `make seed`."
    _ok(request.post("/api/dev/login", data={"userId": match[0]["id"]}), "POST /api/dev/login")
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


def _ok(res, what: str):
    assert res.ok, f"{what} → {res.status}: {res.text()}"
    return res


class Api:
    """One seeded user's API session, at one of their hospitals (Hospital A1 unless named)."""

    def __init__(self, playwright: Playwright, base_url: str, name: str, hospital: str = "Hospital A1"):
        self.request = playwright.request.new_context(base_url=base_url)
        self.base = _sign_in(self.request, name, hospital)

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

    def episode(self, hq: dict, last_name: str) -> dict:
        """A made-up patient and an episode for them on the published HQ."""
        patient = {
            "firstName": "Sam", "lastName": last_name, "dateOfBirth": "1980-05-17", "sex": "female",
            "hospitalNumber": "E2E", "phone": "",
        }
        pid = self.call("POST", self.h("/patients"), patient)["id"]
        eid = self.call("POST", self.h("/episodes"), {
            "patientId": pid, "versionId": hq["versionId"], "procedure": "Knee replacement", "anaesthetic": "General",
            "consultant": "Dr Example",
        })["id"]
        e = self.call("GET", self.h(f"/episodes/{eid}"))["episode"]
        return {"id": eid, "token": e["patientToken"], "name": f"Sam {last_name}", "hq": hq}

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
    """The patient's link, with no sign-in, as their phone would use it."""

    def __init__(self, playwright: Playwright, base_url: str, episode: dict):
        self.request = playwright.request.new_context(base_url=base_url)
        self.episode = episode

    def answer(self, set_name: str, answers: dict) -> None:
        cid = self.episode["hq"]["chapters"][set_name]
        token = self.episode["token"]
        revision = _ok(self.request.get(f"/api/p/{token}"), "GET patient HQ").json()["revisions"].get(cid, 0)
        _ok(self.request.put(f"/api/p/{token}/answers/{cid}", data={"data": answers, "revision": revision}), "PUT patient answers")

    def submit(self) -> None:
        _ok(self.request.post(f"/api/p/{self.episode['token']}/submit"), "POST submit")

    def dispose(self) -> None:
        self.request.dispose()


# The answers the made-up patient gives in J3, and the API sets up for J4 and J5.
PATIENT_ANSWERS = {
    "About you": {"q_smokes": "o_yes", "q_alcohol": "o_some"},
    "Your health": {"q_diabetes": "o_no"},
}
