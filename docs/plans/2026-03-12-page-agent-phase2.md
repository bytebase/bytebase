# Page Agent Phase 2: DOM Engine — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add DOM tree extraction and DOM interaction tools to the page agent, enabling the LLM to read and interact with UI elements when APIs don't cover the task.

**Architecture:** A lazy-loaded DOM engine (`plugins/agent/dom/`) provides two capabilities: (1) extract a compact indexed text representation of interactive elements on the page, and (2) execute actions (click, input, select, scroll) on those elements by index. Two new tools (`get_dom_tree`, `dom_action`) are registered alongside existing tools. The `get_page_state` tool gains a `mode` parameter to optionally return DOM tree as fallback.

**Tech Stack:** TypeScript, Vue 3, Naive UI, Vite dynamic import (lazy loading)

---

## File Overview

```
frontend/src/plugins/agent/
├── dom/                          # NEW — lazy-loaded DOM engine
│   ├── domTree.ts                # DOM tree extraction + element indexing
│   ├── actions.ts                # DOM action execution (click, input, select, scroll)
│   └── index.ts                  # Public API — lazy-load entry point
├── logic/tools/
│   ├── index.ts                  # MODIFY — register dom_action, update get_page_state
│   ├── domAction.ts              # NEW — dom_action tool implementation
│   └── pageState.ts              # MODIFY — add DOM fallback mode
└── logic/
    └── prompt.ts                 # MODIFY — add DOM tool guidance to system prompt
```

---

## Task 1: DOM Tree Extraction

Build the core DOM tree walker that produces an indexed text representation of interactive elements.

**Files:**
- Create: `frontend/src/plugins/agent/dom/domTree.ts`

**Step 1: Create `domTree.ts` with element extraction**

