import { describe, expect, test } from "vitest";
import {
  invalidateSourceDrafts,
  type SourceDraftState,
} from "./data-source-drafts";

const initialState: SourceDraftState = {
  instanceName: "instances/production",
  resetEvent: 0,
};

describe("invalidateSourceDrafts", () => {
  test("keeps drafts while switching providers in the same edit", () => {
    const drafts = new Map([["admin", new Map([[1, "vault token"]])]]);

    const next = invalidateSourceDrafts(drafts, initialState, initialState);

    expect(next).toEqual(initialState);
    expect(drafts.get("admin")?.get(1)).toBe("vault token");
  });

  test("discards abandoned provider credentials after a form reset", () => {
    const drafts = new Map([["admin", new Map([[1, "vault token"]])]]);

    const next = invalidateSourceDrafts(drafts, initialState, {
      ...initialState,
      resetEvent: 1,
    });

    expect(next.resetEvent).toBe(1);
    expect(drafts).toHaveLength(0);
  });
});
