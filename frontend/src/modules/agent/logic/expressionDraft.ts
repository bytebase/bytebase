import type { Code } from "mdast";
import remarkParse from "remark-parse";
import { unified } from "unified";

export type AgentExpressionDraftMode = "selector" | "raw";
export type AgentExpressionDraftTarget = (
  expression: string,
  mode: AgentExpressionDraftMode
) => Promise<boolean>;

interface RegisteredExpressionTarget {
  apply: AgentExpressionDraftTarget;
  allowRaw: boolean;
}

const MAX_EXPRESSION_LENGTH = 20_000;
const markdown = unified().use(remarkParse);
const expressionTargets = new Map<string, RegisteredExpressionTarget>();
const targetListeners = new Set<() => void>();

function notifyTargetListeners() {
  for (const listener of targetListeners) listener();
}

export function extractAgentExpressionDraft(
  answer: string
): string | undefined {
  const blocks = markdown
    .parse(answer)
    .children.filter(
      (node): node is Code =>
        node.type === "code" && node.lang?.toLowerCase() === "cel"
    );
  if (blocks.length !== 1) return undefined;
  const expression = blocks[0].value.trim();
  if (
    !expression ||
    expression.length > MAX_EXPRESSION_LENGTH ||
    expression.includes("\0")
  ) {
    return undefined;
  }
  return expression;
}

export function registerAgentExpressionTarget(
  chatId: string,
  target: AgentExpressionDraftTarget,
  options: { allowRaw?: boolean } = {}
): () => void {
  const registered = { apply: target, allowRaw: options.allowRaw ?? true };
  expressionTargets.set(chatId, registered);
  notifyTargetListeners();
  return () => {
    if (expressionTargets.get(chatId) === registered) {
      expressionTargets.delete(chatId);
      notifyTargetListeners();
    }
  };
}

export function subscribeToAgentExpressionTargets(listener: () => void) {
  targetListeners.add(listener);
  return () => {
    targetListeners.delete(listener);
  };
}

export function canApplyAgentExpressionDraft(
  chatId: string,
  mode: AgentExpressionDraftMode
): boolean {
  const target = expressionTargets.get(chatId);
  return Boolean(target && (mode === "selector" || target.allowRaw));
}

export async function applyAgentExpressionDraft(
  chatId: string,
  expression: string,
  mode: AgentExpressionDraftMode
): Promise<boolean> {
  const target = expressionTargets.get(chatId);
  if (!target || (mode === "raw" && !target.allowRaw)) return false;
  return target.apply(expression, mode);
}
