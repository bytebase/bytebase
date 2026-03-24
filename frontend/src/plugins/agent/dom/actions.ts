export { extractDomTree, getElementByRef } from "./domTree";

export type DomActionType = "click" | "input" | "select" | "scroll" | "read";

export interface DomActionParams {
  type: DomActionType;
  index: string | number;
  value?: string;
}

export interface DomActionResult {
  success: boolean;
  message: string;
}

function getCenter(el: Element): { x: number; y: number } {
  const rect = el.getBoundingClientRect();
  return {
    x: rect.left + rect.width / 2,
    y: rect.top + rect.height / 2,
  };
}

function dispatchClick(el: Element): void {
  const { x, y } = getCenter(el);
  const shared: MouseEventInit = {
    bubbles: true,
    cancelable: true,
    clientX: x,
    clientY: y,
    view: window,
  };
  el.dispatchEvent(new MouseEvent("mousedown", shared));
  el.dispatchEvent(new MouseEvent("mouseup", shared));
  el.dispatchEvent(new MouseEvent("click", shared));
}

function setNativeValue(
  el: HTMLInputElement | HTMLTextAreaElement,
  value: string
): void {
  const proto =
    el instanceof HTMLTextAreaElement
      ? HTMLTextAreaElement.prototype
      : HTMLInputElement.prototype;
  const setter = Object.getOwnPropertyDescriptor(proto, "value")?.set;
  if (setter) {
    setter.call(el, value);
  } else {
    el.value = value;
  }
  el.dispatchEvent(new Event("input", { bubbles: true }));
  el.dispatchEvent(new Event("change", { bubbles: true }));
}

function normalizeMultilineValue(value: string): string {
  if (!value.includes("\\")) {
    return value;
  }

  return value
    .replace(/\\r\\n/g, "\n")
    .replace(/\\n/g, "\n")
    .replace(/\\r/g, "\r")
    .replace(/\\t/g, "\t");
}

function findInnerInput(
  el: Element
): HTMLInputElement | HTMLTextAreaElement | null {
  return el.querySelector<HTMLInputElement | HTMLTextAreaElement>(
    "input, textarea"
  );
}

async function findMonacoEditor(
  el: Element
): Promise<{ getValue(): string; setValue(v: string): void } | null> {
  try {
    const { isMonacoLoaded, getMonacoEditor } = await import(
      "@/components/MonacoEditor/lazy-editor"
    );
    if (!isMonacoLoaded()) return null;
    const monaco = await getMonacoEditor();
    const monacoRoot = el.closest(".monaco-editor");
    for (const editor of monaco.editor.getEditors()) {
      const domNode = editor.getDomNode();
      if (!domNode) {
        continue;
      }
      if (
        el === domNode ||
        el.contains(domNode) ||
        domNode.contains(el) ||
        monacoRoot === domNode
      ) {
        return editor;
      }
    }
  } catch {
    // Monaco not available
  }
  return null;
}

async function handleClick(el: Element): Promise<DomActionResult> {
  dispatchClick(el);
  return { success: true, message: `Clicked ${el.tagName.toLowerCase()}` };
}

async function handleInput(
  el: Element,
  value: string
): Promise<DomActionResult> {
  // Monaco editor
  const monacoEditor = await findMonacoEditor(el);
  if (monacoEditor) {
    const normalizedValue = normalizeMultilineValue(value);
    monacoEditor.setValue(normalizedValue);
    return {
      success: true,
      message: `Set editor content (${normalizedValue.length} chars)`,
    };
  }

  let target: HTMLInputElement | HTMLTextAreaElement | null = null;

  if (el instanceof HTMLInputElement || el instanceof HTMLTextAreaElement) {
    target = el;
  } else {
    // Naive UI wraps inputs — find the inner element
    target = findInnerInput(el);
  }

  if (!target) {
    return {
      success: false,
      message: `No input element found within [${el.tagName.toLowerCase()}]`,
    };
  }

  target.focus();
  const normalizedValue =
    target instanceof HTMLTextAreaElement
      ? normalizeMultilineValue(value)
      : value;
  setNativeValue(target, normalizedValue);
  return { success: true, message: `Set value to "${normalizedValue}"` };
}

async function handleSelect(
  el: Element,
  value: string
): Promise<DomActionResult> {
  // Native <select>
  if (el instanceof HTMLSelectElement) {
    el.value = value;
    el.dispatchEvent(new Event("change", { bubbles: true }));
    return { success: true, message: `Selected "${value}"` };
  }

  // Naive UI select — click to open, then pick option
  dispatchClick(el);

  return new Promise<DomActionResult>((resolve) => {
    setTimeout(() => {
      const options = document.querySelectorAll(".n-base-select-option");
      for (const opt of Array.from(options)) {
        const text = opt.textContent?.trim();
        if (text === value) {
          dispatchClick(opt);
          resolve({ success: true, message: `Selected "${value}"` });
          return;
        }
      }
      resolve({
        success: false,
        message: `Option "${value}" not found. Available options: ${Array.from(
          options
        )
          .map((o) => o.textContent?.trim())
          .filter(Boolean)
          .join(", ")}`,
      });
    }, 200);
  });
}

async function handleRead(el: Element): Promise<DomActionResult> {
  // Monaco editor — read full content via API
  const monacoEditor = await findMonacoEditor(el);
  if (monacoEditor) {
    return { success: true, message: monacoEditor.getValue() };
  }

  // Standard inputs
  if (el instanceof HTMLInputElement || el instanceof HTMLTextAreaElement) {
    return { success: true, message: el.value };
  }

  const inner = findInnerInput(el);
  if (inner) {
    return { success: true, message: inner.value };
  }

  // Fallback: text content
  return { success: true, message: el.textContent?.trim() ?? "" };
}

async function handleScroll(el: Element): Promise<DomActionResult> {
  el.scrollIntoView({ behavior: "smooth", block: "center" });
  return { success: true, message: "Scrolled element into view" };
}

export async function executeDomAction(
  params: DomActionParams,
  element: Element
): Promise<DomActionResult> {
  switch (params.type) {
    case "click":
      return handleClick(element);
    case "input":
      return handleInput(element, params.value ?? "");
    case "select":
      return handleSelect(element, params.value ?? "");
    case "read":
      return handleRead(element);
    case "scroll":
      return handleScroll(element);
    default:
      return {
        success: false,
        message: `Unknown action type: ${params.type}`,
      };
  }
}
