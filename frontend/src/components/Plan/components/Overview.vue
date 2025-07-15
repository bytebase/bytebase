<template>
  <div class="w-full pt-2 pb-4 px-4 overflow-y-auto">
    <div class="max-w-4xl mx-auto space-y-4 mt-4">
      <!-- Plan Header -->
      <div>
        <h2 class="text-2xl font-semibold mb-2">
          {{ $t("common.overview") }}
        </h2>
      </div>

      <!-- Statistics Cards -->
      <div
        :class="
          rollout && issue
            ? 'grid grid-cols-1 lg:grid-cols-3 gap-4'
            : rollout || issue
              ? 'grid grid-cols-1 lg:grid-cols-2 gap-4'
              : 'grid grid-cols-1 md:grid-cols-3 gap-4'
        "
      >
        <!-- Plan Statistics Column -->
        <div :class="issue || rollout ? 'space-y-4' : 'contents'">
          <!-- Plan Statistics Grid -->
          <div
            :class="issue || rollout ? 'grid grid-cols-1 gap-4' : 'contents'"
          >
            <!-- Specs Card -->
            <div class="bg-white rounded-md border px-3 py-2">
              <div class="flex items-center justify-between">
                <div>
                  <p class="textlabel">
                    {{ $t("plan.overview.total-changes") }}
                  </p>
                  <p class="text-xl font-medium mt-1">
                    {{ statistics.totalSpecs }}
                  </p>
                </div>
                <LayersIcon
                  class="w-8 h-8 text-control-light"
                  stroke-width="1"
                />
              </div>
            </div>

            <!-- Targets Card -->
            <div class="bg-white rounded-md border px-3 py-2">
              <div class="flex items-center justify-between">
                <div>
                  <p class="textlabel">
                    {{ $t("plan.overview.total-targets") }}
                  </p>
                  <p class="text-xl font-medium mt-1">
                    {{ statistics.totalTargets }}
                  </p>
                </div>
                <DatabaseIcon
                  class="w-8 h-8 text-control-light"
                  stroke-width="1"
                />
              </div>
            </div>

            <!-- Checks Card -->
            <div class="bg-white rounded-md border px-3 py-2">
              <div class="flex items-center justify-between">
                <div>
                  <p class="textlabel">
                    {{ $t("plan.navigator.checks") }}
                  </p>
                  <div class="flex items-center gap-3 mt-1">
                    <div
                      v-if="statistics.checkStatus.error > 0"
                      class="flex items-center gap-1"
                    >
                      <XCircleIcon class="w-5 h-5 text-error" />
                      <span class="text-xl font-medium text-error">{{
                        statistics.checkStatus.error
                      }}</span>
                    </div>
                    <div
                      v-if="statistics.checkStatus.warning > 0"
                      class="flex items-center gap-1"
                    >
                      <AlertCircleIcon class="w-5 h-5 text-warning" />
                      <span class="text-xl font-medium text-warning">{{
                        statistics.checkStatus.warning
                      }}</span>
                    </div>
                    <div
                      v-if="statistics.checkStatus.success > 0"
                      class="flex items-center gap-1"
                    >
                      <CheckCircleIcon class="w-5 h-5 text-success" />
                      <span class="text-xl font-medium text-success">{{
                        statistics.checkStatus.success
                      }}</span>
                    </div>
                    <span
                      v-if="statistics.checkStatus.total === 0"
                      class="text-xl text-control"
                    >
                      {{ $t("plan.overview.no-checks") }}
                    </span>
                  </div>
                </div>
                <ActivityIcon
                  class="w-8 h-8 text-control-light"
                  stroke-width="1"
                />
              </div>
            </div>
          </div>
        </div>

        <!-- Approval Flow Column -->
        <div v-if="issue">
          <div class="bg-white rounded-md border p-4">
            <ApprovalFlowSection :issue="issue" @issue-updated="() => {}" />
          </div>
        </div>

        <!-- Rollout Statistics Column -->
        <div v-if="rollout" class="space-y-4">
          <!-- Stages Card -->
          <div class="bg-white rounded-md border p-4">
            <div class="flex items-center justify-between mb-3">
              <h3 class="text-lg font-semibold">
                {{ $t("rollout.stage.self", rollout?.stages.length || 0) }}
              </h3>
              <GitBranchIcon
                class="w-6 h-6 text-control-light"
                stroke-width="1"
              />
            </div>
            <div
              v-if="!rollout?.stages.length"
              class="text-sm text-gray-400 italic"
            >
              {{ $t("common.no-data") }}
            </div>
            <div v-else class="flex flex-row gap-2 flex-wrap">
              <div
                v-for="(stage, index) in rollout.stages"
                :key="stage.name"
                class="flex items-center gap-2"
              >
                <TaskStatus :status="getStageStatus(stage)" size="small" />
                <span
                  class="text-sm font-medium text-gray-700 whitespace-nowrap"
                >
                  {{
                    environmentStore.getEnvironmentByName(stage.environment)
                      .title
                  }}
                </span>
                <span
                  v-if="index < (rollout?.stages.length || 0) - 1"
                  class="text-gray-400"
                  >→</span
                >
              </div>
            </div>
          </div>

          <!-- Task Status Card -->
          <div class="bg-white rounded-md border p-4">
            <div class="flex items-center justify-between mb-3">
              <h3 class="text-lg font-semibold">
                {{ $t("common.tasks") }}
              </h3>
              <ListChecksIcon
                class="w-6 h-6 text-control-light"
                stroke-width="1"
              />
            </div>
            <div class="flex flex-wrap gap-2">
              <NTag
                v-for="status in TASK_STATUS_FILTERS"
                :key="status"
                v-show="getTaskCount(status) > 0"
                round
                size="medium"
              >
                <template #avatar>
                  <TaskStatus :status="status" size="small" />
                </template>
                <div class="flex flex-row items-center gap-2">
                  <span class="select-none text-sm">
                    {{ stringifyTaskStatus(status) }}
                  </span>
                  <span class="select-none text-sm font-medium">
                    {{ getTaskCount(status) }}
                  </span>
                </div>
              </NTag>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import {
  LayersIcon,
  DatabaseIcon,
  ActivityIcon,
  CheckCircleIcon,
  AlertCircleIcon,
  XCircleIcon,
  GitBranchIcon,
  ListChecksIcon,
} from "lucide-vue-next";
import { NTag } from "naive-ui";
import { computed, watch } from "vue";
import TaskStatus from "@/components/Rollout/kits/TaskStatus.vue";
import {
  useInstanceV1Store,
  useDBGroupStore,
  useEnvironmentV1Store,
} from "@/store";
import type { ComposedDatabaseGroup, ComposedInstance } from "@/types";
import { PlanCheckRun_Result_Status } from "@/types/proto-es/v1/plan_service_pb";
import { Task_Status as TaskStatusEnum } from "@/types/proto-es/v1/rollout_service_pb";
import type { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";
import {
  extractDatabaseResourceName,
  getStageStatus,
  stringifyTaskStatus,
} from "@/utils";
import { usePlanContext } from "../logic/context";
import { targetsForSpec } from "../logic/plan";
import ApprovalFlowSection from "./IssueReviewView/Sidebar/ApprovalFlowSection/ApprovalFlowSection.vue";

const { plan, issue, rollout } = usePlanContext();
const instanceStore = useInstanceV1Store();
const dbGroupStore = useDBGroupStore();
const environmentStore = useEnvironmentV1Store();

// Task status filters for rollout
const TASK_STATUS_FILTERS: Task_Status[] = [
  TaskStatusEnum.DONE,
  TaskStatusEnum.RUNNING,
  TaskStatusEnum.FAILED,
  TaskStatusEnum.CANCELED,
  TaskStatusEnum.SKIPPED,
  TaskStatusEnum.PENDING,
  TaskStatusEnum.NOT_STARTED,
];

// Get task count by status for rollout
const getTaskCount = (status: Task_Status) => {
  if (!rollout?.value) return 0;
  const allTasks = rollout.value.stages.flatMap((stage) => stage.tasks);
  return allTasks.filter((task) => task.status === status).length;
};

// Calculate statistics
const statistics = computed(() => {
  let totalTargets = 0;
  const checkStatus = {
    total: 0,
    success:
      plan.value.planCheckRunStatusCount[
        PlanCheckRun_Result_Status[PlanCheckRun_Result_Status.SUCCESS]
      ] || 0,
    warning:
      plan.value.planCheckRunStatusCount[
        PlanCheckRun_Result_Status[PlanCheckRun_Result_Status.WARNING]
      ] || 0,
    error:
      plan.value.planCheckRunStatusCount[
        PlanCheckRun_Result_Status[PlanCheckRun_Result_Status.ERROR]
      ] || 0,
  };
  checkStatus.total =
    checkStatus.success + checkStatus.warning + checkStatus.error;
  for (const spec of plan.value.specs) {
    totalTargets += targetsForSpec(spec).length;
  }
  return {
    totalSpecs: plan.value.specs.length,
    totalTargets,
    checkStatus,
  };
});

// Get affected resources
const affectedResources = computed(() => {
  const specs = plan.value?.specs || [];
  const resourceList: Array<
    | {
        type: "instance";
        name: string;
        instance: ComposedInstance;
        databases: string[];
      }
    | {
        type: "databaseGroup";
        name: string;
        databaseGroup?: ComposedDatabaseGroup;
        databases: string[];
      }
  > = [];

  const instanceMap = new Map<string, Set<string>>();
  const dbGroupSet = new Set<string>();

  for (const spec of specs) {
    const targets = targetsForSpec(spec);
    for (const target of targets) {
      // Check if it's a database group
      if (target.includes("/databaseGroups/")) {
        dbGroupSet.add(target);
      } else {
        // Parse instance/database format
        const { instance, databaseName } = extractDatabaseResourceName(target);
        if (instance && databaseName) {
          if (!instanceMap.has(instance)) {
            instanceMap.set(instance, new Set());
          }
          instanceMap.get(instance)!.add(databaseName);
        }
      }
    }
  }

  // Add instances to resource list
  for (const [instanceName, databases] of instanceMap.entries()) {
    const instance = instanceStore.getInstanceByName(instanceName);
    resourceList.push({
      type: "instance",
      name: instanceName,
      instance: instance,
      databases: Array.from(databases),
    });
  }

  // Add database groups to resource list
  for (const dbGroupName of dbGroupSet) {
    const dbGroup = dbGroupStore.getDBGroupByName(dbGroupName);
    resourceList.push({
      type: "databaseGroup",
      name: dbGroupName,
      databaseGroup: dbGroup,
      databases: dbGroup?.matchedDatabases.map((db) => db.name) || [],
    });
  }

  return resourceList;
});

watch(
  () => affectedResources.value,
  () => {
    for (const resource of affectedResources.value) {
      if (resource.type === "instance") {
        instanceStore.getOrFetchInstanceByName(resource.name);
      } else if (resource.type === "databaseGroup") {
        dbGroupStore.getOrFetchDBGroupByName(resource.name);
      }
    }
  },
  { immediate: true }
);
</script>