```typescript
// frontend/src/plugins/agent/dom/domTree.ts

export interface IndexedElement {
  index: number;
  tag: string;
  role?: string;
  label: string;
  value?: string;
  // Reference to the actual DOM node for action execution
  element: Element;
}

// Module-level registry, rebuilt on each extraction
let elementRegistry: IndexedElement[] = [];

export function getElementByIndex(index: number): IndexedElement | undefined {
  return elementRegistry.find((e) => e.index === index);
}

const INTERACTIVE_TAGS = new Set([
  "a",
  "button",
  "input",
  "select",
  "textarea",
  "details",
  "summary",
]);

const INTERACTIVE_ROLES = new Set([
  "button",
  "link",
  "textbox",
  "combobox",
  "listbox",
  "option",
  "menuitem",
  "tab",
  "checkbox",
  "radio",
  "switch",
  "slider",
]);

const SKIP_TAGS = new Set(["script", "style", "noscript", "svg", "path"]);

function isVisible(el: Element): boolean {
  const style = window.getComputedStyle(el);
  return (
    style.display !== "none" &&
    style.visibility !== "hidden" &&
    style.opacity !== "0"
  );
}

function isInteractive(el: Element): boolean {
  const tag = el.tagName.toLowerCase();
  if (INTERACTIVE_TAGS.has(tag)) return true;

  const role = el.getAttribute("role");
  if (role && INTERACTIVE_ROLES.has(role)) return true;

  // Clickable elements
  if (el.hasAttribute("onclick") || el.hasAttribute("tabindex")) return true;
  if (el.classList.contains("n-button")) return true;
  if (el.classList.contains("n-switch")) return true;
  if (el.classList.contains("n-checkbox")) return true;

  return false;
}

function getLabel(el: Element): string {
  // Priority: aria-label > aria-labelledby > placeholder > title > text content
  const ariaLabel = el.getAttribute("aria-label");
  if (ariaLabel) return ariaLabel;

  const labelledBy = el.getAttribute("aria-labelledby");
  if (labelledBy) {
    const labelEl = document.getElementById(labelledBy);
    if (labelEl) return labelEl.textContent?.trim() ?? "";
  }

  if (el instanceof HTMLInputElement || el instanceof HTMLTextAreaElement) {
    if (el.placeholder) return `placeholder="${el.placeholder}"`;
    // Walk up to find associated label or n-form-item label
    const formItem = el.closest(".n-form-item");
    if (formItem) {
      const labelEl = formItem.querySelector(".n-form-item-label__text");
      if (labelEl?.textContent) return labelEl.textContent.trim();
    }
  }

  const title = el.getAttribute("title");
  if (title) return title;

  // For buttons/links, use direct text content (not deep children)
  const text = el.textContent?.trim() ?? "";
  // Truncate long text
  return text.length > 80 ? text.slice(0, 77) + "..." : text;
}

function getValue(el: Element): string | undefined {
  if (el instanceof HTMLInputElement) return el.value || undefined;
  if (el instanceof HTMLTextAreaElement) return el.value || undefined;
  if (el instanceof HTMLSelectElement) return el.value || undefined;

  // Naive UI select — find displayed value
  if (el.classList.contains("n-base-selection")) {
    const input = el.querySelector(".n-base-selection-label, .n-tag");
    return input?.textContent?.trim() || undefined;
  }

  // Checkbox/switch state
  if (
    el.classList.contains("n-checkbox") ||
    el.classList.contains("n-switch")
  ) {
    const isChecked =
      el.classList.contains("n-checkbox--checked") ||
      el.classList.contains("n-switch--active");
    return isChecked ? "checked" : "unchecked";
  }

  return undefined;
}

function formatTag(el: Element): string {
  const tag = el.tagName.toLowerCase();
  const type = el.getAttribute("type");
  if (tag === "input" && type) return `input[type=${type}]`;
  return tag;
}

export function extractDomTree(
  root: Element = document.body
): { tree: string; count: number } {
  elementRegistry = [];
  let index = 0;
  const lines: string[] = [];

  function walk(node: Element, depth: number) {
    const tag = node.tagName.toLowerCase();
    if (SKIP_TAGS.has(tag)) return;
    if (!isVisible(node)) return;

    if (isInteractive(node)) {
      index++;
      const label = getLabel(node);
      const value = getValue(node);
      const formattedTag = formatTag(node);
      const role = node.getAttribute("role");

      elementRegistry.push({
        index,
        tag: formattedTag,
        role: role ?? undefined,
        label,
        value,
        element: node,
      });

      const indent = "  ".repeat(depth);
      let line = `${indent}[${index}]<${formattedTag}`;
      if (role) line += ` role="${role}"`;
      line += `>`;
      if (label) line += label;
      if (value) line += ` (value="${value}")`;
      line += `</${tag}>`;
      lines.push(line);

      // Don't recurse into interactive elements — they're already captured
      return;
    }

    // Non-interactive: recurse into children
    for (const child of node.children) {
      walk(child, depth);
    }
  }

  walk(root, 0);
  return { tree: lines.join("\n"), count: index };
}
```

**Step 2: Verify manually**

Open the browser console on any Bytebase page and test `extractDomTree()`. This will be validated properly once wired into the tool.

**Step 3: Commit**

```
feat(agent): add DOM tree extraction engine
```

---

## Task 2: DOM Actions

Implement click, input, select, and scroll actions that dispatch proper DOM events for Vue reactivity.

**Files:**
- Create: `frontend/src/plugins/agent/dom/actions.ts`

**Step 1: Create `actions.ts`**

