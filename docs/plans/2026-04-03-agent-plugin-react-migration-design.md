# Agent Plugin React Migration Design

## Goal

Migrate the agent plugin (`frontend/src/plugins/agent/`) from Vue to React. This is a pattern-setting migration — decisions here establish conventions for future Vue-to-React work.

## Current State

- ~7,000 LOC across 22+ files
- Floating AI chat window with tool execution, DOM interaction, multi-chat persistence
- ~50% of code (logic/, dom/, skills/) is framework-agnostic TypeScript

## Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| State management | Zustand | Lightweight, similar to Pinia, minimal boilerplate |
| Floating window | Custom pointer events | Vue version already has the logic (~150 lines), no new dep |
| @-mention autocomplete | shadcn Combobox | Consistent with component library, handles filtered list + popover |
| Markdown rendering | react-markdown + remark-gfm | Standard React solution, replaces custom AST-to-VNode pipeline |
| Table library | None (shadcn Table if needed) | Server-side sort/pagination via Connect RPC, tanstack adds little |

## Architecture

```
frontend/src/react/plugins/agent/
├── index.ts                          # Export AgentWindow + useAgentShortcut
├── store/
│   └── agent.ts                      # Zustand store (replaces Pinia store)
├── components/
│   ├── AgentWindow.tsx               # Floating panel: drag, resize, minimize, portal
│   ├── AgentChat.tsx                 # Message list + react-markdown
│   ├── AgentInput.tsx                # Input with @-mention via shadcn Combobox
│   └── ToolCallCard.tsx              # Collapsible tool call/result display
├── logic/                            # COPIED AS-IS from Vue version
│   ├── types.ts
│   ├── prompt.ts
│   ├── context.ts                    # Reads Pinia stores directly (singletons)
│   ├── agentLoop.ts
│   ├── outboundHistory.ts
│   ├── aiConfiguration.ts
│   ├── tools/                        # All tool files copied; navigate.ts imports Vue router singleton
│   └── skills/                       # All skill files copied unchanged
└── dom/                              # COPIED AS-IS (pure DOM APIs)
    ├── index.ts
    ├── domTree.ts
    └── actions.ts
```

## Store Design (Zustand)

Three concerns, same as current Pinia store:

**UI state** — `visible`, `minimized`, `position`, `size`, `sidebarWidth`
- Persisted to localStorage key `bb-agent-window`

**Chat state** — `chats`, `messagesByChatId`, `pendingAskByChatId`, `currentChatId`
- Persisted to localStorage key `bb-agent-state-v2`
- Normalized structure (chats and messages stored separately)

**Runtime state** — `abortControllersByChatId`, derived `loading`/`error`/`runningChatIds`
- Not persisted

Zustand `persist` middleware handles the two localStorage keys. Derived values become inline selectors or helper functions.

## Component Design

### AgentWindow.tsx
- `createPortal(document.body)` replaces Vue `<Teleport>`
- Drag: `onPointerDown` on header, track delta in `useRef`, update store on `pointermove`
- Resize: Same pattern on SE corner handle
- Sidebar resize: Same pattern on vertical divider
- Viewport clamping: `useEffect` on window `resize` event
- Minimized: Floating button (bottom-right) instead of full panel
- Layout: flex row — sidebar (chat list) + main area

### AgentChat.tsx
- `react-markdown` with `remarkGfm` plugin
- Auto-scroll via `useEffect` + `scrollIntoView` on message count change
- shadcn `Button` for retry/interrupt, `ScrollArea` for container

### AgentInput.tsx
- shadcn `Textarea` for message input
- On `@` keystroke, open shadcn `Combobox` popover with DOM ref suggestions
- Selecting inserts `[eN]` ref into textarea
- Pending ask states render different UIs: text input, Yes/No buttons, radio choices
- Enter to send, Shift+Enter for newline

### ToolCallCard.tsx
- shadcn `Collapsible`: header = tool name + status, body = pretty-printed JSON
- Status indicators: spinner while running, check/x on complete

## Integration & Coexistence

| Bridge Point | Approach |
|-------------|----------|
| Mount in Vue app | `AgentWindowMount.vue` — thin wrapper that mounts React root, placed in `BodyLayout.vue` |
| Pinia stores in `context.ts` | Import directly — they're singletons, callable outside Vue |
| Vue Router in `navigate.ts` | Import router instance directly — singleton |
| i18n | `react-i18next` with same locale JSON files |
| Zustand from Vue | `useAgentStore.getState().toggle()` for keyboard shortcut |

Clean swap: delete Vue version once React version is mounted. No gradual coexistence needed — no other Vue components depend on the agent plugin.

## Dependencies to Add

- `zustand`
- `react-markdown` + `remark-gfm`

## Migration Order

1. **Store** — Zustand store, unit tests, verify localStorage backward compat
2. **ToolCallCard** — Simplest component, validates React + shadcn pipeline
3. **AgentChat** — Message list with react-markdown
4. **AgentInput** — Combobox @-mention (most complex UI)
5. **AgentWindow** — Shell with drag/resize/portal
6. **Integration** — AgentWindowMount.vue, swap in BodyLayout, delete Vue version

## What We're NOT Changing

- No backend changes
- No changes to agent loop, tools, skills, or DOM interaction layer
- No changes to localStorage schema (existing chat history carries over)
- No UX redesign — functional port of current behavior
