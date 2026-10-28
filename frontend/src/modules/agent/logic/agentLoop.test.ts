// @vitest-environment node
import { beforeEach, describe, expect, test, vi } from "vitest";
import { aiServiceClientConnect } from "@/api";
import { AIChatMessageRole } from "@/types/proto-es/v1/ai_service_pb";
import { runAgentLoop } from "./agentLoop";
import type { ToolExecutionResult, ToolExecutor } from "./types";

vi.mock("@/api", () => ({
  aiServiceClientConnect: {
    chat: vi.fn(),
  },
}));

describe("runAgentLoop", () => {
  beforeEach(() => {
    vi.mocked(aiServiceClientConnect.chat).mockReset();
  });

  test("treats a non-AbortError response after user cancellation as aborted", async () => {
    let rejectChat!: (error: Error) => void;
    vi.mocked(aiServiceClientConnect.chat).mockImplementation(
      () =>
        new Promise((_, reject) => {
          rejectChat = reject;
        })
    );
    const controller = new AbortController();
    const run = runAgentLoop(
      [{ role: "user", content: "Check the project" }],
      [],
      vi.fn(),
      undefined,
      controller.signal
    );
    await vi.waitFor(() =>
      expect(aiServiceClientConnect.chat).toHaveBeenCalled()
    );
    controller.abort();
    rejectChat(new Error("Transport closed"));

    expect(await run).toEqual({ kind: "aborted", totalTokensUsed: 0 });
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
    const run = runAgentLoop(
      [{ role: "user", content: "Check the project" }],
      [{ name: "test_tool", description: "test", parametersSchema: {} }],
      executeTool,
      { onAssistantMessage },
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
    } as never);

    expect(await run).toEqual({ kind: "aborted", totalTokensUsed: 0 });
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

    const outcome = await runAgentLoop(
      [{ role: "system", content: "system" }],
      [],
      executeTool
    );

    expect(outcome).toEqual({
      kind: "completed",
      text: "Final answer",
      success: true,
      totalTokensUsed: 18,
    });
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
      "Agent must finish with done({ text, success }) or ask_user(...), not plain assistant text."
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

  test("clears a prior API rejection after a corrected retry for the same target", async () => {
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

  test("does not clear a rejected API request after the same operation succeeds for another target", async () => {
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

    expect(outcome).toEqual({
      kind: "completed",
      text: apiError,
      success: false,
      totalTokensUsed: 0,
    });
  });

  test.each([
    {
      label: "a different issue status",
      first: {
        operationId: "IssueService/BatchUpdateIssuesStatus",
        body: { issues: ["projects/a/issues/101"], status: "OPEN" },
      },
      second: {
        operationId: "IssueService/BatchUpdateIssuesStatus",
        body: { issues: ["projects/a/issues/101"], status: "DONE" },
      },
    },
    {
      label: "a different SQL statement",
      first: {
        operationId: "SQLService/Query",
        body: {
          name: "instances/i/databases/db",
          statement: "SELECT missing FROM t",
        },
      },
      second: {
        operationId: "SQLService/Query",
        body: {
          name: "instances/i/databases/db",
          statement: "SELECT id FROM t",
        },
      },
    },
  ])(
    "retains an API rejection after $label succeeds",
    async ({ first, second }) => {
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

      expect(outcome).toEqual({
        kind: "completed",
        text: apiError,
        success: false,
        totalTokensUsed: 0,
      });
    }
  );

  test("retains a rejected write after an unrelated successful API read", async () => {
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

    expect(outcome).toEqual({
      kind: "completed",
      text: apiError,
      success: false,
      totalTokensUsed: 0,
    });
  });

  test("does not accept a successful done after an unresolved API rejection", async () => {
    const apiError = "cannot change status from DONE to OPEN for issue 101";
    vi.mocked(aiServiceClientConnect.chat).mockResolvedValue({
      content: "",
      toolCalls: [
        {
          id: "done-1",
          name: "done",
          arguments: JSON.stringify({ text: "The issue was reopened." }),
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
          ],
        },
        {
          role: "tool",
          toolCallId: "api-1",
          content: JSON.stringify({ status: 400, error: apiError }),
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
      text: apiError,
      success: false,
      totalTokensUsed: 0,
    });
    expect(onToolResult).toHaveBeenCalledWith(
      "done-1",
      JSON.stringify({ text: apiError, success: false })
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

  test("records skipped tool results after ask_user and does not execute later tool calls", async () => {
    vi.mocked(aiServiceClientConnect.chat).mockResolvedValue({
      content: "",
      toolCalls: [
        {
          id: "tool-ask",
          name: "ask_user",
          arguments: JSON.stringify({
            prompt: "Which environment should I use?",
            kind: "input",
          }),
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

    const executeTool: ToolExecutor = vi.fn(async (name, _args, toolCallId) => {
      if (name === "ask_user") {
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
    });
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
      "ask_user",
      {
        prompt: "Which environment should I use?",
        kind: "input",
      },
      "tool-ask"
    );
    expect(onToolResult).toHaveBeenCalledTimes(1);
    expect(onToolResult).toHaveBeenCalledWith(
      "tool-late",
      JSON.stringify({
        skipped: true,
        blockedByToolCallId: "tool-ask",
        blockedByKind: "ask_user",
        reason: "Skipped because this assistant turn already emitted ask_user",
      })
    );
  });

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
