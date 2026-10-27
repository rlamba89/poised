"""J11 Triage. The authoring half (S40): an HQ's ASA rules.

A new HQ has the two default rules, in plain words. Alex changes Low risk's target to Ready for POA
and back; tests the whole HQ as a smoker (ASA II) and saves it as a test case, which passes; the
Preview of one set shows the rules that fire for its answers alone, and a red flag stops Low risk.
The Publish window says what the version auto-triages. After Low risk's ASA limit is lowered to I,
the test case fails and Publish is blocked. Sue Super (one hospital) sees the rules read-only.

The patient half (S42): a low-risk patient (a smoker, ASA II, no flags) sends the HQ on their phone.
Cara finds the episode on the Auto-triaged worklist as Ready for admission: the card names the rule,
every Question Set has the stamp, the history says the rule moved it, and the POA Summary shows the
patient's own answers. She clicks Checked: who and when are shown, and it leaves the list."""

import re

from playwright.sync_api import Browser, BrowserContext, Page, expect

from support import choose, invite_path, pick, sign_in, unique

LOW_RISK = "When the suggested ASA is I or II and there are no red flags: auto-triage to Ready for admission"
HIGH_RISK = ("When the suggested ASA is III or more: add the task “Book anaesthetic review” for the anaesthetist team; "
             "flag “Anaesthetic review”; require the anaesthetist's ASA before Ready for admission")


def edit_rule(page: Page, name: str):
    page.get_by_role("button", name=f"Edit {name}").click()
    return page.get_by_test_id("rule-form")


