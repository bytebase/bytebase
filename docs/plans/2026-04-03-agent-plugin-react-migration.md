# Agent Plugin React Migration — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Migrate the agent plugin from Vue to React, establishing patterns for the broader Vue-to-React migration.

**Architecture:** Copy logic/dom/skills layers unchanged (~3,400 LOC). Rewrite store as Zustand (~886 LOC). Rewrite 4 components in React with shadcn UI. Mount via a thin Vue wrapper in BodyLayout, then delete the Vue version.

**Tech Stack:** React 19, Zustand, react-markdown + remark-gfm, shadcn (Button, Combobox, Textarea, Input, Dialog), Tailwind CSS v4

---

### Task 1: Install Dependencies

**Files:**
- Modify: `frontend/package.json`

**Step 1: Add zustand and react-markdown**

```bash
cd frontend && pnpm add zustand react-markdown
```

`remark-gfm` is already a devDependency.

**Step 2: Verify installation**

```bash
cd frontend && pnpm list zustand react-markdown remark-gfm
```

Expected: All three listed with versions.

**Step 3: Commit**

```bash
git add frontend/package.json frontend/pnpm-lock.yaml
git commit -m "deps: add zustand and react-markdown for agent plugin React migration"
```

---

### Task 2: Copy Logic/DOM/Skills Layers

**Files:**
- Copy: `frontend/src/plugins/agent/logic/` → `frontend/src/react/plugins/agent/logic/`
- Copy: `frontend/src/plugins/agent/dom/` → `frontend/src/react/plugins/agent/dom/`

These layers are framework-agnostic TypeScript. Copy as-is, no modifications.

**Step 1: Copy the files**

```bash
mkdir -p frontend/src/react/plugins/agent
cp -r frontend/src/plugins/agent/logic frontend/src/react/plugins/agent/logic
cp -r frontend/src/plugins/agent/dom frontend/src/react/plugins/agent/dom
```

**Step 2: Fix import paths**

The logic layer imports from `../logic/types` etc. which remain valid. However, `logic/context.ts` imports Pinia stores via `@/store/modules/v1/...` and `logic/tools/navigate.ts` imports `@/router`. These paths resolve the same way since `@/` maps to `frontend/src/` in both Vue and React tsconfig. No changes needed.

Verify no broken imports:

```bash
cd frontend && pnpm type-check 2>&1 | grep 'react/plugins/agent' | head -20
```

If type errors appear from Vue-specific imports in `context.ts`, they will be caught by `tsconfig.react.json` — but since these files import from `@/store` (plain TS/Pinia singletons) and `@/router` (plain TS), they should work. If `context.ts` causes issues because it imports Vue `computed`/`ref`, we'll address in Task 3.

**Step 3: Run existing logic tests to verify nothing broke**

```bash
cd frontend && pnpm vitest run src/plugins/agent/logic/ --reporter=verbose 2>&1 | tail -20
```

Expected: All tests pass (these test the original files, confirming the logic layer works).

**Step 4: Commit**

```bash
git add frontend/src/react/plugins/agent/logic frontend/src/react/plugins/agent/dom
git commit -m "feat(agent): copy framework-agnostic logic and dom layers for React migration"
```

---

### Task 3: Create Zustand Store

**Files:**
- Create: `frontend/src/react/plugins/agent/store/agent.ts`
- Create: `frontend/src/react/plugins/agent/store/agent.test.ts`

This is a port of the Pinia store at `frontend/src/plugins/agent/store/agent.ts` (886 lines).

**Step 1: Write the store test**

Port the test from `frontend/src/plugins/agent/store/agent.test.ts`. Replace `createPinia`/`setActivePinia` with direct Zustand store creation. The test structure stays the same — same assertions, same mock localStorage.

