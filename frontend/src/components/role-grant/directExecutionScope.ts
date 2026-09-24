import type { ConditionExpression } from "@/utils/issue/cel";

/**
 * Where a binding lets its members run DDL/DML straight from SQL Editor,
 * as the grant condition's environment clause says it.
 *
 * - `some`: `resource.environment_id in [...]` with entries.
 * - `none`: the clause with an empty list — the switch off; the binding
 *   adds no direct execution anywhere.
 * - `all`: no clause at all — workspace-level grants, API-created bindings,
 *   and bindings the 3.15 migration left alone.
 * - `custom`: an expression the forms could not have written; the raw text
 *   is shown because no summary of it can be trusted.
 */
export type DirectExecutionScope =
  | { type: "some"; environments: string[] }
  | { type: "none" }
  | { type: "all" }
  | { type: "custom"; expression: string };

export const directExecutionScopeFromCondition = (
  condition: ConditionExpression | undefined,
  expression: string
): DirectExecutionScope => {
  if (condition?.unrecognized) {
    return { type: "custom", expression };
  }
  if (condition?.environments === undefined) {
    return { type: "all" };
  }
  if (condition.environments.length === 0) {
    return { type: "none" };
  }
  return { type: "some", environments: condition.environments };
};
