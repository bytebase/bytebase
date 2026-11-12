import { create } from "@bufbuild/protobuf";
import { Code, ConnectError, createContextValues } from "@connectrpc/connect";
import { isEqual } from "lodash-es";
import { v4 as uuidv4 } from "uuid";
import {
  databaseServiceClientConnect,
  orgPolicyServiceClientConnect,
  projectServiceClientConnect,
} from "@/api";
import { silentContextKey } from "@/api/context-key";
import {
  buildCELExpr,
  type ConditionGroupExpr,
  ExprType,
  type Factor,
  resolveCELExpr,
  type SimpleExpr,
  type StringFactor,
  validateSimpleExpr,
  wrapAsGroup,
} from "@/modules/cel";
import { canRenderAsSelector } from "@/modules/cel/logic/selector";
import { useAppStore } from "@/stores/app";
import { State } from "@/types/proto-es/v1/common_pb";
import { GetDatabaseRequestSchema } from "@/types/proto-es/v1/database_service_pb";
import {
  GetPolicyRequestSchema,
  PolicyType,
} from "@/types/proto-es/v1/org_policy_service_pb";
import { GetProjectRequestSchema } from "@/types/proto-es/v1/project_service_pb";
import { getSemanticTypeListWithBuiltins } from "@/types/semanticTypes";
import {
  CEL_ATTRIBUTE_RESOURCE_CLASSIFICATION_LEVEL,
  CEL_ATTRIBUTE_RESOURCE_COLUMN_NAME,
  CEL_ATTRIBUTE_RESOURCE_DATABASE_NAME,
  CEL_ATTRIBUTE_RESOURCE_ENVIRONMENT_ID,
  CEL_ATTRIBUTE_RESOURCE_INSTANCE_ID,
  CEL_ATTRIBUTE_RESOURCE_PROJECT_ID,
  CEL_ATTRIBUTE_RESOURCE_TABLE_NAME,
} from "@/utils/cel-attributes";
import {
  batchConvertCELStringToParsedExpr,
  batchConvertParsedExprToCELString,
} from "@/utils/v1/cel";
import { getMaskingDraftCELShape } from "./maskingDraftCEL";
import {
  getMaskingRuleEditorSnapshot,
  type MaskingRuleEditorSnapshot,
} from "./maskingRuleDraftTarget";
import { readSemanticTypeSetting } from "./semanticTypeSetting";
import { getMaskingSetup } from "./tools/maskingSetup";

export type MaskingRuleDraftScope =
  | { kind: "workspace" }
  | { kind: "project"; project: string }
  | { kind: "database"; database: string; project: string };

export interface MaskingRuleDraft {
  title: string;
  condition: string;
  semanticType: string;
  scope: MaskingRuleDraftScope;
}

export interface MaskingRuleDraftContext {
  workspaceName: string;
  userName: string;
  chatId: string;
}

interface DraftEntry {
  proposals: MaskingRuleDraft[];
  drafts: MaskingRuleDraft[];
  context: MaskingRuleDraftContext;
  createdAt: number;
  source?: MaskingRuleEditorSnapshot;
}

type MaskingPermissionEvidence = Awaited<
  ReturnType<typeof getMaskingSetup>
>["steps"][number]["readAccess"];

const MAX_AGE = 30 * 60 * 1000;
const MAX_DRAFTS = 100;
const MAX_CONDITION_LENGTH = 20_000;
const drafts = new Map<string, DraftEntry>();
const factors: Factor[] = [
  CEL_ATTRIBUTE_RESOURCE_ENVIRONMENT_ID,
  CEL_ATTRIBUTE_RESOURCE_PROJECT_ID,
  CEL_ATTRIBUTE_RESOURCE_INSTANCE_ID,
  CEL_ATTRIBUTE_RESOURCE_DATABASE_NAME,
  CEL_ATTRIBUTE_RESOURCE_TABLE_NAME,
  CEL_ATTRIBUTE_RESOURCE_COLUMN_NAME,
  CEL_ATTRIBUTE_RESOURCE_CLASSIFICATION_LEVEL,
];
const projectPattern = /^projects\/(?!-$)[A-Za-z0-9_-]+$/;
const databasePattern =
  /^(?:projects\/[A-Za-z0-9_-]+\/)?instances\/([A-Za-z0-9_-]+)\/databases\/([^/]+)$/;

