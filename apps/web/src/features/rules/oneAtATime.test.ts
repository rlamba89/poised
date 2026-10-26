import { describe, expect, it } from "vitest";
import { oneAtATime, WAIT_FOR_SAVE } from "./oneAtATime";

// The Rules page saves the whole rule set with its revision, so two saves at once would send the same
// revision and the second would get a false "Someone else changed these rules" (S40 test report 3).
describe("oneAtATime", () => {
  it("refuses a second save while one is in flight, and runs one after it lands", async () => {
    let finish: (v: string) => void = () => {};
    const sends: string[] = [];
    const save = oneAtATime((what: string) => {
      sends.push(what);
      return new Promise<string>((resolve) => (finish = resolve));
    });
    expect(save.busy()).toBe(false);
    const first = save.run("setting");
    expect(save.busy()).toBe(true);
    expect(await save.run("rule")).toBe(WAIT_FOR_SAVE);
    expect(sends).toEqual(["setting"]);
    finish("");
    expect(await first).toBe("");
    expect(save.busy()).toBe(false);
    const third = save.run("rule");
    finish("refused");
    expect(await third).toBe("refused");
    expect(sends).toEqual(["setting", "rule"]);
  });

  it("is free again after a save that throws", async () => {
    const save = oneAtATime(async () => {
      throw new Error("down");
    });
    await expect(save.run(1)).rejects.toThrow("down");
    expect(save.busy()).toBe(false);
  });
});
