import dayjs from "@/plugins/dayjs";
import { useSheetV1Store } from "@/store";
import { useSchemaDesignStore } from "@/store/modules/schemaDesign";
import { ComposedDatabase } from "@/types";
import {
  Changelist_Change as Change,
  Changelist,
} from "@/types/proto/v1/changelist_service";
import {
  generateDDLByBranchAndDatabase,
  getChangelistChangeSourceType,
  getSheetStatement,
} from "@/utils";

export const generateSQLForChangeToDatabase = async (
  change: Change,
  database: ComposedDatabase
) => {
  const type = getChangelistChangeSourceType(change);
  if (type === "CHANGE_HISTORY" || type === "RAW_SQL") {
    const sheet = await useSheetV1Store().fetchSheetByName(
      change.sheet,
      true /* raw */
    );
    if (!sheet) {
      return "";
    }

    return getSheetStatement(sheet);
  }
  if (type === "BRANCH") {
    const branch = await useSchemaDesignStore().getOrFetchSchemaDesignByName(
      change.source
    );
    if (!branch) {
      return "";
    }
    const diffResult = await generateDDLByBranchAndDatabase(branch, database);
    return diffResult.statement;
  }

  console.error("Should never reach this line");
  return "";
};

export const generateIssueName = (
  databaseNameList: string[],
  changelist: Changelist
) => {
  // Create a user friendly default issue name
  const issueNameParts: string[] = [];
  if (databaseNameList.length === 1) {
    issueNameParts.push(`[${databaseNameList[0]}]`);
  } else {
    issueNameParts.push(`[${databaseNameList.length} databases]`);
  }
  issueNameParts.push(`Apply changelist [${changelist.description}]`);
  const datetime = dayjs().format("@MM-DD HH:mm");
  const tz = "UTC" + dayjs().format("ZZ");
  issueNameParts.push(`${datetime} ${tz}`);

  return issueNameParts.join(" ");
};