def test_rules_on_a_new_hq_are_tested_and_block_publish(page: Page, browser: Browser, base_url, as_user):
    hq = as_user("Alex Author").draft_hq("rules-hq.json", f"E2E J11 {unique()}")
    base = sign_in(page.context, "Alex Author")
    page.goto(f"{base}/questionnaires/{hq['id']}")

    # The editor's Rules button opens the two default rules in plain words.
    page.get_by_role("banner").get_by_role("link", name="Rules").click()
    expect(page).to_have_url(re.compile(r"/rules$"))
    rules = page.get_by_test_id("rule")
    expect(rules).to_have_count(2)
    expect(rules.nth(0)).to_contain_text("1. Low risk")
    expect(rules.nth(0)).to_contain_text(LOW_RISK)
    expect(rules.nth(1)).to_contain_text("2. High risk")
    expect(rules.nth(1)).to_contain_text(HIGH_RISK)
    expect(page.get_by_role("checkbox", name=re.compile("no ASA grade"))).to_be_checked()

    # Low risk's target: Ready for POA, then back. Each save is kept across a reload.
    form = edit_rule(page, "Low risk")
    pick(page, "Target", "Ready for POA")
    form.get_by_role("button", name="Save rule").click()
    expect(rules.nth(0)).to_contain_text("auto-triage to Ready for POA")
    page.reload()
    expect(page.get_by_test_id("rule").nth(0)).to_contain_text("auto-triage to Ready for POA")
    form = edit_rule(page, "Low risk")
    pick(page, "Target", "Ready for admission")
    form.get_by_role("button", name="Save rule").click()
    expect(page.get_by_test_id("rule").nth(0)).to_contain_text(LOW_RISK)

    # A bad rule is refused in plain words: a task with no title.
    page.get_by_role("button", name="Add rule").click()
    form = page.get_by_test_id("rule-form")
    form.get_by_role("textbox", name="Name").fill("Diabetes")
    form.get_by_role("button", name="+ Add action").click()
    form.get_by_role("button", name="Save rule").click()
    expect(form.get_by_text("Rule “Diabetes”: Enter the task.")).to_be_visible()
    form.get_by_role("button", name="Cancel").click()
    expect(page.get_by_test_id("rule")).to_have_count(2)

    # Test the whole HQ as a smoker: ASA II, Low risk fires. Saved as a test case, it passes.
    page.get_by_role("link", name="Test the whole HQ").click()
    choose(page, "Do you smoke?", "Yes")
    # Unanswered, the snoring question leaves its score Incomplete, which holds the auto-triage as a
    # real send does (S42, test report 1, E-2).
    expect(page.get_by_test_id("hq-fired")).to_have_text("Low risk: no auto-triage: a score is Incomplete")
    choose(page, "How often do you snore?", "Never")
    result = page.get_by_test_id("hq-result")
    expect(result).to_contain_text("Suggested ASA II")
    expect(page.get_by_test_id("hq-fired")).to_have_text("Low risk: auto-triage to Ready for admission")
    page.get_by_role("tab", name="Your heart").click()
    choose(page, "Have you been told you have heart failure?", "No")
    page.get_by_role("textbox", name="Test case name").fill("Smoker")
    page.get_by_role("button", name="Save as test case").click()
    expect(page.get_by_test_id("hq-save-case")).to_contain_text("“Smoker” passes")
    page.get_by_role("link", name="← Rules").click()
    case = page.get_by_test_id("hq-test-case")
    expect(case).to_contain_text("pass")
    expect(case).to_contain_text("Rule fires: Low risk (auto-triage to Ready for admission)")
    expect(case).to_contain_text("Suggested ASA II")

    # The Preview of About you: Rules that fire for its answers alone; a red flag stops Low risk.
    page.goto(f"{base}/chapters/{hq['chapters']['About you']}/preview")
    choose(page, "Do you smoke?", "Yes")
    choose(page, "How often do you snore?", "Never")
    fired = page.get_by_test_id("preview-rules")
    expect(fired).to_contain_text("Low risk: auto-triage to Ready for admission")
    choose(page, "Do you get chest pain when you walk?", "Yes")
    expect(fired).to_contain_text("None")
    expect(fired).not_to_contain_text("Low risk")

    # The Publish window says what the version auto-triages.
    page.goto(f"{base}/questionnaires/{hq['id']}")
    header = page.get_by_role("banner")
    header.get_by_role("button", name="Publish").click()
    dialog = page.get_by_role("dialog", name="Publish version 1?")
    expect(dialog.get_by_test_id("publish-rules")).to_have_text(
        "This version auto-triages: ASA I–II, no red flags → Ready for admission. Patients with no ASA grade count as ASA I. "
        "Never with a red or amber flag, a score Incomplete, or the anaesthetist's ASA required first.")
    expect(dialog.get_by_role("button", name="Publish")).to_be_enabled()
    dialog.get_by_role("button", name="Cancel").click()

    # Low risk's limit lowered to ASA I: the smoker's case fails, and Publish is blocked.
    header.get_by_role("link", name="Rules").click()
    form = edit_rule(page, "Low risk")
    pick(page, "ASA grade", "I")
    form.get_by_role("button", name="Save rule").click()
    expect(page.get_by_test_id("rule").nth(0)).to_contain_text("When the suggested ASA is I and there are no red flags")
    expect(page.get_by_test_id("hq-test-case")).to_contain_text("fail")
    page.get_by_role("link", name=f"← {hq['name']}").click()
    page.get_by_role("banner").get_by_role("button", name="Publish").click()
    dialog = page.get_by_role("dialog", name="Publish version 1?")
    expect(dialog.get_by_test_id("publish-problems")).to_contain_text("Rules · Test case “Smoker” fails")
    expect(dialog.get_by_role("button", name="Publish")).to_be_disabled()
    dialog.get_by_role("button", name="Cancel").click()

    # The failing case, opened, is saved again under its name (Enter saves): it's replaced, not
    # copied, and passes with the new lines (test report 1, E-2, E-4).
    page.get_by_role("banner").get_by_role("link", name="Rules").click()
    page.get_by_test_id("hq-test-case").get_by_role("link", name="Open").click()
    name = page.get_by_role("textbox", name="Test case name")
    expect(name).to_have_value("Smoker")
    expect(page.get_by_role("button", name="Replace test case")).to_be_visible()
    name.press("Enter")
    expect(page.get_by_test_id("hq-save-case")).to_contain_text("“Smoker” passes")
    page.get_by_role("link", name="← Rules").click()
    case = page.get_by_test_id("hq-test-case")
    expect(case).to_have_count(1)
    expect(case).to_contain_text("pass")
    expect(case).to_contain_text("Suggested ASA II")
    expect(case).not_to_contain_text("Rule fires: Low risk")

    # Its bin asks first, as a rule's does (E-3).
    page.get_by_role("button", name="Delete Smoker").click()
    case.get_by_role("button", name="Keep it").click()
    expect(case).to_have_count(1)
    page.get_by_role("button", name="Delete Smoker").click()
    case.get_by_role("button", name="Delete", exact=True).click()
    expect(page.get_by_test_id("hq-test-case")).to_have_count(0)

    # Sue Super (Hospital A1 only) sees the trust's rules read-only.
    sue = browser.new_context(viewport={"width": 1440, "height": 900}, base_url=base_url)
    try:
        sue_page = sue.new_page()
        sue_base = sign_in(sue, "Sue Super")
        sue_page.goto(f"{sue_base}/questionnaires/{hq['id']}/rules")
        expect(sue_page.get_by_test_id("rule")).to_have_count(2)
        expect(sue_page.get_by_text("Read only", exact=True)).to_be_visible()
        expect(sue_page.get_by_role("button", name="Add rule")).to_have_count(0)
        expect(sue_page.get_by_role("button", name="Edit Low risk")).to_have_count(0)
        expect(sue_page.get_by_role("checkbox", name=re.compile("no ASA grade"))).to_be_disabled()
    finally:
        sue.close()


