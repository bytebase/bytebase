# Page Agent Phase 1 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a working in-app AI agent with API tools, navigation, and a floating chat window — backed by a new `AIService` Connect RPC endpoint.

**Architecture:** Frontend runs the tool-calling loop (agent core). Backend provides a stateless LLM proxy (`AIService.Chat`) that reads AI settings and forwards to the configured provider (OpenAI/Claude/Gemini) with tool-calling support. Tools execute locally in the browser.

**Tech Stack:** Proto + buf (API definition), Go + Connect RPC (backend), Vue 3 + TypeScript + Pinia (frontend), Naive UI (components)

**Reference:** See `docs/plans/2026-03-12-page-agent-design.md` for full design context.

---

### Task 1: Proto — Define `AIService`

**Files:**
- Create: `proto/v1/v1/ai_service.proto`

**Step 1: Create the proto file**

```proto
syntax = "proto3";

package bytebase.v1;

import "google/api/annotations.proto";
import "v1/annotation.proto";

option go_package = "github.com/bytebase/bytebase/backend/generated-go/v1";

service AIService {
  rpc Chat(AIChatRequest) returns (AIChatResponse) {
    option (google.api.http) = {
      post: "/v1/ai/chat"
      body: "*"
    };
    option (bytebase.v1.auth_method) = CUSTOM;
  }
}

message AIChatRequest {
  repeated AIChatMessage messages = 1;
  repeated AIChatToolDefinition tools = 2;
}

message AIChatMessage {
  // "system", "user", "assistant", "tool"
  string role = 1;
  // Text content (for user/assistant/system messages).
  optional string content = 2;
  // Tool calls returned by the assistant.
  repeated AIChatToolCall tool_calls = 3;
  // Tool call ID (when role = "tool", identifies which tool call this is a response to).
  optional string tool_call_id = 4;
}

message AIChatToolDefinition {
  string name = 1;
  string description = 2;
  // JSON Schema for parameters.
  string parameters_schema = 3;
}

message AIChatToolCall {
  string id = 1;
  string name = 2;
  // JSON-encoded arguments.
  string arguments = 3;
}

message AIChatResponse {
  // Text content from the assistant (may be empty if only tool calls).
  optional string content = 1;
  // Tool calls the assistant wants to make.
  repeated AIChatToolCall tool_calls = 2;
}
```

**Step 2: Format and lint**

Run: `buf format -w proto && buf lint proto`
Expected: No errors.

**Step 3: Generate code**

Run: `cd proto && buf generate`
Expected: Generated files appear in `backend/generated-go/v1/` and `frontend/src/types/proto-es/v1/`.

**Step 4: Verify generated files exist**

Check that these files were created:
- `backend/generated-go/v1/ai_service.pb.go`
- `backend/generated-go/v1/v1connect/ai_service.connect.go`
- `frontend/src/types/proto-es/v1/ai_service_pb.ts` (or `.js` + `.d.ts`)

**Step 5: Commit**

```
feat: add AIService proto with Chat RPC for tool-calling
```

---

### Task 2: Backend — Implement `AIService.Chat`

**Files:**
- Create: `backend/api/v1/ai_service.go`
- Modify: `backend/server/grpc_routes.go` (register the service)

**Step 1: Create the service struct**

Create `backend/api/v1/ai_service.go`. The service needs access to the store (for AI settings). Reference the existing `SQLService` pattern in `backend/api/v1/sql_service_ai.go` for provider-specific request/response translation.

```go
package v1

import (
	"context"

	"connectrpc.com/connect"

	"github.com/bytebase/bytebase/backend/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	v1connect "github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
)

type AIService struct {
	v1connect.UnimplementedAIServiceHandler
	store *store.Store
}

func NewAIService(store *store.Store) *AIService {
	return &AIService{store: store}
}
```

**Step 2: Implement the `Chat` method**

