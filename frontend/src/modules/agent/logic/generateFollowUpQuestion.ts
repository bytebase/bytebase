import { create } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { aiServiceClientConnect } from "@/api";
import { silentContextKey } from "@/api/context-key";
import {
  AIChatMessageRole,
  AIChatMessageSchema,
  AIChatResponse_StopReason,
} from "@/types/proto-es/v1/ai_service_pb";
import { normalizeAgentResponseActions } from "./responseActions";
import type { AgentHandoff, AgentResponseAction } from "./types";

type ContinueAction = Extract<AgentResponseAction, { kind: "continue" }>;

interface FollowUpResult {
  actions: ContinueAction[];
  totalTokensUsed: number;
}

const MAX_REQUEST_CONTEXT_LENGTH = 2_000;
const MAX_ANSWER_CONTEXT_LENGTH = 4_000;
const MAX_FOLLOW_UP_QUESTIONS = 3;
const FOLLOW_UP_TIMEOUT_MS = 8_000;

const systemPrompt =
  "Suggest distinct next questions for this Bytebase Agent answer. Reply with only a JSON array of up to the requested number of English question strings, or [] if no useful question exists. Each question should advance the user's task beyond the answer and available click action, and differ from existing questions. Do not add filler. Do not invent resources, changes, or permissions. Do not include @ mentions, SQL, API calls, Markdown, or line breaks. Keep each question under 140 characters.";

export async function generateFollowUpQuestion(
  request: string,
  answer: string,
  context: {
    action?: AgentHandoff | Exclude<AgentResponseAction, ContinueAction>;
    existingQuestions: string[];
  },
  signal: AbortSignal
): Promise<FollowUpResult> {
  const existingQuestions = context.existingQuestions.slice(
    0,
    MAX_FOLLOW_UP_QUESTIONS
  );
  const neededQuestions = MAX_FOLLOW_UP_QUESTIONS - existingQuestions.length;
  if (signal.aborted || neededQuestions === 0) {
    return { actions: [], totalTokensUsed: 0 };
  }

  const requestController = new AbortController();
  const abortRequest = () => requestController.abort();
  signal.addEventListener("abort", abortRequest, { once: true });
  const timeout = setTimeout(abortRequest, FOLLOW_UP_TIMEOUT_MS);
  try {
    const response = await aiServiceClientConnect.chat(
      {
        messages: [
          create(AIChatMessageSchema, {
            role: AIChatMessageRole.AI_CHAT_MESSAGE_ROLE_SYSTEM,
            content: systemPrompt,
          }),
          create(AIChatMessageSchema, {
            role: AIChatMessageRole.AI_CHAT_MESSAGE_ROLE_USER,
            content: JSON.stringify({
              request: request.slice(0, MAX_REQUEST_CONTEXT_LENGTH),
              answer: answer.slice(0, MAX_ANSWER_CONTEXT_LENGTH),
              action: context.action,
              existingQuestions,
              neededQuestions,
            }),
          }),
        ],
        toolDefinitions: [],
      },
      {
        signal: requestController.signal,
        contextValues: createContextValues().set(silentContextKey, true),
      }
    );
    const totalTokensUsed = response.usage?.totalTokens ?? 0;
    if (
      requestController.signal.aborted ||
      response.toolCalls.length > 0 ||
      (response.stopReason !==
        AIChatResponse_StopReason.STOP_REASON_UNSPECIFIED &&
        response.stopReason !== AIChatResponse_StopReason.TURN_COMPLETED)
    ) {
      return { actions: [], totalTokensUsed };
    }

    let proposed: unknown;
    try {
      proposed = JSON.parse(response.content?.trim() ?? "");
    } catch {
      return { actions: [], totalTokensUsed };
    }
    if (!Array.isArray(proposed)) {
      return { actions: [], totalTokensUsed };
    }
    const seen = new Set(
      existingQuestions.map((question) => question.trim().toLowerCase())
    );
    const candidates = proposed.filter((question): question is string => {
      if (typeof question !== "string") return false;
      const normalized = question.trim().toLowerCase();
      if (!normalized.endsWith("?") || seen.has(normalized)) return false;
      seen.add(normalized);
      return true;
    });
    const actions = normalizeAgentResponseActions(
      candidates.map((prompt) => ({ kind: "continue", prompt }))
    )
      .filter(
        (candidate): candidate is ContinueAction =>
          candidate.kind === "continue"
      )
      .slice(0, neededQuestions);
    return { actions, totalTokensUsed };
  } catch {
    return { actions: [], totalTokensUsed: 0 };
  } finally {
    clearTimeout(timeout);
    signal.removeEventListener("abort", abortRequest);
  }
}
