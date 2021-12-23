<template>
  <h1>Connection Tree</h1>
  <NTree 
    block-line
    :data="connectionTree"
  />
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useStore } from "vuex";

const store = useStore();

const connectionTree = computed(() => store.state.sqlEditor.connectionTree);
</script>
