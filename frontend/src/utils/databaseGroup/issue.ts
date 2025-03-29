import dayjs from "dayjs";
import {
  PROJECT_V1_ROUTE_ISSUE_DETAIL,
  PROJECT_V1_ROUTE_REVIEW_CENTER_DETAIL,
} from "@/router/dashboard/projectV1";
import type { ComposedDatabaseGroup } from "@/types";
import { extractProjectResourceName } from "../v1";

export const generateDatabaseGroupIssueRoute = (
  type: "bb.issue.database.schema.update" | "bb.issue.database.data.update",
  databaseGroup: ComposedDatabaseGroup,
  sql = "",
  planOnly = false
) => {
  const issueNameParts: string[] = [];
  issueNameParts.push(`[${databaseGroup.databasePlaceholder}]`);
  issueNameParts.push(
    type === "bb.issue.database.schema.update" ? `Edit schema` : `Change data`
  );
  const datetime = dayjs().format("@MM-DD HH:mm");
  const tz = "UTC" + dayjs().format("ZZ");
  issueNameParts.push(`${datetime} ${tz}`);

  const query: Record<string, any> = {
    template: type,
    name: issueNameParts.join(" "),
    databaseGroupName: databaseGroup.name,
    sql,
  };

  return {
    name: planOnly
      ? PROJECT_V1_ROUTE_REVIEW_CENTER_DETAIL
      : PROJECT_V1_ROUTE_ISSUE_DETAIL,
    params: {
      projectId: extractProjectResourceName(databaseGroup.name),
      issueSlug: "create",
      planSlug: "create",
    },
    query,
  };
};
