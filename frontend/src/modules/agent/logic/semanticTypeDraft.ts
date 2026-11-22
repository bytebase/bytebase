import { create } from "@bufbuild/protobuf";
import { isEqual } from "lodash-es";
import { v4 as uuidv4 } from "uuid";
import { useAppStore } from "@/stores/app";
import {
  type Algorithm,
  Algorithm_InnerOuterMask_MaskType,
  AlgorithmSchema,
  type SemanticTypeSetting_SemanticType,
} from "@/types/proto-es/v1/setting_service_pb";
import type { MaskingRuleDraftContext } from "./maskingRuleDraft";
import {
  getSemanticTypeDraftEditorSnapshot,
  type SemanticTypeDraftEditorSnapshot,
} from "./semanticTypeDraftTarget";
import { readSemanticTypeSetting } from "./semanticTypeSetting";
import { getMaskingSetup } from "./tools/maskingSetup";

export interface SemanticTypeDraft {
  id: string;
  title: string;
  description: string;
  algorithm: Algorithm;
}

export type SemanticTypeDraftAlgorithm =
  | { kind: "full-mask"; substitution: string }
  | {
      kind: "range-mask";
      slices: { start: number; end: number; substitution: string }[];
    }
  | { kind: "md5-mask" }
  | {
      kind: "inner-outer-mask";
      type: "inner" | "outer";
      prefixLen: number;
      suffixLen: number;
      substitution: string;
    };

interface DraftEntry {
  draft: SemanticTypeDraft;
  context: MaskingRuleDraftContext;
  createdAt: number;
  source?: SemanticTypeDraftEditorSnapshot;
}

const MAX_AGE = 30 * 60 * 1000;
const MAX_DRAFTS = 100;
const drafts = new Map<string, DraftEntry>();
const uuidPattern =
  /^[\da-f]{8}-[\da-f]{4}-4[\da-f]{3}-[89ab][\da-f]{3}-[\da-f]{12}$/i;

function record(value: unknown): Record<string, unknown> {
  if (!value || typeof value !== "object" || Array.isArray(value))
    throw new Error("A semantic type draft object is required.");
  return value as Record<string, unknown>;
}

function fields(value: Record<string, unknown>, allowed: string[]): void {
  if (Object.keys(value).some((key) => !allowed.includes(key)))
    throw new Error("Unexpected semantic type draft fields.");
}

function text(value: unknown, limit: number, allowEmpty = false): string {
  if (
    typeof value !== "string" ||
    (!allowEmpty && !value.trim()) ||
    value.length > limit ||
    value.includes("\0")
  )
    throw new Error("The semantic type draft text is invalid or too long.");
  return value;
}

function substitution(value: unknown): string {
  const result = text(value, 16, true);
  if (!result.length || new TextEncoder().encode(result).length > 16)
    throw new Error("A masking substitution must be at most 16 bytes.");
  return result;
}

function index(value: unknown): number {
  if (
    typeof value !== "number" ||
    !Number.isInteger(value) ||
    value < 0 ||
    value > 2147483647
  )
    throw new Error(
      "Masking positions must be nonnegative int32 character indices."
    );
  return value;
}

function parseAlgorithm(raw: unknown): SemanticTypeDraftAlgorithm {
  const value = record(raw);
  switch (value.kind) {
    case "full-mask":
      fields(value, ["kind", "substitution"]);
      return {
        kind: value.kind,
        substitution: substitution(value.substitution),
      };
    case "md5-mask":
      fields(value, ["kind"]);
      return { kind: value.kind };
    case "range-mask": {
      fields(value, ["kind", "slices"]);
      if (
        !Array.isArray(value.slices) ||
        value.slices.length === 0 ||
        value.slices.length > 100
      )
        throw new Error("Range masking requires between 1 and 100 slices.");
      const slices = value.slices.map((rawSlice) => {
        const slice = record(rawSlice);
        fields(slice, ["start", "end", "substitution"]);
        const start = index(slice.start);
        const end = index(slice.end);
        if (start >= end)
          throw new Error("A masking slice start must be less than its end.");
        return { start, end, substitution: substitution(slice.substitution) };
      });
      for (let i = 0; i < slices.length; i++)
        if (
          slices
            .slice(0, i)
            .some(
              (prior) =>
                slices[i].start < prior.end && prior.start < slices[i].end
            )
        )
          throw new Error("Masking slices must not overlap.");
      return { kind: value.kind, slices };
    }
    case "inner-outer-mask": {
      fields(value, ["kind", "type", "prefixLen", "suffixLen", "substitution"]);
      if (value.type !== "inner" && value.type !== "outer")
        throw new Error("Choose inner or outer masking.");
      const prefixLen = index(value.prefixLen);
      const suffixLen = index(value.suffixLen);
      if (prefixLen + suffixLen > 2147483647)
        throw new Error("The combined masking margins exceed the int32 range.");
      return {
        kind: value.kind,
        type: value.type,
        prefixLen,
        suffixLen,
        substitution: substitution(value.substitution),
      };
    }
    default:
      throw new Error("The semantic type masking algorithm is unsupported.");
  }
}

