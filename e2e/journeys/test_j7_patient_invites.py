"""J7 Patient invites (S14; A-4, PAT-03…05, PAT-09), with the patient on a phone (iPhone 13).

  - J7: a clinician creates a patient and episode; the captured-messages page shows the text and
    the email; the patient opens the text's link, taps Continue (the page shows nothing about
    them), gets their date of birth wrong once, then right, and fills in and sends their HQ.
  - J7b: five wrong dates of birth lock the link; the clinician sees it locked and resends the
    invite; the old link no longer works and the new one does.
  - On the patient's behalf: a clinician fills in the HQ for a patient on the phone; the answers
    are marked "entered by" her on the episode and the POA Summary.
"""

import re
import secrets

from playwright.sync_api import BrowserContext, Page, expect

from support import choose, invite_path, pick, sign_in, unique

DOB = re.compile(r"^Date of birth\W*$")  # the new patient's (Mantine adds " *"), not the search's


def made_up_mobile() -> tuple[str, str]:
    """A made-up mobile in Ofcom's drama range, typed and in E.164, different on most runs."""
    n = f"{secrets.randbelow(1000):03d}"
    return f"07700 900{n}", f"+447700900{n}"


def enter_dob(patient: Page, day: str, month: str, year: str) -> None:
    patient.get_by_label("Day").fill(day)
    patient.get_by_label("Month").fill(month)
    patient.get_by_label("Year").fill(year)
    patient.get_by_role("button", name="Continue").click()


def no_sideways_scroll(patient: Page) -> None:
    assert patient.evaluate("document.documentElement.scrollWidth <= window.innerWidth")


