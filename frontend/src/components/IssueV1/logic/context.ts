import { ComputedRef, InjectionKey, Ref, inject, provide } from "vue";
import { useDialog } from "naive-ui";
import Emittery from "emittery";

import { ComposedIssue } from "@/types";
import { IssueReviewContext } from "@/plugins/issue/logic/review/context";
import { Stage, Task } from "@/types/proto/v1/rollout_service";

export type IssuePhase = "CREATE" | "REVIEW" | "ROLLOUT";

export type IssueEvents = Emittery<{
  "status-changed": { eager: boolean };
  "select-task": { task: Task };
}>;

export type IssueContext = {
  // Basic fields
  isCreating: Ref<boolean>;
  ready: Ref<boolean>;
  issue: Ref<ComposedIssue>;
  phase: Ref<IssuePhase>;

  // review status
  reviewContext: IssueReviewContext;

  // rollout status
  activeStage: ComputedRef<Stage>;
  activeTask: ComputedRef<Task>;

  // UI status
  selectedStage: ComputedRef<Stage>;
  selectedTask: ComputedRef<Task>;

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
