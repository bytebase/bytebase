<template>
  <div class="rollout-view-v2 w-full min-h-screen bg-gray-50">
    <BBSpin v-if="!ready" class="flex justify-center py-12" />

    <template v-else>
      <StageNavigationBar
        :stages="mergedStages"
        :selected-stage-id="selectedStage?.name"
        :is-stage-created="isStageCreated"
        @select-stage="handleStageSelect"
      />

      <StageContentView
        :selected-stage="selectedStage"
        :rollout="rollout"
        :is-stage-created="isStageCreated"
        @run-stage="handleRunStage"
        @create-stage="handleCreateStage"
      />
    </template>

    <!-- Stage run action panel -->
    <TaskRolloutActionPanel
      v-if="showStageActionPanel && stageActionTarget"
      :show="showStageActionPanel"
      :action="stageAction"
      :target="stageActionTarget"
      @close="showStageActionPanel = false"
      @confirm="handleStageActionConfirmed"
    />
  </div>
</template>

<script lang="ts" setup>
import { create } from "@bufbuild/protobuf";
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute, useRouter } from "vue-router";
import BBSpin from "@/bbkit/BBSpin.vue";
import { usePlanContextWithRollout } from "@/components/Plan/logic";
import { useRolloutPreview } from "@/components/RolloutV1/logic";
import { rolloutServiceClientConnect } from "@/connect";
import { PROJECT_V1_ROUTE_PLAN_ROLLOUT_STAGE } from "@/router/dashboard/projectV1";
import { useCurrentProjectV1 } from "@/store";
import { pushNotification } from "@/store/modules/notification";
import type { Stage } from "@/types/proto-es/v1/rollout_service_pb";
import { CreateRolloutRequestSchema } from "@/types/proto-es/v1/rollout_service_pb";
import {
  extractPlanNameFromRolloutName,
  extractPlanUIDFromRolloutName,
  extractProjectResourceName,
} from "@/utils";
import { useStageSelection } from "./composables/useStageSelection";
import { useTaskInstancePreload } from "./composables/useTaskInstancePreload";
import StageContentView from "./StageContentView.vue";
import StageNavigationBar from "./StageNavigationBar.vue";
import type { TargetType } from "./TaskRolloutActionPanel.vue";
import TaskRolloutActionPanel from "./TaskRolloutActionPanel.vue";

const route = useRoute();
const router = useRouter();
const { t } = useI18n();
const { project } = useCurrentProjectV1();
const { rollout, plan, events } = usePlanContextWithRollout();
const { ready, mergedStages } = useRolloutPreview(
  rollout,
  plan,
  project.value.name
);

const routeStageId = computed(() => route.params.stageId as string | undefined);

const { selectedStage, isStageCreated } = useStageSelection(
  mergedStages,
  routeStageId,
  rollout
);

// Preload instances for tasks in the selected stage to ensure engine icons display
useTaskInstancePreload(() =>
  selectedStage.value ? [selectedStage.value] : []
);

// Stage action panel state
const showStageActionPanel = ref(false);
const stageAction = ref<"RUN" | "SKIP" | "CANCEL">("RUN");
const stageActionTarget = ref<TargetType | null>(null);

// Auto-navigate to the selected stage if no stageId in route
watch(
  () => [selectedStage.value, routeStageId.value, ready.value] as const,
  ([stage, currentRouteStageId, isReady]) => {
    if (isReady && stage && !currentRouteStageId) {
      const stageId = stage.name.split("/").pop();
      const planId = extractPlanUIDFromRolloutName(rollout.value.name);
      router.replace({
        name: PROJECT_V1_ROUTE_PLAN_ROLLOUT_STAGE,
        params: {
          projectId: extractProjectResourceName(project.value.name),
          planId: planId || "_",
          stageId: stageId || "_",
        },
      });
    }
  },
  { immediate: true }
);

const handleStageSelect = (stage: Stage) => {
  // Navigate to the proper stage route
  // Navigate to the proper stage route
  const stageId = stage.name.split("/").pop();
  const planId = extractPlanUIDFromRolloutName(rollout.value.name);

  router.push({
    name: PROJECT_V1_ROUTE_PLAN_ROLLOUT_STAGE,
    params: {
      projectId: extractProjectResourceName(project.value.name),
      planId: planId || "_",
      stageId: stageId || "_",
    },
  });
};

const handleRunStage = (stage: Stage) => {
  // Show confirmation panel for running all runnable tasks in the stage
  stageAction.value = "RUN";
  stageActionTarget.value = {
    type: "tasks",
    stage,
  };
  showStageActionPanel.value = true;
};

const handleCreateStage = async (stage: Stage) => {
  try {
    const request = create(CreateRolloutRequestSchema, {
      parent: extractPlanNameFromRolloutName(rollout.value.name),
      target: stage.environment,
    });
    await rolloutServiceClientConnect.createRollout(request);

    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.success"),
      description: t("common.created"),
    });

    // Trigger immediate refresh of rollout data
    events.emit("status-changed", { eager: true });
  } catch (error) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("common.error"),
      description: String(error),
    });
  }
};

const handleStageActionConfirmed = () => {
  // Refresh the rollout data after task action is confirmed
  events.emit("status-changed", { eager: true });
  // Close the panel and reset state
  showStageActionPanel.value = false;
  stageActionTarget.value = null;
};
</script>
