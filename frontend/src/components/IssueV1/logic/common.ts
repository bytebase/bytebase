import { RouteLocation } from "vue-router";

import { RollbackDetail, UNKNOWN_ID } from "@/types";

// Try to find out the relationship between databaseId and rollback issue/task
// Id from the URL query.
export const getRollbackTaskMappingFromQuery = (route: RouteLocation) => {
  const mapping = new Map<string, RollbackDetail>();

  const { query } = route;
  const databaseUIDListInQuery = (query.databaseList as string) || "";
  const databaseUIDList = databaseUIDListInQuery.split(",");

  const rollbackIssueUIDInQuery = (query.rollbackIssueId as string) || "";
  const issueUID = rollbackIssueUIDInQuery || String(UNKNOWN_ID);
  if (issueUID === String(UNKNOWN_ID)) {
    return mapping;
  }

  const rollbackTaskIdListInQuery = (query.rollbackTaskIdList as string) || "";
  const taskIdList = rollbackTaskIdListInQuery
    .split(",")
    .map((maybeUID) => maybeUID || String(UNKNOWN_ID));

  databaseUIDList.forEach((databaseId, index) => {
    const taskUID = taskIdList[index] || String(UNKNOWN_ID);
    if (taskUID !== String(UNKNOWN_ID)) {
      mapping.set(databaseId, {
        issueId: issueUID,
        taskId: taskUID,
      });
    }
  });
  return mapping;
};
