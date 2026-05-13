# SQL Editor React Migration — Stage 22 Design

**Date:** 2026-05-12
**Author:** d@bytebase.com
**Status:** Draft

## 1. Goal & non-goals

**Goal:** Port the AI plugin's Vue component tree under
`frontend/src/plugins/ai/components/` to React, swap the two
`<VueMount component={AIChatToSQLBridgeHost}>` call sites in the SQL
Editor's host shells to render the React `<AIChatToSQL>` directly, and
delete both the Vue tree and the reverse-bridge primitive
(`frontend/src/react/components/VueMount.tsx`) — that primitive's only
consumer is the AI plugin. After this stage the SQL Editor surface and
everything mounted inside it is 100% React.

**Non-goals:**
- The framework-agnostic TS layers under `plugins/ai/{logic,store,types}/`
  are already shared by React callers (`OpenAIButton.tsx`,
  `StandardPanel/SQLEditor.tsx`, etc.). They stay. Only behavioural fixes
  needed to bridge Vue-Ref-shaped APIs (`useChatByTab` returning
  `ComputedRef<AIChatInfo>`) to React-friendly shapes are in scope. New
  parallel React helpers may be added alongside; the Vue helpers can be
  retired once nothing imports them.
- The AI feature itself (prompt format, conversation/store schema,
  AICompletion RPC, persistence). Parity port only.
