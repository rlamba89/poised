"""The audit log (S11; STF-06): every request that reads or changes patient data writes one row,
list screens included. A trust's whole-trust admins read it on the Audit screen, filter it by
person, days and patient, and export it as a CSV. Clinicians don't see it; another trust's admin
sees none of it.

Each run makes a patient with a unique surname, so filtering by it finds only this run's rows.

Each of the pre-fill card's buttons has its own words in the log (S29): Start fresh, pressed by
staff for the patient, reads "Started a Question Set fresh for the patient"."""

import csv
import io
import re
from datetime import date, timedelta

from playwright.sync_api import Browser, Page, expect

from support import PATIENT_ANSWERS, choose, pick, sign_in, unique
from test_j16b_prefill_cross_trust import trust_b_then_trust_a


def open_audit(page: Page, base: str) -> None:
    page.goto(f"{base}/episodes")
    page.get_by_role("banner").get_by_role("link", name="Audit").click()
    expect(page.get_by_role("heading", name="Audit log")).to_be_visible()


def show(page: Page, *in_url: str) -> None:
    """Applies the filters, and waits for the page's URL to hold them (the list loads from it)."""
    page.get_by_role("button", name="Show", exact=True).click()
    for part in in_url:
        expect(page).to_have_url(re.compile(re.escape(part)))


def log_rows(page: Page):
    return page.get_by_role("table", name="Audit log").locator("tbody tr")


def test_an_admin_finds_a_clinicians_actions_filters_them_and_exports_the_csv(page: Page, browser: Browser, base_url, as_user, patient_link):
    hq = as_user("Alex Author").published_hq("workflow-hq.json", f"E2E Audit {unique()}")
    surname = f"Audit {unique()}"
    episode = as_user("Cara Clinician").episode(hq, surname)  # the set-up's own rows are Cara's too
    patient = patient_link(episode)
    for set_name, answers in PATIENT_ANSWERS.items():
        patient.answer(set_name, answers)
    patient.submit()

    # Cara, in her own browser, opens the patient's episode and changes an answer.
    theirs = browser.new_context(viewport={"width": 1440, "height": 900}, base_url=base_url)
    cara = theirs.new_page()
    try:
        cara_base = sign_in(theirs, "Cara Clinician")
        cara.goto(f"{cara_base}/episodes")
        cara.get_by_role("row", name=f"Sam {surname}").click()
        expect(cara.get_by_role("combobox", name="Episode status")).to_have_value("Ready for review")
        cara.get_by_role("row", name="About you").get_by_role("link", name="Validate").click()
        with cara.expect_response(lambda r: r.request.method == "PUT" and "/answers/" in r.url) as saved:
            choose(cara, "Do you smoke?", "No")
        assert saved.value.ok
    finally:
        theirs.close()

    # Ada, the trust's admin, finds those actions: by Cara, for this patient.
    open_audit(page, sign_in(page.context, "Ada Admin"))
    pick(page, "Who", "Cara Clinician (cara.clinician@hospital-a.example)")
    page.get_by_label("Patient").fill(surname)
    show(page, "?user=", "&patient=")
    rows = log_rows(page)
    for what in ["Saved the clinician's answers", "Opened the answers", "Opened an episode", "Listed episodes", "Created an episode"]:
        expect(rows.filter(has_text=what).first).to_be_visible()
    expect(rows.first).to_contain_text("Cara Clinician")
    expect(rows.first).to_contain_text("Hospital A1")
    expect(rows.filter(has_text="Opened an episode").first).to_contain_text(f"Sam {surname}")
    expect(page.get_by_text("Patient (their link)")).to_have_count(0)  # Cara's rows only
    shown = rows.count()

    # The filters stay after a reload.
    page.reload()
    expect(log_rows(page)).to_have_count(shown)
    expect(page.get_by_label("Patient")).to_have_value(surname)

    # The patient's own link is "Patients (their own link)".
    pick(page, "Who", "Patients (their own link)")
    show(page, "user=patient")
    expect(log_rows(page).filter(has_text="Patient sent their answers")).to_have_count(1)
    expect(log_rows(page).first).to_contain_text("Patient (their link)")

    # Days: today finds them; from tomorrow finds nothing.
    today = date.today()
    page.get_by_label("From").fill(today.isoformat())
    page.get_by_label("To").fill(today.isoformat())
    show(page, f"from={today.isoformat()}&to={today.isoformat()}")
    expect(log_rows(page).filter(has_text="Patient sent their answers")).to_have_count(1)
    tomorrow = (today + timedelta(days=1)).isoformat()
    page.get_by_label("From").fill(tomorrow)
    page.get_by_label("To").fill(tomorrow)
    show(page, f"from={tomorrow}&to={tomorrow}")
    expect(page.get_by_text("Nothing matches these filters.")).to_be_visible()
    # Back goes to the previous filters, form included.
    page.go_back()
    expect(page.get_by_label("From")).to_have_value(today.isoformat())
    expect(log_rows(page).filter(has_text="Patient sent their answers")).to_have_count(1)

    # Back to Cara's rows: the CSV export has the same rows as the list.
    page.get_by_role("button", name="Clear").click()
    expect(page).to_have_url(re.compile(r"/audit$"))
    pick(page, "Who", "Cara Clinician (cara.clinician@hospital-a.example)")
    page.get_by_label("Patient").fill(surname)
    show(page, "patient=")
    # The list has reloaded with Cara's rows (the patient's rows shown before can be as many: S12, on QA).
    expect(page.get_by_text("Patient (their link)")).to_have_count(0)
    expect(log_rows(page)).to_have_count(shown)
    listed = [r.inner_text() for r in log_rows(page).all()]
    with page.expect_download() as download:
        page.get_by_role("link", name="Export CSV").click()
    assert download.value.suggested_filename.endswith(".csv")
    records = list(csv.DictReader(io.StringIO(open(download.value.path(), encoding="utf-8").read())))
    assert len(records) == shown, f"{len(records)} CSV rows, {shown} listed"
    for rec, text in zip(records, listed):
        assert rec["Who"] == "Cara Clinician" and rec["Action"] in text and rec["Hospital"] in text, (rec, text)
        assert rec["Result"] in {"200", "201", "204"}
    assert any(f"Sam {surname}" in rec["Patients"] for rec in records)


