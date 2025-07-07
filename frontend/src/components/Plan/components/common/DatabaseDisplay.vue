<template>
  <div class="flex items-center text-sm truncate">
    <InstanceV1EngineIcon
      v-if="instanceResource"
      class="inline-block mr-1"
      :instance="instanceResource"
    />
    <span class="truncate text-gray-600">
      {{ instanceDisplayName }}
    </span>
    <ChevronRightIcon
      class="inline opacity-60 text-gray-600 w-4 h-4 shrink-0"
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
import { useInstanceV1Store } from "@/store";
import { unknownInstance } from "@/types";
import {
  extractDatabaseResourceName,
  extractInstanceResourceName,
} from "@/utils";

const props = defineProps<{
  database: string;
}>();

const instanceStore = useInstanceV1Store();

const instanceResource = computed(() => {
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
</script>
