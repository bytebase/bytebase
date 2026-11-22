import Emittery from "emittery";

export type PlanEvents = {
  "database-change-issue-created": {
    issue: string;
    project: string;
  };
};

// Annotated rather than inferred: Emittery's default type argument names the
// listenerAdded and listenerRemoved symbols, which it does not export, so tsc
// cannot write the inferred type into a declaration file (TS4023).
export const planEvents: Emittery<PlanEvents> = new Emittery<PlanEvents>();
