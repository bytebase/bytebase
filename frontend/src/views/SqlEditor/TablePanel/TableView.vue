<template>
  <NDataTable :columns="columns" :data="data" :bordered="false">
    <template #empty>
      <div class="p-20 h-full">Click Run to execute the query.</div>
    </template>
  </NDataTable>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useStore } from "vuex";
import { useVuex } from "@vueblocks/vue-use-vuex";

const store = useStore();

const { useState } = useVuex("sqlEditor", store);

const { queryResult } = useState(["queryResult"]) as any;

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
</script>
