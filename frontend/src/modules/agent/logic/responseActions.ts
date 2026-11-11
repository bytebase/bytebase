import type { Code } from "mdast";
import remarkParse from "remark-parse";
import { unified } from "unified";
import { normalizeAgentHandoff } from "./handoff";
import {
  hasUnboundResourceMention,
  normalizeAgentResourceReference,
} from "./resourceMention";
import type { AgentHandoff, AgentResponseAction } from "./types";

const MAX_ACTIONS = 4;
const MAX_CONTINUE_ACTIONS = MAX_ACTIONS - 1;
const MAX_FOLLOW_UP_LENGTH = 140;
const MAX_DRAFT_SQL_LENGTH = 100_000;
const MAX_COMMENT_LENGTH = 10_000;
const markdown = unified().use(remarkParse);

const isRecord = (value: unknown): value is Record<string, unknown> =>
  typeof value === "object" && value !== null && !Array.isArray(value);

const normalizeOpenTarget = (raw: unknown): AgentHandoff | undefined => {
  if (
    !isRecord(raw) ||
    typeof raw.kind !== "string" ||
    Object.keys(raw).some(
      (key) =>
        key !== "kind" &&
        key !== "workspace" &&
        key !== "project" &&
        key !== "database" &&
        key !== "plan" &&
        key !== "issue" &&
        key !== "task"
    )
  ) {
    return undefined;
  }
  const fieldsByKind: Record<string, readonly string[]> = {
    "open-global-masking": [],
    "open-semantic-types": [],
    "open-data-classification": [],
    "open-project-masking-exemptions": ["project"],
    "open-database-catalog": ["project", "database"],
    "review-workspace-issues": ["workspace"],
    "review-project-issues": ["project"],
    "review-database-changes": ["project"],
    "open-project-instances": ["project"],
    "transfer-project-databases": ["project"],
    "connect-project-instance": ["project"],
    "create-project-database": ["project"],
    "view-failed-tasks": ["plan"],
    "open-issue": ["issue"],
    "review-issue-plan": ["plan"],
    "view-failed-task": ["task"],
  };
  const usedFields = fieldsByKind[raw.kind];
  if (!usedFields) return undefined;

  const optionalFields = [
    "workspace",
    "project",
    "database",
    "plan",
    "issue",
    "task",
  ];
  if (
    optionalFields.some(
      (field) =>
        !usedFields.includes(field) &&
        raw[field] !== undefined &&
        raw[field] !== ""
    )
  ) {
    return undefined;
  }

  return normalizeAgentHandoff({
    kind: raw.kind,
    ...Object.fromEntries(usedFields.map((field) => [field, raw[field]])),
  });
};

export function normalizeAgentResponseActions(
  raw: unknown
): AgentResponseAction[] {
  if (!Array.isArray(raw)) return [];

  let primaryAction:
    | Exclude<AgentResponseAction, { kind: "continue" }>
    | undefined;
  const continueActions: Extract<AgentResponseAction, { kind: "continue" }>[] =
    [];
  const seenPrompts = new Set<string>();
  for (const candidate of raw) {
    if (primaryAction && continueActions.length === MAX_CONTINUE_ACTIONS) break;
    if (!isRecord(candidate)) continue;
    if (candidate.kind === "review-sql-fix") {
      if (
        Object.keys(candidate).every(
          (key) => key === "kind" || key === "draftId"
        ) &&
        typeof candidate.draftId === "string" &&
        /^[A-Za-z0-9_-]{1,100}$/.test(candidate.draftId) &&
        !primaryAction
      ) {
        primaryAction = { kind: "review-sql-fix", draftId: candidate.draftId };
      }
      continue;
    }
    if (
      !isRecord(candidate) ||
      Object.keys(candidate).some(
        (key) =>
          key !== "kind" &&
          key !== "target" &&
          key !== "prompt" &&
          key !== "project" &&
          key !== "database" &&
          key !== "issue" &&
          key !== "comment" &&
          key !== "draftId"
      )
    ) {
      continue;
    }
    if (
      candidate.kind === "draft-masking-rule" ||
      candidate.kind === "draft-masking-exemption" ||
      candidate.kind === "draft-semantic-type"
    ) {
      if (
        Object.entries(candidate).every(
          ([key, value]) =>
            key === "kind" ||
            key === "draftId" ||
            value === undefined ||
            value === "" ||
            (key === "target" &&
              isRecord(value) &&
              Object.keys(value).length === 0)
        ) &&
        typeof candidate.draftId === "string" &&
        /^[A-Za-z0-9_-]{1,100}$/.test(candidate.draftId) &&
        !primaryAction
      ) {
        primaryAction = {
          kind: candidate.kind,
          draftId: candidate.draftId,
        };
      }
      continue;
    }
    if (candidate.kind === "open") {
      const target = normalizeOpenTarget(candidate.target);
      if (target && !primaryAction) {
        primaryAction = { kind: "open", target };
      }
      continue;
    }
    if (candidate.kind === "draft-plan") {
      const project = candidate.project;
      const database = candidate.database;
      const reference = normalizeAgentResourceReference({
        kind: "database",
        name: database,
        project,
        label: "database",
      });
      if (
        reference &&
        !reference.name.split("/").includes("-") &&
        !reference.project.endsWith("/-") &&
        !primaryAction
      ) {
        primaryAction = {
          kind: "draft-plan",
          project: reference.project,
          database: reference.name,
        };
      }
      continue;
    }
    if (candidate.kind === "draft-issue-comment") {
      const target = normalizeAgentHandoff({
        kind: "open-issue",
        issue: candidate.issue,
      });
      const comment =
        typeof candidate.comment === "string" ? candidate.comment.trim() : "";
      if (
        target?.kind === "open-issue" &&
        comment &&
        comment.length <= MAX_COMMENT_LENGTH &&
        !comment.includes("\0") &&
        !primaryAction
      ) {
        primaryAction = {
          kind: "draft-issue-comment",
          issue: target.issue,
          comment,
        };
      }
      continue;
    }
    if (candidate.kind === "continue") {
      const rawPrompt = candidate.prompt;
      const prompt = typeof rawPrompt === "string" ? rawPrompt.trim() : "";
      if (
        typeof rawPrompt === "string" &&
        prompt &&
        prompt.endsWith("?") &&
        prompt.length <= MAX_FOLLOW_UP_LENGTH &&
        !/[\r\n]/.test(rawPrompt) &&
        !hasUnboundResourceMention(prompt, []) &&
        continueActions.length < MAX_CONTINUE_ACTIONS &&
        !seenPrompts.has(prompt.toLowerCase())
      ) {
        continueActions.push({ kind: "continue", prompt });
        seenPrompts.add(prompt.toLowerCase());
      }
    }
  }
  return primaryAction ? [primaryAction, ...continueActions] : continueActions;
}

