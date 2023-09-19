<template>
  <div
    role="table"
    class="bb-grid border-block-border"
    :class="[showHeader ? 'show-header' : 'hide-header', gridClass]"
    v-bind="$attrs"
  >
    <template v-if="customHeader">
      <slot name="header" />
    </template>
    <div
      v-else-if="showHeader"
      role="table-row"
      class="bb-grid-row bb-grid-header-row group"
    >
      <div
        v-for="(column, i) in columnList"
        :key="i"
        role="table-cell"
        class="bb-grid-header-cell"
        :class="[headerClass, column.class]"
      >
        {{ column.title }}
      </div>
    </div>

    <template v-if="ready">
      <template v-for="(item, row) in dataSource" :key="rowKeyOf(item, row)">
        <div
          row="table-row"
          class="bb-grid-row group"
          :class="{
            clickable: rowClickable && isRowClickable(item, row),
          }"
          @click="handleClick(item, 0, row, $event)"
        >
          <slot name="item" :item="item" :row="row" />
        </div>
        <div
          v-if="isRowExpanded(item, row)"
          row="table-row"
          class="bb-grid-row"
        >
          <div
            class="bb-grid-cell"
            :class="expandedRowClass"
            :style="{
              gridColumnStart: 1,
              gridColumnEnd: columnList.length + 1,
            }"
          >
            <slot name="expanded-item" :item="item" :row="row" />
          </div>
        </div>
      </template>
    </template>

    <slot name="placeholder">
      <div
        v-if="ready && dataSource.length === 0 && showPlaceholder"
        class="flex flex-col items-center justify-center text-control-placeholder border-t"
        :style="{
          'grid-column': `auto / span ${columnList.length}`,
        }"
      >
        <slot name="placeholder-content">
          <p class="py-8">{{ $t("common.no-data") }}</p>
        </slot>
      </div>
    </slot>
    <slot name="loading">
      <div
        v-if="!ready"
        class="flex flex-col items-center justify-center text-control-placeholder border-t"
        :style="{
          'grid-column': `auto / span ${columnList.length}`,
        }"
      >
        <p class="py-8">
          <BBSpin />
        </p>
      </div>
    </slot>
  </div>

  <slot name="footer" />
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { VueClass } from "@/utils";
import { BBGridColumn } from "../types";
import { useResponsiveGridColumns } from "./useResponsiveGridColumns";

type DataType = any; // vue does not support generic typed components yet

const emit = defineEmits<{
  (
    event: "click-row",
    item: DataType,
    section: number,
    row: number,
    e: MouseEvent
  ): void;
}>();

const props = withDefaults(
  defineProps<{
    columnList?: BBGridColumn[];
    dataSource?: DataType[];
    rowKey?: string | ((item: DataType, row: number) => any);
    showHeader?: boolean;
    customHeader?: boolean;
    headerClass?: VueClass;
    rowClickable?: boolean;
    showPlaceholder?: boolean;
    ready?: boolean;
    isRowExpanded?: (item: DataType, row: number) => boolean;
    isRowClickable?: (item: DataType, row: number) => boolean;
    expandedRowClass?: VueClass;
  }>(),
  {
    columnList: () => [],
    dataSource: () => [],
    rowKey: undefined,
    showHeader: true,
    customHeader: false,
    headerClass: undefined,
    rowClickable: true,
    showPlaceholder: false,
    ready: true,
    isRowExpanded: () => false,
    isRowClickable: () => true,
    expandedRowClass: undefined,
  }
);

const gridClass = useResponsiveGridColumns(
  computed(() => props.columnList.map((col) => col.width))
);

const rowKeyOf = (item: DataType, row: number) => {
  const { rowKey } = props;
  if (typeof rowKey === "function") {
    return rowKey(item, row);
  }
  if (typeof rowKey === "string") {
    return item[rowKey];
  }
  return row;
};

const handleClick = (
  item: DataType,
  section: number,
  row: number,
  e: MouseEvent
) => {
  if (props.rowClickable && props.isRowClickable(item, row)) {
    emit("click-row", item, section, row, e);
  }
};
</script>
