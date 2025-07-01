import type { Group } from "@/types/proto-es/v1/group_service_pb";
import type { Binding } from "@/types/proto-es/v1/iam_policy_pb";
import { type User } from "@/types/proto-es/v1/user_service_pb";

export interface MemberRole {
  workspaceLevelRoles: Set<string>;
  projectRoleBindings: Binding[];
}

export interface GroupBinding extends Group {
  deleted?: boolean;
}

export interface MemberBinding extends MemberRole {
  title: string;
  // binidng is the fullname for binding member,
  // like user:{email} or group:{email}
  binding: string;
  type: "users" | "groups";
  user?: User;
  group?: GroupBinding;
}
