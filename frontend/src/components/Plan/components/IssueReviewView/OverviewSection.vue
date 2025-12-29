<template>
  <div class="w-full overflow-y-auto">
    <div class="grid grid-cols-[auto_1fr] gap-y-4 gap-x-8">
      <!-- SQL Checks -->
      <h3 class="text-base text-control">
        {{ $t("plan.navigator.checks") }}
      </h3>
      <div class="flex items-center gap-4">
        <PlanCheckStatusCount
          :plan="plan"
          clickable
          @click="selectedResultStatus = $event"
        />
        <span v-if="!hasAnyChecks" class="text-sm text-control">
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
                <EnvironmentV1Name
                  :environment="getEnvironmentEntity(stage.environment)"
                  :link="false"
                  :null-environment-placeholder="'Null'"
                />
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
import { ref } from "vue";
import { useRouter } from "vue-router";
import TaskStatus from "@/components/Rollout/kits/TaskStatus.vue";
import { EnvironmentV1Name } from "@/components/v2";
import { PROJECT_V1_ROUTE_PLAN_ROLLOUT_STAGE } from "@/router/dashboard/projectV1";
import { useCurrentProjectV1, useEnvironmentV1Store } from "@/store";
import { Advice_Level } from "@/types/proto-es/v1/sql_service_pb";
import {
  extractPlanUIDFromRolloutName,
  extractProjectResourceName,
  getStageStatus,
} from "@/utils";
import { usePlanCheckStatus, usePlanContext } from "../../logic";
import ChecksDrawer from "../ChecksView/ChecksDrawer.vue";
import PlanCheckStatusCount from "../PlanCheckStatusCount.vue";

const { plan, rollout } = usePlanContext();
const environmentStore = useEnvironmentV1Store();
const router = useRouter();
const { project } = useCurrentProjectV1();
const { hasAnyStatus: hasAnyChecks } = usePlanCheckStatus(plan);

const selectedResultStatus = ref<Advice_Level | undefined>(undefined);

const getEnvironmentEntity = (environmentName: string) => {
  return environmentStore.getEnvironmentByName(environmentName);
};

const navigateToStage = (stageName: string) => {
  if (!rollout?.value) return;

  const planId = extractPlanUIDFromRolloutName(rollout.value.name);
  const stageId = stageName.split("/").pop();

  if (!planId || !stageId) return;

  router.push({
    name: PROJECT_V1_ROUTE_PLAN_ROLLOUT_STAGE,
    params: {
      projectId: extractProjectResourceName(project.value.name),
      planId,
      stageId,
    },
  });
};
</script>
