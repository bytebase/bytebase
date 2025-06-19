import { fromJson, toJson } from "@bufbuild/protobuf";
import type { InstanceRole as OldInstanceRole } from "@/types/proto/v1/instance_role_service";
import { InstanceRole as OldInstanceRoleProto } from "@/types/proto/v1/instance_role_service";
import type { InstanceRole as NewInstanceRole } from "@/types/proto-es/v1/instance_role_service_pb";
import { InstanceRoleSchema } from "@/types/proto-es/v1/instance_role_service_pb";

// Convert old proto to proto-es
export const convertOldInstanceRoleToNew = (oldRole: OldInstanceRole): NewInstanceRole => {
  // Use toJSON to convert old proto to JSON, then fromJson to convert to proto-es
  const json = OldInstanceRoleProto.toJSON(oldRole) as any; // Type assertion needed due to proto type incompatibility
  return fromJson(InstanceRoleSchema, json);
};

// Convert proto-es to old proto
export const convertNewInstanceRoleToOld = (newRole: NewInstanceRole): OldInstanceRole => {
  // Use toJson to convert proto-es to JSON, then fromJSON to convert to old proto
  const json = toJson(InstanceRoleSchema, newRole);
  return OldInstanceRoleProto.fromJSON(json);
};