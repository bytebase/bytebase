import type Emittery from "emittery";
import { v4 as uuidv4 } from "uuid";
import type { ComputedRef, InjectionKey, Ref } from "vue";
import { computed, inject, provide } from "vue";
import { useCurrentProjectV1, useCurrentUserV1 } from "@/store";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import type { Plan, PlanCheckRun } from "@/types/proto-es/v1/plan_service_pb";
import { type Project } from "@/types/proto-es/v1/project_service_pb";
import type { Rollout, TaskRun } from "@/types/proto-es/v1/rollout_service_pb";
import { hasProjectPermissionV2 } from "@/utils";
import type {
  IssueReviewAction,
  IssueStatusAction,
} from "../components/HeaderSection/Actions/registry";

export type PlanEvents = Emittery<{
  "status-changed": { eager: boolean };
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
  issue: Ref<Issue | undefined>;
  rollout: Ref<Rollout | undefined>;
  taskRuns: Ref<TaskRun[]>;

  readonly: ComputedRef<boolean>;
  allowEdit: ComputedRef<boolean>;
  project: ComputedRef<Project>;
  isCreator: ComputedRef<boolean>;

  // UI events
  events: PlanEvents;
};

type InitPlanContext = Omit<PlanContext, "allowEdit" | "project" | "isCreator">;

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

export const tryUsePlanContext = () => {
  return inject(KEY);
};

export const usePlanContextWithIssue = () => {
  const context = inject(KEY)!;
  if (!context.issue.value) {
    throw new Error("Issue is required but not available in plan context");
  }
  return {
    ...context,
    issue: context.issue as Ref<Issue>,
  };
};

export const usePlanContextWithRollout = () => {
  const context = inject(KEY)!;
  if (!context.rollout.value) {
    throw new Error("Rollout is required but not available in plan context");
  }
  return {
    ...context,
    rollout: context.rollout as Ref<Rollout>,
    taskRuns: context.taskRuns,
  };
};

export const providePlanContext = (context: InitPlanContext) => {
  const isCreator = computed(() => {
    const currentUser = useCurrentUserV1();
    return context.plan.value.creator === currentUser.value.name;
  });

  const { project } = useCurrentProjectV1();

  const allowEdit = computed(() => {
    if (isCreator.value) {
      return true;
    }
    return hasProjectPermissionV2(project.value, "bb.plans.update");
  });

  const planContext = {
    ...context,
    isCreator,
    project,
    allowEdit,
  } as PlanContext;

  provide(KEY, planContext);
  return planContext;
};
