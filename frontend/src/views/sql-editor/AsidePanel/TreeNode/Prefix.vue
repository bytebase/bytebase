<template>
  <template v-if="atom.type === 'project'">
    <!-- nothing -->
  </template>
  <template v-else-if="atom.type === 'instance'">
    <span class="flex items-center gap-x-1">
      <InstancePrefix
        :instance="instance"
        :environment="environment"
        :disabled="atom.disabled"
      />
    </span>
  </template>
  <template v-else-if="atom.type === 'database'">
    <span class="flex items-center gap-x-1">
      <InstancePrefix
        :instance="database.instanceEntity"
        :environment="environment"
        :disabled="atom.disabled"
      />

      <heroicons-outline:database class="w-4 h-4" />
    </span>
  </template>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import {
  useDatabaseV1Store,
  useInstanceV1Store,
  useEnvironmentV1Store,
} from "@/store";
import {
  ConnectionAtom,
  unknownInstance,
  unknownDatabase,
  unknownEnvironment,
} from "@/types";
import InstancePrefix from "./InstancePrefix.vue";

const props = defineProps<{
  atom: ConnectionAtom;
}>();

const instanceStore = useInstanceV1Store();
const databaseStore = useDatabaseV1Store();
const environmentStore = useEnvironmentV1Store();

const instance = computed(() => {
  const { atom } = props;
  if (atom.type === "instance") {
    return instanceStore.getInstanceByUID(atom.id);
  }

  return unknownInstance();
});

const database = computed(() => {
  const { atom } = props;
  if (atom.type === "database") {
    return databaseStore.getDatabaseByUID(atom.id);
  }
  return unknownDatabase();
});

const environment = computed(() => {
  const { atom } = props;
  if (atom.type === "instance") {
    return instance.value.environmentEntity;
  }
  return (
    environmentStore.getEnvironmentByName(
      database.value.effectiveEnvironment
    ) ?? unknownEnvironment()
  );
});
</script>
