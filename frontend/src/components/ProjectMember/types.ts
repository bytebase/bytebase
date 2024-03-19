import { User } from "@/types/proto/v1/auth_service";
import { Binding } from "@/types/proto/v1/iam_policy";

export interface ProjectMember {
  user: User;
  // Format: "roles/{roleName}"
  workspaceLevelProjectRoles: string[];
  projectRoleBindings: Binding[];
}
