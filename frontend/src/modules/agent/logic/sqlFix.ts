import type { Code, RootContent } from "mdast";
import remarkParse from "remark-parse";
import { unified } from "unified";
import { useAppStore } from "@/stores/app";
import { normalizeAgentDiagnosticCapability } from "./diagnosticCapability";
import type { AgentDiagnosticContext } from "./diagnosticHelp";
import type { AgentResourceReference } from "./types";

export interface AgentSqlFixSource {
  identity: string;
  pageKey: string;
  resource: AgentResourceReference;
  sourceId: string;
  plan: string;
  specId: string;
  sheet: string;
  targets: string[];
  sha256: string;
  sql: string;
}

export interface AgentSqlFixSourceHint {
  specId?: string;
  sha256: string;
  target?: string;
}

export interface AgentSqlFixDraft {
  id: string;
  chatId: string;
  source: AgentSqlFixSource;
  sql: string;
}

export interface AgentSqlFixTargetState {
  status:
    | "ready"
    | "loading"
    | "stale"
    | "readonly"
    | "permission-denied"
    | "unavailable";
  currentSql?: string;
}

export interface AgentSqlFixTargetAdapter {
  capture: (
    context: AgentDiagnosticContext
  ) => AgentSqlFixSource | undefined | Promise<AgentSqlFixSource | undefined>;
  reveal: (
    source: AgentSqlFixSource
  ) => boolean | void | Promise<boolean | void>;
  inspect: (source: AgentSqlFixSource) => AgentSqlFixTargetState;
  apply: (source: AgentSqlFixSource, sql: string) => boolean;
}

const markdown = unified().use(remarkParse);
const encoder = new TextEncoder();
const MAX_PROPOSAL_BYTES = 100_000;
let registration: { adapter: AgentSqlFixTargetAdapter } | undefined;
let version = 0;
const listeners = new Set<() => void>();

export function getAgentSqlFixIdentity(): string | undefined {
  const app = useAppStore.getState();
  const workspace = app.workspaceResourceName?.();
  const user = app.currentUser?.name;
  return workspace &&
    /^workspaces\/[A-Za-z0-9_-]+$/.test(workspace) &&
    user &&
    /^users\/[^/\s]+$/.test(user)
    ? JSON.stringify([workspace, user])
    : undefined;
}

