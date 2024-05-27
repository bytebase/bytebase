import type { Binding } from "@/types/proto/v1/iam_policy";

export interface ProjectRole {
  // Format: "roles/{roleName}"
  workspaceLevelProjectRoles: string[];
  projectRoleBindings: Binding[];
}

export interface ProjectBinding extends ProjectRole {
  title: string;
  email: string;
  // binidng is the fullname for binding member,
  // like user:{email} or group:{email}
  binding: string;
  type: "user" | "group";
}
