import dayjs from "dayjs";
import type JSZip from "jszip";
import { padStart } from "lodash-es";
import { useChangelogStore, useSheetV1Store } from "@/store";
import {
  getDateForPbTimestampProtoEs,
  type Changelist_Change_Source as ChangeSource,
} from "@/types";
import type { Changelist_Change as Change } from "@/types/proto/v1/changelist_service";
import { ChangelogView } from "@/types/proto-es/v1/database_service_pb";
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

const zipFileForChangelog = async (
  zip: JSZip,
  change: Change,
  index: number
) => {
  const sheet = await useSheetV1Store().fetchSheetByName(change.sheet, "FULL");
  if (!sheet) {
    return;
  }
  const changelog = await useChangelogStore().getOrFetchChangelogByName(
    change.source,
    ChangelogView.FULL
  );
  if (!changelog) {
    return;
  }

  const parts: string[] = [
    dayjs(getDateForPbTimestampProtoEs(changelog.createTime)).format(
      "YYYY-MM-DD HH:mm:ss"
    ),
  ];
  if (changelog.version) {
    parts.push(changelog.version);
  }

  const filename = buildFileName("CHANGELOG", parts.join("-"), index);
  zip.file(filename, getSheetStatement(sheet));
};

const zipFileForRawSQL = async (zip: JSZip, change: Change, index: number) => {
  const sheet = await useSheetV1Store().fetchSheetByName(change.sheet, "FULL");
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
  if (type === "CHANGELOG") {
    await zipFileForChangelog(zip, change, index);
  }
  if (type === "RAW_SQL") {
    await zipFileForRawSQL(zip, change, index);
  }
};
