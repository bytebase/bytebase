import { Role } from "../proto/v1/role_service";
import { ProjectPermission, WorkspacePermission } from "./permission";

export interface ComposedRole extends Role {
  permissions: WorkspacePermission[] | ProjectPermission[];
}
