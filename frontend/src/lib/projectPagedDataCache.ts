import { invalidatePagedDataCacheScope } from "@/hooks/pagedDataCache";
import { extractProjectResourceName } from "@/utils/v1/project";

type VersionedResource = {
  name: string;
  updateTime?: {
    seconds: bigint;
    nanos: number;
  };
};

const projectPagedDataCacheScope = (
  resource: "issues" | "plans",
  projectId: string
): string => `projects/${projectId}/${resource}`;

export const projectIssuesPagedDataCacheScope = (projectId: string): string =>
  projectPagedDataCacheScope("issues", projectId);

export const projectPlansPagedDataCacheScope = (projectId: string): string =>
  projectPagedDataCacheScope("plans", projectId);

export const invalidateProjectPlansPagedDataCache = (
  projectId: string
): void => {
  invalidatePagedDataCacheScope(projectPlansPagedDataCacheScope(projectId));
};

export const invalidateProjectPlansPagedDataCacheForIssues = (
  issues: { name: string; plan: string }[]
): void => {
  for (const projectId of new Set(
    issues
      .filter((issue) => issue.plan !== "")
      .map((issue) => extractProjectResourceName(issue.name))
  )) {
    invalidateProjectPlansPagedDataCache(projectId);
  }
};

export const invalidateProjectPagedDataCacheIfChanged = (
  projectId: string,
  resource: "issues" | "plans",
  previous: VersionedResource | undefined,
  next: VersionedResource | undefined
): void => {
  const previousTime = previous?.updateTime;
  const nextTime = next?.updateTime;
  if (
    previous?.name !== next?.name ||
    !previousTime ||
    !nextTime ||
    (previousTime.seconds === nextTime.seconds &&
      previousTime.nanos === nextTime.nanos)
  ) {
    return;
  }

  if (resource === "plans") {
    invalidateProjectPlansPagedDataCache(projectId);
    return;
  }
  invalidatePagedDataCacheScope(projectIssuesPagedDataCacheScope(projectId));
};
