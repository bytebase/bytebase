<template>
  <div class="border-t border-b border-block-border">
    <table class="min-w-full divide-y divide-block-border">
      <thead v-if="showHeader" class="bg-gray-50">
        <tr>
          <th
            v-for="(column, index) in columnList"
            :key="index"
            scope="col"
            class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
          >
            {{ column.title }}
          </th>
        </tr>
      </thead>
      <tbody class="bg-normal divide-y divide-block-border">
        <tr
          v-for="(item, index) in dataSource"
          :key="index"
          class="cursor-pointer hover:bg-gray-200"
          @click.stop="$emit('click-row', index)"
        >
          <slot name="body" :rowData="item" />
        </tr>
      </tbody>
    </table>
  </div>
</template>

<script lang="ts">
import { computed, reactive, PropType } from "vue";
import { BBTableColumn } from "../types";

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
      default: function () {
        return new Array();
      },
      type: Object as PropType<Object[]>,
    },
    showHeader: {
      default: true,
      type: Boolean,
    },
  },
  setup(props, ctx) {},
};
</script>
