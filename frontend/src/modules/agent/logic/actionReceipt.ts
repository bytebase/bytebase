import { fromJson, type JsonObject, toJson } from "@bufbuild/protobuf";
import { isEqual } from "lodash-es";
import { v4 as uuidv4 } from "uuid";
import {
  type AccessGrant,
  AccessGrantSchema,
} from "@/types/proto-es/v1/access_grant_service_pb";
import { IssueSchema } from "@/types/proto-es/v1/issue_service_pb";
import {
  type MaskingExemptionPolicy_Exemption,
  MaskingExemptionPolicy_ExemptionSchema,
  type MaskingRulePolicy_MaskingRule,
  MaskingRulePolicy_MaskingRuleSchema,
} from "@/types/proto-es/v1/org_policy_service_pb";
import {
  accessGrantStatusReading,
  getAccessGrantDisplayStatus,
} from "@/utils/accessGrant";
import type { TimeReading } from "@/utils/datetime";
import {
  createSubmittedAccessGrant,
  decodeSubmittedAccessGrant,
  normalizeSubmittedAccessGrants,
} from "./submittedAccessGrant";
import type {
  AgentActionObservation,
  AgentActionReceipt,
  AgentMessageMetadata,
} from "./types";

const isRecord = (value: unknown): value is Record<string, unknown> =>
  !!value && typeof value === "object" && !Array.isArray(value);
const isTime = (value: unknown): value is number =>
  typeof value === "number" && Number.isFinite(value) && value >= 0;
const policyPattern = {
  "masking-rule": /^workspaces\/[^/\s]+\/policies\/MASKING_RULE$/,
  "masking-exemption":
    /^projects\/(?!-\/)[^/\s]+\/policies\/MASKING_EXEMPTION$/,
};
const grantStatuses = new Set([
  "ACTIVE",
  "PENDING",
  "EXPIRED",
  "REVOKED",
  "REJECTED",
  "CANCELED",
  "UNKNOWN",
  "UNAVAILABLE",
]);

export function createAccessGrantActionReceipt(
  grant: AccessGrant,
  savedAt = Date.now()
): Extract<AgentActionReceipt, { kind: "access-grant" }> | undefined {
  const legacy = createSubmittedAccessGrant(grant, savedAt);
  return legacy
    ? { kind: "access-grant", savedAt, grant: legacy.grant }
    : undefined;
}

export function createMaskingRuleActionReceipt(
  policy: string,
  rule: MaskingRulePolicy_MaskingRule,
  draftId: string,
  savedAt = Date.now()
): Extract<AgentActionReceipt, { kind: "masking-rule" }> | undefined {
  const receipt = normalizeReceipt({
    kind: "masking-rule",
    policy,
    rule: toJson(MaskingRulePolicy_MaskingRuleSchema, rule),
    draftId,
    savedAt,
  });
  return receipt?.kind === "masking-rule" ? receipt : undefined;
}

export function createMaskingExemptionActionReceipt(
  policy: string,
  exemption: MaskingExemptionPolicy_Exemption,
  draftId: string,
  savedAt = Date.now()
): Extract<AgentActionReceipt, { kind: "masking-exemption" }> | undefined {
  const receipt = normalizeReceipt({
    kind: "masking-exemption",
    policy,
    saveId: uuidv4(),
    exemption: toJson(MaskingExemptionPolicy_ExemptionSchema, exemption),
    draftId,
    savedAt,
  });
  return receipt?.kind === "masking-exemption" ? receipt : undefined;
}

export const decodeAccessGrantActionReceipt = (
  receipt: Extract<AgentActionReceipt, { kind: "access-grant" }>
) =>
  decodeSubmittedAccessGrant({
    grant: receipt.grant,
    submittedAt: receipt.savedAt,
  });

