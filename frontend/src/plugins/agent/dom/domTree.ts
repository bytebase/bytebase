export interface DomRefSuggestion {
  ref: string;
  tag: string;
  role?: string;
  label: string;
  value?: string;
}

export interface IndexedElement extends DomRefSuggestion {
  element: Element;
}

const INTERACTIVE_TAGS = new Set([
  "A",
  "BUTTON",
  "INPUT",
  "SELECT",
  "TEXTAREA",
  "DETAILS",
  "SUMMARY",
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

const NAIVE_INTERACTIVE_CLASSES = ["n-button", "n-switch", "n-checkbox"];

const TERMINAL_INTERACTIVE_TAGS = new Set([
  "A",
  "BUTTON",
  "INPUT",
  "SELECT",
  "TEXTAREA",
  "SUMMARY",
]);

const SKIP_TAGS = new Set(["SCRIPT", "STYLE", "NOSCRIPT", "SVG", "PATH"]);
const MAX_TEXT_NODE_LENGTH = 160;
const MAX_LABEL_LENGTH = 120;
const MAX_EDITOR_PREVIEW_LENGTH = 160;
const MAX_DOM_TREE_LINES = 120;
const MAX_DOM_TREE_CHARS = 6000;

const LANDMARK_TAGS = new Set([
  "NAV",
  "MAIN",
  "HEADER",
  "FOOTER",
  "ASIDE",
  "SECTION",
  "ARTICLE",
  "FORM",
  "TABLE",
  "THEAD",
  "TBODY",
  "TR",
  "UL",
  "OL",
  "LI",
  "DIALOG",
]);

function isLandmark(el: Element): boolean {
  if (LANDMARK_TAGS.has(el.tagName)) return true;
  if (el.getAttribute("role")) return true;
  if (el.getAttribute("aria-label")) return true;
  return false;
}

const elementRegistry = new Map<string, IndexedElement>();
let nextElementRef = 1;

interface DomLine {
  text: string;
  kind: "interactive" | "context" | "text";
  entry?: IndexedElement;
  dedupeKey?: string;
}

interface WalkContext {
  lines: DomLine[];
  textSeenByParent: Map<string, Set<string>>;
  parentPath: string;
  interactiveLabels: string[];
}

function createElementRef(): string {
  return `e${nextElementRef++}`;
}

function createIndexedElement(entry: Omit<IndexedElement, "ref">): IndexedElement {
  return { ref: createElementRef(), ...entry };
}

export function getElementByRef(ref: string): IndexedElement | undefined {
  return elementRegistry.get(ref);
}

function isVisible(el: Element): boolean {
  const style = window.getComputedStyle(el);
  return (
    style.display !== "none" &&
    style.visibility !== "hidden" &&
    style.opacity !== "0"
  );
}

function hasHiddenAncestor(el: Element): boolean {
  return el.closest('[hidden], [aria-hidden="true"], [inert]') !== null;
}

function isPerceivable(el: Element): boolean {
  return isVisible(el) && !hasHiddenAncestor(el);
}

function hasDisabledState(el: Element): boolean {
  if (el.matches(":disabled") || el.hasAttribute("disabled")) return true;
  return el.closest('[aria-disabled="true"], [disabled]') !== null;
}

function isContentEditableElement(el: Element): boolean {
  const contentEditable = el.getAttribute("contenteditable");
  return (
    el instanceof HTMLElement &&
    (el.isContentEditable ||
      contentEditable === "" ||
      contentEditable === "true" ||
      contentEditable === "plaintext-only")
  );
}

function hasPointerCursor(el: Element): boolean {
  return window.getComputedStyle(el).cursor === "pointer";
}

function isMonacoEditor(el: Element): boolean {
  return el.classList.contains("monaco-editor");
}

function getMonacoContent(el: Element): string | undefined {
  // Read content directly from Monaco's rendered DOM lines
  const viewLines = el.querySelector(".view-lines");
  if (!viewLines) return undefined;
  const lines: string[] = [];
  for (const line of Array.from(viewLines.querySelectorAll(".view-line"))) {
    lines.push(line.textContent ?? "");
  }
  const content = lines.join("\n").trim();
  return content || undefined;
}

function isInteractive(el: Element): boolean {
  if (!isPerceivable(el) || hasDisabledState(el)) return false;
  if (INTERACTIVE_TAGS.has(el.tagName)) return true;

  const role = el.getAttribute("role");
  if (role && INTERACTIVE_ROLES.has(role)) return true;

  for (const cls of NAIVE_INTERACTIVE_CLASSES) {
    if (el.classList.contains(cls)) return true;
  }

  if (isContentEditableElement(el)) return true;
  if (hasPointerCursor(el)) return true;

  return false;
}

function shouldRecurseIntoInteractive(el: Element): boolean {
  if (TERMINAL_INTERACTIVE_TAGS.has(el.tagName)) return false;
  if (isContentEditableElement(el)) return false;
  return Array.from(el.children).length > 0;
}

function truncateText(text: string, maxLength: number): string {
  if (text.length <= maxLength) return text;
  return `${text.slice(0, maxLength)}...`;
}

function normalizeTextContent(text: string): string | undefined {
  const normalized = text.trim().replace(/\s+/g, " ");
  if (!normalized) return undefined;
  return truncateText(normalized, MAX_TEXT_NODE_LENGTH);
}

function extractLabel(el: Element): string {
  // aria-label
  const ariaLabel = normalizeTextContent(el.getAttribute("aria-label") ?? "");
  if (ariaLabel) return truncateText(ariaLabel, MAX_LABEL_LENGTH);

  // aria-labelledby
  const labelledBy = el.getAttribute("aria-labelledby");
  if (labelledBy) {
    const parts = labelledBy
      .split(/\s+/)
      .map((id) =>
        normalizeTextContent(document.getElementById(id)?.textContent ?? "")
      )
      .filter((part): part is string => Boolean(part));
    if (parts.length > 0) {
      return truncateText(parts.join(" "), MAX_LABEL_LENGTH);
    }
  }

  // placeholder
  if (el instanceof HTMLInputElement || el instanceof HTMLTextAreaElement) {
    const placeholder = normalizeTextContent(el.placeholder);
    if (placeholder) return truncateText(placeholder, MAX_LABEL_LENGTH);
  }

  // Naive UI form-item label
  const formItem = el.closest(".n-form-item");
  if (formItem) {
    const label = normalizeTextContent(
      formItem.querySelector(".n-form-item-label__text")?.textContent ?? ""
    );
    if (label) return truncateText(label, MAX_LABEL_LENGTH);
  }

  // title
  const title = normalizeTextContent(el.getAttribute("title") ?? "");
  if (title) return truncateText(title, MAX_LABEL_LENGTH);

  // text content (truncated)
  const text = normalizeTextContent(el.textContent ?? "");
  return text ? truncateText(text, MAX_LABEL_LENGTH) : "";
}

function extractValue(el: Element): string | undefined {
  if (el instanceof HTMLInputElement) {
    if (el.type === "checkbox" || el.type === "radio") {
      return el.checked ? "checked" : "unchecked";
    }
    const value = normalizeTextContent(el.value);
    return value ? truncateText(value, MAX_LABEL_LENGTH) : undefined;
  }
  if (el instanceof HTMLTextAreaElement) {
    const value = normalizeTextContent(el.value);
    return value ? truncateText(value, MAX_LABEL_LENGTH) : undefined;
  }
  if (el instanceof HTMLSelectElement) {
    const value = normalizeTextContent(
      el.options[el.selectedIndex]?.text || el.value || ""
    );
    return value ? truncateText(value, MAX_LABEL_LENGTH) : undefined;
  }

  // Naive UI select
  if (el.classList.contains("n-base-selection")) {
    const value = normalizeTextContent(
      el.querySelector(".n-base-selection-label")?.textContent ?? ""
    );
    return value ? truncateText(value, MAX_LABEL_LENGTH) : undefined;
  }

  // Naive UI checkbox/switch
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

function toDomRefSuggestion({
  ref,
  tag,
  role,
  label,
  value,
}: IndexedElement): DomRefSuggestion {
  return {
    ref,
    tag,
    role,
    label,
    value,
  };
}

function classifyTextLine(parent: Element, depth: number, text: string): DomLine["kind"] {
  if (depth <= 1 || isLandmark(parent) || text.length <= 40) {
    return "context";
  }
  return "text";
}

function pushTextLine(node: Text, depth: number, context: WalkContext): void {
  if (!node.parentElement || !isPerceivable(node.parentElement)) return;
  const text = normalizeTextContent(node.textContent ?? "");
  if (!text) return;
  if (context.interactiveLabels.includes(text)) return;

  const seenForParent =
    context.textSeenByParent.get(context.parentPath) ?? new Set<string>();
  if (seenForParent.has(text)) return;
  seenForParent.add(text);
  context.textSeenByParent.set(context.parentPath, seenForParent);

  const indent = "  ".repeat(depth);
  context.lines.push({
    text: `${indent}${text}`,
    kind: classifyTextLine(node.parentElement, depth, text),
    dedupeKey: `${depth}:${text}`,
  });
}

function pushInteractiveLine(
  depth: number,
  entry: IndexedElement,
  value: string | undefined,
  context: WalkContext
): void {
  const indent = "  ".repeat(depth);
  const valueAttr = value ? ` value="${value}"` : "";
  context.lines.push({
    text: `${indent}[${entry.ref}]<${entry.tag}${valueAttr}>${entry.label}</${entry.tag}>`,
    kind: "interactive",
    entry,
  });
}

function walkDomNode(node: Node, depth: number, context: WalkContext, path: string): void {
  if (node instanceof Text) {
    pushTextLine(node, depth, context);
    return;
  }

  if (!(node instanceof Element)) return;
  if (SKIP_TAGS.has(node.tagName)) return;
  if (node.hasAttribute("data-agent-window")) return;
  if (!isPerceivable(node)) return;

  // Monaco editor — register as a single interactive element
  if (isMonacoEditor(node)) {
    const content = getMonacoContent(node) ?? "";
    const preview = truncateText(content, MAX_EDITOR_PREVIEW_LENGTH);
    const label = content ? `SQL: ${preview}` : "empty editor";
    const entry = createIndexedElement({
      tag: "editor",
      label,
      value: content,
      element: node,
    });

    pushInteractiveLine(depth, entry, undefined, context);
    return;
  }

  const isInteractiveNode = isInteractive(node);
  let nextInteractiveLabels = context.interactiveLabels;
  if (isInteractiveNode) {
    const tag = node.tagName.toLowerCase();
    const label = extractLabel(node);
    const value = extractValue(node);
    const role = node.getAttribute("role") ?? undefined;
    const entry = createIndexedElement({ tag, role, label, value, element: node });

    pushInteractiveLine(depth, entry, value, context);
    nextInteractiveLabels = label
      ? [...context.interactiveLabels, label]
      : context.interactiveLabels;

    if (!shouldRecurseIntoInteractive(node)) {
      return;
    }
  }

  const childDepth = isInteractiveNode
    ? depth + 1
    : isLandmark(node)
      ? depth + 1
      : depth;
  const childContext: WalkContext = {
    ...context,
    parentPath: path,
    interactiveLabels: nextInteractiveLabels,
  };
  const childNodes = Array.from(node.childNodes);
  childNodes.forEach((child, index) => {
    walkDomNode(
      child,
      childDepth,
      childContext,
      `${path}/${node.tagName.toLowerCase()}[${index}]`
    );
  });
}

function dedupeAdjacentLines(lines: DomLine[]): DomLine[] {
  const deduped: DomLine[] = [];
  for (const line of lines) {
    const previous = deduped[deduped.length - 1];
    if (
      previous &&
      !line.entry &&
      !previous.entry &&
      previous.dedupeKey === line.dedupeKey
    ) {
      continue;
    }
    deduped.push(line);
  }
  return deduped;
}

function selectBudgetedLines(lines: DomLine[]): DomLine[] {
  const selected = new Set<number>();
  let usedLines = 0;
  let usedChars = 0;

  const trySelect = (index: number): void => {
    if (selected.has(index)) return;
    const line = lines[index];
    const cost = line.text.length + (usedLines > 0 ? 1 : 0);
    if (usedLines >= MAX_DOM_TREE_LINES || usedChars + cost > MAX_DOM_TREE_CHARS) {
      return;
    }
    selected.add(index);
    usedLines += 1;
    usedChars += cost;
  };

  for (const kind of ["interactive", "context", "text"] as const) {
    lines.forEach((line, index) => {
      if (line.kind === kind) {
        trySelect(index);
      }
    });
  }

  return lines.filter((_, index) => selected.has(index));
}

function extractDomState(root?: Element): {
  tree: string;
  count: number;
  refs: DomRefSuggestion[];
} {
  elementRegistry.clear();
  nextElementRef = 1;
  const rootEl = root ?? document.body;
  const lines: DomLine[] = [];
  const context: WalkContext = {
    lines,
    textSeenByParent: new Map(),
    parentPath: "root",
    interactiveLabels: [],
  };

  Array.from(rootEl.childNodes).forEach((child, index) => {
    walkDomNode(child, 0, context, `root[${index}]`);
  });

  const keptLines = selectBudgetedLines(dedupeAdjacentLines(lines));
  const keptEntries = keptLines
    .flatMap((line) => (line.entry ? [line.entry] : []))
    .map((entry) => [entry.ref, entry] as const);
  elementRegistry.clear();
  keptEntries.forEach(([ref, entry]) => {
    elementRegistry.set(ref, entry);
  });

  return {
    tree: keptLines.map((line) => line.text).join("\n"),
    count: elementRegistry.size,
    refs: Array.from(elementRegistry.values(), toDomRefSuggestion),
  };
}

export function extractDomTree(root?: Element): {
  tree: string;
  count: number;
} {
  const { tree, count } = extractDomState(root);
  return { tree, count };
}

export function extractDomRefSuggestions(root?: Element): DomRefSuggestion[] {
  return extractDomState(root).refs;
}
