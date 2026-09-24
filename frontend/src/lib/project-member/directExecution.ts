import type { ConditionExpression } from "@/utils/issue/cel";
import {
  type EnvLimitationKind,
  getRoleEnvironmentLimitationKind,
} from "./utils";

/**
 * Where a role assignment lets its members run DDL/DML in SQL Editor without
 * approval. `all` is a condition with no environment clause, as the workspace
 * member sheet writes.
 */
export type DirectExecutionScope =
  | { type: "some"; environments: string[] }
  | { type: "none" }
  | { type: "all" };

const directExecutionScopeFromCondition = (
  condition: ConditionExpression | undefined
): DirectExecutionScope => {
  if (condition?.environments === undefined) {
    return { type: "all" };
  }
  if (condition.environments.length === 0) {
    return { type: "none" };
  }
  return { type: "some", environments: condition.environments };
};

/**
 * The form's view of the field. `enabled` is kept beside the list rather
 * than derived from it: off and on-with-nothing-picked serialize to the
 * same empty clause, and only the form knows which one the reader meant.
 */
export interface DirectExecutionValue {
  enabled: boolean;
  environments: string[];
}

export const EMPTY_DIRECT_EXECUTION: DirectExecutionValue = {
  enabled: false,
  environments: [],
};

export const isDirectExecutionValid = (value: DirectExecutionValue): boolean =>
  !value.enabled || value.environments.length > 0;

/** What the binding's environment clause should carry; off is the empty list. */
export const directExecutionEnvironments = (
  value: DirectExecutionValue
): string[] => (value.enabled ? value.environments : []);

// What a role's binding lets its members run directly, from the environments
// its condition proves (`readableCondition`).
export const directExecutionOf = (
  role: string,
  condition: ConditionExpression | undefined
): { kind: EnvLimitationKind; scope: DirectExecutionScope } | undefined => {
  const kind = getRoleEnvironmentLimitationKind(role);
  return kind
    ? { kind, scope: directExecutionScopeFromCondition(condition) }
    : undefined;
};