export function isCurrentAgentSqlFixSource(
  source: AgentSqlFixSource,
  resource: AgentResourceReference
): boolean {
  if (
    !source ||
    ![
      source.identity,
      source.pageKey,
      source.sourceId,
      source.plan,
      source.specId,
      source.sheet,
      source.sha256,
      source.sql,
    ].every((field) => typeof field === "string") ||
    !Array.isArray(source.targets) ||
    !source.targets.every((target) => typeof target === "string")
  )
    return false;
  const capability = normalizeAgentDiagnosticCapability({
    kind: "diagnostic",
    resource: source.resource,
  });
  return Boolean(
    capability &&
      source.identity === getAgentSqlFixIdentity() &&
      capability.resource.kind === resource.kind &&
      capability.resource.name === resource.name &&
      capability.resource.project === resource.project &&
      source.pageKey &&
      source.sourceId &&
      source.specId &&
      source.plan.startsWith(`${resource.project}/plans/`) &&
      source.plan.split("/").length === 4 &&
      (/\/plans\/[1-9]\d*$/.test(source.plan) ||
        (/\/plans\/-(?:[1-9]\d*)$/.test(source.plan) &&
          (source.pageKey ===
            `${resource.project.slice("projects/".length)}/create` ||
            source.pageKey.split(/[?#]/)[0].endsWith("/plans/create")) &&
          /\/sheets\/-[1-9]\d*$/.test(source.sheet))) &&
      source.sheet.startsWith(`${resource.project}/sheets/`) &&
      /^projects\/[^/]+\/sheets\/[^/]+$/.test(source.sheet) &&
      /^[a-f0-9]{64}$/i.test(source.sha256) &&
      source.targets.length > 0 &&
      source.targets.every((target) =>
        normalizeAgentDiagnosticCapability({
          kind: "diagnostic",
          resource: {
            kind: "database",
            name: target,
            project: resource.project,
            label: target,
          },
        })
      ) &&
      source.sql.trim() &&
      encoder.encode(source.sql).byteLength <= 2000
  );
}

export function extractAgentSqlFixSQL(answer: string): string | undefined {
  const tree = markdown.parse(answer);
  const blocks: Code[] = [];
  const visit = (nodes: readonly RootContent[]) => {
    for (const node of nodes) {
      if (node.type === "code" && node.lang?.toLowerCase() === "sql")
        blocks.push(node);
      if ("children" in node) visit(node.children as RootContent[]);
    }
  };
  visit(tree.children);
  const replacements = blocks.filter(
    (block) => block.meta?.toLowerCase() === "replacement"
  );
  const candidates = replacements.length ? replacements : blocks;
  if (candidates.length !== 1) return undefined;
  const block = candidates[0];
  // Container indentation can change literal SQL bytes in a raw Markdown slice.
  if (!tree.children.includes(block)) return undefined;
  const raw = answer.slice(
    block.position?.start.offset,
    block.position?.end.offset
  );
  const opening =
    /^ {0,3}(`{3,}|~{3,})sql(?:[ \t]+replacement)?[ \t]*\r?\n/i.exec(raw);
  if (!opening) return undefined;
  const closing = new RegExp(
    `(?:^|\\r?\\n) {0,3}${opening[1][0]}{${opening[1].length},}[ \\t]*$`
  ).exec(raw);
  if (!closing || closing.index < opening[0].length) return undefined;
  const sql = raw.slice(opening[0].length, closing.index);
  if (
    !sql.trim() ||
    encoder.encode(sql).byteLength > MAX_PROPOSAL_BYTES ||
    sql.includes("\0") ||
    /\[\s*(?:redacted|omitted|placeholder)[^\]]*\]|<\s*(?:table|column|database|sql|your[_ -]?\w+)[^>]*>|\.{3}|\b(?:rest|remaining|other|original)\s+(?:of\s+(?:the\s+)?)?(?:sql|statements?|code)\s+(?:is\s+)?(?:unchanged|omitted)|\b(?:insert|replace|add)\s+(?:your\s+)?(?:sql|code)\s+here\b/i.test(
      sql
    )
  )
    return undefined;
  return sql;
}

export function notifyAgentSqlFixTargetChanged(): void {
  version++;
  for (const listener of listeners) listener();
}

export function subscribeAgentSqlFixTarget(listener: () => void): () => void {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

export function getAgentSqlFixTargetVersion(): number {
  return version;
}

export function registerAgentSqlFixTarget(
  adapter: AgentSqlFixTargetAdapter
): () => void {
  const current = { adapter };
  registration = current;
  notifyAgentSqlFixTargetChanged();
  return () => {
    if (registration !== current) return;
    registration = undefined;
    notifyAgentSqlFixTargetChanged();
  };
}

export function captureAgentSqlFixSource(
  context: AgentDiagnosticContext
): AgentSqlFixSource | undefined | Promise<AgentSqlFixSource | undefined> {
  const current = registration;
  const adapter = current?.adapter;
  const hint = context.sqlFixSource;
  if (!adapter || !hint) return undefined;
  const accept = (source: AgentSqlFixSource | undefined) =>
    source &&
    registration === current &&
    isCurrentAgentSqlFixSource(source, context.resource) &&
    source.sourceId === context.sourceId &&
    source.sha256 === hint.sha256 &&
    (!hint.specId || source.specId === hint.specId) &&
    (!hint.target || source.targets.includes(hint.target))
      ? source
      : undefined;
  const captured = adapter.capture(context);
  return captured instanceof Promise ? captured.then(accept) : accept(captured);
}

export function revealAgentSqlFixTarget(
  source: AgentSqlFixSource
): boolean | void | Promise<boolean | void> {
  if (isCurrentAgentSqlFixSource(source, source.resource))
    return registration?.adapter.reveal(source);
}

export function inspectAgentSqlFixTarget(
  source: AgentSqlFixSource
): AgentSqlFixTargetState {
  if (!registration || !isCurrentAgentSqlFixSource(source, source.resource))
    return { status: "unavailable" };
  try {
    return registration.adapter.inspect(source);
  } catch {
    return { status: "unavailable" };
  }
}

export function applyAgentSqlFixToTarget(
  source: AgentSqlFixSource,
  sql: string
): boolean {
  const current = registration;
  const adapter = current?.adapter;
  const state = inspectAgentSqlFixTarget(source);
  if (
    !adapter ||
    state.status !== "ready" ||
    state.currentSql !== source.sql ||
    !sql.trim()
  )
    return false;
  try {
    return registration === current && adapter.apply(source, sql);
  } catch {
    return false;
  }
}
