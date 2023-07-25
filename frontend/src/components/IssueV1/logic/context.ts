import { InjectionKey, Ref, inject, provide } from "vue";
import { useDialog } from "naive-ui";
import Emittery from "emittery";

import { ComposedIssue, ReviewFlow } from "@/types";
import { Stage, Task } from "@/types/proto/v1/rollout_service";
import { Issue_Approver_Status } from "@/types/proto/v1/issue_service";
import { IssueStatusAction, TaskRolloutAction } from "./action";

export type IssuePhase = "CREATE" | "REVIEW" | "ROLLOUT";

export type IssueEvents = Emittery<{
  "status-changed": { eager: boolean };
  "select-task": { task: Task };
  "perform-issue-status-action": { action: IssueStatusAction };
  "perform-task-rollout-action": { action: TaskRolloutAction; tasks: Task[] };
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
  isPITRMode: Ref<boolean>;
  isTenantMode: Ref<boolean>;

  // review status
  reviewContext: ReviewContext;

  // rollout status
  activeStage: Ref<Stage>;
  activeTask: Ref<Task>;

  // UI status
  selectedStage: Ref<Stage>;
  selectedTask: Ref<Task>;

  // UI events
  events: IssueEvents;

  // misc
  dialog: ReturnType<typeof useDialog>;
};

const KEY = Symbol("bb.issue.context") as InjectionKey<IssueContext>;

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
