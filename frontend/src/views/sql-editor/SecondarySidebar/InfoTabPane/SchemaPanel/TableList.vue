<template>
  <div class="flex flex-col overflow-hidden">
    <VirtualList
      :items="filteredTableOrViewList"
      :key-field="`key`"
      :item-resizable="true"
      :item-size="24"
    >
      <template #default="{ item, index }: VirtualListItem">
        <div
          class="text-sm leading-6 px-1"
          :class="`bb-table-list--item`"
          :data-key="item.key"
          :data-index="index"
          @mouseenter="handleMouseEnter($event, item)"
          @mouseleave="handleMouseLeave($event, item)"
        >
          <div
            class="flex items-center text-gray-600 whitespace-pre-wrap break-words rounded-sm px-1"
            :class="
              rowClickable && ['hover:bg-control-bg-hover/50', 'cursor-pointer']
            "
            @click="handleClickItem(item)"
          >
            <TableIcon v-if="item.table" class="w-4 h-4 mr-1 shrink-0" />
            <ViewIcon v-if="item.view" class="w-4 h-4 mr-1 shrink-0" />
            <!-- eslint-disable-next-line vue/no-v-html -->
            <div v-html="renderItem(item)" />
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
import { TableIcon, ViewIcon } from "@/components/Icon";
import { ComposedDatabase } from "@/types";
import {
  DatabaseMetadata,
  SchemaMetadata,
  TableMetadata,
  ViewMetadata,
} from "@/types/proto/v1/database_service";
import { findAncestor, getHighlightHTMLByRegExp } from "@/utils";
import { useHoverStateContext } from "./HoverPanel";
import { useSchemaPanelContext } from "./context";

export type SchemaAndTableOrView = {
  key: string;
  schema: SchemaMetadata;
  table?: TableMetadata;
  view?: ViewMetadata;
};

export type VirtualListItem = {
  item: SchemaAndTableOrView;
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

const flattenTableOrViewList = computed(() => {
  const { schemaList } = props;
  const tables = schemaList.flatMap((schema) => {
    return schema.tables.map<SchemaAndTableOrView>((table) => ({
      key: `schemas/${schema.name}/tables/${table.name}`,
      schema,
      table,
    }));
  });
  const views = schemaList.flatMap((schema) => {
    return schema.views.map<SchemaAndTableOrView>((view) => ({
      key: `schemas/${schema.name}/views/${view.name}`,
      schema,
      view,
    }));
  });
  return [...tables, ...views];
});

const filteredTableOrViewList = computed(() => {
  const kw = keyword.value.toLowerCase().trim();
  if (!kw) {
    return flattenTableOrViewList.value;
  }
  return flattenTableOrViewList.value.filter(({ schema, table, view }) => {
    return (
      schema.name.toLowerCase().includes(kw) ||
      table?.name.toLowerCase().includes(kw) ||
      view?.name.toLowerCase().includes(kw)
    );
  });
});

const handleClickItem = (item: SchemaAndTableOrView) => {
  if (!props.rowClickable) {
    return;
  }
  const { schema, table } = item;
  if (table) {
    emit("select-table", schema, table);
  }
};

const renderItem = (item: SchemaAndTableOrView) => {
  const parts: string[] = [];
  const { schema, table, view } = item;
  const name = table?.name ?? view?.name ?? "";
  if (!name) return null;
  if (schema.name) {
    parts.push(`${schema.name}.`);
  }
  parts.push(name);
  if (!keyword.value.trim()) {
    return parts.join("");
  }

  return getHighlightHTMLByRegExp(
    escape(parts.join("")),
    escape(keyword.value.trim()),
    false /* !caseSensitive */
  );
};

const handleMouseEnter = (e: MouseEvent, item: SchemaAndTableOrView) => {
  const { db, database } = props;
  const { schema, table, view } = item;

  if (hoverState.value) {
    updateHoverState(
      { db, database, schema, table, view },
      "before",
      0 /* overrideDelay */
    );
  } else {
    updateHoverState({ db, database, schema, table, view }, "before");
  }
  nextTick().then(() => {
    // Find the node element and put the database panel to the top-left corner
    // of the node
    const wrapper = findAncestor(
      e.target as HTMLElement,
      ".bb-table-list--item"
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

const handleMouseLeave = (e: MouseEvent, item: SchemaAndTableOrView) => {
  updateHoverState(undefined, "after");
};
</script>
