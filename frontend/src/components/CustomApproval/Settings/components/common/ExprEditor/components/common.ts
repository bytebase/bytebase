import { type Ref, computed } from "vue";
import type { SelectOption } from "naive-ui";

import { ConditionExpr, Factor, SQLTypeList } from "@/plugins/cel";
import { useExprEditorContext } from "../context";
import { useCurrentUser, useEnvironmentStore, useProjectStore } from "@/store";
import { engineName, PresetRiskLevelList, SupportedSourceList } from "@/types";
import { Risk_Source, risk_SourceToJSON } from "@/types/proto/v1/risk_service";
import { levelText } from "../../utils";
import { supportedEngineList } from "@/utils";

export const useSelectOptions = (expr: Ref<ConditionExpr>) => {
  const context = useExprEditorContext();
  const { riskSource } = context;

  const getEnvironmentOptions = () => {
    const environmentList = useEnvironmentStore().getEnvironmentList();
    return environmentList.map<SelectOption>((env) => ({
      label: env.name,
      value: env.resourceId,
    }));
  };

  const getProjectOptions = () => {
    const user = useCurrentUser().value;
    const projectList = useProjectStore().getProjectListByUser(user.id);
    return projectList.map<SelectOption>((proj) => ({
      label: proj.name,
      value: proj.resourceId,
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
      return getEnvironmentOptions();
    }
    if (factor === "project_id") {
      return getProjectOptions();
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
