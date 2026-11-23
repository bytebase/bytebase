import { fromJson, type JsonObject, toJson } from "@bufbuild/protobuf";
import type { AccessGrantDraft } from "@/modules/sql-editor/model/accessGrantDraft";
import {
  type AccessGrant,
  AccessGrantSchema,
} from "@/types/proto-es/v1/access_grant_service_pb";
import type { AgentSubmittedAccessGrant } from "./types";

const isRecord = (value: unknown): value is Record<string, unknown> =>
  typeof value === "object" && value !== null && !Array.isArray(value);
const PROJECT = /^projects\/([^/\s]+)$/;
const GRANT = /^(projects\/[^/\s]+)\/accessGrants\/([^/\s]+)$/;
const ISSUE = /^(projects\/[^/\s]+)\/issues\/([^/\s]+)$/;
const DATABASE =
  /^(?:(projects\/[^/\s]+)\/)?instances\/([^/\s]+)\/databases\/([^/]+)$/;
const hasConcreteTokens = (name: string) => !name.split("/").includes("-");
const isDatabase = (name: unknown, project: string) => {
  if (typeof name !== "string" || !hasConcreteTokens(name)) return false;
  const match = DATABASE.exec(name);
  return !!match && (!match[1] || match[1] === project);
};

export const isAccessGrantSourceDraft = (
  value: unknown
): value is AccessGrantDraft =>
  isRecord(value) &&
  (value.kind === undefined || value.kind === "accessGrant") &&
  typeof value.project === "string" &&
  PROJECT.test(value.project) &&
  hasConcreteTokens(value.project) &&
  isDatabase(value.sourceDatabase, value.project) &&
  Array.isArray(value.targets) &&
  value.targets.length > 0 &&
  value.targets.every((target) =>
    isDatabase(target, value.project as string)
  ) &&
  typeof value.statement === "string" &&
  !!value.statement.trim();

export function decodeSubmittedAccessGrant(
  receipt: unknown
): AccessGrant | undefined {
  if (
    !isRecord(receipt) ||
    !isRecord(receipt.grant) ||
    typeof receipt.submittedAt !== "number" ||
    !Number.isFinite(receipt.submittedAt) ||
    receipt.submittedAt < 0
  )
    return undefined;
  try {
    const grant = fromJson(AccessGrantSchema, receipt.grant as JsonObject);
    const name = GRANT.exec(grant.name);
    if (
      !name ||
      !hasConcreteTokens(grant.name) ||
      !/^users\/[^/\s]+$/.test(grant.creator) ||
      !hasConcreteTokens(grant.creator) ||
      !grant.query.trim() ||
      !grant.targets.length ||
      !grant.targets.every((target) => isDatabase(target, name[1])) ||
      (grant.issue &&
        (ISSUE.exec(grant.issue)?.[1] !== name[1] ||
          !hasConcreteTokens(grant.issue)))
    )
      return undefined;
    return grant;
  } catch {
    return undefined;
  }
}

export function createSubmittedAccessGrant(
  grant: AccessGrant,
  submittedAt = Date.now()
): AgentSubmittedAccessGrant | undefined {
  try {
    const receipt = {
      grant: toJson(AccessGrantSchema, grant) as JsonObject,
      submittedAt,
    };
    return decodeSubmittedAccessGrant(receipt) ? receipt : undefined;
  } catch {
    return undefined;
  }
}

export function normalizeSubmittedAccessGrants(
  raw: unknown,
  ownerEmail: string | undefined,
  sourceDraft: unknown
): AgentSubmittedAccessGrant[] | undefined {
  if (!Array.isArray(raw) || !isAccessGrantSourceDraft(sourceDraft))
    return undefined;
  const receipts = new Map<string, AgentSubmittedAccessGrant>();
  for (const value of raw) {
    const grant = decodeSubmittedAccessGrant(value);
    if (
      !grant ||
      !grant.name.startsWith(`${sourceDraft.project}/accessGrants/`) ||
      (ownerEmail && grant.creator !== `users/${ownerEmail}`) ||
      receipts.has(grant.name)
    )
      continue;
    receipts.set(grant.name, {
      grant: toJson(AccessGrantSchema, grant) as JsonObject,
      submittedAt: (value as AgentSubmittedAccessGrant).submittedAt,
    });
  }
  return receipts.size ? [...receipts.values()] : undefined;
}
