<template>
  <Splitpanes class="default-theme">
    <Pane size="100">
      <TableView />
    </Pane>
    <Pane v-if="isDev()" size="0">
      <HistroyView />
    </Pane>
  </Splitpanes>
</template>

<script lang="ts" setup>
import TableView from "./TableView.vue";
import HistroyView from "./HistoryView.vue";

import { isDev } from "../../../utils";
</script>
