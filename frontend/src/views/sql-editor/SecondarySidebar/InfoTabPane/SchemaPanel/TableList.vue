<template>
  <div class="flex flex-col overflow-hidden">
    <VirtualList
      :items="filteredTableList"
      :key-field="`key`"
      :item-resizable="true"
      :item-size="24"
    >
      <template
        #default="{ item: { key, schema, table }, index }: VirtualListItem"
      >
        <div class="text-sm leading-6 px-2" :data-key="key" :data-index="index">
          <div
            class="flex items-center text-gray-600 whitespace-pre-wrap break-words rounded-sm"
            :class="
              rowClickable && ['hover:bg-[rgb(243,243,245)]', 'cursor-pointer']
            "
            @click="handleClickTable(schema, table)"
          >
            <heroicons-outline:table class="h-4 w-4 mr-1 shrink-0" />
            <!-- eslint-disable-next-line vue/no-v-html -->
            <div v-html="renderTableName(schema, table)" />
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
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto/v1/database_service";
import { getHighlightHTMLByRegExp } from "@/utils";
import { useSchemaPanelContext } from "./context";

export type SchemaAndTable = {
  key: string;
  schema: SchemaMetadata;
  table: TableMetadata;
};

export type VirtualListItem = {
  item: SchemaAndTable;
  index: number;
};

const props = defineProps<{
  schemaList: SchemaMetadata[];
  rowClickable?: boolean;
}>();

const emit = defineEmits<{
  (e: "select-table", schema: SchemaMetadata, table: TableMetadata): void;
}>();

const { keyword } = useSchemaPanelContext();

const flattenTableList = computed(() => {
  return props.schemaList.flatMap((schema) => {
    return schema.tables.map<SchemaAndTable>((table) => ({
      key: `${schema.name}.${table.name}`,
      schema,
      table,
    }));
  });
});

const filteredTableList = computed(() => {
  const kw = keyword.value.toLowerCase().trim();
  if (!kw) {
    return flattenTableList.value;
  }
  return flattenTableList.value.filter(({ schema, table }) => {
    return schema.name.includes(kw) || table.name.includes(kw);
  });
});

const handleClickTable = (schema: SchemaMetadata, table: TableMetadata) => {
  if (!props.rowClickable) {
    return;
  }
  emit("select-table", schema, table);
};

const renderTableName = (schema: SchemaMetadata, table: TableMetadata) => {
  const parts: string[] = [];
  if (schema.name) {
    parts.push(`${schema.name}.`);
  }
  parts.push(table.name);
  if (!keyword.value.trim()) {
    return parts.join("");
  }

  return getHighlightHTMLByRegExp(
    escape(parts.join("")),
    escape(keyword.value.trim()),
    false /* !caseSensitive */
  );
};
</script>