```typescript
// frontend/src/plugins/agent/dom/actions.ts

import { getElementByIndex } from "./domTree";

export type DomActionType = "click" | "input" | "select" | "scroll";

export interface DomActionParams {
  type: DomActionType;
  index: number;
  value?: string;
}

export interface DomActionResult {
  success: boolean;
  message: string;
}

function dispatchMouseClick(el: Element) {
  const rect = el.getBoundingClientRect();
  const x = rect.left + rect.width / 2;
  const y = rect.top + rect.height / 2;
  const opts: MouseEventInit = {
    bubbles: true,
    cancelable: true,
    clientX: x,
    clientY: y,
  };
  el.dispatchEvent(new MouseEvent("mousedown", opts));
  el.dispatchEvent(new MouseEvent("mouseup", opts));
  el.dispatchEvent(new MouseEvent("click", opts));
}

function performInput(el: Element, value: string): DomActionResult {
  // Native input/textarea
  if (el instanceof HTMLInputElement || el instanceof HTMLTextAreaElement) {
    // Use native setter to bypass React/Vue proxy
    const nativeSetter = Object.getOwnPropertyDescriptor(
      el instanceof HTMLInputElement
        ? HTMLInputElement.prototype
        : HTMLTextAreaElement.prototype,
      "value"
    )?.set;
    nativeSetter?.call(el, value);
    el.dispatchEvent(new Event("input", { bubbles: true }));
    el.dispatchEvent(new Event("change", { bubbles: true }));
    return { success: true, message: `Set value to "${value}"` };
  }

  // Naive UI input — find the inner <input> or <textarea>
  const inner =
    el.querySelector("input") ?? el.querySelector("textarea");
  if (inner) {
    return performInput(inner, value);
  }

  return { success: false, message: "Element is not an input" };
}

function performSelect(el: Element, value: string): DomActionResult {
  // Native <select>
  if (el instanceof HTMLSelectElement) {
    el.value = value;
    el.dispatchEvent(new Event("change", { bubbles: true }));
    return { success: true, message: `Selected "${value}"` };
  }

  // Naive UI select — click to open dropdown, then find and click option
  dispatchMouseClick(el);

  // Wait briefly for dropdown to render, then find option
  return new Promise<DomActionResult>((resolve) => {
    setTimeout(() => {
      // Naive UI renders dropdown options in a teleported container
      const options = document.querySelectorAll(
        ".n-base-select-option, .v-binder-follower-content .n-base-select-option"
      );
      for (const opt of options) {
        const text = opt.textContent?.trim();
        if (text === value || opt.getAttribute("data-value") === value) {
          dispatchMouseClick(opt);
          resolve({ success: true, message: `Selected "${value}"` });
          return;
        }
      }
      // Close dropdown if option not found
      document.body.click();
      resolve({
        success: false,
        message: `Option "${value}" not found in dropdown`,
      });
    }, 200);
  }) as unknown as DomActionResult;
}

function performScroll(el: Element): DomActionResult {
  el.scrollIntoView({ behavior: "smooth", block: "center" });
  return { success: true, message: "Scrolled element into view" };
}

export async function executeDomAction(
  params: DomActionParams
): Promise<DomActionResult> {
  const entry = getElementByIndex(params.index);
  if (!entry) {
    return {
      success: false,
      message: `No element found at index ${params.index}. Run get_page_state with mode="dom" first to refresh the element index.`,
    };
  }

  const { element } = entry;

  switch (params.type) {
    case "click":
      dispatchMouseClick(element);
      return {
        success: true,
        message: `Clicked [${params.index}] <${entry.tag}> "${entry.label}"`,
      };
    case "input":
      if (!params.value) {
        return { success: false, message: "value is required for input action" };
      }
      return performInput(element, params.value);
    case "select":
      if (!params.value) {
        return {
          success: false,
          message: "value is required for select action",
        };
      }
      return performSelect(element, params.value);
    case "scroll":
      return performScroll(element);
    default:
      return { success: false, message: `Unknown action type: ${params.type}` };
  }
}
```

**Step 2: Commit**

```
feat(agent): add DOM action execution (click, input, select, scroll)
```

---

## Task 3: DOM Engine Lazy-Load Entry Point

Create the public API that lazy-loads the DOM engine bundle on first use.

**Files:**
- Create: `frontend/src/plugins/agent/dom/index.ts`

**Step 1: Create `dom/index.ts`**

```typescript
// frontend/src/plugins/agent/dom/index.ts

// Lazy-loaded DOM engine. The domTree.ts + actions.ts bundle (~30-50KB)
// is only imported when the LLM first requests DOM interaction.

let loaded = false;
let extractDomTree: typeof import("./domTree").extractDomTree;
let executeDomAction: typeof import("./actions").executeDomAction;

async function ensureLoaded() {
  if (loaded) return;
  const [treeModule, actionsModule] = await Promise.all([
    import("./domTree"),
    import("./actions"),
  ]);
  extractDomTree = treeModule.extractDomTree;
  executeDomAction = actionsModule.executeDomAction;
  loaded = true;
}

export async function lazyExtractDomTree(): Promise<{
  tree: string;
  count: number;
}> {
  await ensureLoaded();
  return extractDomTree();
}

export async function lazyExecuteDomAction(params: {
  type: "click" | "input" | "select" | "scroll";
  index: number;
  value?: string;
}): Promise<{ success: boolean; message: string }> {
  await ensureLoaded();
  return executeDomAction(params);
}
```

