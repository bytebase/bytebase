<template>
  <NSelect
    :value="selectedKey"
    :options="affectedTableOptions"
    @update:value="updateSelectedKey"
  />
</template>

<script setup lang="ts">
import { orderBy, uniqBy } from "lodash-es";
import { SelectOption } from "naive-ui";
import { computed, h } from "vue";
import { AffectedTable, EmptyAffectedTable } from "@/types";
import { ChangeHistory } from "@/types/proto/v1/database_service";
import {
  getAffectedTableDisplayName,
  getAffectedTablesOfChangeHistory,
} from "@/utils";

type AffectedTableSelectOption = SelectOption & {
  affectedTable: AffectedTable;
  value: string;
};

const props = defineProps<{
  affectedTable: AffectedTable;
  changeHistoryList: ChangeHistory[];
}>();
const emit = defineEmits<{
  (event: "update:affected-table", affectedTable: AffectedTable): void;
}>();

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

const affectedTableOptions = computed(() => {
  return affectedTables.value.map<AffectedTableSelectOption>((item) => {
    const key = JSON.stringify(item);
    const name = getAffectedTableDisplayName(item);
    return {
      label: name,
      value: key,
      affectedTable: item,
      renderLabel() {
        const classes = ["truncate"];
        if (item.dropped) {
          classes.push("text-gray-400");
        }
        return h("span", { class: classes, "data-key": key }, name);
      },
    };
  });
});

const selectedKey = computed(() => {
  return JSON.stringify(props.affectedTable);
});

const updateSelectedKey = (key: string, opt: AffectedTableSelectOption) => {
  emit("update:affected-table", opt.affectedTable);
};
</script>
