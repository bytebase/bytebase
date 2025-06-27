import { fromJson, toJson } from "@bufbuild/protobuf";
import type { Setting as NewSetting } from "@/types/proto-es/v1/setting_service_pb";
import {
  SettingSchema,
  Setting_SettingName as NewSettingName,
} from "@/types/proto-es/v1/setting_service_pb";
import type { Setting as OldSetting } from "@/types/proto/v1/setting_service";
import {
  Setting as OldSettingProto,
  Setting_SettingName as OldSettingName,
} from "@/types/proto/v1/setting_service";

// Convert old proto to proto-es
export const convertOldSettingToNew = (oldSetting: OldSetting): NewSetting => {
  // Use toJSON to convert old proto to JSON, then fromJson to convert to proto-es
  const json = OldSettingProto.toJSON(oldSetting) as any; // Type assertion needed due to proto type incompatibility
  return fromJson(SettingSchema, json);
};

// Convert proto-es to old proto
export const convertNewSettingToOld = (newSetting: NewSetting): OldSetting => {
  // Use toJson to convert proto-es to JSON, then fromJSON to convert to old proto
  const json = toJson(SettingSchema, newSetting);
  return OldSettingProto.fromJSON(json);
};

// Convert old enum to new string format
export const convertOldSettingNameToNew = (oldName: OldSettingName): string => {
  // Map string enum to numeric enum
  const mapping: Record<OldSettingName, NewSettingName> = {
    [OldSettingName.SETTING_NAME_UNSPECIFIED]:
      NewSettingName.SETTING_NAME_UNSPECIFIED,
    [OldSettingName.AUTH_SECRET]: NewSettingName.AUTH_SECRET,
    [OldSettingName.BRANDING_LOGO]: NewSettingName.BRANDING_LOGO,
    [OldSettingName.WORKSPACE_ID]: NewSettingName.WORKSPACE_ID,
    [OldSettingName.WORKSPACE_PROFILE]: NewSettingName.WORKSPACE_PROFILE,
    [OldSettingName.WORKSPACE_APPROVAL]: NewSettingName.WORKSPACE_APPROVAL,
    [OldSettingName.WORKSPACE_EXTERNAL_APPROVAL]:
      NewSettingName.WORKSPACE_EXTERNAL_APPROVAL,
    [OldSettingName.ENTERPRISE_LICENSE]: NewSettingName.ENTERPRISE_LICENSE,
    [OldSettingName.APP_IM]: NewSettingName.APP_IM,
    [OldSettingName.WATERMARK]: NewSettingName.WATERMARK,
    [OldSettingName.AI]: NewSettingName.AI,
    [OldSettingName.SCHEMA_TEMPLATE]: NewSettingName.SCHEMA_TEMPLATE,
    [OldSettingName.DATA_CLASSIFICATION]: NewSettingName.DATA_CLASSIFICATION,
    [OldSettingName.SEMANTIC_TYPES]: NewSettingName.SEMANTIC_TYPES,
    [OldSettingName.SQL_RESULT_SIZE_LIMIT]:
      NewSettingName.SQL_RESULT_SIZE_LIMIT,
    [OldSettingName.SCIM]: NewSettingName.SCIM,
    [OldSettingName.PASSWORD_RESTRICTION]: NewSettingName.PASSWORD_RESTRICTION,
    [OldSettingName.ENVIRONMENT]: NewSettingName.ENVIRONMENT,
    [OldSettingName.UNRECOGNIZED]: NewSettingName.SETTING_NAME_UNSPECIFIED,
  };
  const newEnumValue =
    mapping[oldName] ?? NewSettingName.SETTING_NAME_UNSPECIFIED;
  return NewSettingName[newEnumValue];
};

// Convert new string format to old enum
export const convertNewSettingNameToOld = (
  newNameString: string
): OldSettingName => {
  // Find the numeric enum value from the string
  const newEnumValue = Object.entries(NewSettingName).find(
    ([key]) => key === newNameString
  )?.[1] as NewSettingName | undefined;
  if (newEnumValue === undefined) {
    return OldSettingName.UNRECOGNIZED;
  }
  return convertNewSettingNameEnumToOld(newEnumValue);
};

// Convert new enum to old enum (internal helper)
const convertNewSettingNameEnumToOld = (
  newName: NewSettingName
): OldSettingName => {
  // Map numeric enum to string enum
  const mapping: Record<NewSettingName, OldSettingName> = {
    [NewSettingName.SETTING_NAME_UNSPECIFIED]:
      OldSettingName.SETTING_NAME_UNSPECIFIED,
    [NewSettingName.AUTH_SECRET]: OldSettingName.AUTH_SECRET,
    [NewSettingName.BRANDING_LOGO]: OldSettingName.BRANDING_LOGO,
    [NewSettingName.WORKSPACE_ID]: OldSettingName.WORKSPACE_ID,
    [NewSettingName.WORKSPACE_PROFILE]: OldSettingName.WORKSPACE_PROFILE,
    [NewSettingName.WORKSPACE_APPROVAL]: OldSettingName.WORKSPACE_APPROVAL,
    [NewSettingName.WORKSPACE_EXTERNAL_APPROVAL]:
      OldSettingName.WORKSPACE_EXTERNAL_APPROVAL,
    [NewSettingName.ENTERPRISE_LICENSE]: OldSettingName.ENTERPRISE_LICENSE,
    [NewSettingName.APP_IM]: OldSettingName.APP_IM,
    [NewSettingName.WATERMARK]: OldSettingName.WATERMARK,
    [NewSettingName.AI]: OldSettingName.AI,
    [NewSettingName.SCHEMA_TEMPLATE]: OldSettingName.SCHEMA_TEMPLATE,
    [NewSettingName.DATA_CLASSIFICATION]: OldSettingName.DATA_CLASSIFICATION,
    [NewSettingName.SEMANTIC_TYPES]: OldSettingName.SEMANTIC_TYPES,
    [NewSettingName.SQL_RESULT_SIZE_LIMIT]:
      OldSettingName.SQL_RESULT_SIZE_LIMIT,
    [NewSettingName.SCIM]: OldSettingName.SCIM,
    [NewSettingName.PASSWORD_RESTRICTION]: OldSettingName.PASSWORD_RESTRICTION,
    [NewSettingName.ENVIRONMENT]: OldSettingName.ENVIRONMENT,
  };
  return mapping[newName] ?? OldSettingName.UNRECOGNIZED;
};
