import { beforeEach, describe, expect, test, vi } from "vitest";
import {
  type AccessGrantDraft,
  consumeAccessGrantDraft,
  markDeniedQueryRequestDraftDelivered,
  saveAccessGrantDraft,
  wasDeniedQueryRequestDraftDelivered,
} from "./accessGrantDraft";

const draft: AccessGrantDraft = {
  project: "projects/sample",
  sourceDatabase: "instances/prod/databases/app",
  targets: ["instances/prod/databases/app"],
  statement: "SELECT id FROM users",
  schema: "public",
};

beforeEach(() => {
  vi.useRealTimers();
  sessionStorage.clear();
});

describe("access grant draft storage", () => {
  test("preserves the exact member-stated reason", () => {
    const reasonDraft = { ...draft, reason: "  Investigate a discrepancy.\n" };
    expect(consumeAccessGrantDraft(saveAccessGrantDraft(reasonDraft))).toEqual(
      reasonDraft
    );
  });

  test.each([42, null, {}, []])(
    "rejects non-string stored reason %j",
    (reason) => {
      const key = saveAccessGrantDraft(draft);
      sessionStorage.setItem(
        key,
        JSON.stringify({ createdAt: Date.now(), draft: { ...draft, reason } })
      );
      expect(consumeAccessGrantDraft(key)).toBeUndefined();
    }
  );

  test("holds the exact draft in session storage without putting SQL in its key", () => {
    const key = saveAccessGrantDraft(draft);
    expect(key).not.toContain("SELECT");
    expect(consumeAccessGrantDraft(key)).toEqual(draft);
    expect(sessionStorage.getItem(key)).toBeNull();
  });

  test.each([true, false])(
    "preserves unmask=%s in a stored draft",
    (unmask) => {
      const unmaskDraft = { ...draft, unmask };
      const key = saveAccessGrantDraft(unmaskDraft);
      expect(consumeAccessGrantDraft(key)).toEqual(unmaskDraft);
    }
  );

  test.each(["true", 1, null, {}, []].map((unmask) => ({ unmask })))(
    "rejects a non-boolean stored unmask value $unmask",
    ({ unmask }) => {
      const key = saveAccessGrantDraft(draft);
      sessionStorage.setItem(
        key,
        JSON.stringify({
          createdAt: Date.now(),
          draft: { ...draft, unmask },
        })
      );
      expect(consumeAccessGrantDraft(key)).toBeUndefined();
      expect(sessionStorage.getItem(key)).toBeNull();
    }
  );

  test("rejects unknown keys and malformed stored drafts", () => {
    expect(consumeAccessGrantDraft("other-key")).toBeUndefined();
    const key = saveAccessGrantDraft(draft);
    sessionStorage.setItem(key, "not json");
    expect(consumeAccessGrantDraft(key)).toBeUndefined();
  });

  test("expires a draft instead of opening stale prefilled data", () => {
    vi.useFakeTimers();
    const key = saveAccessGrantDraft(draft);
    vi.advanceTimersByTime(11 * 60 * 1000);
    expect(consumeAccessGrantDraft(key)).toBeUndefined();
  });

  test("reports delivery only after a consumed draft opens a drawer", () => {
    const key = saveAccessGrantDraft(draft);
    markDeniedQueryRequestDraftDelivered(key);
    expect(wasDeniedQueryRequestDraftDelivered(key)).toBe(false);
    expect(consumeAccessGrantDraft(key)).toEqual(draft);
    expect(wasDeniedQueryRequestDraftDelivered(key)).toBe(false);
    markDeniedQueryRequestDraftDelivered(key);
    expect(wasDeniedQueryRequestDraftDelivered(key)).toBe(true);
  });
});
