<template>
  <template v-if="atom.type === 'project'">
    <!-- nothing -->
  </template>
  <template v-else-if="atom.type === 'instance'">
    <span class="flex items-center gap-x-1">
      <InstanceEngineIcon :instance="instance" />
      <ProtectedEnvironmentIcon
        :environment="instance.environment"
        class="w-4 h-4 text-inherit"
      />
      <span class="text-sm" :class="[!atom.disabled && 'text-gray-500']">
        {{ instance.environment.name }}
      </span>
    </span>
  </template>
  <template v-else-if="atom.type === 'database'">
    <span class="flex items-center gap-x-1">
      <template
        v-if="connectionTreeStore.tree.mode === ConnectionTreeMode.PROJECT"
      >
        <ProtectedEnvironmentIcon
          :environment="database.instance.environment"
          class="w-4 h-4 text-inherit"
        />
        <span class="text-sm" :class="[!atom.disabled && 'text-gray-500']">
          {{ database.instance.environment.name }}
        </span>
      </template>

      <heroicons-outline:database class="w-4 h-4" />
    </span>
  </template>
  <template v-else-if="atom.type === 'table'">
    <heroicons-outline:table class="w-4 h-4" />
  </template>
</template>

<script lang="ts" setup>
import { computed } from "vue";

import InstanceEngineIcon from "@/components/InstanceEngineIcon.vue";
import ProtectedEnvironmentIcon from "@/components/Environment/ProtectedEnvironmentIcon.vue";
import { ConnectionAtom, unknown, ConnectionTreeMode } from "@/types";
import {
  useConnectionTreeStore,
  useDatabaseStore,
  useInstanceStore,
} from "@/store";

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
