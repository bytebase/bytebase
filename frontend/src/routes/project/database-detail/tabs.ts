export const PROJECT_DATABASE_DETAIL_TAB_OVERVIEW = "overview";
export const PROJECT_DATABASE_DETAIL_TAB_CHANGELOG = "changelog";
export const PROJECT_DATABASE_DETAIL_TAB_REVISION = "revision";
export const PROJECT_DATABASE_DETAIL_TAB_CATALOG = "catalog";
export const PROJECT_DATABASE_DETAIL_TAB_SETTING = "setting";

export const PROJECT_DATABASE_DETAIL_TABS = [
  PROJECT_DATABASE_DETAIL_TAB_OVERVIEW,
  PROJECT_DATABASE_DETAIL_TAB_CHANGELOG,
  PROJECT_DATABASE_DETAIL_TAB_REVISION,
  PROJECT_DATABASE_DETAIL_TAB_CATALOG,
  PROJECT_DATABASE_DETAIL_TAB_SETTING,
] as const;

export type ProjectDatabaseDetailTab =
  (typeof PROJECT_DATABASE_DETAIL_TABS)[number];

const projectDatabaseDetailTabSet = new Set<string>(
  PROJECT_DATABASE_DETAIL_TABS
);

export function isProjectDatabaseDetailTab(
  value: string
): value is ProjectDatabaseDetailTab {
  return projectDatabaseDetailTabSet.has(value);
}

export function parseProjectDatabaseDetailTabHash(
  hash?: string | null
): ProjectDatabaseDetailTab {
  const value = hash?.replace(/^#/, "") ?? "";
  return isProjectDatabaseDetailTab(value)
    ? value
    : PROJECT_DATABASE_DETAIL_TAB_OVERVIEW;
}
