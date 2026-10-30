import { create } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import {
  databaseServiceClientConnect,
  issueServiceClientConnect,
  projectServiceClientConnect,
} from "@/api";
import type { AppRouterInstance } from "@/app/router";
import { silentContextKey } from "@/api/context-key";
import {
  getNameParentTokens,
  issueNamePrefix,
  projectNamePrefix,
} from "@/stores/modules/v1/common";
import { State } from "@/types/proto-es/v1/common_pb";
import { GetDatabaseRequestSchema } from "@/types/proto-es/v1/database_service_pb";
import { GetIssueRequestSchema } from "@/types/proto-es/v1/issue_service_pb";
import { GetProjectRequestSchema } from "@/types/proto-es/v1/project_service_pb";
import { getSkill, type GetSkillArgs } from "../skills";
import { normalizeAgentHandoff } from "../handoff";
import { normalizeAgentResponseActions } from "../responseActions";
import type {
  AgentAskUserKind,
  AgentAskUserOption,
  AgentPendingAsk,
  AgentRunMode,
  ToolDefinition,
  ToolExecutionResult,
  ToolExecutor,
} from "../types";
import { apiCallNeedsConfirmation } from "./apiMethodClass";
import { checkPreparedApiPermission } from "./apiPermissionPreflight";
import {
  dispatchPreparedApiCall,
  prepareApiCall,
  type CallApiArgs,
} from "./callApi";
import { createDatabaseCountTool } from "./databaseCount";
import { createDatabaseOverviewTool } from "./databaseOverview";
import { createDomActionTool, type DomActionArgs } from "./domAction";
import { createIssueOverviewTool } from "./issueOverview";
import { prepareIssueDecisionPermission } from "./issueDecisionPermission";
import { createNavigateTool } from "./navigate";
import { createPageStateTool, type PageStateArgs } from "./pageState";
import { createPendingApiCall } from "./pendingApiCall";
import { createProjectOpenIssuesTool } from "./projectOpenIssues";
import { createProjectGovernanceOverviewTool } from "./projectGovernanceOverview";
import { createProjectRecentActivityTool } from "./projectRecentActivity";
import { validateResourceChoices } from "./resourceChoices";
import { createProjectOverviewTool } from "./projectOverview";
import { apiTargetsAnotherProject } from "./referencedProjectApiScope";
import { searchApi, type SearchApiArgs } from "./searchApi";
import { createWorkspaceOverviewTool } from "./workspaceOverview";
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
  if (typeof args.success !== "boolean") {
    throw new Error("done requires a boolean success value");
  }
  const handoff = args.success
    ? normalizeAgentHandoff(args.handoff)
    : undefined;
  const actions = args.success
    ? normalizeAgentResponseActions(args.actions)
    : [];
  return {
    kind: "done",
    text: args.text,
    success: args.success,
    ...(handoff ? { handoff } : {}),
    ...(actions.length > 0 ? { actions } : {}),
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
  "get_project_overview",
  "get_project_governance_overview",
  "get_issue_overview",
  "get_database_overview",
  "get_database_count",
  "get_workspace_overview",
  "get_project_open_issues",
  "get_project_recent_activity",
  "get_workspace_attention",
  "get_workspace_recent_activity",
  "get_skill",
  "ask_user",
  "done",
]);
const REFERENCED_PROJECT_TOOL_NAMES = new Set([
  ...READ_ONLY_TOOL_NAMES,
  "search_api",
  "call_api",
]);

const READ_ONLY_TOOL_ERROR =
  "This tool is not available during a read-only request. Ask a follow-up question to take action.";
const REFERENCED_PROJECT_TOOL_ERROR =
  "This tool is not available for a referenced-project turn. Use the available API tools or ask a follow-up question without the reference.";

const PROJECT_SCOPED_TOOL_NAMES = new Set([
  "get_project_overview",
  "get_project_governance_overview",
  "get_issue_overview",
  "get_database_overview",
  "get_database_count",
  "get_project_open_issues",
  "get_project_recent_activity",
]);

const isCanonicalProjectName = (value: string | undefined): value is string =>
  !!value && getNameParentTokens(value, [projectNamePrefix]).length === 1;

