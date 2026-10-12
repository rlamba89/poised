"""J3 Patient: the clinician creates an episode and copies the link. The patient opens it on a phone,
fills every Question Set (leaving part-way and coming back keeps the answers) and sends it. The
episode is then Ready for review."""

from playwright.sync_api import BrowserContext, Page, expect

from support import choose, pick, sign_in, unique


def test_patient_fills_in_and_sends_their_hq_on_a_phone(page: Page, phone: BrowserContext, as_user):
    hq = as_user("Alex Author").published_hq("workflow-hq.json", f"E2E J3 {unique()}")
    last_name = f"J3 {unique()}"

    # The clinician creates the episode, for a new (made-up) patient.
    hospital = sign_in(page.context, "Cara Clinician")
    page.context.grant_permissions(["clipboard-read", "clipboard-write"])
    page.goto(f"/h/{hospital}/episodes")
    page.get_by_role("button", name="New episode").click()
    drawer = page.get_by_role("dialog", name="New episode")
    drawer.get_by_text("New patient").click()
    drawer.get_by_label("First name").fill("Sam")
    drawer.get_by_label("Last name").fill(last_name)
    drawer.get_by_label("Date of birth").fill("1980-05-17")
    pick(page, "Sex", "Female")
    pick(page, "Health questionnaire", f"{hq['name']} v1")
    drawer.get_by_label("Procedure").fill("Knee replacement")
    drawer.get_by_role("button", name="Create episode").click()
    expect(page.get_by_role("heading", name=f"Sam {last_name}")).to_be_visible()
    expect(page.get_by_role("combobox", name="Episode status")).to_have_value("HQ not complete")
    expect(page.get_by_role("combobox", name="Episode status")).to_be_disabled()
    expect(page.get_by_text("Episode created", exact=True)).to_be_visible()  # General notes
    # Dates use the deployment's locale from GET /api/config (A-20): en-GB, make dev's default.
    expect(page.get_by_text("17 May 1980 (age")).to_be_visible()

    # …and copies the patient's link.
    page.get_by_role("button", name="Copy").click()
    expect(page.get_by_role("button", name="Copied")).to_be_visible()
    link = page.evaluate("navigator.clipboard.readText()")
    expect(page.get_by_role("textbox", name="Patient link")).to_have_value(link)

    # The patient opens it on their phone.
    patient = phone.new_page()
    patient.goto(link)
    expect(patient.get_by_role("heading", name=hq["name"])).to_be_visible()
    expect(patient.get_by_text("Hello Sam.")).to_be_visible()
    # Only the patient's two Question Sets: the clinician's "Nurse checks" never appears.
    expect(patient.get_by_text("0 of 2 sections complete")).to_be_visible()
    expect(patient.get_by_text("Nurse checks")).to_have_count(0)
    # Nothing is wider than the phone's screen (no sideways scrolling).
    assert patient.evaluate("document.documentElement.scrollWidth <= window.innerWidth")
    send = patient.get_by_role("button", name="Send my answers")
    expect(send).to_be_disabled()

    # They answer the first page of "About you", then leave part-way…
    about = patient.get_by_test_id("set-About you")
    about.get_by_role("button", name="Start").click()
    choose(patient, "Do you smoke?", "Yes")
    patient.get_by_role("button", name="All sections").click()
    expect(about.get_by_text("In progress")).to_be_visible()

    # …and come back later: the answer was kept.
    patient.reload()
    about.get_by_role("button", name="Continue").click()
    expect(patient.get_by_role("radiogroup", name="Do you smoke?").get_by_role("radio", name="Yes")).to_be_checked()
    patient.get_by_role("button", name="Next", exact=True).click()
    choose(patient, "How often do you drink alcohol?", "Sometimes")
    patient.get_by_role("button", name="Finish this section").click()
    expect(about.get_by_text("Complete")).to_be_visible()

    # The second Question Set.
    health = patient.get_by_test_id("set-Your health")
    health.get_by_role("button", name="Start").click()
    choose(patient, "Do you have diabetes?", "No")
    patient.get_by_role("button", name="Finish this section").click()
    expect(health.get_by_text("Complete")).to_be_visible()
    expect(patient.get_by_text("2 of 2 sections complete")).to_be_visible()

    # They send their answers, after confirming.
    send.click()
    confirm = patient.get_by_role("dialog", name="Send your answers?")
    confirm.get_by_role("button", name="Send my answers").click()
    expect(patient.get_by_role("heading", name="Thank you, Sam")).to_be_visible()
    patient.reload()  # opening the link again shows the thank-you page, not the form
    expect(patient.get_by_role("heading", name="Thank you, Sam")).to_be_visible()

    # The clinician sees the episode is ready for review, and the link is gone.
    page.reload()
    expect(page.get_by_role("combobox", name="Episode status")).to_have_value("Ready for review")
    expect(page.get_by_role("textbox", name="Patient link")).to_be_hidden()
