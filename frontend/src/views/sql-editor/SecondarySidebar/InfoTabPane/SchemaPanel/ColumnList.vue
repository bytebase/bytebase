<template>
  <div class="overflow-hidden">
    <VirtualList
      :items="filteredColumnList"
      :key-field="`name`"
      :item-resizable="true"
      :item-size="24"
    >
      <template #default="{ item: column }: VirtualListItem">
        <div class="flex items-start px-4">
          <!-- eslint-disable vue/no-v-html -->
          <div
            class="text-sm leading-6 text-gray-600whitespace-pre-wrap break-words flex-2 min-w-[4rem]"
            v-html="renderColumnName(column)"
          />
          <div
            class="shrink-0 text-right text-sm leading-6 text-gray-400 overflow-x-hidden whitespace-nowrap flex-1 min-w-[4rem]"
          >
            {{ column.type }}
          </div>
        </div>
      </template>
    </VirtualList>
  </div>
</template>

<script setup lang="ts">
import { escape } from "lodash-es";
import { computed } from "vue";
import { VirtualList } from "vueuc";
import {
  ColumnMetadata,
  TableMetadata,
} from "@/types/proto/v1/database_service";
import { getHighlightHTMLByRegExp } from "@/utils";
import { useSchemaPanelContext } from "./context";

export type VirtualListItem = {
  item: ColumnMetadata;
  index: number;
};

const props = defineProps<{
  table: TableMetadata;
}>();

const { keyword } = useSchemaPanelContext();
const filteredColumnList = computed(() => {
  const kw = keyword.value.toLowerCase().trim();
  if (!kw) {
    return props.table.columns;
  }
  return props.table.columns.filter((column) => {
    return column.name.toLowerCase().includes(kw);
  });
});

const renderColumnName = (column: ColumnMetadata) => {
  if (!keyword.value.trim()) {
    return column.name;
  }

  return getHighlightHTMLByRegExp(
    escape(column.name),
    escape(keyword.value.trim()),
    false /* !caseSensitive */
  );
};
</script>