export function extractPlanDraftSQL(answer: string): string | undefined {
  const blocks = markdown
    .parse(answer)
    .children.filter(
      (node): node is Code =>
        node.type === "code" && node.lang?.toLowerCase() === "sql"
    );
  if (blocks.length !== 1) return undefined;
  const sql = blocks[0].value.trim();
  return sql && sql.length <= MAX_DRAFT_SQL_LENGTH ? sql : undefined;
}

export function acceptAgentResponseActions(
  proposed: unknown,
  observedOpenTargets: AgentHandoff[],
  planDraftContext?: {
    verifiedDatabase: { name: string; project: string };
    answer: string;
  },
  observedMaskingDraftIds: readonly string[] = [],
  observedExemptionDraftIds: readonly string[] = [],
  observedSemanticTypeDraftIds: readonly string[] = []
): AgentResponseAction[] {
  const normalizedActions = normalizeAgentResponseActions(proposed);
  const preparedMaskingDraft = normalizeAgentResponseActions([
    {
      kind: observedSemanticTypeDraftIds.length
        ? "draft-semantic-type"
        : observedExemptionDraftIds.length
          ? "draft-masking-exemption"
          : "draft-masking-rule",
      draftId: observedSemanticTypeDraftIds.length
        ? observedSemanticTypeDraftIds.at(-1)
        : observedExemptionDraftIds.length
          ? observedExemptionDraftIds.at(-1)
          : observedMaskingDraftIds.at(-1),
    },
  ])[0];
  if (
    preparedMaskingDraft?.kind === "draft-masking-rule" ||
    preparedMaskingDraft?.kind === "draft-masking-exemption" ||
    preparedMaskingDraft?.kind === "draft-semantic-type"
  ) {
    return [
      preparedMaskingDraft,
      ...normalizedActions.filter((action) => action.kind === "continue"),
    ];
  }
  const actions = normalizedActions.filter((action) => {
    if (action.kind === "continue") return true;
    if (action.kind === "review-sql-fix") return false;
    if (
      action.kind === "draft-masking-rule" ||
      action.kind === "draft-masking-exemption" ||
      action.kind === "draft-semantic-type"
    )
      return false;
    if (action.kind === "open")
      return observedOpenTargets.some(
        (target) => JSON.stringify(target) === JSON.stringify(action.target)
      );
    if (action.kind === "draft-issue-comment")
      return observedOpenTargets.some(
        (target) =>
          target.kind === "open-issue" && target.issue === action.issue
      );
    return (
      planDraftContext?.verifiedDatabase.name === action.database &&
      planDraftContext.verifiedDatabase.project === action.project &&
      Boolean(extractPlanDraftSQL(planDraftContext.answer))
    );
  });
  const proposedPrimary =
    Array.isArray(proposed) &&
    proposed.some(
      (action) =>
        isRecord(action) &&
        (action.kind === "open" ||
          action.kind === "draft-plan" ||
          action.kind === "draft-masking-rule" ||
          action.kind === "draft-masking-exemption" ||
          action.kind === "draft-semantic-type" ||
          action.kind === "draft-issue-comment")
    );
  if (!proposedPrimary && planDraftContext) {
    const sql = extractPlanDraftSQL(planDraftContext.answer);
    if (
      sql &&
      /^\s*(?:CREATE|ALTER|DROP|RENAME|TRUNCATE|COMMENT)\b/i.test(sql)
    ) {
      const draft = normalizeAgentResponseActions([
        {
          kind: "draft-plan",
          project: planDraftContext.verifiedDatabase.project,
          database: planDraftContext.verifiedDatabase.name,
        },
      ])[0];
      if (draft?.kind === "draft-plan") return [draft, ...actions];
    }
  }
  return actions;
}
