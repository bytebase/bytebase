import type Emittery from "emittery";
import { v4 as uuidv4 } from "uuid";
import type { InjectionKey, Ref } from "vue";
import { inject, provide } from "vue";
import type { Issue } from "@/types/proto/v1/issue_service";
import type { Plan, PlanCheckRun } from "@/types/proto/v1/plan_service";
import type { Rollout } from "@/types/proto/v1/rollout_service";
import type {
  IssueReviewAction,
  IssueStatusAction,
} from "../components/HeaderSection/Actions/unified";

export type PlanEvents = Emittery<{
  "status-changed": { eager?: boolean };
  "perform-issue-review-action": {
    action: IssueReviewAction;
  };
  "perform-issue-status-action": { action: IssueStatusAction };
  "resource-refresh-completed": {
    resources: string[];
    isManual: boolean;
  };
}>;

export type PlanContext = {
  // Basic fields
  isCreating: Ref<boolean>;
  plan: Ref<Plan>;
  planCheckRuns: Ref<PlanCheckRun[]>;
  issue?: Ref<Issue | undefined>;
  rollout?: Ref<Rollout | undefined>;

  // UI events
  events: PlanEvents;
};

const KEY = Symbol(`bb.plan.context.${uuidv4()}`) as InjectionKey<PlanContext>;

export const usePlanContext = () => {
  const context = inject(KEY);
  if (!context) {
    throw new Error(
      "usePlanContext must be called within a component that provides PlanContext"
    );
  }
  return context;
};

export const usePlanContextWithIssue = () => {
  const context = inject(KEY)!;
  if (!context.issue?.value) {
    throw new Error("Issue is required but not available in plan context");
  }
  return {
    ...context,
    issue: context.issue as Ref<Issue>,
  };
};

export const usePlanContextWithRollout = () => {
  const context = inject(KEY)!;
  if (!context.rollout?.value) {
    throw new Error("Rollout is required but not available in plan context");
  }
  return {
    ...context,
    rollout: context.rollout as Ref<Rollout>,
  };
};

export const providePlanContext = (
  context: Partial<PlanContext>,
  root = false
) => {
  if (!root) {
    const parent = usePlanContext();
    context = {
      ...parent,
      ...context,
    };
  }
  provide(KEY, context as PlanContext);
  return context as PlanContext;
};
