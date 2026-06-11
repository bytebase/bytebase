import { UNKNOWN_ID } from "@/types";

// Extracts the query history UID from a resource name like
// `projects/{project}/queryHistories/{id}`.
export const extractQueryHistoryUID = (name: string) => {
  const pattern = /(?:^|\/)queryHistories\/([^/]+)(?:$|\/)/;
  const matches = pattern.exec(name);
  return matches?.[1] ?? `${UNKNOWN_ID}`;
};