**Step 2: Commit**

```
feat(agent): add lazy-load entry point for DOM engine
```

---

## Task 4: `dom_action` Tool

Wire `dom_action` as a new tool in the agent tool registry.

**Files:**
- Create: `frontend/src/plugins/agent/logic/tools/domAction.ts`
- Modify: `frontend/src/plugins/agent/logic/tools/index.ts`

**Step 1: Create `domAction.ts`**

```typescript
// frontend/src/plugins/agent/logic/tools/domAction.ts

import { lazyExecuteDomAction } from "../../dom";

export interface DomActionArgs {
  type: "click" | "input" | "select" | "scroll";
  index: number;
  value?: string;
}

export async function domAction(args: DomActionArgs): Promise<string> {
  const result = await lazyExecuteDomAction(args);
  return JSON.stringify(result);
}
```

**Step 2: Register `dom_action` in `index.ts`**

Add the tool definition and executor case to `frontend/src/plugins/agent/logic/tools/index.ts`:

Add import at top:
```typescript
import { domAction, type DomActionArgs } from "./domAction";
```

Add to the `getToolDefinitions()` return array:
```typescript
{
  name: "dom_action",
  description: `Last-resort DOM interaction — use only when no API endpoint covers the action.
**Always call get_page_state(mode="dom") first** to get the element index.

| Action | When to use | value param |
|--------|-------------|-------------|
| click  | Buttons, links, tabs, checkboxes | not needed |
| input  | Text fields, textareas | required — the text to enter |
| select | Dropdowns | required — the option text to select |
| scroll | Bring element into viewport | not needed |`,
  parametersSchema: {
    type: "object",
    properties: {
      type: {
        type: "string",
        enum: ["click", "input", "select", "scroll"],
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
```

Add to the `switch` in `createToolExecutor`:
```typescript
case "dom_action":
  return domAction(args as unknown as DomActionArgs);
```

**Step 3: Commit**

```
feat(agent): register dom_action tool
```

---

## Task 5: Enhance `get_page_state` with DOM Fallback

Add a `mode` parameter to `get_page_state` so the LLM can request DOM tree extraction.

**Files:**
- Modify: `frontend/src/plugins/agent/logic/tools/pageState.ts`
- Modify: `frontend/src/plugins/agent/logic/tools/index.ts` (update definition + executor)

**Step 1: Update `pageState.ts`**

Replace the file contents with:

```typescript
// frontend/src/plugins/agent/logic/tools/pageState.ts

import type { Router } from "vue-router";
import { lazyExtractDomTree } from "../../dom";

export interface PageStateArgs {
  mode?: "semantic" | "dom";
}

export function createPageStateTool(router: Router) {
  return async (args?: PageStateArgs): Promise<string> => {
    const route = router.currentRoute.value;
    const base = {
      path: route.fullPath,
      name: String(route.name ?? ""),
      params: route.params,
      query: route.query,
      title: document.title,
    };

    if (args?.mode === "dom") {
      const { tree, count } = await lazyExtractDomTree();
      return JSON.stringify({
        ...base,
        domTree: tree,
        interactiveElements: count,
      });
    }

    return JSON.stringify(base);
  };
}
```

**Step 2: Update `get_page_state` definition in `index.ts`**

Replace the existing `get_page_state` definition:

```typescript
{
  name: "get_page_state",
  description: `Get information about the current page.

| Mode | Result |
|------|--------|
| semantic (default) | Route path, params, title |
| dom | Above + indexed DOM tree of interactive elements |

Use mode="dom" before dom_action to get element indices.`,
  parametersSchema: {
    type: "object",
    properties: {
      mode: {
        type: "string",
        enum: ["semantic", "dom"],
        description:
          'Default "semantic" returns route info. "dom" adds an indexed tree of interactive elements for use with dom_action.',
      },
    },
  },
},
```

