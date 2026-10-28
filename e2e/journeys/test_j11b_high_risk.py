"""J11b A high-risk patient (S43): the HQ rules' task, flag and the anaesthetist's ASA.

On an HQ published with the default rules, a patient with heart failure (ASA III) sends the HQ on
their phone. High risk fires: Cara finds the episode on Anaesthetic review needed with the red
"Anaesthetic review" flag, the task "Book anaesthetic review" added by the rule, and the episode
Ready for review (Low risk doesn't fire at ASA III). Once the review is complete, Ready for admission
is refused, in plain words, until the anaesthetist's ASA grade is added on the POA Summary; then it
works. S44b: the grade is a history. Cara adds ASA III (no reason asked), re-grades to ASA IV with a
reason (both stay; IV counts), marks IV entered in error (struck through, with who and why; III
counts again); at Ready for admission the only grade can't be marked. Cara clears the flag with a
reason: the badge goes, and who, when and why stay. Then Admit works.

Its second test: a rule that suggests a status shows the line on the episode and changes nothing."""

import datetime as dt
import re

from playwright.sync_api import BrowserContext, Page, expect

from support import change_status, choose, invite_path, pick, sign_in, unique

NEEDS_GRADE = "Add the anaesthetist's ASA grade first."


def test_a_high_risk_patient_gets_the_rules_task_flag_and_asa_requirement(page: Page, phone: BrowserContext, as_user):
    hq = as_user("Alex Author").published_hq("rules-hq.json", f"E2E J11b {unique()}", auto_triage=True)
    cara = as_user("Cara Clinician")
    episode = cara.episode(hq, f"J11b {unique()}")

    # The patient, on their phone: no smoking, chest pain or snoring, but heart failure (ASA III).
    patient = phone.new_page()
    patient.goto(invite_path(episode["email"]))
    patient.get_by_role("button", name="Continue").click()
    patient.get_by_label("Day").fill("17")
    patient.get_by_label("Month").fill("5")
    patient.get_by_label("Year").fill("1980")
    patient.get_by_role("button", name="Continue").click()
    patient.get_by_test_id("set-About you").get_by_role("button", name="Start").click()
    choose(patient, "Do you smoke?", "No")
    choose(patient, "Do you get chest pain when you walk?", "No")
    choose(patient, "How often do you snore?", "Never")
    patient.get_by_role("button", name="Finish this section").click()
    patient.get_by_test_id("set-Your heart").get_by_role("button", name="Start").click()
    choose(patient, "Have you been told you have heart failure?", "Yes")
    patient.get_by_role("button", name="Finish this section").click()
    assert patient.evaluate("document.documentElement.scrollWidth <= window.innerWidth")
    patient.get_by_role("button", name="Send my answers").click()
    patient.get_by_role("dialog", name="Send your answers?").get_by_role("button", name="Send my answers").click()
    expect(patient.get_by_role("heading", name="Thank you, Sam")).to_be_visible()
    # The patient sees nothing of the rules.
    expect(patient.get_by_text("Anaesthetic review")).to_have_count(0)

    # Cara: on Anaesthetic review needed, with the flag on the row.
    base = sign_in(page.context, "Cara Clinician")
    page.goto(f"{base}/episodes")
    page.get_by_test_id("worklist-anaesthetic_review").click()
    row = page.get_by_role("row").filter(has_text=episode["name"])
    expect(row.get_by_test_id("flag-badge")).to_have_text("Anaesthetic review")
    expect(row).to_contain_text("Ready for review")
    row.click()

    # The episode: the flag badge and card, the rule's task, the requirement, the notes.
    expect(page).to_have_url(re.compile(rf"/episodes/{episode['id']}$"))
    expect(page.get_by_role("combobox", name="Episode status")).to_have_value("Ready for review")
    expect(page.get_by_test_id("flag-badge")).to_have_text("Anaesthetic review")
    flag = page.get_by_test_id("flag")
    expect(flag).to_contain_text("Added by the rule “High risk” on")
    task = page.get_by_test_id("task").filter(has_text="Book anaesthetic review")
    expect(task).to_contain_text("Anaesthetist")
    expect(task).to_contain_text("Added by rule: High risk on")
    expect(page.get_by_test_id("asa-required")).to_have_text(
        "Ready for admission needs the anaesthetist's ASA grade first (the rule “High risk”). Add it on the POA Summary.")
    for line in ("Task added by the rule “High risk” for the anaesthetist team: Book anaesthetic review",
                 "Flag added by the rule “High risk”: Anaesthetic review",
                 "The rule “High risk” requires the anaesthetist's ASA grade before Ready for admission"):
        expect(page.get_by_text(line)).to_be_visible()
    expect(page.get_by_test_id("auto-triage")).to_have_count(0)

    # The review completed (by the API), Ready for admission is offered but refused until the grade is saved.
    for name, answers in (("About you", {"q_smokes": "o_no", "q_chest_pain": "o_no", "q_snore": "o_never"}),
                          ("Your heart", {"q_heart_failure": "o_yes"}), ("Nurse checks", {})):
        cara.validate(episode, name, answers)
    cara.complete_review(episode)
    page.reload()
    page.get_by_role("combobox", name="Episode status").click()
    blocked = page.get_by_role("option", name="Ready for admission (add the anaesthetist's ASA grade first)")
    expect(blocked).to_be_visible()
    expect(blocked).to_have_attribute("data-combobox-disabled", "true")
    page.keyboard.press("Escape")
    path = cara.h(f"/episodes/{episode['id']}")
    revision = cara.call("GET", path)["episode"]["revision"]
    res = cara.request.fetch(f"/api{path}/status", method="PATCH", data={"to": "ready_for_admission", "revision": revision})
    assert res.status == 409 and NEEDS_GRADE in res.text(), f"{res.status} {res.text()}"

    # The anaesthetist's grade, added on the POA Summary, meets it (S44b: a history, never edited).
    page.get_by_role("link", name="POA Summary").click()
    expect(page.get_by_test_id("poa-asa-required")).to_have_text(
        "Ready for admission needs the anaesthetist's ASA grade first (the rule “High risk”).")
    add = page.get_by_test_id("asa-add")
    expect(add.get_by_role("textbox", name="Why is the grade changed?")).to_have_count(0)  # the first grade needs no reason
    pick(page, "Anaesthetist ASA grade", "ASA III")
    add.get_by_role("button", name="Add grade").click()
    expect(page.get_by_test_id("anaesthetist-asa")).to_have_text("ASA III")
    expect(page.get_by_test_id("poa-asa-required")).to_have_text("The anaesthetist's ASA grade is added, as the rule “High risk” asks.")
    expect(page.get_by_text("Anaesthetist ASA grade set to 3")).to_be_visible()

    # A re-grade needs a reason; both grades stay, and the latest counts.
    pick(page, "Anaesthetist ASA grade", "ASA IV")
    add.get_by_role("button", name="Add grade").click()
    expect(add.get_by_text("Say why the grade is changed.")).to_be_visible()
    add.get_by_role("textbox", name="Why is the grade changed?").fill("New echo result")
    add.get_by_role("button", name="Add grade").click()
    expect(page.get_by_test_id("anaesthetist-asa")).to_have_text("ASA IV")
    history = page.get_by_test_id("asa-grade")
    expect(history).to_have_count(2)
    expect(history.first).to_contain_text("ASA IV")
    expect(history.first).to_contain_text("Counts")
    expect(history.first).to_contain_text("Given by Cara Clinician on")
    expect(history.first).to_contain_text("New echo result")
    expect(history.last).to_contain_text("ASA III")
    expect(page.get_by_text("Anaesthetist ASA grade changed from 3 to 4: New echo result")).to_be_visible()

    # At about 500 px the buttons keep their whole label, even beside a wrapped line (test report 1, F1).
    size = page.viewport_size
    page.set_viewport_size({"width": 500, "height": 800})
    for button in history.get_by_role("button", name=re.compile("entered in error")).all():
        label = button.locator(".mantine-Button-label")
        assert label.evaluate("el => el.scrollWidth <= el.clientWidth"), "an Entered in error button is cut off"
    page.set_viewport_size(size)

    # ASA IV marked entered in error: struck through, still shown, and ASA III counts again.
    history.first.get_by_role("button", name=re.compile(r"^Mark ASA IV, given on .*, entered in error$")).click()
    form = page.get_by_test_id("asa-error-form")
    form.get_by_role("button", name="Mark entered in error").click()
    expect(form.get_by_text("Say why the grade was entered in error.")).to_be_visible()
    form.get_by_role("textbox", name="Why was it entered in error?").fill("Typed in the wrong box")
    form.get_by_role("button", name="Mark entered in error").click()
    expect(page.get_by_test_id("anaesthetist-asa")).to_have_text("ASA III")
    struck = page.locator('[data-testid="asa-grade"][data-error="true"]')
    expect(struck).to_contain_text("ASA IV")
    expect(struck.get_by_test_id("asa-grade-error")).to_contain_text("Entered in error: marked by Cara Clinician on")
    expect(struck.get_by_test_id("asa-grade-error")).to_contain_text("Typed in the wrong box")
    expect(struck.get_by_role("button", name=re.compile("entered in error"))).to_have_count(0)  # can't be unmarked
    expect(history.last).to_contain_text("Counts")
    page.reload()
    expect(page.get_by_test_id("anaesthetist-asa")).to_have_text("ASA III")
    expect(page.locator('[data-testid="asa-grade"][data-error="true"]')).to_contain_text("ASA IV")

    page.go_back()
    expect(page.get_by_test_id("asa-required")).to_have_count(0)
    change_status(page, "Ready for admission")

    # At Ready for admission, the only grade that counts can't be marked entered in error.
    page.get_by_role("link", name="POA Summary").click()
    only = page.locator('[data-testid="asa-grade"][data-error="false"]')
    only.get_by_role("button", name=re.compile(r"^Mark ASA III, given on .*, entered in error$")).click()
    form = page.get_by_test_id("asa-error-form")
    form.get_by_role("textbox", name="Why was it entered in error?").fill("Trying it")
    form.get_by_role("button", name="Mark entered in error").click()
    expect(form.get_by_text("Add the correct grade first, or move the episode back from Ready for admission.")).to_be_visible()
    form.get_by_role("button", name="Back").click()
    expect(page.get_by_test_id("anaesthetist-asa")).to_have_text("ASA III")
    page.go_back()

    # Clearing the flag: a reason first; then the badge goes, and who, when and why stay.
    flag.get_by_role("button", name="Clear the flag Anaesthetic review").click()
    form = page.get_by_test_id("flag-form")
    form.get_by_role("button", name="Clear flag").click()
    expect(form.get_by_text("Say why the flag is cleared.")).to_be_visible()
    form.get_by_role("textbox", name="Why is it cleared?").fill("Reviewed by Dr Example")
    form.get_by_role("textbox", name="Why is it cleared?").press("Enter")
    expect(page.get_by_test_id("flag-badge")).to_have_count(0)
    expect(flag.get_by_test_id("flag-cleared")).to_contain_text("Cleared by Cara Clinician on")
    expect(flag.get_by_test_id("flag-cleared")).to_contain_text("Reviewed by Dr Example")
    expect(page.get_by_text("Flag cleared: Anaesthetic review: Reviewed by Dr Example")).to_be_visible()
    page.reload()
    expect(page.get_by_test_id("flag-badge")).to_have_count(0)
    expect(page.get_by_test_id("flag-cleared")).to_be_visible()
    # Admit, as before (the grade can't have gone missing).
    change_status(page, "Admitted", date=(dt.date.today() - dt.timedelta(days=1)).isoformat())
    page.goto(f"{base}/episodes")
    expect(page.get_by_role("row").filter(has_text=episode["name"]).get_by_test_id("flag-badge")).to_have_count(0)