function boundedText(value: unknown, limit: number): value is string {
  return (
    typeof value === "string" &&
    value.trim().length > 0 &&
    value.length <= limit &&
    !value.includes("\0")
  );
}

function parseProposal(raw: unknown): MaskingRuleDraft {
  if (!raw || typeof raw !== "object" || Array.isArray(raw))
    throw new Error("A masking rule draft is required.");
  const value = raw as Record<string, unknown>;
  if (
    Object.keys(value).some(
      (key) => !["title", "condition", "semanticType", "scope"].includes(key)
    )
  )
    throw new Error("Unexpected masking rule draft fields.");
  if (
    !boundedText(value.title, 200) ||
    !boundedText(value.condition, MAX_CONDITION_LENGTH) ||
    !boundedText(value.semanticType, 200) ||
    !value.scope ||
    typeof value.scope !== "object" ||
    Array.isArray(value.scope)
  )
    throw new Error("The masking rule draft fields are invalid or too long.");
  const scope = value.scope as Record<string, unknown>;
  const scopeFields =
    scope.kind === "workspace"
      ? ["kind"]
      : scope.kind === "project"
        ? ["kind", "project"]
        : ["kind", "project", "database"];
  if (
    Object.keys(scope).some(
      (key) =>
        !scopeFields.includes(key) &&
        !(scope[key] === "" && ["project", "database"].includes(key))
    )
  )
    throw new Error("Unexpected masking rule scope fields.");
  let validatedScope: MaskingRuleDraftScope;
  if (scope.kind === "workspace") validatedScope = { kind: "workspace" };
  else if (
    (scope.kind === "project" || scope.kind === "database") &&
    typeof scope.project === "string" &&
    projectPattern.test(scope.project)
  ) {
    if (scope.kind === "project")
      validatedScope = { kind: "project", project: scope.project };
    else {
      if (
        !boundedText(scope.database, 1_000) ||
        !databasePattern.test(scope.database) ||
        (scope.database.startsWith("projects/") &&
          !scope.database.startsWith(`${scope.project}/`))
      )
        throw new Error("Select an exact database in the selected project.");
      validatedScope = {
        kind: "database",
        database: scope.database,
        project: scope.project,
      };
    }
  } else throw new Error("Select an exact masking rule scope.");
  return {
    title: value.title.trim(),
    condition: value.condition.trim(),
    semanticType: value.semanticType.trim(),
    scope: validatedScope,
  };
}

function assertContext(context: MaskingRuleDraftContext): void {
  const state = useAppStore.getState();
  if (
    !/^workspaces\/[A-Za-z0-9_-]+$/.test(context.workspaceName) ||
    !boundedText(context.chatId, 200) ||
    !boundedText(context.userName, 500) ||
    state.currentUser?.name !== context.userName ||
    state.serverInfo?.workspace !== context.workspaceName
  )
    throw new Error("The masking rule draft context is no longer current.");
}

function assertPermissionEvidence(
  evidence: MaskingPermissionEvidence | undefined,
  required: MaskingPermissionEvidence["requiredPermissions"],
  scope: string
): void {
  if (evidence?.status === "allowed") return;
  const status = evidence?.status ?? "unknown";
  const requiredPermissions = evidence?.requiredPermissions.length
    ? evidence.requiredPermissions
    : required;
  const missingPermissions =
    status === "unknown" || status === "needs-scope"
      ? "unverified"
      : evidence?.missingPermissions.join(", ") || "none reported";
  throw new Error(
    `Masking rule permission check failed. Permission status: ${status}. Scope: ${evidence?.scope ?? scope}. Required permissions: ${requiredPermissions.join(", ")}. Missing permissions: ${missingPermissions}.`
  );
}

