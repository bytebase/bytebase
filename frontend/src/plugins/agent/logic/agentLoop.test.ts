import { beforeEach, describe, expect, test, vi } from "vitest";
import { aiServiceClientConnect } from "@/connect";
import { AIChatMessageRole } from "@/types/proto-es/v1/ai_service_pb";
import { runAgentLoop } from "./agentLoop";
import type { ToolExecutor } from "./types";

vi.mock("@/connect", () => ({
  aiServiceClientConnect: {
    chat: vi.fn(),
  },
}));

describe("runAgentLoop", () => {
  beforeEach(() => {
    vi.mocked(aiServiceClientConnect.chat).mockReset();
  });

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
      explicit: true,
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
  test("keeps plain assistant text as a compatibility fallback", async () => {
    vi.mocked(aiServiceClientConnect.chat).mockResolvedValue({
      content: "Here is the answer",
      toolCalls: [],
    } as never);

    const executeTool: ToolExecutor = vi.fn();
    const onText = vi.fn();

    const outcome = await runAgentLoop(
      [{ role: "system", content: "system" }],
      [],
      executeTool,
      { onText }
    );

    expect(outcome).toEqual({
      kind: "completed",
      text: "Here is the answer",
      success: true,
      explicit: false,
    });
    expect(onText).toHaveBeenCalledWith("Here is the answer");
    expect(executeTool).not.toHaveBeenCalled();
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
      explicit: true,
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
