import { fromJson, toJson } from "@bufbuild/protobuf";
import type { 
  Group as OldGroup,
  GroupMember as OldGroupMember,
  GroupMember_Role as OldGroupMember_Role
} from "@/types/proto/v1/group_service";
import { 
  Group as OldGroupProto,
  GroupMember as OldGroupMemberProto,
  GroupMember_Role as OldGroupMember_RoleEnum
} from "@/types/proto/v1/group_service";
import type { 
  Group as NewGroup,
  GroupMember as NewGroupMember
} from "@/types/proto-es/v1/group_service_pb";
import { 
  GroupSchema,
  GroupMemberSchema,
  GroupMember_Role as NewGroupMember_Role
} from "@/types/proto-es/v1/group_service_pb";

// Convert old proto Group to proto-es Group
export const convertOldGroupToNew = (oldGroup: OldGroup): NewGroup => {
  const json = OldGroupProto.toJSON(oldGroup) as any;
  return fromJson(GroupSchema, json);
};

// Convert proto-es Group to old proto Group
export const convertNewGroupToOld = (newGroup: NewGroup): OldGroup => {
  const json = toJson(GroupSchema, newGroup);
  return OldGroupProto.fromJSON(json);
};

// Convert old proto GroupMember to proto-es GroupMember
export const convertOldGroupMemberToNew = (oldMember: OldGroupMember): NewGroupMember => {
  const json = OldGroupMemberProto.toJSON(oldMember) as any;
  return fromJson(GroupMemberSchema, json);
};

// Convert proto-es GroupMember to old proto GroupMember
export const convertNewGroupMemberToOld = (newMember: NewGroupMember): OldGroupMember => {
  const json = toJson(GroupMemberSchema, newMember);
  return OldGroupMemberProto.fromJSON(json);
};

// Convert old GroupMember_Role enum to new
export const convertOldGroupMemberRoleToNew = (oldRole: OldGroupMember_Role): NewGroupMember_Role => {
  switch (oldRole) {
    case OldGroupMember_RoleEnum.ROLE_UNSPECIFIED:
      return NewGroupMember_Role.ROLE_UNSPECIFIED;
    case OldGroupMember_RoleEnum.OWNER:
      return NewGroupMember_Role.OWNER;
    case OldGroupMember_RoleEnum.MEMBER:
      return NewGroupMember_Role.MEMBER;
    case OldGroupMember_RoleEnum.UNRECOGNIZED:
    default:
      return NewGroupMember_Role.ROLE_UNSPECIFIED;
  }
};

// Convert new GroupMember_Role enum to old
export const convertNewGroupMemberRoleToOld = (newRole: NewGroupMember_Role): OldGroupMember_Role => {
  switch (newRole) {
    case NewGroupMember_Role.ROLE_UNSPECIFIED:
      return OldGroupMember_RoleEnum.ROLE_UNSPECIFIED;
    case NewGroupMember_Role.OWNER:
      return OldGroupMember_RoleEnum.OWNER;
    case NewGroupMember_Role.MEMBER:
      return OldGroupMember_RoleEnum.MEMBER;
    default:
      return OldGroupMember_RoleEnum.ROLE_UNSPECIFIED;
  }
};