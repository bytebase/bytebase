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

const STRUCTURAL_WRAPPER_TAGS = new Set(["DIV", "SPAN", "TD", "TH"]);
const STATEFUL_INTERACTIVE_ROLES = new Set([
  "checkbox",
  "radio",
  "switch",
  "textbox",
  "combobox",
  "listbox",
  "option",
  "slider",
]);
const TEXTBOX_INPUT_TYPES = new Set([
  "",
  "email",
  "number",
  "password",
  "search",
  "tel",
  "text",
  "url",
]);
const CHECKABLE_INPUT_TYPES = new Set(["checkbox", "radio"]);
const BUTTON_LIKE_INPUT_TYPES = new Set(["button", "image", "reset", "submit"]);
const INTERACTIVE_STATE_ATTRIBUTES = [
  "aria-checked",
  "aria-expanded",
  "aria-pressed",
  "aria-selected",
  "aria-current",
];

const SKIP_TAGS = new Set(["SCRIPT", "STYLE", "NOSCRIPT", "SVG", "PATH"]);
const TABLE_SECTION_TAGS = new Set(["THEAD", "TBODY", "TFOOT"]);
const TABLE_CELL_TAGS = new Set(["TH", "TD"]);
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
  sourceElement?: Element;
  signature?: string;
  semanticTag?: string;
}

type InteractiveReason = "native" | "role" | "class" | "editable" | "pointer";

interface WalkContext {
  lines: DomLine[];
  textSeenByParent: Map<string, Set<string>>;
  parentPath: string;
  interactiveLabels: string[];
  hasInteractiveAncestor: boolean;
}

function createElementRef(): string {
  return `e${nextElementRef++}`;
}