def test_the_preview_fires_rules_on_a_clinician_only_score_in_the_patient_view(page: Page, as_user):
    """Test report 1, F-1: the Preview's "Rules that fire" is worked out as the rules see the answers on
    send (the clinician's view), so a clinician-only score counts in the Patient view too."""
    alex = as_user("Alex Author")
    hq = alex.draft_hq("rules-hq.json", f"E2E J11 preview {unique()}")
    path = alex.h(f"/questionnaires/{hq['id']}")
    q = alex.call("GET", path)["questionnaire"]
    rules = q["rules"]
    rules["rules"].append({"id": "snorer", "name": "Snorer",
                           "when": {"asa": {"op": ">=", "grade": 1}, "band": {"score": "q_snore_score", "bands": ["b_high"]}},
                           "then": [{"kind": "flag", "name": "Sleep apnoea"}]})
    alex.call("PUT", f"{path}/rules", {"rules": rules, "revision": q["rulesRevision"], "versionId": q["versionId"]})

    base = sign_in(page.context, "Alex Author")
    page.goto(f"{base}/chapters/{hq['chapters']['About you']}/preview")
    expect(page.get_by_test_id("viewer-switch").get_by_text("Patient")).to_be_visible()
    fired = page.get_by_test_id("preview-rules")
    expect(fired).not_to_contain_text("Snorer")
    choose(page, "How often do you snore?", "Every night")
    expect(fired).to_contain_text("Snorer: flag “Sleep apnoea”")
    choose(page, "How often do you snore?", "Never")
    expect(fired).not_to_contain_text("Snorer")


def test_one_change_saves_at_a_time(page: Page, as_user):
    """Test report 3, F-1: while one change is saving (its PUT held here), an open rule form's Save rule,
    Add rule and every other control wait, so no second save goes out with the same revision; once
    the first lands, the rule saves."""
    hq = as_user("Alex Author").draft_hq("rules-hq.json", f"E2E J11 one save {unique()}")
    base = sign_in(page.context, "Alex Author")
    page.goto(f"{base}/questionnaires/{hq['id']}/rules")
    page.get_by_role("button", name="Add rule").click()
    form = page.get_by_test_id("rule-form")
    form.get_by_role("textbox", name="Name").fill("Race")
    form.get_by_role("button", name="+ Add action").click()
    form.get_by_role("textbox", name="Task").fill("Race task")

    held = []
    page.route("**/rules", lambda route: held.append(route) if route.request.method == "PUT" else route.continue_())
    page.get_by_role("checkbox", name=re.compile("no ASA grade")).click()
    expect(form.get_by_role("button", name="Save rule")).to_be_disabled()
    expect(page.get_by_role("button", name="Edit Low risk")).to_be_disabled()
    assert len(held) == 1, f"{len(held)} saves sent"
    held[0].continue_()
    page.unroute("**/rules")

    save = form.get_by_role("button", name="Save rule")
    expect(save).to_be_enabled()
    save.click()
    rules = page.get_by_test_id("rule")
    expect(rules).to_have_count(3)
    expect(rules.nth(2)).to_contain_text("add the task “Race task” for the nurse team")
    page.reload()
    expect(page.get_by_test_id("rule")).to_have_count(3)
    expect(page.get_by_role("checkbox", name=re.compile("no ASA grade"))).not_to_be_checked()


