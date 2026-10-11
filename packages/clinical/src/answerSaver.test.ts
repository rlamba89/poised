import { describe, expect, it } from "vitest";
import { answerSaver } from "./answerSaver";

/** A fake server: one answers row, which takes a save only from its current revision (S02a). */
function server(revision = 0) {
  const row = { revision, data: "" };
  const sent: { body: string; revision: number }[] = [];
  const send = async (body: string, from: number) => {
    sent.push({ body, revision: from });
    await Promise.resolve(); // a save takes a moment, so callers can overlap
    if (from !== row.revision) throw Object.assign(new Error("These answers were changed somewhere else."), { status: 409 });
    row.revision++;
    row.data = body;
    return { revision: row.revision };
  };
  return { row, sent, send };
}

describe("answerSaver", () => {
  it("sends each save with the revision the last one returned", async () => {
    const s = server(3);
    const saver = answerSaver(3, s.send);
    expect(await saver.save("a")).toEqual({ ok: true, revision: 4 });
    expect(await saver.save("b")).toEqual({ ok: true, revision: 5 });
    expect(s.sent).toEqual([{ body: "a", revision: 3 }, { body: "b", revision: 4 }]);
    expect(saver.revision).toBe(5);
  });

  it("starts at 0 for a set with no answers yet", async () => {
    const s = server();
    expect(await answerSaver(0, s.send).save("a")).toEqual({ ok: true, revision: 1 });
  });

  it("runs saves one at a time, so a tab's own saves don't clash", async () => {
    const s = server(0);
    const saver = answerSaver(0, s.send);
    // An autosave and Validate, sent together: the second waits for the first's revision.
    const [first, second] = await Promise.all([saver.save("autosave"), saver.save("validate")]);
    expect(first).toEqual({ ok: true, revision: 1 });
    expect(second).toEqual({ ok: true, revision: 2 });
    expect(s.row.data).toBe("validate");
  });

  it("after a 409, stops: later saves aren't sent, and report the conflict", async () => {
    const s = server(5); // another tab saved since this one loaded revision 4
    const saver = answerSaver(4, s.send);
    expect(await saver.save("stale")).toEqual({ ok: false, conflict: true, message: "These answers were changed somewhere else." });
    expect(saver.conflict).toBe("These answers were changed somewhere else.");
    expect(await saver.save("later")).toEqual({ ok: false, conflict: true, message: "These answers were changed somewhere else." });
    expect(s.sent).toHaveLength(1);
    expect(s.row).toEqual({ revision: 5, data: "" });
  });

  it("any other failure is not a conflict, and the next save is still sent", async () => {
    let fail = true;
    const saver = answerSaver(0, async (_: string, from: number) => {
      if (fail) throw Object.assign(new Error("Something went wrong. Please try again."), { status: 500 });
      return { revision: from + 1 };
    });
    expect(await saver.save("a")).toEqual({ ok: false, conflict: false, message: "Something went wrong. Please try again." });
    expect(saver.conflict).toBeNull();
    fail = false;
    expect(await saver.save("a")).toEqual({ ok: true, revision: 1 });
  });

  // S02a verify.md finding 1: a page that compares with its own copy of the answers thinks a
  // failed save went through, and never sends it again. The baseline is what the server accepted.
  it("confirmed starts as the loaded answers, and changes only when the server accepts a save", async () => {
    let up = false;
    const saver = answerSaver<string>(2, async (_: string, from: number) => {
      if (!up) throw new TypeError("Failed to fetch");
      return { revision: from + 1 };
    }, "loaded");
    expect(saver.confirmed).toBe("loaded");
    await saver.save("typed while offline");
    expect(saver.confirmed).toBe("loaded");
    up = true;
    await saver.save("typed while offline");
    expect(saver.confirmed).toBe("typed while offline");
  });

  it("a refused save doesn't change what's confirmed", async () => {
    const s = server(5);
    const saver = answerSaver<string>(4, s.send, "loaded");
    await saver.save("stale");
    expect(saver.confirmed).toBe("loaded");
  });
});