- The reverse-bridge file `VueMount.tsx` is deleted once both mount
  points swap, but **`ReactPageMount`** (the *forward* bridge that lets
  Vue host React) stays — it's still used for `<ReactPageMount
  page="SQLEditorLayout">` from `router/sqlEditor.ts`.
- New AI-side features (e.g. richer markdown, attachments, streaming
  rendering) — explicitly out of scope.
- Test coverage for AI flows beyond what the migration directly
  benefits from. The Vue tree has zero existing tests; we add tests
  only for new React seams whose contract is non-obvious (see §5).

## 2. Inventory

**In scope (~1,256 LOC of `plugins/ai/components/`):**

| File | LOC | Notes |
|---|---|---|
| `components/ChatPanel.vue` | 187 | Root visible shell. Hosts `ActionBar`, `ChatView`, `DynamicSuggestions`, `PromptInput`, `HistoryPanel`. Owns the `requestAI(query)` orchestrator (createMessage USER → createMessage AI LOADING → `sqlServiceClientConnect.aICompletion` → update). |
| `components/ProvideAIContext.vue` | 124 | Sets up the `AIContext` via Vue `provide`. Subscribes to two events on the shared `aiContextEvents`: `new-conversation` (reuse-if-empty / create-new) and `send-chat`. Fetches the AI setting on mount. |
| `components/ChatView/Markdown/CodeBlock.vue` | 129 | SQL `<MonacoEditor>` snippet card with Run / Insert-at-Caret / Copy actions. Reads `useElementSize` of nearest `.message` ancestor for adaptive width. Emits `run-statement` / `insert-at-caret`. |
| `components/HistoryPanel/ConversationList.vue` | 129 | Listing of past conversations: select, rename (via `ConversationRenameDialog`), delete (with `NPopconfirm`). |
| `components/PromptInput.vue` | 116 | `NInput type="textarea"` autosize 1-10 rows. Emits `enter` on plain Enter (Shift+Enter inserts newline). Focuses on mount + on `new-conversation`. Reacts to `pendingPreInput` (one-shot input prefill). |
| `components/DynamicSuggestions.vue` | 102 | Calls `useDynamicSuggestions()` which streams 3 prompt suggestions. Click → `@enter`. |
| `components/ChatView/ChatView.vue` | 92 | Auto-scroll-to-bottom message list. `mode: "CHAT" \| "VIEW"`. Provides `ChatViewContext` so child views can read `mode`. |
| `components/HistoryPanel/ConversationRenameDialog.vue` | 88 | Rename modal with a single `NInput`. |
| `components/ActionBar.vue` | 58 | Top bar: New chat / History / Close buttons. Emits `new-conversation`. |
| `components/ChatView/AIMessageView.vue` | 42 | AI bubble: spinner while LOADING, error pill on FAILED, otherwise `<Markdown>` of `message.content` with `codeBlockProps={ width: 0.85 }`. |
| `components/ChatView/Markdown/Markdown.vue` | 38 | `unified().use(remarkParse).use(remarkGfm)` parses content, hands the AST to `AstToVNode` with custom `code` / `inlineCode` / `image` slots. |
| `components/HistoryPanel/HistoryPanel.vue` | 29 | Drawer wrapping `ConversationList` + `ChatView mode="VIEW"`. |
| `components/ChatView/Markdown/InsertAtCaretIcon.vue` | 28 | Inline SVG icon (parametrized by `size`). |
| `components/ChatView/Markdown/AstToVNode.vue` | 24 | Receives mdast root, walks via `mdastToVNode[type]` (in `utils.ts`), uses Vue slots for custom rendering of `code` / `inlineCode` / `image`. |
| `components/AIChatToSQLBridgeHost.vue` | 23 | Bridge entry: `<ProvideAIContext><Suspense><AIChatToSQL/></Suspense></ProvideAIContext>`. **Deleted in Phase 4** — React host renders `<AIChatToSQL />` directly. |
| `components/ChatView/UserMessageView.vue` | 21 | User bubble (text). |
| `components/ChatView/EmptyView.vue` | 16 | Empty-state placeholder shown when a `VIEW`-mode conversation has no messages. |
| `components/AIChatToSQL.vue` | 10 | Top-level async wrapper: `await import("./ChatPanel.vue")`, then renders `<ChatPanel v-if="aiSetting.enabled">`. |

**Already TS (read by React; no migration needed):**
- `plugins/ai/logic/{context,events,prompt,useChatByTab,useDynamicSuggestions,utils,index}.ts`
- `plugins/ai/store/{conversation,index}.ts`
- `plugins/ai/types/{context,conversation,index}.ts`
- `plugins/ai/components/{state,editor-actions}.ts`
- `plugins/ai/components/ChatView/{context,types,index}.ts`
- `plugins/ai/components/ChatView/Markdown/{utils,index}.ts` — utils stays but loses the `h(...)` Vue imports (see §3).
- `plugins/ai/components/HistoryPanel/index.ts`

**External mount points (only two):**
- `frontend/src/react/components/sql-editor/StandardPanel/StandardPanel.tsx:107` — `<VueMount component={AIChatToSQLBridgeHost} />`
- `frontend/src/react/components/sql-editor/Panels/Panels.tsx:233` — same

Both swap to `<AIChatToSQL />` in Phase 4.

**Files that import the AI plugin from React (post-migration these
keep importing from `plugins/ai/` — TS layers only):**
- `react/components/sql-editor/OpenAIButton.tsx`
- `react/components/sql-editor/StandardPanel/SQLEditor.tsx`
- `react/components/sql-editor/Panels/common/CodeViewer.tsx`

## 3. External deps & primitives

| Vue dep | React replacement | Status |
|---|---|---|
| `NInput type="textarea"` autosize (PromptInput) | `react-textarea-autosize` (already a Stage-17 dep for the Monaco-adjacent prompt boxes) or a hand-rolled `<textarea>` with rows-by-content. Probably the autosize lib for parity. | Existing dep — verify |
| `NPopover` (PromptInput suffix shortcut hint, CodeBlock action tooltips, ConversationList delete confirm) | Shared `Tooltip` from `@/react/components/ui/tooltip`. | ✓ |
| `NPopconfirm` (ConversationList delete) | shadcn `AlertDialog` from `@/react/components/ui/alert-dialog`. | ✓ |
| `NSpin` (ChatPanel loading, AIMessageView LOADING) | `Loader2` from `lucide-react` with `animate-spin`, matches Stage 17/18 pattern. | ✓ |
| `Drawer` / `DrawerContent` (HistoryPanel) | shadcn `Sheet` from `@/react/components/ui/sheet` with `width="wide"` (the drawer is `w-[calc(100vw-8rem)] max-w-6xl`; ~the wide tier). | ✓ |
| `Vue's provide / inject` (`provideAIContext`) | React Context. New file: `plugins/ai/react/context.tsx` exposing `<AIContextProvider>` + `useAIContext()`. The provider reads Pinia + module-level singletons via `useVueState` and Memoizes the `AIContext`-shaped value. | New |
| `useElementSize` from `@vueuse/core` (CodeBlock, ChatView auto-scroll) | `ResizeObserver` in a `useResizeObserver` hook — already a pattern used by Stage 20's `useTableResize` and Stage 22's Navigator (`SchemaDiagram`). | Local hook |
| `Vue h()` in `Markdown/utils.ts` | `React.createElement` (`createElement` / `cloneElement`). Same mdast-walk logic; replace `h(tag, props, children)` with `createElement(tag, props, children)` and `class` → `className`, plus React-style props (`onClick`, `aria-…`). | Rewrite |
| `useEmitteryEventListener` composable | Plain `useEffect(() => { const off = events.on(name, cb); return off; }, deps)`. Pattern already in use across Stages 16-20. | ✓ |
| `MonacoEditor` (Vue component, used by CodeBlock) | The React `MonacoEditor` shipped in Stage 14. Import path: `@/react/components/MonacoEditor`. | ✓ |
| `CopyButton` from `@/components/v2` (CodeBlock copy action) | A small React copy icon button (lucide `Copy` + `navigator.clipboard.writeText`, with a 1-second "Copied" state). Either import the existing React `CopyButton` if one exists in `react/components/`, or inline it — it's ~20 LOC. | Verify / inline |
| `HighlightCodeBlock` (Vue, used by Markdown for inline code) | React equivalent in `react/components/` if it exists, otherwise a thin styled `<code>` with the project's Shiki tokens. Inline code is short enough that a plain styled `<code>` is fine for parity. | Verify / inline |
| `i18n-t` with `<template #create>` slot (ChatView's "select-or-create" prompt) | `Trans` component manual split (mirrors the SelectionCopyTooltips fix from Stage 20 — `Trans` v17 wipes slot children on empty placeholder tags). | Pattern already established |
| `Suspense` around `<AIChatToSQL>` (BridgeHost) | `React.Suspense` around `React.lazy(() => import('./ChatPanel'))` inside `AIChatToSQL.tsx`. Mirrors the Vue `await import('./ChatPanel.vue')` pattern. | ✓ |

## 4. Architecture & phases

Single PR, four internal phases. Each phase is internally consistent
(builds + lints cleanly) so review can pick up at phase boundaries.

```
plugins/ai/
├── components/                  ← deleted at end
│   └── *.vue
└── react/                       ← NEW (the React tree)
    ├── context.tsx              ← AIContextProvider + useAIContext
    ├── AIChatToSQL.tsx          ← lazy + Suspense wrapper, gated on aiSetting.enabled
    ├── ChatPanel.tsx            ← orchestrator (requestAI), composes panels
    ├── ActionBar.tsx
    ├── PromptInput.tsx
    ├── DynamicSuggestions.tsx
    ├── ChatView/
    │   ├── ChatView.tsx
    │   ├── AIMessageView.tsx
    │   ├── UserMessageView.tsx
    │   ├── EmptyView.tsx
    │   ├── context.tsx          ← Mode context (CHAT | VIEW)
    │   └── Markdown/
    │       ├── Markdown.tsx
    │       ├── AstToReact.tsx   ← renamed from AstToVNode
    │       ├── CodeBlock.tsx
    │       ├── InsertAtCaretIcon.tsx
    │       └── utils.ts         ← walker rewritten on createElement
    └── HistoryPanel/
        ├── HistoryPanel.tsx
        ├── ConversationList.tsx
        └── ConversationRenameDialog.tsx
```

Re-exports from `plugins/ai/index.ts` flip from the Vue components to
the React components at the end of Phase 4.

### Phase 1 — React context provider (`plugins/ai/react/context.tsx`)

Replaces `ProvideAIContext.vue`. The React provider:

1. Reads Pinia / Vue refs via `useVueState`:
   - `aiSetting` (from `settingV1Store.getSettingByName(AI)`), with `getOrFetchSettingByName(AI, true)` fired in a mount-effect.
   - `instance` + `engine` (from `useConnectionOfCurrentSQLEditorTab()`).
   - `database` + `databaseMetadata` (from `useMetadata`).
   - `schema` (from `tabStore.currentTab?.connection.schema`).
2. Sets up React-side state: `showHistoryDialog` (`useState(false)`),
   `pendingSendChat` (`useRef<{content:string} | undefined>()` — these
   are one-shot triggers, not rendered state, so refs suffice with a
   `bumpSignal` counter for the dependent effects).
3. Subscribes to `aiContextEvents`:
   - `new-conversation` → "reuse if current chat is empty, otherwise
     create" + flip `pendingPreInput`. Identical semantics to the Vue
     `useEmitteryEventListener` block.
   - `send-chat` → close history dialog, optionally create a new
     conversation, then set `pendingSendChat`.
4. `useCurrentChat(context)` is replaced by a React-side
   `useCurrentChat()` hook that reads through the context, exposing
   plain values (`list`, `ready`, `selected`, `setSelected`) instead of
   Vue refs. The existing TS `useCurrentChat` Vue helper stays for
   imports outside the React tree (none after this stage; will be
   reviewed at the end of Phase 4).

**Behavioural pitfalls to preserve:**
- The `pendingSendChat` write is wrapped in `requestAnimationFrame(...)`
  in the Vue version. The reason is the consumer (`ChatPanel`) has a
  `watch([ready, pendingSendChat], …, { flush: "post" })` that fires
  after the conversation creation settles. The React equivalent:
  reuse the same `requestAnimationFrame` so the consumer effect runs
  on the next frame, after `selected` mutates. **Don't** drop the rAF
  — without it, a race fires the send-chat effect against the still-old
  conversation.
- `aiContextEvents` is a module-level singleton. The provider stores
  the same reference; React's `useEffect(off, [])` cleanup unsubs.
- `useChatByTab` keeps the module-level `chatsByTab: Map<string,
  AIChatInfo>` cache. We don't move that cache into the React provider
  — React mount/unmount cycles must NOT invalidate per-tab fetch state.

### Phase 2 — Leaf components (Markdown + ChatView + HistoryPanel)

Bottom-up. Each leaf has zero React siblings yet, so they slot into the
still-Vue tree via `<ReactPageMount>` for in-progress testing if needed
(but in practice the full Phase-2-through-4 chain lands together).

**Markdown subtree** (4 files):
- `utils.ts` — port the mdast walker from `h(...)` to
  `createElement(...)`. Identical control flow. Slots become a small
  `Slots` interface `{ code?: (node)=>ReactNode; inlineCode?: ...; image?: ...; }`.
- `AstToReact.tsx` — receives `ast: Root` + `slots: Slots`, returns
  `mdastToReact[ast.type](ast, { slots, definitionById: new Map() })`.
- `Markdown.tsx` — `useMemo` over the `unified().use(remarkParse).use(remarkGfm).parse(content)` AST. Passes slots `{ code, inlineCode, image }`.
- `CodeBlock.tsx`:
  - Adaptive width via `useElementSize`-shaped local hook on the
    nearest `.message` ancestor (`findAncestor(containerRef, ".message")`).
  - `<MonacoEditor>` from the React port (autoHeight `min: 20`, `max:
    120`, `padding: 2`; same opts as Vue).
  - Run button → `events.emit("run-statement", { statement: code })` +
    `setShowHistoryDialog(false)`.
  - Insert-at-caret button → `sqlEditorEvents.emit("insert-at-caret", { content: code })` +
    `setShowHistoryDialog(false)`.
  - Copy → `<CopyButton content={code} />` (use existing React copy
    primitive; if none, inline ~15 LOC).
- `InsertAtCaretIcon.tsx` — inline SVG, prop `size`.

**ChatView subtree** (4 + 1):
- `context.tsx` — React Context for `mode: "CHAT" | "VIEW"` (replaces
  Vue's `provideChatViewContext`).
- `ChatView.tsx` — scroller div + `auto-scroll-to-bottom` via
  `useResizeObserver` on the inner container; renders a message-list
  fork (`UserMessageView` / `AIMessageView`) keyed by `message.id`.
- `AIMessageView.tsx` — LOADING → spinner; FAILED → `text-error` pill;
  DONE → `<Markdown content={message.content} codeBlockProps={{ width: 0.85 }} />`.
- `UserMessageView.tsx` — plain text bubble.
- `EmptyView.tsx` — placeholder card.

**HistoryPanel subtree** (3):
- `HistoryPanel.tsx` — `<Sheet>` keyed off `showHistoryDialog`,
  wide width tier. Two columns: `<ConversationList>` (sidebar) +
  `<ChatView mode="VIEW" conversation={selected} />`.
- `ConversationList.tsx` — list + rename + delete (AlertDialog confirm). State:
  `editing: Conversation | undefined` drives the `ConversationRenameDialog`.
- `ConversationRenameDialog.tsx` — simple shadcn `<Dialog>` + `<Input>` + Save/Cancel.

### Phase 3 — Top panels

- `ActionBar.tsx` — three Buttons: New chat (emits `new-conversation`),
  History (toggles `showHistoryDialog`), Close (only when host listens
  for a `close` event; verify against existing Vue `defineEmits` —
  current ActionBar has no `close` so this is just New / History).
- `PromptInput.tsx`:
  - `react-textarea-autosize` (or a hand-rolled rows controller —
    decision per the dep check in §3) bound to local state.
  - Enter → `applyValue` if non-empty, Shift+Enter inserts newline.
  - `useEffect` on mount focuses the textarea (`autoFocus` won't
    survive a mode-switch remount in some flows; explicit focus is
    safer).
  - Reacts to `pendingPreInput`: when it transitions to a non-empty
    string, set `state.value = pendingPreInput`, clear the ref,
    optionally `requestAnimationFrame` so the parent effect fires
    after layout. Mirrors the Vue `watch(pendingPreInput, … { flush: "post" })`.
  - Listens for `new-conversation` to re-focus the input.
- `DynamicSuggestions.tsx`:
  - `useDynamicSuggestions()` already exists in `logic/`. Verify its
    return shape works inside a React component — if it returns Vue
    refs, wrap with `useVueState` getters. If it imperatively kicks
    off a fetch on mount, wrap that call in `useEffect`.
  - Renders up to 3 clickable suggestion chips. Click → `@enter`.
- `ChatPanel.tsx`:
  - The heavy `requestAI(query)` orchestrator ports as a `useCallback`
    over `selectedConversation`, `tab`, `context`. Watch out:
    `store.createMessage` mutates the conversation list — verify the
    React state subscription (`useVueState(() => store.conversationList)`
    triggers re-renders).
  - The two Vue `watch` blocks:
    - `watch([ready, conversationList], …)` — converts to a `useEffect`
      with `[ready, conversationList.length === 0]` (we only need the
      transition to "empty + ready", not full identity).
    - `watch([ready, pendingSendChat], … { flush: "post" })` — converts
      to a `useEffect` with a `setTimeout(0)` or `requestAnimationFrame`
      to mimic `flush: "post"` (run after the next commit). The
      pendingSendChat trigger comes via the provider's ref + bumpSignal;
      the effect checks `if (!pendingSendChat.current) return;`.
  - `onConnectionChanged` Vue helper → if it currently only sets
    `showHistoryDialog.value = false`, a single `useEffect` on
    `[currentTab?.connection.instance, currentTab?.connection.database]`
    that calls `setShowHistoryDialog(false)` is sufficient.

### Phase 4 — Outer wrapper + mount-point flip + cleanup

- `plugins/ai/react/AIChatToSQL.tsx`:
  ```tsx
  const ChatPanel = lazy(() => import("./ChatPanel"));
  export function AIChatToSQL() {
    const { aiSetting } = useAIContext();
    if (!aiSetting.enabled) return null;
    return <Suspense fallback={null}><ChatPanel /></Suspense>;
  }
  ```
- `plugins/ai/index.ts` — re-exports flip:
  ```ts
  export * from "./types";
  export * from "./components/editor-actions";
  export { AIChatToSQL } from "./react/AIChatToSQL";
  export { AIContextProvider } from "./react/context";
  // `AIChatToSQLBridgeHost` and `ProvideAIContext` no longer exported.
  ```
- Mount points:
  - `StandardPanel/StandardPanel.tsx:107` —
    ```diff
    -<VueMount component={AIChatToSQLBridgeHost} />
    +<AIContextProvider>
    +  <AIChatToSQL />
    +</AIContextProvider>
    ```
  - `Panels/Panels.tsx:233` — identical change.
- Delete:
  - All 18 `.vue` files under `plugins/ai/components/`.
  - `plugins/ai/components/AIChatToSQLBridgeHost.vue` (already counted).
  - `frontend/src/react/components/VueMount.tsx` + its smoke tests (3
    tests in `VueMount.test.tsx`) — no other consumers.
- Update memory + any inline comments referencing `VueMount` / `Stage 22`
  / "AI plugin stub" as live state.

## 5. Per-phase checklist

### Phase 1
- [ ] `plugins/ai/react/context.tsx` with `AIContextProvider`, `useAIContext`.
- [ ] React `useCurrentChat()` helper (returns `{ list, ready, selected, setSelected }`, all plain values).
- [ ] `new-conversation` and `send-chat` listeners attached + cleaned up on unmount.
- [ ] AI setting fetched on mount (`getOrFetchSettingByName(AI, true)`).
- [ ] Type-check: `pnpm --dir frontend type-check` green.

### Phase 2
- [ ] `Markdown/utils.ts` ported (no `h(...)`, no `import "vue"`).
- [ ] `Markdown.tsx`, `AstToReact.tsx`, `CodeBlock.tsx`, `InsertAtCaretIcon.tsx`.
- [ ] `ChatView/{ChatView,AIMessageView,UserMessageView,EmptyView,context}.tsx`.
- [ ] `HistoryPanel/{HistoryPanel,ConversationList,ConversationRenameDialog}.tsx`.
- [ ] No `.vue` imports left in any new `.tsx` file under `plugins/ai/react/`.
- [ ] `pnpm --dir frontend test` covering `Markdown` (renders code block / inline code / table / image), `CodeBlock` (run / insert / copy emits events), `ConversationList` (rename / delete confirm wiring).

### Phase 3
- [ ] `ActionBar.tsx`, `PromptInput.tsx`, `DynamicSuggestions.tsx`, `ChatPanel.tsx`.
- [ ] `PromptInput` keyboard behaviour matches Vue: Enter = submit non-empty, Shift+Enter = newline, focus on mount + on `new-conversation`.
- [ ] `ChatPanel.requestAI` builds the prompt-with-schema preamble on the first message exactly like Vue (`promptUtils.declaration(databaseMetadata, engine, schema)`).
- [ ] `pendingSendChat` flush-post effect uses rAF/setTimeout-0 — verify with a manual test (send-chat → conversation auto-created → request fires once, not on the previous conversation).

### Phase 4
- [ ] `AIChatToSQL.tsx` with lazy `ChatPanel` + Suspense fallback.
- [ ] `plugins/ai/index.ts` exports updated.
- [ ] `StandardPanel.tsx` + `Panels.tsx` swapped.
- [ ] `plugins/ai/components/` directory deleted.
- [ ] `react/components/VueMount.tsx` + `VueMount.test.tsx` deleted.
- [ ] `pnpm --dir frontend check` (no raw `bg-*` colors, no `space-x-*` between buttons, etc.) green.
- [ ] `pnpm --dir frontend type-check` green.
- [ ] `pnpm --dir frontend test` green.
- [ ] No `import.*\.vue` left under `frontend/src/plugins/ai/` (`grep -rn '\.vue"' frontend/src/plugins/ai/` should be empty).

## 6. Manual UX verification

Cover the full happy path + the historically-touchy seams:

1. **Open SQL editor → AI side pane**. `aiSetting.enabled = true`. Panel
   mounts, prompt input focused, `ActionBar` visible.
2. **Type a prompt → Enter**. New conversation auto-created if list
   empty, USER message appears immediately, AI message in LOADING,
   spinner. AICompletion resolves → markdown rendered, code block
   shows with Run / Insert-at-Caret / Copy actions.
3. **Run** the code block → SQL editor receives `run-statement` event,
   statement executes.
4. **Insert-at-caret** the code block → SQL editor caret receives the
   snippet (the existing `insert-at-caret` event on `sqlEditorEvents`).
5. **Copy** the code block → clipboard contains the snippet, ephemeral
   "Copied" indicator.
6. **History** button → drawer opens; conversation list on the left,
   selected conversation on the right (`mode="VIEW"`, no input).
7. **Rename** a conversation → dialog opens with current title, save
   updates the list.
8. **Delete** a conversation → AlertDialog confirm, then removed from
   the list.
9. **Switch tabs / databases**: conversation list updates to the
   per-`(instance, database)` cached value. The module-level
   `chatsByTab` cache survives the React unmount/remount cycle.
10. **`Shift+Enter`** in PromptInput inserts a newline; `Enter` on
    empty input is a no-op.
11. **`pendingPreInput`**: triggered when an external action emits
    `new-conversation` with an `input` payload (e.g., from
    `OpenAIButton.tsx`'s "Ask AI about this query"). The PromptInput
    populates with the seed text without auto-submitting.
12. **`aiSetting.enabled = false`** → `<AIChatToSQL />` renders `null`;
    the side pane shows nothing (or the host's "disabled" state).

## 7. Out of scope (deferred)

- Streaming markdown / incremental Monaco code-block render — Vue does
  one-shot render after `aICompletion` resolves; React mirror does
  the same. Future enhancement.
- Conversation export / sharing.
- Multi-model selection UI — the AI setting picker lives in workspace
  settings, not in this side pane.
- Replacing `unified() + remark-parse + remark-gfm` with a lighter
  parser. Same dep on both sides; revisit only if bundle size becomes
  a concern.

## 8. Risks & open questions

- **`useDynamicSuggestions()` return shape** — currently a Vue ref
  bundle. The React port either needs a thin React wrapper that
  subscribes via `useVueState`, or the original helper needs to be
  re-shaped to expose plain values. Resolve during Phase 3 by reading
  the helper source. If the helper is internal-only (no other
  consumer), refactor in place.

- **`HighlightCodeBlock` Vue dep** — used by `Markdown.vue` for inline
  code highlighting. Need to either (a) port the Vue component to React,
  or (b) substitute a plain `<code>` for inline code (Vue inline code
  uses the highlighter for short snippets where the highlighter usually
  no-ops anyway). Prefer (b) for parity-with-less-work; revisit if QA
  shows missing highlights.

- **`CopyButton` import path** — `@/components/v2` is Vue. If no React
  equivalent exists in `react/components/`, inline a small `<button>`
  + `lucide` `Copy` icon + clipboard call inside `CodeBlock.tsx`.
  Decision deferred to first encounter in Phase 2.

- **`<Suspense>` placement** — Vue wraps `<AIChatToSQL>` itself in
  `<Suspense>` from outside. React's `lazy()` requires the consumer to
  wrap in `<Suspense>`. The current design puts the boundary inside
  `AIChatToSQL.tsx`. Alternative: defer the `lazy()` import and ship
  `ChatPanel` synchronously — saves one Suspense boundary but inflates
  the initial bundle of every SQL Editor page load (the Vue pattern
  defers ChatPanel + its mdast deps until the side pane opens). Stick
  with `lazy()`.

- **Memory update** — `project_sql_editor_react_migration_roadmap.md`
  needs a Stage 22 status entry once this lands. The "(post-Stage-21,
  optional)" framing flips to "Complete (2026-MM-DD): AI plugin ported,
  VueMount bridge deleted." Roadmap is now closed.
