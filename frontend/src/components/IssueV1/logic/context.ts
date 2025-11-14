import type Emittery from "emittery";
import { v4 as uuidv4 } from "uuid";
import type { ComputedRef, InjectionKey, Ref } from "vue";
import { inject, provide } from "vue";
import type { ComposedIssue } from "@/types";
import type { PlanCheckRun } from "@/types/proto-es/v1/plan_service_pb";
import type { Stage, Task } from "@/types/proto-es/v1/rollout_service_pb";
import type {
  IssueReviewAction,
  IssueStatusAction,
  TaskRolloutAction,
} from "./action";

export type IssueEvents = Emittery<{
  "status-changed": { eager: boolean };
  "select-task": { task: Task };
  "perform-issue-review-action": { action: IssueReviewAction };
  "perform-issue-status-action": { action: IssueStatusAction };
  "perform-task-rollout-action": { action: TaskRolloutAction; tasks: Task[] };
}>;

export type IssueContext = {
  // Basic fields
  isCreating: Ref<boolean>;
  ready: Ref<boolean>;
  issue: Ref<ComposedIssue>;
  allowChange: ComputedRef<boolean>;

  // UI status
  selectedStage: Ref<Stage>;
  selectedTask: Ref<Task>;

  // UI events
  events: IssueEvents;

  // misc
  reInitialize: (overrides?: Record<string, string>) => Promise<void>;

  // utility functions
  getPlanCheckRunsForTask: (task: Task) => PlanCheckRun[];
};

const KEY = Symbol(
  `bb.issue.context.${uuidv4()}`
) as InjectionKey<IssueContext>;

export const useIssueContext = () => {
  return inject(KEY)!;
};

export const provideIssueContext = (context: Partial<IssueContext>) => {
  provide(KEY, context as IssueContext);
};
