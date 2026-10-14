# S07: answers

Asked by the conductor (`/run-sessions S06 S17`) on 14 Oct 2026, before the AWS-phase run started.

1. **The one-time "add a country" proof** (a throwaway QA copy in a second region, J1–J5, timed, then destroyed): **yes** — Rahul allowed the paid resources for it.
2. **Pushing to GitHub is allowed** in this run (the pipeline runs on a push to `main` of `rlamba89/poised`; 37 local commits will be published to the private repo).
3. **No domain yet** (see S06 answers): QA stays on the Amplify URL.
4. **The GitHub connection** (asked 14 Oct before S07): **CDK creates it** (it starts Pending), and Rahul approves it once in the console ("Update pending connection", install the AWS Connector app for `rlamba89/poised` only). That one click is the only manual step when moving to a new account.
5. **Getting S07 onto GitHub** (14 Oct, after the builder's push was refused by Claude Code's permission system): Rahul pushed `master` himself (to S06's end, `b58a952`). **The builder makes a local pre-verification commit of S07**; **Rahul runs `git push origin master` himself** each time (now and for the brief's test pushes), and approves the `poised-github` connection only after S07 is on GitHub, so the self-updating pipeline never runs from code without its own definition.
