import { WorkspaceLevelRoles, ProjectLevelRoles } from "@/types";

export const isCustomRole = (role: string) => {
  return (
    !WorkspaceLevelRoles.includes(role) && !ProjectLevelRoles.includes(role)
  );
};
