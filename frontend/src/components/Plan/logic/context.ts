import type Emittery from "emittery";
import { v4 as uuidv4 } from "uuid";
import type { InjectionKey, Ref } from "vue";
import { inject, provide } from "vue";
import type { Plan, PlanCheckRun } from "@/types/proto/v1/plan_service";

export type PlanEvents = Emittery<{
  "status-changed": { eager: boolean };
}>;

export type PlanContext = {
  // Basic fields
  isCreating: Ref<boolean>;
  plan: Ref<Plan>;
  planCheckRunList: Ref<PlanCheckRun[]>;
  // TODO(steven): save related issue/rollout for checking if the plan is changable.

  // UI events
  events: PlanEvents;
};

const KEY = Symbol(`bb.plan.context.${uuidv4()}`) as InjectionKey<PlanContext>;

export const usePlanContext = () => {
  return inject(KEY)!;
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
};
