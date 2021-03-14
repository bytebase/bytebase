<template>
  <div class="border-block-border" :class="borderVisibility">
    <table class="min-w-full divide-y divide-block-border">
      <thead v-if="showHeader && !sectionDataSource" class="bg-gray-50">
        <tr>
          <th
            v-for="(column, index) in columnList"
            :key="index"
            scope="col"
            class="py-2 text-left text-xs font-medium text-gray-500 tracking-wider"
            :class="index == 0 ? 'pl-4' : 'pl-2'"
          >
            {{ column.title }}
          </th>
        </tr>
      </thead>

      <template v-if="sectionDataSource">
        <template v-for="(section, i) in sectionDataSource" :key="i">
          <tbody class="bg-normal divide-y divide-block-border">
            <th
              :colspan="columnList.length"
              class="text-left pl-4 pt-4 pb-2 py-text-base leading-6 font-medium text-gray-900"
            >
              {{ section.title }}
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
                  () => {
                    if (rowClickable) {
                      $emit('click-row', i, j);
                    }
                  }
                "
              >
                <slot name="body" :rowData="item" />
              </tr>
            </template>
            <template v-else>
              <tr>
                <td
                  :colspan="columnList.length"
                  class="text-center text-gray-400"
                >
                  -
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
              () => {
                if (rowClickable) {
                  $emit('click-row', 0, index);
                }
              }
            "
          >
            <slot name="body" :rowData="item" />
          </tr></tbody
      ></template>
    </table>
  </div>
</template>

<script lang="ts">
import { computed, PropType } from "vue";
import { BBTableColumn, BBTableSectionDataSource } from "../types";

export default {
  name: "BBTable",
  components: {},
  emits: ["click-row"],
  props: {
    columnList: {
      required: true,
      type: Object as PropType<BBTableColumn[]>,
    },
    dataSource: {
      default: new Array(),
      type: Object as PropType<Object[]>,
    },
    sectionDataSource: {
      type: Object as PropType<BBTableSectionDataSource<Object>[]>,
    },
    showHeader: {
      default: true,
      type: Boolean,
    },
    rowClickable: {
      default: true,
      type: Boolean,
    },
    leftBordered: {
      default: true,
      type: Boolean,
    },
    rightBordered: {
      default: true,
      type: Boolean,
    },
    topBordered: {
      default: true,
      type: Boolean,
    },
    bottomBordered: {
      default: true,
      type: Boolean,
    },
  },
  setup(props, ctx) {
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
    return { borderVisibility };
  },
};
</script>
