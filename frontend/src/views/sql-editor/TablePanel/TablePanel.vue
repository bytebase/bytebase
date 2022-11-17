<template>
  <TableView class="w-full h-full" :query-result="queryResult" />
</template>

<script lang="ts" setup>
import { computed } from "vue";

import TableView from "../EditorCommon/TableView.vue";
import { useTabStore } from "@/store";

const tabStore = useTabStore();

const queryResult = computed(() => tabStore.currentTab.queryResult);
</script>