Key changes from the Pinia test:
- Replace `setActivePinia(createPinia()); useAgentStore()` with `useAgentStore.getState()` or create a fresh store per test
- Zustand stores are singletons — use `useAgentStore.setState()` to reset between tests, or use `createStore` from zustand for test isolation
- Replace `await nextTick()` (Vue reactivity flush) — Zustand updates are synchronous, so remove `nextTick()` calls
- The deep watch auto-save (`watch([chats, ...], saveState, { deep: true })`) becomes a Zustand `subscribe` call in the store

Test file skeleton:

```typescript
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { createAgentStore, AGENT_STATE_KEY, AGENT_WINDOW_KEY } from "./agent";

// Same createMockStorage helper as original test

describe("useAgentStore (Zustand)", () => {
  let store: ReturnType<typeof createAgentStore>;

  beforeEach(() => {
    mockStorage = createMockStorage();
    vi.stubGlobal("localStorage", mockStorage);
    store = createAgentStore(); // fresh store per test
  });

  test("creates a default chat when no persisted state exists", () => {
    const state = store.getState();
    expect(state.chats).toHaveLength(1);
    expect(state.currentChatId).toBe(state.chats[0].id);
  });

  // ... port all 20 tests from original
});
```

**Step 2: Run test to verify it fails**

```bash
cd frontend && pnpm vitest run src/react/plugins/agent/store/agent.test.ts --reporter=verbose 2>&1 | tail -20
```

Expected: FAIL — store module doesn't exist yet.

**Step 3: Write the Zustand store**

Port `frontend/src/plugins/agent/store/agent.ts`. Key mapping:

| Pinia pattern | Zustand pattern |
|---------------|----------------|
| `defineStore("agent", () => { ... })` | `create<AgentState>()((set, get) => ({ ... }))` |
| `ref(false)` | Direct property: `visible: false` |
| `computed(() => ...)` | Not stored — use selectors or derive in `get()` |
| `watch([...], saveState, { deep: true })` | `subscribe` with shallow equality + `JSON.stringify` diff |
| `const chat = getChat(chatId); chat.status = "running"` | `set(state => { const chat = state.chats.find(...); return { chats: state.chats.map(c => c.id === chatId ? { ...c, status: "running" } : c) }; })` |

Important: The Pinia store mutates objects directly (e.g., `chat.status = "running"`). Zustand requires immutable updates. Use immer middleware OR manual spread operators.

**Recommendation: Use `zustand/middleware/immer`** — the store has many direct mutations, and immer makes the port nearly 1:1. This avoids rewriting every mutation to use spread operators.

```bash
cd frontend && pnpm add immer
```

Store skeleton:

```typescript
import { create } from "zustand";
import { persist } from "zustand/middleware";
import { immer } from "zustand/middleware/immer";
import { v4 as uuidv4 } from "uuid";

// Import types from copied logic layer
import type { AgentChat, AgentMessage, AgentPendingAsk, ... } from "../logic/types";

// Copy helper functions as-is: createChatRecord, normalizePersistedState, etc.
// These are pure functions and work unchanged.

export const AGENT_STATE_KEY = "bb-agent-state-v2";
export const AGENT_WINDOW_KEY = "bb-agent-window";

interface AgentState {
  // UI state
  visible: boolean;
  minimized: boolean;
  position: { x: number; y: number };
  size: { width: number; height: number };
  sidebarWidth: number;

  // Chat state
  chats: AgentChat[];
  messagesByChatId: Record<string, AgentMessage[]>;
  pendingAskByChatId: Record<string, AgentPendingAsk>;
  currentChatId: string | null;

  // Runtime (not persisted)
  abortControllersByChatId: Record<string, AbortController>;

  // Actions — same signatures as Pinia store
  toggle: () => void;
  minimize: () => void;
  restore: () => void;
  getChat: (chatId?: string | null) => AgentChat | null;
  getMessages: (chatId?: string | null) => AgentMessage[];
  // ... all 36 actions from Pinia store
}

// Export both hook (for React) and vanilla (for non-React callers like Vue shortcut)
export const useAgentStore = create<AgentState>()(
  immer((set, get) => {
    // Load persisted state on creation
    const persisted = loadPersistedState();

    return {
      visible: false,
      minimized: false,
      position: { x: window.innerWidth - 420, y: window.innerHeight - 520 },
      size: { width: 400, height: 500 },
      sidebarWidth: 256,

      chats: persisted.chats,
      messagesByChatId: persisted.messagesByChatId,
      pendingAskByChatId: persisted.pendingAskByChatId,
      currentChatId: persisted.currentChatId,
      abortControllersByChatId: {},

      toggle() {
        set(state => {
          state.visible = !state.visible;
          if (state.visible) state.minimized = false;
        });
      },

      // ... port all actions, using immer's mutable syntax
      // Most actions can be ported nearly verbatim since immer allows mutation
    };
  })
);

// Derived selectors (replace Pinia computed)
export const selectOrderedChats = (state: AgentState) =>
  [...state.chats].sort((a, b) => b.updatedTs - a.updatedTs || b.createdTs - a.createdTs);

export const selectCurrentChat = (state: AgentState) =>
  state.chats.find(c => c.id === state.currentChatId) ?? null;

export const selectMessages = (state: AgentState) =>
  (state.currentChatId ? state.messagesByChatId[state.currentChatId] : undefined) ?? [];

export const selectLoading = (state: AgentState) =>
  selectCurrentChat(state)?.status === "running";

// ... etc.

// Subscribe for auto-persistence (replaces deep watch)
useAgentStore.subscribe((state) => {
  localStorage.setItem(AGENT_STATE_KEY, JSON.stringify({
    currentChatId: state.currentChatId,
    chats: state.chats,
    messagesByChatId: state.messagesByChatId,
    pendingAskByChatId: state.pendingAskByChatId,
  }));
});
```

For test isolation, export a `createAgentStore` factory function that creates a fresh store instance (not a singleton). The singleton `useAgentStore` calls `createAgentStore` internally. Tests use the factory directly.

**Step 4: Run tests**

```bash
cd frontend && pnpm vitest run src/react/plugins/agent/store/agent.test.ts --reporter=verbose 2>&1 | tail -30
```

Expected: All 20 tests pass.

**Step 5: Commit**

```bash
git add frontend/src/react/plugins/agent/store/
git commit -m "feat(agent): add Zustand store for React agent plugin"
```

---

### Task 4: Create ToolCallCard Component

**Files:**
- Create: `frontend/src/react/plugins/agent/components/ToolCallCard.tsx`

Simplest component. Port from `frontend/src/plugins/agent/components/ToolCallCard.vue` (275 lines).

**Step 1: Write the component**

Direct port. Replace:
- `<script setup>` → function component
- `defineProps<{ toolCall, result }>` → `{ toolCall, result }: Props`
- `ref(false)` → `useState(false)`
- `computed(() => ...)` → `useMemo(() => ..., [deps])`
- `v-if` / `v-for` → JSX conditionals / `.map()`
- `$t(...)` → `t(...)` from `useTranslation()`
- All template markup stays nearly identical (Tailwind classes are the same)

No shadcn components needed — this is pure Tailwind + native HTML.

The helper functions (`parseJson`, `formatJson`, `parseAskUserOption`) are pure functions — copy verbatim.

```typescript
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import type { AgentAskUserOption, AgentAskUserResponse, ToolCall } from "../logic/types";

interface ToolCallCardProps {
  toolCall: ToolCall;
  result?: string;
}

// Copy parseJson, formatJson, parseAskUserOption as-is

export function ToolCallCard({ toolCall, result }: ToolCallCardProps) {
  const { t } = useTranslation();
  const [expanded, setExpanded] = useState(false);

  const parsedArguments = useMemo(() => parseJson(toolCall.arguments), [toolCall.arguments]);
  const parsedResult = useMemo(() => parseJson(result ?? ""), [result]);
  const isAskUser = toolCall.name === "ask_user";
  const isDone = toolCall.name === "done";
  // ... derive askPrompt, askKind, etc. same as Vue version

  return (
    <div className="rounded border bg-gray-50 text-xs">
      <div
        className="flex cursor-pointer items-center gap-x-2 px-2 py-1.5"
        onClick={() => setExpanded(!expanded)}
      >
        {/* Same markup as Vue template, using JSX */}
      </div>
      {expanded && (
        <div className="space-y-1 border-t px-2 py-1.5">
          {/* Same conditional sections */}
        </div>
      )}
    </div>
  );
}
```

