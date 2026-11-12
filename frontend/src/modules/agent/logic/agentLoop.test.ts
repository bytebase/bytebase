// @vitest-environment node
import { beforeEach, describe, expect, test, vi } from "vitest";
import { aiServiceClientConnect } from "@/api";
import {
  AIChatMessageRole,
  AIChatResponse_StopReason,
} from "@/types/proto-es/v1/ai_service_pb";
import { runAgentLoop } from "./agentLoop";
import type {
  AgentModelAttempt,
  ToolExecutionResult,
  ToolExecutor,
} from "./types";

const preferenceDefaults = vi.hoisted(() => ({ tokenLimitPerRun: 100_000 }));

vi.mock("../store/preferences", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../store/preferences")>();
  return {
    ...actual,
    DEFAULT_AGENT_PREFERENCES: {
      ...actual.DEFAULT_AGENT_PREFERENCES,
      get tokenLimitPerRun() {
        return preferenceDefaults.tokenLimitPerRun;
      },
    },
  };
});

vi.mock("@/api", () => ({
  aiServiceClientConnect: {
    chat: vi.fn(),
  },
}));

describe("runAgentLoop", () => {
  beforeEach(() => {
    vi.mocked(aiServiceClientConnect.chat).mockReset();
    preferenceDefaults.tokenLimitPerRun = 100_000;
  });

  test("settles a whole tool group before pausing without fabricating an answer", async () => {
    vi.mocked(aiServiceClientConnect.chat).mockResolvedValueOnce({
      content: "",
      toolCalls: [
        { id: "one", name: "lookup", arguments: "{}" },
        { id: "two", name: "lookup", arguments: "{}" },
      ],
      usage: { totalTokens: 120 },
    } as never);
    const settled: AgentModelAttempt[] = [];
    const results: string[] = [];
    const onUsage = vi.fn();
    const onText = vi.fn();
    let reservations = 0;
    const outcome = await runAgentLoop(
      [{ role: "user", content: "Inspect" }],
      [],
      async () => ({ kind: "tool_result", result: "ok" }),
      {
        onUsage,
        onText,
        onToolResult: (id) => {
          results.push(id);
        },
      },
      undefined,
      {
        runId: "run",
        maxTotalTokens: 1,
        reserveProviderAttempt: async () =>
          ++reservations === 1
            ? { kind: "allowed", attemptId: "run:1", ordinal: 1 }
            : { kind: "token-limit" },
        onAttemptSettled: async (attempt) => {
          settled.push(attempt);
          return true;
        },
      }
    );
    expect(outcome).toEqual({
      kind: "paused",
      reason: "token-limit",
      phase: "working",
      totalTokensUsed: 120,
    });
    expect(results).toEqual(["one", "two"]);
    expect(settled).toMatchObject([
      {
        attemptId: "run:1",
        runId: "run",
        ordinal: 1,
        outcome: "response",
        usage: { totalTokens: 120 },
      },
    ]);
    expect(onUsage).not.toHaveBeenCalled();
    expect(onText).not.toHaveBeenCalled();
    expect(aiServiceClientConnect.chat).toHaveBeenCalledTimes(1);
  });

  test("reserves retries separately and pauses if the next retry is refused", async () => {
    vi.useFakeTimers();
    try {
      vi.mocked(aiServiceClientConnect.chat).mockRejectedValue(
        new Error("503 private endpoint")
      );
      const settled: AgentModelAttempt[] = [];
      let reservations = 0;
      const run = runAgentLoop(
        [{ role: "user", content: "Inspect" }],
        [],
        vi.fn(),
        undefined,
        undefined,
        {
          runId: "run",
          reserveProviderAttempt: async () =>
            ++reservations === 1
              ? { kind: "allowed", attemptId: "run:1", ordinal: 1 }
              : { kind: "token-limit" },
          onAttemptSettled: async (attempt) => {
            settled.push(attempt);
            return true;
          },
        }
      );
      await vi.runAllTimersAsync();
      expect(await run).toMatchObject({
        kind: "paused",
        reason: "token-limit",
      });
      expect(settled).toEqual([
        { attemptId: "run:1", runId: "run", ordinal: 1, outcome: "error" },
      ]);
      expect(aiServiceClientConnect.chat).toHaveBeenCalledTimes(1);
    } finally {
      vi.useRealTimers();
    }
  });

  test("reconstructs a finalization-only request from its checkpoint", async () => {
    vi.mocked(aiServiceClientConnect.chat).mockResolvedValueOnce({
      content: "",
      toolCalls: [{ id: "done", name: "done", arguments: "{}" }],
      usage: { totalTokens: 5 },
    } as never);
    const outcome = await runAgentLoop(
      [{ role: "user", content: "Inspect" }],
      [
        { name: "lookup", description: "Lookup", parametersSchema: {} },
        { name: "done", description: "Done", parametersSchema: {} },
      ],
      async () => ({ kind: "done", text: "Finished", success: true }),
      undefined,
      undefined,
      {
        checkpoint: {
          phase: "finalizing",
          finalizationText: "The evidence is ready",
        },
      }
    );
    expect(outcome).toMatchObject({ kind: "completed", success: true });
    const request = vi.mocked(aiServiceClientConnect.chat).mock.calls[0]![0];
    expect(request.toolDefinitions?.map((tool) => tool.name)).toEqual(["done"]);
    expect(request.messages?.map((message) => message.content)).toContain(
      "The evidence is ready"
    );
    expect(request.messages?.at(-1)?.content).toContain(
      "Finalize your preceding answer"
    );
  });

  test("refuses side effects when the durable tool boundary fails", async () => {
    vi.mocked(aiServiceClientConnect.chat).mockResolvedValueOnce({
      content: "",
      toolCalls: [{ id: "write", name: "write", arguments: "{}" }],
    } as never);
    const executeTool = vi.fn();
    const outcome = await runAgentLoop(
      [{ role: "user", content: "Inspect" }],
      [],
      executeTool,
      undefined,
      undefined,
      {
        beforeToolExecution: async () => false,
      }
    );
    expect(outcome).toMatchObject({ kind: "error" });
    expect(executeTool).not.toHaveBeenCalled();
  });

  test("does not call the provider after cancellation during reservation", async () => {
    const controller = new AbortController();
    const outcome = await runAgentLoop(
      [],
      [],
      vi.fn(),
      undefined,
      controller.signal,
      {
        runId: "run",
        reserveProviderAttempt: async () => {
          controller.abort();
          return { kind: "allowed", attemptId: "run:1", ordinal: 1 };
        },
      }
    );
    expect(outcome).toMatchObject({ kind: "aborted" });
    expect(aiServiceClientConnect.chat).not.toHaveBeenCalled();
  });

  test("finishes a terminal tool group even when the response crosses the token limit", async () => {
    vi.mocked(aiServiceClientConnect.chat).mockResolvedValueOnce({
      content: "",
      toolCalls: [
        { id: "done", name: "done", arguments: "{}" },
        { id: "skipped", name: "write", arguments: "{}" },
      ],
      usage: { totalTokens: 120 },
    } as never);
    let calls = 0;
    const results: string[] = [];
    const outcome = await runAgentLoop(
      [],
      [],
      async () => ({ kind: "done", text: "Finished", success: true }),
      {
        onToolResult: (id) => {
          results.push(id);
        },
      },
      undefined,
      {
        runId: "run",
        maxTotalTokens: 1,
        reserveProviderAttempt: async () =>
          ++calls === 1
            ? { kind: "allowed", attemptId: "run:1", ordinal: 1 }
            : { kind: "token-limit" },
        onAttemptSettled: async () => true,
      }
    );
    expect(outcome).toMatchObject({
      kind: "completed",
      success: true,
      totalTokensUsed: 120,
    });
    expect(results).toEqual(["done", "skipped"]);
    expect(calls).toBe(1);
  });

  test("does not execute tools after a late response loses invocation ownership", async () => {
    vi.mocked(aiServiceClientConnect.chat).mockResolvedValueOnce({
      content: "",
      toolCalls: [{ id: "write", name: "write", arguments: "{}" }],
      usage: { totalTokens: 7 },
    } as never);
    const executeTool = vi.fn();
    const onAssistantMessage = vi.fn();
    const settled: AgentModelAttempt[] = [];
    const outcome = await runAgentLoop(
      [],
      [],
      executeTool,
      { onAssistantMessage },
      undefined,
      {
        runId: "run",
        reserveProviderAttempt: async () => ({
          kind: "allowed",
          attemptId: "run:1",
          ordinal: 1,
        }),
        onAttemptSettled: async (attempt) => {
          settled.push(attempt);
          return false;
        },
      }
    );
    expect(outcome).toMatchObject({ kind: "aborted", totalTokensUsed: 7 });
    expect(settled).toMatchObject([
      { outcome: "response", usage: { totalTokens: 7 } },
    ]);
    expect(executeTool).not.toHaveBeenCalled();
    expect(onAssistantMessage).not.toHaveBeenCalled();
  });

  test("settles failed transport after cancellation as unknown aborted consumption", async () => {
    const controller = new AbortController();
    vi.mocked(aiServiceClientConnect.chat).mockImplementationOnce(async () => {
      controller.abort();
      throw new Error("private endpoint closed");
    });
    const settled: AgentModelAttempt[] = [];
    const outcome = await runAgentLoop(
      [],
      [],
      vi.fn(),
      undefined,
      controller.signal,
      {
        runId: "run",
        reserveProviderAttempt: async () => ({
          kind: "allowed",
          attemptId: "run:1",
          ordinal: 1,
        }),
        onAttemptSettled: async (attempt) => {
          settled.push(attempt);
          return true;
        },
      }
    );
    expect(outcome).toMatchObject({ kind: "aborted" });
    expect(settled).toEqual([
      { attemptId: "run:1", runId: "run", ordinal: 1, outcome: "aborted" },
    ]);
  });

  test("saves the finalization phase before a token pause", async () => {
    vi.mocked(aiServiceClientConnect.chat).mockResolvedValueOnce({
      content: "Evidence is ready",
      toolCalls: [],
      usage: { totalTokens: 12 },
    } as never);
    let calls = 0;
    const checkpoints: unknown[] = [];
    const outcome = await runAgentLoop(
      [],
      [{ name: "done", description: "Done", parametersSchema: {} }],
      vi.fn(),
      undefined,
      undefined,
      {
        runId: "run",
        reserveProviderAttempt: async () =>
          ++calls === 1
            ? { kind: "allowed", attemptId: "run:1", ordinal: 1 }
            : { kind: "token-limit" },
        onAttemptSettled: async () => true,
        onCheckpoint: async (checkpoint) => {
          checkpoints.push(checkpoint);
          return true;
        },
      }
    );
    expect(outcome).toMatchObject({
      kind: "paused",
      phase: "finalizing",
      finalizationText: "Evidence is ready",
    });
    expect(checkpoints).toContainEqual({
      phase: "finalizing",
      finalizationText: "Evidence is ready",
    });
  });

  test.each(["attempt-limit", "storage-unavailable", "superseded"] as const)(
    "diagnoses %s reservation independently of token exhaustion",
    async (kind) => {
      const onText = vi.fn();
      const outcome = await runAgentLoop(
        [],
        [],
        vi.fn(),
        { onText },
        undefined,
        {
          runId: "run",
          reserveProviderAttempt: async () => ({ kind }),
        }
      );
      expect(outcome.kind).toBe(
        kind === "attempt-limit"
          ? "stopped"
          : kind === "superseded"
            ? "aborted"
            : "error"
      );
      if (kind === "attempt-limit")
        expect(outcome).toEqual({
          kind: "stopped",
          reason: "provider-attempt-limit",
          totalTokensUsed: 0,
        });
      expect(onText).not.toHaveBeenCalled();
      expect(aiServiceClientConnect.chat).not.toHaveBeenCalled();
    }
  );

  test("retains settled tools and usage when the next provider attempt is refused", async () => {
    vi.mocked(aiServiceClientConnect.chat).mockResolvedValueOnce({
      content: "",
      toolCalls: [
        { id: "one", name: "lookup", arguments: "{}" },
        { id: "two", name: "lookup", arguments: "{}" },
      ],
      usage: { totalTokens: 120 },
    } as never);
    const settled: AgentModelAttempt[] = [];
    const results: string[] = [];
    const onText = vi.fn();
    let reservations = 0;
    const outcome = await runAgentLoop(
      [{ role: "user", content: "Inspect" }],
      [],
      async () => ({ kind: "tool_result", result: "ok" }),
      {
        onText,
        onToolResult: (id) => {
          results.push(id);
        },
      },
      undefined,
      {
        runId: "run",
        reserveProviderAttempt: async () =>
          ++reservations === 1
            ? { kind: "allowed", attemptId: "run:1", ordinal: 1 }
            : { kind: "attempt-limit" },
        onAttemptSettled: async (attempt) => {
          settled.push(attempt);
          return true;
        },
      }
    );
    expect(outcome).toEqual({
      kind: "stopped",
      reason: "provider-attempt-limit",
      totalTokensUsed: 120,
    });
    expect(results).toEqual(["one", "two"]);
    expect(settled).toMatchObject([
      { outcome: "response", usage: { totalTokens: 120 } },
    ]);
    expect(onText).not.toHaveBeenCalled();
    expect(aiServiceClientConnect.chat).toHaveBeenCalledTimes(1);
  });

  test("traces sanitized terminal counts without including settlement latency or payloads", async () => {
    let now = 100;
    const clock = vi.spyOn(performance, "now").mockImplementation(() => now);
    try {
      vi.mocked(aiServiceClientConnect.chat).mockImplementationOnce(
        async () => {
          now = 125;
          return {
            content: "private SQL",
            toolCalls: [],
            usage: {
              totalTokens: 7,
              inputTokens: 5,
              outputTokens: 2,
              cachedInputTokens: 3,
            },
            stopReason: AIChatResponse_StopReason.TOKEN_LIMIT,
          } as never;
        }
      );
      const records: unknown[] = [];
      await runAgentLoop(
        [{ role: "user", content: "private question" }],
        [],
        vi.fn(),
        {
          onProviderAttempt: (record) => records.push(record),
        },
        undefined,
        {
          runId: "run",
          allowPlainTextCompletion: true,
          reserveProviderAttempt: async () => ({
            kind: "allowed",
            attemptId: "run:1",
            ordinal: 1,
          }),
          onAttemptSettled: async () => {
            now = 500;
            return true;
          },
        }
      );
      expect(records).toMatchObject([
        {
          attemptId: "run:1",
          runId: "run",
          ordinal: 1,
          outcome: "response",
          phase: "working",
          roundTripMs: 25,
          stopReason: "token-limit",
          usage: { totalTokens: 7, cachedInputTokens: 3 },
          request: { messageCount: 1, toolCount: 0 },
        },
      ]);
      expect(JSON.stringify(records)).not.toContain("private");
      expect(JSON.stringify(records)).not.toContain("toolNames");
    } finally {
      clock.mockRestore();
    }
  });

  test("does not count a truncated or refused answer as a successful plain-text completion", async () => {
    vi.mocked(aiServiceClientConnect.chat).mockResolvedValueOnce({
      content: "Incomplete answer",
      toolCalls: [],
      usage: { totalTokens: 4 },
      stopReason: AIChatResponse_StopReason.TOKEN_LIMIT,
    } as never);
    const outcome = await runAgentLoop([], [], vi.fn(), undefined, undefined, {
      allowPlainTextCompletion: true,
    });
    expect(outcome).toMatchObject({ kind: "error", totalTokensUsed: 4 });
  });

  test("assigns distinct trace identities to standalone invocations", async () => {
    vi.mocked(aiServiceClientConnect.chat).mockResolvedValue({
      content: "Done",
      toolCalls: [],
    } as never);
    const identities: string[] = [];
    for (let index = 0; index < 2; index++) {
      await runAgentLoop(
        [],
        [],
        vi.fn(),
        { onProviderAttempt: (attempt) => identities.push(attempt.attemptId) },
        undefined,
        { allowPlainTextCompletion: true }
      );
    }
    expect(new Set(identities).size).toBe(2);
  });

  test("retains a known managed side-effect result after Stop without running downstream work", async () => {
    vi.mocked(aiServiceClientConnect.chat).mockResolvedValueOnce({
      content: "",
      toolCalls: [
        { id: "write", name: "write", arguments: "{}" },
        { id: "later", name: "write", arguments: "{}" },
      ],
      usage: { totalTokens: 7 },
    } as never);
    const controller = new AbortController();
    let resolveTool!: (result: ToolExecutionResult) => void;
    let releaseResult!: () => void;
    const executeTool: ToolExecutor = vi.fn(
      () =>
        new Promise<ToolExecutionResult>((resolve) => {
          resolveTool = resolve;
        })
    );
    const retained: string[] = [];
    const onText = vi.fn();
    const onToolResult = vi.fn(async (id: string, result: string) => {
      await new Promise<void>((resolve) => {
        releaseResult = resolve;
      });
      retained.push(`${id}:${result}`);
    });
    let finished = false;
    const run = runAgentLoop(
      [],
      [],
      executeTool,
      { onToolResult, onText },
      controller.signal,
      {
        runId: "run",
        reserveProviderAttempt: async () => ({
          kind: "allowed",
          attemptId: "run:1",
          ordinal: 1,
        }),
        onAttemptSettled: async () => true,
      }
    ).then((outcome) => {
      finished = true;
      return outcome;
    });
    await vi.waitFor(() => expect(executeTool).toHaveBeenCalledTimes(1));
    controller.abort();
    resolveTool({ kind: "tool_result", result: "write completed" });
    await vi.waitFor(() =>
      expect(onToolResult).toHaveBeenCalledExactlyOnceWith(
        "write",
        "write completed"
      )
    );
    expect(finished).toBe(false);
    releaseResult();
    expect(await run).toMatchObject({ kind: "aborted", totalTokensUsed: 7 });
    expect(retained).toEqual(["write:write completed"]);
    expect(executeTool).toHaveBeenCalledTimes(1);
    expect(aiServiceClientConnect.chat).toHaveBeenCalledTimes(1);
    expect(onText).not.toHaveBeenCalled();
  });

  test.each([false, true])(
    "reports durable settlement failure distinctly from supersession even when stopped: %s",
    async (stop) => {
      vi.mocked(aiServiceClientConnect.chat).mockResolvedValueOnce({
        content: "",
        toolCalls: [{ id: "write", name: "write", arguments: "{}" }],
        usage: { totalTokens: 7 },
      } as never);
      const controller = new AbortController();
      const executeTool = vi.fn();
      const outcome = await runAgentLoop(
        [],
        [],
        executeTool,
        undefined,
        controller.signal,
        {
          runId: "run",
          reserveProviderAttempt: async () => ({
            kind: "allowed",
            attemptId: "run:1",
            ordinal: 1,
          }),
          onAttemptSettled: async () => {
            if (stop) controller.abort();
            throw new Error("storage unavailable");
          },
        }
      );
      expect(outcome).toMatchObject({
        kind: "error",
        totalTokensUsed: 7,
        error: expect.objectContaining({
          message: "Agent attempt could not be saved.",
        }),
      });
      expect(executeTool).not.toHaveBeenCalled();
      expect(aiServiceClientConnect.chat).toHaveBeenCalledTimes(1);
    }
  );

  test("treats a non-AbortError response after user cancellation as aborted", async () => {
    let rejectChat!: (error: Error) => void;
    vi.mocked(aiServiceClientConnect.chat).mockImplementation(
      () =>
        new Promise((_, reject) => {
          rejectChat = reject;
        })
    );
    const controller = new AbortController();
    const onUsage = vi.fn();
    const run = runAgentLoop(
      [{ role: "user", content: "Check the project" }],
      [],
      vi.fn(),
      { onUsage },
      controller.signal
    );
    await vi.waitFor(() =>
      expect(aiServiceClientConnect.chat).toHaveBeenCalled()
    );
    controller.abort();
    rejectChat(new Error("Transport closed"));

    expect(await run).toEqual({ kind: "aborted", totalTokensUsed: 0 });
    expect(onUsage).not.toHaveBeenCalled();
  });

  test("does not execute tool calls returned after cancellation", async () => {
    let resolveChat!: (
      response: Awaited<ReturnType<typeof aiServiceClientConnect.chat>>
    ) => void;
    vi.mocked(aiServiceClientConnect.chat).mockImplementation(
      () =>
        new Promise((resolve) => {
          resolveChat = resolve;
        })
    );
    const controller = new AbortController();
    const executeTool = vi.fn();
    const onAssistantMessage = vi.fn();
    const onUsage = vi.fn();
    const run = runAgentLoop(
      [{ role: "user", content: "Check the project" }],
      [{ name: "test_tool", description: "test", parametersSchema: {} }],
      executeTool,
      { onAssistantMessage, onUsage },
      controller.signal
    );
    await vi.waitFor(() =>
      expect(aiServiceClientConnect.chat).toHaveBeenCalled()
    );
    controller.abort();
    resolveChat({
      content: "",
      toolCalls: [
        { id: "tool-1", name: "test_tool", arguments: "{}", metadata: "" },
      ],
      usage: { totalTokens: 7, inputTokens: 5, outputTokens: 2 },
    } as never);

    expect(await run).toEqual({ kind: "aborted", totalTokensUsed: 7 });
    expect(onUsage).toHaveBeenCalledExactlyOnceWith({
      purpose: "agent",
      toolNames: ["test_tool"],
      totalTokens: 7,
      inputTokens: 5,
      outputTokens: 2,
    });
    expect(executeTool).not.toHaveBeenCalled();
    expect(onAssistantMessage).not.toHaveBeenCalled();
  });

  test.each([
    {
      name: "tool result",
      result: { kind: "tool_result", result: "late result" },
    },
    {
      name: "completion",
      result: { kind: "done", text: "Finished", success: true },
    },
    {
      name: "request for input",
      result: {
        kind: "ask_user",
        ask: { toolCallId: "tool-1", prompt: "Which database?", kind: "input" },
      },
    },
  ] satisfies { name: string; result: ToolExecutionResult }[])(
    "does not publish a late $name after aborting a tool",
    async ({ result }) => {
      vi.mocked(aiServiceClientConnect.chat).mockResolvedValue({
        content: "",
        toolCalls: [
          {
            id: "tool-1",
            name: "test_tool",
            arguments: "{}",
            metadata: "",
          },
        ],
      } as never);
      let resolveTool!: (result: ToolExecutionResult) => void;
      const executeTool: ToolExecutor = vi.fn(
        () =>
          new Promise<ToolExecutionResult>((resolve) => {
            resolveTool = resolve;
          })
      );
      const controller = new AbortController();
      const onToolResult = vi.fn();
      const onText = vi.fn();
      const onError = vi.fn();

      const run = runAgentLoop(
        [{ role: "user", content: "Do the task" }],
        [{ name: "test_tool", description: "test", parametersSchema: {} }],
        executeTool,
        { onToolResult, onText, onError },
        controller.signal
      );
      await vi.waitFor(() => expect(executeTool).toHaveBeenCalledTimes(1));
      controller.abort();
      resolveTool(result);

      expect(await run).toEqual({ kind: "aborted", totalTokensUsed: 0 });
      expect(onToolResult).not.toHaveBeenCalled();
      expect(onText).not.toHaveBeenCalled();
      expect(onError).not.toHaveBeenCalled();
      expect(aiServiceClientConnect.chat).toHaveBeenCalledTimes(1);
    }
  );

  test("returns a completed outcome for explicit done", async () => {
    vi.mocked(aiServiceClientConnect.chat).mockResolvedValue({
      content: "",
      toolCalls: [
        {
          id: "tool-1",
          name: "done",
          arguments: JSON.stringify({
            text: "Finished successfully",
            success: true,
          }),
          metadata: "",
        },
      ],
    } as never);

    const executeTool: ToolExecutor = vi.fn(
      async (_name, args: Record<string, unknown>) => ({
        kind: "done" as const,
        text: String(args.text),
        success: args.success !== false,
      })
    );
    const onText = vi.fn();
    const onAssistantMessage = vi.fn();
    const onToolResult = vi.fn();

    const outcome = await runAgentLoop(
      [{ role: "system", content: "system" }],
      [],
      executeTool,
      { onText, onAssistantMessage, onToolResult }
    );

    expect(outcome).toEqual({
      kind: "completed",
      text: "Finished successfully",
      success: true,
      totalTokensUsed: 0,
    });
    expect(onToolResult).toHaveBeenCalledWith(
      "tool-1",
      JSON.stringify({
        text: "Finished successfully",
        success: true,
      })
    );
    expect(onText).toHaveBeenCalledWith("Finished successfully");
    expect(onAssistantMessage).toHaveBeenCalledWith({
      role: "assistant",
      content: "",
      toolCalls: [
        {
          id: "tool-1",
          name: "done",
          arguments: JSON.stringify({
            text: "Finished successfully",
            success: true,
          }),
          metadata: "",
        },
      ],
    });
  });

  test("forwards a successful handoff with the final answer", async () => {
    vi.mocked(aiServiceClientConnect.chat).mockResolvedValue({
      content: "",
      toolCalls: [
        { id: "done-1", name: "done", arguments: "{}", metadata: "" },
      ],
    } as never);
    const handoff = {
      kind: "review-project-issues" as const,
      project: "projects/sample",
    };
    const onText = vi.fn();
    const outcome = await runAgentLoop(
      [{ role: "system", content: "system" }],
      [],
      vi.fn(async () => ({
        kind: "done" as const,
        text: "Review the queue",
        success: true,
        handoff,
      })),
      { onText }
    );
    expect(outcome).toEqual({
      kind: "completed",
      text: "Review the queue",
      success: true,
      handoff,
      totalTokensUsed: 0,
    });
    expect(onText).toHaveBeenCalledWith("Review the queue", handoff);
  });

  test("forwards response actions only for a successful final answer", async () => {
    vi.mocked(aiServiceClientConnect.chat).mockResolvedValue({
      content: "",
      toolCalls: [
        { id: "done-1", name: "done", arguments: "{}", metadata: "" },
      ],
    } as never);
    const actions = [
      {
        kind: "open" as const,
        target: {
          kind: "review-project-issues" as const,
          project: "projects/sample",
        },
      },
      { kind: "continue" as const, prompt: "What should I check next?" },
    ];
    const outcome = await runAgentLoop(
      [{ role: "system", content: "system" }],
      [],
      vi.fn(async () => ({
        kind: "done" as const,
        text: "Review the queue",
        success: true,
        actions,
      }))
    );
    expect(outcome).toMatchObject({ kind: "completed", actions });
  });

  test("drops a proposed handoff when the final answer fails", async () => {
    vi.mocked(aiServiceClientConnect.chat).mockResolvedValue({
      content: "",
      toolCalls: [
        { id: "done-1", name: "done", arguments: "{}", metadata: "" },
      ],
    } as never);
    const onText = vi.fn();
    const outcome = await runAgentLoop(
      [{ role: "system", content: "system" }],
      [],
      vi.fn(async () => ({
        kind: "done" as const,
        text: "The request failed",
        success: false,
        handoff: {
          kind: "review-project-issues" as const,
          project: "projects/sample",
        },
      })),
      { onText }
    );
    expect(outcome).toEqual({
      kind: "completed",
      text: "The request failed",
      success: false,
      totalTokensUsed: 0,
    });
    expect(onText).toHaveBeenCalledExactlyOnceWith("The request failed");
  });

  test("accumulates token usage across repeated provider calls", async () => {
    vi.mocked(aiServiceClientConnect.chat)
      .mockResolvedValueOnce({
        content: "",
        toolCalls: [
          {
            id: "tool-1",
            name: "search_api",
            arguments: JSON.stringify({ service: "SQLService" }),
            metadata: "",
          },
        ],
        usage: {
          totalTokens: 11,
          inputTokens: 9,
          outputTokens: 2,
          cachedInputTokens: 3,
        },
      } as never)
      .mockResolvedValueOnce({
        content: "",
        toolCalls: [
          {
            id: "tool-2",
            name: "done",
            arguments: JSON.stringify({
              text: "Final answer",
              success: true,
            }),
            metadata: "",
          },
        ],
        usage: {
          totalTokens: 7,
        },
      } as never);

    const executeTool: ToolExecutor = vi.fn(async (name, args) => {
      if (name === "done") {
        return {
          kind: "done" as const,
          text: String(args.text),
          success: args.success !== false,
        };
      }
      return {
        kind: "tool_result" as const,
        result: JSON.stringify({ ok: true }),
      };
    });

    const onUsage = vi.fn();
    const outcome = await runAgentLoop(
      [{ role: "system", content: "system" }],
      [],
      executeTool,
      { onUsage }
    );

    expect(outcome).toEqual({
      kind: "completed",
      text: "Final answer",
      success: true,
      totalTokensUsed: 18,
    });
    expect(onUsage.mock.calls).toEqual([
      [
        {
          purpose: "agent",
          toolNames: ["search_api"],
          totalTokens: 11,
          inputTokens: 9,
          outputTokens: 2,
          cachedInputTokens: 3,
        },
      ],
      [{ purpose: "agent", toolNames: ["done"], totalTokens: 7 }],
    ]);
  });

  test.each([
    { usage: undefined, expected: {} },
    {
      usage: {
        totalTokens: 0,
        inputTokens: 0,
        outputTokens: 0,
        cachedInputTokens: 0,
      },
      expected: {
        totalTokens: 0,
        inputTokens: 0,
        outputTokens: 0,
        cachedInputTokens: 0,
      },
    },
  ])(
    "reports unknown usage separately from explicit zero: $usage",
    async ({ usage, expected }) => {
      vi.mocked(aiServiceClientConnect.chat).mockResolvedValue({
        content: "Done",
        toolCalls: [],
        usage,
      } as never);
      const onUsage = vi.fn();

      await runAgentLoop(
        [{ role: "user", content: "ping" }],
        [],
        vi.fn(),
        { onUsage },
        undefined,
        { allowPlainTextCompletion: true }
      );

      expect(onUsage).toHaveBeenCalledExactlyOnceWith({
        purpose: "agent",
        toolNames: [],
        ...expected,
      });
    }
  );

  test.each([
    { limit: undefined, usage: 100_000, calls: 1, success: false },
    { limit: undefined, usage: 60_000, calls: 2, success: true },
    { limit: 1_200, usage: 1_200, calls: 1, success: false },
    { limit: 90_000, usage: 60_000, calls: 2, success: true },
    { limit: 0, usage: 100_000, calls: 1, success: false },
    { limit: -1, usage: 100_000, calls: 1, success: false },
    { limit: 1.5, usage: 100_000, calls: 1, success: false },
    { limit: Number.NaN, usage: 100_000, calls: 1, success: false },
    {
      limit: Number.POSITIVE_INFINITY,
      usage: 100_000,
      calls: 1,
      success: false,
    },
    {
      limit: Number.MAX_SAFE_INTEGER + 1,
      usage: 100_000,
      calls: 1,
      success: false,
    },
  ])(
    "applies per-run budget $limit after provider usage $usage",
    async ({ limit, usage, calls, success }) => {
      vi.mocked(aiServiceClientConnect.chat)
        .mockResolvedValueOnce({
          content: "",
          toolCalls: [
            {
              id: "tool-1",
              name: "lookup",
              arguments: "{}",
              metadata: "",
            },
          ],
          usage: { totalTokens: usage },
        } as never)
        .mockResolvedValueOnce({
          content: "",
          toolCalls: [
            {
              id: "done-1",
              name: "done",
              arguments: JSON.stringify({ text: "Done", success: true }),
              metadata: "",
            },
          ],
        } as never);

      const outcome = await runAgentLoop(
        [
          { role: "system", content: "system" },
          { role: "user", content: "Inspect" },
        ],
        [{ name: "lookup", description: "Lookup", parametersSchema: {} }],
        vi.fn(async (name, args) =>
          name === "done"
            ? { kind: "done" as const, text: String(args.text), success: true }
            : { kind: "tool_result" as const, result: "ok" }
        ),
        undefined,
        undefined,
        { maxTotalTokens: limit }
      );

      expect(outcome.kind).toBe("completed");
      expect(outcome).toMatchObject({ success, totalTokensUsed: usage });
      expect(aiServiceClientConnect.chat).toHaveBeenCalledTimes(calls);
    }
  );

  test("refuses a request that exceeds the serialized payload budget", async () => {
    const outcome = await runAgentLoop(
      [
        { role: "system", content: "s".repeat(300_000) },
        { role: "user", content: "Inspect" },
      ],
      [],
      vi.fn()
    );

    expect(outcome).toMatchObject({ kind: "completed", success: false });
    expect(aiServiceClientConnect.chat).not.toHaveBeenCalled();
  });

  test("uses the preference default when no per-run token budget is supplied", async () => {
    preferenceDefaults.tokenLimitPerRun = 24_000;
    vi.mocked(aiServiceClientConnect.chat)
      .mockResolvedValueOnce({
        content: "",
        toolCalls: [{ id: "lookup", name: "lookup", arguments: "{}" }],
        usage: { totalTokens: 24_000 },
      } as never)
      .mockResolvedValueOnce({ content: "Done", toolCalls: [] } as never);

    const outcome = await runAgentLoop(
      [{ role: "user", content: "Inspect" }],
      [{ name: "lookup", description: "Lookup", parametersSchema: {} }],
      vi.fn(async () => ({ kind: "tool_result" as const, result: "ok" })),
      undefined,
      undefined,
      { allowPlainTextCompletion: true }
    );

    expect(outcome).toMatchObject({ success: false, totalTokensUsed: 24_000 });
    expect(aiServiceClientConnect.chat).toHaveBeenCalledTimes(1);
  });

  test("prunes completed tool output while retaining the current question", async () => {
    vi.mocked(aiServiceClientConnect.chat).mockResolvedValueOnce({
      content: "",
      toolCalls: [
        {
          id: "done-1",
          name: "done",
          arguments: JSON.stringify({ text: "Finished", success: true }),
          metadata: "",
        },
      ],
    } as never);
    const largeResult = "schema".repeat(50_000);
    const outcome = await runAgentLoop(
      [
        { role: "system", content: "system" },
        { role: "user", content: "Inspect the schema" },
        {
          role: "assistant",
          content: "Reading schema",
          toolCalls: [{ id: "old-tool", name: "get_schema", arguments: "{}" }],
        },
        { role: "tool", toolCallId: "old-tool", content: largeResult },
        { role: "user", content: "Now summarize it" },
      ],
      [],
      vi.fn(async () => ({
        kind: "done" as const,
        text: "Finished",
        success: true,
      }))
    );

    expect(outcome).toMatchObject({ kind: "completed", success: true });
    const request = vi.mocked(aiServiceClientConnect.chat).mock.calls[0]?.[0];
    const sentMessages = request?.messages?.map((message) => message.content);
    expect(sentMessages).toContain("Now summarize it");
    expect(sentMessages?.join(" ")).toContain(
      "omitted to fit the context limit"
    );
    expect(sentMessages?.join(" ")).not.toContain(largeResult);
  });

  test("limits provider attempts within one agent run", async () => {
    vi.mocked(aiServiceClientConnect.chat).mockImplementation(async () => {
      const callNumber = vi.mocked(aiServiceClientConnect.chat).mock.calls
        .length;
      return {
        content: "",
        toolCalls: [
          {
            id: `tool-${callNumber}`,
            name: "lookup",
            arguments: "{}",
            metadata: "",
          },
        ],
        usage: { totalTokens: 5 },
      } as never;
    });

    const results: string[] = [];
    const onText = vi.fn();
    const outcome = await runAgentLoop(
      [{ role: "user", content: "Inspect" }],
      [{ name: "lookup", description: "Lookup", parametersSchema: {} }],
      vi.fn(async () => ({ kind: "tool_result" as const, result: "ok" })),
      {
        onText,
        onToolResult: (id) => {
          results.push(id);
        },
      }
    );

    expect(outcome).toEqual({
      kind: "stopped",
      reason: "provider-attempt-limit",
      totalTokensUsed: 60,
    });
    expect(results).toHaveLength(12);
    expect(onText).not.toHaveBeenCalled();
    expect(aiServiceClientConnect.chat).toHaveBeenCalledTimes(12);
  });

  test("counts retries toward the per-run provider attempt limit", async () => {
    vi.useFakeTimers();
    try {
      vi.mocked(aiServiceClientConnect.chat).mockImplementation(async () => {
        const attempt = vi.mocked(aiServiceClientConnect.chat).mock.calls
          .length;
        if (attempt % 2 === 1) throw new Error("503 temporary failure");
        return {
          content: "",
          toolCalls: [
            { id: `lookup-${attempt}`, name: "lookup", arguments: "{}" },
          ],
        } as never;
      });
      const executeTool = vi.fn(async () => ({
        kind: "tool_result" as const,
        result: "ok",
      }));
      const run = runAgentLoop(
        [{ role: "user", content: "Inspect" }],
        [{ name: "lookup", description: "Lookup", parametersSchema: {} }],
        executeTool
      );
      await vi.runAllTimersAsync();

      expect(await run).toMatchObject({
        kind: "stopped",
        reason: "provider-attempt-limit",
      });
      expect(aiServiceClientConnect.chat).toHaveBeenCalledTimes(12);
      expect(executeTool).toHaveBeenCalledTimes(6);
    } finally {
      vi.useRealTimers();
    }
  });

  test("resolves loaded guidance and terminal tools together before each request", async () => {
    let loaded = false;
    vi.mocked(aiServiceClientConnect.chat)
      .mockResolvedValueOnce({
        content: "",
        toolCalls: [{ id: "load", name: "get_skill", arguments: "{}" }],
      } as never)
      .mockResolvedValueOnce({
        content: "Ready to clarify",
        toolCalls: [],
      } as never)
      .mockResolvedValueOnce({
        content: "",
        toolCalls: [
          { id: "scope", name: "ask_masking_rule_scope", arguments: "{}" },
        ],
      } as never);
    const onRequestDiagnostics = vi.fn();
    const outcome = await runAgentLoop(
      [
        { role: "system", content: "Core safety" },
        { role: "user", content: "Help" },
      ],
      [],
      vi.fn(async (name) => {
        if (name === "get_skill") {
          loaded = true;
          return { kind: "tool_result" as const, result: "loaded" };
        }
        return {
          kind: "ask_user" as const,
          ask: {
            toolCallId: "scope",
            prompt: "Which scope?",
            kind: "input" as const,
          },
        };
      }),
      { onRequestDiagnostics },
      undefined,
      {
        resolveRequestContext: () => ({
          systemGuidance: loaded ? "Registered masking rules" : "",
          tools: [
            { name: "get_skill", description: "Load", parametersSchema: {} },
            { name: "done", description: "Finish", parametersSchema: {} },
            ...(loaded
              ? [
                  {
                    name: "ask_masking_rule_scope",
                    description: "Scope",
                    parametersSchema: {},
                  },
                ]
              : []),
          ],
        }),
      }
    );

    expect(outcome.kind).toBe("awaiting_user");
    const requests = vi
      .mocked(aiServiceClientConnect.chat)
      .mock.calls.map(([request]) => request);
    expect(requests[0]?.messages?.[0]?.content).toBe("Core safety");
    for (const request of requests.slice(1)) {
      expect(
        request.messages?.filter(
          (message) =>
            message.role === AIChatMessageRole.AI_CHAT_MESSAGE_ROLE_SYSTEM
        )
      ).toHaveLength(1);
      expect(request.messages?.[0]?.content).toBe(
        "Core safety\n\nRegistered masking rules"
      );
      expect(request.toolDefinitions?.map((tool) => tool.name)).toContain(
        "ask_masking_rule_scope"
      );
    }
    expect(
      requests[2]?.toolDefinitions?.map((tool) => tool.name)
    ).not.toContain("get_skill");
    expect(onRequestDiagnostics).toHaveBeenCalledTimes(3);
    expect(onRequestDiagnostics.mock.calls[0]![0]).toEqual({
      serializedBytes: expect.any(Number),
      systemBytes: expect.any(Number),
      conversationBytes: expect.any(Number),
      toolBytes: expect.any(Number),
      messageCount: expect.any(Number),
      toolCount: expect.any(Number),
    });
  });

  test("preserves core and registered guidance when compacting skill results", async () => {
    vi.mocked(aiServiceClientConnect.chat).mockResolvedValue({
      content: "Finished",
      toolCalls: [],
    } as never);
    await runAgentLoop(
      [
        { role: "system", content: "Core safety" },
        {
          role: "assistant",
          toolCalls: [{ id: "load", name: "get_skill", arguments: "{}" }],
        },
        { role: "tool", toolCallId: "load", content: "x".repeat(300_000) },
        { role: "user", content: "Continue" },
      ],
      [],
      vi.fn(),
      undefined,
      undefined,
      {
        allowPlainTextCompletion: true,
        resolveRequestContext: () => ({
          tools: [],
          systemGuidance: "Registered guide",
        }),
      }
    );
    const request = vi.mocked(aiServiceClientConnect.chat).mock.calls[0]![0];
    const systems = request.messages?.filter(
      (message) =>
        message.role === AIChatMessageRole.AI_CHAT_MESSAGE_ROLE_SYSTEM
    );
    expect(systems).toHaveLength(1);
    expect(systems?.[0]?.content).toContain("Core safety");
    expect(systems?.[0]?.content).toContain("Registered guide");
    expect(systems?.[0]?.content).toContain("omitted to fit the context limit");
    expect(systems?.[0]?.content?.match(/Registered guide/g)).toHaveLength(1);
  });

  test("does not drop oversized required guidance to send a request", async () => {
    const outcome = await runAgentLoop(
      [
        { role: "system", content: "Core safety" },
        { role: "user", content: "Help" },
      ],
      [],
      vi.fn(),
      undefined,
      undefined,
      {
        resolveRequestContext: () => ({
          tools: [],
          systemGuidance: "x".repeat(300_000),
        }),
      }
    );
    expect(outcome).toMatchObject({ kind: "completed", success: false });
    expect(aiServiceClientConnect.chat).not.toHaveBeenCalled();
  });

  test("replays completed tool flows with matched tool results", async () => {
    vi.mocked(aiServiceClientConnect.chat)
      .mockResolvedValueOnce({
        content: "",
        toolCalls: [
          {
            id: "tool-1",
            name: "done",
            arguments: JSON.stringify({
              text: "Finished successfully",
              success: true,
            }),
            metadata: "",
          },
        ],
      } as never)
      .mockResolvedValueOnce({
        content: "Follow-up answer",
        toolCalls: [],
      } as never);

    const executeTool: ToolExecutor = vi.fn(
      async (_name, args: Record<string, unknown>) => ({
        kind: "done" as const,
        text: String(args.text),
        success: args.success !== false,
      })
    );

    const completed = await runAgentLoop(
      [{ role: "system", content: "system" }],
      [],
      executeTool
    );
    expect(completed.kind).toBe("completed");

    await runAgentLoop(
      [
        { role: "system", content: "system" },
        {
          role: "assistant",
          content: "",
          toolCalls: [
            {
              id: "tool-1",
              name: "done",
              arguments: JSON.stringify({
                text: "Finished successfully",
                success: true,
              }),
              metadata: "",
            },
          ],
        },
        {
          role: "tool",
          toolCallId: "tool-1",
          content: JSON.stringify({
            text: "Finished successfully",
            success: true,
          }),
        },
        { role: "assistant", content: "Finished successfully" },
        { role: "user", content: "What next?" },
      ],
      [],
      vi.fn()
    );

    expect(aiServiceClientConnect.chat).toHaveBeenCalledTimes(2);
    const secondCallMessages = vi.mocked(aiServiceClientConnect.chat).mock
      .calls[1]?.[0].messages;
    expect(secondCallMessages).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          role: AIChatMessageRole.AI_CHAT_MESSAGE_ROLE_ASSISTANT,
          toolCalls: [
            expect.objectContaining({
              id: "tool-1",
              name: "done",
            }),
          ],
        }),
        expect.objectContaining({
          role: AIChatMessageRole.AI_CHAT_MESSAGE_ROLE_TOOL,
          toolCallId: "tool-1",
          content: JSON.stringify({
            text: "Finished successfully",
            success: true,
          }),
        }),
      ])
    );
  });
  test("returns an error when the assistant replies with plain text instead of done", async () => {
    vi.mocked(aiServiceClientConnect.chat).mockResolvedValue({
      content: "Here is the answer",
      toolCalls: [],
    } as never);

    const executeTool: ToolExecutor = vi.fn();
    const onText = vi.fn();
    const onError = vi.fn();

    const outcome = await runAgentLoop(
      [{ role: "system", content: "system" }],
      [],
      executeTool,
      { onText, onError }
    );

    expect(outcome.kind).toBe("error");
    if (outcome.kind !== "error") {
      throw new Error(`Expected error outcome, got ${outcome.kind}`);
    }
    expect(outcome.error.message).toBe(
      "Agent must finish with done({ text, success, actions }), ask_user(...), ask_masking_exemption_scope(...), ask_masking_rule_scope(...), or ask_masking_semantic_type(...), not plain assistant text."
    );
    expect(outcome.totalTokensUsed).toBe(0);
    expect(onError).toHaveBeenCalledWith(outcome.error);
    expect(onText).not.toHaveBeenCalled();
    expect(executeTool).not.toHaveBeenCalled();
  });

  test("asks for a terminal tool call after a plain-text reply to a database count", async () => {
    vi.mocked(aiServiceClientConnect.chat)
      .mockResolvedValueOnce({
        content: "",
        toolCalls: [
          {
            id: "count-1",
            name: "get_database_count",
            arguments: "{}",
            metadata: "",
          },
        ],
      } as never)
      .mockResolvedValueOnce({
        content: "The project has 26 active databases.",
        toolCalls: [],
      } as never)
      .mockResolvedValueOnce({
        content: "",
        toolCalls: [
          {
            id: "done-1",
            name: "done",
            arguments: JSON.stringify({
              text: "The project has 26 active databases.",
              success: true,
            }),
            metadata: "",
          },
        ],
      } as never);

    const executeTool: ToolExecutor = vi.fn(async (name, args) =>
      name === "done"
        ? {
            kind: "done" as const,
            text: String(args.text),
            success: true,
          }
        : {
            kind: "tool_result" as const,
            result: JSON.stringify({ count: 26 }),
          }
    );
    const onText = vi.fn();
    const onError = vi.fn();
    const tools = ["get_database_count", "call_api", "ask_user", "done"].map(
      (name) => ({ name, description: name, parametersSchema: {} })
    );

    const outcome = await runAgentLoop(
      [{ role: "system", content: "system" }],
      tools,
      executeTool,
      { onText, onError }
    );

    expect(outcome).toEqual({
      kind: "completed",
      text: "The project has 26 active databases.",
      success: true,
      totalTokensUsed: 0,
    });
    expect(onText).toHaveBeenCalledOnce();
    expect(onText).toHaveBeenCalledWith("The project has 26 active databases.");
    expect(onError).not.toHaveBeenCalled();
    expect(executeTool).toHaveBeenCalledTimes(2);
    expect(
      vi
        .mocked(aiServiceClientConnect.chat)
        .mock.calls[2]?.[0]?.toolDefinitions?.map((tool) => tool.name)
    ).toEqual(["ask_user", "done"]);
  });

  test("recovers an empty model response by requesting a terminal answer", async () => {
    vi.mocked(aiServiceClientConnect.chat)
      .mockResolvedValueOnce({ content: "", toolCalls: [] } as never)
      .mockResolvedValueOnce({
        content: "",
        toolCalls: [
          {
            id: "done-1",
            name: "done",
            arguments: JSON.stringify({
              text: "Issue is canceled.",
              success: true,
            }),
            metadata: "",
          },
        ],
      } as never);
    const executeTool: ToolExecutor = vi.fn(async (_name, args) => ({
      kind: "done" as const,
      text: String(args.text),
      success: true,
    }));

    const outcome = await runAgentLoop(
      [{ role: "user", content: "Summarize the selected issue" }],
      ["ask_user", "done"].map((name) => ({
        name,
        description: name,
        parametersSchema: {},
      })),
      executeTool,
      undefined,
      undefined,
      { allowPlainTextCompletion: true }
    );

    expect(outcome).toEqual({
      kind: "completed",
      text: "Issue is canceled.",
      success: true,
      totalTokensUsed: 0,
    });
    expect(aiServiceClientConnect.chat).toHaveBeenCalledTimes(2);
    expect(
      vi
        .mocked(aiServiceClientConnect.chat)
        .mock.calls[1]?.[0]?.messages?.at(-1)?.content
    ).toMatch(/previous response was empty/i);
  });

  test("still rejects plain text after one terminal-tool retry", async () => {
    vi.mocked(aiServiceClientConnect.chat).mockResolvedValue({
      content: "Answer without done",
      toolCalls: [],
    } as never);
    const executeTool: ToolExecutor = vi.fn();

    const outcome = await runAgentLoop(
      [{ role: "system", content: "system" }],
      [{ name: "done", description: "done", parametersSchema: {} }],
      executeTool
    );

    expect(outcome.kind).toBe("error");
    expect(aiServiceClientConnect.chat).toHaveBeenCalledTimes(2);
    expect(executeTool).not.toHaveBeenCalled();
  });

  test("reports a rejected API request instead of a terminal-tool error", async () => {
    const apiError = "cannot change status from DONE to OPEN for issue 101";
    vi.mocked(aiServiceClientConnect.chat)
      .mockResolvedValueOnce({
        content: "",
        toolCalls: [
          {
            id: "api-1",
            name: "call_api",
            arguments: JSON.stringify({
              operationId: "IssueService/BatchUpdateIssuesStatus",
            }),
            metadata: "",
          },
        ],
      } as never)
      .mockResolvedValue({
        content: "The issue was reopened.",
        toolCalls: [],
      } as never);
    const executeTool: ToolExecutor = vi.fn(async () => ({
      kind: "tool_result" as const,
      result: JSON.stringify({ status: 400, error: apiError }),
    }));
    const onText = vi.fn();
    const onError = vi.fn();

    const outcome = await runAgentLoop(
      [{ role: "user", content: "Reopen the issue" }],
      ["call_api", "done", "ask_user"].map((name) => ({
        name,
        description: name,
        parametersSchema: {},
      })),
      executeTool,
      { onText, onError }
    );

    expect(outcome).toEqual({
      kind: "completed",
      text: apiError,
      success: false,
      totalTokensUsed: 0,
    });
    expect(onText).toHaveBeenCalledExactlyOnceWith(apiError);
    expect(onError).not.toHaveBeenCalled();
    expect(executeTool).toHaveBeenCalledTimes(1);
    expect(aiServiceClientConnect.chat).toHaveBeenCalledTimes(3);
  });

  test("reports an API rejection after permission approval from replayed history", async () => {
    const apiError = "cannot change status from DONE to OPEN for issue 101";
    vi.mocked(aiServiceClientConnect.chat).mockResolvedValue({
      content: "This issue cannot be reopened.",
      toolCalls: [],
    } as never);
    const executeTool: ToolExecutor = vi.fn();
    const onText = vi.fn();

    const outcome = await runAgentLoop(
      [
        { role: "user", content: "Reopen the issue" },
        {
          role: "assistant",
          toolCalls: [
            {
              id: "api-1",
              name: "call_api",
              arguments: JSON.stringify({
                operationId: "IssueService/BatchUpdateIssuesStatus",
              }),
            },
          ],
        },
        {
          role: "tool",
          toolCallId: "api-1",
          content: JSON.stringify({ status: 400, error: apiError }),
        },
      ],
      [{ name: "done", description: "done", parametersSchema: {} }],
      executeTool,
      { onText }
    );

    expect(outcome).toEqual({
      kind: "completed",
      text: apiError,
      success: false,
      totalTokensUsed: 0,
    });
    expect(onText).toHaveBeenCalledExactlyOnceWith(apiError);
    expect(executeTool).not.toHaveBeenCalled();
  });

  test("does not treat a rejected API call as a successful plain-text completion", async () => {
    const apiError = "cannot change status from DONE to OPEN for issue 101";
    vi.mocked(aiServiceClientConnect.chat).mockResolvedValue({
      content: "The issue was reopened.",
      toolCalls: [],
    } as never);

    const outcome = await runAgentLoop(
      [
        { role: "user", content: "Reopen the issue" },
        {
          role: "assistant",
          toolCalls: [{ id: "api-1", name: "call_api", arguments: "{}" }],
        },
        {
          role: "tool",
          toolCallId: "api-1",
          content: JSON.stringify({ status: 400, error: apiError }),
        },
      ],
      [{ name: "done", description: "done", parametersSchema: {} }],
      vi.fn(),
      undefined,
      undefined,
      { allowPlainTextCompletion: true }
    );

    expect(outcome).toEqual({
      kind: "completed",
      text: apiError,
      success: false,
      totalTokensUsed: 0,
    });
    expect(aiServiceClientConnect.chat).toHaveBeenCalledTimes(1);
  });

  test("requires done after a failed call followed by a successful call", async () => {
    vi.mocked(aiServiceClientConnect.chat).mockResolvedValue({
      content: "Recovered answer without done",
      toolCalls: [],
    } as never);

    const outcome = await runAgentLoop(
      [
        { role: "user", content: "Update the issue" },
        {
          role: "assistant",
          toolCalls: [
            {
              id: "api-1",
              name: "call_api",
              arguments: JSON.stringify({
                operationId: "IssueService/BatchUpdateIssuesStatus",
                body: { issues: ["projects/a/issues/101"], status: "OPEN" },
              }),
            },
            {
              id: "api-2",
              name: "call_api",
              arguments: JSON.stringify({
                body: {
                  parent: "projects/a",
                  status: "OPEN",
                  issues: ["projects/a/issues/101"],
                },
                operationId: "IssueService/BatchUpdateIssuesStatus",
              }),
            },
          ],
        },
        {
          role: "tool",
          toolCallId: "api-1",
          content: JSON.stringify({ status: 400, error: "Initial failure" }),
        },
        {
          role: "tool",
          toolCallId: "api-2",
          content: JSON.stringify({ status: 200, response: {} }),
        },
      ],
      [{ name: "done", description: "done", parametersSchema: {} }],
      vi.fn()
    );

    expect(outcome.kind).toBe("error");
    expect(aiServiceClientConnect.chat).toHaveBeenCalledTimes(2);
  });

  test("does not substitute an earlier API error for a missing done", async () => {
    const apiError = "cannot change status from DONE to OPEN for issue 101";
    vi.mocked(aiServiceClientConnect.chat).mockResolvedValue({
      content: "Both issues were reopened.",
      toolCalls: [],
    } as never);

    const outcome = await runAgentLoop(
      [
        { role: "user", content: "Reopen two issues" },
        {
          role: "assistant",
          toolCalls: [
            {
              id: "api-1",
              name: "call_api",
              arguments: JSON.stringify({
                operationId: "IssueService/BatchUpdateIssuesStatus",
                body: { issues: ["projects/a/issues/101"], status: "OPEN" },
              }),
            },
            {
              id: "api-2",
              name: "call_api",
              arguments: JSON.stringify({
                operationId: "IssueService/BatchUpdateIssuesStatus",
                body: { issues: ["projects/a/issues/102"], status: "OPEN" },
              }),
            },
          ],
        },
        {
          role: "tool",
          toolCallId: "api-1",
          content: JSON.stringify({ status: 400, error: apiError }),
        },
        {
          role: "tool",
          toolCallId: "api-2",
          content: JSON.stringify({ status: 200, response: {} }),
        },
      ],
      [{ name: "done", description: "done", parametersSchema: {} }],
      vi.fn()
    );

    expect(outcome.kind).toBe("error");
  });

  test("requires done after a failed write followed by another successful write", async () => {
    const first = {
      operationId: "IssueService/BatchUpdateIssuesStatus",
      body: { issues: ["projects/a/issues/101"], status: "OPEN" },
    };
    const second = {
      operationId: "IssueService/BatchUpdateIssuesStatus",
      body: { issues: ["projects/a/issues/101"], status: "DONE" },
    };
    const apiError = "First request failed";
    vi.mocked(aiServiceClientConnect.chat).mockResolvedValue({
      content: "Both requests succeeded.",
      toolCalls: [],
    } as never);

    const outcome = await runAgentLoop(
      [
        { role: "user", content: "Run both requests" },
        {
          role: "assistant",
          toolCalls: [
            {
              id: "api-1",
              name: "call_api",
              arguments: JSON.stringify(first),
            },
            {
              id: "api-2",
              name: "call_api",
              arguments: JSON.stringify(second),
            },
          ],
        },
        {
          role: "tool",
          toolCallId: "api-1",
          content: JSON.stringify({ status: 400, error: apiError }),
        },
        {
          role: "tool",
          toolCallId: "api-2",
          content: JSON.stringify({ status: 200, response: {} }),
        },
      ],
      [{ name: "done", description: "done", parametersSchema: {} }],
      vi.fn()
    );

    expect(outcome.kind).toBe("error");
  });

  test.each([
    {
      operationId: "SQLService/Query",
      failure: { status: 400, error: "Unknown column missing" },
      correctedStatement: "SELECT id FROM t WHERE id = 1",
    },
    {
      operationId: "bytebase.v1.SQLService.Query",
      failure: {
        status: 200,
        response: { results: [{ error: "Unknown column missing" }] },
      },
      correctedStatement: "SELECT t.id FROM t JOIN u ON t.id = u.t_id",
    },
  ])(
    "preserves a failed $operationId attempt after a successful retry",
    async ({ operationId, failure, correctedStatement }) => {
      vi.mocked(aiServiceClientConnect.chat).mockResolvedValue({
        content: "",
        toolCalls: [
          {
            id: "done-1",
            name: "done",
            arguments: JSON.stringify({
              text: "The corrected query returned one row.",
              success: true,
            }),
            metadata: "",
          },
        ],
      } as never);

      const onToolResult = vi.fn();
      const outcome = await runAgentLoop(
        [
          {
            role: "user",
            content: "Query the database and correct any SQL error",
          },
          {
            role: "assistant",
            toolCalls: [
              {
                id: "api-1",
                name: "call_api",
                arguments: JSON.stringify({
                  operationId,
                  body: {
                    name: "instances/i/databases/db",
                    statement: "SELECT missing FROM t",
                  },
                }),
              },
              {
                id: "api-2",
                name: "call_api",
                arguments: JSON.stringify({
                  operationId,
                  body: {
                    name: "instances/i/databases/db",
                    statement: correctedStatement,
                  },
                }),
              },
            ],
          },
          {
            role: "tool",
            toolCallId: "api-1",
            content: JSON.stringify(failure),
          },
          {
            role: "tool",
            toolCallId: "api-2",
            content: JSON.stringify({
              status: 200,
              response: { results: [{}] },
            }),
          },
        ],
        [{ name: "done", description: "done", parametersSchema: {} }],
        vi.fn(async () => ({
          kind: "done" as const,
          text: "The corrected query returned one row.",
          success: true,
        })),
        { onToolResult }
      );

      expect(outcome).toEqual({
        kind: "completed",
        text: "The corrected query returned one row.",
        success: true,
        totalTokensUsed: 0,
      });
      expect(onToolResult).toHaveBeenCalledWith(
        "done-1",
        JSON.stringify({
          text: "The corrected query returned one row.",
          success: true,
          failedApiCallCount: 1,
        })
      );
    }
  );

  test("requires done after a failed query followed by a successful query on another database", async () => {
    const apiError = "Unknown column missing";
    vi.mocked(aiServiceClientConnect.chat).mockResolvedValue({
      content: "The query returned rows.",
      toolCalls: [],
    } as never);

    const outcome = await runAgentLoop(
      [
        { role: "user", content: "Query both databases" },
        {
          role: "assistant",
          toolCalls: [
            {
              id: "api-1",
              name: "call_api",
              arguments: JSON.stringify({
                operationId: "SQLService/Query",
                body: {
                  name: "instances/i/databases/db-one",
                  statement: "SELECT missing FROM t",
                },
              }),
            },
            {
              id: "api-2",
              name: "call_api",
              arguments: JSON.stringify({
                operationId: "SQLService/Query",
                body: {
                  name: "instances/i/databases/db-two",
                  statement: "SELECT id FROM t",
                },
              }),
            },
          ],
        },
        {
          role: "tool",
          toolCallId: "api-1",
          content: JSON.stringify({ status: 400, error: apiError }),
        },
        {
          role: "tool",
          toolCallId: "api-2",
          content: JSON.stringify({ status: 200, response: { results: [{}] } }),
        },
      ],
      [{ name: "done", description: "done", parametersSchema: {} }],
      vi.fn()
    );

    expect(outcome.kind).toBe("error");
  });

  test("requires done after a failed write followed by a successful query", async () => {
    const apiError = "Update denied";
    vi.mocked(aiServiceClientConnect.chat).mockResolvedValue({
      content: "The database was updated.",
      toolCalls: [],
    } as never);

    const outcome = await runAgentLoop(
      [
        { role: "user", content: "Update and verify the database" },
        {
          role: "assistant",
          toolCalls: [
            {
              id: "api-1",
              name: "call_api",
              arguments: JSON.stringify({
                operationId: "DatabaseService/UpdateDatabase",
                body: {
                  database: { name: "instances/i/databases/db" },
                },
              }),
            },
            {
              id: "api-2",
              name: "call_api",
              arguments: JSON.stringify({
                operationId: "SQLService/Query",
                body: {
                  name: "instances/i/databases/db",
                  statement: "SELECT id FROM t",
                },
              }),
            },
          ],
        },
        {
          role: "tool",
          toolCallId: "api-1",
          content: JSON.stringify({ status: 403, error: apiError }),
        },
        {
          role: "tool",
          toolCallId: "api-2",
          content: JSON.stringify({ status: 200, response: { results: [{}] } }),
        },
      ],
      [{ name: "done", description: "done", parametersSchema: {} }],
      vi.fn()
    );

    expect(outcome.kind).toBe("error");
  });

  test.each([
    "UPDATE t SET id = 2",
    "SELECT id INTO backup FROM t",
    "SELECT id FROM t FOR UPDATE",
    "SELECT mutate_data()",
    "SELECT id FROM t; DELETE FROM t",
    "EXPLAIN ANALYZE UPDATE t SET id = 2",
  ])(
    "does not infer task completion from SQL syntax: %s",
    async (statement) => {
      const apiError = "Statement failed";
      vi.mocked(aiServiceClientConnect.chat).mockResolvedValue({
        content: "The statements succeeded.",
        toolCalls: [],
      } as never);

      const outcome = await runAgentLoop(
        [
          { role: "user", content: "Run the statements" },
          {
            role: "assistant",
            toolCalls: [
              {
                id: "api-1",
                name: "call_api",
                arguments: JSON.stringify({
                  operationId: "SQLService/Query",
                  body: { name: "instances/i/databases/db", statement },
                }),
              },
              {
                id: "api-2",
                name: "call_api",
                arguments: JSON.stringify({
                  operationId: "SQLService/Query",
                  body: {
                    name: "instances/i/databases/db",
                    statement: "SELECT id FROM t",
                  },
                }),
              },
            ],
          },
          {
            role: "tool",
            toolCallId: "api-1",
            content: JSON.stringify({ status: 400, error: apiError }),
          },
          {
            role: "tool",
            toolCallId: "api-2",
            content: JSON.stringify({
              status: 200,
              response: { results: [{}] },
            }),
          },
        ],
        [{ name: "done", description: "done", parametersSchema: {} }],
        vi.fn()
      );

      expect(outcome.kind).toBe("error");
    }
  );

  test("requires done after a rejected write followed by a successful read", async () => {
    const apiError = "cannot change status from DONE to OPEN for issue 101";
    vi.mocked(aiServiceClientConnect.chat).mockResolvedValue({
      content: "The issue was reopened.",
      toolCalls: [],
    } as never);

    const outcome = await runAgentLoop(
      [
        { role: "user", content: "Reopen the issue" },
        {
          role: "assistant",
          toolCalls: [
            {
              id: "api-1",
              name: "call_api",
              arguments: JSON.stringify({
                operationId: "IssueService/BatchUpdateIssuesStatus",
                body: { issues: ["projects/a/issues/101"], status: "OPEN" },
              }),
            },
            {
              id: "api-2",
              name: "call_api",
              arguments: JSON.stringify({
                operationId: "IssueService/GetIssue",
                body: { name: "projects/a/issues/101" },
              }),
            },
          ],
        },
        {
          role: "tool",
          toolCallId: "api-1",
          content: JSON.stringify({ status: 400, error: apiError }),
        },
        {
          role: "tool",
          toolCallId: "api-2",
          content: JSON.stringify({
            status: 200,
            response: { status: "DONE" },
          }),
        },
      ],
      [{ name: "done", description: "done", parametersSchema: {} }],
      vi.fn()
    );

    expect(outcome.kind).toBe("error");
  });

  test("counts failed API attempts when the agent completes successfully", async () => {
    const apiError = "cannot change status from DONE to OPEN for issue 101";
    vi.mocked(aiServiceClientConnect.chat).mockResolvedValue({
      content: "",
      toolCalls: [
        {
          id: "done-1",
          name: "done",
          arguments: JSON.stringify({
            text: "The issue was reopened.",
            success: true,
          }),
          metadata: "",
        },
      ],
    } as never);
    const onToolResult = vi.fn();

    const outcome = await runAgentLoop(
      [
        { role: "user", content: "Reopen the issue" },
        {
          role: "assistant",
          toolCalls: [
            {
              id: "api-1",
              name: "call_api",
              arguments: JSON.stringify({
                operationId: "IssueService/BatchUpdateIssuesStatus",
                body: { issues: ["projects/a/issues/101"], status: "OPEN" },
              }),
            },
            {
              id: "api-2",
              name: "call_api",
              arguments: JSON.stringify({
                operationId: "IssueService/GetIssue",
                body: { name: "projects/a/issues/101" },
              }),
            },
            {
              id: "api-3",
              name: "call_api",
              arguments: JSON.stringify({
                operationId: "IssueService/BatchUpdateIssuesStatus",
                body: { issues: ["projects/a/issues/101"], status: "OPEN" },
              }),
            },
          ],
        },
        {
          role: "tool",
          toolCallId: "api-1",
          content: JSON.stringify({ status: 400, error: apiError }),
        },
        {
          role: "tool",
          toolCallId: "api-2",
          content: JSON.stringify({ status: 403, error: "Read denied" }),
        },
        {
          role: "tool",
          toolCallId: "api-3",
          content: JSON.stringify({ status: 200, response: {} }),
        },
      ],
      [{ name: "done", description: "done", parametersSchema: {} }],
      vi.fn(async () => ({
        kind: "done" as const,
        text: "The issue was reopened.",
        success: true,
      })),
      { onToolResult }
    );

    expect(outcome).toEqual({
      kind: "completed",
      text: "The issue was reopened.",
      success: true,
      totalTokensUsed: 0,
    });
    expect(onToolResult).toHaveBeenCalledWith(
      "done-1",
      JSON.stringify({
        text: "The issue was reopened.",
        success: true,
        failedApiCallCount: 2,
      })
    );
  });
  test.each(["SQLService/Query", "bytebase.v1.SQLService.Query"])(
    "reports a query result error for %s inside a successful API response",
    async (operationId) => {
      const queryError = "permission denied for relation records";
      vi.mocked(aiServiceClientConnect.chat).mockResolvedValue({
        content: "The query returned rows.",
        toolCalls: [],
      } as never);

      const outcome = await runAgentLoop(
        [
          { role: "user", content: "Query the database" },
          {
            role: "assistant",
            toolCalls: [
              {
                id: "api-1",
                name: "call_api",
                arguments: JSON.stringify({ operationId }),
              },
            ],
          },
          {
            role: "tool",
            toolCallId: "api-1",
            content: JSON.stringify({
              status: 200,
              response: { results: [{ error: queryError }] },
            }),
          },
        ],
        [{ name: "done", description: "done", parametersSchema: {} }],
        vi.fn()
      );

      expect(outcome).toEqual({
        kind: "completed",
        text: queryError,
        success: false,
        totalTokensUsed: 0,
      });
    }
  );

  test("does not reuse an API rejection from an earlier user turn", async () => {
    vi.mocked(aiServiceClientConnect.chat).mockResolvedValue({
      content: "Answer without done",
      toolCalls: [],
    } as never);

    const outcome = await runAgentLoop(
      [
        { role: "user", content: "Reopen the issue" },
        {
          role: "assistant",
          toolCalls: [
            {
              id: "api-1",
              name: "call_api",
              arguments: "{}",
            },
          ],
        },
        {
          role: "tool",
          toolCallId: "api-1",
          content: JSON.stringify({ status: 400, error: "Not allowed" }),
        },
        { role: "user", content: "What is next?" },
      ],
      [{ name: "done", description: "done", parametersSchema: {} }],
      vi.fn()
    );

    expect(outcome.kind).toBe("error");
    expect(aiServiceClientConnect.chat).toHaveBeenCalledTimes(2);
  });

  test("does not execute a non-terminal tool requested during finalization", async () => {
    vi.mocked(aiServiceClientConnect.chat)
      .mockResolvedValueOnce({
        content: "The project has 26 active databases.",
        toolCalls: [],
      } as never)
      .mockResolvedValueOnce({
        content: "",
        toolCalls: [
          {
            id: "api-1",
            name: "call_api",
            arguments: JSON.stringify({ operationId: "DeleteDatabase" }),
            metadata: "",
          },
        ],
      } as never);
    const executeTool: ToolExecutor = vi.fn();

    const outcome = await runAgentLoop(
      [{ role: "system", content: "system" }],
      ["call_api", "done"].map((name) => ({
        name,
        description: name,
        parametersSchema: {},
      })),
      executeTool
    );

    expect(outcome.kind).toBe("error");
    expect(executeTool).not.toHaveBeenCalled();
  });

  test.each([
    ["ask_masking_exemption_scope", ""],
    ["ask_masking_exemption_scope", "I need more masking configuration."],
    ["ask_masking_rule_scope", ""],
    ["ask_masking_rule_scope", "I need more masking configuration."],
    ["ask_masking_semantic_type", ""],
    ["ask_masking_semantic_type", "I need more masking configuration."],
  ])("can call %s while finalizing after %j", async (questionTool, content) => {
    vi.mocked(aiServiceClientConnect.chat)
      .mockResolvedValueOnce({ content, toolCalls: [] } as never)
      .mockResolvedValueOnce({
        content: "",
        toolCalls: [
          {
            id: "scope",
            name: questionTool,
            arguments: "{}",
            metadata: "",
          },
        ],
      } as never);
    const ask = {
      toolCallId: "scope",
      prompt: "Which masking configuration should I use?",
      kind: "choose" as const,
      options: [
        { label: "All databases in this project", value: "projects/orders" },
        { label: "Another database", value: "another-database" },
      ],
    };
    const outcome = await runAgentLoop(
      [{ role: "user", content: "Help me configure masking." }],
      [
        "call_api",
        "ask_user",
        "ask_masking_exemption_scope",
        "ask_masking_rule_scope",
        "ask_masking_semantic_type",
        "done",
      ].map((name) => ({ name, description: name, parametersSchema: {} })),
      async () => ({ kind: "ask_user", ask })
    );

    expect(outcome).toEqual({ kind: "awaiting_user", ask, totalTokensUsed: 0 });
    expect(
      vi
        .mocked(aiServiceClientConnect.chat)
        .mock.calls[1]?.[0]?.toolDefinitions?.map((tool) => tool.name)
    ).toEqual([
      "ask_user",
      "ask_masking_exemption_scope",
      "ask_masking_rule_scope",
      "ask_masking_semantic_type",
      "done",
    ]);
    expect(
      vi
        .mocked(aiServiceClientConnect.chat)
        .mock.calls[1]?.[0]?.messages?.at(-1)?.content
    ).toContain(questionTool);
  });

  test("completes a read-only starter when the model answers in plain text after a tool result", async () => {
    vi.mocked(aiServiceClientConnect.chat)
      .mockResolvedValueOnce({
        content: "",
        toolCalls: [
          {
            id: "tool-open-issues",
            name: "get_project_open_issues",
            arguments: "{}",
            metadata: "",
          },
        ],
      } as never)
      .mockResolvedValueOnce({
        content: "You have two open Issues in this Project.",
        toolCalls: [],
      } as never);

    const executeTool: ToolExecutor = vi.fn(async () => ({
      kind: "tool_result" as const,
      result: JSON.stringify({ issues: [] }),
    }));
    const onText = vi.fn();

    const outcome = await runAgentLoop(
      [{ role: "system", content: "system" }],
      [],
      executeTool,
      { onText },
      undefined,
      { allowPlainTextCompletion: true }
    );

    expect(outcome).toEqual({
      kind: "completed",
      text: "You have two open Issues in this Project.",
      success: true,
      totalTokensUsed: 0,
    });
    expect(onText).toHaveBeenCalledWith(
      "You have two open Issues in this Project."
    );
  });

  test("returns awaiting_user without synthesizing an immediate tool result", async () => {
    vi.mocked(aiServiceClientConnect.chat).mockResolvedValue({
      content: "",
      toolCalls: [
        {
          id: "tool-ask",
          name: "ask_user",
          arguments: JSON.stringify({
            prompt: "Which environment should I use?",
            kind: "choose",
            options: [
              { label: "Production", value: "prod" },
              { label: "Staging", value: "staging" },
            ],
          }),
          metadata: "",
        },
      ],
    } as never);

    const executeTool: ToolExecutor = vi.fn(
      async (_name, _args, toolCallId) => ({
        kind: "ask_user" as const,
        ask: {
          toolCallId,
          prompt: "Which environment should I use?",
          kind: "choose" as const,
          options: [
            { label: "Production", value: "prod" },
            { label: "Staging", value: "staging" },
          ],
        },
      })
    );
    const onToolResult = vi.fn();

    const outcome = await runAgentLoop(
      [{ role: "system", content: "system" }],
      [],
      executeTool,
      { onToolResult }
    );

    expect(outcome).toEqual({
      kind: "awaiting_user",
      ask: {
        toolCallId: "tool-ask",
        prompt: "Which environment should I use?",
        kind: "choose",
        options: [
          { label: "Production", value: "prod" },
          { label: "Staging", value: "staging" },
        ],
      },
      totalTokensUsed: 0,
    });
    expect(onToolResult).not.toHaveBeenCalled();
  });

  test.each(["ask_user", "ask_masking_exemption_scope"])(
    "records skipped tool results after %s and does not execute later tool calls",
    async (questionTool) => {
      vi.mocked(aiServiceClientConnect.chat).mockResolvedValue({
        content: "",
        toolCalls: [
          {
            id: "tool-ask",
            name: questionTool,
            arguments: JSON.stringify(
              questionTool === "ask_user"
                ? {
                    prompt: "Which environment should I use?",
                    kind: "input",
                  }
                : {}
            ),
            metadata: "",
          },
          {
            id: "tool-late",
            name: "search_api",
            arguments: JSON.stringify({ service: "SQLService" }),
            metadata: "",
          },
        ],
      } as never);

      const executeTool: ToolExecutor = vi.fn(
        async (name, _args, toolCallId) => {
          if (name === questionTool) {
            return {
              kind: "ask_user" as const,
              ask: {
                toolCallId,
                prompt: "Which environment should I use?",
                kind: "input" as const,
              },
            };
          }
          return {
            kind: "tool_result" as const,
            result: "should not run",
          };
        }
      );
      const onToolResult = vi.fn();

      const outcome = await runAgentLoop(
        [{ role: "system", content: "system" }],
        [],
        executeTool,
        { onToolResult }
      );

      expect(outcome).toEqual({
        kind: "awaiting_user",
        ask: {
          toolCallId: "tool-ask",
          prompt: "Which environment should I use?",
          kind: "input",
        },
        totalTokensUsed: 0,
      });
      expect(executeTool).toHaveBeenCalledTimes(1);
      expect(executeTool).toHaveBeenCalledWith(
        questionTool,
        questionTool === "ask_user"
          ? {
              prompt: "Which environment should I use?",
              kind: "input",
            }
          : {},
        "tool-ask"
      );
      expect(onToolResult).toHaveBeenCalledTimes(1);
      expect(onToolResult).toHaveBeenCalledWith(
        "tool-late",
        JSON.stringify({
          skipped: true,
          blockedByToolCallId: "tool-ask",
          blockedByKind: "ask_user",
          reason:
            "Skipped because this assistant turn already emitted ask_user",
        })
      );
    }
  );

  test("records skipped tool results after done and does not execute later tool calls", async () => {
    vi.mocked(aiServiceClientConnect.chat).mockResolvedValue({
      content: "",
      toolCalls: [
        {
          id: "tool-done",
          name: "done",
          arguments: JSON.stringify({
            text: "Finished successfully",
            success: true,
          }),
          metadata: "",
        },
        {
          id: "tool-late",
          name: "navigate",
          arguments: JSON.stringify({ path: "/projects/demo" }),
          metadata: "",
        },
      ],
    } as never);

    const executeTool: ToolExecutor = vi.fn(async (name, args) => {
      if (name === "done") {
        return {
          kind: "done" as const,
          text: String(args.text),
          success: args.success !== false,
        };
      }
      return {
        kind: "tool_result" as const,
        result: "should not run",
      };
    });
    const onToolResult = vi.fn();
    const onText = vi.fn();

    const outcome = await runAgentLoop(
      [{ role: "system", content: "system" }],
      [],
      executeTool,
      { onToolResult, onText }
    );

    expect(outcome).toEqual({
      kind: "completed",
      text: "Finished successfully",
      success: true,
      totalTokensUsed: 0,
    });
    expect(executeTool).toHaveBeenCalledTimes(1);
    expect(executeTool).toHaveBeenCalledWith(
      "done",
      {
        text: "Finished successfully",
        success: true,
      },
      "tool-done"
    );
    expect(onToolResult).toHaveBeenNthCalledWith(
      1,
      "tool-done",
      JSON.stringify({
        text: "Finished successfully",
        success: true,
      })
    );
    expect(onToolResult).toHaveBeenNthCalledWith(
      2,
      "tool-late",
      JSON.stringify({
        skipped: true,
        blockedByToolCallId: "tool-done",
        blockedByKind: "done",
        reason: "Skipped because this assistant turn already emitted done",
      })
    );
    expect(onText).toHaveBeenCalledWith("Finished successfully");
  });
});