The method should:
1. Read `AISetting` from store (same as `sql_service_ai.go` line 104)
2. Check `aiSetting.Enabled` — return `FailedPrecondition` if not
3. Switch on `aiSetting.Provider` and call the appropriate provider
4. Each provider function translates:
   - `AIChatRequest.messages` → provider message format (including tool calls and tool results)
   - `AIChatRequest.tools` → provider tool definition format
   - Provider response → `AIChatResponse` (extract text content + tool calls)

Provider-specific translation:

**OpenAI/Azure OpenAI:**
- Messages map directly (role, content, tool_calls, tool_call_id)
- Tools use `{"type": "function", "function": {"name": ..., "description": ..., "parameters": <parsed JSON schema>}}`
- Response: `choices[0].message.content` + `choices[0].message.tool_calls`

**Claude:**
- System message extracted from messages array, passed as top-level `system` param
- User/assistant messages use content blocks
- Tool results use `{"type": "tool_result", "tool_use_id": ..., "content": ...}`
- Tools use `{"name": ..., "description": ..., "input_schema": <parsed JSON schema>}`
- Response: extract `text` blocks for content, `tool_use` blocks for tool calls

**Gemini:**
- Messages map to `contents` with `parts`
- Tools use `{"functionDeclarations": [{"name": ..., "description": ..., "parameters": <parsed JSON schema>}]}`
- Response: extract `text` parts for content, `functionCall` parts for tool calls

Reference `backend/api/v1/sql_service_ai.go` for the HTTP request patterns (endpoint URLs, auth headers, response parsing) for each provider. Reuse the same HTTP client approach but with tool-calling fields added.

**Step 3: Register the service in `grpc_routes.go`**

Add to `backend/server/grpc_routes.go` following the existing pattern:

1. Instantiate: `aiService := apiv1.NewAIService(stores)`
2. Create handler: `aiPath, aiHandler := v1connect.NewAIServiceHandler(aiService, handlerOpts)`
3. Register: `connectHandlers[aiPath] = aiHandler`
4. Add to reflector: `v1connect.AIServiceName`

**Step 4: Build**

Run: `go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go`
Expected: Compiles without errors.

**Step 5: Lint**

Run: `golangci-lint run --allow-parallel-runners`
Expected: No new lint errors. Run repeatedly until clean.

**Step 6: Commit**

```
feat: implement AIService.Chat backend proxy with tool-calling support
```

---

### Task 3: Frontend — Connect Client for `AIService`

**Files:**
- Modify: `frontend/src/connect/index.ts`

**Step 1: Add the client**

In `frontend/src/connect/index.ts`, add:

```typescript
import { AIService } from "@/types/proto-es/v1/ai_service_pb";

export const aiServiceClientConnect = createClient(AIService, transport);
```

Follow the exact same pattern as the existing `projectServiceClientConnect` and others in that file.

**Step 2: Verify types compile**

Run: `pnpm --dir frontend type-check`
Expected: No type errors.

**Step 3: Commit**

```
feat: add frontend Connect client for AIService
```

---

### Task 4: Frontend — Agent Loop

**Files:**
- Create: `frontend/src/plugins/agent/logic/agentLoop.ts`
- Create: `frontend/src/plugins/agent/logic/types.ts`

**Step 1: Define types**

Create `frontend/src/plugins/agent/logic/types.ts`:

```typescript
export interface ToolDefinition {
  name: string;
  description: string;
  parametersSchema: Record<string, unknown>; // JSON Schema
}

export interface ToolCall {
  id: string;
  name: string;
  arguments: string; // JSON-encoded
}

export interface Message {
  role: "system" | "user" | "assistant" | "tool";
  content?: string;
  toolCalls?: ToolCall[];
  toolCallId?: string;
}

export type ToolExecutor = (
  name: string,
  args: Record<string, unknown>
) => Promise<string>;
```

**Step 2: Implement the agent loop**

Create `frontend/src/plugins/agent/logic/agentLoop.ts`:

