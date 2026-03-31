<template>
  <div class="flex flex-col">
    <template v-if="rollout">
      <!-- Empty state -->
      <div v-if="!hasStages" class="flex flex-col items-center justify-center py-10 gap-4">
        <p class="text-control-placeholder">{{ $t("rollout.no-tasks-created") }}</p>
        <NButton v-if="hasPendingTasks" type="primary" @click="showPreviewDrawer = true">
          {{ $t("rollout.pending-tasks-preview.action") }}
        </NButton>
      </div>

      <!-- Stages + Tasks -->
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
    </template>
  </div>
</template>

<script setup lang="ts">
import { create } from "@bufbuild/protobuf";
import { NButton } from "naive-ui";
import { computed, type Ref, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute, useRouter } from "vue-router";
import StageContentView from "@/components/RolloutV1/components/Stage/StageContentView.vue";
import StageNavigationBar from "@/components/RolloutV1/components/Stage/StageNavigationBar.vue";
import { useStageSelection } from "@/components/RolloutV1/components/Stage/useStageSelection";
import PendingTasksPreviewDrawer from "@/components/RolloutV1/components/Task/PendingTasksPreviewDrawer.vue";
import type { TargetType } from "@/components/RolloutV1/components/Task/TaskRolloutActionPanel.vue";
import TaskRolloutActionPanel from "@/components/RolloutV1/components/Task/TaskRolloutActionPanel.vue";
import { useExpectedTaskCount } from "@/components/RolloutV1/components/Task/useExpectedTaskCount";
import { useTaskInstancePreload } from "@/components/RolloutV1/components/Task/useTaskInstancePreload";
import { rolloutServiceClientConnect } from "@/connect";
import {
  buildPlanDeployRouteFromPlanName,
  getRouteQueryString,
} from "@/router/dashboard/projectV1RouteHelpers";
import { pushNotification, useCurrentProjectV1 } from "@/store";
import type { Rollout } from "@/types/proto-es/v1/rollout_service_pb";
import {
  CreateRolloutRequestSchema,
  type Stage,
} from "@/types/proto-es/v1/rollout_service_pb";
import { extractPlanNameFromRolloutName } from "@/utils";
import { emitPlanStatusChanged, usePlanContext } from "../../logic";

const { t } = useI18n();
const route = useRoute();
const router = useRouter();
const { plan, rollout, events } = usePlanContext();
const { project } = useCurrentProjectV1();

const hasStages = computed(() => (rollout.value?.stages.length ?? 0) > 0);

const routeStageId = computed<string | undefined>(() => {
  return getRouteQueryString(route.query.stageId);
});
const { selectedStage } = useStageSelection(
  computed(() => rollout.value?.stages ?? []),
  routeStageId,
  rollout as Ref<Rollout>
);

useTaskInstancePreload(() =>
  selectedStage.value ? [selectedStage.value] : []
);

const { expectedTaskCount } = useExpectedTaskCount(plan);
const hasPendingTasks = computed(() => {
  if (!rollout.value) return false;
  const actualCount = rollout.value.stages.flatMap((s) => s.tasks).length;
  return expectedTaskCount.value > actualCount;
});

const showPreviewDrawer = ref(false);
const showStageActionPanel = ref(false);
const stageAction = ref<"RUN" | "SKIP" | "CANCEL">("RUN");
const stageActionTarget = ref<TargetType | null>(null);

const handleStageSelect = (stage: Stage) => {
  const stageId = stage.name.split("/").pop();
  router.push(
    buildPlanDeployRouteFromPlanName(plan.value.name, {
      stageId,
    })
  );
};

const handleRunStage = (stage: Stage) => {
  stageAction.value = "RUN";
  stageActionTarget.value = { type: "tasks", stage };
  showStageActionPanel.value = true;
};

const handleCreateStage = async (stage: Stage) => {
  if (!rollout.value) return;
  try {
    const request = create(CreateRolloutRequestSchema, {
      parent: extractPlanNameFromRolloutName(rollout.value.name),
      target: stage.environment,
    });
    await rolloutServiceClientConnect.createRollout(request);
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.created"),
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
  emitPlanStatusChanged(events, { refreshMode: "fast-follow" });
  showStageActionPanel.value = false;
  stageActionTarget.value = null;
};

const handleTasksCreated = () => {
  emitPlanStatusChanged(events);
};
</script>
