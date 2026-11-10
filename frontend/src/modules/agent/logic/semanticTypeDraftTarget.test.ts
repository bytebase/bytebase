import { create } from "@bufbuild/protobuf";
import { afterEach, describe, expect, it, vi } from "vitest";
import { AlgorithmSchema } from "@/types/proto-es/v1/setting_service_pb";
import type { SemanticTypeDraft } from "./semanticTypeDraft";
import {
  getSemanticTypeDraftEditorSnapshot,
  isSemanticTypeDraftTargetReady,
  openSemanticTypeDraftInEditor,
  registerSemanticTypeDraftTarget,
} from "./semanticTypeDraftTarget";

const context = {
  workspaceName: "workspaces/example",
  userName: "users/member@example.com",
  chatId: "semantic-chat",
};
const draft: SemanticTypeDraft = {
  id: "9abb13f9-ad01-4b64-8d1d-b8197f984c6d",
  title: "Email",
  description: "",
  algorithm: create(AlgorithmSchema, {
    mask: { case: "fullMask", value: { substitution: "******" } },
  }),
};
const snapshot = { sessionId: "page-1", revision: "1", busy: false };
const cleanups: (() => void)[] = [];
afterEach(() => {
  for (const cleanup of cleanups.splice(0)) cleanup();
});

describe("semanticTypeDraftTarget", () => {
  it("fails closed without a mounted matching page", async () => {
    expect(isSemanticTypeDraftTargetReady(context)).toBe(false);
    expect(await getSemanticTypeDraftEditorSnapshot(context)).toBeUndefined();
    await expect(
      openSemanticTypeDraftInEditor(draft, context)
    ).rejects.toThrow();
  });
  it("opens a new draft with exact origin and isolates workspace and user", async () => {
    const open = vi.fn(async () => {});
    cleanups.push(
      registerSemanticTypeDraftTarget(context, {
        getSnapshot: async () => snapshot,
        open,
      })
    );
    expect(isSemanticTypeDraftTargetReady(context)).toBe(true);
    expect(await getSemanticTypeDraftEditorSnapshot(context)).toEqual(snapshot);
    expect(
      isSemanticTypeDraftTargetReady({
        ...context,
        userName: "users/other@example.com",
      })
    ).toBe(false);
    await expect(
      openSemanticTypeDraftInEditor(draft, {
        ...context,
        workspaceName: "workspaces/other",
      })
    ).rejects.toThrow();
    await openSemanticTypeDraftInEditor(
      draft,
      context,
      { messageId: "message-1", draftId: "draft-1" },
      snapshot
    );
    expect(open).toHaveBeenCalledWith(draft, context.chatId, {
      ...context,
      messageId: "message-1",
      draftId: "draft-1",
    });
  });
  it("rejects unsaved or changed page snapshots without opening", async () => {
    const open = vi.fn(async () => {});
    const getSnapshot = vi.fn().mockResolvedValue({ ...snapshot, busy: true });
    cleanups.push(
      registerSemanticTypeDraftTarget(context, { getSnapshot, open })
    );
    await expect(openSemanticTypeDraftInEditor(draft, context)).rejects.toThrow(
      /busy|unsaved/i
    );
    getSnapshot.mockResolvedValue({ ...snapshot, revision: "2" });
    await expect(
      openSemanticTypeDraftInEditor(draft, context, undefined, snapshot)
    ).rejects.toThrow(/changed/i);
    expect(open).not.toHaveBeenCalled();
  });
  it("does not let old cleanup unregister a newer target or a stale snapshot open it", async () => {
    let resolve!: (value: typeof snapshot) => void;
    const oldOpen = vi.fn(async () => {});
    const oldCleanup = registerSemanticTypeDraftTarget(context, {
      getSnapshot: () =>
        new Promise((done) => {
          resolve = done;
        }),
      open: oldOpen,
    });
    cleanups.push(oldCleanup);
    const pending = openSemanticTypeDraftInEditor(draft, context);
    const open = vi.fn(async () => {});
    cleanups.push(
      registerSemanticTypeDraftTarget(context, {
        getSnapshot: async () => snapshot,
        open,
      })
    );
    oldCleanup();
    resolve(snapshot);
    await expect(pending).rejects.toThrow(/changed|unavailable/i);
    expect(oldOpen).not.toHaveBeenCalled();
    expect(open).not.toHaveBeenCalled();
    expect(isSemanticTypeDraftTargetReady(context)).toBe(true);
  });
  it("blocks concurrent handoffs and returns detached snapshots", async () => {
    let resolve!: () => void;
    cleanups.push(
      registerSemanticTypeDraftTarget(context, {
        getSnapshot: async () => snapshot,
        open: () =>
          new Promise<void>((done) => {
            resolve = done;
          }),
      })
    );
    const copy = await getSemanticTypeDraftEditorSnapshot(context);
    copy!.revision = "mutated";
    expect(snapshot.revision).toBe("1");
    const pending = openSemanticTypeDraftInEditor(draft, context);
    await expect(openSemanticTypeDraftInEditor(draft, context)).rejects.toThrow(
      /busy/i
    );
    await expect(getSemanticTypeDraftEditorSnapshot(context)).rejects.toThrow(
      /busy/i
    );
    resolve();
    await pending;
  });
});
