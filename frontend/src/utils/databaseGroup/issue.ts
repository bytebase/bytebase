import dayjs from "dayjs";
import type { LocationQueryRaw } from "vue-router";
import {
  PROJECT_V1_ROUTE_ISSUE_DETAIL,
  PROJECT_V1_ROUTE_PLAN_DETAIL,
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
  issueNameParts.push(`[${databaseGroup.title}]`);
  issueNameParts.push(
    type === "bb.issue.database.schema.update" ? `Edit schema` : `Change data`
  );
  const datetime = dayjs().format("@MM-DD HH:mm");
  const tz = "UTC" + dayjs().format("ZZ");
  issueNameParts.push(`${datetime} ${tz}`);

  const query: LocationQueryRaw = {
    template: type,
    name: issueNameParts.join(" "),
    databaseGroupName: databaseGroup.name,
    sql,
  };

  return {
    name: planOnly
      ? PROJECT_V1_ROUTE_PLAN_DETAIL
      : PROJECT_V1_ROUTE_ISSUE_DETAIL,
    params: {
      projectId: extractProjectResourceName(databaseGroup.name),
      issueSlug: "create",
      planId: "create",
    },
    query,
  };
};
