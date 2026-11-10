import { create } from "@bufbuild/protobuf";
import type { LoaderFunctionArgs } from "react-router";
import { planServiceClientConnect } from "@/api";
import { canonicalRedirect } from "@/app/router/canonicalRedirect";
import { projectNamePrefix } from "@/stores/modules/v1/common";
import { ListPlansRequestSchema } from "@/types/proto-es/v1/plan_service_pb";
import { buildSearchTextBySearchParams } from "@/utils/v1/advanced-search/common";
import { celString } from "@/utils/v1/celLiteral";
import { extractPlanUID } from "@/utils/v1/issue/plan";

// The release detail page is retired: a Release is tracked through the Plan
// that deployed it. Old links, including the CI summary bytebase-action
// writes, open that Plan, or the Releases list filtered to the release when
// zero or several active Plans reference it.
export async function releaseDetailRedirectLoader({
  params,
}: LoaderFunctionArgs): Promise<Response | null> {
  const { projectId, releaseId } = params;
  if (!projectId || !releaseId) {
    return null;
  }
  const projectName = `${projectNamePrefix}${projectId}`;
  const releaseName = `${projectName}/releases/${releaseId}`;
  try {
    const { plans } = await planServiceClientConnect.listPlans(
      create(ListPlansRequestSchema, {
        parent: projectName,
        filter: `release == ${celString(releaseName)} && state == "ACTIVE"`,
        pageSize: 2,
      })
    );
    if (plans.length === 1) {
      const plan = plans[0];
      const planPath = `/projects/${projectId}/plans/${extractPlanUID(plan.name)}`;
      return canonicalRedirect(
        plan.hasRollout ? `${planPath}/rollout` : planPath
      );
    }
  } catch {
    // Fall through to the filtered list, which reports its own errors.
  }
  const search = buildSearchTextBySearchParams({
    query: "",
    scopes: [{ id: "release", value: releaseName }],
  });
  return canonicalRedirect(
    `/projects/${projectId}/releases?q=${encodeURIComponent(search)}`
  );
}
