import type { AppRouterInstance } from "@/app/router";
import { getSkill, type GetSkillArgs } from "../skills";
import type {
  AgentAskUserKind,
  AgentAskUserOption,
  AgentPendingAsk,
  AgentRunMode,
  ToolDefinition,
  ToolExecutionResult,
  ToolExecutor,
} from "../types";
import { callApi, type CallApiArgs } from "./callApi";
import { createDomActionTool, type DomActionArgs } from "./domAction";
import { createNavigateTool } from "./navigate";
import { createPageStateTool, type PageStateArgs } from "./pageState";
import { createProjectOpenIssuesTool } from "./projectOpenIssues";
import { createProjectRecentActivityTool } from "./projectRecentActivity";
import { searchApi, type SearchApiArgs } from "./searchApi";
import { createWorkspaceRecentActivityTool } from "./workspaceRecentActivity";
import { createWorkspaceAttentionTool } from "./workspaceAttention";

const toToolResult = (result: string): ToolExecutionResult => ({
  kind: "tool_result",
  result,
});

const parseAskUserOption = (value: unknown): AgentAskUserOption | null => {
  if (!value || typeof value !== "object") {
    return null;
  }

  const option = value as Record<string, unknown>;
  const label =
    typeof option.label === "string"
      ? option.label.trim()
      : typeof option.value === "string"
        ? option.value.trim()
        : "";
  const optionValue =
    typeof option.value === "string" ? option.value.trim() : "";

  if (!label || !optionValue) {
    return null;
  }

  return {
    label,
    value: optionValue,
    description:
      typeof option.description === "string" && option.description.trim()
        ? option.description.trim()
        : undefined,
  };
};

const parseAskUserOptions = (value: unknown): AgentAskUserOption[] => {
  if (!Array.isArray(value)) {
    return [];
  }

  return value
    .map((option) => parseAskUserOption(option))
    .filter((option): option is AgentAskUserOption => !!option);
};

