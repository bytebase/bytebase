import type { Router } from "vue-router";
import type { ToolDefinition, ToolExecutor } from "../types";
import { searchApi, type SearchApiArgs } from "./searchApi";
import { callApi, type CallApiArgs } from "./callApi";
import { createNavigateTool } from "./navigate";
import { createPageStateTool } from "./pageState";

export function getToolDefinitions(): ToolDefinition[] {
  return [
    {
      name: "search_api",
      description: `Discover Bytebase API endpoints. **Always call before call_api - never guess schemas.**

| Mode | Parameters | Result |
|------|------------|--------|
| List | (none) | All services |
| Browse | service="SQLService" | All endpoints in service |
| Search | query="database" | Matching endpoints |
| Filter | service+query | Search within service |
| Details | operationId="SQLService/Query" | Request/response schema |
| Schema | schema="Instance" | Message type definition |

**Workflow:** search_api() → search_api(operationId="...") → call_api(...)`,
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
          query: {
            type: "string",
            description:
              'Free-text search for API endpoints. Examples: "create database", "execute sql"',
          },
          service: {
            type: "string",
            description:
              'Filter to a specific service. Examples: "SQLService", "DatabaseService"',
          },
          limit: {
            type: "number",
            description:
              "Maximum number of results to return (default: 5, max: 50)",
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
        return searchApi(args as SearchApiArgs);
      case "call_api":
        return callApi(args as unknown as CallApiArgs);
      case "navigate":
        return navigateTool(args as { path: string });
      case "get_page_state":
        return pageStateTool();
      default:
        return JSON.stringify({ error: `Unknown tool: ${name}` });
    }
  };
}
