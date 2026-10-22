"""The trust's procedure list (S19; PRC-01, EPI-01): a whole-trust super clinician adds, edits and
retires procedures on the Procedures page; a clinician picking one on a new episode gets its default
HQ, and can still choose another. A hospital-only procedure is offered at that hospital alone, and
a retired one at none. A super clinician of one hospital only doesn't manage the list.

Each run adds procedures with unique names (they're never deleted, only retired, so they stay in
the local database, retired, like the journeys' published questionnaires)."""

import re

from playwright.sync_api import Browser, Page, expect

from support import PATIENT_ANSWERS, pick, sign_in, unique


def open_procedures(page: Page, base: str) -> None:
    page.goto(f"{base}/episodes")
    page.get_by_role("banner").get_by_role("link", name="Procedures").click()
    expect(page.get_by_role("heading", name="Procedures", exact=True)).to_be_visible()


def add_procedure(page: Page, name: str, code: str = "", specialty: str = "", offered_at: str | None = None, default_hq: str | None = None) -> None:
    page.get_by_role("button", name="Add procedure").click()
    form = page.get_by_test_id("procedure-form")
    form.get_by_label("Name").fill(name)
    form.get_by_label("Code").fill(code)
    form.get_by_label("Specialty").fill(specialty)
    if offered_at:
        pick(page, "Offered at", offered_at)
    if default_hq:
        pick(page, "Default HQ", default_hq)
    form.get_by_role("button", name="Add procedure").click()
    expect(page.get_by_text(f"{name} is added.")).to_be_visible()


