import {
  PROJECT_V1_ROUTE_DETAIL,
  PROJECT_V1_ROUTE_ISSUES,
} from "@/app/router/handles";
import { isSaaSOrDev } from "@/utils/featureGates";

// Project Home is available on SaaS and development builds. Keep every
// self-hosted production project-entry path on the established Issues route.
export const buildProjectEntryRoute = (
  projectId: string,
  isSaaSMode: boolean
) => ({
  name: isSaaSOrDev(isSaaSMode)
    ? PROJECT_V1_ROUTE_DETAIL
    : PROJECT_V1_ROUTE_ISSUES,
  params: { projectId },
});