The loop should:
1. Accept: `messages: Message[]`, `tools: ToolDefinition[]`, `executeToolCall: ToolExecutor`
2. Call `aiServiceClientConnect.chat()` with messages + tools
3. If response has `toolCalls`: execute each tool call via `executeToolCall`, append assistant message (with tool calls) and tool result messages, then loop (go to step 2)
4. If response has only `content` (no tool calls): return the final text
5. Cap iterations (e.g., max 10 loops) to prevent infinite loops
6. Use an `AbortSignal` parameter so the UI can cancel mid-loop

The function should yield intermediate state (tool calls, results) for the UI to display. Use a callback pattern:

```typescript
export interface AgentCallbacks {
  onToolCall?: (toolCall: ToolCall) => void;
  onToolResult?: (toolCallId: string, result: string) => void;
  onText?: (text: string) => void;
}

export async function runAgentLoop(
  messages: Message[],
  tools: ToolDefinition[],
  executeTool: ToolExecutor,
  callbacks?: AgentCallbacks,
  signal?: AbortSignal
): Promise<string> {
  // implementation
}
```

**Step 3: Verify types compile**

Run: `pnpm --dir frontend type-check`
Expected: No type errors.

**Step 4: Commit**

```
feat: implement frontend agent tool-calling loop
```

---

### Task 5: Frontend — Tool Implementations

**Files:**
- Create: `frontend/src/plugins/agent/logic/tools/searchApi.ts`
- Create: `frontend/src/plugins/agent/logic/tools/callApi.ts`
- Create: `frontend/src/plugins/agent/logic/tools/navigate.ts`
- Create: `frontend/src/plugins/agent/logic/tools/pageState.ts`
- Create: `frontend/src/plugins/agent/logic/tools/index.ts`

**Step 1: Implement `search_api`**

Port the OpenAPI index from `backend/api/mcp/tool_search.go`. The MCP server embeds an OpenAPI spec and does keyword matching. For the frontend:
- Option A: Embed the same OpenAPI index as a JSON asset, do keyword search client-side
- Option B: Call the MCP server's `search_api` tool via HTTP from the browser

Start with **Option B** if the MCP server is accessible from the browser, otherwise embed the index. The tool returns matching operation IDs with descriptions.

Reference `backend/api/mcp/openapi_index.go` for the index structure and `tool_search.go` for the search logic.

**Step 2: Implement `call_api`**

Execute a Bytebase API endpoint using the existing Connect RPC clients. This tool:
1. Receives `operation_id` (e.g., `"bytebase.v1.ProjectService.ListProjects"`) and optional `body`
2. Maps the operation to the corresponding Connect client method
3. Calls it with the provided body
4. Returns the JSON response

Reference `backend/api/mcp/tool_call.go` for how the MCP server maps operation IDs to API calls. The frontend equivalent uses the Connect clients from `frontend/src/connect/index.ts`.

For v1, a simpler approach: make a direct HTTP POST to the API endpoint URL (which `search_api` returns) using `fetch` with credentials, since all Bytebase APIs are HTTP+JSON via Connect.

**Step 3: Implement `navigate`**

```typescript
import { useRouter } from "vue-router";

export function createNavigateTool(router: ReturnType<typeof useRouter>) {
  return async (args: { path: string }): Promise<string> => {
    await router.push(args.path);
    return JSON.stringify({
      navigated: true,
      currentPath: router.currentRoute.value.fullPath,
    });
  };
}
```

**Step 4: Implement `get_page_state`**

Returns current page context. For v1, start simple:

```typescript
export function createPageStateTool(router: ReturnType<typeof useRouter>) {
  return async (): Promise<string> => {
    const route = router.currentRoute.value;
    return JSON.stringify({
      path: route.fullPath,
      name: route.name,
      params: route.params,
      query: route.query,
      title: document.title,
    });
  };
}
```

Later phases will add Pinia store data per route.

**Step 5: Create tool registry**

