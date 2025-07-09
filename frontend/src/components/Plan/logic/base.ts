import Emittery from "emittery";
import { computed } from "vue";
import { IssueStatus } from "@/types/proto-es/v1/issue_service_pb";
import type { PlanContext, PlanEvents } from "./context";

export const useBasePlanContext = ({
  issue,
}: Pick<
  PlanContext,
  "isCreating" | "plan" | "issue"
>): Partial<PlanContext> => {
  const events: PlanEvents = new Emittery();

  const readonly = computed(() => {
    if (issue?.value) {
      return issue.value.status !== IssueStatus.OPEN;
    }
    return false;
  });

  return {
    readonly,
    events,
  };
};
