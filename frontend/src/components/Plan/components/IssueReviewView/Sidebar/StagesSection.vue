<template>
  <div v-if="rollout" class="flex flex-col gap-y-1">
    <h3 class="textlabel">
      {{ $t("rollout.stage.self", 2) }}
    </h3>
    <div class="flex flex-wrap items-center gap-y-1">
      <template v-if="rollout.stages.length > 0">
        <template v-for="(stage, index) in rollout.stages" :key="stage.name">
          <div
            class="flex items-center gap-1 cursor-pointer hover:opacity-80 transition-opacity"
            @click="navigateToStage(stage.name)"
          >
            <TaskStatus :status="getStageStatus(stage)" size="small" disabled />
            <EnvironmentV1Name
              :environment="getEnvironmentEntity(stage.environment)"
              :link="false"
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
</template>

<script setup lang="ts">
import { useRouter } from "vue-router";
import TaskStatus from "@/components/RolloutV1/components/Task/TaskStatus.vue";
import { EnvironmentV1Name } from "@/components/v2";
import { buildStageRoute } from "@/router/dashboard/projectV1RouteHelpers";
import { useEnvironmentV1Store } from "@/store";
import { getStageStatus } from "@/utils";
import { usePlanContext } from "../../../logic";

const { rollout } = usePlanContext();
const environmentStore = useEnvironmentV1Store();
const router = useRouter();

const getEnvironmentEntity = (environmentName: string) => {
  return environmentStore.getEnvironmentByName(environmentName);
};

const navigateToStage = (stageName: string) => {
  router.push(buildStageRoute(stageName));
};
</script>