async function resolveCondition(condition: string) {
  if (!boundedText(condition, MAX_CONDITION_LENGTH))
    throw new Error("The masking rule condition is invalid or too long.");
  const [parsed] = await batchConvertCELStringToParsedExpr([condition]);
  if (!parsed?.exprKind.case)
    throw new Error("The masking rule condition could not be parsed.");
  const resolved = resolveCELExpr(parsed);
  if (!canRenderAsSelector(resolved, factors) || !validateSimpleExpr(resolved))
    throw new Error(
      "The condition cannot be represented by the rule selector."
    );
  const built = await buildCELExpr(resolved);
  if (
    !built ||
    !isEqual(
      getMaskingDraftCELShape(parsed, "rule"),
      getMaskingDraftCELShape(built, "rule")
    )
  )
    throw new Error(
      "The condition cannot be represented by the rule selector."
    );
  return { group: wrapAsGroup(resolved), built };
}

export async function parseMaskingRuleDraftCondition(
  condition: string
): Promise<ConditionGroupExpr> {
  const { group, built } = await resolveCondition(condition);
  const [converted] = await batchConvertParsedExprToCELString([built]);
  if (!converted)
    throw new Error("The masking rule condition could not be converted.");
  const roundtrip = await resolveCondition(converted);
  if (
    !isEqual(
      getMaskingDraftCELShape(built, "rule"),
      getMaskingDraftCELShape(roundtrip.built, "rule")
    )
  )
    throw new Error("The masking rule condition changed during conversion.");
  return group;
}

