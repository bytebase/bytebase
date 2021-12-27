<template>
  <div class="table-view h-full" ref="tableViewRef">
    <NDataTable
      size="mini"
      :columns="columns"
      :data="data"
      :bordered="false"
      flex-height
      :style="{ height: `${tableMaxHeight}px` }"
    >
      <template #empty>
        <div class="p-20 h-full flex justify-center items-center">
          Click Run to execute the query.
        </div>
      </template>
    </NDataTable>
  </div>
</template>

<script lang="ts" setup>
import { computed, ref } from "vue";
import { useStore } from "vuex";
import { useVuex } from "@vueblocks/vue-use-vuex";
import { useResizeObserver } from "@vueuse/core";

const store = useStore();

const { useState } = useVuex("sqlEditor", store);

const { queryResult } = useState(["queryResult"]) as any;

const tableViewRef = ref<HTMLDivElement>();
const tableMaxHeight = ref(0);

const columns = computed(() => {
  return queryResult.value.length > 0
    ? Object.keys(queryResult.value[0]).map((item) => {
        return {
          title: item.toUpperCase(),
          key: item,
        };
      })
    : [];
});
const data = computed(() => queryResult.value);

// make sure the table view is always full of the pane
useResizeObserver(tableViewRef, (entries) => {
  const entry = entries[0];
  const { height } = entry.contentRect;
  tableMaxHeight.value = height;
});
</script>
