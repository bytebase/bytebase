import { type OptionConfig } from "@/components/ExprEditor/context";
import type { ResourceSelectOption } from "@/components/v2/Select/RemoteResourceSelector/types";
import { type Factor, SQLTypeList } from "@/plugins/cel";
import { t } from "@/plugins/i18n";
import { useRoleStore } from "@/store";
import { PRESET_WORKSPACE_ROLES, PresetRiskLevelList } from "@/types";
import { Engine, RiskLevel } from "@/types/proto-es/v1/common_pb";
import { WorkspaceApprovalSetting_Rule_Source } from "@/types/proto-es/v1/setting_service_pb";
import {
  displayRoleTitle,
  engineNameV1,
  getEnvironmentIdOptions,
  getInstanceIdOptionConfig,
  getProjectIdOptionConfig,
  supportedEngineV1List,
} from "@/utils";
import {
  CEL_ATTRIBUTE_LEVEL,
  CEL_ATTRIBUTE_REQUEST_EXPIRATION_DAYS,
  CEL_ATTRIBUTE_REQUEST_ROLE,
  CEL_ATTRIBUTE_RESOURCE_DATABASE_NAME,
  CEL_ATTRIBUTE_RESOURCE_DB_ENGINE,
  CEL_ATTRIBUTE_RESOURCE_ENVIRONMENT_ID,
  CEL_ATTRIBUTE_RESOURCE_INSTANCE_ID,
  CEL_ATTRIBUTE_RESOURCE_PROJECT_ID,
  CEL_ATTRIBUTE_RESOURCE_SCHEMA_NAME,
  CEL_ATTRIBUTE_RESOURCE_TABLE_NAME,
  CEL_ATTRIBUTE_RISK_LEVEL,
  CEL_ATTRIBUTE_STATEMENT_AFFECTED_ROWS,
  CEL_ATTRIBUTE_STATEMENT_SQL_TYPE,
  CEL_ATTRIBUTE_STATEMENT_TABLE_ROWS,
  CEL_ATTRIBUTE_STATEMENT_TEXT,
} from "@/utils/cel-attributes";

export const levelText = (level: RiskLevel) => {
  switch (level) {
    case RiskLevel.RISK_LEVEL_UNSPECIFIED:
      return t("custom-approval.risk-rule.risk.risk-level.default");
    case RiskLevel.LOW:
      return t("custom-approval.risk-rule.risk.risk-level.low");
    case RiskLevel.MODERATE:
      return t("custom-approval.risk-rule.risk.risk-level.moderate");
    case RiskLevel.HIGH:
      return t("custom-approval.risk-rule.risk.risk-level.high");
    default:
      return String(level);
  }
};

const commonFactorList: Factor[] = [
  CEL_ATTRIBUTE_RESOURCE_ENVIRONMENT_ID,
  CEL_ATTRIBUTE_RESOURCE_PROJECT_ID,
  CEL_ATTRIBUTE_RESOURCE_INSTANCE_ID,
  CEL_ATTRIBUTE_RESOURCE_DB_ENGINE,
] as const;

const schemaObjectNameFactorList: Factor[] = [
  CEL_ATTRIBUTE_RESOURCE_DATABASE_NAME,
  CEL_ATTRIBUTE_RESOURCE_SCHEMA_NAME,
  CEL_ATTRIBUTE_RESOURCE_TABLE_NAME,
] as const;

const migrationFactorList: Factor[] = [
  CEL_ATTRIBUTE_STATEMENT_AFFECTED_ROWS,
  CEL_ATTRIBUTE_STATEMENT_TABLE_ROWS,
  CEL_ATTRIBUTE_STATEMENT_SQL_TYPE,
  CEL_ATTRIBUTE_STATEMENT_TEXT,
] as const;

const getDBEndingOptions = () => {
  return supportedEngineV1List().map<ResourceSelectOption<unknown>>((type) => ({
    label: engineNameV1(type),
    value: Engine[type],
  }));
};

const getLevelOptions = () => {
  return PresetRiskLevelList.map<ResourceSelectOption<unknown>>(
    ({ level }) => ({
      label: levelText(level),
      value: RiskLevel[level],
    })
  );
};

const getRiskLevelOptions = () => {
  // Risk level values as strings for CEL conditions (e.g., risk_level == "HIGH")
  const levels = [
    { label: t("custom-approval.risk-rule.risk.risk-level.low"), value: "LOW" },
    {
      label: t("custom-approval.risk-rule.risk.risk-level.moderate"),
      value: "MODERATE",
    },
    {
      label: t("custom-approval.risk-rule.risk.risk-level.high"),
      value: "HIGH",
    },
  ];
  return levels.map<ResourceSelectOption<unknown>>(({ label, value }) => ({
    label,
    value,
  }));
};

