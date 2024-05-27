import type Emittery from "emittery";
import type { useDialog } from "naive-ui";
import { v4 as uuidv4 } from "uuid";
import type { InjectionKey, Ref } from "vue";
import { inject, provide } from "vue";
import type { Plan_Spec, Plan_Step } from "@/types/proto/v1/plan_service";
import type { ComposedPlan } from "@/types/v1/issue/plan";

export type PlanEvents = Emittery<{
  "status-changed": { eager: boolean };
  "select-spec": { spec: Plan_Spec };
}>;

export type PlanContext = {
  // Basic fields
  isCreating: Ref<boolean>;
  ready: Ref<boolean>;
  plan: Ref<ComposedPlan>;

  selectedStep: Ref<Plan_Step>;
  selectedSpec: Ref<Plan_Spec>;
  formatOnSave: Ref<boolean>;

  // UI events
  events: PlanEvents;

  // misc
  dialog: ReturnType<typeof useDialog>;
  reInitialize: (overrides?: Record<string, string>) => Promise<void>;
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