export async function validateMaskingRuleDraft(
  proposal: MaskingRuleDraft,
  context: MaskingRuleDraftContext,
  validation?: { purpose: "review" }
): Promise<MaskingRuleDraft> {
  assertContext(context);
  const scope = proposal.scope;
  const setup = await getMaskingSetup({
    workspaceName: context.workspaceName,
    ...(scope.kind !== "workspace" ? { projectName: scope.project } : {}),
  });
  assertContext(context);
  if (setup.features.masking !== "available")
    throw new Error("The masking feature is unavailable or unverified.");
  const global = setup.steps.find((step) => step.id === "global-masking");
  assertPermissionEvidence(
    global?.readAccess,
    ["bb.policies.getMaskingRulePolicy"],
    context.workspaceName
  );
  assertPermissionEvidence(
    global?.configure,
    ["bb.policies.updateMaskingRulePolicy"],
    context.workspaceName
  );
  assertPermissionEvidence(
    setup.steps.find((step) => step.id === "semantic-types")?.readAccess,
    ["bb.settings.get"],
    context.workspaceName
  );
  const options = {
    contextValues: createContextValues().set(silentContextKey, true),
  };
  try {
    const policy = await orgPolicyServiceClientConnect.getPolicy(
      create(GetPolicyRequestSchema, {
        name: `${context.workspaceName}/policies/${PolicyType[PolicyType.MASKING_RULE]}`,
      }),
      options
    );
    if (policy.policy.case !== "maskingRulePolicy")
      throw new Error("The masking rule policy could not be verified.");
  } catch (error) {
    if (ConnectError.from(error).code !== Code.NotFound) throw error;
    const creation = global?.additionalRequirements?.find(
      (requirement) =>
        requirement.source === "backend" &&
        requirement.permission === "bb.policies.createMaskingRulePolicy"
    );
    assertPermissionEvidence(
      creation && "status" in creation ? creation : undefined,
      ["bb.policies.createMaskingRulePolicy"],
      context.workspaceName
    );
  }
  const { setting } = await readSemanticTypeSetting();
  const semanticTypes = setting.value?.value;
  // Review can retain an unavailable type for repair; preparation and save cannot.
  if (
    semanticTypes?.case !== "semanticType" ||
    (validation?.purpose !== "review" &&
      !getSemanticTypeListWithBuiltins(semanticTypes.value.types).some(
        (type) => type.id === proposal.semanticType
      ))
  )
    throw new Error("The semantic type is not available in current settings.");

  const constraints: SimpleExpr[] = [];
  const equals = (factor: StringFactor, value: string): SimpleExpr => ({
    type: ExprType.Condition,
    operator: "_==_",
    args: [factor, value],
  });
  if (scope.kind !== "workspace") {
    const observed = await projectServiceClientConnect.getProject(
      create(GetProjectRequestSchema, { name: scope.project }),
      options
    );
    if (observed.name !== scope.project || observed.state !== State.ACTIVE)
      throw new Error("The selected project is unavailable or inactive.");
    constraints.push(
      equals(
        CEL_ATTRIBUTE_RESOURCE_PROJECT_ID,
        scope.project.slice("projects/".length)
      )
    );
  }
  if (scope.kind === "database") {
    const observed = await databaseServiceClientConnect.getDatabase(
      create(GetDatabaseRequestSchema, { name: scope.database }),
      options
    );
    if (
      observed.name !== scope.database ||
      observed.project !== scope.project ||
      observed.state !== State.ACTIVE
    )
      throw new Error(
        "The selected database is unavailable, inactive, or outside the project."
      );
    const match = databasePattern.exec(observed.name)!;
    constraints.push(
      equals(CEL_ATTRIBUTE_RESOURCE_INSTANCE_ID, match[1]),
      equals(CEL_ATTRIBUTE_RESOURCE_DATABASE_NAME, match[2])
    );
  }
  const proposed = await parseMaskingRuleDraftCondition(proposal.condition);
  // An equality inside OR cannot replace an independent scope restriction.
  const conjuncts = (expr: SimpleExpr): SimpleExpr[] =>
    expr.type === ExprType.ConditionGroup && expr.operator === "_&&_"
      ? expr.args.flatMap(conjuncts)
      : [expr];
  const scoped: ConditionGroupExpr = {
    type: ExprType.ConditionGroup,
    operator: "_&&_",
    args: [
      ...constraints,
      ...conjuncts(proposed).filter(
        (expr) => !constraints.some((constraint) => isEqual(expr, constraint))
      ),
    ],
  };
  const built = await buildCELExpr(scoped);
  if (!built) throw new Error("The scoped masking rule could not be built.");
  const [condition] = await batchConvertParsedExprToCELString([built]);
  if (!condition)
    throw new Error("The scoped masking rule could not be converted.");
  const roundtrip = await buildCELExpr(
    await parseMaskingRuleDraftCondition(condition)
  );
  if (
    !roundtrip ||
    !isEqual(
      getMaskingDraftCELShape(built, "rule"),
      getMaskingDraftCELShape(roundtrip, "rule")
    )
  )
    throw new Error("The scoped masking rule changed during conversion.");
  assertContext(context);
  return { ...proposal, condition, scope: { ...scope } };
}

