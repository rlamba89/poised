"""The Video element (S34, INF-01/02).

An author pastes a video's link: only a YouTube or Vimeo video is taken, and Publish is refused until
the video has a good link and a length. The preview plays YouTube through youtube-nocookie.com and
Vimeo with dnt=1 (and a private link's hash). The patient, on a phone, sees the player the full width
of the screen, with nothing wider than it, and an optional tick doesn't hold them up; the clinician
sees that it wasn't ticked. The real video sites are blocked, so this runs offline: only the player's
address is checked."""

from playwright.sync_api import BrowserContext, Page, expect

from support import block_video_sites, choose, invite_path, pick, sign_in, unique


def test_pasting_a_link_and_publishing(page: Page, as_user):
    block_video_sites(page.context)
    hq = as_user("Alex Author").draft_hq("video-hq.json", f"E2E Video {unique()}")
    base = sign_in(page.context, "Alex Author")
    page.goto(f"{base}/questionnaires/{hq['id']}")
    page.get_by_role("complementary").get_by_text("Before your operation").click()
    page.get_by_role("complementary").get_by_text("Getting ready").click()
    main = page.get_by_role("main")

    # A new video: no link and no length yet, said on its card and in its settings.
    main.get_by_role("button", name="Video", exact=True).last.click()
    settings = page.get_by_role("tabpanel", name="Settings")
    card = main.locator("[data-testid^=card-q_]").filter(has_text="No link yet")
    expect(card).to_contain_text("No link yet · No length yet")
    expect(settings).to_contain_text("Paste the video's link, or choose one from the library.")
    settings.get_by_role("textbox", name="Title").fill("Your anaesthetic")
    settings.get_by_role("textbox", name="Title").press("Tab")
    card = main.locator("[data-testid^=card-q_]").filter(has_text="Your anaesthetic")

    # Any other site is refused, and so is Publish, naming the Question Set.
    settings.get_by_role("textbox", name="Link").fill("https://www.dailymotion.com/video/x7tgad0")
    settings.get_by_role("textbox", name="Link").press("Tab")
    expect(settings).to_contain_text("Only YouTube or Vimeo links can be used.")
    expect(card).to_contain_text("Not a YouTube or Vimeo link · No length yet")
    expect(page.get_by_test_id("save-status")).to_have_text("All changes saved")
    page.get_by_role("banner").get_by_role("button", name="Publish").click()
    dialog = page.get_by_role("dialog", name="Publish version 1?")
    problems = dialog.get_by_test_id("publish-problems")
    expect(problems).to_contain_text("Before your operation · The video “Your anaesthetic” has a link that isn't a YouTube or Vimeo video")
    expect(problems).to_contain_text("Before your operation · The video “Your anaesthetic” has no length")
    expect(dialog.get_by_role("button", name="Publish")).to_be_disabled()
    dialog.get_by_role("button", name="Cancel").click()

    # A youtu.be link and a length: the card shows both.
    settings.get_by_role("textbox", name="Link").fill("https://youtu.be/aqz-KE-bpKQ")
    settings.get_by_role("textbox", name="Link").press("Tab")
    expect(settings.get_by_test_id("video-plays")).to_have_text("Plays from YouTube (privacy-enhanced mode)")
    settings.get_by_role("textbox", name="Length").fill("3:2")
    settings.get_by_role("textbox", name="Length").press("Tab")
    expect(settings).to_contain_text("Type the length as minutes and seconds")
    settings.get_by_role("textbox", name="Length").fill("3:20")
    settings.get_by_role("textbox", name="Length").press("Tab")
    expect(card).to_contain_text("YouTube · Length 3:20")
    expect(page.get_by_test_id("save-status")).to_have_text("All changes saved")

    # The preview plays it through youtube-nocookie.com; the fixture's Vimeo video with dnt=1 and its hash.
    page.get_by_role("link", name="Preview").click()
    expect(page.locator("iframe[title='Your anaesthetic']")).to_have_attribute("src", "https://www.youtube-nocookie.com/embed/aqz-KE-bpKQ")
    expect(page.locator("iframe[title='Fasting before your operation']")).to_have_attribute(
        "src", "https://player.vimeo.com/video/76979871?dnt=1&h=8272103f6e")
    # The video's title and tick are translated with the rest of the HQ (the fixture's German).
    pick(page, "Language", "German")
    expect(page.get_by_text("Nüchtern vor der Operation")).to_be_visible()
    expect(page.get_by_role("checkbox", name="Ich habe dieses Video gesehen")).to_be_visible()
    page.go_back()

    # Now it publishes.
    page.get_by_role("banner").get_by_role("button", name="Publish").click()
    page.get_by_role("dialog", name="Publish version 1?").get_by_role("button", name="Publish").click()
    expect(page.get_by_role("banner").get_by_text("v1 · published")).to_be_visible()


def test_the_patient_watches_on_a_phone_and_an_optional_tick_doesnt_hold_them_up(page: Page, phone: BrowserContext, as_user):
    hq = as_user("Alex Author").published_hq("video-hq.json", f"E2E Video {unique()}")
    episode = as_user("Cara Clinician").episode(hq, f"Video {unique()}")
    block_video_sites(phone)
    patient = phone.new_page()
    patient.goto(invite_path(episode["email"]))
    patient.get_by_role("button", name="Continue").click()
    patient.get_by_label("Day").fill("17")
    patient.get_by_label("Month").fill("5")
    patient.get_by_label("Year").fill("1980")
    patient.get_by_role("button", name="Continue").click()
    patient.get_by_test_id("set-Before your operation").get_by_role("button", name="Start").click()

    player = patient.locator("iframe[title='Fasting before your operation']")
    expect(player).to_have_attribute("src", "https://player.vimeo.com/video/76979871?dnt=1&h=8272103f6e")
    expect(patient.get_by_text("Length 3:20")).to_be_visible()
    # The player fits the phone's screen, and nothing on the page is wider than it.
    box = player.bounding_box()
    width = patient.viewport_size["width"]
    assert box and box["x"] >= 0 and box["x"] + box["width"] <= width, f"the player {box} is wider than the screen ({width})"
    assert box["height"] > box["width"] / 2, f"the player {box} isn't about 16:9"
    assert patient.evaluate("document.documentElement.scrollWidth") <= width, "the page scrolls sideways"

    choose(patient, "Do you smoke?", "No")
    expect(patient.get_by_role("checkbox", name="I have watched this video")).not_to_be_checked()
    patient.get_by_role("button", name="Finish this section").click()
    patient.get_by_role("button", name="Send my answers").click()
    patient.get_by_role("dialog", name="Send your answers?").get_by_role("button", name="Send my answers").click()
    expect(patient.get_by_role("heading", name="Thank you, Sam")).to_be_visible()

    # The clinician sees the video, and that it wasn't ticked.
    block_video_sites(page.context)
    base = sign_in(page.context, "Cara Clinician")
    page.goto(f"{base}/episodes/{episode['id']}")
    page.get_by_role("row", name="Before your operation").get_by_role("link", name="Validate").click()
    expect(page.locator("iframe[title='Fasting before your operation']")).to_be_visible()
    expect(page.get_by_role("checkbox", name="I have watched this video")).not_to_be_checked()
    expect(page.get_by_role("checkbox", name="I have watched this video")).to_be_disabled()
    expect(page.get_by_test_id("video-ticked-q_fasting")).to_have_text("The patient hasn't ticked this.")
