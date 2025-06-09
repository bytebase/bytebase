import dayjs from "dayjs";
import { orderBy, uniq } from "lodash-es";
import { defineStore } from "pinia";
import { planServiceClient } from "@/grpcweb";
import { EMPTY_ID, UNKNOWN_ID } from "@/types";
import type { Plan, PlanCheckRun } from "@/types/proto/v1/plan_service";
import {
  emptyPlan,
  unknownPlan,
} from "@/types/v1/issue/plan";
import {
  extractProjectResourceName,
  getTsRangeFromSearchParams,
  getValueFromSearchParams,
  hasProjectPermissionV2,
  type SearchParams,
} from "@/utils";
import { useUserStore } from "../user";
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
    filter.push(`creator == "${find.creator}"`);
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
    filter.push(`has_issue == ${find.hasIssue}`);
  }
  if (find.hasPipeline !== undefined) {
    filter.push(`has_pipeline == ${find.hasPipeline}`);
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

export const fetchPlanCheckRuns = async (plan: Plan): Promise<PlanCheckRun[]> => {
  const project = `projects/${extractProjectResourceName(plan.name)}`;
  const projectEntity =
    await useProjectV1Store().getOrFetchProjectByName(project);

  if (hasProjectPermissionV2(projectEntity, "bb.planCheckRuns.list")) {
    const { planCheckRuns } = await planServiceClient.listPlanCheckRuns({
      parent: plan.name,
      latestOnly: true,
    });
    return orderBy(planCheckRuns, "name", "desc");
  }
  return [];
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
    // Prepare creator for the plans.
    const users = uniq(resp.plans.map((plan) => plan.creator));
    await useUserStore().batchGetUsers(users);
    return {
      nextPageToken: resp.nextPageToken,
      plans: resp.plans,
    };
  };

  const fetchPlanByName = async (name: string): Promise<Plan> => {
    const plan = await planServiceClient.getPlan({
      name,
    });
    return plan;
  };

  const fetchPlanByUID = async (
    uid: string,
    project = "-"
  ): Promise<Plan> => {
    if (uid === "undefined") {
      console.warn("undefined plan uid");
      return emptyPlan();
    }

    if (uid === String(EMPTY_ID)) return emptyPlan();
    if (uid === String(UNKNOWN_ID)) return unknownPlan();

    return fetchPlanByName(`projects/${project}/plans/${uid}`);
  };

  return {
    searchPlans,
    fetchPlanByName,
    fetchPlanByUID,
    fetchPlanCheckRuns,
  };
});
