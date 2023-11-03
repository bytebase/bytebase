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
        <div
          class="bb-table-list--table-item text-sm leading-6 px-1"
          :data-key="key"
          :data-index="index"
          @mouseenter="handleMouseEnter($event, schema, table)"
          @mouseleave="handleMouseLeave($event, schema, table)"
        >
          <div
            class="flex items-center text-gray-600 whitespace-pre-wrap break-words rounded-sm px-1"
            :class="
              rowClickable && ['hover:bg-control-bg-hover/50', 'cursor-pointer']
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
import { computed, nextTick } from "vue";
import { VirtualList } from "vueuc";
import { ComposedDatabase } from "@/types";
import {
  DatabaseMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto/v1/database_service";
import { findAncestor, getHighlightHTMLByRegExp } from "@/utils";
import { useHoverStateContext } from "./HoverPanel";
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
  db: ComposedDatabase;
  database: DatabaseMetadata;
  schemaList: SchemaMetadata[];
  rowClickable?: boolean;
}>();

const emit = defineEmits<{
  (e: "select-table", schema: SchemaMetadata, table: TableMetadata): void;
}>();

const { keyword } = useSchemaPanelContext();
const {
  state: hoverState,
  position: hoverPosition,
  update: updateHoverState,
} = useHoverStateContext();

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
    return (
      schema.name.toLowerCase().includes(kw) ||
      table.name.toLowerCase().includes(kw)
    );
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

const handleMouseEnter = (
  e: MouseEvent,
  schema: SchemaMetadata,
  table: TableMetadata
) => {
  const { db, database } = props;
  if (hoverState.value) {
    updateHoverState(
      { db, database, schema, table },
      "before",
      0 /* overrideDelay */
    );
  } else {
    updateHoverState({ db, database, schema, table }, "before");
  }
  nextTick().then(() => {
    // Find the node element and put the database panel to the top-left corner
    // of the node
    const wrapper = findAncestor(
      e.target as HTMLElement,
      ".bb-table-list--table-item"
    );
    if (!wrapper) {
      updateHoverState(undefined, "after", 0 /* overrideDelay */);
      return;
    }
    const bounding = wrapper.getBoundingClientRect();
    hoverPosition.value.x = bounding.left;
    hoverPosition.value.y = bounding.top;
  });
};

const handleMouseLeave = (
  e: MouseEvent,
  schema: SchemaMetadata,
  table: TableMetadata
) => {
  updateHoverState(undefined, "after");
};
</script>
