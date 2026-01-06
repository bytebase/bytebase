<template>
  <div class="w-full flex flex-col gap-y-4">
    <!-- Database Information Section -->
    <div class="flex flex-col gap-y-2">
      <h3 class="text-base font-medium">
        {{ $t("common.overview") }}
      </h3>
      <div class="flex items-center gap-x-5 gap-y-2 flex-wrap">
        <!-- Environment -->
        <div class="flex items-center gap-2">
          <span class="text-sm font-medium text-gray-600"
            >{{ $t("common.environment") }}:</span
          >
          <EnvironmentV1Name :environment="environment" />
        </div>

        <!-- Instance -->
        <div class="flex items-center gap-2">
          <span class="text-sm font-medium text-gray-600"
            >{{ $t("common.instance") }}:</span
          >
          <InstanceV1Name
            v-if="isValidInstanceName(instance.name)"
            :instance="instance"
          />
          <span v-else class="text-gray-900">{{ displayInstanceName }}</span>
        </div>

        <!-- Database Name -->
        <div class="flex items-center gap-2">
          <span class="text-sm font-medium text-gray-600"
            >{{ $t("common.database") }}:</span
          >
          <template v-if="isTaskDone && createdDatabase">
            <DatabaseV1Name :database="createdDatabase" :link="true" />
            <span class="text-sm text-gray-500"
              >({{ $t("common.created") }})</span
            >
          </template>
          <span v-else class="text-gray-900">
            {{ databaseName }}
          </span>
        </div>

        <!-- Task Status (only when rollout exists) -->
        <div v-if="createDatabaseTask" class="flex items-center gap-2">
          <span class="text-sm font-medium text-gray-600"
            >{{ $t("common.status") }}:</span
          >
          <TaskStatus :status="createDatabaseTask.status" size="small" />
        </div>
      </div>

      <!-- Task Run Table (only when rollout exists and has task runs) -->
      <div
        v-if="rollout && createDatabaseTask && taskRunsForCreateDatabase.length > 0"
        class="mt-4"
      >
        <TaskRunTable
          :task="createDatabaseTask"
          :task-runs="taskRunsForCreateDatabase"
        />
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, watchEffect } from "vue";
import { extractCoreDatabaseInfoFromDatabaseCreateTask } from "@/components/IssueV1/logic/utils";
import TaskStatus from "@/components/Rollout/kits/TaskStatus.vue";
import TaskRunTable from "@/components/RolloutV1/components/TaskRunTable.vue";
import {
  DatabaseV1Name,
  EnvironmentV1Name,
  InstanceV1Name,
} from "@/components/v2";
import {
  useCurrentProjectV1,
  useEnvironmentV1Store,
  useInstanceV1Store,
  useSheetV1Store,
} from "@/store";
import type { Plan_CreateDatabaseConfig } from "@/types/proto-es/v1/plan_service_pb";
import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";
import { isValidInstanceName } from "@/types/v1/instance";
import { extractInstanceResourceName } from "@/utils/v1/instance";
import { usePlanContext } from "../..";

const { plan, rollout, taskRuns } = usePlanContext();
const { project } = useCurrentProjectV1();
const environmentStore = useEnvironmentV1Store();
const instanceStore = useInstanceV1Store();
const sheetStore = useSheetV1Store();

// Get the first (and typically only) create database spec
const createDatabaseSpec = computed(() => {
  return plan.value.specs.find(
    (spec) => spec.config?.case === "createDatabaseConfig"
  );
});

const createDatabaseConfig = computed(() => {
  return createDatabaseSpec.value?.config?.value as Plan_CreateDatabaseConfig;
});

// Extract database information
const environment = computed(() => {
  return environmentStore.getEnvironmentByName(
    createDatabaseConfig.value.environment
  );
});

const instance = computed(() => {
  return instanceStore.getInstanceByName(createDatabaseConfig.value.target);
});

// The instance resource name from the target, in case the instance is not found or no permission to view.
const displayInstanceName = computed(() => {
  return extractInstanceResourceName(createDatabaseConfig.value.target);
});

const databaseName = computed(() => {
  return createDatabaseConfig.value.database || "";
});

// Find the task related to this create database spec (only exists after rollout is created)
const createDatabaseTask = computed(() => {
  if (!rollout.value || !createDatabaseSpec.value) return null;

  // Find the task that matches this spec
  for (const stage of rollout.value.stages) {
    for (const task of stage.tasks) {
      if (task.specId === createDatabaseSpec.value.id) {
        return task;
      }
    }
  }
  return null;
});

// Get the sheet name from the task payload (when rollout exists)
const sheetName = computed(() => {
  if (!createDatabaseTask.value) return null;

  if (createDatabaseTask.value.payload.case === "databaseCreate") {
    return createDatabaseTask.value.payload.value.sheet;
  }
  return null;
});

// Fetch the sheet when sheetName changes
watchEffect(() => {
  if (sheetName.value) {
    sheetStore.getOrFetchSheetByName(sheetName.value);
  }
});

// Fetch the instance when target changes
watchEffect(() => {
  const target = createDatabaseConfig.value?.target;
  if (target) {
    instanceStore.getOrFetchInstanceByName(target);
  }
});

// Get task runs for this specific create database task
const taskRunsForCreateDatabase = computed(() => {
  if (!rollout.value || !createDatabaseTask.value) return [];

  const taskName = createDatabaseTask.value.name;
  return taskRuns.value.filter((taskRun) =>
    taskRun.name.startsWith(taskName + "/")
  );
});

// Check if the task is completed
const isTaskDone = computed(() => {
  return createDatabaseTask.value?.status === Task_Status.DONE;
});

// Get the created database info for completed tasks
const createdDatabase = computed(() => {
  if (!isTaskDone.value || !createDatabaseTask.value) return null;

  return extractCoreDatabaseInfoFromDatabaseCreateTask(
    project.value,
    createDatabaseTask.value,
    plan.value
  );
});
</script>
