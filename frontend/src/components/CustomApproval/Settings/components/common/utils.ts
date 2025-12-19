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
} from "@/types";
import { Engine, RiskLevel } from "@/types/proto-es/v1/common_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
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

export const getDatabasFullNameOptions = (databases: ComposedDatabase[]) => {
  return databases.map<SelectOption>((database) => {
    return {
      label: database.name,
      value: database.name,
      render: getRenderOptionFunc({
        name: database.name,
        title: () =>
          h(RichDatabaseName, {
            database,
            showEngineIcon: true,
            showInstance: true,
            showProject: false,
            showArrow: true,
          }),
      }),
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
  return levels.map<SelectOption>(({ label, value }) => ({
    label,
    value,
  }));
};

const getSQLTypeOptions = (source: WorkspaceApprovalSetting_Rule_Source) => {
  const mapOptions = (values: readonly string[]) => {
    return values.map<SelectOption>((v) => ({
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

    map.set(factor, { options, remote: false });

    return map;
  }, new Map<Factor, OptionConfig>());
};