def test_a_low_risk_patient_is_auto_triaged_and_checked(page: Page, phone: BrowserContext, as_user):
    hq = as_user("Alex Author").published_hq("rules-hq.json", f"E2E J11 send {unique()}", auto_triage=True)
    episode = as_user("Cara Clinician").episode(hq, f"J11 {unique()}")

    # The patient, on their phone: a smoker with no chest pain or heart failure, who answers the
    # snoring question too (an unanswered one would leave its score Incomplete, which stops auto-triage).
    patient = phone.new_page()
    patient.goto(invite_path(episode["email"]))
    patient.get_by_role("button", name="Continue").click()
    patient.get_by_label("Day").fill("17")
    patient.get_by_label("Month").fill("5")
    patient.get_by_label("Year").fill("1980")
    patient.get_by_role("button", name="Continue").click()
    patient.get_by_test_id("set-About you").get_by_role("button", name="Start").click()
    choose(patient, "Do you smoke?", "Yes")
    choose(patient, "Do you get chest pain when you walk?", "No")
    choose(patient, "How often do you snore?", "Never")
    patient.get_by_role("button", name="Finish this section").click()
    patient.get_by_test_id("set-Your heart").get_by_role("button", name="Start").click()
    choose(patient, "Have you been told you have heart failure?", "No")
    patient.get_by_role("button", name="Finish this section").click()
    assert patient.evaluate("document.documentElement.scrollWidth <= window.innerWidth")
    patient.get_by_role("button", name="Send my answers").click()
    patient.get_by_role("dialog", name="Send your answers?").get_by_role("button", name="Send my answers").click()
    expect(patient.get_by_role("heading", name="Thank you, Sam")).to_be_visible()

    # Cara: on Auto-triaged, as Ready for admission.
    base = sign_in(page.context, "Cara Clinician")
    page.goto(f"{base}/episodes")
    page.get_by_test_id("worklist-auto_triaged").click()
    expect(page).to_have_url(f"{base}/episodes?worklist=auto_triaged")
    expect(page.get_by_test_id("worklist-help")).to_have_text("Moved on automatically by the HQ's rules; not yet checked by a clinician.")
    row = page.get_by_role("row").filter(has_text=episode["name"])
    expect(row).to_contain_text("Ready for admission")
    row.click()

    # The episode: the card names the rule, each set has the stamp, the history names the rule.
    card = page.get_by_test_id("auto-triage")
    expect(card).to_contain_text("Auto-triaged by the rule “Low risk”")
    expect(card).to_contain_text("Moved to Ready for admission automatically when the patient sent the HQ (ASA II, no red or amber flags)")
    expect(card).to_contain_text("Not yet checked by a clinician.")
    expect(page.get_by_test_id("triage-stamp")).to_have_count(3)  # About you, Your heart, Nurse checks
    expect(page.get_by_test_id("triage-stamp").first).to_have_text("Auto-triaged (ASA II, no red or amber flags)")
    history = page.get_by_test_id("status-history-row").filter(has_text="Auto-triaged")
    expect(history).to_contain_text("Auto-triaged: Ready for review → Ready for admission")
    expect(history).to_contain_text("Automatic (Low risk)")
    expect(page.get_by_text("Auto-triaged by the rule “Low risk”: ASA II, no red or amber flags → Ready for admission")).to_be_visible()

    # The POA Summary: the patient's answers, with the stamp.
    page.get_by_role("link", name="POA Summary").click()
    expect(page.get_by_test_id("poa-auto-triaged")).to_contain_text("No clinician validated these answers")
    expect(page.get_by_test_id("poa-set-About you")).to_contain_text("Auto-triaged (ASA II, no red or amber flags) · Patient on")
    expect(page.get_by_test_id("poa-set-About you")).to_contain_text("Smoker")
    page.go_back()

    # Checked: who and when, and it leaves the list.
    card.get_by_role("button", name="Checked").click()
    expect(page.get_by_test_id("triage-checked")).to_contain_text("Checked by Cara Clinician on")
    expect(card.get_by_role("button", name="Checked")).to_have_count(0)
    page.reload()
    expect(page.get_by_test_id("triage-checked")).to_be_visible()
    page.goto(f"{base}/episodes?worklist=auto_triaged")
    expect(page.get_by_test_id("worklist-help")).to_be_visible()
    expect(page.get_by_role("row").filter(has_text=episode["name"])).to_have_count(0)


def test_a_second_tabs_checked_is_refused_in_plain_words(page: Page, as_user, patient_link):
    """Test report 1, F-1: Checked clicked in a tab opened before someone else checked it says so, and
    the card shows who checked it."""
    hq = as_user("Alex Author").published_hq("rules-hq.json", f"E2E J11 two tabs {unique()}", auto_triage=True)
    episode = as_user("Cara Clinician").episode(hq, f"J11 tabs {unique()}")
    p = patient_link(episode)
    p.answer("About you", {"q_smokes": "o_yes", "q_chest_pain": "o_no", "q_snore": "o_never"})
    p.answer("Your heart", {"q_heart_failure": "o_no"})
    p.submit()

    base = sign_in(page.context, "Cara Clinician")
    page.goto(f"{base}/episodes/{episode['id']}")
    second = page.context.new_page()
    second.goto(f"{base}/episodes/{episode['id']}")
    expect(second.get_by_test_id("auto-triage").get_by_role("button", name="Checked")).to_be_visible()
    page.get_by_test_id("auto-triage").get_by_role("button", name="Checked").click()
    expect(page.get_by_test_id("triage-checked")).to_be_visible()

    second.get_by_test_id("auto-triage").get_by_role("button", name="Checked").click()
    card = second.get_by_test_id("auto-triage")
    expect(card.get_by_test_id("triage-check-error")).to_contain_text("This episode changed since you opened it")
    expect(card.get_by_test_id("triage-checked")).to_contain_text("Checked by Cara Clinician on")
    expect(card.get_by_role("button", name="Checked")).to_have_count(0)
