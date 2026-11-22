import type { MaskingOverviewArgs } from "./tools/maskingOverview";
import type { Message } from "./types";

const parseObject = (
  value: string | undefined
): Record<string, unknown> | undefined => {
  try {
    const parsed: unknown = JSON.parse(value ?? "");
    return parsed && typeof parsed === "object" && !Array.isArray(parsed)
      ? (parsed as Record<string, unknown>)
      : undefined;
  } catch {
    return undefined;
  }
};

export function getMaskingResumeArgs(
  messages: Message[]
): MaskingOverviewArgs | undefined {
  const userIndex = messages.findLastIndex(
    (message) => message.role === "user"
  );
  if (userIndex < 0) return undefined;
  const turn = messages.slice(userIndex + 1);
  const last = turn.at(-1);
  const calls = new Map(
    turn.flatMap((message) =>
      message.role === "assistant"
        ? (message.toolCalls ?? []).map((call) => [call.id, call] as const)
        : []
    )
  );
  const questionCall = last?.toolCallId
    ? calls.get(last.toolCallId)
    : undefined;
  const isExemptionScope = questionCall?.name === "ask_masking_exemption_scope";
  const isRuleScope = questionCall?.name === "ask_masking_rule_scope";
  const isSemanticType = questionCall?.name === "ask_masking_semantic_type";
  const isUnmaskMethod = questionCall?.name === "ask_unmask_method";
  if (
    last?.role !== "tool" ||
    !last.toolCallId ||
    (questionCall?.name !== "ask_user" &&
      !isExemptionScope &&
      !isRuleScope &&
      !isSemanticType &&
      !isUnmaskMethod) ||
    [...calls.values()].some((call) => call.name === "done")
  )
    return undefined;
  const answer = parseObject(last.content);
  if (
    !answer ||
    typeof answer.answer !== "string" ||
    (answer.kind !== "input" &&
      answer.kind !== "choose" &&
      answer.kind !== "confirm")
  ) {
    return undefined;
  }

  const question = parseObject(questionCall?.arguments);
  if (isUnmaskMethod) {
    const project =
      typeof question?.project === "string" && !question.project.trim()
        ? undefined
        : question?.project;
    const databaseArg =
      typeof question?.database === "string" && !question.database.trim()
        ? undefined
        : question?.database;
    if (
      answer.kind !== "choose" ||
      (answer.value !== "masking-exemption" &&
        answer.value !== "just-in-time") ||
      !question ||
      Object.keys(question).some(
        (key) =>
          key !== "project" &&
          key !== "database" &&
          key !== "queryAccessMissing" &&
          key !== "forCurrentUser"
      ) ||
      (project !== undefined &&
        (typeof project !== "string" ||
          !/^projects\/(?!-$)[A-Za-z0-9_-]+$/.test(project))) ||
      (databaseArg !== undefined && typeof databaseArg !== "string") ||
      (question.queryAccessMissing !== undefined &&
        typeof question.queryAccessMissing !== "boolean") ||
      (question.forCurrentUser !== undefined &&
        typeof question.forCurrentUser !== "boolean")
    )
      return undefined;
    const database = typeof databaseArg === "string" ? databaseArg : undefined;
    const selectedDatabase = database
      ? /^(?:(projects\/(?!-\/)[A-Za-z0-9_-]+)\/)?instances\/(?!-\/)[A-Za-z0-9_-]+\/databases\/(?!-$)[^/*]+$/.exec(
          database
        )
      : undefined;
    if (
      database !== undefined &&
      (!selectedDatabase ||
        (project !== undefined &&
          selectedDatabase[1] &&
          selectedDatabase[1] !== project))
    )
      return undefined;
    return {
      ...(typeof project === "string" ? { project } : {}),
      ...(database !== undefined ? { database } : {}),
    };
  }

  const overview = turn.findLast(
    (message) =>
      message.role === "tool" &&
      message.toolCallId &&
      calls.get(message.toolCallId)?.name === "get_masking_overview"
  );
  if (!overview?.toolCallId) return undefined;
  const args = parseObject(calls.get(overview.toolCallId)?.arguments);
  if (
    !args ||
    Object.keys(args).some(
      (key) =>
        key !== "project" && key !== "database" && key !== "semanticTypeId"
    ) ||
    (args.project !== undefined && typeof args.project !== "string") ||
    (args.database !== undefined && typeof args.database !== "string") ||
    (args.semanticTypeId !== undefined &&
      typeof args.semanticTypeId !== "string")
  ) {
    return undefined;
  }
  const result = parseObject(overview.content);
  const project =
    typeof args.project === "string"
      ? args.project
      : typeof result?.project === "string"
        ? result.project
        : undefined;
  if (isRuleScope) {
    if (answer.kind !== "choose" || typeof answer.value !== "string")
      return undefined;
    if (answer.value === "workspace" || answer.value === "choose-project")
      return {};
    if (/^projects\/(?!-$)[A-Za-z0-9_-]+$/.test(answer.value))
      return { project: answer.value };
    const selectedProject =
      typeof question?.project === "string" ? question.project : project;
    if (
      !selectedProject ||
      !/^projects\/(?!-$)[A-Za-z0-9_-]+$/.test(selectedProject)
    )
      return undefined;
    if (answer.value === "another-database")
      return { project: selectedProject };
    const selectedDatabase =
      /^(?:(projects\/[A-Za-z0-9_-]+)\/)?instances\/(?!-\/)[A-Za-z0-9_-]+\/databases\/(?!-$)[^/]+$/.exec(
        answer.value
      );
    if (
      !selectedDatabase ||
      (selectedDatabase[1] && selectedDatabase[1] !== selectedProject)
    )
      return undefined;
    return { project: selectedProject, database: answer.value };
  }
  if (
    isSemanticType &&
    (answer.kind !== "choose" ||
      typeof answer.value !== "string" ||
      !answer.value.trim())
  )
    return undefined;
  if (isExemptionScope || question?.purpose === "masking-exemption-scope") {
    const questionProject =
      typeof question?.project === "string" && !question.project.trim()
        ? undefined
        : question?.project;
    const selectedProject =
      isExemptionScope && questionProject !== undefined
        ? questionProject
        : project;
    if (
      (!isExemptionScope && question?.kind !== "choose") ||
      answer.kind !== "choose" ||
      typeof answer.value !== "string" ||
      typeof selectedProject !== "string" ||
      !/^projects\/(?!-$)[A-Za-z0-9_-]+$/.test(selectedProject)
    ) {
      return undefined;
    }
    if (
      answer.value === selectedProject ||
      answer.value === "another-database"
    ) {
      return { project: selectedProject };
    }
    const selectedDatabase =
      /^(?:(projects\/[A-Za-z0-9_-]+)\/)?instances\/(?!-\/)[A-Za-z0-9_-]+\/databases\/(?!-$)[^/*]+$/.exec(
        answer.value
      );
    if (
      !selectedDatabase ||
      (selectedDatabase[1] && selectedDatabase[1] !== selectedProject)
    ) {
      return undefined;
    }
    return { project: selectedProject, database: answer.value };
  }
  const catalog = result?.catalog;
  const database =
    catalog && typeof catalog === "object" && !Array.isArray(catalog)
      ? (catalog as Record<string, unknown>).database
      : undefined;

  // Preserve the previously selected scope; the executor revalidates current access.
  return {
    ...(project !== undefined ? { project } : {}),
    ...(typeof args.database === "string"
      ? { database: args.database }
      : typeof database === "string"
        ? { database }
        : {}),
  };
}
