import dayjs from "@/plugins/dayjs";
import { useProjectV1Store } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { Changelist } from "@/types/proto/v1/changelist_service";
import { extractProjectResourceName } from "@/utils";

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

export const projectForChangelist = (changelist: Changelist) => {
  const proj = extractProjectResourceName(changelist.name);
  return useProjectV1Store().getProjectByName(`${projectNamePrefix}${proj}`);
};
