<template>
  <template v-if="atom.type === 'project'">
    <!-- nothing -->
  </template>
  <template v-else-if="atom.type === 'instance'">
    <span class="flex items-center gap-x-1">
      <InstancePrefix :instance="instance" :disabled="atom.disabled" />
    </span>
  </template>
  <template v-else-if="atom.type === 'database'">
    <span class="flex items-center gap-x-1">
      <InstancePrefix
        v-if="connectionTreeStore.tree.mode === ConnectionTreeMode.PROJECT"
        :instance="database.instance"
        :disabled="atom.disabled"
      />

      <heroicons-outline:database class="w-4 h-4" />
    </span>
  </template>
</template>

<script lang="ts" setup>
import { computed } from "vue";

import { ConnectionAtom, unknown, ConnectionTreeMode } from "@/types";
import {
  useConnectionTreeStore,
  useDatabaseStore,
  useInstanceStore,
} from "@/store";
import InstancePrefix from "./InstancePrefix.vue";

const props = defineProps<{
  atom: ConnectionAtom;
}>();

const instanceStore = useInstanceStore();
const databaseStore = useDatabaseStore();
const connectionTreeStore = useConnectionTreeStore();

const instance = computed(() => {
  const { atom } = props;
  if (atom.type === "instance") {
    return instanceStore.getInstanceById(atom.id);
  }

  return unknown("INSTANCE");
});

const database = computed(() => {
  const { atom } = props;
  if (atom.type === "database") {
    const database = databaseStore.getDatabaseById(atom.id);
    return database;
  }
  return unknown("DATABASE");
});
</script>