Create `frontend/src/plugins/agent/logic/tools/index.ts` that:
- Exports all tool definitions (name, description, JSON schema)
- Exports a `createToolExecutor` function that routes tool calls to the right implementation

```typescript
import type { ToolDefinition, ToolExecutor } from "../types";

export function getToolDefinitions(): ToolDefinition[] {
  return [
    {
      name: "search_api",
      description: "Search for available Bytebase API endpoints by keyword.",
      parametersSchema: {
        type: "object",
        properties: {
          query: { type: "string", description: "Search query" },
        },
        required: ["query"],
      },
    },
    {
      name: "call_api",
      description: "Execute a Bytebase API endpoint.",
      parametersSchema: {
        type: "object",
        properties: {
          operation_id: { type: "string" },
          body: { type: "object" },
        },
        required: ["operation_id"],
      },
    },
    {
      name: "navigate",
      description: "Navigate to a page in Bytebase.",
      parametersSchema: {
        type: "object",
        properties: {
          path: { type: "string", description: "URL path to navigate to" },
        },
        required: ["path"],
      },
    },
    {
      name: "get_page_state",
      description: "Get information about the current page.",
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
  // ... other tools

  return async (name: string, args: Record<string, unknown>): Promise<string> => {
    switch (name) {
      case "navigate": return navigateTool(args as { path: string });
      case "get_page_state": return pageStateTool();
      case "search_api": return searchApiTool(args as { query: string });
      case "call_api": return callApiTool(args as { operation_id: string; body?: object });
      default: return JSON.stringify({ error: `Unknown tool: ${name}` });
    }
  };
}
```

**Step 6: Fix and type-check**

Run: `pnpm --dir frontend fix && pnpm --dir frontend type-check`
Expected: No errors.

**Step 7: Commit**

```
feat: implement agent tools (search_api, call_api, navigate, get_page_state)
```

---

### Task 6: Frontend — System Prompt

**Files:**
- Create: `frontend/src/plugins/agent/logic/prompt.ts`

**Step 1: Create the system prompt builder**

```typescript
export function buildSystemPrompt(pageContext: {
  path: string;
  title: string;
  role?: string;
}): string {
  return `You are Bytebase Assistant, an AI agent embedded in the Bytebase console.
You help DBAs and developers manage databases, write SQL, review changes,
and navigate the platform.

Rules:
- Use search_api + call_api for actions. Prefer API over DOM interaction.
- Use navigate for "show me" / "go to" requests.
- Always confirm destructive actions (drop database, delete project) before executing.
- You can see the current page state. Use it to provide contextual help.

Core concepts:
- Workspace: top-level container. One workspace per deployment.
- Project: groups databases and members. All changes happen within a project.
- Database: belongs to a project, hosted on an instance.
- Instance: a database server (MySQL, PostgreSQL, etc.) in an environment.
- Environment: dev, staging, prod. Controls approval policies.
- Change ticket (Issue): the review workflow for schema/data changes.
  Flow: create → review → approve → roll out.
- SQL Editor: interactive query tool with access control.

Current page: ${pageContext.path}
Page title: ${pageContext.title}${pageContext.role ? `\nYour role: ${pageContext.role}` : ""}`;
}
```

**Step 2: Fix and type-check**

Run: `pnpm --dir frontend fix && pnpm --dir frontend type-check`

**Step 3: Commit**

```
feat: add agent system prompt builder
```

---

### Task 7: Frontend — Pinia Store

**Files:**
- Create: `frontend/src/plugins/agent/store/agent.ts`

**Step 1: Create the agent store**