def test_a_suggested_status_is_shown_and_changes_nothing(page: Page, as_user, patient_link):
    alex = as_user("Alex Author")
    hq = alex.draft_hq("rules-hq.json", f"E2E J11b suggest {unique()}")
    path = alex.h(f"/questionnaires/{hq['id']}")
    q = alex.call("GET", path)["questionnaire"]
    rules = q["rules"]
    rules["rules"] = [{"id": "fit", "name": "Fit for POA", "when": {"asa": {"op": "<=", "grade": 2}},
                       "then": [{"kind": "suggestStatus", "status": "ready_for_poa"}]}]
    alex.call("PUT", f"{path}/rules", {"rules": rules, "revision": q["rulesRevision"], "versionId": q["versionId"]})
    alex.call("POST", f"{path}/publish", {"versionId": hq["versionId"]})

    episode = as_user("Cara Clinician").episode(hq, f"J11b suggest {unique()}")
    p = patient_link(episode)
    p.answer("About you", {"q_smokes": "o_yes", "q_chest_pain": "o_no", "q_snore": "o_never"})
    p.answer("Your heart", {"q_heart_failure": "o_no"})
    p.submit()

    base = sign_in(page.context, "Cara Clinician")
    page.goto(f"{base}/episodes/{episode['id']}")
    expect(page.get_by_test_id("rule-suggestion")).to_have_text("The HQ's rules suggest: Ready for POA (the rule “Fit for POA”)")
    expect(page.get_by_role("combobox", name="Episode status")).to_have_value("Ready for review")
    expect(page.get_by_test_id("flags")).to_have_count(0)
    expect(page.get_by_test_id("asa-required")).to_have_count(0)
    page.reload()
    expect(page.get_by_test_id("rule-suggestion")).to_be_visible()
    expect(page).to_have_url(re.compile(rf"/episodes/{episode['id']}$"))

    # S44a: once the episode reaches the suggested status, the line goes (the history keeps it).
    cara = as_user("Cara Clinician")
    cara.validate(episode, "About you", {"q_smokes": "o_yes", "q_chest_pain": "o_no", "q_snore": "o_never"})
    cara.validate(episode, "Your heart", {"q_heart_failure": "o_no"})
    cara.validate(episode, "Nurse checks", {})
    cara.complete_review(episode)
    page.reload()
    expect(page.get_by_role("combobox", name="Episode status")).to_have_value("Ready for POA")
    expect(page.get_by_test_id("rule-suggestion")).to_have_count(0)


