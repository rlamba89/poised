# Preopsis brand

**The name (20 Oct 2026):** *Preopsis* is "pre-op" plus *-opsis*, Greek for "view", as in *synopsis*: a view of the patient before the operation. Say it "pree-OP-sis". Write it **Preopsis** (one word, capital P). The SMS sender ID is `PREOPSIS`, since sender IDs take capitals only.

**The mark:** an eye with a tick in it, meaning the patient is seen and checked before the operation. The wordmark shows **op** in the brand colour.

| File | Use |
|---|---|
| [preopsis-logo.svg](preopsis-logo.svg) | The logo on white or light backgrounds |
| [preopsis-logo-inverse.svg](preopsis-logo-inverse.svg) | The logo on the brand teal or a dark background |
| [preopsis-mark.svg](preopsis-mark.svg) | The mark alone: favicon, app icon, avatar (the same as `apps/web/public/icon.svg`) |

**Colours** (Mantine's `brand` colour in `apps/web/src/theme.ts`):

| | Hex | Use |
|---|---|---|
| Brand teal (shade 7) | `#0f766e` | The mark, "op", buttons and links. White text on it has a contrast ratio of about 5.5:1 (WCAG AA). |
| Dark teal (shade 9) | `#134e4a` | Hover, strong text on teal tints |
| Ink | `#12302e` | The wordmark's text |
| Tint (shade 0) | `#f0fdfa` | Light backgrounds |

**Rules:**
- Keep it clearly distinct from the NHS identity. Don't use NHS blue (`#005EB8`) or the NHS logo with ours.
- The logo SVGs set the name in text (Inter, Segoe UI or Helvetica, bold). For print or a deck, use a machine that has one of these fonts, or convert the text to outlines first.
- In the app, use `<Logo />` (`apps/web/src/features/brand/Logo.tsx`) rather than an image, so the name stays real text for screen readers and tests.
