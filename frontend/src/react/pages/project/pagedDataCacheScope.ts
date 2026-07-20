import { invalidatePagedDataCacheScope } from "@/react/hooks/pagedDataCache";

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

  invalidatePagedDataCacheScope(
    resource === "plans"
      ? projectPlansPagedDataCacheScope(projectId)
      : projectIssuesPagedDataCacheScope(projectId)
  );
};
