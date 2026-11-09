import { create } from "@bufbuild/protobuf";
import { Code, ConnectError, createContextValues } from "@connectrpc/connect";
import { isEqual } from "lodash-es";
import { v4 as uuidv4 } from "uuid";
import {
  databaseServiceClientConnect,
  orgPolicyServiceClientConnect,
  projectServiceClientConnect,
  userServiceClientConnect,
} from "@/api";
import { silentContextKey } from "@/api/context-key";
import { getMaskingExemptionSelectorConfig } from "@/lib/sensitive-data/maskingExemption";
import {
  buildCELExpr,
  type ConditionGroupExpr,
  ExprType,
  resolveCELExpr,
  type SimpleExpr,
  validateSimpleExpr,
  wrapAsGroup,
} from "@/modules/cel";
import { canRenderAsSelector } from "@/modules/cel/logic/selector";
import { useAppStore } from "@/stores/app";
import { AccountType, getAccountTypeByEmail } from "@/types";
import { State } from "@/types/proto-es/v1/common_pb";
import { GetDatabaseRequestSchema } from "@/types/proto-es/v1/database_service_pb";
import {
  GetPolicyRequestSchema,
  PolicyType,
} from "@/types/proto-es/v1/org_policy_service_pb";
import { GetProjectRequestSchema } from "@/types/proto-es/v1/project_service_pb";
import { GetUserRequestSchema } from "@/types/proto-es/v1/user_service_pb";
import { CEL_ATTRIBUTE_RESOURCE_DATABASE } from "@/utils/cel-attributes";
import {
  batchConvertCELStringToParsedExpr,
  batchConvertParsedExprToCELString,
} from "@/utils/v1/cel";
import { getMaskingDraftCELShape } from "./maskingDraftCEL";
import {
  getMaskingExemptionEditorSnapshot,
  type MaskingExemptionEditorSnapshot,
} from "./maskingExemptionDraftTarget";
import type { MaskingRuleDraftContext } from "./maskingRuleDraft";
import { getMaskingSetup } from "./tools/maskingSetup";

export interface MaskingExemptionDraft {
  project: string;
  scope: { kind: "project" } | { kind: "database"; database: string };
  members: string[];
  condition: string;
  description: string;
  expirationTimestamp: string | null;
}
interface Entry {
  proposal: MaskingExemptionDraft;
  draft: MaskingExemptionDraft;
  context: MaskingRuleDraftContext;
  createdAt: number;
  source?: MaskingExemptionEditorSnapshot;
}
const entries = new Map<string, Entry>();
const projectPattern = /^projects\/(?!-$)[A-Za-z0-9_-]+$/;
const databasePattern =
  /^(?:projects\/[A-Za-z0-9_-]+\/)?instances\/[A-Za-z0-9_-]+\/databases\/[^/]+$/;
const emailPattern = /^[^\s@/:]+@[^\s@/:]+\.[^\s@/:]+$/;
const options = () => ({
  contextValues: createContextValues().set(silentContextKey, true),
});
const bounded = (value: unknown, max: number): value is string =>
  typeof value === "string" && value.length <= max && !value.includes("\0");

function assertContext(context: MaskingRuleDraftContext): void {
  const state = useAppStore.getState();
  if (
    !/^workspaces\/[A-Za-z0-9_-]+$/.test(context.workspaceName) ||
    !bounded(context.chatId, 200) ||
    !context.chatId ||
    !bounded(context.userName, 500) ||
    !context.userName.startsWith("users/") ||
    state.currentUser?.name !== context.userName ||
    state.serverInfo?.workspace !== context.workspaceName
  )
    throw new Error(
      "The masking exemption draft context is no longer current."
    );
}

function validExpiration(value: unknown): value is string {
  if (!bounded(value, 100)) return false;
  const match =
    /^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2}):(\d{2})(?:\.\d+)?(?:Z|[+-]\d{2}:\d{2})$/.exec(
      value
    );
  if (
    !match ||
    !Number.isFinite(Date.parse(value)) ||
    Date.parse(value) <= Date.now()
  )
    return false;
  const [, year, month, day, hour, minute, second] = match.map(Number);
  return (
    month >= 1 &&
    month <= 12 &&
    day >= 1 &&
    day <= new Date(Date.UTC(year, month, 0)).getUTCDate() &&
    hour < 24 &&
    minute < 60 &&
    second < 60
  );
}

