"""Every status (S20; EPI-01, EPI-03, EPI-07, S20 answers 2 and 6).

  - Cara walks a reviewed episode to Discharged, entering the admission and discharge dates; the
    status history shows each move with who made it. She undoes the discharge (with a reason).
  - Sue Super re-opens the HQ review with a reason; Cara has no Re-open.
  - Two tabs: a move made in a stale tab is refused with "changed since you opened it", and the
    ASA grade typed in the stale tab is kept.
"""

from playwright.sync_api import Page, expect

from support import PATIENT_ANSWERS, change_status, sign_in, unique


def reviewed_episode(as_user, patient_link, name: str) -> dict:
    """An episode whose patient has sent the HQ and whose review Cara has completed."""
    hq = as_user("Alex Author").published_hq("workflow-hq.json", f"E2E {name} {unique()}")
    cara = as_user("Cara Clinician")
    episode = cara.episode(hq, f"{name} {unique()}")
    patient = patient_link(episode)
    for set_name, answers in PATIENT_ANSWERS.items():
        patient.answer(set_name, answers)
    patient.submit()
    for set_name, answers in {**PATIENT_ANSWERS, "Nurse checks": {"q_bp": "120/80"}}.items():
        cara.validate(episode, set_name, answers)
    cara.complete_review(episode)
    return episode


def test_walk_to_discharged_then_undo(page: Page, as_user, patient_link):
    episode = reviewed_episode(as_user, patient_link, "Walk")
    base = sign_in(page.context, "Cara Clinician")
    page.goto(f"{base}/episodes/{episode['id']}")
    control = page.get_by_test_id("status-control")
    expect(control.get_by_role("button", name="Re-open HQ review")).to_have_count(0)  # a clinician can't

    # Enter in the reason sends the form (test report 1, F-1); the reason is one line, at most 1,000 characters.
    control.get_by_role("combobox", name="Episode status").click()
    page.get_by_role("option", name="Ready for admission", exact=True).click()
    reason = page.get_by_test_id("status-form").get_by_label("Reason (optional)")
    reason.fill("x" * 1001)
    assert len(reason.input_value()) == 1000
    reason.fill("Made-up: all clear")
    reason.press("Enter")
    expect(control.get_by_role("combobox", name="Episode status")).to_have_value("Ready for admission")
    change_status(page, "Admitted", date="2026-01-10")
    change_status(page, "Discharged", date="2026-01-12")
    expect(page.get_by_text("Admission date", exact=True)).to_be_visible()
    rows = page.get_by_test_id("status-history-row")
    expect(rows.last).to_contain_text("Admitted → Discharged")
    expect(rows.last).to_contain_text("Cara Clinician")
    expect(page.get_by_test_id("status-history")).to_contain_text("Made-up: all clear")

    # Discharged pressed by mistake: undo, with a reason.
    control.get_by_role("button", name="Undo: back to Admitted").click()
    form = page.get_by_test_id("status-form")
    form.get_by_role("button", name="Undo").click()
    expect(form.get_by_text("Say why the status change is being undone.")).to_be_visible()
    form.get_by_label("Reason").fill("Made-up: pressed by mistake")
    form.get_by_label("Reason").press("Enter")
    expect(control.get_by_role("combobox", name="Episode status")).to_have_value("Admitted")
    expect(rows.last).to_contain_text("Undone: Discharged → Admitted")
    expect(page.get_by_text("Discharge date", exact=True)).to_have_count(0)


def test_reopen_needs_a_super_clinician(page: Page, as_user, patient_link):
    episode = reviewed_episode(as_user, patient_link, "Reopen")
    base = sign_in(page.context, "Sue Super")
    page.goto(f"{base}/episodes/{episode['id']}")
    control = page.get_by_test_id("status-control")
    control.get_by_role("button", name="Re-open HQ review").click()
    form = page.get_by_test_id("status-form")
    form.get_by_label("Reason").fill("Made-up: the cardiac history was missed")
    form.get_by_role("button", name="Re-open review").click()
    expect(control.get_by_role("combobox", name="Episode status")).to_have_value("Ready for review")
    expect(page.get_by_role("button", name="Complete HQ review")).to_be_visible()
    expect(page.get_by_test_id("status-history-row").last).to_contain_text("Made-up: the cardiac history was missed")


