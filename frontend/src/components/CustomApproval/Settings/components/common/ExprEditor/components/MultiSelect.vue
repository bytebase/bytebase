<template>
  <NSelect
    :value="value"
    :options="options"
    :multiple="true"
    :consistent-menu-width="false"
    :placeholder="$t('custom-approval.security-rule.condition.select-value')"
    :disabled="!allowAdmin"
    max-tag-count="responsive"
    size="small"
    style="min-width: 7rem; width: auto; overflow-x: hidden"
    @update:value="$emit('update:value', $event)"
  >
    <template #action>
      <NCheckbox
        :label="$t('common.all')"
        v-bind="checkAllState"
        @update:checked="toggleCheckAll"
      />
    </template>
  </NSelect>
</template>

<script lang="ts" setup>
import { computed, watch } from "vue";
import { NCheckbox, NSelect, SelectOption } from "naive-ui";

import { type ConditionExpr, SQLTypeList } from "@/plugins/cel";
import { useCurrentUser, useEnvironmentStore, useProjectStore } from "@/store";
import {
  engineName,
  EngineTypeList,
  PresetRiskLevelList,
  SupportedSourceList,
} from "@/types";
import { Risk_Source, risk_SourceToJSON } from "@/types/proto/v1/risk_service";
import { useExprEditorContext } from "../context";

const props = defineProps<{
  value: string[] | number[];
  expr: ConditionExpr;
}>();

const emit = defineEmits<{
  (event: "update:value", value: string[] | number[]): void;
}>();

const context = useExprEditorContext();
const { allowAdmin, riskSource } = context;

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
  return EngineTypeList.map<SelectOption>((type) => ({
    label: engineName(type),
    value: type,
  }));
};

const getRiskOptions = () => {
  return PresetRiskLevelList.map<SelectOption>(({ name, level }) => ({
    label: name,
    value: level,
  }));
};

const getSourceOptions = () => {
  return SupportedSourceList.map<SelectOption>((source) => ({
    label: risk_SourceToJSON(source),
    value: source,
  }));
};

const options = computed(() => {
  const factor = props.expr.args[0];
  if (factor === "environment") {
    return getEnvironmentOptions();
  }
  if (factor === "project") {
    return getProjectOptions();
  }
  if (factor === "db_engine") {
    return getDBEndingOptions();
  }
  if (factor === "risk") {
    return getRiskOptions();
  }
  if (factor === "source") {
    return getSourceOptions();
  }

  const mapOptions = (values: readonly string[]) => {
    return values.map<SelectOption>((v) => ({
      label: v,
      value: v,
    }));
  };
  if (factor === "sql_type") {
    const source = riskSource.value;
    switch (source) {
      case Risk_Source.DDL:
        return mapOptions(SQLTypeList.DDL);
      case Risk_Source.DML:
        return mapOptions(SQLTypeList.DML);
      default:
        // unsupported namespace
        return [];
    }
  }
  return [];
});

const checkAllState = computed(() => {
  const selected = new Set<any>(props.value);
  const checked = options.value.every((opt) => {
    return selected.has(opt.value);
  });
  const indeterminate = props.value.length > 0 && !checked;
  return {
    checked,
    indeterminate,
  };
});

const toggleCheckAll = (on: boolean) => {
  if (on) {
    emit(
      "update:value",
      options.value.map((opt) => opt.value as string)
    );
  } else {
    emit("update:value", []);
  }
};

watch(
  [options, () => props.value],
  () => {
    const values = new Set(options.value.map((opt) => opt.value));
    const filtered = (props.value as any[]).filter((v) => values.has(v));
    if (filtered.length !== props.value.length) {
      // Some values are not suitable for the select options.
      emit("update:value", filtered);
    }
  },
  { immediate: true }
);
</script>
