"""Tasks, worklists and status messages (S21; TSK-01, EPI-02, EPI-08).

  - Cara adds a task for the anaesthetist team on an episode; on the clinician home the episode is
    in the Open tasks and Anaesthetic review needed worklists, with counts on the tabs. She
    completes the task (who and when are shown), and the episode leaves Open tasks; re-opened, it
    comes back. The worklist tab is kept in the URL (reload, Back).
  - A hospital that switches on the Ready for admission email: Cara moves a reviewed episode there,
    and the patient's captured email has the hospital's own wording, with no link. The switch is put
    back off at the end.
"""

from playwright.sync_api import Page, expect

from support import PATIENT_ANSWERS, captured, change_status, sign_in, unique


def episode_row(page: Page, name: str):
    return page.get_by_role("row").filter(has_text=name)


def test_a_task_puts_the_episode_in_open_tasks_until_it_is_done(page: Page, as_user):
    hq = as_user("Alex Author").published_hq("workflow-hq.json", f"E2E Tasks {unique()}")
    episode = as_user("Cara Clinician").episode(hq, f"Tasks {unique()}")
    base = sign_in(page.context, "Cara Clinician")
    page.goto(f"{base}/episodes/{episode['id']}")

    # Add a task for the anaesthetist team; Enter sends the form, and the app's own message
    # shows for a missing team (noValidate, S19).
    tasks = page.get_by_test_id("tasks")
    expect(tasks.get_by_text("No tasks yet.")).to_be_visible()
    tasks.get_by_role("button", name="Add task").click()
    form = page.get_by_test_id("task-form")
    form.get_by_label("Task").fill("Made-up: book anaesthetic review")
    form.get_by_label("Task").press("Enter")
    expect(form.get_by_text("Choose the team.")).to_be_visible()
    form.get_by_label("Team").select_option("Anaesthetist")
    form.get_by_label("Task").press("Enter")
    task = tasks.get_by_test_id("task")
    expect(task).to_have_count(1)
    expect(task).to_contain_text("Made-up: book anaesthetic review")
    expect(task).to_contain_text("Anaesthetist")
    expect(task).to_contain_text("Added by Cara Clinician")
    expect(page.get_by_text("Task added for the anaesthetist team: Made-up: book anaesthetic review")).to_be_visible()

    # The clinician home: the worklist tabs, with counts; the episode is in Open tasks and in
    # Anaesthetic review needed, not in On hold.
    page.goto(f"{base}/episodes")
    open_tab = page.get_by_test_id("worklist-open_tasks")
    expect(open_tab).to_be_visible()
    assert int(open_tab.get_attribute("data-count")) >= 1
    open_tab.click()
    expect(page).to_have_url(f"{base}/episodes?worklist=open_tasks")
    expect(page.get_by_test_id("worklist-help")).to_contain_text("isn't done")
    expect(episode_row(page, episode["name"])).to_contain_text("1 open task")
    # The tab's count is the list's length.
    expect(page.locator("tbody tr")).to_have_count(int(open_tab.get_attribute("data-count")))
    page.get_by_test_id("worklist-anaesthetic_review").click()
    expect(episode_row(page, episode["name"])).to_be_visible()
    page.get_by_test_id("worklist-on_hold").click()
    expect(episode_row(page, episode["name"])).to_have_count(0)
    page.go_back()
    expect(page).to_have_url(f"{base}/episodes?worklist=anaesthetic_review")
    page.reload()
    expect(page.get_by_role("tab", name="Anaesthetic review needed")).to_have_attribute("aria-selected", "true")
    expect(episode_row(page, episode["name"])).to_be_visible()

    # Complete it on the episode: who and when are shown, and it leaves Open tasks.
    page.get_by_test_id("worklist-open_tasks").click()
    episode_row(page, episode["name"]).click()
    tasks.get_by_role("button", name="Complete: Made-up: book anaesthetic review").click()
    expect(task).to_contain_text("Completed by Cara Clinician")
    expect(task).to_have_attribute("data-done", "true")
    page.go_back()
    expect(page).to_have_url(f"{base}/episodes?worklist=open_tasks")
    expect(page.get_by_test_id("worklist-open_tasks")).to_be_visible()
    expect(episode_row(page, episode["name"])).to_have_count(0)

    # Re-opened, it comes back.
    page.goto(f"{base}/episodes/{episode['id']}")
    tasks.get_by_role("button", name="Re-open: Made-up: book anaesthetic review").click()
    expect(task).to_contain_text("Re-opened by Cara Clinician")
    page.goto(f"{base}/episodes?worklist=open_tasks")
    expect(episode_row(page, episode["name"])).to_be_visible()


def test_ready_for_admission_email_uses_the_hospitals_wording(page: Page, as_user, patient_link):
    ada = as_user("Ada Admin")
    hospital_id = ada.base.split("/h/")[1]
    hq = as_user("Alex Author").published_hq("workflow-hq.json", f"E2E Message {unique()}")
    cara = as_user("Cara Clinician")
    episode = cara.episode(hq, f"Message {unique()}")
    patient = patient_link(episode)
    for set_name, answers in PATIENT_ANSWERS.items():
        patient.answer(set_name, answers)
    patient.submit()
    for set_name, answers in {**PATIENT_ANSWERS, "Nurse checks": {"q_bp": "120/80"}}.items():
        cara.validate(episode, set_name, answers)
    cara.complete_review(episode)

    switch = {"statusMessages.readyForAdmissionEmail": True,
              "templates.readyForAdmissionEmail": "Dear {{firstName}}, {{hospital}} is ready for you. (Made-up wording.)"}
    ada.call("PATCH", ada.o("/settings"), {"hospitalId": hospital_id, "values": switch})
    try:
        base = sign_in(page.context, "Cara Clinician")
        page.goto(f"{base}/episodes/{episode['id']}")
        change_status(page, "Ready for admission")
        expect(page.get_by_text("Ready for admission message sent by email.")).to_be_visible()
        email = captured(episode["email"])[0]
        assert email["body"] == "Dear Sam, Hospital A1 is ready for you. (Made-up wording.)", email
        assert "/i/" not in email["body"] and "/s/" not in email["body"]
    finally:
        ada.call("PATCH", ada.o("/settings"), {"hospitalId": hospital_id, "values": {k: None for k in switch}})
