"""A patient closes their account (S22, EPI-09; S22 answers 5), on their phone (iPhone 13).

  - The patient opens their invite, starts nothing, and goes to their home. "Close my account"
    asks first, warning that their unsent questionnaire won't be sent; "Keep my account" backs out.
  - Confirmed, the home is gone ("Your account is closed"), a reload finds them signed out, and the
    old invite link no longer works.
  - Cara still sees the patient and the episode, with "The patient closed their account on …" in
    the General notes. Her new invite starts a fresh account: the patient's home is back.
"""

import secrets

from playwright.sync_api import BrowserContext, Page, expect

from support import invite_path, sign_in, unique


def enter_dob(patient: Page, day: str, month: str, year: str) -> None:
    patient.get_by_label("Day").fill(day)
    patient.get_by_label("Month").fill(month)
    patient.get_by_label("Year").fill(year)
    patient.get_by_role("button", name="Continue").click()


def open_invite(patient: Page, email: str, dob: tuple[int, int, int]) -> None:
    patient.goto(invite_path(email))
    patient.get_by_role("button", name="Continue").click()
    day, month, year = dob
    enter_dob(patient, str(day), str(month), str(year))


def test_a_patient_closes_their_account(page: Page, phone: BrowserContext, as_user):
    dob = (secrets.randbelow(28) + 1, secrets.randbelow(12) + 1, 1940 + secrets.randbelow(50))
    email = f"close.{secrets.token_hex(4)}@e2e.example"
    hq = as_user("Alex Author").published_hq("workflow-hq.json", f"E2E Close {unique()}")
    episode = as_user("Cara Clinician").episode(hq, f"Close {unique()}", email=email, dateOfBirth=f"{dob[2]}-{dob[1]:02d}-{dob[0]:02d}")

    patient = phone.new_page()
    open_invite(patient, email, dob)
    expect(patient.get_by_role("heading", name=hq["name"])).to_be_visible()
    old_invite = invite_path(email)
    patient.get_by_role("link", name="Your questionnaires").click()
    expect(patient.get_by_test_id("home-episode")).to_have_count(1)

    # Close my account asks first, warning about the questionnaire not yet sent.
    card = patient.get_by_test_id("close-account")
    card.get_by_role("button", name="Close my account").click()
    expect(card.get_by_test_id("close-account-unsent")).to_contain_text(f"Your {hq['name']} for Hospital A1 hasn't been sent.")
    expect(card.get_by_test_id("close-account-unsent")).to_contain_text("it won't be sent")
    assert patient.evaluate("document.documentElement.scrollWidth <= window.innerWidth")
    card.get_by_role("button", name="Keep my account").click()
    expect(card.get_by_test_id("close-account-unsent")).to_have_count(0)
    card.get_by_role("button", name="Close my account").click()
    card.get_by_role("button", name="Close my account").click()  # the confirmation's

    # The account is closed: the home is gone, and so is the session.
    expect(patient.get_by_test_id("account-closed")).to_contain_text("Your account is closed")
    patient.reload()
    expect(patient.get_by_test_id("patient-signed-out")).to_be_visible()
    patient.goto(old_invite)
    patient.get_by_role("button", name="Continue").click()
    expect(patient.get_by_role("heading", name="This link doesn't work")).to_be_visible()

    # Staff still see the patient and the episode, with the note.
    base = sign_in(page.context, "Cara Clinician")
    page.goto(f"{base}/episodes/{episode['id']}")
    expect(page.get_by_text("The patient closed their account on")).to_be_visible()
    # Its invite link was revoked with the account, so the card says so and offers a new one (test report 1).
    expect(page.get_by_test_id("invite")).to_contain_text("The patient's link no longer works")
    page.get_by_test_id("invite").get_by_role("button", name="Send invite").click()
    expect(page.get_by_test_id("invite").get_by_text("Invite sent again")).to_be_visible()

    # The new invite starts a fresh account.
    open_invite(patient, email, dob)
    expect(patient.get_by_role("heading", name=hq["name"])).to_be_visible()
    patient.get_by_role("link", name="Your questionnaires").click()
    expect(patient.get_by_test_id("home-episode")).to_have_count(1)