function normalizeObservation(
  raw: unknown,
  receipt: AgentActionReceipt
): AgentActionObservation | undefined {
  if (
    !isRecord(raw) ||
    !isTime(raw.observedAt) ||
    raw.observedAt < receipt.savedAt ||
    typeof raw.status !== "string"
  )
    return undefined;
  if (
    raw.revision !== undefined &&
    (!Number.isSafeInteger(raw.revision) || (raw.revision as number) < 1)
  )
    return undefined;
  const timing = {
    observedAt: raw.observedAt,
    ...(raw.revision === undefined ? {} : { revision: raw.revision as number }),
  };
  if (receipt.kind !== "access-grant") {
    const allowed =
      receipt.kind === "masking-rule"
        ? ["SAVED", "CHANGED", "NOT_FOUND", "UNAVAILABLE"]
        : ["SAVED", "MATCHING", "NOT_FOUND", "UNAVAILABLE"];
    return allowed.includes(raw.status)
      ? { status: raw.status as AgentActionObservation["status"], ...timing }
      : undefined;
  }
  if (!grantStatuses.has(raw.status)) return undefined;
  if (raw.status === "UNAVAILABLE") return { status: "UNAVAILABLE", ...timing };
  const saved = decodeAccessGrantActionReceipt(receipt)!;
  const grant = decodeSubmittedAccessGrant({
    grant: raw.grant,
    submittedAt: raw.observedAt,
  });
  if (!grant || grant.name !== saved.name || grant.creator !== saved.creator)
    return undefined;
  let issue: JsonObject | undefined;
  if (raw.issue !== undefined) {
    try {
      const decoded = fromJson(IssueSchema, raw.issue as JsonObject);
      if (!grant.issue || decoded.name !== grant.issue) return undefined;
      issue = toJson(IssueSchema, decoded) as JsonObject;
    } catch {
      return undefined;
    }
  }
  return {
    status: raw.status as AgentActionObservation["status"],
    ...timing,
    grant: toJson(AccessGrantSchema, grant) as JsonObject,
    ...(issue ? { issue } : {}),
  };
}

export const isActionObservationNewer = (
  incoming: AgentActionObservation,
  previous: AgentActionObservation | undefined
) =>
  !previous ||
  (incoming.revision ?? 0) > (previous.revision ?? 0) ||
  ((incoming.revision ?? 0) === (previous.revision ?? 0) &&
    incoming.observedAt > previous.observedAt);

function normalizeReceipt(raw: unknown): AgentActionReceipt | undefined {
  if (!isRecord(raw) || !isTime(raw.savedAt)) return undefined;
  let receipt: AgentActionReceipt;
  try {
    if (raw.kind === "access-grant") {
      const grant = decodeSubmittedAccessGrant({
        grant: raw.grant,
        submittedAt: raw.savedAt,
      });
      if (!grant) return undefined;
      receipt = {
        kind: raw.kind,
        savedAt: raw.savedAt,
        grant: toJson(AccessGrantSchema, grant) as JsonObject,
      };
    } else if (
      (raw.kind === "masking-rule" || raw.kind === "masking-exemption") &&
      typeof raw.draftId === "string" &&
      raw.draftId.trim() &&
      typeof raw.policy === "string" &&
      policyPattern[raw.kind].test(raw.policy)
    ) {
      if (raw.kind === "masking-rule") {
        const rule = fromJson(
          MaskingRulePolicy_MaskingRuleSchema,
          raw.rule as JsonObject
        );
        if (
          !rule.id.trim() ||
          !rule.semanticType.trim() ||
          !rule.condition?.expression.trim()
        )
          return undefined;
        receipt = {
          kind: raw.kind,
          savedAt: raw.savedAt,
          draftId: raw.draftId,
          policy: raw.policy,
          rule: toJson(MaskingRulePolicy_MaskingRuleSchema, rule) as JsonObject,
        };
      } else {
        if (
          raw.saveId !== undefined &&
          (typeof raw.saveId !== "string" || !raw.saveId.trim())
        )
          return undefined;
        const exemption = fromJson(
          MaskingExemptionPolicy_ExemptionSchema,
          raw.exemption as JsonObject
        );
        if (
          !exemption.members.length ||
          exemption.members.some((member) => !member.trim())
        )
          return undefined;
        receipt = {
          kind: raw.kind,
          savedAt: raw.savedAt,
          draftId: raw.draftId,
          ...(raw.saveId === undefined ? {} : { saveId: raw.saveId as string }),
          policy: raw.policy,
          exemption: toJson(
            MaskingExemptionPolicy_ExemptionSchema,
            exemption
          ) as JsonObject,
        };
      }
    } else return undefined;
  } catch {
    return undefined;
  }
  const observation = normalizeObservation(raw.observation, receipt);
  return observation ? { ...receipt, observation } : receipt;
}

