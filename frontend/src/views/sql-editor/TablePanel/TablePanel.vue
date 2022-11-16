<template>
  <Splitpanes class="default-theme">
    <Pane size="100">
      <TableView :query-result="queryResult" />
    </Pane>
  </Splitpanes>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { Splitpanes, Pane } from "splitpanes";

import TableView from "../EditorCommon/TableView.vue";
import { useTabStore } from "@/store";

const tabStore = useTabStore();

const queryResult = computed(() => tabStore.currentTab.queryResult);
</script>
