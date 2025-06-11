import { type Router } from "vue-router";
import { PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL } from "@/router/dashboard/projectV1";

export const gotoSpec = (router: Router, specId: string) => {
  router.push({
    name: PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL,
    params: {
      ...router.currentRoute.value.params,
      specId,
    },
    query: router.currentRoute.value.query,
  });
};
