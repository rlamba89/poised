"""J9 The full lifecycle, with archive (S22; EPI-01, EPI-04, EPI-05, S22 answers 1 and 2).

  - Cara's episode, with a procedure; the patient sends the HQ; Cara validates it and completes
    the review, then moves it to Ready for admission, Admitted and Discharged, with the dates.
  - Cara (a clinician) can't archive. Sue Super archives it with a reason: the episode is
    read-only (no status choices, no Add comment, no Add task), and it leaves the episode list
    until "Show archived" is ticked, which survives a reload. Cara can't un-archive.
  - The patient, on their phone, still sees it as Sent on their home, with a read-only View
    (S27; it was "Closed by the hospital" before).
  - Sue un-archives it, with a reason ("Say why…" first): back to Discharged exactly.
  - The jobs run as of 31 days on: the daily auto-archive archives it, and the status history
    names the rule, "Automatic (Auto-archive: 30 days after discharge)".

J9b (cancel) is test_j9b_cancel.py.
"""

import datetime as dt
import re

from playwright.sync_api import BrowserContext, Page, expect

from support import PATIENT_ANSWERS, _ok, change_status, invite_path, qa_request, sign_in, unique


def test_j9_full_lifecycle_archive_and_unarchive(page: Page, phone: BrowserContext, as_user, patient_link):
    hq = as_user("Alex Author").published_hq("workflow-hq.json", f"E2E J9 {unique()}")
    cara = as_user("Cara Clinician")
    last = f"J9 {unique()}"
    episode = cara.episode(hq, last, procedure="Knee arthroscopy")
    patient = patient_link(episode)
    for set_name, answers in PATIENT_ANSWERS.items():
        patient.answer(set_name, answers)
    patient.submit()
    for set_name, answers in {**PATIENT_ANSWERS, "Nurse checks": {"q_bp": "120/80"}}.items():
        cara.validate(episode, set_name, answers)

    # Cara completes the review and walks the episode to Discharged.
    base = sign_in(page.context, "Cara Clinician")
    url = f"{base}/episodes/{episode['id']}"
    page.goto(url)
    expect(page.get_by_text("Knee arthroscopy").first).to_be_visible()
    page.get_by_role("button", name="Complete HQ review").click()
    control = page.get_by_test_id("status-control")
    expect(control.get_by_role("combobox", name="Episode status")).to_have_value("Ready for POA")
    change_status(page, "Ready for admission")
    today = dt.date.today()
    change_status(page, "Admitted", date=(today - dt.timedelta(days=2)).isoformat())
    change_status(page, "Discharged", date=(today - dt.timedelta(days=1)).isoformat())
    expect(control.get_by_role("button", name="Archive episode")).to_have_count(0)  # a clinician can't

    # Sue Super (a super clinician at Hospital A1) archives it, with a reason; Enter sends it.
    sign_in(page.context, "Sue Super")
    page.goto(url)
    control.get_by_role("button", name="Archive episode").click()
    form = page.get_by_test_id("status-form")
    expect(form.get_by_role("heading", name="Archive this episode")).to_be_visible()
    form.get_by_label("Reason (optional)").fill("Made-up: treatment finished")
    form.get_by_label("Reason (optional)").press("Enter")
    expect(page.get_by_test_id("archived")).to_contain_text("Archived from Discharged")
    expect(page.get_by_test_id("archived")).to_contain_text("by Sue Super")
    expect(page.get_by_test_id("archived")).to_contain_text("Reason: Made-up: treatment finished")

    # Read-only: every control is off.
    expect(control.get_by_role("combobox", name="Episode status")).to_have_value("Archived")
    expect(control.get_by_role("combobox", name="Episode status")).to_be_disabled()
    for name in ["Archive episode", "Cancel episode", re.compile("^Undo")]:
        expect(control.get_by_role("button", name=name)).to_have_count(0)
    expect(page.get_by_role("button", name="Add comment")).to_have_count(0)
    expect(page.get_by_role("button", name="Add task")).to_have_count(0)
    expect(page.get_by_text("This episode is archived, so its tasks can't be changed.")).to_be_visible()
    expect(control.get_by_role("button", name="Un-archive: back to Discharged")).to_be_visible()
    rows = page.get_by_test_id("status-history-row")
    expect(rows.last).to_contain_text("Archived: Discharged → Archived")
    expect(rows.last).to_contain_text("Sue Super")

    # The patient's copy (S27): their link still opens what they sent, read-only, and their home
    # says Sent, with View.
    mine = phone.new_page()
    mine.goto(invite_path(episode["email"]))
    mine.get_by_role("button", name="Continue").click()
    year, month, day = episode["dob"].split("-")
    mine.get_by_label("Day").fill(str(int(day)))
    mine.get_by_label("Month").fill(str(int(month)))
    mine.get_by_label("Year").fill(year)
    mine.get_by_role("button", name="Continue").click()
    expect(mine.get_by_text("Your answers have been sent to the hospital.")).to_be_visible()
    expect(mine.get_by_role("button", name="Send my answers")).to_have_count(0)
    mine.get_by_role("link", name="Your questionnaires").click()
    card = mine.get_by_test_id("home-episode").filter(has_text=hq["name"])
    expect(card).to_contain_text("Sent")
    expect(card).not_to_contain_text("Closed by the hospital")
    card.get_by_role("link", name="View").click()
    expect(mine.get_by_text("Your answers have been sent to the hospital.")).to_be_visible()

    # It leaves the episode list, until Show archived is ticked (kept in the URL).
    page.goto(f"{base}/episodes")
    expect(page.get_by_role("table").or_(page.get_by_text("No episodes yet."))).to_be_visible()
    expect(page.get_by_text(episode["name"])).to_have_count(0)
    page.get_by_label("Show archived").click()  # the URL changes in a transition, so check() would see it unchanged
    expect(page).to_have_url(re.compile(r"archived=1"))
    expect(page.get_by_label("Show archived")).to_be_checked()
    expect(page.get_by_text(episode["name"])).to_be_visible()
    page.reload()
    expect(page.get_by_label("Show archived")).to_be_checked()
    expect(page.get_by_text(episode["name"])).to_be_visible()

    # Cara can see it, but not un-archive it.
    sign_in(page.context, "Cara Clinician")
    page.goto(url)
    expect(page.get_by_test_id("archived")).to_be_visible()
    expect(control.get_by_role("button", name=re.compile("^Un-archive"))).to_have_count(0)

    # Sue un-archives it, with a reason: back to Discharged exactly.
    sign_in(page.context, "Sue Super")
    page.goto(url)
    control.get_by_role("button", name="Un-archive: back to Discharged").click()
    form.get_by_role("button", name="Un-archive").click()
    expect(form.get_by_text("Say why the episode is being un-archived.")).to_be_visible()
    form.get_by_label("Reason").fill("Made-up: a follow-up visit is booked")
    form.get_by_label("Reason").press("Enter")
    expect(control.get_by_role("combobox", name="Episode status")).to_have_value("Discharged")
    expect(page.get_by_test_id("archived")).to_have_count(0)
    expect(rows.last).to_contain_text("Un-archived: Archived → Discharged")
    expect(rows.last).to_contain_text("Made-up: a follow-up visit is booked")
    expect(page.get_by_role("button", name="Add comment")).to_be_visible()

    # 31 days on, the daily auto-archive archives it, naming its rule.
    as_of = (dt.datetime.now(dt.timezone.utc) + dt.timedelta(days=31)).isoformat()
    _ok(qa_request().post("/api/qa/jobs/run", data={"asOf": as_of}), "POST /api/qa/jobs/run")
    page.reload()
    expect(control.get_by_role("combobox", name="Episode status")).to_have_value("Archived")
    expect(rows.last).to_contain_text("Archived: Discharged → Archived")
    expect(rows.last).to_contain_text("Automatic (Auto-archive: 30 days after discharge)")
    expect(page.get_by_test_id("archived")).to_contain_text("Automatic (Auto-archive: 30 days after discharge)")
