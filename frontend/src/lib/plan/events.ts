import Emittery from "emittery";

export type PlanEvents = {
  "database-change-issue-created": {
    issue: string;
    project: string;
  };
};

export const planEvents = new Emittery<PlanEvents>();
