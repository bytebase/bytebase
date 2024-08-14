import type { ComposedUser } from "@/types";
import type { Group } from "@/types/proto/v1/group";
import type { Binding } from "@/types/proto/v1/iam_policy";

export interface MemberRole {
  workspaceLevelRoles: string[];
  projectRoleBindings: Binding[];
}

export interface MemberBinding extends MemberRole {
  title: string;
  // binidng is the fullname for binding member,
  // like user:{email} or group:{email}
  binding: string;
  type: "users" | "groups";
  user?: ComposedUser;
  group?: Group;
}