function parseProposal(raw: unknown): MaskingExemptionDraft {
  if (!raw || typeof raw !== "object" || Array.isArray(raw))
    throw new Error("A masking exemption draft is required.");
  const value = raw as Record<string, unknown>;
  if (
    Object.keys(value).some(
      (key) =>
        ![
          "project",
          "scope",
          "members",
          "condition",
          "description",
          "expirationTimestamp",
        ].includes(key)
    )
  )
    throw new Error("Unexpected masking exemption draft fields.");
  if (!bounded(value.project, 200) || !projectPattern.test(value.project))
    throw new Error("Select an explicit project.");
  const scope = value.scope as Record<string, unknown> | undefined;
  if (
    !scope ||
    typeof scope !== "object" ||
    Array.isArray(scope) ||
    !["project", "database"].includes(String(scope.kind)) ||
    Object.keys(scope).some(
      (key) =>
        !(scope.kind === "project" ? ["kind"] : ["kind", "database"]).includes(
          key
        ) && !(key === "database" && scope.database === "")
    )
  )
    throw new Error("Select an explicit exemption scope.");
  if (
    scope.kind === "database" &&
    (!bounded(scope.database, 1000) ||
      !databasePattern.test(scope.database) ||
      (scope.database.startsWith("projects/") &&
        !scope.database.startsWith(`${value.project}/`)))
  )
    throw new Error("Select an exact database in the selected project.");
  if (
    !Array.isArray(value.members) ||
    value.members.length < 1 ||
    value.members.length > 20 ||
    value.members.some((member) => !bounded(member, 500) || !member)
  )
    throw new Error("Specify exact exemption members.");
  if (!bounded(value.condition, 20000) || !value.condition.trim())
    throw new Error("Specify a selector-compatible exemption condition.");
  if (!bounded(value.description, 2000))
    throw new Error("The exemption description is invalid or too long.");
  if (
    value.expirationTimestamp !== null &&
    !validExpiration(value.expirationTimestamp)
  )
    throw new Error(
      "Specify a future expiration timestamp with timezone, or explicit null for no expiry."
    );
  return {
    project: value.project,
    scope:
      scope.kind === "project"
        ? { kind: "project" }
        : { kind: "database", database: scope.database as string },
    members: [...value.members] as string[],
    condition: value.condition.trim(),
    description: value.description,
    expirationTimestamp:
      value.expirationTimestamp === null
        ? null
        : new Date(value.expirationTimestamp as string).toISOString(),
  };
}

async function resolveCondition(condition: string) {
  const [parsed] = await batchConvertCELStringToParsedExpr([condition]);
  if (!parsed?.exprKind.case)
    throw new Error("The exemption condition could not be parsed.");
  const resolved = resolveCELExpr(parsed);
  const { factorList, factorOperatorOverrideMap } =
    getMaskingExemptionSelectorConfig();
  if (
    !canRenderAsSelector(resolved, factorList, factorOperatorOverrideMap) ||
    !validateSimpleExpr(resolved)
  )
    throw new Error(
      "The condition cannot be represented by the exemption selector."
    );
  const validateDatabase = (expr: SimpleExpr): void => {
    if (expr.type === ExprType.ConditionGroup) {
      expr.args.forEach(validateDatabase);
      return;
    }
    if (
      expr.type !== ExprType.Condition ||
      expr.args[0] !== CEL_ATTRIBUTE_RESOURCE_DATABASE
    )
      return;
    const values =
      expr.operator === "_==_"
        ? [expr.args[1]]
        : expr.operator === "@in" && Array.isArray(expr.args[1])
          ? expr.args[1]
          : [];
    if (
      !values.length ||
      values.some(
        (value) => typeof value !== "string" || !databasePattern.test(value)
      )
    )
      throw new Error(
        "The database selector requires exact database resources."
      );
  };
  validateDatabase(resolved);
  const built = await buildCELExpr(resolved);
  if (
    !built ||
    !isEqual(
      getMaskingDraftCELShape(parsed, "exemption"),
      getMaskingDraftCELShape(built, "exemption")
    )
  )
    throw new Error(
      "The condition cannot be represented by the exemption selector."
    );
  return { group: wrapAsGroup(resolved), built };
}

export async function parseMaskingExemptionDraftCondition(
  condition: string
): Promise<ConditionGroupExpr> {
  if (!bounded(condition, 20000) || !condition.trim())
    throw new Error("The exemption condition is invalid.");
  const { group, built } = await resolveCondition(condition);
  const [text] = await batchConvertParsedExprToCELString([built]);
  if (
    !text ||
    !isEqual(
      getMaskingDraftCELShape(built, "exemption"),
      getMaskingDraftCELShape((await resolveCondition(text)).built, "exemption")
    )
  )
    throw new Error("The exemption condition changed during conversion.");
  return group;
}

function hasDatabaseConjunct(expr: SimpleExpr, database: string): boolean {
  if (expr.type === ExprType.ConditionGroup)
    return (
      expr.operator === "_&&_" &&
      expr.args.some((arg) => hasDatabaseConjunct(arg, database))
    );
  return (
    expr.type === ExprType.Condition &&
    expr.operator === "_==_" &&
    expr.args[0] === CEL_ATTRIBUTE_RESOURCE_DATABASE &&
    expr.args[1] === database
  );
}

