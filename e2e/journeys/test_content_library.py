"""The content library (S33; INF-05, PX-13, A-6): the platform admin adds Preopsis's videos (YouTube
or Vimeo links only) and retires them; a trust admin adds the trust's own video and leaflet, which
another trust never sees, while both trusts see Preopsis's; the checklist ticks a topic once an item
has its tag; an author reads the library but can't change it, and a clinician has no Content
library at all. The platform admin retires a starter question, and Insert from library stops
offering it.

Items are never deleted, only retired, so each run's items stay in the local database, retired. The
starter question is offered again at the end, so other journeys still find it."""

from playwright.sync_api import Browser, Page, expect

from support import sign_in, unique


def open_admin_content(page: Page) -> None:
    page.goto("/admin")
    page.get_by_role("banner").get_by_role("link", name="Content library").click()
    expect(page.get_by_role("heading", name="Content library", exact=True)).to_be_visible()


def open_trust_content(page: Page, base: str) -> None:
    page.goto(f"{base}/episodes")
    page.get_by_role("banner").get_by_role("link", name="Content library").click()
    expect(page.get_by_role("heading", name="Content library", exact=True)).to_be_visible()


def add_item(page: Page, kind: str, title: str, url: str, length: str = "", tags: tuple[str, ...] = (), ready_made: tuple[str, ...] = ()) -> None:
    page.get_by_role("button", name=f"Add {kind}").first.click()
    form = page.get_by_test_id("content-form")
    form.get_by_label("Title").fill(title)
    form.get_by_label("YouTube or Vimeo link" if kind == "video" else "Link to the PDF").fill(url)
    if length:
        form.get_by_label("Length").fill(length)
    for tag in tags:
        form.get_by_label("Tags").fill(tag)
        form.get_by_label("Tags").press("Enter")
    for tag in ready_made:
        form.get_by_role("button", name=f"+ {tag}").click()
    form.get_by_role("button", name=f"Add {kind}").click()


