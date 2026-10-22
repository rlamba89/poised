# S23 test report 2 (retest after verification 1)

**Blocked: 0 cases run.** The browser available to the tester can't reach the local app.

**Environment:**
- Commit `05a1a5b` on master, with S23 and the verification 1 fixes uncommitted in the working tree.
- 22 Oct 2026, about 09:45 BST.
- App up locally: `curl localhost:8080/api/health` → `{"status":"ok"}`; `curl -I localhost:3000` → `200 OK` (Vite listens on IPv6 localhost only; the API on all addresses).

## What happened

1. `select_browser` with the requested deviceId `bac77f71-5d24-4f96-8f5d-3886dc0e71d6` failed: "No connected browser has deviceId …".
2. The only connected browser is "Browser 1" (macOS) with a **different** deviceId, `8a16a801-7d5b-445d-bedc-ebece06a702f`.
3. In that browser, `https://example.com` loads, but `http://localhost:3000/`, `http://localhost:3000/login`, `http://[::1]:3000/` and `http://localhost:8080/api/health` all show Chrome's error page (in a new tab too, after waiting).
4. The Chrome running on this Mac (checked with AppleScript, listing its tabs' addresses) had no example.com tab. So the connected "Browser 1" is **not the Chrome on this computer**. It's another machine or install, which can't reach this Mac's localhost.

No cases could be run through the UI, so none were run (the tester doesn't switch to API-only testing without Rahul's OK). The test tabs were closed.

## To unblock

Connect Claude in Chrome from the Chrome on this Mac (the one with deviceId `bac77f71-…`, or any Chrome that can open http://localhost:3000), then rerun the retest: T-01 to T-17 and R-01 to R-05 with a new run number. The hand checks from test report 1 are still open: R-02 at phone size and R-05.

## Results

| Case | Result | Note |
| --- | --- | --- |
| T-01 to T-17 | Blocked | No browser that can reach localhost. |
| R-01 to R-05 | Blocked | Same. |