**Step 2: Type check**

```bash
cd frontend && pnpm type-check 2>&1 | grep -i error | head -10
```

Expected: No errors related to ToolCallCard.

**Step 3: Commit**

```bash
git add frontend/src/react/plugins/agent/components/ToolCallCard.tsx
git commit -m "feat(agent): add ToolCallCard React component"
```

---

### Task 5: Create AgentChat Component

**Files:**
- Create: `frontend/src/react/plugins/agent/components/AgentChat.tsx`

Port from `frontend/src/plugins/agent/components/AgentChat.vue` (198 lines).

**Step 1: Write the component**

Key changes from Vue version:
- Replace `unified().use(remarkParse).use(remarkGfm)` + custom `AstToMarkdown` with `react-markdown` + `remarkGfm` plugin — this eliminates the entire markdown pipeline
- Replace `NButton` with shadcn `Button`
- Replace `watch([currentChatId, messages.length], ...)` with `useEffect` + `scrollIntoView`
- Replace `useRouter().push()` with direct Vue router import (singleton) during coexistence

```typescript
import { useEffect, useMemo, useRef } from "react";
import { useTranslation } from "react-i18next";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { Button } from "@/react/components/ui/button";
import router from "@/router"; // Vue router singleton
import { SETTING_ROUTE_WORKSPACE_GENERAL } from "@/router/dashboard/workspaceSetting";
import { hasWorkspacePermissionV2 } from "@/utils";
import type { AgentMessage } from "../logic/types";
import {
  useAgentStore,
  selectCurrentChat,
  selectMessages,
  selectLoading,
} from "../store/agent";
import { ToolCallCard } from "./ToolCallCard";

export function AgentChat({ className }: { className?: string }) {
  const { t } = useTranslation();
  const chatContainerRef = useRef<HTMLDivElement>(null);

  // Zustand selectors
  const messages = useAgentStore(selectMessages);
  const loading = useAgentStore(selectLoading);
  const currentChatId = useAgentStore(s => s.currentChatId);
  const currentChat = useAgentStore(selectCurrentChat);
  const clearError = useAgentStore(s => s.clearError);

  const requiresAIConfiguration = currentChat?.requiresAIConfiguration ?? false;
  const error = currentChat?.lastError ?? null;
  const allowConfigure = hasWorkspacePermissionV2("bb.settings.set");

  const displayMessages = useMemo(
    () => messages.filter((m): m is AgentMessage => m.role === "user" || m.role === "assistant"),
    [messages]
  );

  // Auto-scroll
  useEffect(() => {
    if (chatContainerRef.current) {
      chatContainerRef.current.scrollTop = chatContainerRef.current.scrollHeight;
    }
  }, [currentChatId, messages.length]);

  function getToolResult(messageId: string, toolCallId: string): string | undefined {
    // Same logic as Vue version, reading from messages array
    const fullIndex = messages.findIndex(m => m.id === messageId);
    if (fullIndex < 0) return undefined;
    for (let i = fullIndex + 1; i < messages.length; i++) {
      const m = messages[i];
      if (m.role === "tool" && m.toolCallId === toolCallId) return m.content;
      if (m.role === "assistant" && m.content && !m.toolCalls?.length) break;
    }
    return undefined;
  }

  function goConfigure() {
    clearError(currentChatId);
    router.push({ name: SETTING_ROUTE_WORKSPACE_GENERAL, hash: "#ai-assistant" });
  }

  return (
    <div ref={chatContainerRef} className={cn("overflow-y-auto space-y-3 p-3", className)}>
      {displayMessages.map(msg => (
        msg.role === "user" ? (
          <div key={msg.id} className="flex justify-end">
            <div className="max-w-[80%] rounded-lg bg-blue-50 px-3 py-2 text-sm">{msg.content}</div>
          </div>
        ) : (
          <div key={msg.id} className="flex flex-col gap-y-2">
            {msg.content && (
              <div className="max-w-[80%] rounded-lg bg-gray-50 px-3 py-2 text-sm markdown-content">
                <ReactMarkdown remarkPlugins={[remarkGfm]}>{msg.content}</ReactMarkdown>
              </div>
            )}
            {msg.toolCalls?.map(tc => (
              <ToolCallCard key={tc.id} toolCall={tc} result={getToolResult(msg.id, tc.id)} />
            ))}
          </div>
        )
      ))}
      {loading && (
        <div className="flex items-center gap-x-2 text-sm text-gray-400">
          <span className="animate-pulse">&#9679;</span> {t("common.loading")}
        </div>
      )}
      {/* AI configuration recovery and error display — same markup */}
    </div>
  );
}
```

