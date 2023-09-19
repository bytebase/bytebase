<template>
  <heroicons-outline:link v-if="connected" class="w-4 h-4" />
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useTabStore } from "@/store";
import { ConnectionAtom, UNKNOWN_ID } from "@/types";

const props = defineProps<{
  atom: ConnectionAtom;
}>();

const tabStore = useTabStore();

const connected = computed(() => {
  const { instanceId, databaseId } = tabStore.currentTab.connection;
  const { atom } = props;

  if (atom.type === "database") {
    if (atom.id === databaseId) {
      return true;
    }
  }
  if (atom.type === "instance") {
    if (databaseId === String(UNKNOWN_ID) && atom.id === instanceId) {
      return true;
    }
  }
  return false;
});
</script>
