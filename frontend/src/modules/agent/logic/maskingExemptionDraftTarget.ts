import { isEqual } from "lodash-es";
import type { DatabaseResource } from "@/types";
import type { MaskingExemptionDraft } from "./maskingExemptionDraft";
import type { MaskingRuleDraftContext } from "./maskingRuleDraft";

interface MaskingExemptionDraftAction {
  messageId: string;
  draftId: string;
}

export interface MaskingExemptionDraftOrigin
  extends MaskingRuleDraftContext,
    MaskingExemptionDraftAction {}

export interface MaskingExemptionEditorSnapshot {
  sessionId: string;
  revision: string;
  project: string;
  chatId?: string;
  radioValue: "ALL" | "EXPRESSION" | "SELECT";
  databaseResources: DatabaseResource[];
  condition: string;
  conditionStatus?: "valid" | "incomplete";
  members: string[];
  description: string;
  expirationTimestamp: string | null;
}

type TargetContext = Pick<
  MaskingRuleDraftContext,
  "workspaceName" | "userName"
>;
interface Target {
  getSnapshot: () => Promise<MaskingExemptionEditorSnapshot | undefined>;
  open: (
    draft: MaskingExemptionDraft,
    chatId: string,
    origin?: MaskingExemptionDraftOrigin
  ) => Promise<void>;
  apply: (
    draft: MaskingExemptionDraft,
    expected: MaskingExemptionEditorSnapshot,
    chatId: string,
    origin?: MaskingExemptionDraftOrigin
  ) => Promise<void>;
}
interface Registration {
  context: TargetContext;
  target: Target;
  busy: boolean;
}
let registration: Registration | undefined;
const matching = (context: TargetContext) =>
  registration?.context.workspaceName === context.workspaceName &&
  registration.context.userName === context.userName
    ? registration
    : undefined;

export function registerMaskingExemptionDraftTarget(
  context: TargetContext,
  target: Target
): () => void {
  const next = { context: { ...context }, target, busy: false };
  registration = next;
  return () => {
    if (registration === next) registration = undefined;
  };
}

export function isMaskingExemptionDraftTargetReady(
  context: TargetContext
): boolean {
  return !!matching(context);
}

export async function getMaskingExemptionEditorSnapshot(
  context: TargetContext
): Promise<MaskingExemptionEditorSnapshot | undefined> {
  const current = matching(context);
  if (!current || current.busy) return undefined;
  const snapshot = await current.target.getSnapshot();
  return matching(context) === current ? snapshot : undefined;
}

async function withTarget(
  context: TargetContext,
  run: (target: Target) => Promise<void>
): Promise<void> {
  const current = matching(context);
  if (!current || current.busy)
    throw new Error("The masking exemption editor is unavailable or busy.");
  current.busy = true;
  try {
    await run(current.target);
  } finally {
    current.busy = false;
  }
}

export async function openMaskingExemptionDraftInEditor(
  draft: MaskingExemptionDraft,
  context: MaskingRuleDraftContext,
  action?: MaskingExemptionDraftAction
): Promise<void> {
  await withTarget(context, async (target) => {
    const snapshot = await target.getSnapshot();
    if (
      !snapshot ||
      snapshot.project !== draft.project ||
      (snapshot.chatId && snapshot.chatId !== context.chatId)
    )
      throw new Error(
        "The masking exemption editor project or chat has changed."
      );
    await target.open(
      draft,
      context.chatId,
      action && { ...context, ...action }
    );
  });
}

export async function applyMaskingExemptionDraftInEditor(
  draft: MaskingExemptionDraft,
  expected: MaskingExemptionEditorSnapshot,
  context: MaskingRuleDraftContext,
  action?: MaskingExemptionDraftAction
): Promise<void> {
  await withTarget(context, async (target) => {
    if (
      expected.project !== draft.project ||
      (expected.chatId && expected.chatId !== context.chatId) ||
      !isEqual(expected, await target.getSnapshot())
    )
      throw new Error(
        "The masking exemption draft has changed. Prepare it again."
      );
    await target.apply(
      draft,
      expected,
      context.chatId,
      action && { ...context, ...action }
    );
  });
}
