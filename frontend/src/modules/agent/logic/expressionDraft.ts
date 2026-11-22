import type { Code } from "mdast";
import remarkParse from "remark-parse";
import { unified } from "unified";
import { useAgentStore } from "../store/agent";

export type AgentExpressionDraftMode = "selector" | "raw";
export type AgentExpressionDraftTarget = (
  expression: string,
  mode: AgentExpressionDraftMode
) => Promise<boolean>;

const MAX_EXPRESSION_LENGTH = 20_000;
const markdown = unified().use(remarkParse);

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
  return useAgentStore
    .getState()
    .registerExpressionDraftTarget(chatId, target, options);
}

export function canApplyAgentExpressionDraft(
  chatId: string,
  mode: AgentExpressionDraftMode
): boolean {
  return useAgentStore.getState().canApplyExpressionDraft(chatId, mode);
}

export async function applyAgentExpressionDraft(
  chatId: string,
  expression: string,
  mode: AgentExpressionDraftMode
): Promise<boolean> {
  return useAgentStore
    .getState()
    .applyExpressionDraft(chatId, expression, mode);
}
