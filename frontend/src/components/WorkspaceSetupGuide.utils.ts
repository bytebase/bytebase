import { extractDatabaseResourceName } from "@/utils";

export const LEGACY_SAMPLE_PROJECT_NAME = "projects/project-sample";

export function isSetupProjectName(name: string, defaultProject: string) {
  return !!name && name !== defaultProject;
}

export function isUserProjectName(name: string, defaultProject: string) {
  return (
    isSetupProjectName(name, defaultProject) &&
    name !== LEGACY_SAMPLE_PROJECT_NAME
  );
}

export function isSampleDatabaseName(
  name: string,
  sampleInstanceNames: ReadonlySet<string>
) {
  return sampleInstanceNames.has(extractDatabaseResourceName(name).instance);
}

export async function findFirstPageItem<T>(
  fetchPage: (
    pageToken: string
  ) => Promise<{ items: T[]; nextPageToken: string }>,
  predicate: (item: T) => boolean
): Promise<T | undefined> {
  let pageToken = "";
  while (true) {
    const page = await fetchPage(pageToken);
    const item = page.items.find(predicate);
    if (item !== undefined) return item;
    if (!page.nextPageToken) return undefined;
    pageToken = page.nextPageToken;
  }
}
