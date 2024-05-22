import { planServiceClient } from "@/grpcweb";
import { EMPTY_ID, UNKNOWN_ID } from "@/types";
import type { Plan } from "@/types/proto/v1/plan_service";
import {
  emptyPlan,
  unknownPlan,
  type ComposedPlan,
} from "@/types/v1/issue/plan";
import { extractProjectResourceName, hasProjectPermissionV2 } from "@/utils";
import { useCurrentUserV1 } from "../auth";
import { useProjectV1Store } from "./project";

export const composePlan = async (rawPlan: Plan): Promise<ComposedPlan> => {
  const me = useCurrentUserV1();
  const project = `projects/${extractProjectResourceName(rawPlan.name)}`;
  const projectEntity =
    await useProjectV1Store().getOrFetchProjectByName(project);

  const plan: ComposedPlan = {
    ...rawPlan,
    planCheckRunList: [],
    project,
    projectEntity,
  };

  if (
    hasProjectPermissionV2(projectEntity, me.value, "bb.planCheckRuns.list")
  ) {
    const { planCheckRuns } = await planServiceClient.listPlanCheckRuns({
      parent: rawPlan.name,
    });
    plan.planCheckRunList = planCheckRuns;
  }

  return plan;
};

export const fetchPlanByUID = async (uid: string, project = "-") => {
  if (uid === "undefined") {
    console.warn("undefined plan uid");
    return unknownPlan();
  }

  if (uid === String(EMPTY_ID)) return emptyPlan();
  if (uid === String(UNKNOWN_ID)) return unknownPlan();

  const rawPlan = await planServiceClient.getPlan({
    name: `projects/${project}/plans/${uid}`,
  });

  return composePlan(rawPlan);
};
