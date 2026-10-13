"""J7c A returning patient (S15; A-3, PAT-06, PAT-07, PAT-10), on a phone (iPhone 13).

  - J7c: Sam has a questionnaire from Trust A (they opened its invite once, which made their
    account) and a new one from Trust B, sent to the same email. They ask for a sign-in link by
    email, open it, Continue, date of birth, and their home lists both trusts' questionnaires,
    newest first. They open one and come back, sign out, and the used link no longer works.
  - Changing their email: from the home, a link goes to the new address; opening it there
    changes it, and the home shows it.
  - Twins on one contact with one date of birth (S15 answers 6 and 8): the sign-in link asks for
    the first name and surname too, and each twin sees only their own questionnaires, labelled with their name.
"""

import re
import secrets

from playwright.sync_api import BrowserContext, Page, expect

from support import VERIFY, link_path, unique


def made_up_birthday() -> tuple[str, tuple[str, str, str]]:
    """A date of birth for this run's made-up person, so earlier runs' accounts aren't theirs."""
    day, month, year = secrets.randbelow(28) + 1, secrets.randbelow(12) + 1, 1940 + secrets.randbelow(50)
    return f"{year}-{month:02d}-{day:02d}", (str(day), str(month), str(year))


def enter_dob(patient: Page, day: str, month: str, year: str) -> None:
    patient.get_by_label("Day").fill(day)
    patient.get_by_label("Month").fill(month)
    patient.get_by_label("Year").fill(year)
    patient.get_by_role("button", name="Continue").click()


def no_sideways_scroll(patient: Page) -> None:
    assert patient.evaluate("document.documentElement.scrollWidth <= window.innerWidth")


def test_j7c_returning_patient_signs_in_by_email_and_sees_both_trusts(page: Page, phone: BrowserContext, as_user, patient_link):
    dob, (day, month, year) = made_up_birthday()
    email = f"j7c.{secrets.token_hex(4)}@e2e.example"
    hq_a = as_user("Alex Author").published_hq("workflow-hq.json", f"E2E J7c A {unique()}")
    surname = f"J7c {unique()}"  # one person: the same name, date of birth and email in both trusts
    in_a = as_user("Cara Clinician").episode(hq_a, surname, email=email, dateOfBirth=dob)
    patient_link(in_a)  # their first visit, through Trust A's invite: their account
    bea = as_user("Bea Author", "Hospital B1")
    hq_b = bea.published_hq("workflow-hq.json", f"E2E J7c B {unique()}")
    bea.episode(hq_b, surname, email=email, dateOfBirth=dob)  # Trust B's, never opened

    # They ask for a sign-in link by email.
    patient = phone.new_page()
    patient.goto("/signin")
    patient.get_by_label("Mobile number or email address").fill(email)
    patient.get_by_role("button", name="Send me a link").click()
    expect(patient.get_by_role("heading", name="Check your email")).to_be_visible()
    no_sideways_scroll(patient)

    # The email's link: Continue (nothing about them), then their date of birth.
    link = link_path(page.context.request, email)
    patient.goto(link)
    expect(patient.get_by_role("heading", name="Sign in to Poised")).to_be_visible()
    patient.get_by_role("button", name="Continue").click()
    expect(patient.get_by_text("Poised needs your date of birth")).to_be_visible()
    enter_dob(patient, day, month, year)

    # Their home: both trusts' questionnaires, newest first, with each hospital's name.
    expect(patient).to_have_url(re.compile(r"/home$"))
    expect(patient.get_by_role("heading", name="Hello Sam")).to_be_visible()
    cards = patient.get_by_test_id("home-episode")
    expect(cards).to_have_count(2)
    expect(cards.nth(0)).to_contain_text("Hospital B1")
    expect(cards.nth(0)).to_contain_text(hq_b["name"])
    expect(cards.nth(1)).to_contain_text("Hospital A1")
    expect(cards.nth(1)).to_contain_text("To do")
    expect(cards.nth(1)).to_contain_text("For Sam J7c")  # whose questionnaire it is (S15 answers 6)
    # Nothing they didn't type: no date of birth or full email.
    expect(patient.get_by_text(email)).to_have_count(0)
    expect(patient.get_by_text(year)).to_have_count(0)
    expect(patient.get_by_test_id("contact-email")).to_contain_text("j•••@e2e.example")
    no_sideways_scroll(patient)

    # They open Trust A's questionnaire, and come back.
    cards.nth(1).get_by_role("link", name="Start").click()
    expect(patient.get_by_role("heading", name=hq_a["name"])).to_be_visible()
    patient.get_by_role("link", name="Your questionnaires").click()
    expect(patient.get_by_test_id("home-episode")).to_have_count(2)

    # Another patient's episode address shows nothing of theirs.
    patient.goto("/home/episodes/00000000-0000-4000-8000-000000000000")
    expect(patient.get_by_text("Episode not found")).to_be_visible()
    expect(patient.get_by_role("link", name="Your questionnaires")).to_be_visible()

    # Sign out; the used link no longer works, and offers a new one.
    patient.goto("/home")
    patient.get_by_role("button", name="Sign out").click()
    expect(patient.get_by_test_id("patient-signed-out")).to_be_visible()
    patient.reload()
    expect(patient.get_by_test_id("patient-signed-out")).to_be_visible()
    patient.goto(link)
    patient.get_by_role("button", name="Continue").click()
    expect(patient.get_by_role("heading", name="This link has expired")).to_be_visible()
    expect(patient.get_by_role("link", name="Get a new sign-in link")).to_be_visible()


