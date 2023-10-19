import JSZip from "jszip";
import { padStart } from "lodash-es";
import { useChangeHistoryStore, useSheetV1Store } from "@/store";
import { useSchemaDesignStore } from "@/store/modules/schemaDesign";
import { Changelist_Change_Source as ChangeSource } from "@/types";
import { Changelist_Change as Change } from "@/types/proto/v1/changelist_service";
import { ChangeHistory_Status } from "@/types/proto/v1/database_service";
import {
  escapeFilename,
  getChangelistChangeSourceType,
  getSheetStatement,
} from "@/utils";

const buildFileName = (type: ChangeSource, name: string, index: number) => {
  const parts = [padStart(String(index + 1), 2, "0")];
  parts.push(type);
  if (name) {
    parts.push(escapeFilename(name));
  }
  const basename = parts.join("-");
  return `${basename}.sql`;
};

const zipFileForChangeHistory = async (
  zip: JSZip,
  change: Change,
  index: number
) => {
  const name = change.source;
  const { changeHistory, type, statement } =
    await useChangeHistoryStore().exportChangeHistoryFullStatementByName(name);
  console.log("export", change, changeHistory, type, statement.length);
  if (changeHistory) {
    if (changeHistory.status !== ChangeHistory_Status.DONE) {
      return;
    }

    if (type === "MIGRATE") {
      const filename = buildFileName(
        "CHANGE_HISTORY",
        changeHistory.version,
        index
      );
      zip.file(filename, statement);
    } else if (type === "BASELINE") {
      const filename = buildFileName(
        "CHANGE_HISTORY",
        `${changeHistory.version}_baseline`,
        index
      );
      zip.file(filename, statement);
    } else {
      // NOT SUPPORTED.
      return;
    }
  }
};

const zipFileForBranch = async (zip: JSZip, change: Change, index: number) => {
  const sheet = await useSheetV1Store().fetchSheetByName(
    change.sheet,
    true /* raw */
  );
  if (!sheet) {
    return;
  }
  const branch = await useSchemaDesignStore().fetchSchemaDesignByName(
    change.source,
    false /* !useCache */
  );
  if (!branch) {
    return;
  }

  const filename = buildFileName("BRANCH", branch.title, index);
  zip.file(filename, getSheetStatement(sheet));
};

const zipFileForRawSQL = async (zip: JSZip, change: Change, index: number) => {
  const sheet = await useSheetV1Store().fetchSheetByName(
    change.sheet,
    true /* raw */
  );
  if (!sheet) {
    return;
  }

  const filename = buildFileName("RAW_SQL", "", index);
  zip.file(filename, getSheetStatement(sheet));
};

export const zipFileForChange = async (
  zip: JSZip,
  change: Change,
  index: number
) => {
  const type = getChangelistChangeSourceType(change);
  if (type === "CHANGE_HISTORY") {
    await zipFileForChangeHistory(zip, change, index);
  }
  if (type === "BRANCH") {
    await zipFileForBranch(zip, change, index);
  }
  if (type === "RAW_SQL") {
    await zipFileForRawSQL(zip, change, index);
  }
};
