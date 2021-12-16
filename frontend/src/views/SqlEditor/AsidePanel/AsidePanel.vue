<template>
  <Splitpanes horizontal class="default-theme">
    <Pane size="100"><DatabasesTree /></Pane>
    <Pane size="0" max-size="40"><TableSchema /></Pane>
  </Splitpanes>
</template>

<script lang="ts" setup>
import DatabasesTree from "./DatabasesTree.vue";
import TableSchema from "./TableSchema.vue";
</script>
