// @vitest-environment node
import { create } from "@bufbuild/protobuf";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { AppStoreState } from "@/stores/app/types";
import { SettingSchema } from "@/types/proto-es/v1/setting_service_pb";
import {
  consumeSemanticTypeDraft,
  getSafeSemanticTypeDraftSummary,
  getSemanticTypeDraft,
  getSemanticTypeDraftSource,
  prepareSemanticTypeDraft,
  revalidateSemanticTypeDraft,
  validateSemanticTypeDraft,
} from "./semanticTypeDraft";
import type { getMaskingSetup } from "./tools/maskingSetup";

const mocks = vi.hoisted(() => ({
  state: undefined as unknown as AppStoreState,
  getSetting: vi.fn(),
  getMaskingSetup: vi.fn<typeof getMaskingSetup>(),
  getSnapshot: vi.fn(),
  refreshSubscription: vi.fn(),
}));
vi.mock("@/stores/app", () => ({
  useAppStore: { getState: () => mocks.state },
}));
vi.mock("@/api", () => ({
  settingServiceClientConnect: { getSetting: mocks.getSetting },
}));
vi.mock("./tools/maskingSetup", () => ({
  getMaskingSetup: mocks.getMaskingSetup,
}));
vi.mock("./semanticTypeDraftTarget", () => ({
  getSemanticTypeDraftEditorSnapshot: mocks.getSnapshot,
}));

const context = {
  workspaceName: "workspaces/example",
  userName: "users/member@example.com",
  chatId: "semantic-chat",
};
const proposal = {
  title: "Email preview",
  description: "Synthetic",
  algorithm: { kind: "full-mask", substitution: "******" },
};
const snapshot = { sessionId: "page-1", revision: "1", busy: false };

function setup(): Awaited<ReturnType<typeof getMaskingSetup>> {
  const evidence = {
    status: "allowed" as const,
    scope: context.workspaceName,
    requiredPermissions: [],
    missingPermissions: [],
  };
  return {
    source: "frontend-permission-check",
    features: {
      masking: "available",
      classification: "available",
      catalogMasking: "unknown",
    },
    steps: [
      { id: "semantic-types", readAccess: evidence, configure: evidence },
    ],
  };
}
beforeEach(() => {
  vi.clearAllMocks();
  vi.useFakeTimers();
  vi.setSystemTime(new Date("2026-11-09T00:00:00Z"));
  mocks.state = {
    currentUser: { name: context.userName },
    serverInfo: { workspace: context.workspaceName },
    refreshSubscription: mocks.refreshSubscription,
  } as unknown as AppStoreState;
  mocks.refreshSubscription.mockResolvedValue({});
  mocks.getMaskingSetup.mockResolvedValue(setup());
  mocks.getSetting.mockResolvedValue(
    create(SettingSchema, {
      name: "settings/SEMANTIC_TYPES",
      value: { value: { case: "semanticType", value: { types: [] } } },
    })
  );
  mocks.getSnapshot.mockResolvedValue(undefined);
});
afterEach(() => vi.useRealTimers());

