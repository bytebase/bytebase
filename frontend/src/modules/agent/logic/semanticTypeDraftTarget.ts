import { isEqual } from "lodash-es";
import type { MaskingRuleDraftContext } from "./maskingRuleDraft";
import type { SemanticTypeDraft } from "./semanticTypeDraft";

export interface SemanticTypeDraftEditorSnapshot {
  sessionId: string;
  revision: string;
  busy: boolean;
}
export interface SemanticTypeDraftOrigin extends MaskingRuleDraftContext {
  messageId: string;
  draftId: string;
}
type Context = Pick<MaskingRuleDraftContext, "workspaceName" | "userName">;
interface Target {
  getSnapshot: () => Promise<SemanticTypeDraftEditorSnapshot | undefined>;
  open: (
    draft: SemanticTypeDraft,
    chatId: string,
    origin?: SemanticTypeDraftOrigin
  ) => Promise<void>;
}
export type SemanticTypeEditorSnapshot = SemanticTypeDraftEditorSnapshot;
interface Registration {
  context: Context;
  target: Target;
  busy: boolean;
}
let registration: Registration | undefined;
function matching(context: Context): Registration | undefined {
  return registration?.context.workspaceName === context.workspaceName &&
    registration.context.userName === context.userName
    ? registration
    : undefined;
}
export function registerSemanticTypeDraftTarget(
  context: Context,
  target: Target
): () => void {
  const next = { context: { ...context }, target, busy: false };
  registration = next;
  return () => {
    if (registration === next) registration = undefined;
  };
}
export function isSemanticTypeDraftTargetReady(context: Context): boolean {
  return !!matching(context);
}
export async function getSemanticTypeDraftEditorSnapshot(
  context: Context
): Promise<SemanticTypeDraftEditorSnapshot | undefined> {
  const current = matching(context);
  if (!current) return undefined;
  if (current.busy) throw new Error("The semantic type editor is busy.");
  const snapshot = await current.target.getSnapshot();
  return matching(context) === current && snapshot
    ? structuredClone(snapshot)
    : undefined;
}
export async function openSemanticTypeDraftInEditor(
  draft: SemanticTypeDraft,
  context: MaskingRuleDraftContext,
  action?: { messageId: string; draftId: string },
  expected?: SemanticTypeDraftEditorSnapshot
): Promise<void> {
  const current = matching(context);
  if (!current || current.busy)
    throw new Error("The semantic type editor is unavailable or busy.");
  current.busy = true;
  try {
    const snapshot = await current.target.getSnapshot();
    if (matching(context) !== current || !snapshot)
      throw new Error("The semantic type editor changed or is unavailable.");
    if (snapshot.busy)
      throw new Error(
        "The semantic type editor has unsaved changes or is busy."
      );
    if (expected && !isEqual(snapshot, expected))
      throw new Error("The semantic type editor changed. Prepare a new draft.");
    await current.target.open(
      draft,
      context.chatId,
      action && { ...context, ...action }
    );
  } finally {
    current.busy = false;
  }
}
