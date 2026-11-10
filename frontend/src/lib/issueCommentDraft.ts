import { v4 as uuidv4 } from "uuid";
import type { RouteTarget } from "@/app/router";
import { PROJECT_V1_ROUTE_ISSUE_DETAIL } from "@/app/router/handles";
import { useAppStore } from "@/stores/app";
import { hasProjectPermissionV2 } from "@/utils/iam/permission";

const MAX_COMMENT_LENGTH = 16_000;
const PREPARED_DRAFT_PREFIX = "bb.agent.issue-comment-draft.";
const PREPARED_DRAFT_TTL_MS = 30 * 60 * 1000;
const ISSUE_NAME = /^projects\/([A-Za-z0-9_-]+)\/issues\/([A-Za-z0-9_-]+)$/;

type PreparedDraft = {
  issueName: string;
  comment: string;
  createdAt: number;
};

type DeliveryReceipt = PreparedDraft & { deliveredAt?: number };
const deliveryReceipts = new Map<string, DeliveryReceipt>();

function pruneDeliveryReceipts(): void {
  const now = Date.now();
  for (const [key, receipt] of deliveryReceipts) {
    if (now - receipt.createdAt > PREPARED_DRAFT_TTL_MS) {
      deliveryReceipts.delete(key);
    }
  }
  while (deliveryReceipts.size > 64) {
    deliveryReceipts.delete(deliveryReceipts.keys().next().value!);
  }
}

export function markIssueCommentDraftDelivered(
  key: string,
  issueName: string,
  comment: string
): void {
  pruneDeliveryReceipts();
  const receipt = deliveryReceipts.get(key);
  if (receipt?.issueName === issueName && receipt.comment === comment) {
    receipt.deliveredAt = Date.now();
  }
}

export function wasIssueCommentDraftDelivered(key: string): boolean {
  pruneDeliveryReceipts();
  return Boolean(deliveryReceipts.get(key)?.deliveredAt);
}

export class ExistingIssueCommentDraftError extends Error {
  constructor() {
    super("An unsent comment already exists for this Issue");
    this.name = "ExistingIssueCommentDraftError";
  }
}

export class IssueCommentDraftStorageError extends Error {
  constructor() {
    super("Unable to store the Issue comment draft");
    this.name = "IssueCommentDraftStorageError";
  }
}

export class IssueCommentDraftPermissionError extends Error {
  constructor() {
    super("You do not have permission to comment on this Issue");
    this.name = "IssueCommentDraftPermissionError";
  }
}

export const issueCommentDraftKey = (issueName: string) =>
  `bb.plan-review.draft.${issueName}`;

export const readIssueCommentDraft = (issueName: string): string =>
  localStorage.getItem(issueCommentDraftKey(issueName)) ?? "";

export const persistIssueCommentDraft = (issueName: string, value: string) => {
  const key = issueCommentDraftKey(issueName);
  if (value) {
    localStorage.setItem(key, value);
  } else {
    localStorage.removeItem(key);
  }
};

export function readPreparedIssueCommentDraft(
  key: string,
  issueName: string
): string | undefined {
  if (!key.startsWith(PREPARED_DRAFT_PREFIX)) return undefined;
  try {
    const raw = sessionStorage.getItem(key);
    if (!raw) return undefined;
    const draft = JSON.parse(raw) as Partial<PreparedDraft>;
    if (draft.issueName !== issueName) return undefined;
    if (
      typeof draft.comment !== "string" ||
      typeof draft.createdAt !== "number" ||
      Date.now() - draft.createdAt > PREPARED_DRAFT_TTL_MS
    ) {
      return undefined;
    }
    return draft.comment;
  } catch {
    return undefined;
  }
}

export function consumePreparedIssueCommentDraft(
  key: string,
  issueName: string
): string | undefined {
  const comment = readPreparedIssueCommentDraft(key, issueName);
  if (comment) {
    try {
      sessionStorage.removeItem(key);
    } catch {
      // The applied comment is already in the editor; stale temporary storage
      // can be ignored if the browser declines its removal.
    }
  }
  return comment;
}

export function applyPreparedIssueCommentDraft(
  key: string,
  issueName: string
): string | undefined {
  if (readIssueCommentDraft(issueName)) return undefined;
  const comment = readPreparedIssueCommentDraft(key, issueName);
  if (!comment) return undefined;
  persistIssueCommentDraft(issueName, comment);
  consumePreparedIssueCommentDraft(key, issueName);
  return comment;
}

// Ordinary editing must not depend on browser storage. A prepared draft stays
// in session storage until its strict application succeeds.
export function restoreIssueCommentDraft(
  issueName: string,
  preparedKey?: string
): string {
  let draft: string;
  try {
    draft = readIssueCommentDraft(issueName);
  } catch {
    return "";
  }
  if (draft || !preparedKey) return draft;
  try {
    return applyPreparedIssueCommentDraft(preparedKey, issueName) ?? "";
  } catch {
    return readPreparedIssueCommentDraft(preparedKey, issueName) ?? "";
  }
}

export function persistIssueCommentDraftBestEffort(
  issueName: string,
  value: string
): void {
  try {
    persistIssueCommentDraft(issueName, value);
  } catch {
    // The current editor still holds the draft in memory.
  }
}

function lacksKnownCommentPermission(projectName: string): boolean {
  const state = useAppStore.getState();
  const project = state.projectsByName?.[projectName];
  if (
    project?.name !== projectName ||
    !state.projectPoliciesByName?.[projectName] ||
    !state.workspacePolicy ||
    !state.currentUser?.name ||
    !state.roles?.length
  ) {
    return false;
  }
  try {
    return !hasProjectPermissionV2(project, "bb.issueComments.create");
  } catch {
    return false;
  }
}

export function prepareIssueCommentDraftRoute(
  issueName: string,
  comment: string
): Exclude<RouteTarget, string> {
  const match = ISSUE_NAME.exec(issueName);
  const text = comment.trim();
  if (
    !match ||
    match[1] === "-" ||
    match[2] === "-" ||
    !text ||
    text.length > MAX_COMMENT_LENGTH
  ) {
    throw new Error("Invalid Issue comment draft");
  }
  if (lacksKnownCommentPermission(`projects/${match[1]}`)) {
    throw new IssueCommentDraftPermissionError();
  }

  try {
    if (readIssueCommentDraft(issueName)) {
      throw new ExistingIssueCommentDraftError();
    }
    const key = `${PREPARED_DRAFT_PREFIX}${uuidv4()}`;
    sessionStorage.setItem(
      key,
      JSON.stringify({ issueName, comment: text, createdAt: Date.now() })
    );
    pruneDeliveryReceipts();
    deliveryReceipts.set(key, {
      issueName,
      comment: text,
      createdAt: Date.now(),
    });
    return {
      name: PROJECT_V1_ROUTE_ISSUE_DETAIL,
      params: { projectId: match[1], issueId: match[2] },
      query: { phase: "review", agentCommentDraft: key },
      hash: "#issue-comment-editor",
    };
  } catch (error) {
    if (error instanceof ExistingIssueCommentDraftError) throw error;
    throw new IssueCommentDraftStorageError();
  }
}
