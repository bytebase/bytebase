<template>
  <div class="rollout-view-v2 w-full min-h-screen bg-gray-50">
    <!-- Empty state: no stages created yet -->
    <div v-if="!hasStages" class="flex flex-col items-center justify-center py-20 gap-4">
      <p class="text-gray-500">{{ $t("rollout.no-tasks-created") }}</p>
      <NButton v-if="hasPendingTasks" type="primary" @click="showPreviewDrawer = true">
        <template #icon>
          <EyeIcon class="w-4 h-4" />
        </template>
        {{ $t("rollout.pending-tasks-preview.action") }}
      </NButton>
    </div>

    <!-- Normal view: stages exist -->
    <template v-else>
      <StageNavigationBar
        :selected-stage-id="selectedStage?.name"
        :rollout="rollout"
        :has-pending-tasks="hasPendingTasks"
        @select-stage="handleStageSelect"
        @open-preview="showPreviewDrawer = true"
      />

      <StageContentView
        :selected-stage="selectedStage"
        :rollout="rollout"
        :is-stage-created="() => true"
        @run-stage="handleRunStage"
        @create-stage="handleCreateStage"
      />
    </template>

    <PendingTasksPreviewDrawer
      :show="showPreviewDrawer"
      :plan="plan"
      :rollout="rollout"
      :project-name="project.name"
      @close="showPreviewDrawer = false"
      @created="handleTasksCreated"
    />

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
import { EyeIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute, useRouter } from "vue-router";
import { usePlanContextWithRollout } from "@/components/Plan/logic";
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
import { useExpectedTaskCount } from "./composables/useExpectedTaskCount";
import { useStageSelection } from "./composables/useStageSelection";
import { useTaskInstancePreload } from "./composables/useTaskInstancePreload";
import PendingTasksPreviewDrawer from "./PendingTasksPreviewDrawer.vue";
import StageContentView from "./StageContentView.vue";
import StageNavigationBar from "./StageNavigationBar.vue";
import type { TargetType } from "./TaskRolloutActionPanel.vue";
import TaskRolloutActionPanel from "./TaskRolloutActionPanel.vue";

const route = useRoute();
const router = useRouter();
const { t } = useI18n();
const { project } = useCurrentProjectV1();
const { rollout, plan, events } = usePlanContextWithRollout();

const routeStageId = computed(() => route.params.stageId as string | undefined);

const { selectedStage } = useStageSelection(
  computed(() => rollout.value.stages),
  routeStageId,
  rollout
);

useTaskInstancePreload(() =>
  selectedStage.value ? [selectedStage.value] : []
);

const hasStages = computed(() => rollout.value.stages.length > 0);
const { expectedTaskCount } = useExpectedTaskCount(plan);
const hasPendingTasks = computed(() => {
  const actualCount = rollout.value.stages.flatMap((s) => s.tasks).length;
  return expectedTaskCount.value > actualCount;
});

const showPreviewDrawer = ref(false);
const showStageActionPanel = ref(false);
const stageAction = ref<"RUN" | "SKIP" | "CANCEL">("RUN");
const stageActionTarget = ref<TargetType | null>(null);

watch(
  () => [selectedStage.value, routeStageId.value] as const,
  ([stage, currentRouteStageId]) => {
    if (stage && !currentRouteStageId) {
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
  stageAction.value = "RUN";
  stageActionTarget.value = { type: "tasks", stage };
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
  events.emit("status-changed", { eager: true });
  showStageActionPanel.value = false;
  stageActionTarget.value = null;
};

const handleTasksCreated = () => {
  events.emit("status-changed", { eager: true });
};
</script>
