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
  // The fullname for the binding member: `user:{email}` or `group:{email}`.
  binding: string;
  type: "users" | "groups";
  user?: User;
  group?: GroupBinding;
  // True when the email is in the IAM policy but has no principal (user hasn't
  // signed up yet). Only set when the current user has permission to list/get
  // users — otherwise we can't tell and this stays undefined.
  pending?: boolean;
}
