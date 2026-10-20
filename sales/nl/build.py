"""Build the Preopsis sales material.

    e2e/.venv/bin/python sales/nl/build.py

Writes, next to this file:
  preopsis-brochure-nl.html / -en.html   (page fragments, also publishable as artifacts)
  preopsis-brochure-nl.pdf  / -en.pdf    (A4, via Playwright's Chromium)
  preopsis-seller-sheet.pdf              (from src/seller-sheet.html)
  preview/<name>-p<N>.png                (one image per A4 page, for checking layout)

Fails if a language is missing a key, or if any page's content overflows its A4 sheet.
"""

import sys
from pathlib import Path
from string import Template

from playwright.sync_api import sync_playwright

HERE = Path(__file__).resolve().parent
sys.path.insert(0, str(HERE / "src"))
from copy_text import COPY  # noqa: E402

PREVIEW = HERE / "preview"


def fill(lang: str) -> str:
    missing = set(COPY["nl"]) ^ set(COPY[lang])
    if missing:
        raise SystemExit(f"{lang}: keys differ from nl: {sorted(missing)}")
    values = {
        k: "".join(f"<li>{x}</li>" for x in v) if isinstance(v, list) else v
        for k, v in COPY[lang].items()
    }
    values["lang"] = lang
    template = Template((HERE / "src" / "brochure.template.html").read_text())
    return template.substitute(values)


def wrap(fragment: str, lang: str) -> str:
    # The same skeleton the artifact host adds, so the PDF and the preview render alike.
    return (
        f'<!doctype html><html lang="{lang}"><head><meta charset="utf-8">'
        '<meta name="viewport" content="width=device-width, initial-scale=1, viewport-fit=cover">'
        f"</head><body>{fragment}</body></html>"
    )


def render(browser, fragment: str, lang: str, name: str) -> None:
    page = browser.new_page(viewport={"width": 1000, "height": 1400}, device_scale_factor=1.6)
    page.set_content(wrap(fragment, lang), wait_until="networkidle")
    page.evaluate("document.fonts.ready")
    fonts = page.evaluate(
        """[['650','Bricolage Grotesque'],['400','Figtree'],['500','IBM Plex Mono']]
             .map(([w, f]) => [f, document.fonts.check(`${w} 12px "${f}"`)])"""
    )
    for family, ok in fonts:
        if not ok:
            print(f"  warning: font {family} did not load (fallback used)")

    overflow = page.evaluate(
        """[...document.querySelectorAll('.page')].map((p, i) => {
             const bad = [...p.querySelectorAll('*')].filter(el => {
               const r = el.getBoundingClientRect(), b = p.getBoundingClientRect();
               return r.height > 0 && (r.bottom > b.bottom + 1 || r.right > b.right + 1);
             }).map(el => el.tagName + '.' + el.className).slice(0, 5);
             return {page: i + 1, over: p.scrollHeight - p.clientHeight, bad};
           })"""
    )
    problems = [o for o in overflow if o["over"] > 0 or o["bad"]]

    PREVIEW.mkdir(exist_ok=True)
    for i, el in enumerate(page.query_selector_all(".page"), start=1):
        el.screenshot(path=str(PREVIEW / f"{name}-p{i}.png"))

    page.emulate_media(media="print")
    page.pdf(path=str(HERE / f"{name}.pdf"), format="A4", print_background=True,
             prefer_css_page_size=True)
    print(f"  {name}.pdf  sheets={len(overflow)}")
    page.close()
    if problems:
        for p in problems:
            print(f"  OVERFLOW page {p['page']}: {p['over']}px, elements {p['bad']}")
        raise SystemExit(f"{name}: content overflows its page")


def main() -> None:
    with sync_playwright() as pw:
        browser = pw.chromium.launch()
        for lang in COPY:
            fragment = fill(lang)
            name = f"preopsis-brochure-{lang}"
            (HERE / f"{name}.html").write_text(fragment)
            render(browser, fragment, lang, name)
        seller = HERE / "src" / "seller-sheet.html"
        if seller.exists():
            render(browser, seller.read_text(), "en", "preopsis-seller-sheet")
        browser.close()


if __name__ == "__main__":
    main()