const ISSUE_NAME_API_METHODS = new Set([
  "GetIssue",
  "ApproveIssue",
  "RejectIssue",
  "RequestIssue",
]);
const ISSUE_PARENT_API_METHODS = new Set([
  "ListIssueComments",
  "CreateIssueComment",
  "UpdateIssueComment",
]);

const targetsSelectedIssue = (
  args: Record<string, unknown>,
  referencedIssue: string
): boolean => {
  const operationId = args.operationId;
  const body = args.body;
  if (typeof operationId !== "string" || !body || typeof body !== "object") {
    return false;
  }
  const method = operationId.match(
    /^(?:bytebase\.v1\.)?IssueService[/.]([A-Za-z]+)$/
  )?.[1];
  if (!method) return false;

  const request = body as Record<string, unknown>;
  if (ISSUE_NAME_API_METHODS.has(method)) {
    return request.name === referencedIssue;
  }
  if (ISSUE_PARENT_API_METHODS.has(method)) {
    return request.parent === referencedIssue;
  }
  if (method === "UpdateIssue") {
    const issue = request.issue;
    return (
      !!issue &&
      typeof issue === "object" &&
      (issue as Record<string, unknown>).name === referencedIssue
    );
  }
  return false;
};

export function getToolDefinitions(
  options: { runMode?: AgentRunMode; referencedProject?: string } = {}
): ToolDefinition[] {
  const referencedProject = isCanonicalProjectName(options.referencedProject)
    ? options.referencedProject
    : undefined;
  const projectScope = referencedProject
    ? `the selected referenced project (${referencedProject}) for this turn`
    : "the project bound to this chat";
  const projectScopeRestriction = referencedProject
    ? "This tool cannot read any other project; the chat context remains unchanged."
    : "This tool cannot read another project.";
  const tools: ToolDefinition[] = [
    {
      name: "search_api",
      description: `Find API endpoints and inspect request/response schemas. Search by the user's task before calling an endpoint.

| Mode | Parameters | Result |
|------|------------|--------|
| Search | query="list databases in this project" | Up to five matching endpoints |
| Browse | service="SQLService" | All endpoints in a service |
| Details | operationId="SQLService/Query" | Request/response schema |
| Schema | schema="Instance" | Message type definition |

**Typical workflow:** search_api(query="...") → search_api(operationId="...") → call_api(...)`,
      parametersSchema: {
        type: "object",
        properties: {
          query: {
            type: "string",
            description:
              "Natural-language description of the API capability to find. Returns up to five ranked endpoints.",
          },
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
      description: referencedProject
        ? `Call a Bytebase API for this turn. Use search_api first to inspect the operation and request schema. Use ${referencedProject} for project-scoped requests. Ask the member to split a question that needs another project. Calls use the signed-in member's existing permissions and may change data; do not make changes unless the member explicitly asks. A nextPageToken means the response is partial; do not automatically fetch every page to count resources.`
        : `Execute a Bytebase API endpoint. **Use search_api first to get operationId and schema.**

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
      description: `Navigate to a page in Bytebase, or list a bounded set of valid routes.

| Mode | Parameters | Result |
|------|------------|--------|
| Navigate | path="/projects" | Navigates to the page |
| List | list=true, query="database" | Returns up to 30 matching route patterns, plus the total and whether results were truncated |

Use list=true only when the exact path is unclear. Pass a focused query to avoid broad route discovery. Do not guess paths — a wrong path causes a 404.`,
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
              "Set to true to list up to 30 matching route patterns instead of navigating.",
          },
          query: {
            type: "string",
            description:
              "Optional case-insensitive substring filter for route patterns when list is true.",
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
      name: "get_project_overview",
      description: `Get a bounded snapshot of ${projectScope}. The result includes the current member's relevant open Issues, open database-change Issues, and accessible databases, with partial or unavailable facets made explicit. It also includes instance availability, eligible unassigned database availability, and setup permissions when the project is accessible. A recommended setup step is provided only when the project is confirmed empty. Use this before recommending what to do next or describing current project readiness. Answer the member's actual question from the returned evidence, rather than listing generic capabilities. This tool has no parameters. ${projectScopeRestriction}`,
      parametersSchema: {
        type: "object",
        properties: {},
      },
    },
    {
      name: "get_project_governance_overview",
      description: `Get a bounded snapshot of how Database Changes are governed in ${projectScope}. The result includes the project's approval, plan-check, SQL-review, approver, and rollout-concurrency settings, plus the number of configured database-change and fallback approval rules. Approval-rule conditions are evaluated per Database Change, so rule counts do not prove that a rule applies to a particular change. This tool has no parameters. ${projectScopeRestriction}`,
      parametersSchema: {
        type: "object",
        properties: {},
      },
    },
    {
      name: "get_issue_overview",
      description: `Get a bounded, read-only overview of one Issue in ${projectScope}. It includes the Issue's approval state, linked Plan and plan-check state, rule and AI review-run state, plus Rollout, up to ten currently failed Tasks, and up to ten task-run statuses when available. A failed Task Run can be historical; use the Rollout's failedTasks for current failures. Pass a canonical Issue name from this project. Each facet explicitly reports whether it is available, not configured, not started, or unavailable. A review run marked not started has no recorded run; this does not establish whether that review is configured or applicable. ${projectScopeRestriction} This tool cannot take action.`,
      parametersSchema: {
        type: "object",
        properties: {
          issueName: {
            type: "string",
            description: `Canonical Issue resource name in ${projectScope}, for example projects/my-project/issues/123.`,
          },
        },
        required: ["issueName"],
      },
    },
    {
      name: "get_database_overview",
      description: `Get a bounded, read-only overview of one database in ${projectScope}. It includes the database state, instance, effective environment, last applied release, DML prior-backup readiness (\`dmlPriorBackupAvailable\`), and sync status. This readiness does not prove that a recoverable backup exists. Pass a canonical database name from this project. Each resource facet explicitly reports whether it is available, not configured, or unavailable. This tool reads only DatabaseService.GetDatabase; it does not inspect existing backups or recovery points, schema, metadata, labels, data-source configuration, or sync errors, and cannot take action. ${projectScopeRestriction}`,
      parametersSchema: {
        type: "object",
        properties: {
          databaseName: {
            type: "string",
            description: `Canonical database resource name in ${projectScope}, for example projects/my-project/instances/prod/databases/app.`,
          },
        },
        required: ["databaseName"],
      },
    },
    {
      name: "get_database_count",
      description: `Get a bounded active-database count visible to the current member in ${referencedProject ? projectScope : "the Project or Workspace bound to this chat"}. With a selected @Database in the current turn, set scope to "referenced-instance" to count active databases on that database's canonical instance scope. A workspace instance may span projects; a project instance does not. This tool reads one page and returns its count and partial flag, never database records. If partial is true, report "at least N", not an exact total. It cannot choose an arbitrary instance or apply filters. Do not call repeatedly to enumerate all pages.`,
      parametersSchema: {
        type: "object",
        properties: {
          scope: {
            type: "string",
            enum: ["context", "referenced-instance"],
            description:
              "Omit for the chat or referenced Project total. Use referenced-instance only for the current turn's selected @Database instance.",
          },
        },
      },
    },
    {
      name: "get_workspace_overview",
      description:
        "Get a bounded snapshot of the workspace bound to this chat. The result includes setup readiness for accessible projects, instances, and databases plus the current member's open Issues. Recent activity is deliberately excluded; use get_workspace_recent_activity only when the member asks what changed recently. Answer the member's actual question from the returned evidence, rather than listing generic capabilities. This tool has no parameters and cannot read another workspace.",
      parametersSchema: {
        type: "object",
        properties: {},
      },
    },
    {
      name: "get_project_open_issues",
      description: `List the current user's open Bytebase Issues in ${projectScope}. The result includes Issues awaiting the user's approval and Issues the user created. This tool has no parameters. ${projectScopeRestriction}`,
      parametersSchema: {
        type: "object",
        properties: {},
      },
    },
    {
      name: "get_project_recent_activity",
      description: `List the newest recorded activity for ${projectScope}. The result contains at most five safe summaries with the time, actor, action, resource, and outcome. It excludes audit-log request and response data. This tool has no parameters. ${projectScopeRestriction}`,
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
        "Ask the user for missing information, confirmation, or an explicit choice. Use kind='input' for free-form text, kind='confirm' for confirm/cancel decisions, and kind='choose' with options=[{label,value}] when the user must pick from explicit options. For database choices, set resourceKind='database' and provide canonical option values; labels are refreshed from the API.",
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
          resourceKind: {
            type: "string",
            enum: ["database"],
            description:
              "Validate database choices against fresh database data.",
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
              required: ["value"],
            },
          },
        },
        required: ["prompt"],
      },
    },
    {
      name: "done",
      description:
        "Complete the current task explicitly. Provide final user-visible text, success, and actions. For a successful answer, propose at most one primary action: Open for an evidenced next step, or draft-plan for generated DDL grounded in this turn's selected @Database. A draft-plan action names only its project and database; put the SQL in exactly one fenced `sql` block in text, never in the action. Normally include 2-3 distinct, contextual Continue questions in English, with or without a primary action. Include fewer or none when no further meaningful questions exist; do not invent filler. Use actions: [] for a failed answer or when no useful action exists. Actions require a member click; the runtime validates their targets.",
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
          actions: {
            type: "array",
            description:
              "Optional member-clicked actions. Use at most one primary Open, draft-plan, or draft-issue-comment, plus up to three distinct Continue questions. Open uses a current-turn observed destination, never a URL or invented resource. draft-plan opens a populated, unsaved Plan. draft-issue-comment opens an unsent review comment for an Issue observed in this turn; it never posts automatically. Continue fills the composer without sending. Do not include API requests, SQL, or resource @ mentions in an action.",
            maxItems: 4,
            items: {
              type: "object",
              properties: {
                kind: {
                  type: "string",
                  enum: [
                    "open",
                    "draft-plan",
                    "draft-issue-comment",
                    "continue",
                  ],
                },
                target: {
                  type: "object",
                  properties: {
                    kind: {
                      type: "string",
                      enum: [
                        "review-workspace-issues",
                        "review-project-issues",
                        "review-database-changes",
                        "transfer-project-databases",
                        "connect-project-instance",
                        "create-project-database",
                        "view-failed-tasks",
                        "open-issue",
                        "review-issue-plan",
                        "view-failed-task",
                      ],
                    },
                    workspace: { type: "string" },
                    project: { type: "string" },
                    plan: { type: "string" },
                    issue: { type: "string" },
                    task: { type: "string" },
                  },
                  required: ["kind"],
                },
                project: {
                  type: "string",
                  description:
                    "For draft-plan, the exact owning project of this turn's selected @Database.",
                },
                database: {
                  type: "string",
                  description:
                    "For draft-plan, the exact canonical name of this turn's selected @Database.",
                },
                issue: {
                  type: "string",
                  description:
                    "For draft-issue-comment, the exact canonical name of this turn's observed Issue.",
                },
                comment: {
                  type: "string",
                  description:
                    "For draft-issue-comment, the review comment to place in the Issue editor without posting.",
                },
                prompt: { type: "string" },
              },
              required: ["kind"],
            },
          },
        },
        required: ["text", "success", "actions"],
      },
    },
  ];

  if (referencedProject) {
    return tools.filter((tool) => REFERENCED_PROJECT_TOOL_NAMES.has(tool.name));
  }
  if (options.runMode === "read-only") {
    return tools.filter((tool) => READ_ONLY_TOOL_NAMES.has(tool.name));
  }
  return tools.filter(
    (tool) =>
      (Boolean(referencedProject) || tool.name !== "get_project_open_issues") &&
      (Boolean(referencedProject) ||
        tool.name !== "get_project_recent_activity") &&
      tool.name !== "get_workspace_attention" &&
      tool.name !== "get_workspace_recent_activity"
  );
}