def test_j7_invite_by_text_and_email_then_date_of_birth_and_send(page: Page, phone: BrowserContext, as_user):
    hq = as_user("Alex Author").published_hq("workflow-hq.json", f"E2E J7 {unique()}")
    last_name = f"J7 {unique()}"
    typed, mobile = made_up_mobile()
    email = f"j7.{secrets.token_hex(4)}@e2e.example"

    # The clinician creates the patient and episode on the New episode page.
    base = sign_in(page.context, "Cara Clinician")
    page.goto(f"{base}/episodes/new")
    page.get_by_label("First name").fill("Sam")
    page.get_by_label("Surname").fill(last_name)
    page.get_by_label(DOB).fill("1980-05-17")
    pick(page, "Sex", "Female")
    page.get_by_label("Mobile").fill(typed)
    page.get_by_label("Email").fill(email)
    pick(page, "Health questionnaire", f"{hq['name']} v1")
    page.get_by_role("button", name="Create episode").click()
    expect(page.get_by_role("heading", name=f"Sam {last_name}")).to_be_visible()
    expect(page.get_by_text("Invite sent by text message and email.", exact=True)).to_be_visible()
    expect(page.get_by_test_id("invite")).to_contain_text("Last sent")

    # The captured-messages page shows both: stored, never sent.
    page.goto("/qa/messages")
    page.get_by_label("To").fill(mobile)
    page.get_by_role("button", name="Show").click()
    sms = page.get_by_test_id("captured-message")
    expect(sms).to_have_count(1)
    expect(sms).to_contain_text("Text message")
    expect(sms).to_contain_text("Hospital A1: please complete your pre-op health questionnaire:")
    page.get_by_label("To").fill(email)
    page.get_by_role("button", name="Show").click()
    expect(page.get_by_test_id("captured-message")).to_contain_text("Hospital A1: your health questionnaire")
    link = invite_path(page.context.request, mobile)

    # The patient taps the text's link. The page says nothing about them until their date of birth.
    patient = phone.new_page()
    patient.goto(link)
    expect(patient.get_by_role("heading", name="Your health questionnaire")).to_be_visible()
    expect(patient.get_by_text(last_name)).to_have_count(0)
    # The link's code never leaves in a Referer header (S14 answers 4).
    assert patient.evaluate("document.querySelector('meta[name=referrer]')?.content") == "no-referrer"
    patient.get_by_role("button", name="Continue").click()
    expect(patient.get_by_role("heading", name="Confirm it's you")).to_be_visible()
    expect(patient.get_by_text("Hospital A1 needs your date of birth")).to_be_visible()
    expect(patient.get_by_text("Sam")).to_have_count(0)
    no_sideways_scroll(patient)

    # A wrong date, then the right one.
    enter_dob(patient, "18", "5", "1980")
    expect(patient.get_by_test_id("dob-error")).to_contain_text("You have 4 tries left.")
    enter_dob(patient, "17", "5", "1980")
    expect(patient.get_by_role("heading", name=hq["name"])).to_be_visible()
    expect(patient).to_have_url(re.compile(r"/home/episodes/"))
    expect(patient.get_by_text("Hello Sam.")).to_be_visible()
    # Nothing they didn't type: no date of birth, mobile or email.
    for hidden in ["1980", mobile, email]:
        expect(patient.get_by_text(hidden)).to_have_count(0)
    no_sideways_scroll(patient)

    # Reloading keeps them signed in (their session lasts 30 minutes from their last request).
    patient.reload()
    expect(patient.get_by_text("0 of 2 sections complete")).to_be_visible()

    # They fill in both Question Sets and send their answers.
    about = patient.get_by_test_id("set-About you")
    about.get_by_role("button", name="Start").click()
    choose(patient, "Do you smoke?", "No")
    patient.get_by_role("button", name="Next", exact=True).click()
    choose(patient, "How often do you drink alcohol?", "Never")
    patient.get_by_role("button", name="Finish this section").click()
    health = patient.get_by_test_id("set-Your health")
    health.get_by_role("button", name="Start").click()
    choose(patient, "Do you have diabetes?", "No")
    patient.get_by_role("button", name="Finish this section").click()
    patient.get_by_role("button", name="Send my answers").click()
    patient.get_by_role("dialog", name="Send your answers?").get_by_role("button", name="Send my answers").click()
    expect(patient.get_by_role("heading", name="Thank you, Sam")).to_be_visible()

    # Opening the link again later still works (PAT-05), and shows the thank-you page.
    patient.goto(link)
    patient.get_by_role("button", name="Continue").click()
    enter_dob(patient, "17", "05", "1980")
    expect(patient.get_by_role("heading", name="Thank you, Sam")).to_be_visible()

    # Signing out ends the session: reloading shows nothing until the link and date of birth again.
    patient.get_by_role("button", name="Sign out").click()
    expect(patient.get_by_role("heading", name="You've been signed out")).to_be_visible()
    patient.reload()
    expect(patient.get_by_role("heading", name="You've been signed out")).to_be_visible()
    expect(patient.get_by_text("Thank you")).to_have_count(0)


def test_j7b_five_wrong_dates_of_birth_lock_the_link_until_the_invite_is_resent(page: Page, phone: BrowserContext, as_user):
    hq = as_user("Alex Author").published_hq("workflow-hq.json", f"E2E J7b {unique()}")
    _, mobile = made_up_mobile()
    episode = as_user("Cara Clinician").episode(hq, f"J7b {unique()}", mobile=mobile)
    old = invite_path(page.context.request, mobile)

    patient = phone.new_page()
    patient.goto(old)
    patient.get_by_role("button", name="Continue").click()
    for left in ["4 tries", "3 tries", "2 tries", "1 try"]:
        enter_dob(patient, "1", "1", "1990")
        expect(patient.get_by_test_id("dob-error")).to_contain_text(f"You have {left} left.")
    enter_dob(patient, "1", "1", "1990")
    expect(patient.get_by_role("heading", name="This link is locked")).to_be_visible()
    expect(patient.get_by_text("contact the hospital")).to_be_visible()
    # Even the right date of birth can't open it now.
    patient.goto(old)
    patient.get_by_role("button", name="Continue").click()
    expect(patient.get_by_role("heading", name="This link is locked")).to_be_visible()

    # The clinician sees the lock and resends the invite.
    base = sign_in(page.context, "Cara Clinician")
    page.goto(f"{base}/episodes/{episode['id']}")
    invite = page.get_by_test_id("invite")
    expect(invite.get_by_text("The link is locked")).to_be_visible()
    invite.get_by_role("button", name="Resend invite").click()
    expect(invite.get_by_text("Invite sent again by text message and email.")).to_be_visible()
    expect(invite.get_by_text("The link is locked")).to_be_hidden()
    new = invite_path(page.context.request, mobile)
    assert new != old

    # The old link no longer works; the new one does.
    patient.goto(old)
    patient.get_by_role("button", name="Continue").click()
    expect(patient.get_by_role("heading", name="This link doesn't work")).to_be_visible()
    patient.goto(new)
    patient.get_by_role("button", name="Continue").click()
    enter_dob(patient, "17", "5", "1980")
    expect(patient.get_by_role("heading", name=hq["name"])).to_be_visible()


