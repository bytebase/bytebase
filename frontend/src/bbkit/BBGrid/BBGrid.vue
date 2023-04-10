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
        v-for="(column, row) in columnList"
        :key="row"
        role="table-cell"
        class="bb-grid-header-cell"
        :class="[headerClass, column.class]"
      >
        {{ column.title }}
      </div>
    </div>

    <div
      v-for="(item, row) in dataSource"
      :key="rowKey ? item[rowKey] : row"
      row="table-row"
      class="bb-grid-row group"
      :class="{
        clickable: rowClickable,
      }"
      @click="handleClick(item, 0, row, $event)"
    >
      <slot name="item" :item="item" :row="row" />
    </div>
    <slot name="placeholder">
      <div
        v-if="dataSource.length === 0 && showPlaceholder"
        class="flex flex-col items-center justify-center py-8 text-control-placeholder border-t"
        :style="{
          'grid-column': `auto / span ${columnList.length}`,
        }"
      >
        <p>{{ $t("common.no-data") }}</p>
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
    rowKey?: string;
    showHeader?: boolean;
    customHeader?: boolean;
    headerClass?: VueClass;
    rowClickable?: boolean;
    showPlaceholder?: boolean;
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
  }
);

const gridClass = useResponsiveGridColumns(
  computed(() => props.columnList.map((col) => col.width))
);

const handleClick = (
  item: DataType,
  section: number,
  row: number,
  e: MouseEvent
) => {
  if (props.rowClickable) {
    emit("click-row", item, section, row, e);
  }
};
</script>
