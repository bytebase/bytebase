import type { Router } from "vue-router";
import { getSkill, type GetSkillArgs } from "../skills";
import type {
  AgentAskUserKind,
  AgentPendingAsk,
  ToolDefinition,
  ToolExecutionResult,
  ToolExecutor,
} from "../types";
import { callApi, type CallApiArgs } from "./callApi";
import { createDomActionTool, type DomActionArgs } from "./domAction";
import { createNavigateTool } from "./navigate";
import { createPageStateTool, type PageStateArgs } from "./pageState";
import { searchApi, type SearchApiArgs } from "./searchApi";

const toToolResult = (result: string): ToolExecutionResult => ({
  kind: "tool_result",
  result,
});

const parseAskUserKind = (value: unknown): AgentAskUserKind => {
  return value === "confirm" ? "confirm" : "input";
};

const parseDoneArgs = (args: Record<string, unknown>): ToolExecutionResult => {
  if (typeof args.text !== "string" || !args.text.trim()) {
    throw new Error("done requires a non-empty text string");
  }
  return {
    kind: "done",
    text: args.text,
    success: args.success !== false,
  };
};

const parseAskUserArgs = (
  args: Record<string, unknown>,
  toolCallId: string
): ToolExecutionResult => {
  if (typeof args.prompt !== "string" || !args.prompt.trim()) {
    throw new Error("ask_user requires a non-empty prompt string");
  }

  const ask: AgentPendingAsk = {
    toolCallId,
    prompt: args.prompt,
    kind: parseAskUserKind(args.kind),
  };

  if (typeof args.defaultValue === "string") {
    ask.defaultValue = args.defaultValue;
  }
  if (typeof args.confirmLabel === "string") {
    ask.confirmLabel = args.confirmLabel;
  }
  if (typeof args.cancelLabel === "string") {
    ask.cancelLabel = args.cancelLabel;
  }

  return {
    kind: "ask_user",
    ask,
  };
};

