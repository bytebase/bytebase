import type { ActionDefinition } from "../types";
import { issueActions } from "./issue";
import { planActions } from "./plan";
import { rolloutActions } from "./rollout";

// All actions in priority order for filtering
export const allActions: ActionDefinition[] = [
  ...planActions,
  ...issueActions,
  ...rolloutActions,
].sort((a, b) => a.priority - b.priority);

// Re-export individual actions for testing
export * from "./plan";
export * from "./issue";
export * from "./rollout";