def test_j7c_change_email_from_the_home(page: Page, phone: BrowserContext, as_user, patient_link):
    dob, (day, month, year) = made_up_birthday()
    email = f"j7c.{secrets.token_hex(4)}@e2e.example"
    new_email = f"new.{secrets.token_hex(4)}@e2e.example"
    hq = as_user("Alex Author").published_hq("workflow-hq.json", f"E2E J7c email {unique()}")
    episode = as_user("Cara Clinician").episode(hq, f"J7c {unique()}", email=email, dateOfBirth=dob)

    # Sam opens their invite and goes to their home.
    patient = phone.new_page()
    patient.goto(link_path(page.context.request, email, re.compile(r"/i/[A-Za-z0-9_-]{43}")))
    patient.get_by_role("button", name="Continue").click()
    enter_dob(patient, day, month, year)
    patient.get_by_role("link", name="Your questionnaires").click()

    # Change: a link goes to the new address; nothing changes yet.
    row = patient.get_by_test_id("contact-email")
    row.get_by_role("button", name="Change").click()
    row.get_by_label("New email address").fill(new_email)
    row.get_by_role("button", name="Send a link to confirm").click()
    expect(row.get_by_test_id("contact-sent")).to_contain_text(new_email)
    patient.reload()
    expect(patient.get_by_test_id("contact-email")).to_contain_text("j•••@e2e.example")

    # They open the link (in their email, on any device) and confirm.
    confirm = phone.new_page()
    confirm.goto(link_path(page.context.request, new_email, VERIFY))
    confirm.get_by_role("button", name="Confirm").click()
    expect(confirm.get_by_role("heading", name="Your email address has been updated")).to_be_visible()

    # The home shows it; the hospital's record has it too; the old address was told.
    patient.reload()
    expect(patient.get_by_test_id("contact-email")).to_contain_text("n•••@e2e.example")
    staff = as_user("Cara Clinician")
    assert staff.call("GET", staff.h(f"/episodes/{episode['id']}"))["episode"]["email"] == new_email
    told = [m for m in page.context.request.get("/api/qa/messages", params={"to": email}).json() if "changed" in m["body"]]
    assert told, "the old address wasn't told"


