export function isSetupProjectName(name: string, defaultProject: string) {
  return !!name && name !== defaultProject;
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
