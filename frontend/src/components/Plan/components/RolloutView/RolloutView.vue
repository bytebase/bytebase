<template>
  <div class="w-full h-full flex flex-col">
    <!-- Content area -->
    <div class="w-full flex-1 p-4">
      <div class="flex items-center gap-1 mb-4 pl-4">
        <h3 class="text-base font-medium">
          {{ $t("rollout.stage.self", mergedStages.length) }}
        </h3>
        <span class="text-control-light" v-if="mergedStages.length > 1"
          >({{ mergedStages.length }})</span
        >
      </div>
      <StagesView
        :rollout="rollout"
        :merged-stages="mergedStages"
        :readonly="readonly"
        @run-tasks="handleRunTasks"
        @create-rollout-to-stage="handleCreateRolloutToStage"
      />
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
import { ref } from "vue";
import { useI18n } from "vue-i18n";
import { rolloutServiceClientConnect } from "@/grpcweb";
import { pushNotification, useCurrentProjectV1 } from "@/store";
import { CreateRolloutRequestSchema } from "@/types/proto-es/v1/rollout_service_pb";
import type { Stage, Task } from "@/types/proto-es/v1/rollout_service_pb";
import { usePlanContextWithRollout } from "../../logic";
import StagesView from "./StagesView.vue";
import TaskRolloutActionPanel from "./TaskRolloutActionPanel.vue";
import { useRolloutViewContext } from "./context";

const { t } = useI18n();
const { project } = useCurrentProjectV1();
const { events, readonly } = usePlanContextWithRollout();
const { rollout, mergedStages } = useRolloutViewContext();

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
