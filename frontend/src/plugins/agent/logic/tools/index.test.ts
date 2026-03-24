import { describe, expect, test, vi } from "vitest";
import type { Router } from "vue-router";

vi.mock("../skills", () => ({
  getSkill: vi.fn(async () => "skill"),
}));
vi.mock("./callApi", () => ({
  callApi: vi.fn(async () => "call_api"),
}));
vi.mock("./domAction", () => ({
  createDomActionTool: vi.fn(() => vi.fn(async () => "dom_action")),
}));
vi.mock("./navigate", () => ({
  createNavigateTool: vi.fn(() => vi.fn(async () => "navigate")),
}));
vi.mock("./pageState", () => ({
  createPageStateTool: vi.fn(() => vi.fn(async () => "page_state")),
}));
vi.mock("./searchApi", () => ({
  searchApi: vi.fn(async () => "search_api"),
}));

import { createToolExecutor, getToolDefinitions } from ".";
import { createNavigateTool } from "./navigate";

describe("agent tools navigate", () => {
  test("calls onNavigate after a successful navigation result", async () => {
    const navigateTool = vi.fn(async () =>
      JSON.stringify({
        navigated: true,
        currentPath: "/projects/demo",
      })
    );
    vi.mocked(createNavigateTool).mockReturnValue(navigateTool);
    const onNavigate = vi.fn();
    const executeTool = createToolExecutor({} as Router, { onNavigate });

    const result = await executeTool(
      "navigate",
      {
        path: "/projects/demo",
      },
      "tool-navigate"
    );

    expect(result).toEqual({
      kind: "tool_result",
      result: JSON.stringify({
        navigated: true,
        currentPath: "/projects/demo",
      }),
    });
    expect(onNavigate).toHaveBeenCalledTimes(1);
  });
});

describe("agent tools dom_action", () => {
  test("exposes ref-based dom_action schema", () => {
    const domAction = getToolDefinitions().find(
      (tool) => tool.name === "dom_action"
    );

    expect(domAction).toBeDefined();
    expect(domAction?.parametersSchema).toEqual(
      expect.objectContaining({
        properties: expect.objectContaining({
          type: expect.objectContaining({
            enum: ["click", "input", "select", "read", "scroll"],
          }),
          ref: expect.objectContaining({
            type: "string",
          }),
        }),
        required: ["type", "ref"],
      })
    );
    expect(domAction?.parametersSchema).not.toEqual(
      expect.objectContaining({
        properties: expect.objectContaining({
          index: expect.anything(),
        }),
      })
    );
  });
});

describe("agent tools ask_user", () => {
  test("exposes choose in the ask_user schema", () => {
    const askUser = getToolDefinitions().find(
      (tool) => tool.name === "ask_user"
    );

    expect(askUser).toBeDefined();
    expect(askUser?.parametersSchema).toEqual(
      expect.objectContaining({
        properties: expect.objectContaining({
          kind: expect.objectContaining({
            enum: ["input", "confirm", "choose"],
          }),
          options: expect.any(Object),
        }),
      })
    );
  });

  test("parses choose prompts with sanitized options", async () => {
    const executeTool = createToolExecutor({} as Router);

    const result = await executeTool(
      "ask_user",
      {
        prompt: "Which environment should I use?",
        kind: "choose",
        options: [
          {
            label: " Production ",
            value: " prod ",
            description: " Primary ",
          },
          {
            value: "staging",
          },
          {
            label: "Broken",
          },
        ],
      },
      "tool-choose"
    );

    expect(result).toEqual({
      kind: "ask_user",
      ask: {
        toolCallId: "tool-choose",
        prompt: "Which environment should I use?",
        kind: "choose",
        options: [
          {
            label: "Production",
            value: "prod",
            description: "Primary",
          },
          {
            label: "staging",
            value: "staging",
          },
        ],
      },
    });
  });

  test("falls back to input when choose options are missing", async () => {
    const executeTool = createToolExecutor({} as Router);

    const result = await executeTool(
      "ask_user",
      {
        prompt: "Which environment should I use?",
        kind: "choose",
      },
      "tool-choose"
    );

    expect(result).toEqual({
      kind: "ask_user",
      ask: {
        toolCallId: "tool-choose",
        prompt: "Which environment should I use?",
        kind: "input",
      },
    });
  });
});