export function getToolDefinitions(): ToolDefinition[] {
  return [
    {
      name: "search_api",
      description: `Browse API endpoints and get request/response schemas. Use the API Directory in the system prompt to identify the right service first.

| Mode | Parameters | Result |
|------|------------|--------|
| List | (none) | All services |
| Browse | service="SQLService" | All endpoints in a service |
| Details | operationId="SQLService/Query" | Request/response schema |
| Schema | schema="Instance" | Message type definition |

**Typical workflow:** Read API Directory → search_api(service="...") → search_api(operationId="...") → call_api(...)`,
      parametersSchema: {
        type: "object",
        properties: {
          operationId: {
            type: "string",
            description:
              'Get detailed schema for a specific endpoint. Supports short format: "SQLService/Query"',
          },
          schema: {
            type: "string",
            description:
              'Get the definition of a message type. Examples: "Instance", "Engine", "bytebase.v1.Instance"',
          },
          service: {
            type: "string",
            description:
              'Filter to a specific service. Examples: "SQLService", "DatabaseService"',
          },
        },
      },
    },
    {
      name: "call_api",
      description: `Execute a Bytebase API endpoint. **Use search_api first to get operationId and schema.**

| Parameter | Required | Description |
|-----------|----------|-------------|
| operationId | Yes | e.g., "SQLService/Query" |
| body | No | JSON request body |

**Resource names:** projects/my-project, instances/prod/databases/main

**Example:**
call_api(operationId="SQLService/Query", body={"name": "instances/i/databases/db", "statement": "SELECT 1"})`,
      parametersSchema: {
        type: "object",
        properties: {
          operationId: {
            type: "string",
            description:
              'The operation ID from search_api results. Supports short format: "SQLService/Query"',
          },
          body: {
            type: "object",
            description:
              "JSON request body. Structure depends on the endpoint — use search_api with operationId to see the expected format.",
          },
        },
        required: ["operationId"],
      },
    },
    {
      name: "navigate",
      description: `Navigate to a page in Bytebase, or list all available routes.

| Mode | Parameters | Result |
|------|------------|--------|
| Navigate | path="/projects" | Navigates to the page |
| List | list=true | Returns all valid route patterns |

**Always call with list=true first if you are unsure about the exact path.** Do not guess paths — a wrong path causes a 404.`,
      parametersSchema: {
        type: "object",
        properties: {
          path: {
            type: "string",
            description:
              "URL path to navigate to. Supports param placeholders like /projects/:projectId.",
          },
          list: {
            type: "boolean",
            description:
              "Set to true to list all available route patterns instead of navigating.",
          },
        },
      },
    },
    {
      name: "get_page_state",
      description: `Get information about the current page.

| Mode | Result |
|------|--------|
| semantic (default) | Route path, params, title + context from Pinia stores (project, database, issue, user info when available) |
| dom | Above + indexed DOM tree of interactive elements |

Use mode="dom" before dom_action to get element indices. Use semantic mode (default) to understand the current page context.`,
      parametersSchema: {
        type: "object",
        properties: {
          mode: {
            type: "string",
            enum: ["semantic", "dom"],
            description:
              'Default "semantic" returns route info + store context. "dom" adds an indexed tree of interactive elements for use with dom_action.',
          },
        },
      },
    },
    {
      name: "dom_action",
      description: `Last-resort DOM interaction — use only when no API endpoint covers the action.
**Always call get_page_state(mode="dom") first** to get the element index.

| Action | When to use | value param |
|--------|-------------|-------------|
| click  | Buttons, links, tabs, checkboxes | not needed |
| input  | Text fields, textareas, code editors | required — the text to enter |
| select | Dropdowns | required — the option text to select |
| read   | Get full content of an element (editor, input) | not needed |
| scroll | Bring element into viewport | not needed |`,
      parametersSchema: {
        type: "object",
        properties: {
          type: {
            type: "string",
            enum: ["click", "input", "select", "read", "scroll"],
            description: "The action to perform",
          },
          index: {
            type: "number",
            description:
              "Element index from DOM tree (from get_page_state with mode='dom')",
          },
          value: {
            type: "string",
            description: "Value for input/select actions",
          },
        },
        required: ["type", "index"],
      },
    },
    {
      name: "get_skill",
      description: `Get step-by-step workflow guides for common Bytebase tasks. Load a skill before performing multi-step operations.

| Mode | Parameters | Result |
|------|------------|--------|
| List | (none) | All available skills |
| Load | name="query" | Step-by-step workflow guide |

**Available skills:** query, database-change, grant-permission

**Workflow:** get_skill() → get_skill(name="...") → follow the guide using search_api + call_api`,
      parametersSchema: {
        type: "object",
        properties: {
          name: {
            type: "string",
            description:
              'Skill name to load. Omit to list all skills. Examples: "query", "database-change", "grant-permission"',
          },
        },
      },
    },
    {
      name: "ask_user",
      description:
        "Ask the user for missing information or confirmation. Use kind='input' for free-form text and kind='confirm' for confirm/cancel decisions.",
      parametersSchema: {
        type: "object",
        properties: {
          prompt: {
            type: "string",
            description: "The question to show to the user.",
          },
          kind: {
            type: "string",
            enum: ["input", "confirm"],
            description: 'Defaults to "input" when omitted.',
          },
          defaultValue: {
            type: "string",
            description:
              "Optional default answer to prefill for input prompts.",
          },
          confirmLabel: {
            type: "string",
            description: "Optional confirm button label for confirm prompts.",
          },
          cancelLabel: {
            type: "string",
            description: "Optional cancel button label for confirm prompts.",
          },
        },
        required: ["prompt"],
      },
    },
    {
      name: "done",
      description:
        "Complete the current task explicitly. Provide the final user-visible text and whether the task succeeded.",
      parametersSchema: {
        type: "object",
        properties: {
          text: {
            type: "string",
            description: "Final assistant response shown to the user.",
          },
          success: {
            type: "boolean",
            description:
              "Set to false when the task could not be completed successfully.",
          },
        },
        required: ["text"],
      },
    },
  ];
}

export function createToolExecutor(router: Router): ToolExecutor {
  const navigateTool = createNavigateTool(router);
  const pageStateTool = createPageStateTool(router);
  const domActionTool = createDomActionTool(router);

  return async (
    name: string,
    args: Record<string, unknown>,
    toolCallId: string
  ): Promise<ToolExecutionResult> => {
    switch (name) {
      case "search_api":
        return toToolResult(await searchApi(args as SearchApiArgs));
      case "call_api":
        return toToolResult(await callApi(args as unknown as CallApiArgs));
      case "navigate":
        return toToolResult(
          await navigateTool(args as { path?: string; list?: boolean })
        );
      case "get_page_state":
        return toToolResult(await pageStateTool(args as PageStateArgs));
      case "dom_action":
        return toToolResult(
          await domActionTool(args as unknown as DomActionArgs)
        );
      case "get_skill":
        return toToolResult(await getSkill(args as GetSkillArgs));
      case "ask_user":
        return parseAskUserArgs(args, toolCallId);
      case "done":
        return parseDoneArgs(args);
      default:
        return toToolResult(JSON.stringify({ error: `Unknown tool: ${name}` }));
    }
  };
}
