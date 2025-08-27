<template>
  <div class="w-full h-full flex flex-col">
    <!-- Content area -->
    <div class="w-full flex-1 p-4">
      <div v-if="!ready" class="flex items-center justify-center py-12">
        <BBSpin class="w-6 h-6 text-primary" />
      </div>
      <div
        v-else-if="mergedStages.length === 0"
        class="flex items-center justify-center py-12"
      >
        <div class="flex flex-col items-center gap-4 max-w-md text-center">
          <div
            class="w-16 h-16 rounded-full bg-gray-100 flex items-center justify-center"
          >
            <LayersIcon class="w-8 h-8 text-gray-400" />
          </div>
          <div class="space-y-2">
            <h3 class="font-medium text-xl text-gray-900">
              {{ $t("rollout.stage.no-stages.self") }}
            </h3>
            <p class="text-base text-gray-500">
              {{ $t("rollout.stage.no-stages.description") }}
            </p>
          </div>
        </div>
      </div>
      <template v-else>
        <StagesView
          :rollout="rollout"
          :merged-stages="mergedStages"
          @run-tasks="handleRunTasks"
          @create-rollout-to-stage="handleCreateRolloutToStage"
        />
      </template>

      <!-- Rollback Section -->
      <TaskRunRollbackSection :rollout="rollout" />
    </div>

    <!-- Task Rollout Action Panel -->
    <TaskRolloutActionPanel
      v-if="runTasksPanel.show && runTasksPanel.target"
      :show="runTasksPanel.show"
      action="RUN"
      :target="runTasksPanel.target"
      @confirm="handleTaskActionPanelConfirm"
      @close="handleTaskActionPanelClose"
    />
  </div>
</template>

<script setup lang="ts">
import { create } from "@bufbuild/protobuf";
import { LayersIcon } from "lucide-vue-next";
import { ref } from "vue";
import { useI18n } from "vue-i18n";
import { BBSpin } from "@/bbkit";
import { rolloutServiceClientConnect } from "@/grpcweb";
import { pushNotification, useCurrentProjectV1 } from "@/store";
import { CreateRolloutRequestSchema } from "@/types/proto-es/v1/rollout_service_pb";
import type { Stage, Task } from "@/types/proto-es/v1/rollout_service_pb";
import { usePlanContextWithRollout } from "../../logic";
import StagesView from "./StagesView.vue";
import TaskRolloutActionPanel from "./TaskRolloutActionPanel.vue";
import TaskRunRollbackSection from "./TaskRunRollbackSection.vue";
import { useRolloutViewContext } from "./context";

const { t } = useI18n();
const { project } = useCurrentProjectV1();
const { events } = usePlanContextWithRollout();
const { rollout, mergedStages, ready } = useRolloutViewContext();

const runTasksPanel = ref<{
  show: boolean;
  target?: { type: "tasks"; stage: Stage; tasks: Task[] };
}>({
  show: false,
});

const handleRunTasks = (stage: Stage, tasks: Task[]) => {
  runTasksPanel.value = {
    show: true,
    target: { type: "tasks", stage, tasks },
  };
};

const handleCreateRolloutToStage = async (stage: Stage) => {
  try {
    const request = create(CreateRolloutRequestSchema, {
      parent: project.value.name,
      rollout: {
        plan: rollout.value.plan,
      },
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

const handleTaskActionPanelConfirm = () => {
  // Refresh the rollout data after task action is confirmed.
  events.emit("status-changed", { eager: true });
  handleTaskActionPanelClose();
};

const handleTaskActionPanelClose = () => {
  runTasksPanel.value = {
    show: false,
  };
};
</script>
