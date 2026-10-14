"""J8 Reminders and "not responding" (S16), run from the scheduled-jobs page as of later times.

A clinician's new episode invites the patient. On the jobs page (capture mode only) the tester
runs the jobs as of two days on: the patient gets a reminder with a new link, and running the
same time again sends nothing more. The patient opens the reminder's link on their phone and
starts answering, so day 5's reminder doesn't go; on day 7 the episode is marked "Not
responding" until the patient sends their answers.

The jobs page runs every job due at that time, so other journeys' episodes may get reminders
too; this journey checks only its own patient's messages.
"""

import re
import secrets

from playwright.sync_api import BrowserContext, Page, expect

from support import INVITE, captured, qa_page, sign_in, unique


def run_jobs(page: Page, preset: str) -> None:
    """On the scheduled-jobs page, as the platform admin (`qa`: QA's tools need one, S12)."""
    page.goto("/qa/jobs")
    page.get_by_role("button", name=preset).click()
    page.get_by_role("button", name="Run jobs").click()
    expect(page.get_by_test_id("jobs-run-result")).to_contain_text(re.compile(r"^Ran \d+ jobs? as of "))


def test_j8_reminders_then_not_responding(page: Page, phone: BrowserContext, as_user, patient_link):
    hq = as_user("Alex Author").published_hq("workflow-hq.json", f"E2E J8 {unique()}")
    n = f"{secrets.randbelow(1000):03d}"
    mobile = f"+447700900{n}"
    episode = as_user("Cara Clinician").episode(hq, f"J8 {unique()}", mobile=f"07700 900{n}")
    email = episode["email"]

    def messages() -> list[dict]:
        return captured(email)

    assert len(messages()) == 1  # the invite
    invite = INVITE.search(messages()[0]["body"]).group(0)

    # Day 2: a reminder, by text and email, with a new link.
    base = sign_in(page.context, "Cara Clinician")
    qa = qa_page(page)
    run_jobs(qa, "Now + 2 days")
    got = messages()
    assert len(got) == 2, got
    assert got[0]["subject"] == "Hospital A1: a reminder about your health questionnaire"
    assert "This is a reminder from Hospital A1" in got[0]["body"]
    reminder = INVITE.search(got[0]["body"]).group(0)
    assert reminder != invite
    texts = captured(mobile)
    assert len(texts) == 2 and texts[0]["body"].startswith("Hello Sam, a reminder from Hospital A1:"), texts

    # The captured-messages page shows it too.
    qa.goto("/qa/messages")
    qa.get_by_label("To").fill(email)
    qa.get_by_role("button", name="Show").click()
    expect(qa.get_by_test_id("captured-message")).to_have_count(2)
    expect(qa.get_by_test_id("captured-message").first).to_contain_text("a reminder about your health questionnaire")

    # The same time again: nothing more.
    run_jobs(qa, "Now + 2 days")
    assert len(messages()) == 2

    # The episode's General notes say what went, written automatically.
    page.goto(f"{base}/episodes/{episode['id']}")
    note = page.get_by_text("Day 2 reminder sent by text message and email.", exact=True)
    expect(note).to_be_visible()
    expect(page.get_by_text(re.compile(r"\| Automatic on "))).to_have_count(1)
    expect(page.get_by_test_id("not-responding")).to_have_count(0)

    # The patient opens the reminder's link on their phone: it works like the invite's.
    patient = phone.new_page()
    patient.goto(reminder)
    patient.get_by_role("button", name="Continue").click()
    patient.get_by_label("Day").fill("17")
    patient.get_by_label("Month").fill("5")
    patient.get_by_label("Year").fill("1980")
    patient.get_by_role("button", name="Continue").click()
    expect(patient.get_by_role("heading", name=hq["name"])).to_be_visible()
    assert patient.evaluate("document.documentElement.scrollWidth <= window.innerWidth")

    # They answer one question (started), so day 5's reminder doesn't go.
    p = patient_link(episode)
    p.answer("About you", {"q_smokes": "o_no"})
    run_jobs(qa, "Now + 5 days")
    assert len(messages()) == 2

    # Day 7 without their answers: "Not responding" on the episode.
    run_jobs(qa, "Now + 7 days")
    page.goto(f"{base}/episodes/{episode['id']}")
    expect(page.get_by_test_id("not-responding")).to_have_text("Not responding")
    expect(page.get_by_text("Not responding: the HQ hasn't been sent 7 days after the invite.", exact=True)).to_be_visible()
    assert len(messages()) == 2  # marking it sends the patient nothing

    # Once they send their answers, the flag goes.
    p.submit()
    page.reload()
    expect(page.get_by_role("heading", name=episode["name"])).to_be_visible()
    expect(page.get_by_test_id("not-responding")).to_have_count(0)
