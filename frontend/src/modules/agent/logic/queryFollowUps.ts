import { isCanonicalQueryMaskingQuestion } from "./queryFollowUpQuestions";
import { normalizeAgentResponseActions } from "./responseActions";
import {
  getCurrentTurnToolCalls,
  type ToolObservation,
} from "./toolObservations";
import type {
  AgentQueryMaskingEligibility,
  AgentResourceReference,
  AgentResponseAction,
  Message,
} from "./types";

const isRecord = (value: unknown): value is Record<string, unknown> =>
  value !== null && typeof value === "object" && !Array.isArray(value);
const parseRecord = (raw: string | undefined) => {
  try {
    const value: unknown = JSON.parse(raw ?? "");
    return isRecord(value) ? value : undefined;
  } catch {
    return undefined;
  }
};
const projectPattern = /^projects\/[A-Za-z0-9_-]+$/;
const databasePattern =
  /^(?:projects\/[A-Za-z0-9_-]+\/)?instances\/[A-Za-z0-9_-]+\/databases\/[^/\s@?#]+$/;
const requestPrefixPattern =
  /^(?:(?:can|could|would|will|should)\s+(?:i|we|you)\s+|how\s+(?:can|could|do|should|would)\s+(?:i|we|you)\s+|how\s+to\s+)(?:help\s+(?:me|us)\s+)?/;
const maskingActionPattern = /^(?:un)?mask\b/;
const maskingConfigurationPattern =
  /^(?:set\s+up|configure|apply|enable|disable|change|manage|add|remove)\s+(?:(?:a|the|data)\s+)?(?:masking|masks?)\b/;
const unmaskedAccessPattern =
  /^(?:access|view|see|get(?:\s+access\s+to)?|request(?:\s+access\s+to)?|show(?:\s+(?:me|us))?)\s+(?:(?:the|these|those|my|our|your)\s+)?unmasked\b/;
const maskingSubjectPattern =
  /^(?:(?:which|what)\s+(?:are\s+|is\s+)?|(?:how|what)\s+(?:does|do|can|will|would|is)\s+(?:the\s+)?|(?:is|are|was|were)\s+)(?:data\s+)?(?:masking|unmasking|masks?)\b/;
const maskingVisibilityPattern =
  /^(?:(?:why|when)\s+)?(?:are|is|were|was)\s+(?:(?:these|those|the|my|our|your)\s+)?(?:values?|results?|columns?|fields?|data|rows?)\s+(?:being\s+)?(?:un)?masked\b/;
const unmaskingActorPattern =
  /^who\s+(?:(?:can|may|will|should)\s+)?unmasks?\b/;
const exemptionCommandPattern =
  /^(?:request|renew|grant|revoke|configure|change|remove|enable|disable)\s+(?:(?:a|an|the|my|your|our|current|active|existing|masking)\s+)*exemptions?\b/;
const exemptionInspectionPattern =
  /^(?:check|inspect)\s+(?:(?:whether|if|when)\s+)?(?:my|your|our|this|that|the|current|active|existing|masking)\s+(?:(?:current|active|existing|masking)\s+)*exemptions?\b/;
const exemptionSubjectPattern =
  /\b(?:my|your|our|this|that|the|current|active|existing|masking)\s+(?:(?:current|active|existing|masking)\s+)*exemptions?\b/;
const exemptionStateQuestionPattern =
  /^(?:when|how\s+long|does|do|is|are|has|have|why)\b/;
const exemptionStatePattern =
  /\b(?:expir(?:e[sd]?|ing|ations?|y)|configur(?:e[sd]?|ing|ations?)|request(?:s|ed|ing)?|renew(?:s|ed|ing|als?)?|grant(?:s|ed|ing)?|appl(?:y|ies|ied|ying|icable|icability)|valid(?:ity)?|active|enabl(?:e[sd]?|ing)|eligib(?:le|ility)|access)\b/i;

function isMaskingQuestion(prompt: string): boolean {
  if (isCanonicalQueryMaskingQuestion(prompt)) return true;
  const question = prompt.trim().toLowerCase();
  const request = question.replace(requestPrefixPattern, "");
  return (
    maskingActionPattern.test(request) ||
    maskingConfigurationPattern.test(request) ||
    unmaskedAccessPattern.test(request) ||
    maskingSubjectPattern.test(question) ||
    maskingVisibilityPattern.test(question) ||
    unmaskingActorPattern.test(question) ||
    exemptionCommandPattern.test(request) ||
    exemptionInspectionPattern.test(request) ||
    (exemptionStateQuestionPattern.test(question) &&
      exemptionSubjectPattern.test(question) &&
      exemptionStatePattern.test(question))
  );
}

const isQuery = (call: ToolObservation) =>
  call.name === "call_api" &&
  (call.args.operationId === "SQLService/Query" ||
    call.args.operationId === "bytebase.v1.SQLService.Query");

export function getCurrentTurnQueryCalls(
  messages: readonly Message[]
): { args: Record<string, unknown>; result?: string }[] {
  return getCurrentTurnToolCalls(messages)
    .filter(isQuery)
    .map(({ args, result }) => ({
      args,
      ...(result !== undefined ? { result } : {}),
    }));
}

function hasMaskingReason(value: unknown): boolean {
  return (
    Array.isArray(value) &&
    value.some(
      (reason) =>
        isRecord(reason) &&
        ([
          "semanticTypeId",
          "semanticTypeTitle",
          "maskingRuleId",
          "algorithm",
          "context",
        ].some(
          (field) =>
            typeof reason[field] === "string" && reason[field].trim() !== ""
        ) ||
          (typeof reason.classificationLevel === "number" &&
            reason.classificationLevel > 0))
    )
  );
}

function getDataQuery(call: ToolObservation) {
  const body = call.args.body;
  const value = parseRecord(call.result);
  if (
    !isRecord(body) ||
    typeof body.name !== "string" ||
    !databasePattern.test(body.name) ||
    body.explain === true ||
    !value ||
    typeof value.status !== "number" ||
    value.status < 200 ||
    value.status >= 300 ||
    (value.error !== undefined && value.error !== "") ||
    !isRecord(value.response) ||
    !Array.isArray(value.response.results) ||
    value.response.results.length === 0
  )
    return undefined;
  const results = value.response.results;
  if (
    !results.every(
      (result) =>
        isRecord(result) &&
        (result.error === undefined || result.error === "") &&
        [
          "postgresError",
          "syntaxError",
          "permissionDenied",
          "commandError",
        ].every((field) => result[field] == null) &&
        (result.queryPlan == null ||
          (isRecord(result.queryPlan) &&
            typeof result.queryPlan.content === "string" &&
            result.queryPlan.content.trim() !== ""))
    )
  )
    return undefined;
  if (
    !results.some(
      (result) =>
        Array.isArray(result.columnNames) &&
        result.columnNames.length > 0 &&
        result.columnNames.every(
          (column: unknown) => typeof column === "string"
        )
    )
  )
    return undefined;
  return {
    database: body.name,
    masked: results.some((result) => hasMaskingReason(result.masked)),
  };
}

function getScope(
  database: string,
  references: readonly AgentResourceReference[]
) {
  const project = database.startsWith("projects/")
    ? database.split("/").slice(0, 2).join("/")
    : undefined;
  if (references.length === 0) return { bound: false, project };
  const databases = references.filter(
    (reference) => reference.kind === "database"
  );
  if (databases.length === 0) {
    const selected = references[0].project;
    return projectPattern.test(selected) &&
      (project === undefined || project === selected) &&
      references.every(
        (reference) =>
          reference.kind === "project" &&
          reference.name === selected &&
          reference.project === selected
      )
      ? { bound: false, project: selected }
      : undefined;
  }
  const selected = databases[0];
  if (
    databases.length !== 1 ||
    selected.name !== database ||
    !projectPattern.test(selected.project) ||
    (project !== undefined && selected.project !== project) ||
    references.some((reference) => reference.project !== selected.project)
  )
    return undefined;
  return { bound: true, project: selected.project };
}

function getDataQueryContext(
  tools: ToolObservation[],
  references: readonly AgentResourceReference[]
) {
  const queries = tools.filter(isQuery);
  if (queries.length === 0) return undefined;
  const results = queries
    .map(getDataQuery)
    .filter((result) => result !== undefined);
  if (
    results.length !== queries.length ||
    new Set(results.map((result) => result.database)).size !== 1
  )
    return undefined;
  const database = results[0].database;
  const scope = getScope(database, references);
  return scope
    ? { database, scope, masked: results.some((result) => result.masked) }
    : undefined;
}

const supportsQueryFollowUps = (actions: AgentResponseAction[]) =>
  actions.some((action) => action.kind === "continue") &&
  actions.every(
    (action) => action.kind === "continue" || action.kind === "open"
  );

export function getQueryMaskingSetupTarget(
  actions: AgentResponseAction[],
  turnMessages: readonly Message[],
  references: readonly AgentResourceReference[] = []
): string | undefined {
  if (
    !supportsQueryFollowUps(actions) ||
    actions.filter(
      (action) =>
        action.kind === "continue" && !isMaskingQuestion(action.prompt)
    ).length >= 3
  )
    return undefined;
  const context = getDataQueryContext(
    getCurrentTurnToolCalls(turnMessages),
    references
  );
  return context && !context.masked ? context.database : undefined;
}

// Older answers have no runtime snapshot; retain their original overview-based check.
function canConfigureMasking(
  tools: ToolObservation[],
  database: string,
  scope: { bound: boolean; project?: string }
): boolean {
  let latest: Record<string, unknown> | undefined;
  for (const call of tools) {
    if (
      (call.name === "call_api" && !isQuery(call)) ||
      call.name === "dom_action"
    ) {
      latest = undefined;
      continue;
    }
    if (call.name !== "get_masking_overview") continue;
    const value = parseRecord(call.result);
    if (
      call.args.database === database ||
      (call.args.database === undefined &&
        ((isRecord(value?.catalog) && value.catalog.database === database) ||
          (scope.bound && call.args.project === undefined)))
    )
      latest = value;
  }
  if (
    !latest ||
    typeof latest.project !== "string" ||
    !projectPattern.test(latest.project) ||
    (scope.project !== undefined && latest.project !== scope.project) ||
    (latest.databaseSelection !== undefined &&
      (!isRecord(latest.databaseSelection) ||
        latest.databaseSelection.status !== "available")) ||
    (isRecord(latest.catalog) &&
      latest.catalog.database !== undefined &&
      latest.catalog.database !== database) ||
    !isRecord(latest.maskingPolicy) ||
    latest.maskingPolicy.status !== "available" ||
    !isRecord(latest.semanticTypes) ||
    latest.semanticTypes.status !== "available" ||
    !isRecord(latest.setup) ||
    !isRecord(latest.setup.features) ||
    latest.setup.features.masking !== "available" ||
    !Array.isArray(latest.setup.steps)
  )
    return false;
  if (
    latest.setup.project !== undefined &&
    (!isRecord(latest.setup.project) ||
      latest.setup.project.name !== latest.project)
  )
    return false;
  const global = latest.setup.steps.find(
    (step) => isRecord(step) && step.id === "global-masking"
  );
  const types = latest.setup.steps.find(
    (step) => isRecord(step) && step.id === "semantic-types"
  );
  return (
    isRecord(global?.configure) &&
    global.configure.status === "allowed" &&
    isRecord(global.readAccess) &&
    global.readAccess.status === "allowed" &&
    isRecord(types?.readAccess) &&
    types.readAccess.status === "allowed"
  );
}

// Only masking-access suggestions are rebuilt from evidence; ordinary questions
// and prepared workflows retain their original wording and order.
export function getQueryResponseActions(
  actions: AgentResponseAction[],
  turnMessages: readonly Message[],
  references: readonly AgentResourceReference[] = [],
  t: (key: string, params?: { database: string }) => string,
  maskingEligibility?: AgentQueryMaskingEligibility
): AgentResponseAction[] {
  if (!supportsQueryFollowUps(actions)) return actions;
  const tools = getCurrentTurnToolCalls(turnMessages);
  if (!tools.some(isQuery)) return actions;
  const retained = actions.filter(
    (action) => action.kind !== "continue" || !isMaskingQuestion(action.prompt)
  );
  const context = getDataQueryContext(tools, references);
  if (!context) return retained;
  const { database, scope, masked } = context;
  const followUps: AgentResponseAction[] = [];
  const add = (prompt: string) => {
    if (prompt && prompt.length <= 140 && !/[\r\n]/.test(prompt)) {
      followUps.push({ kind: "continue", prompt });
    }
  };
  if (masked) {
    add(
      scope.bound
        ? t("agent.follow-up.query-unmask")
        : t("agent.follow-up.query-unmask-database", { database })
    );
  } else if (
    maskingEligibility === undefined
      ? canConfigureMasking(tools, database, scope)
      : isRecord(maskingEligibility) &&
        maskingEligibility.database === database &&
        maskingEligibility.allowed === true
  ) {
    add(
      scope.bound
        ? t("agent.follow-up.query-mask")
        : t("agent.follow-up.query-mask-database", { database })
    );
  }
  return normalizeAgentResponseActions([...retained, ...followUps]);
}
