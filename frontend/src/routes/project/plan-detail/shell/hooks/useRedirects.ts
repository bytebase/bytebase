import { useEffect } from "react";
import { router } from "@/app/router";
import { type Issue } from "@/types/proto-es/v1/issue_service_pb";
import type { Plan } from "@/types/proto-es/v1/plan_service_pb";
import { getIssueRoute } from "@/utils";
import { shouldRedirectToIssueDetail } from "@/utils/v1/issue/plan";

export { shouldRedirectToIssueDetail } from "@/utils/v1/issue/plan";

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