export function actionReceiptKey(receipt: AgentActionReceipt): string {
  return receipt.kind === "access-grant"
    ? JSON.stringify([receipt.kind, receipt.grant.name])
    : JSON.stringify([
        receipt.kind,
        receipt.policy,
        receipt.kind === "masking-rule" ? receipt.rule.id : receipt.draftId,
        ...(receipt.kind === "masking-exemption" && receipt.saveId
          ? [receipt.saveId]
          : []),
      ]);
}

export function mergeActionReceipts(
  raw: readonly unknown[]
): AgentActionReceipt[] | undefined {
  const receipts = new Map<string, AgentActionReceipt>();
  for (const value of raw) {
    const receipt = normalizeReceipt(value);
    if (!receipt) continue;
    const key = actionReceiptKey(receipt);
    const previous = receipts.get(key);
    if (!previous) {
      receipts.set(key, receipt);
      continue;
    }
    const { observation: previousObservation, ...previousSave } = previous;
    const { observation, ...save } = receipt;
    if (
      isEqual(previousSave, save) &&
      observation &&
      isActionObservationNewer(observation, previousObservation)
    ) {
      receipts.set(key, { ...previous, observation });
    }
  }
  return receipts.size ? [...receipts.values()] : undefined;
}

export function normalizeActionReceipts(
  raw: unknown,
  metadata: AgentMessageMetadata | undefined,
  ownerEmail?: string
): AgentActionReceipt[] | undefined {
  const legacy = normalizeSubmittedAccessGrants(
    metadata?.submittedAccessGrants,
    ownerEmail,
    metadata?.accessRequestDraft
  )?.map(({ grant, submittedAt }) => ({
    kind: "access-grant",
    grant,
    savedAt: submittedAt,
  }));
  const receipts = mergeActionReceipts([
    ...(Array.isArray(raw) ? raw : []),
    ...(legacy ?? []),
  ]);
  const valid = receipts?.filter((receipt) => {
    if (receipt.kind === "access-grant") {
      return !!normalizeSubmittedAccessGrants(
        [{ grant: receipt.grant, submittedAt: receipt.savedAt }],
        ownerEmail,
        metadata?.accessRequestDraft
      )?.length;
    }
    const kind =
      receipt.kind === "masking-rule"
        ? "draft-masking-rule"
        : "draft-masking-exemption";
    return (
      Array.isArray(metadata?.actions) &&
      metadata.actions.some(
        (action) =>
          action &&
          typeof action === "object" &&
          action.kind === kind &&
          action.draftId === receipt.draftId
      )
    );
  });
  return valid?.length ? valid : undefined;
}

export function getActionReceiptStatus(
  receipt: AgentActionReceipt
): AgentActionObservation["status"] {
  if (receipt.kind !== "access-grant")
    return receipt.observation?.status ?? "SAVED";
  if (receipt.observation?.status === "UNAVAILABLE") return "UNAVAILABLE";
  const grant = decodeSubmittedAccessGrant({
    grant: receipt.observation?.grant ?? receipt.grant,
    submittedAt: receipt.savedAt,
  })!;
  const issue = receipt.observation?.issue
    ? fromJson(IssueSchema, receipt.observation.issue)
    : undefined;
  return getAccessGrantDisplayStatus(grant, issue);
}

export const actionReceiptStatusReading: TimeReading<
  AgentActionReceipt,
  AgentActionObservation["status"]
> = {
  read: getActionReceiptStatus,
  nextChangeAt: (receipt) => {
    if (
      receipt.kind !== "access-grant" ||
      receipt.observation?.status === "UNAVAILABLE"
    )
      return Number.POSITIVE_INFINITY;
    const grant = decodeSubmittedAccessGrant({
      grant: receipt.observation?.grant ?? receipt.grant,
      submittedAt: receipt.savedAt,
    })!;
    return accessGrantStatusReading.nextChangeAt({ grant });
  },
};
