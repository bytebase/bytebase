import { useEffect } from "react";
import { router } from "@/router";
import { type Issue } from "@/types/proto-es/v1/issue_service_pb";
import type { Plan } from "@/types/proto-es/v1/plan_service_pb";
import { getIssueRoute } from "@/utils";

const shouldRedirectToIssueDetail = (plan: Plan, issue?: Issue) => {
  if (!issue?.name) {
    return false;
  }
  if (plan.specs.length === 0) {
    return false;
  }
  return plan.specs.every((spec) => {
    return (
      spec.config?.case === "createDatabaseConfig" ||
      spec.config?.case === "exportDataConfig"
    );
  });
};

export function useRedirects(params: {
  ready: boolean;
  plan: Plan;
  issue?: Issue;
}) {
  const { ready, plan, issue } = params;

  useEffect(() => {
    if (ready && shouldRedirectToIssueDetail(plan, issue)) {
      void router.replace(getIssueRoute({ name: issue?.name ?? "" }));
    }
  }, [issue, plan, ready]);
}