function createIndexedElement(
  entry: Omit<IndexedElement, "ref">
): IndexedElement {
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

function isHiddenInput(el: Element): boolean {
  return el instanceof HTMLInputElement && el.type === "hidden";
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

function getInteractiveReason(el: Element): InteractiveReason | undefined {
  if (!isPerceivable(el) || hasDisabledState(el) || isHiddenInput(el))
    return undefined;
  if (INTERACTIVE_TAGS.has(el.tagName)) return "native";

  const role = el.getAttribute("role");
  if (role && INTERACTIVE_ROLES.has(role)) return "role";

  for (const cls of NAIVE_INTERACTIVE_CLASSES) {
    if (el.classList.contains(cls)) return "class";
  }

  if (isContentEditableElement(el)) return "editable";
  if (hasPointerCursor(el)) return "pointer";

  return undefined;
}

function isInteractive(el: Element): boolean {
  return Boolean(getInteractiveReason(el));
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

  if (el instanceof HTMLInputElement && BUTTON_LIKE_INPUT_TYPES.has(el.type)) {
    const buttonValue = normalizeTextContent(el.value);
    if (buttonValue) return truncateText(buttonValue, MAX_LABEL_LENGTH);
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
function hasInteractiveState(el: Element): boolean {
  return INTERACTIVE_STATE_ATTRIBUTES.some((name) => {
    const value = el.getAttribute(name);
    return value !== null && value !== "false";
  });
}

function getFallbackLabel(el: Element, reason: InteractiveReason): string {
  if (el instanceof HTMLInputElement) {
    if (CHECKABLE_INPUT_TYPES.has(el.type)) return el.type;
    if (TEXTBOX_INPUT_TYPES.has(el.type)) return "textbox";
    if (BUTTON_LIKE_INPUT_TYPES.has(el.type)) {
      return normalizeTextContent(el.value) ?? "";
    }
  }

  if (el instanceof HTMLTextAreaElement) return "textbox";
  if (el instanceof HTMLSelectElement) return "select";

  const role = el.getAttribute("role") ?? "";
  if (STATEFUL_INTERACTIVE_ROLES.has(role)) return role;
  if (
    reason === "class" &&
    (el.classList.contains("n-checkbox") || el.classList.contains("n-switch"))
  ) {
    return el.classList.contains("n-switch") ? "switch" : "checkbox";
  }

  return "";
}

function shouldIndexInteractiveNode(
  el: Element,
  reason: InteractiveReason,
  label: string,
  value: string | undefined,
  context: WalkContext
): boolean {
  if (reason === "pointer" && context.hasInteractiveAncestor) return false;

  const hasDescriptor = Boolean(label || value || hasInteractiveState(el));

  if (reason === "pointer") {
    if (!hasDescriptor) return false;
    if (STRUCTURAL_WRAPPER_TAGS.has(el.tagName)) {
      const childCount = el.children.length;
      const meaningfulDescendantCount = Array.from(el.children).filter(
        (child) =>
          !STRUCTURAL_WRAPPER_TAGS.has(child.tagName) || isInteractive(child)
      ).length;
      return (
        childCount === 0 ||
        meaningfulDescendantCount > 0 ||
        el.tagName === "DIV"
      );
    }
    return true;
  }

  if (el instanceof HTMLButtonElement || el instanceof HTMLAnchorElement) {
    return hasDescriptor;
  }

  if (el instanceof HTMLInputElement) {
    if (
      CHECKABLE_INPUT_TYPES.has(el.type) ||
      TEXTBOX_INPUT_TYPES.has(el.type)
    ) {
      return true;
    }
    if (BUTTON_LIKE_INPUT_TYPES.has(el.type)) {
      return hasDescriptor;
    }
    return hasDescriptor;
  }

  if (el instanceof HTMLTextAreaElement || el instanceof HTMLSelectElement) {
    return true;
  }

  if (reason === "editable") return true;

  return hasDescriptor;
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

function normalizeLineSignature(text: string): string | undefined {
  return normalizeTextContent(text)?.toLowerCase();
}

function splitStructuredSignature(signature: string): string[] {
  return signature
    .split("|")
    .map((segment) => normalizeLineSignature(segment) ?? "")
    .filter(Boolean);
}

function classifyTextLine(
  parent: Element,
  depth: number,
  text: string
): DomLine["kind"] {
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
    sourceElement: node.parentElement,
    signature: normalizeLineSignature(text),
    semanticTag: node.parentElement.tagName.toLowerCase(),
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
    sourceElement: entry.element,
    signature: normalizeLineSignature(entry.label),
    semanticTag: entry.tag,
  });
}

function pushContextLine(
  depth: number,
  text: string,
  context: WalkContext,
  dedupeKey?: string,
  sourceElement?: Element,
  signature?: string,
  semanticTag?: string
): void {
  const indent = "  ".repeat(depth);
  context.lines.push({
    text: `${indent}${text}`,
    kind: "context",
    dedupeKey,
    sourceElement,
    signature,
    semanticTag,
  });
}

function collectSemanticTextSegments(node: Node, segments: string[]): void {
  if (node instanceof Text) {
    const text = normalizeTextContent(node.textContent ?? "");
    if (text) segments.push(text);
    return;
  }

  if (!(node instanceof Element)) return;
  if (SKIP_TAGS.has(node.tagName)) return;
  if (node.hasAttribute("data-agent-window")) return;
  if (!isPerceivable(node)) return;

  if (node instanceof HTMLInputElement) {
    const text =
      extractLabel(node) ||
      extractValue(node) ||
      getFallbackLabel(node, "native");
    if (text) segments.push(text);
    return;
  }

  if (
    node instanceof HTMLTextAreaElement ||
    node instanceof HTMLSelectElement
  ) {
    const text =
      extractLabel(node) ||
      extractValue(node) ||
      getFallbackLabel(node, "native");
    if (text) segments.push(text);
    return;
  }

  if (isMonacoEditor(node)) {
    const content = getMonacoContent(node);
    if (content)
      segments.push(truncateText(content, MAX_EDITOR_PREVIEW_LENGTH));
    return;
  }

  const beforeChildSegments = segments.length;
  Array.from(node.childNodes).forEach((child) => {
    collectSemanticTextSegments(child, segments);
  });

  if (segments.length > beforeChildSegments) return;

  const interactiveReason = getInteractiveReason(node);
  if (!interactiveReason) return;

  const text =
    extractLabel(node) ||
    getFallbackLabel(node, interactiveReason) ||
    extractValue(node);
  if (text) segments.push(text);
}

function extractSemanticText(node: Node): string | undefined {
  const segments: string[] = [];
  collectSemanticTextSegments(node, segments);
  const uniqueSegments: string[] = [];
  const seen = new Set<string>();
  for (const segment of segments) {
    if (seen.has(segment)) continue;
    seen.add(segment);
    uniqueSegments.push(segment);
  }

  if (node instanceof Element && uniqueSegments.length === 0) {
    const interactiveReason = getInteractiveReason(node);
    if (interactiveReason) {
      const label =
        extractLabel(node) || getFallbackLabel(node, interactiveReason);
      const value = extractValue(node);
      const semanticText = normalizeTextContent(label || value || "");
      if (semanticText) return semanticText;
    }
  }

  return normalizeTextContent(uniqueSegments.join(" "));
}

function walkTableActionableDescendants(
  node: Node,
  depth: number,
  context: WalkContext,
  path: string
): void {
  if (!(node instanceof Element)) return;
  if (SKIP_TAGS.has(node.tagName)) return;
  if (node.hasAttribute("data-agent-window")) return;
  if (!isPerceivable(node)) return;
  if (
    TABLE_SECTION_TAGS.has(node.tagName) ||
    node.tagName === "TR" ||
    TABLE_CELL_TAGS.has(node.tagName)
  ) {
    Array.from(node.childNodes).forEach((child, index) => {
      walkTableActionableDescendants(
        child,
        depth,
        context,
        `${path}/${node.tagName.toLowerCase()}[${index}]`
      );
    });
    return;
  }

  if (isMonacoEditor(node)) {
    walkDomNode(node, depth, context, path);
    return;
  }

  const interactiveReason = getInteractiveReason(node);
  if (interactiveReason) {
    const value = extractValue(node);
    const label =
      extractLabel(node) || getFallbackLabel(node, interactiveReason);
    if (
      shouldIndexInteractiveNode(node, interactiveReason, label, value, context)
    ) {
      walkDomNode(node, depth, context, path);
      return;
    }
  }

  Array.from(node.childNodes).forEach((child, index) => {
    walkTableActionableDescendants(
      child,
      depth,
      context,
      `${path}/${node.tagName.toLowerCase()}[${index}]`
    );
  });
}

function walkTableNode(
  table: Element,
  depth: number,
  context: WalkContext,
  path: string
): void {
  pushContextLine(
    depth,
    "<table>",
    context,
    `${path}:table`,
    table,
    undefined,
    "table"
  );

  const sectionChildren = Array.from(table.children).filter(
    (child): child is Element =>
      child instanceof Element && isPerceivable(child)
  );

  const rows = sectionChildren.flatMap((child) => {
    if (TABLE_SECTION_TAGS.has(child.tagName)) {
      return Array.from(child.children).filter(
        (row): row is Element =>
          row instanceof Element && row.tagName === "TR" && isPerceivable(row)
      );
    }
    return child.tagName === "TR" ? [child] : [];
  });

  rows.forEach((row, rowIndex) => {
    const cells = Array.from(row.children).filter(
      (cell): cell is Element =>
        TABLE_CELL_TAGS.has(cell.tagName) && isPerceivable(cell)
    );
    if (cells.length === 0) return;

    const cellTexts = cells.map((cell) => extractSemanticText(cell) ?? "");
    const rowLabel =
      normalizeTextContent(cellTexts.join(" | ")) ?? extractLabel(row);
    const isHeaderRow =
      row.parentElement?.tagName === "THEAD" ||
      cells.every((cell) => cell.tagName === "TH");

    if (isHeaderRow) {
      if (rowLabel) {
        pushContextLine(
          depth + 1,
          `<thead>${rowLabel}</thead>`,
          context,
          `${path}:thead:${rowLabel}`,
          row,
          normalizeLineSignature(rowLabel),
          "thead"
        );
      }
      return;
    }

    const interactiveReason = getInteractiveReason(row);
    let childContext = context;
    if (interactiveReason) {
      const value = extractValue(row);
      const label =
        rowLabel ||
        extractLabel(row) ||
        getFallbackLabel(row, interactiveReason);
      if (
        shouldIndexInteractiveNode(
          row,
          interactiveReason,
          label,
          value,
          context
        )
      ) {
        const entry = createIndexedElement({
          tag: row.tagName.toLowerCase(),
          role: row.getAttribute("role") ?? undefined,
          label,
          value,
          element: row,
        });
        pushInteractiveLine(depth + 1, entry, value, context);
        childContext = {
          ...context,
          interactiveLabels: label
            ? [...context.interactiveLabels, label]
            : context.interactiveLabels,
          hasInteractiveAncestor: true,
        };
      }
    }

    if (childContext === context && rowLabel) {
      pushContextLine(
        depth + 1,
        `<tr>${rowLabel}</tr>`,
        context,
        `${path}:tr:${rowIndex}:${rowLabel}`,
        row,
        normalizeLineSignature(rowLabel),
        "tr"
      );
    }

    cells.forEach((cell, cellIndex) => {
      Array.from(cell.childNodes).forEach((child, childIndex) => {
        walkTableActionableDescendants(
          child,
          depth + 2,
          childContext,
          `${path}/tr[${rowIndex}]/${cell.tagName.toLowerCase()}[${cellIndex}]/child[${childIndex}]`
        );
      });
    });
  });
}

function walkDomNode(
  node: Node,
  depth: number,
  context: WalkContext,
  path: string
): void {
  if (node instanceof Text) {
    pushTextLine(node, depth, context);
    return;
  }

  if (!(node instanceof Element)) return;
  if (SKIP_TAGS.has(node.tagName)) return;
  if (node.hasAttribute("data-agent-window")) return;
  if (!isPerceivable(node)) return;

  if (node.tagName === "TABLE") {
    walkTableNode(node, depth, context, path);
    return;
  }

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

  const interactiveReason = getInteractiveReason(node);
  const isInteractiveNode = Boolean(interactiveReason);
  let nextInteractiveLabels = context.interactiveLabels;
  let hasInteractiveAncestor = context.hasInteractiveAncestor;
  if (interactiveReason) {
    const tag = node.tagName.toLowerCase();
    const value = extractValue(node);
    const rawLabel = extractLabel(node);
    const label = rawLabel || getFallbackLabel(node, interactiveReason);

    if (
      shouldIndexInteractiveNode(node, interactiveReason, label, value, context)
    ) {
      const role = node.getAttribute("role") ?? undefined;
      const entry = createIndexedElement({
        tag,
        role,
        label,
        value,
        element: node,
      });

      pushInteractiveLine(depth, entry, value, context);
      nextInteractiveLabels = label
        ? [...context.interactiveLabels, label]
        : context.interactiveLabels;
      hasInteractiveAncestor = true;
    }

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
    hasInteractiveAncestor,
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

const WRAPPER_TAGS = new Set(["div", "span"]);
const SEMANTIC_LINE_TAG_SCORES = new Map([
  ["table", 50],
  ["thead", 80],
  ["tbody", 60],
  ["tfoot", 60],
  ["tr", 90],
  ["th", 85],
  ["td", 80],
  ["ul", 55],
  ["ol", 55],
  ["li", 88],
  ["button", 85],
  ["a", 85],
  ["input", 85],
  ["select", 85],
  ["textarea", 85],
  ["editor", 85],
  ["summary", 82],
]);

function getLineValueScore(line: DomLine): number {
  const kindScore =
    line.kind === "interactive" ? 200 : line.kind === "context" ? 120 : 40;
  const tagScore = line.semanticTag
    ? (SEMANTIC_LINE_TAG_SCORES.get(line.semanticTag) ??
      (WRAPPER_TAGS.has(line.semanticTag) ? 0 : 20))
    : 0;
  return kindScore + tagScore;
}

function canDropEquivalentLine(line: DomLine): boolean {
  if (line.kind !== "interactive") return true;
  return WRAPPER_TAGS.has(line.semanticTag ?? "") && !line.entry?.value;
}

function isAncestralDuplicate(keeper: DomLine, candidate: DomLine): boolean {
  if (!keeper.signature || !candidate.signature) return false;
  if (!keeper.sourceElement || !candidate.sourceElement) return false;
  if (
    keeper.sourceElement !== candidate.sourceElement &&
    !keeper.sourceElement.contains(candidate.sourceElement) &&
    !candidate.sourceElement.contains(keeper.sourceElement)
  ) {
    return false;
  }

  if (keeper.signature === candidate.signature) {
    return true;
  }

  if (
    candidate.kind === "interactive" &&
    !WRAPPER_TAGS.has(candidate.semanticTag ?? "")
  ) {
    return false;
  }

  const keeperSegments = splitStructuredSignature(keeper.signature);
  if (keeperSegments.length <= 1) return false;

  const candidateSegments = splitStructuredSignature(candidate.signature);
  return (
    candidateSegments.length === 1 &&
    keeperSegments.includes(candidateSegments[0])
  );
}

function dedupeEquivalentAncestorLines(lines: DomLine[]): DomLine[] {
  return lines.filter((line, index) => {
    if (!canDropEquivalentLine(line)) return true;

    return !lines.some((other, otherIndex) => {
      if (index === otherIndex) return false;
      if (!isAncestralDuplicate(other, line)) return false;

      const otherScore = getLineValueScore(other);
      const lineScore = getLineValueScore(line);
      if (otherScore !== lineScore) {
        return otherScore > lineScore;
      }

      const otherIsCloserSemanticAncestor = Boolean(
        other.sourceElement &&
          line.sourceElement &&
          other.sourceElement.contains(line.sourceElement) &&
          other.semanticTag &&
          !WRAPPER_TAGS.has(other.semanticTag)
      );
      if (otherIsCloserSemanticAncestor) {
        return true;
      }

      return otherIndex < index;
    });
  });
}

function selectBudgetedLines(lines: DomLine[]): DomLine[] {
  const selected = new Set<number>();
  let usedLines = 0;
  let usedChars = 0;

  const trySelect = (index: number): void => {
    if (selected.has(index)) return;
    const line = lines[index];
    const cost = line.text.length + (usedLines > 0 ? 1 : 0);
    if (
      usedLines >= MAX_DOM_TREE_LINES ||
      usedChars + cost > MAX_DOM_TREE_CHARS
    ) {
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
    hasInteractiveAncestor: false,
  };

  Array.from(rootEl.childNodes).forEach((child, index) => {
    walkDomNode(child, 0, context, `root[${index}]`);
  });

  const keptLines = selectBudgetedLines(
    dedupeEquivalentAncestorLines(dedupeAdjacentLines(lines))
  );
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
