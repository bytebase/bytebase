import { type User } from "@/types/proto/api/v1alpha/user_service";
import type { Group } from "@/types/proto/api/v1alpha/group_service";
import type { Binding } from "@/types/proto/api/v1alpha/iam_policy";

export interface MemberRole {
  workspaceLevelRoles: Set<string>;
  projectRoleBindings: Binding[];
}

export interface MemberBinding extends MemberRole {
  title: string;
  // binidng is the fullname for binding member,
  // like user:{email} or group:{email}
  binding: string;
  type: "users" | "groups";
  user?: User;
  group?: Group;
}
