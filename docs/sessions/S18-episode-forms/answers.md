# S18 answers

Asked by the conductor (`/run-sessions S18 S22`), 20 Oct 2026.

1. **What does an `episode_forms` row hold now?** **The plan's full set** (not the recommended minimum): id, episode_id, version_id, kind (`hq|assessment|proms`), due_at, closes_at, status, submitted_at.
2. **Existing data: migrate or reset?** **(b) Reshape the tables and re-seed** (not the recommended migration). Local databases are disposable (decisions §1). *Conductor's note:* QA on Neon holds made-up episodes too. It needs the same reset when this reaches the pipeline, which is Rahul's step at push time. The run doesn't touch AWS.
3. **Routes:** **keep today's URLs** (recommended). They resolve the episode's one HQ form on the server.