def test_a_clinician_cant_see_the_audit_screen(page: Page):
    base = sign_in(page.context, "Cara Clinician")
    page.goto(f"{base}/episodes")
    expect(page.get_by_role("heading", name="Episodes")).to_be_visible()
    expect(page.get_by_role("banner").get_by_role("link", name="Audit")).to_have_count(0)
    page.goto(f"{base}/audit")
    expect(page.get_by_text("Only the trust's admins can see the audit log")).to_be_visible()
    expect(page.get_by_role("table", name="Audit log")).to_have_count(0)


def test_another_trusts_admin_sees_nothing_of_trust_a(page: Page, as_user):
    """Trust B's admin finds nothing of this test's own Trust A episode: not by its patient's
    surname, and no row of Trust B's log (its CSV) names the episode or the record. It checks only
    this test's data: since S27, Trust A staff who fill in for a patient with a Trust B history are
    rightly in Trust B's log (its prefill.read rows), so "Cara is never in Trust B's Who list"
    isn't true on a database where that happened (S27 verification 2)."""
    surname = f"Audit {unique()}"
    hq = as_user("Alex Author").published_hq("workflow-hq.json", f"E2E Audit {unique()}")
    cara = as_user("Cara Clinician")
    episode = cara.episode(hq, surname)
    cara.call("GET", cara.h(f"/episodes/{episode['id']}/patient-form"))  # more of Trust A's own activity on it
    open_audit(page, sign_in(page.context, "Max Multi", "Hospital B1"))  # Trust B's admin
    page.get_by_label("Patient").fill(surname)
    show(page, "patient=")
    expect(page.get_by_text("Nothing matches these filters.")).to_be_visible()

    max_ = as_user("Max Multi", "Hospital B1")
    res = max_.request.get(f"/api{max_.o('/audit')}?format=csv")
    assert res.ok, f"GET Trust B's audit CSV: {res.status}"
    trust_b_log = res.text()
    assert trust_b_log.startswith("Time,Who,"), trust_b_log[:80]
    for id_ in (episode["id"], episode["patientId"]):
        assert id_ not in trust_b_log, f"Trust B's audit log names Trust A's {id_}"
    # And Trust A's own log does name them (the check above isn't vacuous).
    ada = as_user("Ada Admin")
    trust_a_log = ada.request.get(f"/api{ada.o('/audit')}?format=csv").text()
    assert episode["id"] in trust_a_log


