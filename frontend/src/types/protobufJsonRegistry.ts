import { file_google_rpc_error_details } from "@buf/googleapis_googleapis.bufbuild_es/google/rpc/error_details_pb";
import { createRegistry } from "@bufbuild/protobuf";
import { AuditDataSchema } from "@/types/proto-es/v1/audit_log_service_pb";
import { PermissionDeniedDetailSchema } from "@/types/proto-es/v1/common_pb";
import { file_v1_plan_service } from "@/types/proto-es/v1/plan_service_pb";
import { SettingSchema } from "@/types/proto-es/v1/setting_service_pb";

export const protobufJsonRegistry = createRegistry(
  file_google_rpc_error_details,
  file_v1_plan_service,
  AuditDataSchema,
  PermissionDeniedDetailSchema,
  SettingSchema
);