const parseAskUserKind = (
  value: unknown,
  options: AgentAskUserOption[]
): AgentAskUserKind => {
  if (value === "confirm") {
    return "confirm";
  }
  if (value === "choose" && options.length > 0) {
    return "choose";
  }
  return "input";
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

  const options = parseAskUserOptions(args.options);
  const ask: AgentPendingAsk = {
    toolCallId,
    prompt: args.prompt,
    kind: parseAskUserKind(args.kind, options),
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
  if (ask.kind === "choose") {
    ask.options = options;
  }

  return {
    kind: "ask_user",
    ask,
  };
};

const READ_ONLY_TOOL_NAMES = new Set([
  "get_page_state",
  "get_project_open_issues",
  "get_project_recent_activity",
  "get_workspace_attention",
  "get_workspace_recent_activity",
  "get_skill",
  "ask_user",
  "done",
]);

const READ_ONLY_TOOL_ERROR =
  "This tool is not available during this starter. Inspect the current page or ask the user to send a follow-up message to take action.";

export function getToolDefinitions(
  options: { runMode?: AgentRunMode } = {}
): ToolDefinition[] {
  const tools: ToolDefinition[] = [
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
| semantic (default) | Route path, params, title + context from the app store (project, database, issue, user info when available) |
| dom | Above + ref-labeled DOM tree of interactive elements |

Use mode="dom" before dom_action to get element refs like [e1]. Use semantic mode (default) to understand the current page context.`,
      parametersSchema: {
        type: "object",
        properties: {
          mode: {
            type: "string",
            enum: ["semantic", "dom"],
            description:
              'Default "semantic" returns route info + store context. "dom" adds a ref-labeled tree of interactive elements for use with dom_action.',
          },
        },
      },
    },
    {
      name: "get_project_open_issues",
      description:
        "List the current user's open Bytebase Issues in the Project bound to this chat. The result includes Issues awaiting the user's approval and Issues the user created. This tool has no parameters and cannot read another Project.",
      parametersSchema: {
        type: "object",
        properties: {},
      },
    },
    {
      name: "get_project_recent_activity",
      description:
        "List the newest recorded activity for the project bound to this chat. The result contains at most five safe summaries with the time, actor, action, resource, and outcome. It excludes audit-log request and response data. This tool has no parameters and cannot read another project.",
      parametersSchema: {
        type: "object",
        properties: {},
      },
    },
    {
      name: "get_workspace_attention",
      description:
        "List the current user's open Issues across the workspace bound to this chat. The result includes Issues awaiting the user's approval and Issues the user created, with at most five of each, and does not navigate. This tool has no parameters and cannot read another workspace.",
      parametersSchema: {
        type: "object",
        properties: {},
      },
    },
    {
      name: "get_workspace_recent_activity",
      description:
        "List the newest recorded activity in the workspace bound to this chat, including project and workspace administration changes. The result contains at most five safe summaries with the time, actor, action, resource, and outcome. It excludes audit-log request and response data. This tool has no parameters and cannot read another workspace.",
      parametersSchema: {
        type: "object",
        properties: {},
      },
    },
    {
      name: "dom_action",
      description: `Last-resort DOM interaction — use only when no API endpoint covers the action.
**Always call get_page_state(mode="dom") first** to get the element ref, such as [e1].

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
          ref: {
            type: "string",
            description:
              "Element ref from DOM tree (from get_page_state with mode='dom'), such as 'e1'",
          },
          value: {
            type: "string",
            description: "Value for input/select actions",
          },
        },
        required: ["type", "ref"],
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
        "Ask the user for missing information, confirmation, or an explicit choice. Use kind='input' for free-form text, kind='confirm' for confirm/cancel decisions, and kind='choose' with options=[{label,value}] when the user must pick from explicit options.",
      parametersSchema: {
        type: "object",
        properties: {
          prompt: {
            type: "string",
            description: "The question to show to the user.",
          },
          kind: {
            type: "string",
            enum: ["input", "confirm", "choose"],
            description:
              'Defaults to "input" when omitted. kind="choose" requires a non-empty options array.',
          },
          defaultValue: {
            type: "string",
            description:
              "Optional default answer to prefill for input prompts, or the default option value for choose prompts.",
          },
          confirmLabel: {
            type: "string",
            description: "Optional confirm button label for confirm prompts.",
          },
          cancelLabel: {
            type: "string",
            description: "Optional cancel button label for confirm prompts.",
          },
          options: {
            type: "array",
            description:
              "Required for choose prompts. Each option should include a user-visible label and a stable value.",
            items: {
              type: "object",
              properties: {
                label: {
                  type: "string",
                  description: "User-visible option label.",
                },
                value: {
                  type: "string",
                  description: "Stable option value returned to the model.",
                },
                description: {
                  type: "string",
                  description: "Optional helper text shown with the option.",
                },
              },
              required: ["label", "value"],
            },
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

  if (options.runMode === "read-only") {
    return tools.filter((tool) => READ_ONLY_TOOL_NAMES.has(tool.name));
  }
  return tools.filter(
    (tool) =>
      tool.name !== "get_project_open_issues" &&
      tool.name !== "get_project_recent_activity" &&
      tool.name !== "get_workspace_attention" &&
      tool.name !== "get_workspace_recent_activity"
  );
}

interface CreateToolExecutorOptions {
  chatId?: string;
  context?: string;
  onNavigate?: () => void;
  runMode?: AgentRunMode;
}

const PAGE_MUTATION_TOOL_NAMES = new Set(["navigate", "dom_action"]);
let pageMutationChatId: string | null = null;

const withPageMutationLock = async <T>(
  chatId: string | undefined,
  toolName: string,
  run: () => Promise<T>
): Promise<T> => {
  if (!PAGE_MUTATION_TOOL_NAMES.has(toolName) || !chatId) {
    return run();
  }
  if (pageMutationChatId && pageMutationChatId !== chatId) {
    throw new Error(
      "Another chat is already using a page-changing tool. Wait for it to finish, then retry this chat."
    );
  }

  pageMutationChatId = chatId;
  try {
    return await run();
  } finally {
    if (pageMutationChatId === chatId) {
      pageMutationChatId = null;
    }
  }
};

export function createToolExecutor(
  router: AppRouterInstance,
  options: CreateToolExecutorOptions = {}
): ToolExecutor {
  const navigateTool = createNavigateTool(router);
  const pageStateTool = createPageStateTool(router);
  const domActionTool = createDomActionTool(router);
  const projectOpenIssuesTool = createProjectOpenIssuesTool({
    projectName: options.context,
  });
  const projectRecentActivityTool = createProjectRecentActivityTool({
    projectName: options.context,
  });
  const workspaceRecentActivityTool = createWorkspaceRecentActivityTool({
    workspaceName: options.context,
  });
  const workspaceAttentionTool = createWorkspaceAttentionTool({
    workspaceName: options.context,
  });

  return async (
    name: string,
    args: Record<string, unknown>,
    toolCallId: string
  ): Promise<ToolExecutionResult> => {
    if (options.runMode === "read-only" && !READ_ONLY_TOOL_NAMES.has(name)) {
      return toToolResult(JSON.stringify({ error: READ_ONLY_TOOL_ERROR }));
    }

    switch (name) {
      case "search_api":
        return toToolResult(await searchApi(args as SearchApiArgs));
      case "call_api":
        return toToolResult(await callApi(args as unknown as CallApiArgs));
      case "navigate": {
        const result = await withPageMutationLock(
          options.chatId,
          name,
          async () =>
            await navigateTool(args as { path?: string; list?: boolean })
        );
        try {
          const payload = JSON.parse(result) as {
            navigated?: boolean;
          };
          if (payload.navigated) {
            options.onNavigate?.();
          }
        } catch {
          // Ignore malformed tool output and return the raw result.
        }
        return toToolResult(result);
      }
      case "get_page_state":
        return toToolResult(await pageStateTool(args as PageStateArgs));
      case "get_project_open_issues":
        return toToolResult(await projectOpenIssuesTool());
      case "get_project_recent_activity":
        return toToolResult(await projectRecentActivityTool());
      case "get_workspace_attention":
        return toToolResult(await workspaceAttentionTool());
      case "get_workspace_recent_activity":
        return toToolResult(await workspaceRecentActivityTool());
      case "dom_action":
        return toToolResult(
          await withPageMutationLock(options.chatId, name, async () => {
            return await domActionTool(args as unknown as DomActionArgs);
          })
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
