<template>
  <Tree @update-is-editing="$emit('update-is-editing', $event)" />
</template>

<script lang="ts" setup>
import Tree from "./Tree.vue";

defineEmits(["update-is-editing"]);
</script>