const getSQLTypeOptions = (source: WorkspaceApprovalSetting_Rule_Source) => {
  const mapOptions = (values: readonly string[]) => {
    return values.map<ResourceSelectOption<unknown>>((v) => ({
      label: v,
      value: v,
    }));
  };
  switch (source) {
    case WorkspaceApprovalSetting_Rule_Source.CHANGE_DATABASE:
      return mapOptions([...SQLTypeList.DDL, ...SQLTypeList.DML]);
  }
  // unsupported source
  return [];
};

const getRoleOptions = () => {
  return useRoleStore()
    .roleList.filter((role) => !PRESET_WORKSPACE_ROLES.includes(role.name))
    .map((role) => ({
      label: displayRoleTitle(role.name),
      value: role.name,
    }));
};

// Overload for new approval rule source enum
export const approvalSourceText = (
  source: WorkspaceApprovalSetting_Rule_Source
) => {
  switch (source) {
    case WorkspaceApprovalSetting_Rule_Source.SOURCE_UNSPECIFIED:
      return t("custom-approval.approval-flow.fallback-rules");
    case WorkspaceApprovalSetting_Rule_Source.CHANGE_DATABASE:
      return t("custom-approval.risk-rule.risk.namespace.change_database");
    case WorkspaceApprovalSetting_Rule_Source.CREATE_DATABASE:
      return t("custom-approval.risk-rule.risk.namespace.create_database");
    case WorkspaceApprovalSetting_Rule_Source.EXPORT_DATA:
      return t("custom-approval.risk-rule.risk.namespace.data_export");
    case WorkspaceApprovalSetting_Rule_Source.REQUEST_ROLE:
      return t("custom-approval.risk-rule.risk.namespace.request-role");
    default:
      return "UNRECOGNIZED";
  }
};

// Map between WorkspaceApprovalSetting_Rule_Source and Factor lists
export const ApprovalSourceFactorMap: Map<
  WorkspaceApprovalSetting_Rule_Source,
  Factor[]
> = new Map([
  [
    WorkspaceApprovalSetting_Rule_Source.CHANGE_DATABASE,
    [
      ...commonFactorList,
      ...schemaObjectNameFactorList,
      ...migrationFactorList,
      CEL_ATTRIBUTE_RISK_LEVEL,
    ],
  ],
  [
    WorkspaceApprovalSetting_Rule_Source.CREATE_DATABASE,
    [...commonFactorList, CEL_ATTRIBUTE_RESOURCE_DATABASE_NAME],
  ],
  [
    WorkspaceApprovalSetting_Rule_Source.EXPORT_DATA,
    [...commonFactorList, ...schemaObjectNameFactorList],
  ],
  [
    WorkspaceApprovalSetting_Rule_Source.REQUEST_ROLE,
    [
      CEL_ATTRIBUTE_RESOURCE_ENVIRONMENT_ID,
      CEL_ATTRIBUTE_RESOURCE_PROJECT_ID,
      CEL_ATTRIBUTE_REQUEST_EXPIRATION_DAYS,
      CEL_ATTRIBUTE_REQUEST_ROLE,
    ],
  ],
]);

export const getApprovalFactorList = (
  source: WorkspaceApprovalSetting_Rule_Source
): Factor[] => {
  // Fallback rules (SOURCE_UNSPECIFIED) can only use resource.project_id
  if (source === WorkspaceApprovalSetting_Rule_Source.SOURCE_UNSPECIFIED) {
    return [CEL_ATTRIBUTE_RESOURCE_PROJECT_ID] as Factor[];
  }
  return ApprovalSourceFactorMap.get(source) ?? [];
};

export const getApprovalOptionConfigMap = (
  source: WorkspaceApprovalSetting_Rule_Source
) => {
  const factorList = getApprovalFactorList(source);
  return factorList.reduce((map, factor) => {
    let options: ResourceSelectOption<unknown>[] = [];
    switch (factor) {
      case CEL_ATTRIBUTE_RESOURCE_ENVIRONMENT_ID:
        options = getEnvironmentIdOptions();
        break;
      case CEL_ATTRIBUTE_RESOURCE_PROJECT_ID:
        map.set(factor, getProjectIdOptionConfig());
        return map;
      case CEL_ATTRIBUTE_RESOURCE_INSTANCE_ID:
        map.set(factor, getInstanceIdOptionConfig());
        return map;
      case CEL_ATTRIBUTE_RESOURCE_DB_ENGINE:
        options = getDBEndingOptions();
        break;
      case CEL_ATTRIBUTE_LEVEL:
        options = getLevelOptions();
        break;
      case CEL_ATTRIBUTE_STATEMENT_SQL_TYPE:
        options = getSQLTypeOptions(source);
        break;
      case CEL_ATTRIBUTE_REQUEST_ROLE:
        options = getRoleOptions();
        break;
      case CEL_ATTRIBUTE_RISK_LEVEL:
        options = getRiskLevelOptions();
        break;
      default:
        break;
    }

    map.set(factor, { options });

    return map;
  }, new Map<Factor, OptionConfig>());
};
