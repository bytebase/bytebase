import { UNKNOWN_ID } from "@/types";

/**
 * The resource of the slow query log.
 * The format is "environments/{environment}/instances/{instance}/databases/{database}".
 */
export const extractDatabaseIdFromSlowQueryLogDatabaseResourceName = (
  resource: string
) => {
  const pattern = /(?:^|\/)databases\/([^/]+)/;
  const matches = resource.match(pattern);
  if (matches) {
    const id = parseInt(matches[1], 10);
    if (id > 0) return id;
  }
  return UNKNOWN_ID;
};