def test_platform_and_trust_libraries(page: Page, browser: Browser, base_url, as_user):
    errors: list[str] = []
    page.on("console", lambda m: m.type == "error" and "Failed to load resource" not in m.text and errors.append(m.text))
    platform_video, trust_video, trust_leaflet = f"E2E platform video {unique()}", f"E2E trust video {unique()}", f"E2E leaflet {unique()}"

    # Sam Superadmin: a link that isn't YouTube or Vimeo is refused in plain words; a YouTube one is added.
    sign_in(page.context, "Sam Superadmin", None)
    open_admin_content(page)
    add_item(page, "video", platform_video, "https://www.dailymotion.com/video/x7tgad0", "3:20", tags=("Knee",))
    form = page.get_by_test_id("content-form")
    expect(form.get_by_text("Only YouTube or Vimeo links can be used for a video")).to_be_visible()
    form.get_by_label("YouTube or Vimeo link").fill("https://youtu.be/aqz-KE-bpKQ")
    form.get_by_role("button", name="Add video").click()
    expect(page.get_by_text(f"{platform_video} is added.")).to_be_visible()
    row = page.get_by_test_id(f"content-{platform_video}")
    expect(row).to_contain_text("3:20")
    expect(row).to_contain_text("Knee")
    # The seeded examples are there too.
    expect(page.get_by_test_id("content-Example: Your anaesthetic explained")).to_be_visible()

    # Ada Admin (Trust A) sees it read-only, adds a video and a leaflet with a ready-made tag, and
    # the checklist ticks that topic.
    ada = browser.new_context(viewport={"width": 1440, "height": 900}, base_url=base_url)
    ada_page = ada.new_page()
    base = sign_in(ada, "Ada Admin")
    open_trust_content(ada_page, base)
    expect(ada_page.get_by_role("table", name="From Preopsis").get_by_text(platform_video)).to_be_visible()
    add_item(ada_page, "video", trust_video, "https://vimeo.com/76979871", "12:05", tags=("Hip",))
    expect(ada_page.get_by_text(f"{trust_video} is added.")).to_be_visible()
    add_item(ada_page, "leaflet", trust_leaflet, "https://www.trust-a.example/leaflets/who-to-call.pdf", ready_made=("Who to call",))
    expect(ada_page.get_by_text(f"{trust_leaflet} is added.")).to_be_visible()
    expect(ada_page.get_by_test_id("topic-Who to call")).to_have_attribute("data-covered", "true")
    expect(ada_page.get_by_role("table", name="Trust A's items").get_by_test_id(f"content-{trust_leaflet}")).to_contain_text("Leaflet")

    # She edits the video's length and tags; Enter saves.
    ada_page.get_by_test_id(f"content-{trust_video}").get_by_role("button", name="Edit").click()
    form = ada_page.get_by_test_id("content-form")
    expect(form.get_by_label("Length")).to_have_value("12:05")
    form.get_by_role("button", name="+ Recovery").click()
    form.get_by_label("Length").fill("12:30")
    form.get_by_label("Length").press("Enter")
    expect(ada_page.get_by_text(f"{trust_video} is saved.")).to_be_visible()
    expect(ada_page.get_by_test_id(f"content-{trust_video}")).to_contain_text("12:30")
    expect(ada_page.get_by_test_id(f"content-{trust_video}")).to_contain_text("Recovery")

    # Bea Author (Trust B) sees Preopsis's items, never Trust A's.
    bea = browser.new_context(viewport={"width": 1440, "height": 900}, base_url=base_url)
    bea_page = bea.new_page()
    bea_base = sign_in(bea, "Bea Author", "Hospital B1")
    open_trust_content(bea_page, bea_base)
    expect(bea_page.get_by_text(platform_video)).to_be_visible()
    expect(bea_page.get_by_text(trust_video)).to_have_count(0)
    expect(bea_page.get_by_role("button", name="Add video")).to_have_count(0)  # an author reads only

    # The author's search (S34's picker) finds the platform and the trust's own, by tag.
    alex = as_user("Alex Author")
    found = [i["title"] for i in alex.call("GET", alex.o("/content/search?tag=hip"))]
    assert trust_video in found and platform_video not in found, found
    assert trust_video not in [i["title"] for i in as_user("Bea Author", "Hospital B1").call("GET", bea_base.split("/h/")[0] + "/content/search?tag=hip")]

    # Cara Clinician has no Content library, and its address refuses her.
    cara = browser.new_context(viewport={"width": 1440, "height": 900}, base_url=base_url)
    cara_page = cara.new_page()
    cara_base = sign_in(cara, "Cara Clinician")
    cara_page.goto(f"{cara_base}/episodes")
    expect(cara_page.get_by_role("heading", name="Episodes")).to_be_visible()
    expect(cara_page.get_by_role("banner").get_by_role("link", name="Content library")).to_have_count(0)
    cara_page.goto(f"{cara_base}/content")
    expect(cara_page.get_by_text("Only super clinicians and admins of the whole trust can see the content library")).to_be_visible()

    # Retiring: Sam's video and Ada's items leave the search, and stay listed as Retired.
    page.reload()
    page.get_by_test_id(f"content-{platform_video}").get_by_role("button", name="Retire").click()
    expect(page.get_by_text(f"{platform_video} is retired")).to_be_visible()
    expect(page.get_by_test_id(f"content-{platform_video}")).to_contain_text("Retired")
    for title in (trust_video, trust_leaflet):
        ada_page.get_by_test_id(f"content-{title}").get_by_role("button", name="Retire").click()
        expect(ada_page.get_by_text(f"{title} is retired")).to_be_visible()
    found = [i["title"] for i in alex.call("GET", alex.o("/content/search"))]
    assert platform_video not in found and trust_video not in found and trust_leaflet not in found, found

    for c in (ada, bea, cara):
        c.close()
    assert not errors, errors