def test_bad_filters_say_so_and_never_leave_old_rows_on_screen(page: Page):
    base = sign_in(page.context, "Ada Admin")
    open_audit(page, base)
    expect(log_rows(page).first).to_be_visible()

    # A search the API refuses (over 100 characters) shows the message and no rows, keeps the
    # person chosen, and can't be exported (S11 test report 2).
    pick(page, "Who", "Ada Admin (ada.admin@trust-a.example)")
    show(page, "user=")
    page.get_by_label("Patient").fill("x" * 101)
    show(page, "patient=x")
    expect(page.get_by_role("alert")).to_contain_text("The patient search is too long.")
    expect(page.get_by_role("table", name="Audit log")).to_have_count(0)
    expect(page.get_by_role("combobox", name="Who")).to_have_value("Ada Admin (ada.admin@trust-a.example)")
    expect(page.get_by_role("link", name="Export CSV")).to_have_count(0)  # greyed out, no link

    # An impossible day in a typed or linked address is reported, not quietly ignored. The message
    # names it (a date box can't show 30 February), the box is emptied, and Show then works.
    page.goto(f"{base}/audit?from=2026-02-30&to=2026-03-05")
    expect(page.get_by_role("alert")).to_contain_text("Enter valid dates. There's no 2026-02-30.")
    expect(page.get_by_role("table", name="Audit log")).to_have_count(0)
    expect(page.get_by_role("link", name="Export CSV")).to_have_count(0)
    expect(page.get_by_label("From")).to_have_value("")
    expect(page.get_by_label("To")).to_have_value("2026-03-05")
    page.get_by_label("To").fill("")
    show(page)
    expect(page).to_have_url(re.compile(r"/audit$"))
    expect(page.get_by_role("table", name="Audit log")).to_be_visible()
    expect(page.get_by_role("link", name="Export CSV")).to_be_visible()


def test_a_list_row_names_ten_patients_and_counts_them_all(page: Page, as_user):
    hq = as_user("Alex Author").published_hq("workflow-hq.json", f"E2E Audit {unique()}")
    cara = as_user("Cara Clinician")
    for _ in range(2):  # at least two, even on a fresh database
        cara.episode(hq, f"List {unique()}")
    cara.call("GET", cara.h("/episodes"))  # lists every episode at A1 and its patient
    base = sign_in(page.context, "Ada Admin")
    open_audit(page, base)
    pick(page, "Who", "Cara Clinician (cara.clinician@hospital-a.example)")
    show(page, "user=")
    row = log_rows(page).filter(has_text="Listed episodes").first
    expect(row).to_contain_text(re.compile(r"\d+ patients"))
    row.get_by_text(re.compile(r"\d+ patients")).hover()
    expect(page.get_by_role("tooltip")).to_be_visible()


def test_start_fresh_for_the_patient_has_its_own_words_in_the_log(page: Page, as_user, patient_link):
    _, second, cara, _ = trust_b_then_trust_a(as_user, patient_link, {"Your health": {"q_diabetes": "o_no", "snoring": "yes"}})
    form = cara.h(f"/episodes/{second['id']}/patient-form")
    cara.call("GET", form)  # Cara fills in for the patient: Trust B's answers are copied in
    cara.call("POST", f"{form}/prefill/{second['hq']['chapters']['Your health']}", {"choice": "start_fresh"})

    open_audit(page, sign_in(page.context, "Ada Admin"))
    pick(page, "Who", "Cara Clinician (cara.clinician@hospital-a.example)")
    page.get_by_label("Patient").fill(second["name"].removeprefix("Sam "))  # the surname
    show(page, "?user=", "&patient=")
    expect(log_rows(page).filter(has_text="Started a Question Set fresh for the patient")).to_have_count(1)
    expect(log_rows(page).filter(has_text="Checked pre-filled answers with the patient")).to_have_count(0)  # the old, shared words