def test_j7c_twins_on_one_phone_are_asked_their_name(page: Page, phone: BrowserContext, as_user, patient_link):
    """Twins with one parent's mobile and the same date of birth have an account each (S15
    answers 6); a sign-in link to that mobile asks for the first name too."""
    dob, (day, month, year) = made_up_birthday()
    email = f"j7c.{secrets.token_hex(4)}@e2e.example"
    hq = as_user("Alex Author").published_hq("workflow-hq.json", f"E2E J7c twins {unique()}")
    cara = as_user("Cara Clinician")
    surname = f"Twin {unique()}"
    patient_link(cara.episode(hq, surname, firstName="Kim", email=email, dateOfBirth=dob))  # each opens their own invite
    patient_link(cara.episode(hq, surname, firstName="Lee", email=email, dateOfBirth=dob))

    patient = phone.new_page()
    patient.goto("/signin")
    patient.get_by_label("Mobile number or email address").fill(email)
    patient.get_by_role("button", name="Send me a link").click()
    expect(patient.get_by_role("heading", name="Check your email")).to_be_visible()
    patient.goto(link_path(page.context.request, email))
    patient.get_by_role("button", name="Continue").click()
    enter_dob(patient, day, month, year)
    expect(patient.get_by_test_id("dob-error")).to_contain_text("Please also enter your first name and surname")
    expect(patient.get_by_label("First name")).to_be_focused()
    patient.get_by_label("First name").fill("lee")
    patient.get_by_label("Surname").fill(surname.upper())
    patient.get_by_role("button", name="Continue").click()
    expect(patient.get_by_role("heading", name="Hello Lee")).to_be_visible()
    cards = patient.get_by_test_id("home-episode")
    expect(cards).to_have_count(1)
    expect(cards).to_contain_text("For Lee")


def test_j7c_leaving_for_the_home_straight_after_typing_keeps_the_answer(page: Page, phone: BrowserContext, as_user):
    """S15 verification 1: "← Your questionnaires" sends the waiting save first, like Sign out, so
    the home shows the questionnaire In progress and the answer is kept; if the save is refused
    (the session ended), the page says what wasn't saved instead of leaving."""
    dob, (day, month, year) = made_up_birthday()
    email = f"j7c.{secrets.token_hex(4)}@e2e.example"
    hq = as_user("Alex Author").published_hq("medicines-hq.json", f"E2E J7c leave {unique()}")
    cara = as_user("Cara Clinician")
    episode = cara.episode(hq, f"Leave {unique()}", email=email, dateOfBirth=dob)

    patient = phone.new_page()
    patient.goto(link_path(page.context.request, email, re.compile(r"/i/[A-Za-z0-9_-]{43}")))
    patient.get_by_role("button", name="Continue").click()
    enter_dob(patient, day, month, year)
    patient.get_by_test_id("set-Your medicines").get_by_role("button").click()  # nothing is required: Review
    patient.get_by_role("textbox", name="What medicines do you take?").fill("Warfarin 5mg")
    patient.get_by_role("link", name="Your questionnaires").click()  # straight away: the text box still has the focus
    expect(patient).to_have_url(re.compile(r"/home$"))
    expect(patient.get_by_test_id("home-episode")).to_contain_text("In progress")  # saved before the home loaded
    patient.get_by_test_id("home-episode").get_by_role("link", name="Continue").click()
    patient.get_by_test_id("set-Your medicines").get_by_role("button").click()
    expect(patient.get_by_role("textbox", name="What medicines do you take?")).to_have_value("Warfarin 5mg")

    # The session ends under them (staff resend the invite); the change can't be saved, so the
    # page stays and names what wasn't kept.
    cara.call("POST", cara.h(f"/episodes/{episode['id']}/resend-invite"))
    patient.get_by_role("textbox", name="What medicines do you take?").fill("Warfarin 3mg")
    patient.get_by_role("link", name="Your questionnaires").click()
    expect(patient.get_by_test_id("patient-lost-work")).to_contain_text("Your latest change in Your medicines wasn't saved.")
    expect(patient).to_have_url(re.compile(r"/home/episodes/"))