type PermissionEvidence = Awaited<
  ReturnType<typeof getMaskingSetup>
>["steps"][number]["readAccess"];
function assertPermissionEvidence(
  evidence: PermissionEvidence | undefined,
  required: PermissionEvidence["requiredPermissions"],
  project: string
): void {
  if (evidence?.status === "allowed") return;
  const status = evidence?.status ?? "unknown";
  const requiredPermissions = evidence?.requiredPermissions.length
    ? evidence.requiredPermissions
    : required;
  const missing =
    status === "unknown" || status === "needs-scope"
      ? "unverified"
      : evidence?.missingPermissions.join(", ") || "none reported";
  throw new Error(
    `Masking exemption permission check failed. Permission status: ${status}. Scope: ${project}. Required permissions: ${requiredPermissions.join(", ")}. Missing permissions: ${missing}.`
  );
}

export async function validateMaskingExemptionDraft(
  proposal: MaskingExemptionDraft,
  context: MaskingRuleDraftContext
): Promise<MaskingExemptionDraft> {
  assertContext(context);
  proposal = parseProposal(proposal);
  const setup = await getMaskingSetup({
    workspaceName: context.workspaceName,
    projectName: proposal.project,
  });
  assertContext(context);
  if (setup.features.masking !== "available")
    throw new Error("The masking feature is unavailable or unverified.");
  const step = setup.steps.find((step) => step.id === "masking-exemptions");
  assertPermissionEvidence(
    step?.readAccess,
    [
      "bb.projects.get",
      "bb.databases.get",
      "bb.policies.getMaskingExemptionPolicy",
    ],
    proposal.project
  );
  assertPermissionEvidence(
    step?.configure,
    [
      "bb.policies.createMaskingExemptionPolicy",
      "bb.policies.updateMaskingExemptionPolicy",
      "bb.policies.getMaskingExemptionPolicy",
      "bb.databases.list",
      "bb.databaseCatalogs.get",
    ],
    proposal.project
  );
  const observed = await projectServiceClientConnect.getProject(
    create(GetProjectRequestSchema, { name: proposal.project }),
    options()
  );
  if (observed.name !== proposal.project || observed.state !== State.ACTIVE)
    throw new Error("The selected project is unavailable or inactive.");
  try {
    const policy = await orgPolicyServiceClientConnect.getPolicy(
      create(GetPolicyRequestSchema, {
        name: `${proposal.project}/policies/${PolicyType[PolicyType.MASKING_EXEMPTION]}`,
      }),
      options()
    );
    if (policy.policy.case !== "maskingExemptionPolicy")
      throw new Error("The masking exemption policy could not be verified.");
  } catch (error) {
    if (ConnectError.from(error).code !== Code.NotFound) throw error;
  }
  const members: string[] = [];
  const currentEmail = useAppStore.getState().currentUser?.email;
  for (const member of proposal.members) {
    const requested =
      member === "self"
        ? currentEmail
          ? `users/${currentEmail}`
          : context.userName
        : member.startsWith("user:")
          ? `users/${member.slice(5)}`
          : member.startsWith("users/")
            ? member
            : `users/${member}`;
    const identifier = requested.slice(6);
    if (
      !requested.startsWith("users/") ||
      (!emailPattern.test(identifier) && !/^\d+$/.test(identifier)) ||
      (emailPattern.test(identifier) &&
        getAccountTypeByEmail(identifier) !== AccountType.USER)
    )
      throw new Error(
        "Only exact verified user members are supported; groups and other identities are not supported."
      );
    const user = await userServiceClientConnect.getUser(
      create(GetUserRequestSchema, { name: requested }),
      options()
    );
    if (
      user.state !== State.ACTIVE ||
      !emailPattern.test(user.email) ||
      getAccountTypeByEmail(user.email) !== AccountType.USER ||
      (emailPattern.test(identifier) && user.email !== identifier) ||
      (member === "self" && user.email !== currentEmail)
    )
      throw new Error(
        "The exemption member is unavailable, inactive, or does not match the requested user."
      );
    members.push(`user:${user.email}`);
  }
  const group =
    proposal.condition === "true"
      ? undefined
      : await parseMaskingExemptionDraftCondition(proposal.condition);
  const databaseNames = new Set<string>();
  const collect = (expr: SimpleExpr): void => {
    if (expr.type === ExprType.ConditionGroup) expr.args.forEach(collect);
    else if (
      expr.type === ExprType.Condition &&
      expr.args[0] === CEL_ATTRIBUTE_RESOURCE_DATABASE
    )
      for (const value of Array.isArray(expr.args[1])
        ? expr.args[1]
        : [expr.args[1]])
        databaseNames.add(value as string);
  };
  if (group) collect(group);
  if (proposal.scope.kind === "database")
    databaseNames.add(proposal.scope.database);
  for (const name of databaseNames) {
    const database = await databaseServiceClientConnect.getDatabase(
      create(GetDatabaseRequestSchema, { name }),
      options()
    );
    if (
      database.name !== name ||
      database.project !== proposal.project ||
      database.state !== State.ACTIVE
    )
      throw new Error(
        "The selected database is unavailable, inactive, or outside the project."
      );
  }
  if (!group && proposal.scope.kind === "project") {
    assertContext(context);
    return { ...proposal, members: [...new Set(members)], condition: "true" };
  }
  const scoped: ConditionGroupExpr =
    proposal.scope.kind === "database" &&
    (!group || !hasDatabaseConjunct(group, proposal.scope.database))
      ? {
          type: ExprType.ConditionGroup,
          operator: "_&&_",
          args: [
            {
              type: ExprType.Condition,
              operator: "_==_",
              args: [CEL_ATTRIBUTE_RESOURCE_DATABASE, proposal.scope.database],
            },
            ...(group ? [group] : []),
          ],
        }
      : group!;
  const built = await buildCELExpr(scoped);
  if (!built) throw new Error("The scoped exemption could not be built.");
  const [condition] = await batchConvertParsedExprToCELString([built]);
  if (
    !condition ||
    !bounded(condition, 20000) ||
    !isEqual(
      getMaskingDraftCELShape(built, "exemption"),
      getMaskingDraftCELShape(
        (await resolveCondition(condition)).built,
        "exemption"
      )
    )
  )
    throw new Error("The scoped exemption changed during conversion.");
  assertContext(context);
  return { ...proposal, members: [...new Set(members)], condition };
}

