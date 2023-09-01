<template>
  <DatabaseSchema v-if="showSchemaPanel" />
  <div v-else>else</div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useInstanceV1Store, useTabStore } from "@/store";
import { UNKNOWN_ID } from "@/types";
import { Engine } from "@/types/proto/v1/common";
import DatabaseSchema from "./SchemaPanel";

const tabStore = useTabStore();

const showSchemaPanel = computed(() => {
  const conn = tabStore.currentTab.connection;
  if (conn.databaseId === String(UNKNOWN_ID)) {
    return false;
  }

  const instance = useInstanceV1Store().getInstanceByUID(conn.instanceId);
  if (instance.engine === Engine.REDIS) {
    return false;
  }
  return true;
});
</script>
