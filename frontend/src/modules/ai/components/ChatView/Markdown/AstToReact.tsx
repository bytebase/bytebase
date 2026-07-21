import type { Root } from "mdast";
import { type CustomSlots, mdastToReact, type State } from "./utils";

type Props = {
  readonly ast: Root;
  readonly slots?: CustomSlots;
};

/**
 * React port of `plugins/ai/components/ChatView/Markdown/AstToVNode.vue`.
 * Walks the mdast root via `mdastToReact[node.type]` and returns the
 * React tree. `slots` overrides the default renderer for specific node
 * types (`code`, `inlineCode`, `image`) — same hook the Vue version
 * exposed via `defineSlots`.
 */
export function AstToReact({ ast, slots = {} }: Props) {
  const state: State = {
    slots,
    definitionById: new Map(),
  };
  return <>{mdastToReact[ast.type](ast, state)}</>;
}
