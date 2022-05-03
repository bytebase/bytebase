<template>
  <Splitpanes class="default-theme">
    <Pane size="100">
      <TableView />
    </Pane>
  </Splitpanes>
</template>

<script lang="ts" setup>
import TableView from "./TableView.vue";
import { Splitpanes, Pane } from "splitpanes";
</script>
