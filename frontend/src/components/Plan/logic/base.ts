import Emittery from "emittery";
import type { PlanContext, PlanEvents } from "./context";

export const useBasePlanContext = (
  _: Pick<PlanContext, "isCreating" | "plan">
): Partial<PlanContext> => {
  const events: PlanEvents = new Emittery();

  return {
    events,
  };
};