Update the executor to pass args:

```typescript
case "get_page_state":
  return pageStateTool(args as PageStateArgs);
```

Update the import:
```typescript
import { createPageStateTool, type PageStateArgs } from "./pageState";
```

**Step 3: Commit**

```
feat(agent): add DOM fallback mode to get_page_state
```

---

## Task 6: Update System Prompt

Add DOM tool guidance to the system prompt so the LLM knows when and how to use DOM tools.

**Files:**
- Modify: `frontend/src/plugins/agent/logic/prompt.ts`

**Step 1: Update `prompt.ts`**

Add DOM tool rules to the system prompt. Insert after the existing "Rules:" block:

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
- Use dom_action only when no API covers the task. Always call get_page_state(mode="dom") first.
- Workflow for DOM interaction: get_page_state(mode="dom") → read element indices → dom_action(type, index, value).
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

**Step 2: Commit**

```
feat(agent): add DOM tool guidance to system prompt
```

---

## Task 7: Add i18n Keys

Add locale strings for DOM-related UI elements.

**Files:**
- Modify: `frontend/src/locales/en-US.json`
- Modify: `frontend/src/locales/zh-CN.json` (if exists)

**Step 1: Add keys under `agent`**

In `en-US.json`, extend the `"agent"` block:

```json
"agent": {
  "assistant-title": "Bytebase Assistant",
  "new-chat": "New chat",
  "minimize": "Minimize",
  "close": "Close",
  "input-placeholder": "Ask anything...",
  "send": "Send",
  "args": "Args:",
  "result": "Result:",
  "dom-loading": "Loading DOM engine...",
  "dom-elements-found": "{count} interactive elements found"
},
```

Add equivalent keys to `zh-CN.json`.

**Step 2: Commit**

```
feat(agent): add i18n keys for DOM engine
```

---

## Task 8: Lint, Type-Check, and Build

Validate everything compiles and passes checks.

**Step 1: Run frontend fix**

```bash
pnpm --dir frontend fix
```

**Step 2: Run type-check**

```bash
pnpm --dir frontend type-check
```

Fix any type errors surfaced.

**Step 3: Run check**

```bash
pnpm --dir frontend check
```

**Step 4: Commit any fixes**

```
fix(agent): lint and type-check fixes for DOM engine
```

---

## Task 9: Manual Integration Test

Verify the DOM engine works end-to-end in the running app.

**Step 1: Start dev server**

```bash
pnpm --dir frontend dev
```

**Step 2: Test DOM tree extraction**

1. Open Bytebase in browser, navigate to any page (e.g., project list)
2. Open the Agent window
3. Type: "What interactive elements are on this page?"
4. The agent should call `get_page_state(mode="dom")` and return an indexed list

**Step 3: Test DOM action**

1. Ask: "Click the first button on this page"
2. The agent should call `get_page_state(mode="dom")`, then `dom_action(type="click", index=N)`
3. Verify the correct element was clicked

**Step 4: Test Naive UI select interaction**

1. Navigate to a page with a dropdown (e.g., environment settings)
2. Ask: "Select 'Production' from the environment dropdown"
3. Verify the select opens, the option is found, and it's selected

**Step 5: Verify lazy loading**

1. Open Network tab in DevTools
2. Load a fresh page — no DOM engine chunk should load
3. Trigger a DOM tool call — a new chunk should appear in the network log

---

## Summary

| Task | Files | What |
|------|-------|------|
| 1 | `dom/domTree.ts` | DOM tree walker + element indexing |
| 2 | `dom/actions.ts` | Click, input, select, scroll with proper event dispatch |
| 3 | `dom/index.ts` | Lazy-load entry point |
| 4 | `logic/tools/domAction.ts`, `logic/tools/index.ts` | `dom_action` tool registration |
| 5 | `logic/tools/pageState.ts`, `logic/tools/index.ts` | `get_page_state` DOM mode |
| 6 | `logic/prompt.ts` | System prompt DOM rules |
| 7 | `locales/en-US.json`, `locales/zh-CN.json` | i18n keys |
| 8 | — | Lint + type-check + build |
| 9 | — | Manual integration test |