def test_a_super_clinician_manages_the_list_and_a_clinician_picks_from_it(page: Page, browser: Browser, base_url, as_user):
    hq = as_user("Alex Author").published_hq("workflow-hq.json", f"E2E Procedures {unique()}")
    hip, eye = f"E2E Hip {unique()}", f"E2E Eye {unique()}"
    # React's warnings (e.g. invalid HTML) are console errors; the duplicate's 409 is expected noise.
    errors: list[str] = []
    page.on("console", lambda m: m.type == "error" and "Failed to load resource" not in m.text and errors.append(m.text))

    # Alex Author (a whole-trust super clinician) adds two: one for every hospital with a code and
    # a default HQ, and one for Hospital A1 only with neither.
    base = sign_in(page.context, "Alex Author")
    open_procedures(page, base)
    add_procedure(page, hip, code="E2E-1", specialty="Orthopaedics", default_hq=hq["name"])
    add_procedure(page, eye, offered_at="Hospital A1 only")
    row = page.get_by_test_id(f"procedure-{hip}")
    expect(row).to_contain_text("E2E-1")
    expect(row).to_contain_text("Every hospital")
    expect(row).to_contain_text(hq["name"])
    row = page.get_by_test_id(f"procedure-{eye}")
    expect(row).to_contain_text("Hospital A1")
    expect(row).to_contain_text("The hospital's default")

    # A second one with the same name, in another case, is refused with a clear message.
    page.get_by_role("button", name="Add procedure").click()
    form = page.get_by_test_id("procedure-form")
    form.get_by_label("Name").fill(hip.upper())
    form.get_by_label("Name").press("Enter")  # Enter sends the form (S19 test report 1, exploring 2)
    expect(form.get_by_text("The trust already has a procedure with this name")).to_be_visible()
    # An empty name gets the app's own message, which replaces the last one (S19 test report 2).
    form.get_by_label("Name").fill("")
    form.get_by_label("Code").press("Enter")
    expect(form.get_by_text("Enter the procedure's name.")).to_be_visible()
    expect(form.get_by_text("The trust already has a procedure with this name")).to_have_count(0)
    form.get_by_role("button", name="Cancel").click()

    # She edits one, from below the fold on a short window: the form comes into view (T-04), and
    # Enter saves it.
    page.set_viewport_size({"width": 1440, "height": 300})
    row = page.get_by_test_id(f"procedure-{hip}")
    row.scroll_into_view_if_needed()
    assert page.evaluate("window.scrollY") > 0, "the row should be below the fold for this check"
    row.get_by_role("button", name="Edit").click()
    form = page.get_by_test_id("procedure-form")
    expect(form.get_by_label("Name")).to_be_in_viewport()
    page.set_viewport_size({"width": 1440, "height": 900})
    expect(form.get_by_label("Name")).to_have_value(hip)
    form.get_by_label("Specialty").fill("Trauma and orthopaedics")
    form.get_by_label("Specialty").press("Enter")
    expect(page.get_by_test_id(f"procedure-{hip}")).to_contain_text("Trauma and orthopaedics")
    # Reload: it was saved.
    page.reload()
    expect(page.get_by_test_id(f"procedure-{hip}")).to_contain_text("Trauma and orthopaedics")

    # Cara Clinician (Hospital A1): picking the procedure chooses its default HQ; she can change it.
    cara = browser.new_context(viewport={"width": 1440, "height": 900}, base_url=base_url)
    try:
        p = cara.new_page()
        p.goto(f"{sign_in(cara, 'Cara Clinician')}/episodes/new")
        pick(p, "Procedure", f"{hip} (E2E-1)")
        expect(p.get_by_role("combobox", name="Health questionnaire")).to_have_value(f"{hq['name']} v1")
        # Any published version of the seeded HQ: a tester may have published a new one (S23 R-05).
        p.get_by_role("combobox", name="Health questionnaire").click()
        p.get_by_role("option", name=re.compile(r"^Demo pre-op HQ v\d+$")).click()
        expect(p.get_by_role("combobox", name="Procedure")).to_have_value(f"{hip} (E2E-1)")
        # A1's own procedure is offered here.
        p.get_by_role("combobox", name="Procedure").fill(eye)
        expect(p.get_by_role("option", name=eye, exact=True)).to_be_visible()
    finally:
        cara.close()

    # Val Clinician (Hospital A2): A1's own isn't offered.
    val = browser.new_context(viewport={"width": 1440, "height": 900}, base_url=base_url)
    try:
        p = val.new_page()
        p.goto(f"{sign_in(val, 'Val Clinician', 'Hospital A2')}/episodes/new")
        p.get_by_role("combobox", name="Procedure").fill(eye)
        expect(p.get_by_text("No procedure of that name")).to_be_visible()
        p.get_by_role("combobox", name="Procedure").fill(hip)
        expect(p.get_by_role("option", name=f"{hip} (E2E-1)", exact=True)).to_be_visible()
    finally:
        val.close()

    # Alex retires it: it's marked Retired, and isn't offered for new episodes. On a short window
    # the row is below the fold, and the message at the top must come into view (S19 test report 3).
    page.set_viewport_size({"width": 1440, "height": 300})
    row = page.get_by_test_id(f"procedure-{hip}")
    row.scroll_into_view_if_needed()
    assert page.evaluate("window.scrollY") > 0, "the row should be below the fold for this check"
    row.get_by_role("button", name="Retire").click()
    expect(page.get_by_test_id(f"procedure-{hip}")).to_contain_text("Retired")
    expect(page.get_by_text(f"{hip} is retired")).to_be_in_viewport()
    page.wait_for_timeout(1000)  # nothing scrolls it away once the list has reloaded
    expect(page.get_by_text(f"{hip} is retired")).to_be_in_viewport(ratio=1)
    assert page.evaluate("window.scrollY") == 0  # at the top: not hidden under the fixed header
    # Bring back, from the bottom of the list, then retire again.
    page.get_by_test_id(f"procedure-{hip}").get_by_role("button", name="Bring back").click()
    expect(page.get_by_text(f"{hip} is offered again.")).to_be_in_viewport()
    page.get_by_test_id(f"procedure-{hip}").get_by_role("button", name="Retire").click()
    expect(page.get_by_text(f"{hip} is retired")).to_be_in_viewport()
    page.set_viewport_size({"width": 1440, "height": 900})
    page.goto(f"{base}/episodes/new")
    page.get_by_role("combobox", name="Procedure").fill(hip)
    expect(page.get_by_text("No procedure of that name")).to_be_visible()
    open_procedures(page, base)
    page.get_by_test_id(f"procedure-{eye}").get_by_role("button", name="Retire").click()
    expect(page.get_by_test_id(f"procedure-{eye}")).to_contain_text("Retired")
    assert not errors, errors


def test_a_super_clinician_of_one_hospital_doesnt_manage_the_list(page: Page):
    base = sign_in(page.context, "Sue Super")
    page.goto(f"{base}/episodes")
    expect(page.get_by_role("banner").get_by_role("link", name="Episodes")).to_be_visible()
    expect(page.get_by_role("banner").get_by_role("link", name="Procedures")).to_have_count(0)
    page.goto(f"{base}/procedures")  # a deep link
    expect(page.get_by_text(re.compile("Only super clinicians and admins of the whole trust"))).to_be_visible()


