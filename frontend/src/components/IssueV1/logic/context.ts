import type Emittery from "emittery";
import type { useDialog } from "naive-ui";
import { v4 as uuidv4 } from "uuid";
import type { InjectionKey, Ref, ComputedRef } from "vue";
import { inject, provide } from "vue";
import type { ComposedIssue, ReviewFlow } from "@/types";
import { type User } from "@/types/proto/v1/auth_service";
import type { Issue_Approver_Status } from "@/types/proto/v1/issue_service";
import type { PlanCheckRun } from "@/types/proto/v1/plan_service";
import type { Stage, Task } from "@/types/proto/v1/rollout_service";
import type {
  IssueReviewAction,
  IssueStatusAction,
  TaskRolloutAction,
} from "./action";

export type IssuePhase = "CREATE" | "REVIEW" | "ROLLOUT";

export type IssueEvents = Emittery<{
  "status-changed": { eager: boolean };
  "select-task": { task: Task };
  "perform-issue-review-action": { action: IssueReviewAction };
  "perform-issue-status-action": { action: IssueStatusAction };
  "perform-task-rollout-action": { action: TaskRolloutAction; tasks: Task[] };
  "toggle-online-migration": { on: boolean };
}>;

export type ReviewContext = {
  // true if the approval flow is generated
  ready: Ref<boolean>;
  // The review flow.
  // Now we have only one flow in an issue
  flow: Ref<ReviewFlow>;
  // The overall status of the entire review flow
  status: Ref<Issue_Approver_Status>;
  // Whether the review flow is finished successfully.
  // A shortcut to `status === Review_Approver_Status.APPROVED`
  done: Ref<boolean>;
  // Whether the review finding has error.
  error: Ref<string | undefined>;
};

export type IssueContext = {
  // Basic fields
  isCreating: Ref<boolean>;
  ready: Ref<boolean>;
  issue: Ref<ComposedIssue>;
  phase: Ref<IssuePhase>;
  isGhostMode: Ref<boolean>;
  isLegacyIssue: Ref<boolean>;
  releaserCandidates: Ref<User[]>;
  allowEditIssue: ComputedRef<boolean>;

  // review status
  reviewContext: ReviewContext;

  // UI status
  selectedStage: Ref<Stage>;
  selectedTask: Ref<Task>;
  formatOnSave: Ref<boolean>;

  // UI events
  events: IssueEvents;

  // misc
  dialog: ReturnType<typeof useDialog>;
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

export const provideIssueContext = (
  context: Partial<IssueContext>,
  root = false
) => {
  if (!root) {
    const parent = useIssueContext();
    context = {
      ...parent,
      ...context,
    };
  }
  provide(KEY, context as IssueContext);
};
