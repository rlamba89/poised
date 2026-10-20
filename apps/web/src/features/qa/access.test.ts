import { describe, expect, it } from "vitest";
import { ApiError } from "@/lib/api";
import { SIGN_IN_FIRST, qaError } from "./access";

describe("qaError", () => {
  it("says to sign in as a platform admin, or that only one can, or that the tools aren't here", () => {
    expect(qaError(new ApiError(401, "Please sign in."), "x")).toBe(SIGN_IN_FIRST);
    expect(qaError(new ApiError(403, "Only Preopsis's platform admins can do this."), "x")).toContain("platform admin");
    expect(qaError(new ApiError(404, "not found"), "Not captured here.")).toBe("Not captured here.");
    expect(qaError(new Error("Network down"), "x")).toBe("Network down");
  });
});
