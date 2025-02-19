import { roleNamePrefix } from "@/store/modules/v1/common";
import { PresetRoleType } from "../iam";

export const VirtualRoleType = {
  CREATOR: `${roleNamePrefix}CREATOR`,
  LAST_APPROVER: `${roleNamePrefix}LAST_APPROVER`,
};

export const IssueReleaserRoleType = {
  WORKSPACE_ADMIN: PresetRoleType.WORKSPACE_ADMIN,
  WORKSPACE_DBA: PresetRoleType.WORKSPACE_DBA,
  PROJECT_OWNER: PresetRoleType.PROJECT_OWNER,
  PROJECT_RELEASER: PresetRoleType.PROJECT_RELEASER,
};
