import dayjs from "dayjs";
import { ComposedDatabaseGroup } from "@/types";

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
    name: "workspace.issue.detail",
    params: {
      issueSlug: "new",
    },
    query,
  };
};
