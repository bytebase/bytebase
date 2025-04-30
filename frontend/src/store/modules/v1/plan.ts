import dayjs from "dayjs";
import { orderBy } from "lodash-es";
import { defineStore } from "pinia";
import { planServiceClient } from "@/grpcweb";
import { useUserStore } from "@/store";
import { EMPTY_ID, UNKNOWN_ID } from "@/types";
import { unknownUser } from "@/types";
import type { Plan } from "@/types/proto/v1/plan_service";
import {
  emptyPlan,
  unknownPlan,
  type ComposedPlan,
} from "@/types/v1/issue/plan";
import {
  extractProjectResourceName,
  getTsRangeFromSearchParams,
  getValueFromSearchParams,
  hasProjectPermissionV2,
  type SearchParams,
} from "@/utils";
import { batchGetOrFetchDatabases } from "./database";
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

export const composePlan = async (rawPlan: Plan): Promise<ComposedPlan> => {
  const userStore = useUserStore();
  const project = `projects/${extractProjectResourceName(rawPlan.name)}`;
  const projectEntity =
    await useProjectV1Store().getOrFetchProjectByName(project);

  const creatorEntity =
    (await userStore.getOrFetchUserByIdentifier(rawPlan.creator)) ??
    unknownUser();

  const plan: ComposedPlan = {
    ...rawPlan,
    planCheckRunList: [],
    project,
    projectEntity,
    creatorEntity,
  };

  await batchGetOrFetchDatabases(
    plan.steps
      .flatMap((step) => step.specs)
      .flatMap((spec) => spec.changeDatabaseConfig?.target ?? "")
  );

  if (hasProjectPermissionV2(projectEntity, "bb.planCheckRuns.list")) {
    // Only show the latest plan check runs.
    // TODO(steven): maybe we need to show all plan check runs on a separate page later.
    const { planCheckRuns } = await planServiceClient.listPlanCheckRuns({
      parent: rawPlan.name,
      latestOnly: true,
    });
    plan.planCheckRunList = orderBy(planCheckRuns, "name", "desc");
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
    const composedPlans = await Promise.all(
      resp.plans.map((plan) => composePlan(plan))
    );
    return {
      nextPageToken: resp.nextPageToken,
      plans: composedPlans,
    };
  };

  const fetchPlanByName = async (name: string): Promise<ComposedPlan> => {
    const rawPlan = await planServiceClient.getPlan({
      name,
    });
    return await composePlan(rawPlan);
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

    return fetchPlanByName(`projects/${project}/plans/${uid}`);
  };

  return {
    searchPlans,
    fetchPlanByName,
    fetchPlanByUID,
  };
});
