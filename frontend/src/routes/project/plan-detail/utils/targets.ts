export const MAX_INLINE_DATABASES = 5;

export const splitInlineDatabases = <T>(
  databases: T[],
  maxInline = MAX_INLINE_DATABASES
) => ({
  extraDatabases: databases.slice(maxInline),
  inlineDatabases: databases.slice(0, maxInline),
});

export const filterPlanTargets = ({
  getDatabaseDisplayName,
  query,
  targets,
}: {
  getDatabaseDisplayName: (target: string) => string;
  query: string;
  targets: string[];
}) => {
  const normalized = query.trim().toLowerCase();
  if (!normalized) {
    return targets;
  }

  return targets.filter((target) => {
    const targetText = target.toLowerCase();
    if (targetText.includes("/databases/")) {
      return getDatabaseDisplayName(target).toLowerCase().includes(normalized);
    }
    return targetText.includes(normalized);
  });
};

export const getDatabaseGroupRouteParams = ({
  databaseGroupName,
  projectName,
}: {
  databaseGroupName: string;
  projectName: string;
}) => ({
  databaseGroupName,
  projectId: projectName.replace(/^projects\//, ""),
});