def test_a_stale_tab_is_told_and_keeps_its_input(page: Page, as_user, patient_link):
    episode = reviewed_episode(as_user, patient_link, "Tabs")
    base = sign_in(page.context, "Cara Clinician")
    poa = f"{base}/episodes/{episode['id']}/poa"
    first, second = page, page.context.new_page()
    first.goto(poa)
    second.goto(poa)

    # The Nurse ASA in one tab, then the Anaesthetist ASA in the other.
    first.get_by_role("combobox", name="Nurse ASA grade").click()
    first.get_by_role("option", name="ASA II", exact=True).click()
    first.get_by_role("button", name="Save").first.click()
    expect(first.get_by_text("Nurse ASA grade set to 2")).to_be_visible()
    second.get_by_role("combobox", name="Anaesthetist ASA grade").click()
    second.get_by_role("option", name="ASA III", exact=True).click()
    second.get_by_role("button", name="Save").last.click()
    expect(second.get_by_text("This episode changed since you opened it.")).to_be_visible()
    expect(second.get_by_role("combobox", name="Anaesthetist ASA grade")).to_have_value("ASA III")
    second.get_by_role("button", name="Reload").click()
    expect(second.get_by_role("combobox", name="Nurse ASA grade")).to_have_value("ASA II")
    expect(second.get_by_role("combobox", name="Anaesthetist ASA grade")).to_have_value("ASA III")
    second.get_by_role("button", name="Save").last.click()
    expect(second.get_by_text("Anaesthetist ASA grade set to 3")).to_be_visible()

    # A status move from a tab that's out of date.
    first.get_by_role("combobox", name="Episode status").click()
    first.get_by_role("option", name="On hold", exact=True).click()
    first.get_by_test_id("status-form").get_by_role("button", name="Change status").click()
    expect(first.get_by_test_id("status-form").get_by_text("This episode changed since you opened it.")).to_be_visible()


def test_after_reload_a_move_no_longer_offered_closes_its_form(page: Page, as_user, patient_link):
    """Test report 1: a stale tab's form for a move the episode no longer offers closes on Reload."""
    episode = reviewed_episode(as_user, patient_link, "Gone")
    base = sign_in(page.context, "Cara Clinician")
    path = f"{base}/episodes/{episode['id']}"
    first, second = page, page.context.new_page()
    first.goto(path)
    change_status(first, "Ready for admission")
    second.goto(path)

    # Tab 1 starts Admitted; tab 2 undoes back to Ready for POA, where Admitted isn't offered.
    first.get_by_role("combobox", name="Episode status").click()
    first.get_by_role("option", name="Admitted", exact=True).click()
    second.get_by_role("button", name="Undo: back to Ready for POA").click()
    second.get_by_test_id("status-form").get_by_label("Reason").fill("Made-up: too soon")
    second.get_by_test_id("status-form").get_by_role("button", name="Undo").click()
    expect(second.get_by_role("combobox", name="Episode status")).to_have_value("Ready for POA")

    form = first.get_by_test_id("status-form")
    form.get_by_role("button", name="Change status").click()
    expect(form.get_by_text("This episode changed since you opened it.")).to_be_visible()
    form.get_by_role("button", name="Reload").click()
    expect(first.get_by_test_id("status-form")).to_have_count(0)
    expect(first.get_by_text("The episode changed, so that change is no longer offered.")).to_be_visible()
    expect(first.get_by_role("combobox", name="Episode status")).to_have_value("Ready for POA")
