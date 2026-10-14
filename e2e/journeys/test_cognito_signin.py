"""Staff sign-in with Cognito (S12; STF-05, A-12, ORG-03), where staff sign in through Cognito's
managed login: QA. Locally (the dev login) these are skipped.

  - A clinician signs in with their password and a code from their authenticator app, lands on
    their Episodes, and signs out, which signs them out of the managed login too: the next Sign in
    asks for the password again.
  - A wrong password, and then a wrong code, are refused on Cognito's pages.
  - A suspended trust's staff are refused at sign-in, with the app's message, until it's re-activated.
  - Nothing works without credentials: no dev login, and the testers' tools need a platform admin.
  - The web app sends its security headers.
"""

import re

import pytest
from playwright.sync_api import Page, expect

from support import (
    PLATFORM_ADMIN,
    authenticator_code,
    managed_login,
    managed_login_form,
    qa_request,
    sign_in,
    qa_users,
)

pytestmark = pytest.mark.cognito_only


def test_a_clinician_signs_in_with_password_and_authenticator_then_signs_out(page: Page):
    cara = qa_users()["Cara Clinician"]
    managed_login(page, cara["email"], cara["password"], cara["totpSecret"])
    expect(page).to_have_url(re.compile(r"/o/[0-9a-f-]{36}/h/[0-9a-f-]{36}/episodes$"))
    expect(page.get_by_role("heading", name="Episodes")).to_be_visible()

    episodes = page.url
    page.get_by_role("button", name="Sign out").click()
    expect(page).to_have_url(re.compile(r"/login$"))
    assert page.context.request.get("/api/me").status == 401
    # Back (a shared workstation, test report 1 finding 1) shows nothing of Cara's: the page isn't
    # kept, and the app sends a signed-out browser to the sign-in page.
    page.go_back()
    expect(page).to_have_url(re.compile(r"/login$"))
    expect(page.get_by_role("heading", name="Episodes")).to_have_count(0)
    assert page.url != episodes
    # Signed out of the managed login too: Sign in asks for the password again.
    page.get_by_role("button", name="Sign in", exact=True).click()
    page.wait_for_url(re.compile(r"amazoncognito\.com/"))
    expect(page.get_by_label("Password", exact=True)).to_be_visible()


def test_a_wrong_password_and_a_wrong_code_are_refused(page: Page):
    val = qa_users()["Val Clinician"]
    page.goto("/login")
    page.get_by_role("button", name="Sign in", exact=True).click()
    managed_login_form(page, val["email"], "Not-the-password-1!")
    expect(page.get_by_text("Incorrect username or password.").first).to_be_visible()

    page.get_by_label("Password", exact=True).fill(val["password"])
    page.get_by_role("button", name="Sign in").click()
    page.get_by_label("Code").fill("000000")
    page.get_by_role("button", name="Sign in").click()
    expect(page.get_by_text("Invalid or expired code entered.").first).to_be_visible()

    # The right code still gets in.
    page.get_by_label("Code").fill(authenticator_code(val["totpSecret"]))
    page.get_by_role("button", name="Sign in").click()
    expect(page.get_by_role("heading", name="Episodes")).to_be_visible()


def test_a_suspended_trusts_staff_are_refused_at_sign_in(page: Page, as_user):
    sam = as_user(PLATFORM_ADMIN, None)
    trust_b = next(t for t in sam.call("GET", "/admin/orgs") if t["name"] == "Trust B")
    bea = qa_users()["Bea Author"]  # Trust B only
    sam.call("PATCH", f"/admin/orgs/{trust_b['id']}", {"status": "suspended"})
    try:
        page.goto("/login")
        page.get_by_role("button", name="Sign in", exact=True).click()
        managed_login_form(page, bea["email"], bea["password"])
        page.get_by_label("Code").fill(authenticator_code(bea["totpSecret"]))
        page.get_by_role("button", name="Sign in").click()
        expect(page).to_have_url(re.compile(r"/login$"))
        expect(page.get_by_text("Your trust or hospital is suspended")).to_be_visible()
        assert page.context.request.get("/api/me").status == 401
        # The managed login's session ended too (finding 2): the next Sign in on this computer
        # asks for a password, so someone else can sign in.
        page.get_by_role("button", name="Sign in", exact=True).click()
        page.wait_for_url(re.compile(r"amazoncognito\.com/"))
        expect(page.get_by_label("Password", exact=True)).to_be_visible()
    finally:
        sam.call("PATCH", f"/admin/orgs/{trust_b['id']}", {"status": "active"})


def test_nothing_works_without_credentials(page: Page):
    # No dev login.
    assert page.request.get("/api/dev/users").status == 404
    assert page.request.post("/api/dev/login", data={"userId": "00000000-0000-4000-8000-0000000000e1"}).status == 404
    page.goto("/login")
    expect(page.get_by_role("button", name="Sign in", exact=True)).to_be_visible()
    expect(page.get_by_text("Development sign-in")).to_have_count(0)

    # The testers' tools: signed out, a clinician, then the platform admin.
    assert page.request.get("/api/qa/messages").status == 401
    page.goto("/qa/messages")
    expect(page.get_by_text("Sign in as a platform admin to use this page.")).to_be_visible()
    sign_in(page.context, "Cara Clinician")
    assert page.request.get("/api/qa/jobs").status == 403
    page.goto("/qa/jobs")
    expect(page.get_by_text("Only a platform admin can use this page.")).to_be_visible()
    assert qa_request().get("/api/qa/messages").status == 200


def test_the_web_app_sends_security_headers(page: Page):
    for path in ("/", "/login", "/assets/does-not-exist.js"):
        res = page.request.get(path)
        h = res.headers
        assert h.get("x-frame-options") == "DENY", (path, h)
        assert h.get("content-security-policy") == "frame-ancestors 'none'", (path, h)
        assert h.get("strict-transport-security", "").startswith("max-age=31536000"), (path, h)
        assert h.get("x-content-type-options") == "nosniff", (path, h)
        assert h.get("referrer-policy") == "no-referrer", (path, h)