```typescript
import { defineStore } from "pinia";
import { ref } from "vue";
import type { Message, ToolCall } from "../logic/types";

export const useAgentStore = defineStore("agent", () => {
  // Window state
  const visible = ref(false);
  const position = ref({ x: window.innerWidth - 420, y: window.innerHeight - 520 });
  const size = ref({ width: 400, height: 500 });
  const minimized = ref(false);

  // Conversation state
  const messages = ref<Message[]>([]);
  const loading = ref(false);
  const abortController = ref<AbortController | null>(null);

  function toggle() {
    visible.value = !visible.value;
    if (visible.value) minimized.value = false;
  }

  function minimize() {
    minimized.value = true;
  }

  function restore() {
    minimized.value = false;
  }

  function addMessage(message: Message) {
    messages.value.push(message);
  }

  function clearMessages() {
    messages.value = [];
  }

  function cancel() {
    abortController.value?.abort();
    abortController.value = null;
    loading.value = false;
  }

  // Persist position/size to localStorage
  function saveWindowState() {
    localStorage.setItem(
      "bb-agent-window",
      JSON.stringify({ position: position.value, size: size.value })
    );
  }

  function loadWindowState() {
    const saved = localStorage.getItem("bb-agent-window");
    if (saved) {
      const state = JSON.parse(saved);
      if (state.position) position.value = state.position;
      if (state.size) size.value = state.size;
    }
  }

  return {
    visible, position, size, minimized,
    messages, loading, abortController,
    toggle, minimize, restore,
    addMessage, clearMessages, cancel,
    saveWindowState, loadWindowState,
  };
});
```

**Step 2: Fix and type-check**

Run: `pnpm --dir frontend fix && pnpm --dir frontend type-check`

**Step 3: Commit**

```
feat: add agent Pinia store for conversation and window state
```

---

### Task 8: Frontend — Floating Agent Window

**Files:**
- Create: `frontend/src/plugins/agent/components/AgentWindow.vue`
- Create: `frontend/src/plugins/agent/components/AgentChat.vue`
- Create: `frontend/src/plugins/agent/components/AgentInput.vue`
- Create: `frontend/src/plugins/agent/components/ToolCallCard.vue`
- Create: `frontend/src/plugins/agent/index.ts`

**Step 1: Create `AgentWindow.vue`**

Floating draggable/resizable container using `<Teleport to="body">`.

Key behavior:
- Render only when `agentStore.visible` is true
- Draggable by title bar (mousedown/mousemove/mouseup on the header)
- Resizable via a resize handle in bottom-right corner
- Min size: 300x400, max size: 800x800
- Title bar with: title "Bytebase Assistant", minimize button, close button
- When minimized, show a small floating icon button instead of the full window
- `z-index: 50` (above page content, below Naive UI modals which use z-index 2000+)
- Call `saveWindowState()` on drag/resize end
- Call `loadWindowState()` on mount

Structure:
```vue
<template>
  <Teleport to="body">
    <!-- Minimized: floating icon button -->
    <div v-if="visible && minimized" class="fixed z-50 cursor-pointer" ...>
      <!-- icon button, click to restore -->
    </div>
    <!-- Full window -->
    <div v-if="visible && !minimized" class="fixed z-50 flex flex-col rounded-lg border shadow-xl bg-white" ...>
      <!-- Title bar (drag handle) -->
      <div class="flex items-center justify-between px-3 py-2 border-b cursor-move" @mousedown="startDrag">
        <span>Bytebase Assistant</span>
        <div class="flex gap-x-1">
          <button @click="minimize">−</button>
          <button @click="close">×</button>
        </div>
      </div>
      <!-- Chat area -->
      <AgentChat class="flex-1 overflow-y-auto" />
      <!-- Input -->
      <AgentInput />
      <!-- Resize handle -->
      <div class="absolute bottom-0 right-0 w-4 h-4 cursor-se-resize" @mousedown="startResize" />
    </div>
  </Teleport>
</template>
```

**Step 2: Create `AgentChat.vue`**

Renders the message list from `agentStore.messages`:
- User messages: right-aligned bubbles
- Assistant messages: left-aligned, rendered as markdown (reuse or fork the Markdown component from `plugins/ai/components/ChatView/Markdown/`)
- Tool call messages: render as `<ToolCallCard />`
- Auto-scroll to bottom on new messages

**Step 3: Create `AgentInput.vue`**

