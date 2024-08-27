import { useProjectV1Store } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import type { Changelist } from "@/types/proto/v1/changelist_service";
import { extractProjectResourceName } from "@/utils";

export const projectForChangelist = (changelist: Changelist) => {
  const proj = extractProjectResourceName(changelist.name);
  return useProjectV1Store().getProjectByName(`${projectNamePrefix}${proj}`);
};