function buildAlgorithm(value: SemanticTypeDraftAlgorithm): Algorithm {
  switch (value.kind) {
    case "full-mask":
      return create(AlgorithmSchema, {
        mask: { case: "fullMask", value: { substitution: value.substitution } },
      });
    case "range-mask":
      return create(AlgorithmSchema, {
        mask: { case: "rangeMask", value: { slices: value.slices } },
      });
    case "md5-mask":
      return create(AlgorithmSchema, {
        mask: { case: "md5Mask", value: { salt: uuidv4() } },
      });
    case "inner-outer-mask":
      return create(AlgorithmSchema, {
        mask: {
          case: "innerOuterMask",
          value: {
            type:
              value.type === "inner"
                ? Algorithm_InnerOuterMask_MaskType.INNER
                : Algorithm_InnerOuterMask_MaskType.OUTER,
            prefixLen: value.prefixLen,
            suffixLen: value.suffixLen,
            substitution: value.substitution,
          },
        },
      });
  }
}

export function getSafeSemanticTypeDraftSummary(draft: SemanticTypeDraft): Omit<
  SemanticTypeDraft,
  "algorithm"
> & {
  algorithm: SemanticTypeDraftAlgorithm;
} {
  const mask = draft.algorithm?.mask;
  let algorithm: SemanticTypeDraftAlgorithm;
  switch (mask?.case) {
    case "fullMask":
      algorithm = { kind: "full-mask", substitution: mask.value.substitution };
      break;
    case "rangeMask":
      algorithm = {
        kind: "range-mask",
        slices: mask.value.slices.map(({ start, end, substitution }) => ({
          start,
          end,
          substitution,
        })),
      };
      break;
    case "md5Mask":
      algorithm = { kind: "md5-mask" };
      break;
    case "innerOuterMask":
      if (
        mask.value.type !== Algorithm_InnerOuterMask_MaskType.INNER &&
        mask.value.type !== Algorithm_InnerOuterMask_MaskType.OUTER
      )
        throw new Error("Choose inner or outer masking.");
      algorithm = {
        kind: "inner-outer-mask",
        type:
          mask.value.type === Algorithm_InnerOuterMask_MaskType.INNER
            ? "inner"
            : "outer",
        prefixLen: mask.value.prefixLen,
        suffixLen: mask.value.suffixLen,
        substitution: mask.value.substitution,
      };
      break;
    default:
      throw new Error("The semantic type masking algorithm is unsupported.");
  }
  return {
    id: draft.id,
    title: draft.title,
    description: draft.description,
    algorithm: parseAlgorithm(algorithm),
  };
}

function assertContext(context: MaskingRuleDraftContext): void {
  const state = useAppStore.getState();
  if (
    !/^workspaces\/[A-Za-z0-9_-]+$/.test(context.workspaceName) ||
    !context.chatId.trim() ||
    context.chatId.length > 200 ||
    context.chatId.includes("\0") ||
    !context.userName.trim() ||
    context.userName.length > 500 ||
    context.userName.includes("\0") ||
    state.currentUser?.name !== context.userName ||
    state.serverInfo?.workspace !== context.workspaceName
  )
    throw new Error("The semantic type draft context is no longer current.");
}

export async function validateSemanticTypeDraft(
  draft: SemanticTypeDraft,
  context: MaskingRuleDraftContext
): Promise<SemanticTypeDraft> {
  return (await validateSemanticTypeDraftForSave(draft, context)).draft;
}

