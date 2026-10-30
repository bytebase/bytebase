import { v4 as uuidv4 } from "uuid";
import type { Permission } from "@/types";

export const ACCESS_GRANT_DRAFT_QUERY_KEY = "accessGrantDraftKey";

export type AccessGrantDraft = {
  kind?: "accessGrant";
  project: string;
  sourceDatabase: string;
  targets: string[];
  statement: string;
  schema?: string;
  container?: string;
};

export type RequestRoleDraft = {
  kind: "requestRole";
  project: string;
  sourceDatabase: string;
  targets: string[];
  resources: string[];
  statement: string;
  requiredPermissions: Permission[];
};

export type DeniedQueryRequestDraft = AccessGrantDraft | RequestRoleDraft;

const STORAGE_PREFIX = "bb.sql-editor.access-grant-draft.";
const TTL_MS = 10 * 60 * 1000;
const MAX_DRAFTS = 20;
const replayedDrafts = new Map<
  string,
  { createdAt: number; draft: DeniedQueryRequestDraft }
>();
const deliveredDrafts = new Map<string, number>();

const parseStoredDraft = (
  raw: string
): { createdAt: number; draft: DeniedQueryRequestDraft } | undefined => {
  try {
    const stored: unknown = JSON.parse(raw);
    if (!stored || typeof stored !== "object" || Array.isArray(stored)) {
      return undefined;
    }
    const { createdAt, draft } = stored as Record<string, unknown>;
    if (
      typeof createdAt !== "number" ||
      !draft ||
      typeof draft !== "object" ||
      Array.isArray(draft)
    ) {
      return undefined;
    }
    const fields = draft as Record<string, unknown>;
    if (
      typeof fields.project !== "string" ||
      typeof fields.sourceDatabase !== "string" ||
      typeof fields.statement !== "string" ||
      !Array.isArray(fields.targets) ||
      fields.targets.length === 0 ||
      !fields.targets.every((target) => typeof target === "string") ||
      (fields.kind !== undefined &&
        fields.kind !== "accessGrant" &&
        fields.kind !== "requestRole") ||
      (fields.schema !== undefined && typeof fields.schema !== "string") ||
      (fields.container !== undefined && typeof fields.container !== "string")
    ) {
      return undefined;
    }
    if (
      fields.kind === "requestRole" &&
      (!Array.isArray(fields.resources) ||
        !fields.resources.length ||
        !fields.resources.every((resource) => typeof resource === "string") ||
        !Array.isArray(fields.requiredPermissions) ||
        !fields.requiredPermissions.length ||
        !fields.requiredPermissions.every(
          (permission) => typeof permission === "string"
        ))
    ) {
      return undefined;
    }
    return { createdAt, draft: fields as DeniedQueryRequestDraft };
  } catch {
    return undefined;
  }
};

const prune = () => {
  const now = Date.now();
  for (const [key, value] of replayedDrafts) {
    if (now - value.createdAt > TTL_MS) replayedDrafts.delete(key);
  }
  for (const [key, deliveredAt] of deliveredDrafts) {
    if (now - deliveredAt > TTL_MS) deliveredDrafts.delete(key);
  }
  const stored: Array<{ key: string; createdAt: number }> = [];
  for (let i = sessionStorage.length - 1; i >= 0; i--) {
    const key = sessionStorage.key(i);
    if (!key?.startsWith(STORAGE_PREFIX)) continue;
    const value = parseStoredDraft(sessionStorage.getItem(key) ?? "");
    if (!value || now - value.createdAt > TTL_MS) {
      sessionStorage.removeItem(key);
    } else {
      stored.push({ key, createdAt: value.createdAt });
    }
  }
  stored
    .sort((a, b) => a.createdAt - b.createdAt)
    .slice(0, Math.max(0, stored.length - MAX_DRAFTS))
    .forEach(({ key }) => sessionStorage.removeItem(key));
};

export function saveAccessGrantDraft(draft: DeniedQueryRequestDraft): string {
  prune();
  const key = `${STORAGE_PREFIX}${uuidv4()}`;
  sessionStorage.setItem(key, JSON.stringify({ createdAt: Date.now(), draft }));
  return key;
}

export function consumeAccessGrantDraft(
  key: string
): DeniedQueryRequestDraft | undefined {
  if (!key.startsWith(STORAGE_PREFIX)) return undefined;
  prune();
  const replayed = replayedDrafts.get(key);
  if (replayed) return replayed.draft;
  try {
    const stored = parseStoredDraft(sessionStorage.getItem(key) ?? "");
    if (!stored || Date.now() - stored.createdAt > TTL_MS) return undefined;
    replayedDrafts.set(key, stored);
    return stored.draft;
  } finally {
    sessionStorage.removeItem(key);
  }
}

export function markDeniedQueryRequestDraftDelivered(key: string): void {
  if (replayedDrafts.has(key)) deliveredDrafts.set(key, Date.now());
}

export function wasDeniedQueryRequestDraftDelivered(key: string): boolean {
  const deliveredAt = deliveredDrafts.get(key);
  if (!deliveredAt) return false;
  if (Date.now() - deliveredAt > TTL_MS) {
    deliveredDrafts.delete(key);
    return false;
  }
  return true;
}
