<template>
  <div class="w-full flex flex-col">
    <NBreadcrumb class="px-4 pt-2">
      <NBreadcrumbItem :clickable="false">
        {{ $t("common.rollout") }}
      </NBreadcrumbItem>
      <NBreadcrumbItem @click="navigateToRollout">
        <span>{{ $t("rollout.stage.self", 2) }}</span>
        <span v-if="rollout.stages.length > 1" class="opacity-80 ml-1"
          >({{ rollout.stages.length }})</span
        >
      </NBreadcrumbItem>
      <NBreadcrumbItem v-if="stageId" @click="navigateToStage">
        <EnvironmentV1Name
          v-if="stage"
          :environment="
            environmentStore.getEnvironmentByName(stage.environment)
          "
          :link="false"
        />
        <span v-else>{{ stageId }}</span>
      </NBreadcrumbItem>
      <NBreadcrumbItem v-if="taskId" :clickable="false">
        {{ $t("common.task") }} #{{ taskId }}
      </NBreadcrumbItem>
    </NBreadcrumb>

    <!-- Main content -->
    <div class="flex-1 min-h-0">
      <router-view />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { NBreadcrumb, NBreadcrumbItem } from "naive-ui";
import { computed } from "vue";
import { useRoute, useRouter } from "vue-router";
import { usePlanContextWithRollout } from "@/components/Plan";
import { provideRolloutViewContext } from "@/components/Plan/components/RolloutView/context";
import { EnvironmentV1Name } from "@/components/v2";
import {
  PROJECT_V1_ROUTE_ROLLOUT_DETAIL,
  PROJECT_V1_ROUTE_ROLLOUT_DETAIL_STAGE_DETAIL,
} from "@/router/dashboard/projectV1";
import { useCurrentProjectV1, useEnvironmentV1Store } from "@/store";
import { extractProjectResourceName } from "@/utils";

const route = useRoute();
const router = useRouter();
const { rollout } = usePlanContextWithRollout();
const { project } = useCurrentProjectV1();
const environmentStore = useEnvironmentV1Store();
const { mergedStages } = provideRolloutViewContext();

// Route parameters
const rolloutId = computed(() => route.params.rolloutId as string);
const stageId = computed(() => route.params.stageId as string);
const taskId = computed(() => route.params.taskId as string);
const stage = computed(() =>
  mergedStages.value.find((s) => s.name.endsWith(`/${stageId.value}`))
);

// Navigation handlers
const navigateToRollout = () => {
  router.push({
    name: PROJECT_V1_ROUTE_ROLLOUT_DETAIL,
    params: {
      projectId: extractProjectResourceName(project.value.name),
      rolloutId: rolloutId.value,
    },
  });
};

const navigateToStage = () => {
  if (!stageId.value) return;

  router.push({
    name: PROJECT_V1_ROUTE_ROLLOUT_DETAIL_STAGE_DETAIL,
    params: {
      projectId: extractProjectResourceName(project.value.name),
      rolloutId: rolloutId.value,
      stageId: stageId.value,
    },
  });
};
</script>