function getEntry(
  id: string,
  context: MaskingRuleDraftContext
): Entry | undefined {
  for (const [id, entry] of entries)
    if (Date.now() - entry.createdAt >= 30 * 60 * 1000) entries.delete(id);
  const entry = entries.get(id);
  if (!entry || !isEqual(entry.context, context)) return undefined;
  try {
    assertContext(context);
    return entry;
  } catch {
    return undefined;
  }
}

export async function prepareMaskingExemptionDraft(
  raw: unknown,
  context: MaskingRuleDraftContext,
  options?: {
    requireSourceMatch: true;
    source?: MaskingExemptionEditorSnapshot;
  }
): Promise<{ draftId: string; draft: MaskingExemptionDraft }> {
  const proposal = parseProposal(raw);
  assertContext(context);
  const source = structuredClone(
    await getMaskingExemptionEditorSnapshot(context)
  );
  if (
    (options?.requireSourceMatch && !isEqual(source, options.source)) ||
    (source &&
      (source.project !== proposal.project ||
        (source.chatId && source.chatId !== context.chatId)))
  )
    throw new Error(
      "The masking exemption editor has changed or belongs to another project or chat."
    );
  const draft = await validateMaskingExemptionDraft(proposal, context);
  if (!isEqual(source, await getMaskingExemptionEditorSnapshot(context)))
    throw new Error(
      "The masking exemption draft has changed. Read the current form again."
    );
  assertContext(context);
  const draftId = uuidv4();
  if (entries.size >= 100) entries.delete(entries.keys().next().value!);
  entries.set(draftId, {
    proposal,
    draft,
    context: { ...context },
    source,
    createdAt: Date.now(),
  });
  return { draftId, draft: structuredClone(draft) };
}
export function getMaskingExemptionDraft(
  id: string,
  context: MaskingRuleDraftContext
): MaskingExemptionDraft | undefined {
  const entry = getEntry(id, context);
  return entry ? structuredClone(entry.draft) : undefined;
}
export function getMaskingExemptionDraftSource(
  id: string,
  context: MaskingRuleDraftContext
): MaskingExemptionEditorSnapshot | undefined {
  return structuredClone(getEntry(id, context)?.source);
}
export async function revalidateMaskingExemptionDraft(
  id: string,
  context: MaskingRuleDraftContext
): Promise<MaskingExemptionDraft> {
  const entry = getEntry(id, context);
  if (!entry)
    throw new Error(
      "The masking exemption draft expired or is no longer available."
    );
  const subscription = await useAppStore.getState().refreshSubscription();
  if (!subscription)
    throw new Error("The current subscription could not be verified.");
  const draft = await validateMaskingExemptionDraft(entry.proposal, context);
  if (getEntry(id, context) !== entry)
    throw new Error("The masking exemption draft is no longer current.");
  entry.draft = draft;
  return structuredClone(draft);
}
export function consumeMaskingExemptionDraft(
  id: string,
  context: MaskingRuleDraftContext
): void {
  if (getEntry(id, context)) entries.delete(id);
}