interface CreateToolExecutorOptions {
  chatId?: string;
  userName?: string;
  context?: string;
  workspaceName?: string;
  referencedProject?: string;
  referencedDatabase?: string;
  referencedIssue?: string;
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
  const referencedProject = isCanonicalProjectName(options.referencedProject)
    ? options.referencedProject
    : undefined;
  const referencedIssue =
    referencedProject &&
    !options.referencedDatabase &&
    options.referencedIssue?.startsWith(`${referencedProject}/issues/`) &&
    getNameParentTokens(options.referencedIssue, [
      projectNamePrefix,
      issueNamePrefix,
    ]).length === 2
      ? options.referencedIssue
      : undefined;
  const referencedDatabase = options.referencedDatabase;
  const projectName = referencedProject ?? options.context;
  const navigateTool = createNavigateTool(router);
  const pageStateTool = createPageStateTool(router);
  const domActionTool = createDomActionTool(router);
  const projectOpenIssuesTool = createProjectOpenIssuesTool({
    projectName,
  });
  const projectOverviewTool = createProjectOverviewTool({
    projectName,
  });
  const projectGovernanceOverviewTool = createProjectGovernanceOverviewTool({
    projectName,
  });
  const issueOverviewTool = createIssueOverviewTool({
    projectName,
  });
  const databaseOverviewTool = createDatabaseOverviewTool({
    projectName,
  });
  const databaseCountTool = createDatabaseCountTool({
    scopeName: projectName,
    ...(options.referencedDatabase
      ? {
          referencedDatabase: options.referencedDatabase,
          referencedProject,
        }
      : {}),
  });
  const workspaceOverviewTool = createWorkspaceOverviewTool({
    workspaceName: options.context,
  });
  const projectRecentActivityTool = createProjectRecentActivityTool({
    projectName,
  });
  const workspaceRecentActivityTool = createWorkspaceRecentActivityTool({
    workspaceName: options.context,
  });
  const workspaceAttentionTool = createWorkspaceAttentionTool({
    workspaceName: options.context,
  });
  let referencedProjectAccess: Promise<boolean> | undefined;
  let referencedDatabaseAccess: Promise<boolean> | undefined;
  let referencedIssueAccess: Promise<boolean> | undefined;
  const hasReferencedProjectAccess = (): Promise<boolean> => {
    if (!referencedProject) return Promise.resolve(true);
    referencedProjectAccess ??= projectServiceClientConnect
      .getProject(
        create(GetProjectRequestSchema, { name: referencedProject }),
        { contextValues: createContextValues().set(silentContextKey, true) }
      )
      .then(
        (project) =>
          project.name === referencedProject && project.state === State.ACTIVE
      )
      .catch(() => false);
    return referencedProjectAccess;
  };
  const hasReferencedDatabaseAccess = (): Promise<boolean> => {
    if (!referencedDatabase || !referencedProject)
      return Promise.resolve(false);
    referencedDatabaseAccess ??= databaseServiceClientConnect
      .getDatabase(
        create(GetDatabaseRequestSchema, { name: referencedDatabase }),
        { contextValues: createContextValues().set(silentContextKey, true) }
      )
      .then(
        (database) =>
          database.name === referencedDatabase &&
          database.project === referencedProject &&
          database.state === State.ACTIVE
      )
      .catch(() => false);
    return referencedDatabaseAccess;
  };
  const hasReferencedIssueAccess = (): Promise<boolean> => {
    if (!referencedIssue) return Promise.resolve(false);
    referencedIssueAccess ??= issueServiceClientConnect
      .getIssue(
        create(GetIssueRequestSchema, { name: referencedIssue, force: true }),
        { contextValues: createContextValues().set(silentContextKey, true) }
      )
      .then((issue) => issue.name === referencedIssue)
      .catch(() => false);
    return referencedIssueAccess;
  };

