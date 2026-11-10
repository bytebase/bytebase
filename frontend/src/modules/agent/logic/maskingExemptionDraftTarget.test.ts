import { afterEach, expect, it, vi } from "vitest";
import {
  applyMaskingExemptionDraftInEditor,
  getMaskingExemptionEditorSnapshot,
  isMaskingExemptionDraftTargetReady,
  openMaskingExemptionDraftInEditor,
  registerMaskingExemptionDraftTarget,
} from "./maskingExemptionDraftTarget";

const context = {
  workspaceName: "workspaces/example",
  userName: "users/member@example.com",
  chatId: "chat",
};
const draft = {
  project: "projects/example",
  scope: { kind: "project" as const },
  members: ["user:member@example.com"],
  condition: 'resource.column_name == "email"',
  description: "Review",
  expirationTimestamp: null,
};
const snapshot = {
  sessionId: "session",
  revision: "1",
  project: draft.project,
  radioValue: "ALL" as const,
  databaseResources: [],
  condition: "",
  members: [],
  description: "",
  expirationTimestamp: null,
};
let cleanup = () => {};
afterEach(() => cleanup());

it("isolates registrations and fails closed after unmount", async () => {
  const open = vi.fn(async () => {});
  cleanup = registerMaskingExemptionDraftTarget(context, {
    getSnapshot: async () => snapshot,
    open,
    apply: async () => {},
  });
  expect(isMaskingExemptionDraftTargetReady(context)).toBe(true);
  expect(await getMaskingExemptionEditorSnapshot(context)).toEqual(snapshot);
  expect(
    isMaskingExemptionDraftTargetReady({
      ...context,
      userName: "users/other@example.com",
    })
  ).toBe(false);
  await openMaskingExemptionDraftInEditor(draft, context);
  expect(open).toHaveBeenCalledWith(draft, context.chatId, undefined);
  cleanup();
  await expect(
    openMaskingExemptionDraftInEditor(draft, context)
  ).rejects.toThrow();
});

it("rejects wrong project, chat, and stale revisions before applying", async () => {
  const apply = vi.fn(async () => {});
  cleanup = registerMaskingExemptionDraftTarget(context, {
    getSnapshot: async () => snapshot,
    open: async () => {},
    apply,
  });
  await expect(
    applyMaskingExemptionDraftInEditor(
      draft,
      { ...snapshot, revision: "old" },
      context
    )
  ).rejects.toThrow();
  await expect(
    applyMaskingExemptionDraftInEditor(
      draft,
      { ...snapshot, project: "projects/other" },
      context
    )
  ).rejects.toThrow();
  await expect(
    applyMaskingExemptionDraftInEditor(
      draft,
      { ...snapshot, chatId: "other" },
      context
    )
  ).rejects.toThrow();
  expect(apply).not.toHaveBeenCalled();
  await applyMaskingExemptionDraftInEditor(draft, snapshot, context);
  expect(apply).toHaveBeenCalledWith(
    draft,
    snapshot,
    context.chatId,
    undefined
  );
});

it("forwards the exact originating action to the opened and refined form", async () => {
  const action = { draftId: "exemption-1", messageId: "message-1" };
  const origin = { ...context, ...action };
  const open = vi.fn(async () => {});
  const apply = vi.fn(async () => {});
  cleanup = registerMaskingExemptionDraftTarget(context, {
    getSnapshot: async () => snapshot,
    open,
    apply,
  });
  await openMaskingExemptionDraftInEditor(draft, context, action);
  expect(open).toHaveBeenCalledWith(draft, context.chatId, origin);
  await applyMaskingExemptionDraftInEditor(draft, snapshot, context, action);
  expect(apply).toHaveBeenCalledWith(draft, snapshot, context.chatId, origin);
});
