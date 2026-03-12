import type { Router } from "vue-router";
import type { ToolDefinition, ToolExecutor } from "../types";
import { searchApi } from "./searchApi";
import { callApi } from "./callApi";
import { createNavigateTool } from "./navigate";
import { createPageStateTool } from "./pageState";

export function getToolDefinitions(): ToolDefinition[] {
  return [
    {
      name: "search_api",
      description:
        "Search for available Bytebase API endpoints by keyword. Returns matching operations with their IDs, paths, and descriptions.",
      parametersSchema: {
        type: "object",
        properties: {
          query: {
            type: "string",
            description:
              "Search query (e.g., 'list projects', 'create database')",
          },
        },
        required: ["query"],
      },
    },
    {
      name: "call_api",
      description:
        "Execute a Bytebase API endpoint. Use search_api first to find the operation_id. All APIs use Connect protocol (POST with JSON body).",
      parametersSchema: {
        type: "object",
        properties: {
          operation_id: {
            type: "string",
            description: "The operation ID from search_api results",
          },
          body: {
            type: "object",
            description:
              "Request body fields. For Get/List operations, include resource name or parent. For mutations, include the full request payload.",
          },
        },
        required: ["operation_id"],
      },
    },
    {
      name: "navigate",
      description:
        "Navigate to a page in Bytebase. Use URL paths like /projects, /sql-editor, /instances, /settings.",
      parametersSchema: {
        type: "object",
        properties: {
          path: {
            type: "string",
            description: "URL path to navigate to",
          },
        },
        required: ["path"],
      },
    },
    {
      name: "get_page_state",
      description:
        "Get information about the current page including path, title, and route parameters.",
      parametersSchema: {
        type: "object",
        properties: {},
      },
    },
  ];
}

export function createToolExecutor(router: Router): ToolExecutor {
  const navigateTool = createNavigateTool(router);
  const pageStateTool = createPageStateTool(router);

  return async (
    name: string,
    args: Record<string, unknown>
  ): Promise<string> => {
    switch (name) {
      case "search_api":
        return searchApi(args as { query: string });
      case "call_api":
        return callApi(
          args as { operation_id: string; body?: Record<string, unknown> }
        );
      case "navigate":
        return navigateTool(args as { path: string });
      case "get_page_state":
        return pageStateTool();
      default:
        return JSON.stringify({ error: `Unknown tool: ${name}` });
    }
  };
}