describe("semanticTypeDraft", () => {
  it("prepares a new unsaved type and preserves duplicate title and configuration", async () => {
    const first = await prepareSemanticTypeDraft(proposal, context);
    mocks.getSetting.mockResolvedValue(
      create(SettingSchema, {
        value: {
          value: { case: "semanticType", value: { types: [first.draft] } },
        },
      })
    );
    const second = await prepareSemanticTypeDraft(proposal, context);
    expect(first.draft.id).toMatch(/^[\da-f-]{36}$/);
    expect(first.draftId).not.toBe(first.draft.id);
    expect(second.draft.id).not.toBe(first.draft.id);
    expect(second.draft.title).toBe(first.draft.title);
    expect(second.draft.algorithm).toEqual(first.draft.algorithm);
    expect(getSemanticTypeDraft(first.draftId, context)).toEqual(first.draft);
    expect(mocks.getSetting).toHaveBeenCalledWith(
      expect.objectContaining({ name: "settings/SEMANTIC_TYPES" }),
      expect.anything()
    );
  });

  it.each([
    [
      {
        kind: "range-mask",
        slices: [
          { start: 0, end: 2, substitution: "*" },
          { start: 4, end: 8, substitution: "x" },
        ],
      },
      "rangeMask",
    ],
    [{ kind: "md5-mask" }, "md5Mask"],
    [
      {
        kind: "inner-outer-mask",
        type: "inner",
        prefixLen: 1,
        suffixLen: 2,
        substitution: "*",
      },
      "innerOuterMask",
    ],
    [
      {
        kind: "inner-outer-mask",
        type: "outer",
        prefixLen: 0,
        suffixLen: 0,
        substitution: "*",
      },
      "innerOuterMask",
    ],
  ])("builds supported algorithm %j", async (algorithm, expected) => {
    const { draft } = await prepareSemanticTypeDraft(
      { ...proposal, algorithm },
      context
    );
    expect(draft.algorithm.mask.case).toBe(expected);
    expect(getSafeSemanticTypeDraftSummary(draft).algorithm).toMatchObject(
      algorithm
    );
  });

  it("generates distinct MD5 salts without putting salt in the safe summary", async () => {
    const first = await prepareSemanticTypeDraft(
      { ...proposal, algorithm: { kind: "md5-mask" } },
      context
    );
    const second = await prepareSemanticTypeDraft(
      { ...proposal, algorithm: { kind: "md5-mask" } },
      context
    );
    expect(first.draft.algorithm.mask.case).toBe("md5Mask");
    if (
      first.draft.algorithm.mask.case !== "md5Mask" ||
      second.draft.algorithm.mask.case !== "md5Mask"
    )
      throw new Error("Expected MD5");
    expect(first.draft.algorithm.mask.value.salt).toBeTruthy();
    expect(first.draft.algorithm.mask.value.salt).not.toBe(
      second.draft.algorithm.mask.value.salt
    );
    expect(
      JSON.stringify(getSafeSemanticTypeDraftSummary(first.draft))
    ).not.toContain(first.draft.algorithm.mask.value.salt);
    expect(getSafeSemanticTypeDraftSummary(first.draft).algorithm).toEqual({
      kind: "md5-mask",
    });
  });

  it("accepts whitespace replacement and the 16-byte Unicode boundary", async () => {
    for (const replacement of [" ", "🙂".repeat(4)]) {
      const { draft } = await prepareSemanticTypeDraft(
        {
          ...proposal,
          algorithm: { kind: "full-mask", substitution: replacement },
        },
        context
      );
      expect(getSafeSemanticTypeDraftSummary(draft).algorithm).toEqual({
        kind: "full-mask",
        substitution: replacement,
      });
    }
  });

  it.each([
    null,
    { ...proposal, id: "bb.default" },
    { ...proposal, title: " " },
    { ...proposal, title: "x".repeat(201) },
    { ...proposal, description: "x".repeat(2001) },
    { ...proposal, description: null },
    { ...proposal, algorithm: { kind: "full-mask", substitution: "" } },
    {
      ...proposal,
      algorithm: { kind: "full-mask", substitution: "🙂".repeat(5) },
    },
    { ...proposal, algorithm: { kind: "full-mask", substitution: "x\0" } },
    { ...proposal, algorithm: { kind: "md5-mask", salt: "from-model" } },
    { ...proposal, algorithm: { kind: "unsupported" } },
    { ...proposal, algorithm: { kind: "range-mask", slices: [] } },
    {
      ...proposal,
      algorithm: {
        kind: "range-mask",
        slices: [{ start: -1, end: 2, substitution: "*" }],
      },
    },
    {
      ...proposal,
      algorithm: {
        kind: "range-mask",
        slices: [{ start: 1, end: 1, substitution: "*" }],
      },
    },
    {
      ...proposal,
      algorithm: {
        kind: "range-mask",
        slices: [{ start: 0.5, end: 2, substitution: "*" }],
      },
    },
    {
      ...proposal,
      algorithm: {
        kind: "range-mask",
        slices: [{ start: 0, end: 2147483648, substitution: "*" }],
      },
    },
    {
      ...proposal,
      algorithm: {
        kind: "range-mask",
        slices: [
          { start: 0, end: 4, substitution: "*" },
          { start: 2, end: 6, substitution: "*" },
        ],
      },
    },
    {
      ...proposal,
      algorithm: {
        kind: "inner-outer-mask",
        type: "unknown",
        prefixLen: 0,
        suffixLen: 0,
        substitution: "*",
      },
    },
    {
      ...proposal,
      algorithm: {
        kind: "inner-outer-mask",
        type: "inner",
        prefixLen: -1,
        suffixLen: 0,
        substitution: "*",
      },
    },
    {
      ...proposal,
      algorithm: {
        kind: "inner-outer-mask",
        type: "inner",
        prefixLen: 2147483647,
        suffixLen: 1,
        substitution: "*",
      },
    },
  ])("rejects invalid or model-owned private fields %j", async (raw) => {
    await expect(prepareSemanticTypeDraft(raw, context)).rejects.toThrow();
    expect(mocks.getSetting).not.toHaveBeenCalled();
  });

  it.each(["readAccess", "configure"] as const)(
    "fails closed for denied or unknown %s",
    async (field) => {
      for (const status of ["denied", "unknown", "needs-scope"] as const) {
        const denied = setup();
        denied.steps[0][field].status = status;
        mocks.getMaskingSetup.mockResolvedValue(denied);
        await expect(
          prepareSemanticTypeDraft(proposal, context)
        ).rejects.toThrow(/permission/i);
      }
      expect(mocks.getSetting).not.toHaveBeenCalled();
    }
  );

  it("requires a fresh subscription, feature availability and actual readable settings", async () => {
    mocks.refreshSubscription.mockResolvedValue(undefined);
    await expect(prepareSemanticTypeDraft(proposal, context)).rejects.toThrow(
      /subscription|feature/i
    );
    mocks.refreshSubscription.mockResolvedValue({});
    const unavailable = setup();
    unavailable.features.masking = "unavailable";
    mocks.getMaskingSetup.mockResolvedValue(unavailable);
    await expect(prepareSemanticTypeDraft(proposal, context)).rejects.toThrow(
      /feature/i
    );
    mocks.getMaskingSetup.mockResolvedValue(setup());
    mocks.getSetting.mockRejectedValue(new Error("Permission denied"));
    await expect(prepareSemanticTypeDraft(proposal, context)).rejects.toThrow(
      /Permission denied/
    );
    mocks.getSetting.mockResolvedValue(create(SettingSchema));
    await expect(prepareSemanticTypeDraft(proposal, context)).rejects.toThrow(
      /setting/i
    );
  });

  it("detects ID collision during click or explicit route save validation", async () => {
    const { draftId, draft } = await prepareSemanticTypeDraft(
      proposal,
      context
    );
    mocks.getSetting.mockResolvedValue(
      create(SettingSchema, {
        value: { value: { case: "semanticType", value: { types: [draft] } } },
      })
    );
    await expect(revalidateSemanticTypeDraft(draftId, context)).rejects.toThrow(
      /already|exists|collision/i
    );
    await expect(validateSemanticTypeDraft(draft, context)).rejects.toThrow(
      /already|exists|collision/i
    );
  });

  it("captures source revisions and refuses a busy or changed page", async () => {
    mocks.getSnapshot.mockResolvedValue(snapshot);
    const { draftId } = await prepareSemanticTypeDraft(proposal, context);
    expect(getSemanticTypeDraftSource(draftId, context)).toEqual(snapshot);
    mocks.getSnapshot.mockResolvedValue({ ...snapshot, revision: "2" });
    await expect(revalidateSemanticTypeDraft(draftId, context)).rejects.toThrow(
      /changed/i
    );
    mocks.getSnapshot.mockResolvedValue({ ...snapshot, busy: true });
    await expect(prepareSemanticTypeDraft(proposal, context)).rejects.toThrow(
      /busy|unsaved/i
    );
  });

  it("revalidates permissions on click and returns detached source snapshots", async () => {
    mocks.getSnapshot.mockResolvedValue(snapshot);
    const { draftId } = await prepareSemanticTypeDraft(proposal, context);
    const source = getSemanticTypeDraftSource(draftId, context)!;
    source.revision = "mutated";
    expect(getSemanticTypeDraftSource(draftId, context)).toEqual(snapshot);
    const denied = setup();
    denied.steps[0].configure.status = "denied";
    mocks.getMaskingSetup.mockResolvedValue(denied);
    await expect(revalidateSemanticTypeDraft(draftId, context)).rejects.toThrow(
      /permission/i
    );
  });

  it("rejects a page or identity change while fresh validation awaits", async () => {
    mocks.getSnapshot
      .mockResolvedValueOnce(snapshot)
      .mockResolvedValueOnce({ ...snapshot, revision: "2" });
    await expect(prepareSemanticTypeDraft(proposal, context)).rejects.toThrow(
      /changed/i
    );
    mocks.getSnapshot.mockResolvedValue(undefined);
    mocks.getSetting.mockImplementation(async () => {
      mocks.state = {
        ...mocks.state,
        currentUser: { name: "users/other@example.com" },
      } as AppStoreState;
      return create(SettingSchema, {
        value: { value: { case: "semanticType", value: { types: [] } } },
      });
    });
    await expect(prepareSemanticTypeDraft(proposal, context)).rejects.toThrow(
      /context/i
    );
  });

  it("isolates workspace, user, chat and returned copies and consumes only the matching draft", async () => {
    const { draftId, draft } = await prepareSemanticTypeDraft(
      proposal,
      context
    );
    draft.title = "mutated";
    expect(getSemanticTypeDraft(draftId, context)?.title).toBe(proposal.title);
    expect(
      getSemanticTypeDraft(draftId, { ...context, chatId: "other" })
    ).toBeUndefined();
    consumeSemanticTypeDraft(draftId, { ...context, chatId: "other" });
    expect(getSemanticTypeDraft(draftId, context)).toBeDefined();
    consumeSemanticTypeDraft(draftId, context);
    expect(getSemanticTypeDraft(draftId, context)).toBeUndefined();
  });

  it("expires drafts and rechecks their lifetime after an asynchronous refresh", async () => {
    const { draftId } = await prepareSemanticTypeDraft(proposal, context);
    mocks.getSetting.mockImplementation(async () => {
      vi.advanceTimersByTime(30 * 60 * 1000);
      return create(SettingSchema, {
        value: { value: { case: "semanticType", value: { types: [] } } },
      });
    });
    await expect(revalidateSemanticTypeDraft(draftId, context)).rejects.toThrow(
      /expired|unavailable/i
    );
    expect(getSemanticTypeDraft(draftId, context)).toBeUndefined();
  });
});
