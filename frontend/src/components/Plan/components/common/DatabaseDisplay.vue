<template>
  <div class="flex items-center text-sm truncate">
    <InstanceV1EngineIcon
      v-if="instanceResource"
      class="inline-block mr-1"
      :instance="instanceResource"
    />
    <span v-if="showEnvironment && environment" class="text-gray-500 mr-1">
      ({{ environment.title }})
    </span>
    <span class="truncate text-gray-600">
      {{ instanceDisplayName }}
    </span>
    <ChevronRightIcon
      class="inline opacity-60 text-gray-500 w-4 h-4 shrink-0"
    />
    <span class="truncate text-gray-800">
      {{ databaseDisplayName }}
    </span>
  </div>
</template>

<script setup lang="ts">
import { ChevronRightIcon } from "lucide-vue-next";
import { computed } from "vue";
import { InstanceV1EngineIcon } from "@/components/v2";
import {
  useDatabaseV1Store,
  useEnvironmentV1Store,
  useInstanceV1Store,
} from "@/store";
import { isValidDatabaseName, unknownInstance } from "@/types";
import {
  extractDatabaseResourceName,
  extractInstanceResourceName,
} from "@/utils";

const props = defineProps<{
  database: string;
  showEnvironment?: boolean;
}>();

const environmentStore = useEnvironmentV1Store();
const instanceStore = useInstanceV1Store();

const databaseEntity = computed(() =>
  useDatabaseV1Store().getDatabaseByName(props.database)
);

const instanceResource = computed(() => {
  if (isValidDatabaseName(databaseEntity.value.name)) {
    return databaseEntity.value.instanceResource;
  }
  // Extract instance name from the database name (which is actually a target string)
  const instanceName = extractInstanceResourceName(props.database);
  if (instanceName) {
    const instance = instanceStore.getInstanceByName(
      `instances/${instanceName}`
    );
    if (instance.name !== unknownInstance().name) {
      return instance;
    }
  }

  return null; // Don't show instance icon if we can't resolve it
});

const instanceDisplayName = computed(() => {
  if (instanceResource.value) {
    return instanceResource.value.title;
  }
  return extractInstanceResourceName(props.database) || "Unknown Instance";
});

const databaseDisplayName = computed(() => {
  const { databaseName } = extractDatabaseResourceName(props.database);
  return databaseName || "Unknown Database";
});

const environment = computed(() => {
  const environmentName =
    databaseEntity.value.environment || instanceResource.value?.environment;
  if (!environmentName) {
    return undefined;
  }

  return environmentStore.getEnvironmentByName(environmentName);
});
</script>
