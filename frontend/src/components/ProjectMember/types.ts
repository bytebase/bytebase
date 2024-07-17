import type { ComposedUser } from "@/types";
import type { Binding } from "@/types/proto/v1/iam_policy";
import type { UserGroup } from "@/types/proto/v1/user_group";

export interface ProjectRole {
  // Format: "roles/{roleName}"
  workspaceLevelProjectRoles: string[];
  projectRoleBindings: Binding[];
}

export interface ProjectBinding extends ProjectRole {
  title: string;
  // binidng is the fullname for binding member,
  // like user:{email} or group:{email}
  binding: string;
  type: "users" | "groups";
  user?: ComposedUser;
  group?: UserGroup;
}
