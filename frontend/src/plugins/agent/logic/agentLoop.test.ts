import { beforeEach, describe, expect, test, vi } from "vitest";
import { aiServiceClientConnect } from "@/connect";
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

    const outcome = await runAgentLoop(
      [{ role: "system", content: "system" }],
      [],
      executeTool,
      { onText, onAssistantMessage }
    );

    expect(outcome).toEqual({
      kind: "completed",
      text: "Finished successfully",
      success: true,
      explicit: true,
    });
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
});
