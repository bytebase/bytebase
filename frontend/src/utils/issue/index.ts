import dayjs from "dayjs";

export const generateIssueName = (
  type:
    | "bb.issue.database.schema.update"
    | "bb.issue.database.data.update"
    | "bb.issue.database.data.export"
    | "bb.issue.sql-review",
  databaseNameList: string[],
  isOnlineMode = false
) => {
  // Create a user friendly default issue name
  const issueNameParts: string[] = [];
  if (databaseNameList.length === 1) {
    issueNameParts.push(`[${databaseNameList[0]}]`);
  } else {
    issueNameParts.push(`[${databaseNameList.length} databases]`);
  }
  if (isOnlineMode) {
    issueNameParts.push("Online schema change");
  } else {
    issueNameParts.push(
      type === "bb.issue.database.schema.update"
        ? `Edit schema`
        : type === "bb.issue.database.data.update"
          ? `Change data`
          : type === "bb.issue.database.data.export"
            ? "Export data"
            : "Review SQL"
    );
  }
  const datetime = dayjs().format("@MM-DD HH:mm");
  const tz = "UTC" + dayjs().format("ZZ");
  issueNameParts.push(`${datetime} ${tz}`);

  return issueNameParts.join(" ");
};
