import dayjs from "dayjs";
import { defineStore } from "pinia";
import { planServiceClient } from "@/grpcweb";
import { useUserStore } from "@/store";
import { EMPTY_ID, isValidProjectName, UNKNOWN_ID } from "@/types";
import { unknownUser } from "@/types";
import type { Plan } from "@/types/proto/v1/plan_service";
import {
  emptyPlan,
  unknownPlan,
  type ComposedPlan,
} from "@/types/v1/issue/plan";
import {
  extractProjectResourceName,
  extractUserResourceName,
  getTsRangeFromSearchParams,
  getValueFromSearchParams,
  hasProjectPermissionV2,
  type SearchParams,
} from "@/utils";
import { useProjectV1Store } from "./project";

export interface PlanFind {
  project: string;
  creator?: string;
  createdTsAfter?: number;
  createdTsBefore?: number;
  hasIssue?: boolean;
  hasPipeline?: boolean;
}

export const buildPlanFilter = (find: PlanFind): string => {
  const filter: string[] = [];
  if (find.creator) {
    filter.push(`creator = "${find.creator}"`);
  }
  if (find.createdTsAfter) {
    filter.push(
      `create_time >= "${dayjs(find.createdTsAfter).utc().format()}"`
    );
  }
  if (find.createdTsBefore) {
    filter.push(
      `create_time <= "${dayjs(find.createdTsBefore).utc().format()}"`
    );
  }
  if (find.hasIssue !== undefined) {
    filter.push(`has_issue = "${find.hasIssue}"`);
  }
  if (find.hasPipeline !== undefined) {
    filter.push(`has_pipeline = "${find.hasPipeline}"`);
  }
  return filter.join(" && ");
};

export const buildPlanFindBySearchParams = (
  params: SearchParams,
  defaultFind?: Partial<PlanFind>
) => {
  const { scopes } = params;
  const projectScope = scopes.find((s) => s.id === "project");

  const createdTsRange = getTsRangeFromSearchParams(params, "created");

  const filter: PlanFind = {
    ...defaultFind,
    project: `projects/${projectScope?.value ?? "-"}`,
    createdTsAfter: createdTsRange?.[0],
    createdTsBefore: createdTsRange?.[1],
    creator: getValueFromSearchParams(params, "creator", "users/"),
  };
  return filter;
};

export const composePlan = async (rawPlan: Plan): Promise<ComposedPlan> => {
  const userStore = useUserStore();
  const project = `projects/${extractProjectResourceName(rawPlan.name)}`;
  const projectEntity =
    await useProjectV1Store().getOrFetchProjectByName(project);

  const creatorEntity =
    userStore.getUserByEmail(extractUserResourceName(rawPlan.creator)) ??
    unknownUser();

  const plan: ComposedPlan = {
    ...rawPlan,
    planCheckRunList: [],
    project,
    projectEntity,
    creatorEntity,
  };

  if (hasProjectPermissionV2(projectEntity, "bb.planCheckRuns.list")) {
    // Only show the latest plan check runs.
    // TODO(steven): maybe we need to show all plan check runs on a separate page later.
    const { planCheckRuns } = await planServiceClient.listPlanCheckRuns({
      parent: rawPlan.name,
      latestOnly: true,
    });
    plan.planCheckRunList = planCheckRuns;
  }

  return plan;
};

export type ListPlanParams = {
  find: PlanFind;
  pageSize?: number;
  pageToken?: string;
};

export const usePlanStore = defineStore("plan", () => {
  const searchPlans = async ({ find, pageSize, pageToken }: ListPlanParams) => {
    const resp = await planServiceClient.searchPlans({
      parent: find.project,
      filter: buildPlanFilter(find),
      pageSize,
      pageToken,
    });

    const projectStore = useProjectV1Store();
    const plans = resp.plans.filter((plan) => {
      const proj = extractProjectResourceName(plan.name);
      return isValidProjectName(
        projectStore.getProjectByName(`projects/${proj}`).name
      );
    });

    const composedPlans = await Promise.all(
      plans.map((plan) => composePlan(plan))
    );
    return {
      nextPageToken: resp.nextPageToken,
      plans: composedPlans,
    };
  };

  const fetchPlanByUID = async (
    uid: string,
    project = "-"
  ): Promise<ComposedPlan> => {
    if (uid === "undefined") {
      console.warn("undefined plan uid");
      return emptyPlan();
    }

    if (uid === String(EMPTY_ID)) return emptyPlan();
    if (uid === String(UNKNOWN_ID)) return unknownPlan();

    const rawPlan = await planServiceClient.getPlan({
      name: `projects/${project}/plans/${uid}`,
    });

    return await composePlan(rawPlan);
  };

  return {
    searchPlans,
    fetchPlanByUID,
  };
});
