# S29 test report 1 (automatic run): BLOCKED

**Summary:** 0 passed, 0 failed, all blocked. The test round couldn't happen: the browser couldn't reach the app.

**Environment:** HEAD `91c2409`, working tree not clean (S29's uncommitted changes). Sunday 25 Oct 2026, 13:53 GMT. URL http://localhost:3000.

## What happened
- **The app is up:** `curl localhost:8080/api/health` returned `{"status":"ok"}`, and `curl -I localhost:3000` returned 200. Vite listens on `[::1]:3000` only.
- **The wrong browser was connected:** the conductor asked for device `bce1f965-a1f0-43df-b2e4-c68ee34568ec` ("rahul chrome"). `select_browser` answered that no connected browser has that ID. The only connected browser was "Browser 1" (`8a16a801-7d5b-445d-bedc-ebece06a702f`, macOS).
- **That browser can't reach the app:** in it, http://localhost:3000, http://127.0.0.1:3000 and http://[::1]:3000 all showed Chrome's error page. It still wasn't reachable after a wait, and no other browser had connected by then.

No case was run, and no test data was created.

## Results
| Case | Result | Note |
|---|---|---|
| T-01 to T-11, R-01 to R-06 | Blocked | The "rahul chrome" profile (`bce1f965-…`) wasn't connected, and the connected browser can't open localhost. |

## Found while exploring
Nothing: nothing could be explored.

## Console and network errors
None to report. No page loaded.

## GIFs
None recorded.
