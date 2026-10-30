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
    Object.keys(raw).some(
      (key) =>
        key !== "kind" &&
        key !== "workspace" &&
        key !== "project" &&
        key !== "plan" &&
        key !== "issue" &&
        key !== "task"
    )
  ) {
    return undefined;
  }
  // Some providers populate every optional schema field; retain only the one the target kind uses.
  for (const field of [
    "workspace",
    "project",
    "plan",
    "issue",
    "task",
  ] as const) {
    const target = normalizeAgentHandoff({
      kind: raw.kind,
      [field]: raw[field],
    });
    if (target) return target;
  }
  return undefined;
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
          key !== "comment"
      )
    ) {
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
  }
): AgentResponseAction[] {
  const actions = normalizeAgentResponseActions(proposed).filter(
    (action) =>
      action.kind === "continue" ||
      (action.kind === "open"
        ? observedOpenTargets.some(
            (target) => JSON.stringify(target) === JSON.stringify(action.target)
          )
        : action.kind === "draft-issue-comment"
          ? observedOpenTargets.some(
              (target) =>
                target.kind === "open-issue" && target.issue === action.issue
            )
          : planDraftContext?.verifiedDatabase.name === action.database &&
            planDraftContext.verifiedDatabase.project === action.project &&
            Boolean(extractPlanDraftSQL(planDraftContext.answer)))
  );
  const proposedPrimary =
    Array.isArray(proposed) &&
    proposed.some(
      (action) =>
        isRecord(action) &&
        (action.kind === "open" ||
          action.kind === "draft-plan" ||
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
