<template>
  <div role="table" class="bb-grid border-block-border" :class="gridClass">
    <template v-if="customHeader">
      <slot name="header" />
    </template>
    <div v-else role="table-row" class="bb-grid-row bb-grid-header-row group">
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
      :key="row"
      row="table-row"
      class="bb-grid-row group"
      :class="{
        clickable: rowClickable,
      }"
      @click="handleClick(item, 0, row, $event)"
    >
      <slot name="item" :item="item" :row="row" />
    </div>

    <slot name="footer" />
  </div>
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
    showHeader?: boolean;
    customHeader?: boolean;
    headerClass?: VueClass;
    rowClickable?: boolean;
    showPlaceholder?: boolean;
  }>(),
  {
    columnList: () => [],
    dataSource: () => [],
    showHeader: true,
    customHeader: false,
    headerClass: undefined,
    rowClickable: true,
    showPlaceholder: true,
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