**Markdown CSS:** The Vue version uses scoped `:deep()` selectors for markdown styling. In React, use a CSS module or add global styles. Since the project uses Tailwind, use `@apply` in a CSS file or use the `components` prop on `ReactMarkdown` for inline styling.

Simplest approach: Add a `markdown-content` class to `frontend/src/assets/css/tailwind.css` with the same `@apply` rules, or use ReactMarkdown's `components` prop to apply Tailwind classes to each element type.

**Step 2: Type check**

```bash
cd frontend && pnpm type-check 2>&1 | grep -i error | head -10
```

**Step 3: Commit**

```bash
git add frontend/src/react/plugins/agent/components/AgentChat.tsx
git commit -m "feat(agent): add AgentChat React component with react-markdown"
```

---

### Task 6: Create AgentInput Component

**Files:**
- Create: `frontend/src/react/plugins/agent/components/AgentInput.tsx`

Port from `frontend/src/plugins/agent/components/AgentInput.vue` (685 lines). Most complex UI piece.

**Step 1: Write the component**

Key changes:
- Replace `NMention` with a custom `@`-triggered popover using the existing shadcn `Combobox` pattern (not the full `Combobox` component — we need inline mention behavior)
- Replace `NButton` with shadcn `Button`
- Replace Vue `watch()` with `useEffect()`
- The `runChat`, `send`, `retryLastTurn`, `submitConfirmation`, `submitChoice` functions are pure logic — port nearly verbatim

**@-mention implementation approach:**

The shadcn `Combobox` component (at `frontend/src/react/components/ui/combobox.tsx`) is a standalone dropdown selector — it doesn't support inline textarea mentions. Build a lightweight mention popover:

1. Use a `<textarea>` for input
2. Track cursor position with `onSelect` / `onKeyUp`
3. When user types `@`, compute popover position from cursor coordinates
4. Show a filtered list of DOM ref suggestions in an absolutely-positioned div
5. On selection, replace `@query` with `[eN]` in the textarea

This is ~80 lines of custom code. The filtering logic and suggestion extraction are copied from the Vue version.

```typescript
// Simplified mention hook
function useMention(textareaRef: RefObject<HTMLTextAreaElement>, input: string) {
  const [mentionQuery, setMentionQuery] = useState<{ query: string; start: number; end: number } | null>(null);
  const [suggestions, setSuggestions] = useState<DomRefSuggestion[]>([]);
  const [menuOpen, setMenuOpen] = useState(false);

  // getDomRefQuery logic — same as Vue version
  const updateMention = useCallback(() => {
    const el = textareaRef.current;
    if (!el) return;
    const query = getDomRefQuery(input, el.selectionStart, el.selectionEnd);
    setMentionQuery(query);
  }, [input, textareaRef]);

  // Load suggestions when @ is detected
  useEffect(() => {
    if (!mentionQuery) { setSuggestions([]); return; }
    if (suggestions.length > 0) return;
    lazyExtractDomRefSuggestions().then(setSuggestions);
  }, [mentionQuery]);

  const filteredSuggestions = useMemo(() => {
    if (!mentionQuery) return [];
    return suggestions.filter(s => matchDomRefSuggestion(s, mentionQuery.query));
  }, [suggestions, mentionQuery]);

  return { mentionQuery, filteredSuggestions, menuOpen, setMenuOpen, updateMention };
}
```

