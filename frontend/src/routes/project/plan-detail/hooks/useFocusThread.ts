import { useCallback } from "react";
import { router } from "@/app/router";
import { PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL } from "@/app/router/handles";
import type { IssueComment } from "@/types/proto-es/v1/issue_service_pb";
import { usePlanDetailStoreApi } from "../shared/stores/usePlanDetailStore";
import { focusPlanPhase } from "../shell/focusPhase";
import { usePlanDetailContext } from "../shell/PlanDetailContext";

// Opens the Changes phase on a thread's spec and asks the statement editor to
// reveal it; the editor picks the request up once it shows that spec. A
// thread without an anchor, or anchored to a change the plan no longer has,
// has only its Activity card.
export function useFocusThread() {
  const page = usePlanDetailContext();
  const storeApi = usePlanDetailStoreApi();
  const { expandPhase, plan, planId, projectId } = page;

  const focusInActivity = useCallback(
    (root: IssueComment) => {
      focusPlanPhase("review", expandPhase);
      storeApi.getState().requestActivityFocus(root.name);
    },
    [expandPhase, storeApi]
  );

  const focusInStatement = useCallback(
    (root: IssueComment, lineNumber?: number) => {
      const specId = root.statementAnchor?.spec;
      if (!specId || !plan.specs.some((spec) => spec.id === specId)) {
        focusInActivity(root);
        return;
      }
      storeApi.getState().requestThreadFocus({
        commentName: root.name,
        specId,
        lineNumber: lineNumber ?? root.statementAnchor?.startPosition?.line,
      });
      focusPlanPhase("changes", expandPhase);
      if (storeApi.getState().displayedSpecId !== specId) {
        void router.push(
          {
            name: PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL,
            params: { planId, projectId, specId },
          },
          { preventScrollReset: true }
        );
      }
    },
    [expandPhase, focusInActivity, plan.specs, planId, projectId, storeApi]
  );

  return { focusInActivity, focusInStatement };
}
