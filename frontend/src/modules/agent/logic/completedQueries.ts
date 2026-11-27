import { isEqual } from "lodash-es";
import { parseCallApiResult } from "./callApiResult";
import {
  type AgentQueryResultPreview,
  parseAgentQueryResult,
} from "./queryResult";
import type { AgentMessage } from "./types";

export interface AgentCompletedQuery {
  callMessageId: string;
  resultMessageId: string;
  toolCallIndex: number;
  database: string;
  statement: string;
  request: Readonly<Record<string, unknown>>;
  preview: AgentQueryResultPreview;
}

const isRecord = (value: unknown): value is Record<string, unknown> =>
  value !== null && typeof value === "object" && !Array.isArray(value);
const databasePattern =
  /^(?:projects\/[A-Za-z0-9_-]+\/)?instances\/[A-Za-z0-9_-]+\/databases\/[^/\s@?#]+$/;

function successfulPreview(result: AgentMessage, operationId: string) {
  if (!result.content) return;
  const outcome = parseCallApiResult(result.content, operationId);
  if (
    !outcome.succeeded ||
    outcome.requiresRecovery ||
    outcome.error ||
    outcome.execution?.reason !== "rpc-success" ||
    (result.metadata?.apiAttemptId !== undefined &&
      result.metadata.apiAttemptId !== outcome.execution.attemptId)
  )
    return;
  try {
    const envelope: unknown = JSON.parse(result.content);
    if (
      !isRecord(envelope) ||
      !isRecord(envelope.response) ||
      !Array.isArray(envelope.response.results) ||
      !envelope.response.results.every(
        (value: unknown) =>
          isRecord(value) &&
          (value.error === undefined || value.error === "") &&
          [
            "postgresError",
            "syntaxError",
            "permissionDenied",
            "commandError",
          ].every((key) => value[key] == null)
      )
    )
      return;
  } catch {
    return;
  }
  const preview = parseAgentQueryResult(result.content);
  return preview?.resultSets.length ? preview : undefined;
}

export function getAgentCompletedQueries(
  answer: AgentMessage,
  messages: readonly AgentMessage[]
): AgentCompletedQuery[] {
  const answerIndex = messages.findIndex(({ id }) => id === answer.id);
  const runId = answer.metadata?.runId;
  if (
    answerIndex < 0 ||
    answerIndex !== messages.findLastIndex(({ id }) => id === answer.id) ||
    !isEqual(answer, messages[answerIndex]) ||
    answer.role !== "assistant" ||
    !answer.content?.trim() ||
    answer.toolCalls?.length ||
    answer.metadata?.pending ||
    answer.metadata?.error ||
    answer.metadata?.interruptionReason ||
    !runId
  )
    return [];
  const userIndex = messages
    .slice(0, answerIndex)
    .findLastIndex(({ role }) => role === "user");
  if (userIndex < 0 || messages[userIndex].chatId !== answer.chatId) return [];
  const completed: AgentCompletedQuery[] = [];
  const messageCounts = new Map<string, number>();
  for (const { id } of messages)
    messageCounts.set(id, (messageCounts.get(id) ?? 0) + 1);
  const uniqueMessage = (id: string) => messageCounts.get(id) === 1;
  for (let index = userIndex + 1; index < answerIndex; index++) {
    const callMessage = messages[index];
    if (
      callMessage.role !== "assistant" ||
      callMessage.chatId !== answer.chatId ||
      callMessage.metadata?.runId !== runId ||
      !uniqueMessage(callMessage.id)
    )
      continue;
    const results: AgentMessage[] = [];
    for (let next = index + 1; next < answerIndex; next++) {
      const result = messages[next];
      if (result.role === "assistant" || result.role === "user") break;
      if (result.role === "tool") results.push(result);
    }
    const calls = callMessage.toolCalls ?? [];
    for (const [toolCallIndex, call] of calls.entries()) {
      if (
        call.name !== "call_api" ||
        !call.id ||
        calls.filter(({ id }) => id === call.id).length !== 1
      )
        continue;
      const matches = results.filter(
        ({ toolCallId }) => toolCallId === call.id
      );
      if (matches.length !== 1) continue;
      const result = matches[0];
      if (
        result.chatId !== answer.chatId ||
        result.metadata?.runId !== runId ||
        !uniqueMessage(result.id)
      )
        continue;
      try {
        const args: unknown = JSON.parse(call.arguments);
        if (
          !isRecord(args) ||
          (args.operationId !== "SQLService/Query" &&
            args.operationId !== "bytebase.v1.SQLService.Query") ||
          !isRecord(args.body) ||
          typeof args.body.name !== "string" ||
          !databasePattern.test(args.body.name) ||
          typeof args.body.statement !== "string" ||
          !args.body.statement.trim() ||
          !args.body.statement.isWellFormed() ||
          args.body.statement.includes("\0") ||
          args.body.explain === true
        )
          continue;
        const preview = successfulPreview(result, args.operationId);
        if (!preview) continue;
        completed.push({
          callMessageId: callMessage.id,
          resultMessageId: result.id,
          toolCallIndex,
          database: args.body.name,
          statement: args.body.statement,
          request: args.body,
          preview,
        });
      } catch {
        // An unreadable request cannot identify which SQL produced a result.
      }
    }
  }
  return completed;
}