**Step 2: Type check**

```bash
cd frontend && pnpm type-check 2>&1 | grep -i error | head -10
```

**Step 3: Commit**

```bash
git add frontend/src/react/plugins/agent/components/AgentInput.tsx
git commit -m "feat(agent): add AgentInput React component with @-mention"
```

---

### Task 7: Create AgentWindow Component

**Files:**
- Create: `frontend/src/react/plugins/agent/components/AgentWindow.tsx`

Port from `frontend/src/plugins/agent/components/AgentWindow.vue` (798 lines).

**Step 1: Write the component**

Key changes:
- `<Teleport to="body">` → `createPortal(..., document.body)`
- `ref<HTMLElement>` → `useRef<HTMLDivElement>`
- `onMounted` / `onBeforeUnmount` → `useEffect` cleanup
- `watch(windowRef, ...)` → `useEffect` with ref dependency
- `computed(...)` → `useMemo(...)` or inline derivation
- `NInput` (for rename) → shadcn `Input`
- `NPopconfirm` → shadcn `Dialog` (AlertDialog pattern)
- `lucide-vue-next` → `lucide-react` (same icon names: `Archive`, `Inbox`)
- `HumanizeTs` Vue component → inline `useRelativeTime` hook or direct `Intl.RelativeTimeFormat`

Drag/resize logic ports directly — the pointer event handlers are pure DOM code:

```typescript
const startDrag = useCallback((event: React.MouseEvent) => {
  // Same guard: skip if target has data-agent-window-action/resize
  if ((event.target as HTMLElement).closest("[data-agent-window-action], [data-agent-window-resize]")) return;

  const store = useAgentStore.getState();
  const offsetX = event.clientX - store.position.x;
  const offsetY = event.clientY - store.position.y;

  const onDrag = (e: MouseEvent) => {
    useAgentStore.getState().setPosition(e.clientX - offsetX, e.clientY - offsetY);
  };
  const stopDrag = () => {
    document.removeEventListener("mousemove", onDrag);
    document.removeEventListener("mouseup", stopDrag);
    useAgentStore.getState().saveWindowState();
  };
  document.addEventListener("mousemove", onDrag);
  document.addEventListener("mouseup", stopDrag);
}, []);
```

The resize and sidebar resize follow the same pattern.

**Viewport resize handler:**

```typescript
useEffect(() => {
  const handleResize = () => {
    setViewportSize({ width: window.innerWidth, height: window.innerHeight });
  };
  window.addEventListener("resize", handleResize);
  return () => window.removeEventListener("resize", handleResize);
}, []);
```

**ResizeObserver:**

```typescript
useEffect(() => {
  const el = windowRef.current;
  if (!el) return;
  const observer = new ResizeObserver(([entry]) => {
    if (!entry || isResizingRef.current) return;
    syncSize(entry.target as HTMLElement);
  });
  observer.observe(el);
  return () => observer.disconnect();
}, []);
```

**Step 2: Type check**

```bash
cd frontend && pnpm type-check 2>&1 | grep -i error | head -10
```

**Step 3: Commit**

```bash
git add frontend/src/react/plugins/agent/components/AgentWindow.tsx
git commit -m "feat(agent): add AgentWindow React component with drag/resize"
```

---

### Task 8: Create Entry Point and Mount Bridge

