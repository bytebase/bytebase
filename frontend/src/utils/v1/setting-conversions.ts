import { fromJson, toJson } from "@bufbuild/protobuf";
import type { Setting as OldSetting } from "@/types/proto/v1/setting_service";
import { Setting as OldSettingProto, Setting_SettingName as OldSettingName } from "@/types/proto/v1/setting_service";
import type { Setting as NewSetting } from "@/types/proto-es/v1/setting_service_pb";
import { SettingSchema, Setting_SettingName as NewSettingName } from "@/types/proto-es/v1/setting_service_pb";
import { State as NewState, VCSType as NewVCSType, ExportFormat as NewExportFormat } from "@/types/proto-es/v1/common_pb";
import { State as OldState, VCSType as OldVCSType, ExportFormat as OldExportFormat } from "@/types/proto/v1/common";

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
    [OldSettingName.SETTING_NAME_UNSPECIFIED]: NewSettingName.SETTING_NAME_UNSPECIFIED,
    [OldSettingName.AUTH_SECRET]: NewSettingName.AUTH_SECRET,
    [OldSettingName.BRANDING_LOGO]: NewSettingName.BRANDING_LOGO,
    [OldSettingName.WORKSPACE_ID]: NewSettingName.WORKSPACE_ID,
    [OldSettingName.WORKSPACE_PROFILE]: NewSettingName.WORKSPACE_PROFILE,
    [OldSettingName.WORKSPACE_APPROVAL]: NewSettingName.WORKSPACE_APPROVAL,
    [OldSettingName.WORKSPACE_EXTERNAL_APPROVAL]: NewSettingName.WORKSPACE_EXTERNAL_APPROVAL,
    [OldSettingName.ENTERPRISE_LICENSE]: NewSettingName.ENTERPRISE_LICENSE,
    [OldSettingName.APP_IM]: NewSettingName.APP_IM,
    [OldSettingName.WATERMARK]: NewSettingName.WATERMARK,
    [OldSettingName.AI]: NewSettingName.AI,
    [OldSettingName.SCHEMA_TEMPLATE]: NewSettingName.SCHEMA_TEMPLATE,
    [OldSettingName.DATA_CLASSIFICATION]: NewSettingName.DATA_CLASSIFICATION,
    [OldSettingName.SEMANTIC_TYPES]: NewSettingName.SEMANTIC_TYPES,
    [OldSettingName.SQL_RESULT_SIZE_LIMIT]: NewSettingName.SQL_RESULT_SIZE_LIMIT,
    [OldSettingName.SCIM]: NewSettingName.SCIM,
    [OldSettingName.PASSWORD_RESTRICTION]: NewSettingName.PASSWORD_RESTRICTION,
    [OldSettingName.ENVIRONMENT]: NewSettingName.ENVIRONMENT,
    [OldSettingName.UNRECOGNIZED]: NewSettingName.SETTING_NAME_UNSPECIFIED,
  };
  const newEnumValue = mapping[oldName] ?? NewSettingName.SETTING_NAME_UNSPECIFIED;
  return NewSettingName[newEnumValue];
};

// Convert new string format to old enum
export const convertNewSettingNameToOld = (newNameString: string): OldSettingName => {
  // Find the numeric enum value from the string
  const newEnumValue = Object.entries(NewSettingName).find(([key]) => key === newNameString)?.[1] as NewSettingName | undefined;
  if (newEnumValue === undefined) {
    return OldSettingName.UNRECOGNIZED;
  }
  return convertNewSettingNameEnumToOld(newEnumValue);
};

// Convert new enum to old enum (internal helper)
const convertNewSettingNameEnumToOld = (newName: NewSettingName): OldSettingName => {
  // Map numeric enum to string enum
  const mapping: Record<NewSettingName, OldSettingName> = {
    [NewSettingName.SETTING_NAME_UNSPECIFIED]: OldSettingName.SETTING_NAME_UNSPECIFIED,
    [NewSettingName.AUTH_SECRET]: OldSettingName.AUTH_SECRET,
    [NewSettingName.BRANDING_LOGO]: OldSettingName.BRANDING_LOGO,
    [NewSettingName.WORKSPACE_ID]: OldSettingName.WORKSPACE_ID,
    [NewSettingName.WORKSPACE_PROFILE]: OldSettingName.WORKSPACE_PROFILE,
    [NewSettingName.WORKSPACE_APPROVAL]: OldSettingName.WORKSPACE_APPROVAL,
    [NewSettingName.WORKSPACE_EXTERNAL_APPROVAL]: OldSettingName.WORKSPACE_EXTERNAL_APPROVAL,
    [NewSettingName.ENTERPRISE_LICENSE]: OldSettingName.ENTERPRISE_LICENSE,
    [NewSettingName.APP_IM]: OldSettingName.APP_IM,
    [NewSettingName.WATERMARK]: OldSettingName.WATERMARK,
    [NewSettingName.AI]: OldSettingName.AI,
    [NewSettingName.SCHEMA_TEMPLATE]: OldSettingName.SCHEMA_TEMPLATE,
    [NewSettingName.DATA_CLASSIFICATION]: OldSettingName.DATA_CLASSIFICATION,
    [NewSettingName.SEMANTIC_TYPES]: OldSettingName.SEMANTIC_TYPES,
    [NewSettingName.SQL_RESULT_SIZE_LIMIT]: OldSettingName.SQL_RESULT_SIZE_LIMIT,
    [NewSettingName.SCIM]: OldSettingName.SCIM,
    [NewSettingName.PASSWORD_RESTRICTION]: OldSettingName.PASSWORD_RESTRICTION,
    [NewSettingName.ENVIRONMENT]: OldSettingName.ENVIRONMENT,
  };
  return mapping[newName] ?? OldSettingName.UNRECOGNIZED;
};