def test_retiring_a_starter_question(page: Page, as_user):
    sign_in(page.context, "Sam Superadmin", None)
    open_admin_content(page)
    row = page.get_by_test_id("starter-regular_medicines")
    expect(row).to_contain_text("Medicines you take regularly")
    if "Offer again" in row.inner_text():  # left retired by an earlier run that failed
        row.get_by_role("button", name="Offer again").click()
        expect(row.get_by_role("button", name="Retire")).to_be_visible()
    row.get_by_role("button", name="Retire").click()
    expect(page.get_by_text("Insert from library no longer offers it")).to_be_visible()
    expect(row).to_contain_text("Retired")
    try:
        alex = as_user("Alex Author")
        keys = [q["key"] for q in alex.call("GET", alex.o("/library-questions?q=medicines"))]
        assert "regular_medicines" not in keys, keys
    finally:
        row.get_by_role("button", name="Offer again").click()
        expect(page.get_by_text("is offered again")).to_be_visible()
    keys = [q["key"] for q in alex.call("GET", alex.o("/library-questions?q=medicines"))]
    assert "regular_medicines" in keys, keys


def cut_short(page: Page) -> list[str]:
    """Every badge and button on the page whose text is cut short (an ellipsis or clipped)."""
    return page.evaluate("""() => [...document.querySelectorAll('.mantine-Badge-root, .mantine-Badge-label, .mantine-Button-label')]
        .filter(el => el.getClientRects().length && el.scrollWidth > el.clientWidth + 1)
        .map(el => el.textContent.trim())""")


def test_at_phone_width_every_tag_badge_and_label_reads_in_full(browser: Browser, base_url, as_user):
    """T-14 (test reports 1 and 2): at 390px (and on a desktop), on both libraries' pages, every
    tag, Retired badge, checklist topic and button reads in full, the tables scroll sideways in
    their own box, and the console's top bar scrolls on its own."""
    title = f"E2E phone leaflet with a long title {unique()}"
    ada = as_user("Ada Admin")
    item = ada.call("POST", ada.o("/content"), {"kind": "leaflet", "title": title, "url": "https://www.trust-a.example/a.pdf",
                                                "tags": ["Who to call", "Measurements", "Appointment details"]})
    ada.call("PATCH", ada.o(f"/content/{item['id']}"), {"status": "retired"})  # still listed, with its Retired badge
    for width, height in ((390, 844), (768, 1024), (1440, 900)):
        context = browser.new_context(viewport={"width": width, "height": height}, base_url=base_url)
        try:
            page = context.new_page()
            base = sign_in(context, "Ada Admin")
            page.goto(f"{base}/content")
            expect(page.get_by_test_id(f"content-{title}")).to_contain_text("Measurements")
            page.get_by_role("button", name="Add video").click()  # the form's buttons too
            expect(page.get_by_test_id("content-form")).to_be_visible()
            assert cut_short(page) == [], f"{width}px, Ada: cut short: {cut_short(page)}"
            assert page.evaluate("document.documentElement.scrollWidth <= window.innerWidth + 1"), f"{width}px: Ada's page scrolls sideways"

            sign_in(context, "Sam Superadmin", None)
            page.goto("/admin/content")
            expect(page.get_by_test_id("starter-height")).to_contain_text("Measurements")
            page.get_by_role("button", name="Add leaflet").click()
            expect(page.get_by_test_id("content-form")).to_be_visible()
            assert cut_short(page) == [], f"{width}px, Sam: cut short: {cut_short(page)}"
            assert page.evaluate("document.documentElement.scrollWidth <= window.innerWidth + 1"), f"{width}px: the console scrolls sideways"
        finally:
            context.close()


