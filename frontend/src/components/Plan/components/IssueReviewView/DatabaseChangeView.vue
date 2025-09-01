<template>
  <div v-if="shouldShowOverview" class="w-full overflow-y-auto">
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
import { computed, ref } from "vue";
import { useRouter } from "vue-router";
import TaskStatus from "@/components/Rollout/kits/TaskStatus.vue";
import { EnvironmentV1Name } from "@/components/v2";
import { PROJECT_V1_ROUTE_ROLLOUT_DETAIL_STAGE_DETAIL } from "@/router/dashboard/projectV1";
import { useCurrentProjectV1, useEnvironmentV1Store } from "@/store";
import { Issue_Type } from "@/types/proto-es/v1/issue_service_pb";
import { PlanCheckRun_Result_Status } from "@/types/proto-es/v1/plan_service_pb";
import { extractProjectResourceName, getStageStatus } from "@/utils";
import { usePlanContextWithIssue } from "../..";
import { usePlanContext, usePlanCheckStatus } from "../../logic";
import ChecksDrawer from "../ChecksView/ChecksDrawer.vue";
import PlanCheckStatusCount from "../PlanCheckStatusCount.vue";

const { issue, rollout } = usePlanContextWithIssue();
const { plan } = usePlanContext();
const environmentStore = useEnvironmentV1Store();
const router = useRouter();
const { project } = useCurrentProjectV1();
const { hasAnyStatus: hasAnyChecks } = usePlanCheckStatus(plan);

const selectedResultStatus = ref<PlanCheckRun_Result_Status | undefined>(
  undefined
);

const shouldShowOverview = computed(() => {
  return issue.value.type === Issue_Type.DATABASE_CHANGE || rollout?.value;
});

const getEnvironmentEntity = (environmentName: string) => {
  return environmentStore.getEnvironmentByName(environmentName);
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
