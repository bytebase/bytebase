import { type Router } from "vue-router";
import { PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL } from "@/router/dashboard/projectV1";

export const gotoSpec = (router: Router, specId: string) => {
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
      specId,
    },
    query: currentRoute.query || {},
  });
};
