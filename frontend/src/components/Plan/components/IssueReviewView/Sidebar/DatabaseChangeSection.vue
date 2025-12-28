<template>
  <template v-if="shouldShowSection">
    <!-- Stages -->
    <div v-if="rollout" class="flex flex-col gap-y-1">
      <h3 class="textlabel">
        {{ $t("rollout.stage.self", 2) }}
      </h3>
      <div class="flex flex-wrap items-center gap-y-1">
        <template v-if="rollout.stages.length > 0">
          <template
            v-for="(stage, index) in rollout.stages"
            :key="stage.name"
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
              class="mx-2 text-control-placeholder"
            >
              →
            </span>
          </template>
        </template>
        <span v-else class="text-sm text-control-placeholder">
          {{ $t("common.no-data") }}
        </span>
      </div>
    </div>

    <!-- SQL Checks -->
    <div class="flex flex-col gap-y-1">
      <h3 class="textlabel">
        {{ $t("plan.navigator.checks") }}
      </h3>
      <div class="flex items-center gap-2">
        <PlanCheckStatusCount
          :plan="plan"
          clickable
          @click="selectedResultStatus = $event"
        />
        <span v-if="!hasAnyChecks" class="text-sm text-control-placeholder">
          {{ $t("plan.overview.no-checks") }}
        </span>
      </div>
    </div>

    <NDivider class="my-0!" />

    <ChecksDrawer
      v-if="selectedResultStatus"
      :status="selectedResultStatus"
      @close="selectedResultStatus = undefined"
    />
  </template>
</template>

<script setup lang="ts">
import { NDivider } from "naive-ui";
import { computed, ref } from "vue";
import { useRouter } from "vue-router";
import TaskStatus from "@/components/Rollout/kits/TaskStatus.vue";
import { EnvironmentV1Name } from "@/components/v2";
import { PROJECT_V1_ROUTE_ROLLOUT_DETAIL_STAGE_DETAIL } from "@/router/dashboard/projectV1";
import { useCurrentProjectV1, useEnvironmentV1Store } from "@/store";
import { Issue_Type } from "@/types/proto-es/v1/issue_service_pb";
import { Advice_Level } from "@/types/proto-es/v1/sql_service_pb";
import {
  extractPlanUIDFromRolloutName,
  extractProjectResourceName,
  getStageStatus,
} from "@/utils";
import { usePlanCheckStatus, usePlanContext } from "../../../logic";
import ChecksDrawer from "../../ChecksView/ChecksDrawer.vue";
import PlanCheckStatusCount from "../../PlanCheckStatusCount.vue";

const { plan, rollout, issue } = usePlanContext();
const environmentStore = useEnvironmentV1Store();
const router = useRouter();
const { project } = useCurrentProjectV1();
const { hasAnyStatus: hasAnyChecks } = usePlanCheckStatus(plan);

const selectedResultStatus = ref<Advice_Level | undefined>(undefined);

const shouldShowSection = computed(() => {
  return issue.value?.type === Issue_Type.DATABASE_CHANGE || rollout?.value;
});

const getEnvironmentEntity = (environmentName: string) => {
  return environmentStore.getEnvironmentByName(environmentName);
};

const navigateToStage = (stageName: string) => {
  if (!rollout?.value) return;

  const planId = extractPlanUIDFromRolloutName(rollout.value.name);
  const stageId = stageName.split("/").pop();

  if (!planId || !stageId) return;

  router.push({
    name: PROJECT_V1_ROUTE_ROLLOUT_DETAIL_STAGE_DETAIL,
    params: {
      projectId: extractProjectResourceName(project.value.name),
      planId,
      stageId,
    },
  });
};
</script>