def test_a_stale_tabs_clear_is_told_and_closes_once_reloaded(page: Page, as_user, patient_link):
    """Test report 1, F-1: a Clear sent from a tab opened before someone else cleared the flag is told
    the episode changed (the reason kept); after Reload the flag shows as cleared and its form closes.
    The reason box is focused when the form opens (exploring note 3)."""
    hq = as_user("Alex Author").published_hq("rules-hq.json", f"E2E J11b tabs {unique()}", auto_triage=True)
    episode = as_user("Cara Clinician").episode(hq, f"J11b tabs {unique()}")
    p = patient_link(episode)
    p.answer("About you", {"q_smokes": "o_no", "q_chest_pain": "o_no", "q_snore": "o_never"})
    p.answer("Your heart", {"q_heart_failure": "o_yes"})
    p.submit()

    base = sign_in(page.context, "Cara Clinician")
    page.goto(f"{base}/episodes/{episode['id']}")
    second = page.context.new_page()
    second.goto(f"{base}/episodes/{episode['id']}")
    expect(second.get_by_test_id("flag-badge")).to_be_visible()

    page.get_by_role("button", name="Clear the flag Anaesthetic review").click()
    reason = page.get_by_role("textbox", name="Why is it cleared?")
    expect(reason).to_be_focused()
    page.keyboard.type("Tab one reason")
    reason.press("Enter")
    expect(page.get_by_test_id("flag-cleared")).to_contain_text("Tab one reason")

    second.get_by_role("button", name="Clear the flag Anaesthetic review").click()
    second.keyboard.type("Tab two reason")
    second.keyboard.press("Enter")
    form = second.get_by_test_id("flag-form")
    expect(form.get_by_text("This episode changed since you opened it")).to_be_visible()
    expect(form.get_by_role("textbox", name="Why is it cleared?")).to_have_value("Tab two reason")
    form.get_by_role("button", name="Reload").click()
    expect(second.get_by_test_id("flag-cleared")).to_contain_text("Tab one reason")
    expect(second.get_by_test_id("flag-form")).to_have_count(0)
    expect(second.get_by_role("button", name="Clear flag")).to_have_count(0)
    expect(second.get_by_role("button", name="Clear the flag Anaesthetic review")).to_have_count(0)