def test_a_clinician_edits_the_procedure_and_dates_until_the_review_is_complete(page: Page, as_user, patient_link):
    hq = as_user("Alex Author").published_hq("workflow-hq.json", f"E2E Dates {unique()}")
    cara = as_user("Cara Clinician")
    episode = cara.episode(hq, f"Dates {unique()}")  # Knee arthroscopy, no dates
    base = sign_in(page.context, "Cara Clinician")
    page.goto(f"{base}/episodes/{episode['id']}")
    expect(page.get_by_text("Knee arthroscopy", exact=True)).to_be_visible()

    page.get_by_role("button", name="Edit procedure and dates").click()
    form = page.get_by_test_id("edit-procedure")
    pick(page, "Procedure", "Total hip replacement (HIP-01)")
    form.get_by_label("Planned procedure date").fill("2026-11-03")
    form.get_by_label("POA appointment date").fill("2026-10-28")
    form.get_by_label("POA appointment date").press("Enter")  # Enter sends it too (S19 test report 2)
    expect(form).to_have_count(0)
    expect(page.get_by_text("Total hip replacement", exact=True)).to_be_visible()
    expect(page.get_by_text("03/11/2026", exact=True)).to_be_visible()  # the hospital's date format (en-GB here)
    expect(page.get_by_text("Procedure set to Total hip replacement", exact=True)).to_be_visible()  # General notes
    expect(page.get_by_text("Planned procedure date set to 2026-11-03", exact=True)).to_be_visible()

    # The list and the POA Summary show them too.
    page.goto(f"{base}/episodes")
    row = page.get_by_role("row", name=re.compile(episode["name"]))
    expect(row).to_contain_text("Total hip replacement")
    expect(row).to_contain_text("03/11/2026")
    expect(row).to_contain_text("28/10/2026")
    page.goto(f"{base}/episodes/{episode['id']}/poa")
    expect(page.get_by_text("28/10/2026", exact=True)).to_be_visible()

    # Once the review is complete, they can't be changed: no Edit, and the API refuses.
    patient = patient_link(episode)
    for set_name, answers in PATIENT_ANSWERS.items():
        patient.answer(set_name, answers)
    patient.submit()
    for set_name, answers in {**PATIENT_ANSWERS, "Nurse checks": {"q_bp": "120/80"}}.items():
        cara.validate(episode, set_name, answers)
    cara.complete_review(episode)
    page.goto(f"{base}/episodes/{episode['id']}")
    expect(page.get_by_text("HQ review completed by Cara Clinician", exact=True)).to_be_visible()  # General notes
    expect(page.get_by_role("button", name="Edit procedure and dates")).to_have_count(0)
    expect(page.get_by_text("Total hip replacement", exact=True)).to_be_visible()
    revision = cara.call("GET", cara.h(f"/episodes/{episode['id']}"))["episode"]["revision"]  # S20: the page's revision
    res = cara.request.fetch(f"/api{cara.h(f'/episodes/{episode['id']}')}", method="PATCH", data={"poaDate": "2026-12-01", "revision": revision})
    assert res.status == 409, res.status


def test_a_partly_typed_date_is_refused_not_dropped(page: Page, as_user):
    """S19 test report 1, F1: the browser gives a half-typed date field the value "", so the page
    sent it as no date and the episode was created without it. It must say so instead."""
    hq = as_user("Alex Author").published_hq("workflow-hq.json", f"E2E Partial {unique()}")
    cara = as_user("Cara Clinician")
    episode = cara.episode(hq, f"Partial {unique()}")
    base = sign_in(page.context, "Cara Clinician")
    before = len(cara.call("GET", cara.h("/episodes")))

    # New episode for the same patient: the POA date typed as day and month only.
    page.goto(f"{base}/episodes/new?patient={episode['patientId']}")
    pick(page, "Health questionnaire", f"{hq['name']} v1")
    page.get_by_label("POA appointment date").click()
    page.keyboard.type("2810")
    page.get_by_role("button", name="Create episode").click()
    expect(page.get_by_text("Enter the POA appointment date as a date.")).to_be_visible()
    expect(page).to_have_url(re.compile(r"/episodes/new"))
    assert len(cara.call("GET", cara.h("/episodes"))) == before

    # Edit procedure and dates: a half-typed planned date isn't saved as none.
    page.goto(f"{base}/episodes/{episode['id']}")
    page.get_by_role("button", name="Edit procedure and dates").click()
    form = page.get_by_test_id("edit-procedure")
    form.get_by_label("Planned procedure date").click()
    page.keyboard.type("31")
    form.get_by_role("button", name="Save").click()
    expect(form.get_by_text("Enter the planned procedure date as a date.")).to_be_visible()
    expect(form).to_be_visible()  # still open
