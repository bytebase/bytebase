import type {
  AgentResourceReference,
  AgentResponseAction,
  Message,
} from "./types";

interface ToolObservation {
  name: string;
  args: Record<string, unknown>;
  result?: string;
}

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

function getTurnTools(messages: readonly Message[]): ToolObservation[] {
  const lastUser = messages.findLastIndex((message) => message.role === "user");
  if (lastUser < 0) return [];
  const observations: ToolObservation[] = [];
  const calls = new Map<string, ToolObservation>();
  for (const message of messages.slice(lastUser + 1)) {
    if (message.role === "assistant") {
      for (const call of message.toolCalls ?? []) {
        const args = parseRecord(call.arguments);
        if (!args) continue;
        const observation = { name: call.name, args };
        observations.push(observation);
        calls.set(call.id, observation);
      }
    } else if (message.role === "tool" && message.toolCallId) {
      const call = calls.get(message.toolCallId);
      if (call) call.result = message.content;
    }
  }
  return observations;
}

const isQuery = (call: ToolObservation) =>
  call.name === "call_api" &&
  (call.args.operationId === "SQLService/Query" ||
    call.args.operationId === "bytebase.v1.SQLService.Query");

export function getCurrentTurnQueryCalls(
  messages: readonly Message[]
): { args: Record<string, unknown>; result?: string }[] {
  return getTurnTools(messages)
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
          "queryPlan",
        ].every((field) => result[field] == null)
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

// Query-result questions are runtime-owned so model text cannot invent access state.
// Prepared workflows retain their own actions.
export function getQueryResponseActions(
  actions: AgentResponseAction[],
  turnMessages: readonly Message[],
  references: readonly AgentResourceReference[] = [],
  t: (key: string, params?: { database: string }) => string
): AgentResponseAction[] {
  if (
    !actions.some((action) => action.kind === "continue") ||
    actions.some(
      (action) => action.kind !== "continue" && action.kind !== "open"
    )
  )
    return actions;
  const tools = getTurnTools(turnMessages);
  const queries = tools.filter(isQuery);
  if (queries.length === 0) return actions;
  const primary = actions.filter((action) => action.kind !== "continue");
  const results = queries
    .map(getDataQuery)
    .filter((result) => result !== undefined);
  if (
    results.length !== queries.length ||
    new Set(results.map((result) => result.database)).size !== 1
  )
    return primary;
  const database = results[0].database;
  const followUps: AgentResponseAction[] = [];
  const add = (prompt: string) => {
    if (prompt && prompt.length <= 140 && !/[\r\n]/.test(prompt)) {
      followUps.push({ kind: "continue", prompt });
    }
  };
  add(t("agent.follow-up.query-analysis"));
  const scope = getScope(database, references);
  if (!scope) return [...primary, ...followUps];
  if (results.some((result) => result.masked)) {
    add(
      scope.bound
        ? t("agent.follow-up.query-unmask")
        : t("agent.follow-up.query-unmask-database", { database })
    );
  } else if (canConfigureMasking(tools, database, scope)) {
    add(
      scope.bound
        ? t("agent.follow-up.query-mask")
        : t("agent.follow-up.query-mask-database", { database })
    );
  }
  return [...primary, ...followUps];
}
