import { CheckIcon } from "lucide-vue-next";
import type { SelectOption } from "naive-ui";
import type { VNode } from "vue";
import { h } from "vue";
import { type OptionConfig } from "@/components/ExprEditor/context";
import { getInstanceIdOptions } from "@/components/SensitiveData/components/utils";
import { EnvironmentV1Name, RichDatabaseName } from "@/components/v2";
import { type Factor, SQLTypeList } from "@/plugins/cel";
import { t } from "@/plugins/i18n";
import {
  useEnvironmentV1Store,
  useInstanceV1Store,
  useProjectV1Store,
  useRoleStore,
} from "@/store";
import {
  type ComposedDatabase,
  DEFAULT_PROJECT_NAME,
  PRESET_WORKSPACE_ROLES,
  PresetRiskLevelList,
  useSupportedSourceList,
} from "@/types";
import { Engine, RiskLevel } from "@/types/proto-es/v1/common_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { type Risk, Risk_Source } from "@/types/proto-es/v1/risk_service_pb";
import { WorkspaceApprovalSetting_Rule_Source } from "@/types/proto-es/v1/setting_service_pb";
import {
  displayRoleTitle,
  engineNameV1,
  extractProjectResourceName,
  getDefaultPagination,
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
  CEL_ATTRIBUTE_SOURCE,
  CEL_ATTRIBUTE_STATEMENT_AFFECTED_ROWS,
  CEL_ATTRIBUTE_STATEMENT_SQL_TYPE,
  CEL_ATTRIBUTE_STATEMENT_TABLE_ROWS,
  CEL_ATTRIBUTE_STATEMENT_TEXT,
} from "@/utils/cel-attributes";

export const sourceText = (source: Risk_Source) => {
  switch (source) {
    case Risk_Source.SOURCE_UNSPECIFIED:
      return t("common.all");
    case Risk_Source.DDL:
      return t("custom-approval.risk-rule.risk.namespace.ddl");
    case Risk_Source.DML:
      return t("custom-approval.risk-rule.risk.namespace.dml");
    case Risk_Source.CREATE_DATABASE:
      return t("custom-approval.risk-rule.risk.namespace.create_database");
    case Risk_Source.DATA_EXPORT:
      return t("custom-approval.risk-rule.risk.namespace.data_export");
    case Risk_Source.REQUEST_ROLE:
      return t("custom-approval.risk-rule.risk.namespace.request-role");
    default:
      return "UNRECOGNIZED";
  }
};

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

