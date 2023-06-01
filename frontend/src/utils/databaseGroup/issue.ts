import { ComposedDatabaseGroup } from "@/types";
import dayjs from "dayjs";

export const generateIssueRoute = (
  type: "bb.issue.database.schema.update" | "bb.issue.database.data.update",
  databaseGroup: ComposedDatabaseGroup,
  schemaGroupNameList: string[] = []
) => {
  const issueNameParts: string[] = [];
  issueNameParts.push(`[${databaseGroup.databaseGroupName}]`);
  issueNameParts.push(
    type === "bb.issue.database.schema.update" ? `Alter schema` : `Change data`
  );
  const datetime = dayjs().format("@MM-DD HH:mm");
  const tz = "UTC" + dayjs().format("ZZ");
  issueNameParts.push(`${datetime} ${tz}`);

  const query: Record<string, any> = {
    template: type,
    mode: "tenant",
    name: issueNameParts.join(" "),
    project: databaseGroup.project.uid,
    databaseGroupName: databaseGroup.name,
    schemaGroupNames: schemaGroupNameList.join(","),
  };

  return {
    name: "workspace.issue.detail",
    params: {
      issueSlug: "new",
    },
    query,
  };
};
