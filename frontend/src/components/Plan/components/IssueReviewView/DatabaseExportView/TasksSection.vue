<template>
  <div class="space-y-2">
    <!-- Tasks Display -->
    <div class="space-y-3">
      <!-- Task status summary with title -->
      <div
        v-if="exportTasks.length > 0"
        class="flex items-center justify-between"
      >
        <div class="flex items-center gap-2">
          <h3 class="text-base font-medium">
            {{ $t("common.task", exportTasks.length) }}
          </h3>
        </div>
      </div>

      <!-- Task status display -->
      <div v-if="exportTasks.length > 0" class="flex flex-wrap gap-2">
        <div
          v-for="task in exportTasks"
          :key="task.name"
          class="inline-flex items-center gap-2 px-2 py-1.5 border rounded transition-colors min-w-0"
        >
          <!-- Task Status -->
          <div class="flex-shrink-0">
            <TaskStatus :status="task.status" size="tiny" />
          </div>

          <!-- Database Display -->
          <div class="flex-1 min-w-0">
            <DatabaseDisplay
              :database="task.target"
              :show-environment="true"
              size="medium"
            />
          </div>
        </div>
      </div>

      <!-- No tasks message -->
      <div
        v-else-if="exportTasks.length === 0"
        class="text-center text-control-light py-8"
      >
        {{ $t("common.no-data") }}
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import DatabaseDisplay from "@/components/Plan/components/common/DatabaseDisplay.vue";
import TaskStatus from "@/components/Rollout/kits/TaskStatus.vue";
import { usePlanContext } from "../../../logic";

const { plan, rollout } = usePlanContext();

// Get the export data spec
const exportDataSpec = computed(() => {
  return plan.value.specs.find(
    (spec) => spec.config?.case === "exportDataConfig"
  );
});

// Find export tasks related to this spec
const exportTasks = computed(() => {
  if (!rollout.value || !exportDataSpec.value) return [];

  const tasks = [];
  for (const stage of rollout.value.stages) {
    for (const task of stage.tasks) {
      if (task.specId === exportDataSpec.value.id) {
        tasks.push(task);
      }
    }
  }
  return tasks;
});
</script>
