import { afterEach, describe, expect, it, vi } from "vitest";
import type { MaskingRuleDraft } from "./maskingRuleDraft";
import {
  applyMaskingRuleDraftInEditor,
  getMaskingRuleEditorSnapshot,
  isMaskingRuleDraftTargetReady,
  openMaskingRuleDraftInEditor,
  openMaskingRuleDraftsInEditor,
  registerMaskingRuleDraftTarget,
} from "./maskingRuleDraftTarget";

const context = {
  workspaceName: "workspaces/default",
  userName: "users/member@example.com",
  chatId: "chat-1",
};
const draft: MaskingRuleDraft = {
  title: "Mask email",
  condition: 'resource.column_name == "email"',
  semanticType: "DEFAULT",
  scope: { kind: "workspace" },
};
const snapshot = {
  sessionId: "editor-1",
  revision: "1",
  title: "Mask email",
  condition: draft.condition,
  semanticType: "DEFAULT",
};
const cleanups: (() => void)[] = [];

afterEach(() => {
  for (const cleanup of cleanups.splice(0)) cleanup();
});

describe("maskingRuleDraftTarget", () => {
  it("delivers a batch in one handoff and never falls back to partial singleton opens", async () => {
    const open = vi.fn(async () => {});
    const openMany = vi.fn(async () => {});
    cleanups.push(
      registerMaskingRuleDraftTarget(context, {
        getSnapshot: async () => undefined,
        open,
        openMany,
        apply: async () => {},
      })
    );
    await openMaskingRuleDraftsInEditor([draft, draft], context);
    expect(openMany).toHaveBeenCalledWith([draft, draft], context.chatId);
    expect(open).not.toHaveBeenCalled();
  });
  it("fails closed without a matching mounted editor target", async () => {
    expect(isMaskingRuleDraftTargetReady(context)).toBe(false);
    expect(await getMaskingRuleEditorSnapshot(context)).toBeUndefined();
    await expect(
      openMaskingRuleDraftInEditor(draft, context)
    ).rejects.toThrow();
    await expect(
      applyMaskingRuleDraftInEditor(draft, snapshot, context)
    ).rejects.toThrow();
  });

  it("isolates the target by workspace and user and unregisters on unmount", async () => {
    const open = vi.fn(async () => {});
    const cleanup = registerMaskingRuleDraftTarget(context, {
      getSnapshot: async () => snapshot,
      open,
      apply: async () => {},
    });
    cleanups.push(cleanup);
    expect(isMaskingRuleDraftTargetReady(context)).toBe(true);
    expect(await getMaskingRuleEditorSnapshot(context)).toEqual(snapshot);
    expect(
      await getMaskingRuleEditorSnapshot({
        ...context,
        userName: "users/other@example.com",
      })
    ).toBeUndefined();
    await expect(
      openMaskingRuleDraftInEditor(draft, {
        ...context,
        workspaceName: "workspaces/other",
      })
    ).rejects.toThrow();
    await openMaskingRuleDraftInEditor(draft, context);
    expect(open).toHaveBeenCalledWith(draft, context.chatId);
    cleanup();
    expect(isMaskingRuleDraftTargetReady(context)).toBe(false);
  });

  it("exposes an existing-rule editor snapshot for read-only review", async () => {
    const editSnapshot = { ...snapshot, mode: "edit" as const };
    cleanups.push(
      registerMaskingRuleDraftTarget(context, {
        getSnapshot: async () => editSnapshot,
        open: async () => {},
        apply: async () => {},
      })
    );

    expect(await getMaskingRuleEditorSnapshot(context)).toEqual(editSnapshot);
  });

  it("does not hand a full-rule proposal to an existing-rule editor", async () => {
    const editSnapshot = { ...snapshot, mode: "edit" as const };
    const apply = vi.fn(async () => {});
    cleanups.push(
      registerMaskingRuleDraftTarget(context, {
        getSnapshot: async () => editSnapshot,
        open: async () => {},
        apply,
      })
    );

    await expect(
      applyMaskingRuleDraftInEditor(draft, editSnapshot, context)
    ).rejects.toThrow(/existing.*review-only/i);
    expect(apply).not.toHaveBeenCalled();
  });

  it("does not let an old cleanup unregister a newer target", async () => {
    const target = {
      getSnapshot: async () => snapshot,
      open: async () => {},
      apply: async () => {},
    };
    const oldCleanup = registerMaskingRuleDraftTarget(context, target);
    const cleanup = registerMaskingRuleDraftTarget(context, target);
    cleanups.push(oldCleanup, cleanup);
    oldCleanup();
    expect(isMaskingRuleDraftTargetReady(context)).toBe(true);
  });

  it("rejects a snapshot that completed after the target unmounted", async () => {
    let resolve!: (value: typeof snapshot) => void;
    const cleanup = registerMaskingRuleDraftTarget(context, {
      getSnapshot: () =>
        new Promise((done) => {
          resolve = done;
        }),
      open: async () => {},
      apply: async () => {},
    });
    cleanups.push(cleanup);
    const pending = getMaskingRuleEditorSnapshot(context);
    cleanup();
    resolve(snapshot);
    expect(await pending).toBeUndefined();
  });

  it("prevents concurrent handoffs and forwards exact apply evidence", async () => {
    let resolve!: () => void;
    const apply = vi.fn(async () => {});
    cleanups.push(
      registerMaskingRuleDraftTarget(context, {
        getSnapshot: async () => snapshot,
        open: () =>
          new Promise<void>((done) => {
            resolve = done;
          }),
        apply,
      })
    );
    const pending = openMaskingRuleDraftInEditor(draft, context);
    await expect(
      openMaskingRuleDraftInEditor(draft, context)
    ).rejects.toThrow();
    await expect(
      applyMaskingRuleDraftInEditor(draft, snapshot, context)
    ).rejects.toThrow();
    resolve();
    await pending;
    await applyMaskingRuleDraftInEditor(draft, snapshot, context);
    expect(apply).toHaveBeenCalledWith(draft, snapshot, context.chatId);
  });
});
