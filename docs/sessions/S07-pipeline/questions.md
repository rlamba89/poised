# S07: questions for Rahul (builder, automatic run, 14 Oct 2026)

The pipeline is built and deployed (`poised-pipeline`, CodePipeline `poised`, eu-west-2), and the second-region proof is done and destroyed. **Two things stop the pipeline from running, and they need you.** Everything else is in [build-notes.md](build-notes.md).

## 1. Approve the GitHub connection (agreed in answers.md item 4)

CDK created the connection **`poised-github`** (ARN `arn:aws:codeconnections:eu-west-2:026090536258:connection/88e04dda-62e7-4ccd-8af8-a329008eec7f`). It is **Pending**, so the pipeline's first run failed at Source, as expected.

1. Open the AWS console signed in to account **026090536258**, region **Europe (London) eu-west-2**: https://eu-west-2.console.aws.amazon.com/codesuite/settings/connections?region=eu-west-2
   (or: **Developer Tools → Settings → Connections**).
2. Click **`poised-github`** (status **Pending**), then **Update pending connection**.
3. A GitHub window opens. Sign in to GitHub as **rlamba89** if asked.
4. Under **GitHub Apps**, click **Install a new app** (or pick an existing **AWS Connector for GitHub** installation if one is listed). On GitHub choose **Only select repositories → `rlamba89/poised`**, then **Install** (or **Save**).
5. Back in the AWS window, click **Connect**. The status becomes **Available**.

Don't click "Release change" yet: GitHub's `master` doesn't have the pipeline's code (question 2), so a run would fail at UpdatePipeline.

## 2. Getting S07's code onto GitHub, so the pipeline can run it

The pipeline builds only what's on GitHub's `master`. GitHub's `master` is **41 commits behind** the local `master` (S04 to S06), and S07's own code (`infra/pipeline_stack.py`, the new `app.py`, the Makefile, the seed test) is uncommitted, because commits are the verifier's. So nothing in the brief's "Done when" (a push goes green through QA e2e; a failing test stops the run) can be shown until S07's code is committed and pushed.

**Pushing needs you:** my `git push origin master` (of the 41 commits already approved for publishing) was refused by Claude Code's permission system in this run, so I stopped there.

**Options:**

- **A (recommended): commit S07 now, you push, then the pipeline is tested for real; verification after.**
  - The conductor commits S07's work on `master` as it stands ("S07: the pipeline, the seed run-twice test and the runbook (pre-verification)"), and you push `master`.
  - The builder watches the run and fixes what the real cloud build shows (likely small things, such as the build image's tools); each fix is another commit, pushed the same way.
  - Then the brief's two test pushes: **a trivial visible change** (one line of text on the sign-in page, kept) and **a deliberately failing Go unit test, reverted at once** (plan F4: "or revert at once"; it leaves two commits in the history).
  - The verifier then reviews S07 as a commit range rather than a working-tree diff.
  - **Why:** the pipeline can't be exercised any other way, so verifying before it has run once would sign off untested code.
- **B: a short-lived branch.** As A, but on a branch `s07-pipeline`: the pipeline follows it while it's brought up, then goes back to `master`, and the branch is merged after verification. Keeps `master` to verified commits, at the cost of a second deploy of the pipeline and a merge.
- **C: verify first on local evidence, then commit and push; the cloud runs become a second test round.** Keeps the usual order, but the verifier signs off a pipeline that has never run, and any fix it then needs goes round the loop again.

Reply in `answers.md` (e.g. "2: A") and say when the connection is approved and `master` is pushed.