Text input with send button:
- `<textarea>` with auto-resize
- Enter to send, Shift+Enter for newline
- Disabled when `agentStore.loading` is true
- On send: add user message to store, call `runAgentLoop()`, add assistant response

This component wires everything together:
1. Gets tools from `getToolDefinitions()`
2. Creates tool executor from `createToolExecutor(router)`
3. Builds system prompt from `buildSystemPrompt()`
4. Calls `runAgentLoop()` with callbacks that update the store

**Step 4: Create `ToolCallCard.vue`**

Compact card showing:
- Tool name (e.g., "search_api")
- Collapsible section with arguments (JSON) and result (JSON)
- Status indicator (pending/done/error)

**Step 5: Create `index.ts`**

```typescript
export { default as AgentWindow } from "./components/AgentWindow.vue";
export { useAgentStore } from "./store/agent";
```

**Step 6: Fix and type-check**

Run: `pnpm --dir frontend fix && pnpm --dir frontend type-check`

**Step 7: Commit**

```
feat: add floating agent window UI with chat, input, and tool call display
```

---

### Task 9: Frontend — Mount Agent Window in App

**Files:**
- Modify: `frontend/src/App.vue` (or `frontend/src/layouts/BodyLayout.vue`)
- Modify: `frontend/src/layouts/DashboardHeader.vue` (or equivalent header component — add toggle button)

**Step 1: Add `<AgentWindow />` to the app**

Since `AgentWindow` uses `<Teleport to="body">`, it can be added anywhere in the component tree. Add it in `App.vue` alongside the existing providers, or in `BodyLayout.vue` (the main dashboard layout).

The component is self-contained — it reads visibility from the Pinia store, so it just needs to be mounted once.

**Step 2: Add toggle button to the header**

Add a button to the dashboard header that calls `agentStore.toggle()`. Use an AI/chat icon. Only show when AI is enabled (check AI settings like `OpenAIButton.vue` does).

Look at how the existing header is structured in the layout components to find the right place.

**Step 3: Fix and type-check**

Run: `pnpm --dir frontend fix && pnpm --dir frontend type-check`

**Step 4: Manual test**

Run: `pnpm --dir frontend dev` (and backend)
1. Click the agent button in the header → floating window appears
2. Drag the window → moves
3. Resize → resizes
4. Type a message and send → agent loop calls backend → response appears
5. Minimize → window collapses to icon
6. Close → window disappears

**Step 5: Commit**

```
feat: mount agent window in app layout with header toggle button
```

---

### Task 10: Smoke Test End-to-End

**Step 1: Start backend**

Run: `PG_URL=postgresql://bbdev@localhost/bbdev go run ./backend/bin/server/main.go --port 8080 --data . --debug`

**Step 2: Start frontend**

Run: `pnpm --dir frontend dev`

**Step 3: Configure AI settings**

In Bytebase settings, enable AI and configure a provider (OpenAI, Claude, or Gemini).

**Step 4: Test the agent**

1. Open any page (e.g., project list)
2. Click the agent button → floating window opens
3. Type: "What projects exist?" → agent should call `search_api` to find ListProjects, then `call_api` to execute it, then return the results
4. Type: "Go to the SQL editor" → agent should call `navigate` with `/sql-editor`
5. Type: "What page am I on?" → agent should call `get_page_state` and describe the current page

**Step 5: Fix any issues found**

**Step 6: Final commit**

```
feat: page agent phase 1 complete — AI agent with API tools and floating window
```

---

## Task Dependency Graph

```
Task 1 (proto) → Task 2 (backend) → Task 3 (frontend client)
                                          ↓
Task 6 (prompt) ──────────────────→ Task 4 (agent loop)
Task 5 (tools) ────────────────────→    ↓
Task 7 (store) ──────────────────→ Task 8 (UI) → Task 9 (mount) → Task 10 (smoke test)
```

Tasks 5, 6, 7 can run in parallel once Task 3 is done. Tasks 1→2→3 are sequential.
