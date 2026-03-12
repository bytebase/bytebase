export interface IndexedElement {
  index: number;
  tag: string;
  role?: string;
  label: string;
  value?: string;
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

const SKIP_TAGS = new Set(["SCRIPT", "STYLE", "NOSCRIPT", "SVG", "PATH"]);

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

let elementRegistry: IndexedElement[] = [];

export function getElementByIndex(index: number): IndexedElement | undefined {
  return elementRegistry[index];
}

function isVisible(el: Element): boolean {
  const style = window.getComputedStyle(el);
  return (
    style.display !== "none" &&
    style.visibility !== "hidden" &&
    style.opacity !== "0"
  );
}

function isInteractive(el: Element): boolean {
  if (INTERACTIVE_TAGS.has(el.tagName)) return true;

  const role = el.getAttribute("role");
  if (role && INTERACTIVE_ROLES.has(role)) return true;

  for (const cls of NAIVE_INTERACTIVE_CLASSES) {
    if (el.classList.contains(cls)) return true;
  }

  return false;
}

function extractLabel(el: Element): string {
  // aria-label
  const ariaLabel = el.getAttribute("aria-label");
  if (ariaLabel) return ariaLabel.trim();

  // aria-labelledby
  const labelledBy = el.getAttribute("aria-labelledby");
  if (labelledBy) {
    const parts = labelledBy
      .split(/\s+/)
      .map((id) => document.getElementById(id)?.textContent?.trim())
      .filter(Boolean);
    if (parts.length > 0) return parts.join(" ");
  }

  // placeholder
  if (el instanceof HTMLInputElement || el instanceof HTMLTextAreaElement) {
    if (el.placeholder) return el.placeholder.trim();
  }

  // Naive UI form-item label
  const formItem = el.closest(".n-form-item");
  if (formItem) {
    const labelEl = formItem.querySelector(".n-form-item-label__text");
    if (labelEl?.textContent) return labelEl.textContent.trim();
  }

  // title
  const title = el.getAttribute("title");
  if (title) return title.trim();

  // text content (truncated)
  const text = (el.textContent ?? "").trim().replace(/\s+/g, " ");
  if (text.length > 80) return text.slice(0, 80) + "...";
  return text;
}

function extractValue(el: Element): string | undefined {
  if (el instanceof HTMLInputElement) {
    if (el.type === "checkbox" || el.type === "radio") {
      return el.checked ? "checked" : "unchecked";
    }
    return el.value || undefined;
  }
  if (el instanceof HTMLTextAreaElement) {
    return el.value || undefined;
  }
  if (el instanceof HTMLSelectElement) {
    return el.options[el.selectedIndex]?.text || el.value || undefined;
  }

  // Naive UI select
  if (el.classList.contains("n-base-selection")) {
    const label = el.querySelector(".n-base-selection-label");
    const text = label?.textContent?.trim();
    if (text) return text;
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

function walkDom(node: Element, depth: number, lines: string[]): void {
  if (SKIP_TAGS.has(node.tagName)) return;
  if (node.hasAttribute("data-agent-window")) return;
  if (!isVisible(node)) return;

  if (isInteractive(node)) {
    const index = elementRegistry.length;
    const tag = node.tagName.toLowerCase();
    const label = extractLabel(node);
    const value = extractValue(node);
    const role = node.getAttribute("role") ?? undefined;

    elementRegistry.push({ index, tag, role, label, value, element: node });

    const indent = "  ".repeat(depth);
    const valueAttr = value ? ` value="${value}"` : "";
    lines.push(`${indent}[${index}]<${tag}${valueAttr}>${label}</${tag}>`);

    // Don't recurse into interactive elements
    return;
  }

  // Only increment depth at landmark/semantic containers
  const childDepth = isLandmark(node) ? depth + 1 : depth;
  for (const child of Array.from(node.children)) {
    walkDom(child, childDepth, lines);
  }
}

export function extractDomTree(root?: Element): {
  tree: string;
  count: number;
} {
  elementRegistry = [];
  const lines: string[] = [];
  const rootEl = root ?? document.body;

  for (const child of Array.from(rootEl.children)) {
    walkDom(child, 0, lines);
  }

  return {
    tree: lines.join("\n"),
    count: elementRegistry.length,
  };
}
