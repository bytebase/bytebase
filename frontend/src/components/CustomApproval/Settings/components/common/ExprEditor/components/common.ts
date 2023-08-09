import type { SelectOption } from "naive-ui";
import { type Ref, computed } from "vue";
import { ConditionExpr, Factor, SQLTypeList } from "@/plugins/cel";
import { useEnvironmentV1Store, useProjectV1ListByCurrentUser } from "@/store";
import { engineName, PresetRiskLevelList, SupportedSourceList } from "@/types";
import { Risk_Source, risk_SourceToJSON } from "@/types/proto/v1/risk_service";
import {
  extractEnvironmentResourceName,
  extractProjectResourceName,
  supportedEngineList,
} from "@/utils";
import { levelText } from "../../utils";
import { useExprEditorContext } from "../context";

export const useSelectOptions = (expr: Ref<ConditionExpr>) => {
  const context = useExprEditorContext();
  const { riskSource } = context;

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
    const { projectList } = useProjectV1ListByCurrentUser();
    return projectList.value.map<SelectOption>((proj) => ({
      label: proj.title,
      value: extractProjectResourceName(proj.name),
    }));
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

  const getSQLTypeOptions = () => {
    const source = riskSource.value;
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

  const options = computed(() => {
    const factor = expr.value.args[0];
    if (factor === "environment_id") {
      return getEnvironmentIdOptions();
    }
    if (factor === "project_id") {
      return getProjectIdOptions();
    }
    if (factor === "db_engine") {
      return getDBEndingOptions();
    }
    if (factor === "level") {
      return getLevelOptions();
    }
    if (factor === "source") {
      return getSourceOptions();
    }
    if (factor === "sql_type") {
      return getSQLTypeOptions();
    }
    return [];
  });

  return options;
};

export const factorSupportDropdown = (factor: Factor): boolean => {
  const list: Factor[] = [
    "environment_id",
    "db_engine",
    "sql_type",
    "level",
    "source",
  ];
  return list.includes(factor);
};
