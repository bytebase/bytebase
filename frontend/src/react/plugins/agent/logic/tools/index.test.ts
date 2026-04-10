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

describe("agent tools get_page_state", () => {
  test("describes a ref-labeled DOM tree", () => {
    const pageState = getToolDefinitions().find(
      (tool) => tool.name === "get_page_state"
    );

    expect(pageState).toBeDefined();
    expect(pageState?.description).toContain("ref-labeled DOM tree");
    expect(pageState?.description).toContain("element refs like [e1]");
    expect(pageState?.description).not.toContain("element indices");
    expect(pageState?.parametersSchema).toEqual(
      expect.objectContaining({
        properties: expect.objectContaining({
          mode: expect.objectContaining({
            description: expect.stringContaining("ref-labeled tree"),
          }),
        }),
      })
    );
  });
});

describe("agent tools dom_action", () => {
  test("exposes ref-based dom_action schema", () => {
    const domAction = getToolDefinitions().find(
      (tool) => tool.name === "dom_action"
    );

    expect(domAction).toBeDefined();
    expect(domAction?.description).toContain(
      '**Always call get_page_state(mode="dom") first** to get the element ref, such as [e1].'
    );
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

describe("agent tools search_api", () => {
  test("renders byte fields as bytes with the UTF-8 call_api note", async () => {
    const { searchApi: actualSearchApi } =
      await vi.importActual<typeof import("./searchApi")>("./searchApi");

    const result = await actualSearchApi({
      operationId: "SheetService/CreateSheet",
    });

    const requestBodyStart = result.indexOf("### Request Body");
    const responseBodyStart = result.indexOf("### Response Body");
    const requestBody =
      requestBodyStart >= 0
        ? result.slice(
            requestBodyStart,
            responseBodyStart >= 0 ? responseBodyStart : undefined
          )
        : "";
    const responseBody =
      responseBodyStart >= 0 ? result.slice(responseBodyStart) : "";
    const requestNote =
      "request body includes protobuf bytes fields; plain strings passed to call_api are UTF-8 encoded automatically.";
    const requestNoteCount = requestBody.split(requestNote).length - 1;

    expect(requestBody).toContain('"sheet": bytebase.v1.Sheet');
    expect(requestBody).toContain(requestNote);
    expect(requestNoteCount).toBe(1);
    expect(responseBody).toContain('"content": bytes');
    expect(responseBody).not.toContain(requestNote);
  });

  test("detects transitive byte fields through external oneof request schemas", async () => {
    const { searchApi: actualSearchApi } =
      await vi.importActual<typeof import("./searchApi")>("./searchApi");

    const result = await actualSearchApi({
      operationId: "CelService/BatchDeparse",
    });

    expect(result).toContain(
      '"expressions": array<google.api.expr.v1alpha1.Expr>'
    );
    expect(result).toContain(
      "Note: request body includes protobuf bytes fields; plain strings passed to call_api are UTF-8 encoded automatically."
    );
  });
});

describe("agent tools concurrency guard", () => {
  test("blocks concurrent page-changing tools across threads", async () => {
    let releaseNavigation: (() => void) | undefined;
    vi.mocked(createNavigateTool).mockImplementation(() => {
      return vi.fn(
        () =>
          new Promise<string>((resolve) => {
            releaseNavigation = () =>
              resolve(
                JSON.stringify({
                  navigated: true,
                  currentPath: "/projects/demo",
                })
              );
          })
      );
    });

    const firstExecutor = createToolExecutor({} as Router, {
      chatId: "thread-1",
    });
    const secondExecutor = createToolExecutor({} as Router, {
      chatId: "thread-2",
    });

    const firstRun = firstExecutor(
      "navigate",
      { path: "/projects/demo" },
      "tool-navigate-1"
    );

    await expect(
      secondExecutor("navigate", { path: "/projects/other" }, "tool-navigate-2")
    ).rejects.toThrow("Another chat is already using a page-changing tool");

    releaseNavigation?.();
    await expect(firstRun).resolves.toEqual({
      kind: "tool_result",
      result: JSON.stringify({
        navigated: true,
        currentPath: "/projects/demo",
      }),
    });
  });
});
