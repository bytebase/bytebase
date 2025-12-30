<template>
  <div class="flex flex-col gap-y-2">
    <template v-if="tasks.length > 0">
      <h3 class="text-base font-medium">
        {{ $t("common.task", tasks.length) }}
      </h3>
      <div class="flex flex-wrap gap-2">
        <div
          v-for="task in tasks"
          :key="task.name"
          class="inline-flex items-center gap-2 px-2 py-1.5 border rounded-sm min-w-0"
        >
          <TaskStatus :status="task.status" size="tiny" class="shrink-0" />
          <DatabaseDisplay
            :database="task.target"
            :show-environment="true"
            size="medium"
            class="flex-1 min-w-0"
          />
        </div>
      </div>
    </template>
    <div v-else class="text-center text-control-light py-8">
      {{ $t("common.no-data") }}
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, watchEffect } from "vue";
import DatabaseDisplay from "@/components/Plan/components/common/DatabaseDisplay.vue";
import TaskStatus from "@/components/Rollout/kits/TaskStatus.vue";
import { useDatabaseV1Store } from "@/store";
import { usePlanContextWithRollout } from "../../../logic";

const { plan, rollout } = usePlanContextWithRollout();
const dbStore = useDatabaseV1Store();

const tasks = computed(() => {
  const exportDataSpec = plan.value.specs.find(
    (spec) => spec.config?.case === "exportDataConfig"
  );
  if (!exportDataSpec) return [];

  return rollout.value.stages
    .flatMap((stage) => stage.tasks)
    .filter((task) => task.specId === exportDataSpec.id);
});

// Fetch task target databases
watchEffect(() => {
  const targets = tasks.value.map((task) => task.target);
  if (targets.length > 0) {
    dbStore.batchGetOrFetchDatabases(targets);
  }
});
</script>
