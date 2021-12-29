<template>
  <Splitpanes class="default-theme">
    <Pane size="100">
      <TableView />
    </Pane>
    <Pane size="0">
      <HistroyView />
    </Pane>
  </Splitpanes>
</template>

<script lang="ts" setup>
import TableView from "./TableView.vue";
import HistroyView from "./HistoryView.vue";
</script>
