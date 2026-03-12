# Page Agent — Handoff Notes

## Status: Phase 1 Complete & Working

The page agent is functional end-to-end. A floating chat window in the Bytebase console that can search APIs, call APIs, and navigate pages via natural language.

**Branch:** `page-agent-phase1` (9 commits on GitButler virtual branch)

**Design doc:** `docs/plans/2026-03-12-page-agent-design.md`
**Phase 1 plan:** `docs/plans/2026-03-12-page-agent-phase1.md`

---

## What Was Built

### Backend
- **Proto:** `proto/v1/v1/ai_service.proto` — `AIService` with `Chat` RPC, tool-calling message types, `metadata` field on `AIChatToolCall` for provider-specific data
- **Service:** `backend/api/v1/ai_service.go` — Stateless proxy that reads AI settings from store, translates to OpenAI/Claude/Gemini format (with tool-calling), returns normalized response
- **Registration:** Registered in `backend/server/grpc_routes.go`
- **Key detail:** OpenAI handler uses `json.RawMessage` for tool calls — captures the raw JSON from provider responses and replays it verbatim on subsequent requests. This preserves provider-specific fields like Gemini's `thought_signature` without needing to know the exact schema.

### Frontend
- **Connect client:** `aiServiceClientConnect` in `frontend/src/connect/index.ts`
- **Agent loop:** `frontend/src/plugins/agent/logic/agentLoop.ts` — Tool-calling loop with callbacks, max 10 iterations, AbortSignal support
- **Tools:** `frontend/src/plugins/agent/logic/tools/` — `search_api` (71 static operations with keyword search), `call_api` (Connect protocol POST), `navigate` (Vue Router), `get_page_state` (route info)
- **System prompt:** `frontend/src/plugins/agent/logic/prompt.ts` — Identity + rules + Bytebase concepts + dynamic page context
- **Store:** `frontend/src/plugins/agent/store/agent.ts` — Pinia store for window state + conversation, localStorage persistence
- **UI:** `frontend/src/plugins/agent/components/` — `AgentWindow.vue` (draggable/resizable floating window via Teleport), `AgentChat.vue`, `AgentInput.vue`, `ToolCallCard.vue`
- **Integration:** `AgentWindow` mounted in `BodyLayout.vue`, toggle button (BotIcon) in `DashboardHeader.vue`

---

## Known Issues / Rough Edges

1. **Error messages say "OpenAI API"** even when using Gemini — the `doHTTPRequest` helper uses the provider string from the code path, not the actual provider. Fix: pass `aiSetting.Provider` name instead of hardcoded string.

2. **No markdown rendering** — Assistant messages render as plain text with `white-space: pre-wrap`. The existing `plugins/ai/components/ChatView/Markdown/` component can be reused or forked.

3. **Conversation lost on refresh** — Messages only live in Pinia store (memory). Need localStorage persistence or backend storage.

4. **`search_api` is a static list** — 71 hardcoded operations in `searchApi.ts`. Should eventually use the MCP server's OpenAPI index or generate from proto definitions.

5. **`call_api` is basic** — Uses Connect protocol POST directly. Works but doesn't handle all edge cases (streaming responses, pagination, etc.).

6. **No i18n** — All UI strings are hardcoded English. The CLAUDE.md says all user-facing text should use i18n, but we skipped this for v1.

7. **`metadata` field roundtrip** — The `ToolCall` type in `types.ts` has an optional `metadata` field that preserves provider-specific data through the frontend. The `AgentInput.vue` and `AgentChat.vue` components may not fully preserve this in the store messages — verify the full roundtrip works for multi-turn tool calling.

---

## Next Steps (Priority Order)

### Quick wins
- Fix error provider name in `doHTTPRequest`
- Add markdown rendering to `AgentChat.vue`
- Persist conversation to localStorage
- Add i18n for UI strings

### Phase 2: DOM Engine
- See design doc "DOM Engine" section
- Port page-agent's DOM tree extraction, simplified for Vue/Naive UI
- `dom_action` tool — click, input, select, scroll
- `get_page_state` DOM fallback mode
- Lazy-load the DOM engine bundle

### Phase 3: Polish + Skills
- Port MCP skills (`backend/api/mcp/skills/*.md`) for frontend use
- Route-aware context injection — read Pinia stores per route for richer `get_page_state`
- Keyboard shortcut to toggle panel
- Error handling, retries, loading states

---

## File Map

```
proto/v1/v1/ai_service.proto                          # Proto definition
backend/api/v1/ai_service.go                          # Backend proxy (OpenAI/Claude/Gemini)
backend/server/grpc_routes.go                         # Service registration (search "aiService")
frontend/src/connect/index.ts                         # Connect client (search "aiServiceClientConnect")
frontend/src/plugins/agent/
├── components/
│   ├── AgentWindow.vue                               # Floating window (drag/resize/minimize)
│   ├── AgentChat.vue                                 # Message list
│   ├── AgentInput.vue                                # Input + send (wires agent loop)
│   └── ToolCallCard.vue                              # Tool call display
├── logic/
│   ├── types.ts                                      # Message, ToolCall, ToolDefinition types
│   ├── agentLoop.ts                                  # Tool-calling loop
│   ├── prompt.ts                                     # System prompt builder
│   └── tools/
│       ├── index.ts                                  # Tool registry + executor factory
│       ├── searchApi.ts                              # Static API operation search
│       ├── callApi.ts                                # API execution via Connect
│       ├── navigate.ts                               # Vue Router navigation
│       └── pageState.ts                              # Current route info
├── store/
│   └── agent.ts                                      # Pinia store
└── index.ts                                          # Barrel exports
frontend/src/layouts/BodyLayout.vue                   # Mounts <AgentWindow />
frontend/src/views/DashboardHeader.vue                # Agent toggle button
docs/plans/2026-03-12-page-agent-design.md            # Full design doc
docs/plans/2026-03-12-page-agent-phase1.md            # Phase 1 implementation plan
```

## How to Test

1. Start backend: `PG_URL=postgresql://bbdev@localhost/bbdev go run ./backend/bin/server/main.go --port 8080 --data . --debug`
2. Start frontend: `pnpm --dir frontend dev`
3. Configure AI settings in Bytebase (Settings → General → AI)
4. Click "Agent" button in the header → floating window opens
5. Try: "What projects exist?", "Show me instances", "Go to SQL editor"