def test_a_clinician_fills_in_the_hq_on_the_patients_behalf(page: Page, as_user):
    hq = as_user("Alex Author").published_hq("workflow-hq.json", f"E2E Behalf {unique()}")
    episode = as_user("Cara Clinician").episode(hq, f"Behalf {unique()}")
    base = sign_in(page.context, "Cara Clinician")
    page.goto(f"{base}/episodes/{episode['id']}")
    page.get_by_role("link", name="Fill in for the patient").click()
    expect(page).to_have_url(f"{base}/episodes/{episode['id']}/patient-form")
    expect(page.get_by_test_id("on-behalf-banner")).to_contain_text(f"{episode['name']}'s health questionnaire")

    about = page.get_by_test_id("set-About you")
    about.get_by_role("button", name="Start").click()
    choose(page, "Do you smoke?", "Yes")
    page.get_by_role("button", name="Next", exact=True).click()
    choose(page, "How often do you drink alcohol?", "Sometimes")
    page.get_by_role("button", name="Finish this section").click()
    health = page.get_by_test_id("set-Your health")
    health.get_by_role("button", name="Start").click()
    choose(page, "Do you have diabetes?", "No")
    page.get_by_role("button", name="Finish this section").click()
    page.get_by_role("button", name="Send the answers").click()
    page.get_by_role("dialog", name="Send the answers?").get_by_role("button", name="Send the answers").click()
    expect(page.get_by_role("heading", name=f"Sent for {episode['name']}")).to_be_visible()

    page.get_by_role("link", name="Back to the episode").click()
    expect(page.get_by_role("combobox", name="Episode status")).to_have_value("Ready for review")
    expect(page.get_by_text("HQ completed on the patient's behalf")).to_be_visible()
    expect(page.get_by_text("Sent by Cara Clinician on the patient's behalf")).to_be_visible()
    expect(page.get_by_test_id("hq-set-About you")).to_contain_text("Patient's answers entered by Cara Clinician")

    page.get_by_role("link", name="POA Summary").click()
    expect(page.get_by_text(re.compile(r"Patient \(entered by Cara Clinician\) on")).first).to_be_visible()


def test_a_change_refused_because_the_session_ended_is_named_not_lost(page: Page, phone: BrowserContext, as_user):
    """S14 verification 1: a Resend ends the patient's open session; their next change can't be
    saved, and the page must say so, naming the section, not just "signed out"."""
    hq = as_user("Alex Author").published_hq("workflow-hq.json", f"E2E Ended {unique()}")
    _, mobile = made_up_mobile()
    cara = as_user("Cara Clinician")
    episode = cara.episode(hq, f"Ended {unique()}", mobile=mobile)

    patient = phone.new_page()
    patient.goto(invite_path(page.context.request, mobile))
    patient.get_by_role("button", name="Continue").click()
    enter_dob(patient, "17", "5", "1980")
    patient.get_by_test_id("set-About you").get_by_role("button", name="Start").click()
    with patient.expect_response(lambda r: r.request.method == "PUT" and "/answers/" in r.url) as saved:
        choose(patient, "Do you smoke?", "No")
    assert saved.value.ok

    cara.call("POST", cara.h(f"/episodes/{episode['id']}/resend-invite"))  # the open session ends

    choose(patient, "Do you smoke?", "Yes")
    expect(patient.get_by_role("heading", name="You've been signed out")).to_be_visible()
    expect(patient.get_by_test_id("patient-lost-work")).to_contain_text(
        "Your latest change in About you wasn't saved. When you open the link again, please check your answers in that section.")
    expect(patient.get_by_text("open the link in your text message or email again")).to_be_visible()

    # Back in with the new link: the saved answer is the old one, as the message warned.
    patient.goto(invite_path(page.context.request, mobile))
    patient.get_by_role("button", name="Continue").click()
    enter_dob(patient, "17", "5", "1980")
    patient.get_by_test_id("set-About you").get_by_role("button", name="Continue").click()
    expect(patient.get_by_role("radiogroup", name="Do you smoke?").get_by_role("radio", name="No")).to_be_checked()


