import { CheckIcon } from "lucide-vue-next";
import type { SelectOption } from "naive-ui";
import { h } from "vue";
import type { VNode } from "vue";
import { type OptionConfig } from "@/components/ExprEditor/context";
import { EnvironmentV1Name } from "@/components/v2";
import { SQLTypeList, type Factor } from "@/plugins/cel";
import { t } from "@/plugins/i18n";
import {
  environmentNamePrefix,
  useEnvironmentV1Store,
  useProjectV1Store,
  useRoleStore,
} from "@/store";
import {
  DEFAULT_PROJECT_NAME,
  PRESET_WORKSPACE_ROLES,
  PresetRiskLevelList,
  useSupportedSourceList,
} from "@/types";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import type { Risk } from "@/types/proto-es/v1/risk_service_pb";
import { Risk_Source } from "@/types/proto-es/v1/risk_service_pb";
import {
  displayRoleTitle,
  engineNameV1,
  extractProjectResourceName,
  getDefaultPagination,
  supportedEngineV1List,
} from "@/utils";

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

export const levelText = (level: number) => {
  switch (level) {
    case 0:
      return t("custom-approval.risk-rule.risk.risk-level.default");
    case 100:
      return t("custom-approval.risk-rule.risk.risk-level.low");
    case 200:
      return t("custom-approval.risk-rule.risk.risk-level.moderate");
    case 300:
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

const commonFactorList = [
  "environment_id", // use `environment.resource_id` instead.
  "project_id", // use `project.resource_id` instead.
  "db_engine",
] as const;

const schemaObjectNameFactorList = [
  "database_name",
  "schema_name",
  "table_name",
] as const;

const migrationFactorList = [
  "affected_rows",
  "table_rows",
  "sql_type",
  "sql_statement",
] as const;

export const RiskSourceFactorMap: Map<Risk_Source, string[]> = new Map([
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
  [Risk_Source.CREATE_DATABASE, [...commonFactorList, "database_name"]],
  [
    Risk_Source.DATA_EXPORT,
    [...commonFactorList, ...schemaObjectNameFactorList, "export_rows"],
  ],
  [
    Risk_Source.REQUEST_ROLE,
    [
      "environment_id", // use `environment.resource_id` instead.
      "project_id", // use `project.resource_id` instead.
      "expiration_days",
      "export_rows",
      "role",
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
      { class: "flex items-center justify-between space-x-4" },
      [
        h("div", { class: "flex flex-col px-1 py-1 z-10" }, [
          typeof resource.title === "string"
            ? h(
                "div",
                { class: `textlabel ${info.selected ? "!text-accent" : ""}` },
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
        name: `${environmentNamePrefix}${env.id}`,
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
      case "environment_id":
        options = getEnvironmentIdOptions();
        break;
      case "project_id":
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
      case "db_engine":
        options = getDBEndingOptions();
        break;
      case "level":
        options = getLevelOptions();
        break;
      case "source":
        options = getSourceOptions();
        break;
      case "sql_type":
        options = getSQLTypeOptions(source);
        break;
      case "role":
        options = getRoleOptions();
        break;
    }
    map.set(factor, {
      remote: false,
      options,
    });
    return map;
  }, new Map<Factor, OptionConfig>());
};

export const factorSupportDropdown: Factor[] = [
  "environment_id",
  "project_id",
  "db_engine",
  "sql_type",
  "level",
  "source",
  "role",
];
