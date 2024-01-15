import dayjs from "dayjs";
import { PROJECT_V1_ROUTE_ISSUE_DETAIL } from "@/router/dashboard/projectV1";
import { ComposedDatabaseGroup } from "@/types";
import { extractProjectResourceName } from "../v1";

export const generateDatabaseGroupIssueRoute = (
  type: "bb.issue.database.schema.update" | "bb.issue.database.data.update",
  databaseGroup: ComposedDatabaseGroup,
  sql = ""
) => {
  const issueNameParts: string[] = [];
  issueNameParts.push(`[${databaseGroup.databasePlaceholder}]`);
  issueNameParts.push(
    type === "bb.issue.database.schema.update" ? `Alter schema` : `Change data`
  );
  const datetime = dayjs().format("@MM-DD HH:mm");
  const tz = "UTC" + dayjs().format("ZZ");
  issueNameParts.push(`${datetime} ${tz}`);

  const query: Record<string, any> = {
    template: type,
    batch: "1",
    name: issueNameParts.join(" "),
    project: databaseGroup.project.uid,
    databaseGroupName: databaseGroup.name,
    sql,
  };

  return {
    name: PROJECT_V1_ROUTE_ISSUE_DETAIL,
    params: {
      projectId: extractProjectResourceName(databaseGroup.project.name),
      issueSlug: "create",
    },
    query,
  };
};
