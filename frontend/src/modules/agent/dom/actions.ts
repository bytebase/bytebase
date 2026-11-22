import { getElementByRef } from "./domTree";
import { containsExcluded, isPerceivable, visibleText } from "./guards";
import {
  type AgentDomContext,
  type AgentDomPreparation,
  matchesDomScope,
  prepareCapturedInteraction,
} from "./interactionRegistry";

export type { DomRefSuggestion } from "./domTree";
export {
  extractDomRefSuggestions,
  extractDomTree,
  getElementByRef,
} from "./domTree";

export type DomActionType = "click" | "input" | "select" | "scroll" | "read";
export interface DomActionParams {
  type: DomActionType;
  ref: string;
  value?: string;
}
export interface DomActionResult {
  success: boolean;
  message: string;
  code?: "manual-required";
}
export const BACKGROUND_PAGE_ACTION_RESULT: DomActionResult = {
  success: false,
  message:
    "This chat is not selected or the page has changed. Return to it before using the current page.",
};
const unavailable: DomActionResult = {
  success: false,
  message: "This target is unavailable or contains sensitive content.",
};

async function readEditor(
  element: Element,
  canRead: () => boolean
): Promise<DomActionResult | undefined> {
  if (!element.matches(".monaco-editor")) return;
  try {
    const { isMonacoLoaded, getMonacoEditor } = await import(
      "@/components/monaco/core"
    );
    if (!isMonacoLoaded() || !canRead()) return;
    const monaco = await getMonacoEditor();
    if (!canRead()) return unavailable;
    const editor = monaco.editor
      .getEditors()
      .find((editor) => editor.getDomNode() === element);
    if (editor) return { success: true, message: editor.getValue() };
  } catch {
    /* Rendered text remains available without the editor model. */
  }
}

async function inspectDomElement(
  type: "read" | "scroll",
  element: Element,
  canUseCurrentPage: () => boolean
): Promise<DomActionResult> {
  if (!canUseCurrentPage()) return BACKGROUND_PAGE_ACTION_RESULT;
  if (!isPerceivable(element) || containsExcluded(element)) return unavailable;
  if (type === "scroll") {
    element.scrollIntoView({ behavior: "smooth", block: "center" });
    return { success: true, message: "Scrolled element into view" };
  }
  const canRead = () =>
    canUseCurrentPage() && isPerceivable(element) && !containsExcluded(element);
  if (element.matches(".monaco-editor")) {
    const result = await readEditor(element, canRead);
    if (!canRead()) return unavailable;
    if (result) return result;
  }
  const input =
    element instanceof HTMLInputElement ||
    element instanceof HTMLTextAreaElement
      ? element
      : element.querySelector<HTMLInputElement | HTMLTextAreaElement>(
          "input,textarea"
        );
  if (input)
    return isPerceivable(input)
      ? { success: true, message: input.value }
      : unavailable;
  return { success: true, message: visibleText(element).trim() };
}

export async function prepareAgentDomInteraction(
  params: DomActionParams,
  element: Element,
  context: AgentDomContext
): Promise<AgentDomPreparation> {
  const entry = getElementByRef(params.ref);
  if (
    context.runMode !== "default" ||
    !entry ||
    entry.element !== element ||
    !matchesDomScope(entry.scope, context)
  )
    return { kind: "result", result: BACKGROUND_PAGE_ACTION_RESULT };
  const isCurrentRef = () => getElementByRef(params.ref) === entry;
  if (params.type === "read" || params.type === "scroll")
    return {
      kind: "result",
      result: await inspectDomElement(
        params.type,
        element,
        () => isCurrentRef() && matchesDomScope(entry.scope, context)
      ),
    };
  return prepareCapturedInteraction(params, entry, context);
}
