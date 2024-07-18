import type Emittery from "emittery";
import { v4 as uuidv4 } from "uuid";
import type { InjectionKey } from "vue";
import { inject, provide } from "vue";

export type PlanCheckRunEvents = Emittery<{
  "status-changed": undefined;
}>;

export type PlanCheckRunContext = {
  events: PlanCheckRunEvents;
};

const KEY = Symbol(
  `bb.plan-check-run.context.${uuidv4()}`
) as InjectionKey<PlanCheckRunContext>;

export const usePlanCheckRunContext = () => {
  return inject(KEY)!;
};

export const providePlanCheckRunContext = (
  context: Partial<PlanCheckRunContext>,
  root = false
) => {
  if (!root) {
    const parent = usePlanCheckRunContext();
    context = {
      ...parent,
      ...context,
    };
  }
  provide(KEY, context as PlanCheckRunContext);
};
