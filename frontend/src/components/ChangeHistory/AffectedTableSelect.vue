<template>
  <NSelect
    :value="selectedKey"
    :options="affectedTableOptions"
    :filter="filterByTitle"
    filterable
    :render-label="renderLabel"
    @update:value="updateSelectedKey"
  />
</template>

<script setup lang="tsx">
import { isEqual, orderBy, uniqBy } from "lodash-es";
import { NSelect } from "naive-ui";
import type { SelectOption } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import type { AffectedTable } from "@/types";
import { EmptyAffectedTable } from "@/types";
import type { ChangeHistory } from "@/types/proto/v1/database_service";
import {
  getAffectedTablesOfChangeHistory,
  stringifyAffectedTable,
} from "@/utils";

type AffectedTableSelectOption = SelectOption & {
  affectedTable: AffectedTable;
  value: string;
};

const props = defineProps<{
  changeHistoryList: ChangeHistory[];
  affectedTable?: AffectedTable;
}>();
const emit = defineEmits<{
  (event: "update:affected-table", affectedTable?: AffectedTable): void;
}>();

const { t } = useI18n();

const affectedTables = computed(() => {
  return [
    EmptyAffectedTable,
    ...orderBy(
      uniqBy(
        props.changeHistoryList
          .map((changeHistory) =>
            getAffectedTablesOfChangeHistory(changeHistory)
          )
          .flat(),
        (affectedTable) => `${affectedTable.schema}.${affectedTable.table}`
      ),
      ["dropped", "table", "schema"]
    ),
  ];
});

const filterByTitle = (pattern: string, option: SelectOption) => {
  const { affectedTable } = option as AffectedTableSelectOption;
  pattern = pattern.toLowerCase();
  return affectedTable.table.toLowerCase().includes(pattern);
};

const affectedTableOptions = computed(() => {
  return affectedTables.value.map<AffectedTableSelectOption>((item) => {
    const key = isEqual(item, EmptyAffectedTable) ? "" : JSON.stringify(item);
    return {
      value: key,
      affectedTable: item,
    };
  });
});

const renderLabel = (option: SelectOption) => {
  const { affectedTable } = option as AffectedTableSelectOption;
  if (!affectedTable || isEqual(affectedTable, EmptyAffectedTable)) {
    return t("change-history.all-tables");
  }
  const name = stringifyAffectedTable(affectedTable);
  if (affectedTable.dropped) {
    return (
      <div class="w-full flex flex-row justify-between items-center gap-1">
        <span class="truncate">{name}</span>
        <span class="shrink-0">(Dropped)</span>
      </div>
    );
  }
  return <span class="truncate">{name}</span>;
};

const selectedKey = computed(() => {
  return props.affectedTable ? JSON.stringify(props.affectedTable) : "";
});

const updateSelectedKey = (_: string, opt: AffectedTableSelectOption) => {
  emit(
    "update:affected-table",
    isEqual(opt.affectedTable, EmptyAffectedTable)
      ? undefined
      : opt.affectedTable
  );
};
</script>
