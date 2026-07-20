import { invalidatePagedDataCacheScope } from "@/react/hooks/pagedDataCache";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import type { Plan } from "@/types/proto-es/v1/plan_service_pb";
import {
  projectIssuesPagedDataCacheScope,
  projectPlansPagedDataCacheScope,
} from "./pagedDataCacheScope";

type MutationPatch = {
  issue?: Issue;
  plan?: Plan;
};

export const invalidateProjectPagedDataCache = (
  projectId: string,
  resource: "issues" | "plans"
): void => {
  invalidatePagedDataCacheScope(
    resource === "plans"
      ? projectPlansPagedDataCacheScope(projectId)
      : projectIssuesPagedDataCacheScope(projectId)
  );
};

export const applyProjectDetailMutationResult = <T extends MutationPatch>(
  page: {
    projectId: string;
    patchState: (patch: T) => void;
  },
  patch: T
): void => {
  if (patch.plan) {
    invalidateProjectPagedDataCache(page.projectId, "plans");
  }
  if (patch.issue) {
    invalidateProjectPagedDataCache(page.projectId, "issues");
  }
  page.patchState(patch);
};
