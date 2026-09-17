import { getTimeForPbTimestampProtoEs, UNKNOWN_ID } from "@/types";
import type { QueryHistory } from "@/types/proto-es/v1/query_history_service_pb";
import { formatAbsoluteDateTime } from "@/utils/datetime";

// Extracts the query history UID from a resource name like
// `projects/{project}/queryHistories/{id}`.
export const extractQueryHistoryUID = (name: string) => {
  const pattern = /(?:^|\/)queryHistories\/([^/]+)(?:$|\/)/;
  const matches = pattern.exec(name);
  return matches?.[1] ?? `${UNKNOWN_ID}`;
};

// A tab title cannot host a tooltip, so it names the time in full.
export const queryHistoryTabTitle = (history: QueryHistory): string =>
  history.createTime
    ? `Query history at ${formatAbsoluteDateTime(getTimeForPbTimestampProtoEs(history.createTime))}`
    : "Query history";
