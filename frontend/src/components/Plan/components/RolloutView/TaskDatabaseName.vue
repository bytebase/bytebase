<template>
  <div class="flex items-center text-sm truncate">
    <InstanceV1EngineIcon
      class="inline-block mr-1"
      :instance="database.instanceResource"
    />
    <span class="truncate text-gray-600">
      {{ instanceDisplayName }}
    </span>
    <ChevronRightIcon
      class="inline opacity-60 text-gray-600 w-4 h-4 shrink-0 mx-0.5"
    />
    <span class="truncate text-gray-800">
      {{ databaseDisplayName }}
    </span>
  </div>
</template>

<script setup lang="ts">
import { ChevronRightIcon } from "lucide-vue-next";
import { computed } from "vue";
import { databaseForTask } from "@/components/Rollout/RolloutDetail/utils";
import { InstanceV1EngineIcon } from "@/components/v2";
import { useCurrentProjectV1 } from "@/store";
import { unknownInstance } from "@/types";
import type { Task } from "@/types/proto/v1/rollout_service";
import { extractInstanceResourceName } from "@/utils";

const props = defineProps<{
  task: Task;
  environmentTitle?: string;
}>();

const { project } = useCurrentProjectV1();

const database = computed(() => databaseForTask(project.value, props.task));

const instanceDisplayName = computed(() => {
  const title = database.value.instanceResource.title;
  // Fallback for unknown instances - try to extract instance name from task target
  if (!title || title === unknownInstance().title) {
    return extractInstanceResourceName(props.task.target);
  }
  return title;
});

const databaseDisplayName = computed(() => {
  const name = database.value.databaseName;
  // Fallback for unknown databases - try to extract database name from task target
  if (name === "<<Unknown database>>" || !name) {
    const target = props.task.target;
    if (target && target.includes("/databases/")) {
      const databaseName = target.split("/databases/")[1];
      return databaseName || name;
    }
  }
  return name;
});
</script>
