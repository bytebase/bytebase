import type { Root } from "mdast";
import { type CustomSlots, mdastToReact, type State } from "./utils";

type Props = {
  readonly ast: Root;
  readonly slots?: CustomSlots;
};

/**
 * Walks the mdast root via `mdastToReact[node.type]` and returns the
 * React tree. `slots` overrides the default renderer for specific node
 * types (`code`, `inlineCode`, `image`).
 */
export function AstToReact({ ast, slots = {} }: Props) {
  const state: State = {
    slots,
    definitionById: new Map(),
  };
  return <>{mdastToReact[ast.type](ast, state)}</>;
}
