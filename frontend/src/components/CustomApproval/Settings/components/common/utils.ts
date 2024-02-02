import { uniq, without } from "lodash-es";
import type { SelectOption } from "naive-ui";
import { type Factor, SQLTypeList } from "@/plugins/cel";
import { t, te } from "@/plugins/i18n";
import { useEnvironmentV1Store, useProjectV1List } from "@/store";
import {
  engineName,
  PresetRiskLevelList,
  SupportedSourceList,
  DEFAULT_PROJECT_V1_NAME,
} from "@/types";
import {
  Risk,
  Risk_Source,
  risk_SourceToJSON,
} from "@/types/proto/v1/risk_service";
import {
  extractEnvironmentResourceName,
  extractProjectResourceName,
  supportedEngineList,
} from "@/utils";

export const sourceText = (source: Risk_Source) => {
  if (source === Risk_Source.SOURCE_UNSPECIFIED) {
    return t("common.all");
  }

  const name = risk_SourceToJSON(source);
  const keypath = `custom-approval.risk-rule.risk.namespace.${name.toLowerCase()}`;
  if (te(keypath)) {
    return t(keypath);
  }
  return name;
};

export const levelText = (level: number) => {
  const keypath = `custom-approval.risk-rule.risk.risk-level.${level}`;
  if (te(keypath)) {
    return t(keypath);
  }
  return String(level);
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
] as const;

const FactorList = {
  DDL: uniq(
    without(
      [...NumberFactorList, ...StringFactorList],
      "level",
      "source",
      "expiration_days",
      "export_rows"
    )
  ),
  DML: uniq(
    without(
      [...NumberFactorList, ...StringFactorList],
      "level",
      "source",
      "expiration_days",
      "export_rows"
    )
  ),
  CreateDatabase: without(
    [...StringFactorList],
    "sql_type",
    "expiration_days",
    "export_rows"
  ),
  RequestQuery: uniq(
    without(
      [...StringFactorList, ...NumberFactorList],
      "level",
      "source",
      "affected_rows",
      "table_rows",
      "sql_type",
      "export_rows"
    )
  ),
  RequestExport: uniq(
    without(
      [...StringFactorList, ...NumberFactorList],
      "level",
      "source",
      "affected_rows",
      "table_rows",
      "sql_type"
    )
  ),
};

export const getFactorList = (source: Risk_Source) => {
  switch (source) {
    case Risk_Source.DDL:
      return [...FactorList.DDL];
    case Risk_Source.DML:
      return [...FactorList.DML];
    case Risk_Source.CREATE_DATABASE:
      return [...FactorList.CreateDatabase];
    case Risk_Source.QUERY:
      return [...FactorList.RequestQuery];
    case Risk_Source.EXPORT:
      return [...FactorList.RequestExport];
    default:
      // unsupported namespace
      return [];
  }
};

const getEnvironmentIdOptions = () => {
  const environmentList = useEnvironmentV1Store().getEnvironmentList();
  return environmentList.map<SelectOption>((env) => {
    const environmentId = extractEnvironmentResourceName(env.name);
    return {
      label: environmentId,
      value: environmentId,
    };
  });
};

const getProjectIdOptions = () => {
  const { projectList } = useProjectV1List();
  return projectList.value
    .filter((proj) => proj.name != DEFAULT_PROJECT_V1_NAME)
    .map<SelectOption>((proj) => {
      const projectId = extractProjectResourceName(proj.name);
      return {
        label: `${projectId} (${proj.title})`,
        value: projectId,
      };
    });
};

const getDBEndingOptions = () => {
  return supportedEngineList().map<SelectOption>((type) => ({
    label: engineName(type),
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
  return SupportedSourceList.map<SelectOption>((source) => ({
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
