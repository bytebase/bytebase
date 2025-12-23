import { create } from "@bufbuild/protobuf";
import dayjs from "dayjs";
import { uniq } from "lodash-es";
import { defineStore } from "pinia";
import { planServiceClientConnect } from "@/grpcweb";
import type { Plan } from "@/types/proto-es/v1/plan_service_pb";
import {
  GetPlanRequestSchema,
  SearchPlansRequestSchema,
  UpdatePlanRequestSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import {
  getTsRangeFromSearchParams,
  getValueFromSearchParams,
  type SearchParams,
} from "@/utils";
import { useUserStore } from "../user";

export interface PlanFind {
  project: string;
  query?: string;
  creator?: string;
  createdTsAfter?: number;
  createdTsBefore?: number;
  hasIssue?: boolean;
  hasPipeline?: boolean;
  specType?: string;
  state?: "ACTIVE" | "DELETED";
}

export const buildPlanFilter = (find: PlanFind): string => {
  const filter: string[] = [];
  if (find.query) {
    filter.push(`title.matches("${find.query.trim().toLowerCase()}")`);
  }
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
  if (find.specType) {
    filter.push(`spec_type == "${find.specType}"`);
  }
  if (find.state) {
    filter.push(`state == "${find.state}"`);
  }
  return filter.join(" && ");
};

export const buildPlanFindBySearchParams = (
  params: SearchParams,
  defaultFind?: Partial<PlanFind>
) => {
  const projectScope = getValueFromSearchParams(params, "project");

  const createdTsRange = getTsRangeFromSearchParams(params, "created");
  const state = getValueFromSearchParams(params, "state", "" /* prefix='' */, [
    "ACTIVE",
    "DELETED",
  ]) as "ACTIVE" | "DELETED" | "";

  const filter: PlanFind = {
    ...defaultFind,
    project: `projects/${projectScope || "-"}`,
    query: params.query,
    createdTsAfter: createdTsRange?.[0],
    createdTsBefore: createdTsRange?.[1],
    creator: getValueFromSearchParams(params, "creator", "users/"),
    state: state || defaultFind?.state,
  };
  return filter;
};

export type ListPlanParams = {
  find: PlanFind;
  pageSize?: number;
  pageToken?: string;
};

export const usePlanStore = defineStore("plan", () => {
  const searchPlans = async ({ find, pageSize, pageToken }: ListPlanParams) => {
    const request = create(SearchPlansRequestSchema, {
      parent: find.project,
      filter: buildPlanFilter(find),
      pageSize,
      pageToken,
    });
    const { plans, nextPageToken } =
      await planServiceClientConnect.searchPlans(request);
    // Prepare creator for the plans.
    const users = uniq(plans.map((plan) => plan.creator));
    await useUserStore().batchGetOrFetchUsers(users);
    return {
      nextPageToken: nextPageToken,
      plans,
    };
  };

  const fetchPlanByName = async (name: string): Promise<Plan> => {
    const request = create(GetPlanRequestSchema, {
      name,
    });
    const response = await planServiceClientConnect.getPlan(request);
    return response;
  };

  const updatePlan = async (
    plan: Plan,
    updateMask: string[]
  ): Promise<Plan> => {
    const request = create(UpdatePlanRequestSchema, {
      plan,
      updateMask: { paths: updateMask },
    });
    const response = await planServiceClientConnect.updatePlan(request);
    return response;
  };

  return {
    searchPlans,
    fetchPlanByName,
    updatePlan,
  };
});
