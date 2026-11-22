import type {
  DomActionParams,
  DomActionResult,
  DomRefSuggestion,
} from "./actions";
import type {
  AgentDomContext,
  AgentDomPreparation,
} from "./interactionRegistry";

export type { DomActionParams, DomActionResult, DomRefSuggestion };

export async function lazyExtractDomTree(
  context?: AgentDomContext
): Promise<{ tree: string; count: number }> {
  const m = await import("./actions");
  return m.extractDomTree(undefined, context);
}
export async function lazyExtractDomRefSuggestions(
  context?: AgentDomContext
): Promise<DomRefSuggestion[]> {
  const m = await import("./actions");
  return m.extractDomRefSuggestions(undefined, context);
}
function refreshError(message: string): AgentDomPreparation {
  return {
    kind: "result",
    result: {
      success: false,
      message: `${message} Run get_page_state(mode="dom") to refresh the DOM tree.`,
    },
  };
}
export async function lazyPrepareDomAction(
  params: DomActionParams,
  context: AgentDomContext
): Promise<AgentDomPreparation> {
  const ref = typeof params.ref === "string" ? params.ref.trim() : "";
  if (!ref)
    return refreshError(
      "Invalid element ref. Use refs like [e1] from the DOM tree."
    );
  if (!/^e\d+$/.test(ref))
    return refreshError(
      "Malformed element ref. Use refs like [e1] from the DOM tree."
    );
  const m = await import("./actions");
  if (!context || !context.isCurrent() || context.signal?.aborted)
    return { kind: "result", result: m.BACKGROUND_PAGE_ACTION_RESULT };
  const entry = m.getElementByRef(ref);
  if (!entry || !entry.element.isConnected)
    return refreshError(`Element [${ref}] is stale or was not found.`);
  return m.prepareAgentDomInteraction(
    { ...params, ref },
    entry.element,
    context
  );
}
