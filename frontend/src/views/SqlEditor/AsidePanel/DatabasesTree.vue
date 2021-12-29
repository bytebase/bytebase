<template>
  <div class="databases-tree p-2">
    <NTree 
      block-line
      :data="connectionTree"
    />
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useStore } from "vuex";

const store = useStore();

const connectionTree = computed(() => store.state.sqlEditor.connectionTree);
</script>