export const orderByLevelDesc = (a: Risk, b: Risk): number => {
  if (a.level !== b.level) return -(a.level - b.level);
  if (a.name === b.name) return 0;
  return a.name < b.name ? -1 : 1;
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

export const RiskSourceFactorMap: Map<Risk_Source, Factor[]> = new Map([
  [
    Risk_Source.DDL,
    [
      ...commonFactorList,
      ...schemaObjectNameFactorList,
      ...migrationFactorList,
    ],
  ],
  [
    Risk_Source.DML,
    [
      ...commonFactorList,
      ...schemaObjectNameFactorList,
      ...migrationFactorList,
    ],
  ],
  [
    Risk_Source.CREATE_DATABASE,
    [...commonFactorList, CEL_ATTRIBUTE_RESOURCE_DATABASE_NAME],
  ],
  [
    Risk_Source.DATA_EXPORT,
    [...commonFactorList, ...schemaObjectNameFactorList],
  ],
  [
    Risk_Source.REQUEST_ROLE,
    [
      CEL_ATTRIBUTE_RESOURCE_ENVIRONMENT_ID,
      CEL_ATTRIBUTE_RESOURCE_PROJECT_ID,
      CEL_ATTRIBUTE_REQUEST_EXPIRATION_DAYS,
      CEL_ATTRIBUTE_REQUEST_ROLE,
    ],
  ],
]);

export const getFactorList = (source: Risk_Source) => {
  return RiskSourceFactorMap.get(source) ?? [];
};

export const getRenderOptionFunc = (resource: {
  title: string | (() => VNode);
  name: string;
}): ((info: { node: VNode; selected: boolean }) => VNode) => {
  return (info: { node: VNode; selected: boolean }) => {
    return h(
      info.node,
      { class: "flex items-center justify-between gap-x-4" },
      [
        h("div", { class: "flex flex-col px-1 py-1 z-10" }, [
          typeof resource.title === "string"
            ? h(
                "div",
                { class: `textlabel ${info.selected ? "text-accent!" : ""}` },
                resource.title
              )
            : resource.title(),
          h("div", { class: "opacity-60 textinfolabel" }, resource.name),
        ]),
        info.selected ? h(CheckIcon, { class: "w-4 z-10" }) : undefined,
      ]
    );
  };
};

export const getEnvironmentIdOptions = () => {
  const environmentList = useEnvironmentV1Store().getEnvironmentList();
  return environmentList.map<SelectOption>((env) => {
    const environmentId = env.id;
    return {
      label: `${env.title} (${environmentId})`,
      value: environmentId,
      render: getRenderOptionFunc({
        name: env.name,
        title: () =>
          h(EnvironmentV1Name, {
            environment: env,
            link: false,
            showColor: true,
          }),
      }),
    };
  });
};

export const getProjectIdOptions = (projects: Project[]) => {
  return projects
    .filter((proj) => proj.name != DEFAULT_PROJECT_NAME)
    .map<SelectOption>((proj) => {
      const projectId = extractProjectResourceName(proj.name);
      return {
        label: `${proj.title} (${projectId})`,
        value: projectId,
        render: getRenderOptionFunc(proj),
      };
    });
};

export const getDatabaseIdOptions = (databases: ComposedDatabase[]) => {
  return databases.map<SelectOption>((database) => {
    return {
      label: database.databaseName,
      value: database.databaseName,
      render: getRenderOptionFunc({
        name: database.name,
        title: () =>
          h(RichDatabaseName, {
            database,
            showEngineIcon: true,
            showInstance: false,
            showProject: false,
            showArrow: false,
          }),
      }),
    };
  });
};

const getDBEndingOptions = () => {
  return supportedEngineV1List().map<SelectOption>((type) => ({
    label: engineNameV1(type),
    value: Engine[type],
  }));
};

const getLevelOptions = () => {
  return PresetRiskLevelList.map<SelectOption>(({ level }) => ({
    label: levelText(level),
    value: level,
  }));
};

const getSourceOptions = () => {
  return useSupportedSourceList().value.map<SelectOption>((source) => ({
    label: Risk_Source[source],
    value: Risk_Source[source],
  }));
};

const getSQLTypeOptions = (source: Risk_Source) => {
  const mapOptions = (values: readonly string[]) => {
    return values.map<SelectOption>((v) => ({
      label: v,
      value: v,
    }));
  };
  switch (source) {
    case Risk_Source.DDL:
      return mapOptions(SQLTypeList.DDL);
    case Risk_Source.DML:
      return mapOptions(SQLTypeList.DML);
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

export const getOptionConfigMap = (source: Risk_Source) => {
  const factorList = getFactorList(source);
  return factorList.reduce((map, factor) => {
    let options: SelectOption[] = [];
    switch (factor) {
      case CEL_ATTRIBUTE_RESOURCE_ENVIRONMENT_ID:
        options = getEnvironmentIdOptions();
        break;
      case CEL_ATTRIBUTE_RESOURCE_PROJECT_ID:
        const projectStore = useProjectV1Store();
        map.set(factor, {
          remote: true,
          options: [],
          search: async (keyword: string) => {
            return projectStore
              .fetchProjectList({
                pageSize: getDefaultPagination(),
                filter: {
                  query: keyword,
                },
              })
              .then((resp) => getProjectIdOptions(resp.projects));
          },
        });
        return map;
      case CEL_ATTRIBUTE_RESOURCE_INSTANCE_ID:
        const store = useInstanceV1Store();
        map.set(factor, {
          remote: true,
          options: [],
          search: async (keyword: string) => {
            return store
              .fetchInstanceList({
                pageSize: getDefaultPagination(),
                filter: {
                  query: keyword,
                },
              })
              .then((resp) => getInstanceIdOptions(resp.instances));
          },
        });
        return map;
      case CEL_ATTRIBUTE_RESOURCE_DB_ENGINE:
        options = getDBEndingOptions();
        break;
      case CEL_ATTRIBUTE_LEVEL:
        options = getLevelOptions();
        break;
      case CEL_ATTRIBUTE_SOURCE:
        options = getSourceOptions();
        break;
      case CEL_ATTRIBUTE_STATEMENT_SQL_TYPE:
        options = getSQLTypeOptions(source);
        break;
      case CEL_ATTRIBUTE_REQUEST_ROLE:
        options = getRoleOptions();
        break;
      default:
        break;
    }

    map.set(factor, { options, remote: false });

    return map;
  }, new Map<Factor, OptionConfig>());
};

// Overload for new approval rule source enum
export const approvalSourceText = (
  source: WorkspaceApprovalSetting_Rule_Source
) => {
  switch (source) {
    case WorkspaceApprovalSetting_Rule_Source.SOURCE_UNSPECIFIED:
      return t("common.all");
    case WorkspaceApprovalSetting_Rule_Source.DDL:
      return t("custom-approval.risk-rule.risk.namespace.ddl");
    case WorkspaceApprovalSetting_Rule_Source.DML:
      return t("custom-approval.risk-rule.risk.namespace.dml");
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
    WorkspaceApprovalSetting_Rule_Source.DDL,
    RiskSourceFactorMap.get(Risk_Source.DDL) || [],
  ],
  [
    WorkspaceApprovalSetting_Rule_Source.DML,
    RiskSourceFactorMap.get(Risk_Source.DML) || [],
  ],
  [
    WorkspaceApprovalSetting_Rule_Source.CREATE_DATABASE,
    RiskSourceFactorMap.get(Risk_Source.CREATE_DATABASE) || [],
  ],
  [
    WorkspaceApprovalSetting_Rule_Source.EXPORT_DATA,
    RiskSourceFactorMap.get(Risk_Source.DATA_EXPORT) || [],
  ],
  [
    WorkspaceApprovalSetting_Rule_Source.REQUEST_ROLE,
    RiskSourceFactorMap.get(Risk_Source.REQUEST_ROLE) || [],
  ],
]);

export const getApprovalFactorList = (
  source: WorkspaceApprovalSetting_Rule_Source
) => {
  return ApprovalSourceFactorMap.get(source) ?? [];
};

export const getApprovalOptionConfigMap = (
  source: WorkspaceApprovalSetting_Rule_Source
) => {
  const riskSource = approvalSourceToRiskSource(source);
  return getOptionConfigMap(riskSource);
};

const approvalSourceToRiskSource = (
  source: WorkspaceApprovalSetting_Rule_Source
): Risk_Source => {
  switch (source) {
    case WorkspaceApprovalSetting_Rule_Source.DDL:
      return Risk_Source.DDL;
    case WorkspaceApprovalSetting_Rule_Source.DML:
      return Risk_Source.DML;
    case WorkspaceApprovalSetting_Rule_Source.CREATE_DATABASE:
      return Risk_Source.CREATE_DATABASE;
    case WorkspaceApprovalSetting_Rule_Source.EXPORT_DATA:
      return Risk_Source.DATA_EXPORT;
    case WorkspaceApprovalSetting_Rule_Source.REQUEST_ROLE:
      return Risk_Source.REQUEST_ROLE;
    default:
      return Risk_Source.SOURCE_UNSPECIFIED;
  }
};