  const canUseIssueAccess = (
    name: string,
    args: Record<string, unknown>
  ): boolean =>
    !!referencedIssue &&
    ((name === "get_issue_overview" && args.issueName === referencedIssue) ||
      (name === "call_api" && targetsSelectedIssue(args, referencedIssue)));
  const canUseDatabaseAccess = (
    name: string,
    args: Record<string, unknown>
  ): boolean =>
    !!referencedDatabase &&
    ((name === "get_database_overview" &&
      args.databaseName === referencedDatabase) ||
      name === "get_database_count" ||
      name === "call_api");

  return async (
    name: string,
    args: Record<string, unknown>,
    toolCallId: string
  ): Promise<ToolExecutionResult> => {
    const allowedTools = referencedProject
      ? REFERENCED_PROJECT_TOOL_NAMES
      : READ_ONLY_TOOL_NAMES;
    if (
      (options.runMode === "read-only" || referencedProject) &&
      !allowedTools.has(name)
    ) {
      return toToolResult(
        JSON.stringify({
          error: referencedProject
            ? REFERENCED_PROJECT_TOOL_ERROR
            : READ_ONLY_TOOL_ERROR,
        })
      );
    }
    if (
      referencedProject &&
      (PROJECT_SCOPED_TOOL_NAMES.has(name) || name === "call_api") &&
      !(
        canUseDatabaseAccess(name, args) &&
        (await hasReferencedDatabaseAccess())
      ) &&
      !(await hasReferencedProjectAccess()) &&
      !(canUseIssueAccess(name, args) && (await hasReferencedIssueAccess()))
    ) {
      return toToolResult(
        JSON.stringify({
          error:
            "The referenced project is unavailable or you do not have access.",
        })
      );
    }

    switch (name) {
      case "search_api":
        return toToolResult(await searchApi(args as SearchApiArgs));
      case "call_api": {
        const apiArgs = args as unknown as CallApiArgs;
        const prepared = prepareApiCall(apiArgs);
        if (prepared.kind === "error") {
          return toToolResult(prepared.result);
        }
        if (
          referencedProject &&
          (await apiTargetsAnotherProject(prepared.call, referencedProject))
        ) {
          return toToolResult(
            JSON.stringify({
              error: `This API request is outside the referenced project (${referencedProject}). Ask in a separate message to use another project.`,
            })
          );
        }
        const access = await checkPreparedApiPermission(prepared.call);
        if (access.kind === "denied") {
          return toToolResult(access.result);
        }
        const issueDecision = await prepareIssueDecisionPermission(
          prepared.call,
          options.referencedIssue
        );
        if (issueDecision.kind === "error") {
          return toToolResult(issueDecision.result);
        }
        if (
          issueDecision.kind === "not-issue-decision" &&
          !apiCallNeedsConfirmation(prepared.call.operationId)
        ) {
          return toToolResult(await dispatchPreparedApiCall(prepared.call));
        }
        if (!options.chatId || !options.userName) {
          return toToolResult(
            JSON.stringify({
              error: "Cannot request API permission without a signed-in chat.",
            })
          );
        }
        const { actionId } = createPendingApiCall({
          chatId: options.chatId,
          toolCallId,
          userName: options.userName,
          call: prepared.call,
          issueDecision:
            issueDecision.kind === "issue-decision"
              ? issueDecision.decision
              : undefined,
        });
        return {
          kind: "ask_user",
          ask: {
            kind: "confirm",
            toolCallId,
            prompt: prepared.call.operationId,
            apiPermission: {
              actionId,
              operationId: prepared.call.operationId,
            },
          },
        };
      }
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
      case "get_project_overview":
        return toToolResult(await projectOverviewTool());
      case "get_project_governance_overview":
        return toToolResult(await projectGovernanceOverviewTool());
      case "get_issue_overview":
        return toToolResult(await issueOverviewTool(args));
      case "get_database_overview":
        return toToolResult(await databaseOverviewTool(args));
      case "get_database_count":
        return toToolResult(await databaseCountTool(args));
      case "get_workspace_overview":
        return toToolResult(await workspaceOverviewTool());
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
      case "ask_user": {
        if (args.kind !== "choose" || !args.resourceKind) {
          return parseAskUserArgs(args, toolCallId);
        }
        if (args.resourceKind !== "database") {
          return toToolResult(
            "Error: Unsupported resource choice kind. Ask a free-form question instead."
          );
        }
        if (
          args.resourceKind === "database" &&
          !isCanonicalProjectName(projectName)
        ) {
          return toToolResult(
            "Error: A Project is required to verify database choices. Ask the member which Project first."
          );
        }
        const rawOptions = Array.isArray(args.options) ? args.options : [];
        const values = rawOptions.map((option) =>
          option &&
          typeof option === "object" &&
          typeof option.value === "string"
            ? option.value
            : undefined
        );
        if (values.some((value) => value === undefined)) {
          return toToolResult(
            "Error: Resource choices require canonical option values. Ask a free-form question instead."
          );
        }
        const verified = await validateResourceChoices(
          args.resourceKind,
          values as string[],
          projectName,
          options.workspaceName
        );
        if (!verified) {
          return toToolResult(
            "Error: Resource choices could not be verified with current access. Ask a free-form question instead; do not describe the list as empty."
          );
        }
        return parseAskUserArgs({ ...args, options: verified }, toolCallId);
      }
      case "done":
        return parseDoneArgs(args);
      default:
        return toToolResult(JSON.stringify({ error: `Unknown tool: ${name}` }));
    }
  };
}
