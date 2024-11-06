import { type User } from "@/types/proto/v1/auth_service";
import type { Group } from "@/types/proto/v1/group_service";
import type { Binding } from "@/types/proto/v1/iam_policy";

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
