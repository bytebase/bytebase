import { create as createProto } from "@bufbuild/protobuf";
import dayjs from "dayjs";
import { uniq } from "lodash-es";
import { planServiceClientConnect } from "@/api";
import {
  ListPlansRequestSchema,
  type Plan,
} from "@/types/proto-es/v1/plan_service_pb";
import {
  getTsRangeFromSearchParams,
  getValueFromSearchParams,
  getValuesFromSearchParams,
  type SearchParams,
} from "@/utils";
import { celString } from "@/utils/v1/celLiteral";
import type { AppSliceCreator, PlanFind, PlanSlice } from "./types";

export const buildPlanFilter = (find: PlanFind): string => {
  const filter: string[] = [];
  if (find.query) {
    filter.push(
      `title.contains(${celString(find.query.trim().toLowerCase())})`
    );
  }
  if (find.creator) {
    filter.push(`creator == ${celString(find.creator)}`);
  }
  if (find.createdTsAfter) {
    filter.push(
      `create_time >= ${celString(dayjs(find.createdTsAfter).utc().format())}`
    );
  }
  if (find.createdTsBefore) {
    filter.push(
      `create_time <= ${celString(dayjs(find.createdTsBefore).utc().format())}`
    );
  }
  if (find.hasRollout !== undefined) {
    filter.push(`has_rollout == ${find.hasRollout}`);
  }
  if (find.hasRelease !== undefined) {
    filter.push(`has_release == ${find.hasRelease}`);
  }
  if (find.release) {
    filter.push(`release == ${celString(find.release)}`);
  }
  if (find.taskStatuses && find.taskStatuses.length > 0) {
    filter.push(
      `task_status in [${find.taskStatuses.map(celString).join(", ")}]`
    );
  }
  if (find.specType) {
    filter.push(`spec_type == ${celString(find.specType)}`);
  }
  if (find.state) {
    filter.push(`state == ${celString(find.state)}`);
  }
  return filter.join(" && ");
};

// The `status` scope groups Task.Status values into the stages a reader scans for.
export const PLAN_STATUS_SCOPE_TASK_STATUSES = {
  FAILED: ["FAILED"],
  RUNNING: ["RUNNING"],
  WAITING: ["NOT_STARTED", "PENDING"],
  DONE: ["DONE"],
} as const satisfies Record<string, readonly string[]>;

export type PlanStatusScopeValue = keyof typeof PLAN_STATUS_SCOPE_TASK_STATUSES;

const isPlanStatusScopeValue = (value: string): value is PlanStatusScopeValue =>
  Object.hasOwn(PLAN_STATUS_SCOPE_TASK_STATUSES, value);

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

  const taskStatuses = uniq(
    getValuesFromSearchParams(params, "status")
      .filter(isPlanStatusScopeValue)
      .flatMap((value) => PLAN_STATUS_SCOPE_TASK_STATUSES[value])
  );

  const filter: PlanFind = {
    ...defaultFind,
    project: `projects/${projectScope || "-"}`,
    query: params.query,
    createdTsAfter: createdTsRange?.[0],
    createdTsBefore: createdTsRange?.[1],
    creator: getValueFromSearchParams(params, "creator", "users/"),
    release:
      getValueFromSearchParams(params, "release") || defaultFind?.release,
    taskStatuses:
      taskStatuses.length > 0 ? taskStatuses : defaultFind?.taskStatuses,
    state: state || defaultFind?.state,
  };
  return filter;
};

export const createPlanSlice: AppSliceCreator<PlanSlice> = (set, get) => ({
  viewedPlan: undefined,
  setViewedPlan: (viewedPlan) => {
    const current = get().viewedPlan;
    if (
      current?.name === viewedPlan.name &&
      current.isReleaseBacked === viewedPlan.isReleaseBacked
    ) {
      return;
    }
    set({ viewedPlan });
  },
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
