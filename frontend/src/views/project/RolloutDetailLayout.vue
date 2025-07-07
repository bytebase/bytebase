<template>
  <div class="w-full flex flex-col">
    <!-- Breadcrumb - only show on stage and task routes -->
    <NBreadcrumb v-if="showBreadcrumb" class="px-4 pt-2">
      <NBreadcrumbItem @click="navigateToRollout">
        {{ $t("common.rollout") }}
      </NBreadcrumbItem>
      <NBreadcrumbItem v-if="stageId" @click="navigateToStage">
        {{ stageTitle }}
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
import { provideRolloutViewContext } from "@/components/Plan/components/RolloutView/context";
import {
  PROJECT_V1_ROUTE_ROLLOUT_DETAIL,
  PROJECT_V1_ROUTE_ROLLOUT_DETAIL_STAGE_DETAIL,
} from "@/router/dashboard/projectV1";
import { useCurrentProjectV1, useEnvironmentV1Store } from "@/store";
import { extractProjectResourceName } from "@/utils";

const route = useRoute();
const router = useRouter();
const { project } = useCurrentProjectV1();
const environmentStore = useEnvironmentV1Store();
const { mergedStages } = provideRolloutViewContext();

// Route parameters
const rolloutId = computed(() => route.params.rolloutId as string);
const stageId = computed(() => route.params.stageId as string);
const taskId = computed(() => route.params.taskId as string);

// Only show breadcrumb on stage and task routes
const showBreadcrumb = computed(() => {
  return Boolean(stageId.value || taskId.value);
});

// Get stage title from environment
const stageTitle = computed(() => {
  if (!stageId.value) return "";

  // Find the stage in merged stages
  const stage = mergedStages.value.find((s) =>
    s.name.endsWith(`/${stageId.value}`)
  );

  if (stage) {
    const environment = environmentStore.getEnvironmentByName(
      stage.environment
    );
    return environment.title;
  }

  return stageId.value;
});

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