**Files:**
- Create: `frontend/src/react/plugins/agent/index.ts`
- Create: `frontend/src/react/plugins/agent/AgentWindowMount.vue`
- Modify: `frontend/src/layouts/BodyLayout.vue:77,92,194`
- Modify: `frontend/src/react/mount.ts:3-5,39`

**Step 1: Create the React entry point**

```typescript
// frontend/src/react/plugins/agent/index.ts
export { AgentWindow } from "./components/AgentWindow";
export { useAgentStore } from "./store/agent";
```

**Step 2: Register the AgentWindow in the React mount system**

Add a new glob for plugin components in `frontend/src/react/mount.ts`:

```typescript
const pluginLoaders = import.meta.glob("./plugins/agent/components/AgentWindow.tsx");
```

Or, simpler: create a dedicated mount file for the agent since it's not a page — it's a floating widget.

**Step 3: Create the Vue mount wrapper**

`frontend/src/react/plugins/agent/AgentWindowMount.vue`:

```vue
<template>
  <div ref="container" />
</template>

<script lang="ts" setup>
import { onMounted, onUnmounted, ref, watch } from "vue";
import { useI18n } from "vue-i18n";

const { locale } = useI18n();
const container = ref<HTMLElement>();
let root: any = null;

async function render() {
  if (!container.value) return;
  const [
    { createElement, StrictMode, createRoot },
    { I18nextProvider },
    i18nModule,
    { AgentWindow },
  ] = await Promise.all([
    import("react").then(m => ({ createElement: m.createElement, StrictMode: m.StrictMode, createRoot: null })),
    import("react-i18next"),
    import("@/react/i18n"),
    import("@/react/plugins/agent"),
  ]);
  // Actually use the pattern from mount.ts
  const { mountReactComponent } = await import("./mountAgent");
  if (!root) {
    root = await mountReactComponent(container.value);
  }
}

// Simpler approach: follow exact ReactPageMount.vue pattern
// but load AgentWindow directly instead of by page name
async function mount() {
  if (!container.value) return;
  const [react, reactDom, { I18nextProvider }, i18nModule, { AgentWindow }] = await Promise.all([
    import("react"),
    import("react-dom/client"),
    import("react-i18next"),
    import("@/react/i18n"),
    import("@/react/plugins/agent"),
  ]);
  await i18nModule.i18nReady;
  root = reactDom.createRoot(container.value);
  root.render(
    react.createElement(
      react.StrictMode,
      null,
      react.createElement(
        I18nextProvider,
        { i18n: i18nModule.default },
        react.createElement(AgentWindow)
      )
    )
  );
}

onMounted(() => mount());
watch(locale, async () => {
  const i18nModule = await import("@/react/i18n");
  await i18nModule.default.changeLanguage(locale.value);
});
onUnmounted(() => {
  root?.unmount();
  root = null;
});
</script>
```

**Step 4: Create keyboard shortcut that calls Zustand store**

The keyboard shortcut in `BodyLayout.vue` currently calls `useAgentShortcut()` which is a Vue composable. Replace with a direct Zustand call:

In `BodyLayout.vue`, change:
```typescript
// Before:
import { AgentWindow, useAgentShortcut } from "@/plugins/agent";
useAgentShortcut();

// After:
import AgentWindowMount from "@/react/plugins/agent/AgentWindowMount.vue";
// Keyboard shortcut — Zustand store is callable from anywhere
import { onMounted, onUnmounted } from "vue";
function setupAgentShortcut() {
  const handler = async (e: KeyboardEvent) => {
    if ((e.ctrlKey || e.metaKey) && e.shiftKey && e.key === "A") {
      e.preventDefault();
      const { useAgentStore } = await import("@/react/plugins/agent");
      useAgentStore.getState().toggle();
    }
  };
  onMounted(() => window.addEventListener("keydown", handler));
  onUnmounted(() => window.removeEventListener("keydown", handler));
}
setupAgentShortcut();
```

In the template, replace `<AgentWindow />` with `<AgentWindowMount />`.

**Step 5: Verify it renders**

```bash
cd frontend && pnpm dev
```

