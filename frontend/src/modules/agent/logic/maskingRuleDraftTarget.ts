import type {
  MaskingRuleDraft,
  MaskingRuleDraftContext,
} from "./maskingRuleDraft";

export interface MaskingRuleEditorSnapshot {
  sessionId: string;
  revision: string;
  mode?: "create" | "edit";
  title: string;
  condition: string;
  conditionStatus?: "valid" | "incomplete";
  semanticType: string;
  chatId?: string;
}

type TargetContext = Pick<
  MaskingRuleDraftContext,
  "workspaceName" | "userName"
>;
interface MaskingRuleDraftTarget {
  getSnapshot: () => Promise<MaskingRuleEditorSnapshot | undefined>;
  open: (draft: MaskingRuleDraft, chatId: string) => Promise<void>;
  openMany?: (drafts: MaskingRuleDraft[], chatId: string) => Promise<void>;
  apply: (
    draft: MaskingRuleDraft,
    expected: MaskingRuleEditorSnapshot,
    chatId: string
  ) => Promise<void>;
}
interface Registration {
  context: TargetContext;
  target: MaskingRuleDraftTarget;
  busy: boolean;
}

let registration: Registration | undefined;

function matchingTarget(context: TargetContext): Registration | undefined {
  return registration?.context.workspaceName === context.workspaceName &&
    registration.context.userName === context.userName
    ? registration
    : undefined;
}

export function registerMaskingRuleDraftTarget(
  context: TargetContext,
  target: MaskingRuleDraftTarget
): () => void {
  const next: Registration = { context: { ...context }, target, busy: false };
  registration = next;
  return () => {
    if (registration === next) registration = undefined;
  };
}

export function isMaskingRuleDraftTargetReady(context: TargetContext): boolean {
  return !!matchingTarget(context);
}

export async function getMaskingRuleEditorSnapshot(
  context: TargetContext
): Promise<MaskingRuleEditorSnapshot | undefined> {
  const current = matchingTarget(context);
  if (!current || current.busy) return undefined;
  const snapshot = await current.target.getSnapshot();
  return matchingTarget(context) === current ? snapshot : undefined;
}

async function withTarget(
  context: TargetContext,
  run: (target: MaskingRuleDraftTarget) => Promise<void>
): Promise<void> {
  const current = matchingTarget(context);
  if (!current) throw new Error("The masking rule editor is not available.");
  if (current.busy) throw new Error("The masking rule editor is busy.");
  current.busy = true;
  try {
    await run(current.target);
  } finally {
    current.busy = false;
  }
}

export async function openMaskingRuleDraftInEditor(
  draft: MaskingRuleDraft,
  context: MaskingRuleDraftContext
): Promise<void> {
  await withTarget(context, (target) => target.open(draft, context.chatId));
}

export async function openMaskingRuleDraftsInEditor(
  drafts: MaskingRuleDraft[],
  context: MaskingRuleDraftContext
): Promise<void> {
  if (drafts.length < 2 || drafts.length > 10)
    throw new Error("A batch requires between 2 and 10 masking rules.");
  await withTarget(context, async (target) => {
    if (!target.openMany)
      throw new Error("The masking rule draft queue is unavailable.");
    await target.openMany(drafts, context.chatId);
  });
}

export async function applyMaskingRuleDraftInEditor(
  draft: MaskingRuleDraft,
  expected: MaskingRuleEditorSnapshot,
  context: MaskingRuleDraftContext
): Promise<void> {
  if (expected.mode === "edit")
    throw new Error("Existing masking rule editors are review-only.");
  await withTarget(context, (target) =>
    target.apply(draft, expected, context.chatId)
  );
}
