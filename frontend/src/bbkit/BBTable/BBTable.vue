<template>
  <table
    class="min-w-full divide-y divide-block-border"
    :class="borderVisibility"
  >
    <thead v-if="showHeader && !sectionDataSource" class="bg-gray-50">
      <template v-if="customHeader">
        <!-- different implements for legacy compatibility -->
        <slot name="header" />
      </template>
      <tr v-else>
        <th
          v-for="(column, index) in columnList"
          :key="index"
          scope="col"
          class="py-2 text-left text-xs font-medium text-gray-500 tracking-wider"
          :class="[
            headerClass,
            index == 0 ? 'pl-4' : 'pl-2',
            column.center && 'text-center pr-2',
            column.nowrap && 'whitespace-nowrap',
          ]"
        >
          {{ column.title }}
        </th>
      </tr>
    </thead>

    <template v-if="sectionDataSource">
      <template v-for="(section, i) in sectionDataSource" :key="i">
        <tbody class="bg-normal divide-y divide-block-border">
          <th
            v-if="!compactSection || sectionDataSource.length > 1"
            :colspan="columnList.length"
            class="text-left pl-4 pt-4 pb-2 py-text-base leading-6 font-medium text-gray-900"
          >
            <slot name="sectionHeader" :section="section">
              <template v-if="section.link">
                <router-link :to="section.link" class="normal-link">
                  {{ section.title }}
                </router-link>
              </template>
              <template v-else>
                {{ section.title }}
              </template>
            </slot>
          </th>
          <template v-if="section.list.length > 0">
            <tr v-if="showHeader" class="bg-gray-50">
              <slot name="header" />
            </tr>
            <tr
              v-for="(item, j) in section.list"
              :key="j"
              :class="rowClickable ? 'cursor-pointer hover:bg-gray-200' : ''"
              @click.stop="
                (e) => {
                  if (rowClickable) {
                    $emit('click-row', i, j, e);
                  }
                }
              "
            >
              <slot name="body" :row-data="item" />
            </tr>
          </template>
          <template v-else>
            <tr v-if="showPlaceholder">
              <td
                :colspan="columnList.length"
                class="text-center text-gray-400"
              >
                <slot name="placeholder">-</slot>
              </td>
            </tr>
          </template>
        </tbody>
      </template>
    </template>
    <template v-else>
      <tbody class="bg-normal divide-y divide-block-border">
        <tr
          v-for="(item, index) in dataSource"
          :key="index"
          :class="rowClickable ? 'cursor-pointer hover:bg-gray-200' : ''"
          @click.stop="
            (e) => {
              if (rowClickable) {
                $emit('click-row', 0, index, e);
              }
            }
          "
        >
          <slot name="body" :row-data="item" />
        </tr></tbody
    ></template>
    <slot name="footer" />
  </table>
</template>

<script lang="ts" setup>
import { computed, withDefaults } from "vue";
import type { VueClass } from "@/utils";
import { BBTableColumn, BBTableSectionDataSource } from "../types";

type DataType = any; // vue does not support generic typed components yet

const props = withDefaults(
  defineProps<{
    columnList?: BBTableColumn[];
    dataSource?: DataType[];
    sectionDataSource?: BBTableSectionDataSource<DataType>[] | undefined;
    // Only applicable to sectionDataSource. If true, when there is only one
    // section, it won't render the extra section header. In another words, it will
    // just look like a non-section list.
    compactSection?: boolean;
    showHeader?: boolean;
    customHeader?: boolean;
    headerClass?: VueClass;
    rowClickable?: boolean;
    leftBordered?: boolean;
    rightBordered?: boolean;
    topBordered?: boolean;
    bottomBordered?: boolean;
    showPlaceholder?: boolean;
  }>(),
  {
    columnList: () => [],
    dataSource: () => [],
    sectionDataSource: undefined,
    compactSection: false,
    showHeader: true,
    customHeader: false,
    headerClass: undefined,
    rowClickable: true,
    leftBordered: true,
    rightBordered: true,
    topBordered: true,
    bottomBordered: true,
    showPlaceholder: true,
  }
);

defineEmits<{
  (event: "click-row", section: number, row: number, e: MouseEvent): void;
}>();

const borderVisibility = computed(() => {
  const style = [];
  if (props.leftBordered) {
    style.push("border-l");
  }

  if (props.rightBordered) {
    style.push("border-r");
  }

  if (props.topBordered) {
    style.push("border-t");
  }

  if (props.bottomBordered) {
    style.push("border-b");
  }
  return style.join(" ");
});
</script>
