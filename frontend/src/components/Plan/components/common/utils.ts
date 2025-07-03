import { type Router } from "vue-router";
import { PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL } from "@/router/dashboard/projectV1";
import type { Plan } from "@/types/proto-es/v1/plan_service_pb";
import { extractPlanUID } from "@/utils";

export const gotoSpec = (router: Router, plan: Plan, specId: string) => {
  // Defensive check for router and currentRoute
  if (!router || !router.currentRoute?.value) {
    console.error("Router or currentRoute is not available in gotoSpec");
    return;
  }

  const currentRoute = router.currentRoute.value;
  router.push({
    name: PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL,
    params: {
      ...(currentRoute.params || {}),
      planId: extractPlanUID(plan.name),
      specId,
    },
    query: currentRoute.query || {},
  });
};
