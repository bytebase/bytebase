<template>
  <div class="w-full py-4 overflow-y-auto">
    <div class="grid grid-cols-[auto_1fr] gap-y-4 gap-x-8">
      <!-- SQL Checks -->
      <h3 class="text-base text-control">
        {{ $t("plan.navigator.checks") }}
      </h3>
      <div class="flex items-center gap-4">
        <div
          v-if="getChecksCount(PlanCheckRun_Result_Status.ERROR) > 0"
          class="flex items-center gap-1 cursor-pointer text-error"
          @click="selectedResultStatus = PlanCheckRun_Result_Status.ERROR"
        >
          <XCircleIcon class="w-5 h-5" />
          <span>
            {{ $t("common.error") }}
          </span>
          <span class="font-semibold">
            {{ getChecksCount(PlanCheckRun_Result_Status.ERROR) }}
          </span>
        </div>
        <div
          v-if="getChecksCount(PlanCheckRun_Result_Status.WARNING) > 0"
          class="flex items-center gap-1 cursor-pointer text-warning"
          @click="selectedResultStatus = PlanCheckRun_Result_Status.WARNING"
        >
          <AlertCircleIcon class="w-5 h-5" />
          <span>
            {{ $t("common.warning") }}
          </span>
          <span class="font-semibold">
            {{ getChecksCount(PlanCheckRun_Result_Status.WARNING) }}
          </span>
        </div>
        <div
          v-if="getChecksCount(PlanCheckRun_Result_Status.SUCCESS) > 0"
          class="flex items-center gap-1 cursor-pointer text-success"
          @click="selectedResultStatus = PlanCheckRun_Result_Status.SUCCESS"
        >
          <CheckCircleIcon class="w-5 h-5" />
          <span>
            {{ $t("common.success") }}
          </span>
          <span class="font-semibold">
            {{ getChecksCount(PlanCheckRun_Result_Status.SUCCESS) }}
          </span>
        </div>
        <span
          v-if="
            getChecksCount(PlanCheckRun_Result_Status.ERROR) +
              getChecksCount(PlanCheckRun_Result_Status.WARNING) +
              getChecksCount(PlanCheckRun_Result_Status.SUCCESS) ===
            0
          "
          class="text-sm text-control"
        >
          {{ $t("plan.overview.no-checks") }}
        </span>
      </div>

      <!-- Stages -->
      <template v-if="rollout">
        <h3 class="text-base text-control">
          {{ $t("rollout.stage.self", 2) }}
        </h3>
        <div class="flex items-center gap-1 flex-wrap">
          <template v-if="rollout.stages.length > 0">
            <div
              v-for="(stage, index) in rollout.stages"
              :key="stage.name"
              class="flex items-center gap-1"
            >
              <div
                class="flex items-center gap-1 cursor-pointer hover:opacity-80 transition-opacity"
                @click="navigateToStage(stage.name)"
              >
                <TaskStatus
                  :status="getStageStatus(stage)"
                  size="small"
                  disabled
                />
                <span class="font-medium text-gray-700 whitespace-nowrap">
                  {{ getEnvironmentTitle(stage.environment) }}
                </span>
              </div>
              <span
                v-if="index < rollout.stages.length - 1"
                class="text-gray-400 text-sm mx-2"
              >
                →
              </span>
            </div>
          </template>
          <span v-else class="text-sm text-control">
            {{ $t("common.no-data") }}
          </span>
        </div>
      </template>
    </div>
  </div>

  <ChecksDrawer
    v-if="selectedResultStatus"
    :status="selectedResultStatus"
    @close="selectedResultStatus = undefined"
  />
</template>

<script setup lang="ts">
import { CheckCircleIcon, AlertCircleIcon, XCircleIcon } from "lucide-vue-next";
import { ref } from "vue";
import { useRouter } from "vue-router";
import TaskStatus from "@/components/Rollout/kits/TaskStatus.vue";
import { PROJECT_V1_ROUTE_ROLLOUT_DETAIL_STAGE_DETAIL } from "@/router/dashboard/projectV1";
import { useCurrentProjectV1, useEnvironmentV1Store } from "@/store";
import { PlanCheckRun_Result_Status } from "@/types/proto-es/v1/plan_service_pb";
import { extractProjectResourceName, getStageStatus } from "@/utils";
import { usePlanContext } from "../../logic";
import ChecksDrawer from "../ChecksView/ChecksDrawer.vue";

const { plan, rollout } = usePlanContext();
const environmentStore = useEnvironmentV1Store();
const router = useRouter();
const { project } = useCurrentProjectV1();

const selectedResultStatus = ref<PlanCheckRun_Result_Status | undefined>(
  undefined
);

const getChecksCount = (status: PlanCheckRun_Result_Status) => {
  return (
    plan.value.planCheckRunStatusCount[PlanCheckRun_Result_Status[status]] || 0
  );
};

const getEnvironmentTitle = (environmentName: string) => {
  const environment = environmentStore.getEnvironmentByName(environmentName);
  return environment.title || environmentName;
};

const navigateToStage = (stageName: string) => {
  if (!rollout?.value) return;

  const rolloutId = rollout.value.name.split("/").pop();
  const stageId = stageName.split("/").pop();

  if (!rolloutId || !stageId) return;

  router.push({
    name: PROJECT_V1_ROUTE_ROLLOUT_DETAIL_STAGE_DETAIL,
    params: {
      projectId: extractProjectResourceName(project.value.name),
      rolloutId,
      stageId,
    },
  });
};
</script>
