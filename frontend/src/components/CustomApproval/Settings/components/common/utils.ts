import { uniq, without } from "lodash-es";
import type { SelectOption } from "naive-ui";
import { type Factor, SQLTypeList } from "@/plugins/cel";
import { t } from "@/plugins/i18n";
import { useEnvironmentV1Store, useProjectV1List } from "@/store";
import {
  PresetRiskLevelList,
  DEFAULT_PROJECT_NAME,
  useSupportedSourceList,
} from "@/types";
import type { Risk } from "@/types/proto/v1/risk_service";
import { Risk_Source, risk_SourceToJSON } from "@/types/proto/v1/risk_service";
import {
  engineNameV1,
  extractEnvironmentResourceName,
  extractProjectResourceName,
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
    case Risk_Source.REQUEST_QUERY:
      return t("custom-approval.risk-rule.risk.namespace.request_query");
    case Risk_Source.REQUEST_EXPORT:
      return t("custom-approval.risk-rule.risk.namespace.request_export");
    default:
      return Risk_Source.UNRECOGNIZED;
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

const NumberFactorList = [
  // Risk related factors
  "affected_rows",
  "table_rows",
  "level",
  "source",
  "expiration_days",
  "export_rows",
] as const;

const StringFactorList = [
  "environment_id", // using `environment.resource_id`
  "project_id", // using `project.resource_id`
  "database_name",
  "db_engine",
  "sql_type",
  "table_name",
] as const;

export const RiskSourceFactorMap: Map<Risk_Source, string[]> = new Map([
  [
    Risk_Source.DDL,
    uniq(
      without(
        [...NumberFactorList, ...StringFactorList],
        "level",
        "source",
        "expiration_days",
        "export_rows"
      )
    ),
  ],
  [
    Risk_Source.DML,
    uniq(
      without(
        [...NumberFactorList, ...StringFactorList],
        "level",
        "source",
        "expiration_days",
        "export_rows"
      )
    ),
  ],
  [
    Risk_Source.CREATE_DATABASE,
    uniq(
      without(
        [...StringFactorList],
        "sql_type",
        "table_name",
        "expiration_days",
        "export_rows"
      )
    ),
  ],
  [
    Risk_Source.DATA_EXPORT,
    uniq(
      without(
        [...StringFactorList, ...NumberFactorList],
        "level",
        "affected_rows",
        "table_rows",
        "source",
        "sql_type",
        "table_name",
        "expiration_days",
        "export_rows"
      )
    ),
  ],
  [
    Risk_Source.REQUEST_QUERY,
    uniq(
      without(
        [...StringFactorList, ...NumberFactorList],
        "level",
        "source",
        "affected_rows",
        "table_rows",
        "sql_type",
        "table_name",
        "export_rows"
      )
    ),
  ],
  [
    Risk_Source.REQUEST_EXPORT,
    uniq(
      without(
        [...StringFactorList, ...NumberFactorList],
        "level",
        "source",
        "affected_rows",
        "table_rows",
        "sql_type",
        "table_name"
      )
    ),
  ],
]);

export const getFactorList = (source: Risk_Source) => {
  return RiskSourceFactorMap.get(source) ?? [];
};

const getEnvironmentIdOptions = () => {
  const environmentList = useEnvironmentV1Store().getEnvironmentList();
  return environmentList.map<SelectOption>((env) => {
    const environmentName = extractEnvironmentResourceName(env.name);
    return {
      label: environmentName,
      value: environmentName,
    };
  });
};

const getProjectIdOptions = () => {
  const { projectList } = useProjectV1List();
  return projectList.value
    .filter((proj) => proj.name != DEFAULT_PROJECT_NAME)
    .map<SelectOption>((proj) => {
      const projectId = extractProjectResourceName(proj.name);
      return {
        label: `${projectId} (${proj.title})`,
        value: projectId,
      };
    });
};

const getDBEndingOptions = () => {
  return supportedEngineV1List().map<SelectOption>((type) => ({
    label: engineNameV1(type),
    value: type,
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
    label: risk_SourceToJSON(source),
    value: source,
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

export const getFactorOptionsMap = (source: Risk_Source) => {
  const factorList = getFactorList(source);
  return factorList.reduce((map, factor) => {
    let options: SelectOption[] = [];
    switch (factor) {
      case "environment_id":
        options = getEnvironmentIdOptions();
        break;
      case "project_id":
        options = getProjectIdOptions();
        break;
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
    }
    map.set(factor, options);
    return map;
  }, new Map<Factor, SelectOption[]>());
};

export const factorSupportDropdown: Factor[] = [
  "environment_id",
  "project_id",
  "db_engine",
  "sql_type",
  "level",
  "source",
];