def test_enter_in_an_empty_tags_box_doesnt_send_the_form(page: Page):
    """Test report 1, exploring: Enter after the last tag (the hint says "Press Enter after each")
    adds nothing and doesn't send the form; Enter in another field still does. Switching an unsent
    form to Leaflet clears a video's error."""
    title = f"E2E enter {unique()}"
    base = sign_in(page.context, "Ada Admin")
    open_trust_content(page, base)
    page.get_by_role("button", name="Add leaflet").click()
    form = page.get_by_test_id("content-form")
    form.get_by_label("Title").fill(title)
    form.get_by_label("Link to the PDF").fill("https://www.trust-a.example/b.pdf")
    tags = form.get_by_label("Tags")
    tags.fill("Hip")
    tags.press("Enter")
    tags.press("Enter")
    tags.fill("  ")
    tags.press("Enter")
    page.wait_for_timeout(500)
    # Leaving the box with only spaces typed adds no empty tag (test report 2).
    form.get_by_label("Title").click()
    expect(form.locator(".mantine-Pill-root")).to_have_count(1)
    expect(form.locator(".mantine-Pill-root")).to_have_text("Hip")
    expect(page.get_by_text(f"{title} is added.")).to_have_count(0)
    expect(form).to_be_visible()
    # A video's error goes when the form is switched to Leaflet.
    form.get_by_text("Video", exact=True).click()
    form.get_by_role("button", name="Add video").click()
    expect(form.get_by_text("Only YouTube or Vimeo links")).to_be_visible()
    form.get_by_text("Leaflet", exact=True).click()
    expect(form.get_by_text("Only YouTube or Vimeo links")).to_have_count(0)
    form.get_by_label("Title").press("Enter")
    expect(page.get_by_text(f"{title} is added.")).to_be_visible()
    page.get_by_test_id(f"content-{title}").get_by_role("button", name="Retire").click()
    expect(page.get_by_text(f"{title} is retired")).to_be_visible()


def test_a_capitalised_link_is_saved_and_an_old_error_doesnt_come_back(page: Page, as_user):
    """Verification 1: a phone keyboard's "Https://" is saved (as https://), not a server error;
    and after a failed Retire, opening Edit and pressing Cancel doesn't bring the old error back."""
    title, other = f"E2E capital {unique()}", f"E2E failed retire {unique()}"
    ada = as_user("Ada Admin")
    item = ada.call("POST", ada.o("/content"), {"kind": "leaflet", "title": other, "url": "https://www.trust-a.example/c.pdf", "tags": []})
    base = sign_in(page.context, "Ada Admin")
    open_trust_content(page, base)

    add_item(page, "leaflet", title, "Https://WWW.Trust-A.example/Leaflets/Capital.pdf")
    expect(page.get_by_text(f"{title} is added.")).to_be_visible()
    expect(page.get_by_test_id(f"content-{title}")).to_contain_text("https://www.trust-a.example/Leaflets/Capital.pdf")
    # The link field is a web address field: no capital from a phone keyboard, a keyboard with "/".
    page.get_by_role("button", name="Add video").click()
    expect(page.get_by_test_id("content-form").get_by_label("YouTube or Vimeo link")).to_have_attribute("type", "url")
    page.get_by_test_id("content-form").get_by_role("button", name="Cancel").click()

    # A Retire the server refuses shows its message…
    page.route(f"**/api/o/*/content/{item['id']}", lambda route: route.fulfill(
        status=409, content_type="application/json", body='{"error":"E2E: this retire was refused."}'))
    page.get_by_test_id(f"content-{other}").get_by_role("button", name="Retire").click()
    expect(page.get_by_text("E2E: this retire was refused.")).to_be_visible()
    page.unroute(f"**/api/o/*/content/{item['id']}")
    # …which Edit then Cancel doesn't bring back.
    page.get_by_test_id(f"content-{title}").get_by_role("button", name="Edit").click()
    expect(page.get_by_test_id("content-form")).to_be_visible()
    page.get_by_test_id("content-form").get_by_role("button", name="Cancel").click()
    expect(page.get_by_test_id("content-form")).to_have_count(0)
    expect(page.get_by_text("E2E: this retire was refused.")).to_have_count(0)
    # Clean up: both retired.
    page.get_by_test_id(f"content-{other}").get_by_role("button", name="Retire").click()
    expect(page.get_by_text(f"{other} is retired")).to_be_visible()
    page.get_by_test_id(f"content-{title}").get_by_role("button", name="Retire").click()
    expect(page.get_by_text(f"{title} is retired")).to_be_visible()