Open browser, press Ctrl+Shift+A — agent window should appear.

**Step 6: Commit**

```bash
git add frontend/src/react/plugins/agent/index.ts frontend/src/react/plugins/agent/AgentWindowMount.vue frontend/src/layouts/BodyLayout.vue
git commit -m "feat(agent): mount React agent window in BodyLayout"
```

---

### Task 9: Port Tests

**Files:**
- Create: `frontend/src/react/plugins/agent/components/AgentWindow.test.ts`
- Create: `frontend/src/react/plugins/agent/components/AgentInput.test.ts`

Port the 3 main test files from the Vue version. The store tests were already written in Task 3.

**Step 1: Port AgentWindow tests**

Replace Vue Test Utils (`mount`, `wrapper.find()`) with `@testing-library/react` (`render`, `screen.getByText()`). The 11 test cases cover sidebar rendering, chat switching, rename, archive — all of which have equivalent RTL patterns.

**Step 2: Port AgentInput tests**

Replace NMention/NButton interactions with native DOM events. The 27 test cases are mostly about state transitions (send, retry, confirm, choose) which are store-level — they should be similar to the Vue tests.

**Step 3: Run all tests**

```bash
cd frontend && pnpm vitest run src/react/plugins/agent/ --reporter=verbose 2>&1 | tail -30
```

Expected: All tests pass.

**Step 4: Commit**

```bash
git add frontend/src/react/plugins/agent/components/AgentWindow.test.ts frontend/src/react/plugins/agent/components/AgentInput.test.ts
git commit -m "test(agent): port component tests to React Testing Library"
```

---

### Task 10: Lint, Format, and Final Verification

**Step 1: Fix lint/format**

```bash
pnpm --dir frontend fix
```

**Step 2: Type check**

```bash
pnpm --dir frontend type-check
```

**Step 3: Run all agent tests**

```bash
cd frontend && pnpm vitest run src/react/plugins/agent/ --reporter=verbose
```

**Step 4: Run full frontend tests**

```bash
pnpm --dir frontend test
```

**Step 5: Manual smoke test**

1. Open browser with dev server
2. Press Ctrl+Shift+A — agent window appears
3. Create a chat, type a message, verify it sends
4. Verify tool calls render in ToolCallCard
5. Test drag/resize/minimize
6. Test @-mention shows DOM refs
7. Refresh page — verify chat history persists
8. Verify sidebar: rename, archive, delete chats

**Step 6: Commit any lint fixes**

```bash
git add -u
git commit -m "fix(agent): lint and format fixes"
```

---

### Task 11: Delete Vue Agent Plugin

**Files:**
- Delete: `frontend/src/plugins/agent/` (entire directory)
- Modify: Any remaining imports of `@/plugins/agent` elsewhere

**Step 1: Search for remaining imports**

```bash
grep -r '@/plugins/agent' frontend/src/ --include='*.ts' --include='*.vue' --include='*.tsx' | grep -v 'node_modules'
```

The only import should be `BodyLayout.vue` which was already updated in Task 8. If there are others, update them.

**Step 2: Delete the Vue version**

```bash
rm -rf frontend/src/plugins/agent/
```

**Step 3: Verify build**

```bash
pnpm --dir frontend type-check
pnpm --dir frontend test
```

**Step 4: Commit**

```bash
git add -A
git commit -m "refactor(agent): remove Vue agent plugin after React migration"
```

---

## Appendix: shadcn Components Needed

Components already available in `frontend/src/react/components/ui/`:
- `Button` — replaces `NButton`
- `Input` — replaces `NInput` (for rename)
- `Textarea` — for chat input
- `Dialog` — replaces `NPopconfirm` (archive/delete confirmations)
- `Combobox` — reference for @-mention pattern (not used directly)

Components NOT needed (simpler alternatives):
- `Collapsible` — ToolCallCard uses a simple `useState(false)` toggle, no need for a component
- `ScrollArea` — native `overflow-y-auto` is sufficient
