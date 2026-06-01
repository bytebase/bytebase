import { create as createProto } from "@bufbuild/protobuf";
import dayjs from "dayjs";
import { uniq } from "lodash-es";
import { planServiceClientConnect } from "@/connect";
import {
  ListPlansRequestSchema,
  type Plan,
} from "@/types/proto-es/v1/plan_service_pb";
import {
  getTsRangeFromSearchParams,
  getValueFromSearchParams,
  type SearchParams,
} from "@/utils";
import type { AppSliceCreator, PlanFind, PlanSlice } from "./types";

export const buildPlanFilter = (find: PlanFind): string => {
  const filter: string[] = [];
  if (find.query) {
    filter.push(`title.contains("${find.query.trim().toLowerCase()}")`);
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
  if (find.hasRollout !== undefined) {
    filter.push(`has_rollout == ${find.hasRollout}`);
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

export const createPlanSlice: AppSliceCreator<PlanSlice> = (_set, get) => ({
  listPlans: async ({ find, pageSize, pageToken }) => {
    const { plans, nextPageToken } = await planServiceClientConnect.listPlans(
      createProto(ListPlansRequestSchema, {
        parent: find.project,
        filter: buildPlanFilter(find),
        pageSize,
        pageToken,
      })
    );
    // Prefetch the plan creators so consumers can resolve them synchronously.
    await get().batchGetOrFetchUsers(
      uniq(plans.map((plan: Plan) => plan.creator))
    );
    return { nextPageToken, plans };
  },
});
