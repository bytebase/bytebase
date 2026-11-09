export const AGENT_PLAN_DRAFT_EVENT = "bytebase:agent-plan-draft";

export interface AgentPlanDraftRequest {
  project: string;
  database: string;
  sql: string;
  onApplied?: () => void;
}

export interface AgentPlanDraftEventDetail extends AgentPlanDraftRequest {
  handled: boolean;
}

export function dispatchAgentPlanDraftToCurrentPlan(
  request: AgentPlanDraftRequest
): boolean {
  if (typeof window === "undefined") return false;
  const detail: AgentPlanDraftEventDetail = { ...request, handled: false };
  window.dispatchEvent(
    new CustomEvent<AgentPlanDraftEventDetail>(AGENT_PLAN_DRAFT_EVENT, {
      detail,
    })
  );
  return detail.handled;
}

export function appendAgentPlanDraftSQL(
  existingSQL: string,
  draftSQL: string
): string {
  const existing = existingSQL.trimEnd();
  const draft = draftSQL.trim();
  if (!existing) return draft;
  if (!draft) return existing;
  const terminator = existing.endsWith(";") ? "" : "\n;";
  return `${existing}${terminator}\n\n${draft}`;
}