// Convert proto-es State to old State for utility functions
export const convertStateToOld = (state: NewState): OldState => {
  switch (state) {
    case NewState.STATE_UNSPECIFIED:
      return OldState.STATE_UNSPECIFIED;
    case NewState.ACTIVE:
      return OldState.ACTIVE;
    case NewState.DELETED:
      return OldState.DELETED;
    default:
      return OldState.UNRECOGNIZED;
  }
};

// Convert old State to proto-es State
export const convertStateToNew = (state: OldState): NewState => {
  switch (state) {
    case OldState.STATE_UNSPECIFIED:
      return NewState.STATE_UNSPECIFIED;
    case OldState.ACTIVE:
      return NewState.ACTIVE;
    case OldState.DELETED:
      return NewState.DELETED;
    case OldState.UNRECOGNIZED:
      return NewState.STATE_UNSPECIFIED;
    default:
      return NewState.STATE_UNSPECIFIED;
  }
};

// Convert proto-es VCSType to old VCSType
export const convertVCSTypeToOld = (vcsType: NewVCSType): OldVCSType => {
  switch (vcsType) {
    case NewVCSType.GITHUB:
      return OldVCSType.GITHUB;
    case NewVCSType.GITLAB:
      return OldVCSType.GITLAB;
    case NewVCSType.BITBUCKET:
      return OldVCSType.BITBUCKET;
    case NewVCSType.AZURE_DEVOPS:
      return OldVCSType.AZURE_DEVOPS;
    default:
      return OldVCSType.VCS_TYPE_UNSPECIFIED;
  }
};

// Convert old VCSType to proto-es VCSType
export const convertVCSTypeToNew = (vcsType: OldVCSType): NewVCSType => {
  switch (vcsType) {
    case OldVCSType.GITHUB:
      return NewVCSType.GITHUB;
    case OldVCSType.GITLAB:
      return NewVCSType.GITLAB;
    case OldVCSType.BITBUCKET:
      return NewVCSType.BITBUCKET;
    case OldVCSType.AZURE_DEVOPS:
      return NewVCSType.AZURE_DEVOPS;
    case OldVCSType.VCS_TYPE_UNSPECIFIED:
      return NewVCSType.VCS_TYPE_UNSPECIFIED;
    case OldVCSType.UNRECOGNIZED:
      return NewVCSType.VCS_TYPE_UNSPECIFIED;
    default:
      return NewVCSType.VCS_TYPE_UNSPECIFIED;
  }
};

// Convert proto-es ExportFormat to old ExportFormat
export const convertExportFormatToOld = (format: NewExportFormat): OldExportFormat => {
  switch (format) {
    case NewExportFormat.FORMAT_UNSPECIFIED:
      return OldExportFormat.FORMAT_UNSPECIFIED;
    case NewExportFormat.CSV:
      return OldExportFormat.CSV;
    case NewExportFormat.JSON:
      return OldExportFormat.JSON;
    case NewExportFormat.SQL:
      return OldExportFormat.SQL;
    case NewExportFormat.XLSX:
      return OldExportFormat.XLSX;
    default:
      return OldExportFormat.FORMAT_UNSPECIFIED;
  }
};

// Convert old ExportFormat to proto-es ExportFormat
export const convertExportFormatToNew = (format: OldExportFormat): NewExportFormat => {
  switch (format) {
    case OldExportFormat.FORMAT_UNSPECIFIED:
      return NewExportFormat.FORMAT_UNSPECIFIED;
    case OldExportFormat.CSV:
      return NewExportFormat.CSV;
    case OldExportFormat.JSON:
      return NewExportFormat.JSON;
    case OldExportFormat.SQL:
      return NewExportFormat.SQL;
    case OldExportFormat.XLSX:
      return NewExportFormat.XLSX;
    case OldExportFormat.UNRECOGNIZED:
      return NewExportFormat.FORMAT_UNSPECIFIED;
    default:
      return NewExportFormat.FORMAT_UNSPECIFIED;
  }
};