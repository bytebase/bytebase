const projectPagedDataCacheScope = (
  resource: "issues" | "plans",
  projectId: string
): string => `projects/${projectId}/${resource}`;

export const projectIssuesPagedDataCacheScope = (projectId: string): string =>
  projectPagedDataCacheScope("issues", projectId);

export const projectPlansPagedDataCacheScope = (projectId: string): string =>
  projectPagedDataCacheScope("plans", projectId);