export async function validateSemanticTypeDraftForSave(
  draft: SemanticTypeDraft,
  context: MaskingRuleDraftContext
): Promise<{
  draft: SemanticTypeDraft;
  types: SemanticTypeSetting_SemanticType[];
}> {
  assertContext(context);
  if (!uuidPattern.test(draft.id))
    throw new Error("A new semantic type requires its own generated UUID.");
  text(draft.title, 200);
  text(draft.description, 2000, true);
  getSafeSemanticTypeDraftSummary(draft);
  if (draft.algorithm.mask.case === "md5Mask")
    text(draft.algorithm.mask.value.salt, 1000);
  let subscriptionVerified = false;
  try {
    subscriptionVerified = Boolean(
      await useAppStore.getState().refreshSubscription()
    );
  } catch {
    subscriptionVerified = false;
  }
  assertContext(context);
  if (!subscriptionVerified)
    throw new Error(
      "The masking feature is unverified because its subscription could not be refreshed."
    );
  const setup = await getMaskingSetup({ workspaceName: context.workspaceName });
  assertContext(context);
  if (setup.features.masking !== "available")
    throw new Error("The masking feature is unavailable or unverified.");
  const semanticTypes = setup.steps.find(
    (step) => step.id === "semantic-types"
  );
  for (const evidence of [semanticTypes?.readAccess, semanticTypes?.configure])
    if (evidence?.status !== "allowed")
      throw new Error(
        `Semantic type permission check failed. Permission status: ${evidence?.status ?? "unknown"}. Required permissions: ${evidence?.requiredPermissions.join(", ") || "bb.settings.get, bb.settings.set, bb.policies.update"}.`
      );
  const { setting } = await readSemanticTypeSetting();
  assertContext(context);
  const value = setting.value?.value;
  if (value?.case !== "semanticType")
    throw new Error("The semantic type setting could not be verified.");
  if (value.value.types.some((existing) => existing.id === draft.id))
    throw new Error(
      "The semantic type ID already exists. Prepare a new draft."
    );
  return { draft: structuredClone(draft), types: value.value.types };
}

function purgeExpiredDrafts(): void {
  for (const [key, entry] of drafts)
    if (Date.now() - entry.createdAt >= MAX_AGE) drafts.delete(key);
}

function getEntry(
  id: string,
  context: MaskingRuleDraftContext
): DraftEntry | undefined {
  purgeExpiredDrafts();
  const entry = drafts.get(id);
  if (!entry || !isEqual(entry.context, context)) return undefined;
  try {
    assertContext(context);
  } catch {
    return undefined;
  }
  return entry;
}

function assertSource(
  source: SemanticTypeDraftEditorSnapshot | undefined,
  current: SemanticTypeDraftEditorSnapshot | undefined
): void {
  if (!isEqual(source, current))
    throw new Error("The semantic type page changed. Prepare a new draft.");
  if (current?.busy)
    throw new Error("The semantic type page has unsaved changes or is busy.");
}

export async function prepareSemanticTypeDraft(
  raw: unknown,
  context: MaskingRuleDraftContext
): Promise<{ draftId: string; draft: SemanticTypeDraft }> {
  const value = record(raw);
  fields(value, ["title", "description", "algorithm"]);
  const title = text(value.title, 200).trim();
  const description = text(
    value.description === undefined ? "" : value.description,
    2000,
    true
  );
  const algorithm = buildAlgorithm(parseAlgorithm(value.algorithm));
  assertContext(context);
  const source = await getSemanticTypeDraftEditorSnapshot(context);
  assertSource(source, source);
  const draft = await validateSemanticTypeDraft(
    { id: uuidv4(), title, description, algorithm },
    context
  );
  assertSource(source, await getSemanticTypeDraftEditorSnapshot(context));
  assertContext(context);
  purgeExpiredDrafts();
  if (drafts.size >= MAX_DRAFTS) drafts.delete(drafts.keys().next().value!);
  const draftId = uuidv4();
  drafts.set(draftId, {
    draft,
    context: { ...context },
    createdAt: Date.now(),
    source: source ? structuredClone(source) : undefined,
  });
  return { draftId, draft: structuredClone(draft) };
}

export function getSemanticTypeDraft(
  id: string,
  context: MaskingRuleDraftContext
): SemanticTypeDraft | undefined {
  const draft = getEntry(id, context)?.draft;
  return draft ? structuredClone(draft) : undefined;
}

export function getSemanticTypeDraftSource(
  id: string,
  context: MaskingRuleDraftContext
): SemanticTypeDraftEditorSnapshot | undefined {
  const source = getEntry(id, context)?.source;
  return source ? structuredClone(source) : undefined;
}

export async function revalidateSemanticTypeDraft(
  id: string,
  context: MaskingRuleDraftContext
): Promise<SemanticTypeDraft> {
  const entry = getEntry(id, context);
  if (!entry)
    throw new Error("The semantic type draft is unavailable or expired.");
  assertSource(entry.source, await getSemanticTypeDraftEditorSnapshot(context));
  const draft = await validateSemanticTypeDraft(entry.draft, context);
  assertSource(entry.source, await getSemanticTypeDraftEditorSnapshot(context));
  if (getEntry(id, context) !== entry)
    throw new Error("The semantic type draft is unavailable or expired.");
  return draft;
}

export function consumeSemanticTypeDraft(
  id: string,
  context: MaskingRuleDraftContext
): void {
  if (getEntry(id, context)) drafts.delete(id);
}