def open_hq(patient: Page, link: str) -> None:
    patient.goto(link)
    patient.get_by_role("button", name="Continue").click()
    enter_dob(patient, "17", "5", "1980")


def test_sign_out_straight_after_typing_keeps_the_answer(page: Page, phone: BrowserContext, as_user):
    """S14 verification 2: Sign out sends the waiting save first. A text typed and a choice tapped
    just before it are both kept."""
    hq = as_user("Alex Author").published_hq("medicines-hq.json", f"E2E Medicines {unique()}")
    _, mobile = made_up_mobile()
    as_user("Cara Clinician").episode(hq, f"SignOut {unique()}", mobile=mobile)

    patient = phone.new_page()
    open_hq(patient, invite_path(page.context.request, mobile))
    patient.get_by_test_id("set-Your medicines").get_by_role("button").click()  # nothing is required: Review
    patient.get_by_role("textbox", name="What medicines do you take?").fill("Warfarin 5mg")
    patient.get_by_role("button", name="Sign out").click()  # straight away: the text box still has the focus
    expect(patient.get_by_role("heading", name="You've been signed out")).to_be_visible()
    expect(patient.get_by_test_id("patient-lost-work")).to_have_count(0)

    open_hq(patient, invite_path(page.context.request, mobile))
    patient.get_by_test_id("set-Your medicines").get_by_role("button").click()  # nothing is required: Review
    expect(patient.get_by_role("textbox", name="What medicines do you take?")).to_have_value("Warfarin 5mg")

    # A choice tapped just before Sign out (well inside the 600 ms autosave wait) is kept too.
    choose(patient, "Do you take blood thinners?", "Yes")
    patient.get_by_role("button", name="Sign out").click()
    expect(patient.get_by_role("heading", name="You've been signed out")).to_be_visible()
    open_hq(patient, invite_path(page.context.request, mobile))
    patient.get_by_test_id("set-Your medicines").get_by_role("button").click()  # nothing is required: Review
    expect(patient.get_by_role("radiogroup", name="Do you take blood thinners?").get_by_role("radio", name="Yes")).to_be_checked()


def test_leaving_the_on_behalf_form_straight_after_typing_keeps_the_answer(page: Page, as_user):
    """S14 verification 2: the on-behalf form's "Episode" link sends the waiting save first."""
    hq = as_user("Alex Author").published_hq("medicines-hq.json", f"E2E Medicines {unique()}")
    episode = as_user("Cara Clinician").episode(hq, f"Leave {unique()}")
    base = sign_in(page.context, "Cara Clinician")
    page.goto(f"{base}/episodes/{episode['id']}/patient-form")
    page.get_by_test_id("set-Your medicines").get_by_role("button").click()  # nothing is required: Review
    page.get_by_role("textbox", name="What medicines do you take?").fill("Aspirin 75mg")
    page.get_by_role("link", name="Episode", exact=True).click()  # straight away
    expect(page).to_have_url(f"{base}/episodes/{episode['id']}")
    expect(page.get_by_test_id("hq-set-Your medicines")).to_contain_text("Patient's answers entered by Cara Clinician")
    page.goto(f"{base}/episodes/{episode['id']}/patient-form")
    page.get_by_test_id("set-Your medicines").get_by_role("button").click()  # nothing is required: Review
    expect(page.get_by_role("textbox", name="What medicines do you take?")).to_have_value("Aspirin 75mg")