function purgeExpiredDrafts(): void {
  for (const [draftId, entry] of drafts) {
    if (Date.now() - entry.createdAt >= MAX_AGE) drafts.delete(draftId);
  }
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

export async function prepareMaskingRuleDraft(
  raw: unknown,
  context: MaskingRuleDraftContext,
  options?: {
    requireSourceMatch: true;
    source?: MaskingRuleEditorSnapshot;
  }
): Promise<{ draftId: string; draft: MaskingRuleDraft }> {
  const proposal = parseProposal(raw);
  assertContext(context);
  const observedSource = await getMaskingRuleEditorSnapshot(context);
  const source = observedSource ? structuredClone(observedSource) : undefined;
  assertContext(context);
  if (options?.requireSourceMatch && !isEqual(source, options.source))
    throw new Error(
      "The masking rule draft has changed. Read current unsaved draft again."
    );
  if (source?.chatId !== undefined && source.chatId !== context.chatId)
    throw new Error("The active masking rule draft belongs to another chat.");
  if (source?.mode === "edit")
    throw new Error("Existing masking rule editors are review-only.");
  const draft = await validateMaskingRuleDraft(proposal, context);
  const currentSource = await getMaskingRuleEditorSnapshot(context);
  assertContext(context);
  if (!isEqual(source, currentSource))
    throw new Error(
      "The masking rule draft has changed. Read current unsaved draft again."
    );
  purgeExpiredDrafts();
  if (drafts.size >= MAX_DRAFTS) drafts.delete(drafts.keys().next().value!);
  const draftId = uuidv4();
  drafts.set(draftId, {
    proposals: [proposal],
    drafts: [draft],
    context: { ...context },
    createdAt: Date.now(),
    source: source ? structuredClone(source) : undefined,
  });
  return { draftId, draft: structuredClone(draft) };
}

export function getMaskingRuleDraft(
  id: string,
  context: MaskingRuleDraftContext
): MaskingRuleDraft | undefined {
  const draft = getEntry(id, context)?.drafts[0];
  return draft ? structuredClone(draft) : undefined;
}

export async function prepareMaskingRuleDrafts(
  raw: unknown,
  context: MaskingRuleDraftContext,
  options?: { requireSourceMatch: true; source?: MaskingRuleEditorSnapshot }
): Promise<{ draftId: string; drafts: MaskingRuleDraft[] }> {
  const value = raw as { rules?: unknown } | null;
  if (
    !value ||
    typeof value !== "object" ||
    Array.isArray(value) ||
    Object.keys(value).some((key) => key !== "rules") ||
    !Array.isArray(value.rules) ||
    value.rules.length < 2 ||
    value.rules.length > 10
  )
    throw new Error("A batch requires between 2 and 10 masking rules.");
  const proposals = value.rules.map(parseProposal);
  if (
    proposals.some((proposal) => !isEqual(proposal.scope, proposals[0].scope))
  )
    throw new Error(
      "All masking rule drafts must use the same confirmed scope."
    );
  assertContext(context);
  const source = await getMaskingRuleEditorSnapshot(context);
  if (source || options?.source)
    throw new Error(
      "Close the active masking rule editor before preparing a batch."
    );
  const batch: MaskingRuleDraft[] = [];
  for (const proposal of proposals)
    batch.push(await validateMaskingRuleDraft(proposal, context));
  assertContext(context);
  if (await getMaskingRuleEditorSnapshot(context))
    throw new Error(
      "The masking rule editor changed while preparing the batch."
    );
  assertContext(context);
  purgeExpiredDrafts();
  if (drafts.size >= MAX_DRAFTS) drafts.delete(drafts.keys().next().value!);
  const draftId = uuidv4();
  drafts.set(draftId, {
    proposals,
    drafts: batch,
    context: { ...context },
    createdAt: Date.now(),
  });
  return { draftId, drafts: structuredClone(batch) };
}

export function getMaskingRuleDrafts(
  id: string,
  context: MaskingRuleDraftContext
): MaskingRuleDraft[] | undefined {
  const entry = getEntry(id, context);
  return entry ? structuredClone(entry.drafts) : undefined;
}

export function getMaskingRuleDraftSource(
  id: string,
  context: MaskingRuleDraftContext
): MaskingRuleEditorSnapshot | undefined {
  const source = getEntry(id, context)?.source;
  return source ? structuredClone(source) : undefined;
}

export async function revalidateMaskingRuleDraft(
  id: string,
  context: MaskingRuleDraftContext
): Promise<MaskingRuleDraft> {
  return (await revalidateMaskingRuleDrafts(id, context))[0];
}

export async function revalidateMaskingRuleDrafts(
  id: string,
  context: MaskingRuleDraftContext
): Promise<MaskingRuleDraft[]> {
  const entry = getEntry(id, context);
  if (!entry)
    throw new Error("The masking rule draft is unavailable or expired.");
  assertContext(context);
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
  const batch: MaskingRuleDraft[] = [];
  for (const proposal of entry.proposals)
    batch.push(await validateMaskingRuleDraft(proposal, context));
  if (getEntry(id, context) !== entry)
    throw new Error("The masking rule draft is unavailable or expired.");
  entry.drafts = batch;
  return structuredClone(batch);
}

export function consumeMaskingRuleDraft(
  id: string,
  context: MaskingRuleDraftContext
): void {
  if (getEntry(id, context)) drafts.delete(id);
}
