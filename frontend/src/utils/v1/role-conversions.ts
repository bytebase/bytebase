import { fromJson, toJson } from "@bufbuild/protobuf";
import type { 
  Role as OldRole,
  Role_Type as OldRoleType
} from "@/types/proto/v1/role_service";
import { 
  Role as OldRoleProto,
  Role_Type as OldRoleTypeEnum
} from "@/types/proto/v1/role_service";
import type { 
  Role as NewRole
} from "@/types/proto-es/v1/role_service_pb";
import { 
  RoleSchema,
  Role_Type as NewRoleType
} from "@/types/proto-es/v1/role_service_pb";

// Convert old proto to proto-es
export const convertOldRoleToNew = (oldRole: OldRole): NewRole => {
  const json = OldRoleProto.toJSON(oldRole) as any;
  return fromJson(RoleSchema, json);
};

// Convert proto-es to old proto
export const convertNewRoleToOld = (newRole: NewRole): OldRole => {
  const json = toJson(RoleSchema, newRole);
  return OldRoleProto.fromJSON(json);
};

// Convert old role type enum to new enum
export const convertOldRoleTypeToNew = (oldType: OldRoleType): NewRoleType => {
  const mapping: Record<OldRoleType, NewRoleType> = {
    [OldRoleTypeEnum.TYPE_UNSPECIFIED]: NewRoleType.TYPE_UNSPECIFIED,
    [OldRoleTypeEnum.BUILT_IN]: NewRoleType.BUILT_IN,
    [OldRoleTypeEnum.CUSTOM]: NewRoleType.CUSTOM,
    [OldRoleTypeEnum.UNRECOGNIZED]: NewRoleType.TYPE_UNSPECIFIED,
  };
  return mapping[oldType] ?? NewRoleType.TYPE_UNSPECIFIED;
};

// Convert new role type enum to old enum
export const convertNewRoleTypeToOld = (newType: NewRoleType): OldRoleType => {
  const mapping: Record<NewRoleType, OldRoleType> = {
    [NewRoleType.TYPE_UNSPECIFIED]: OldRoleTypeEnum.TYPE_UNSPECIFIED,
    [NewRoleType.BUILT_IN]: OldRoleTypeEnum.BUILT_IN,
    [NewRoleType.CUSTOM]: OldRoleTypeEnum.CUSTOM,
  };
  return mapping[newType] ?? OldRoleTypeEnum.UNRECOGNIZED;
};