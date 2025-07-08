<template>
  <div class="w-full h-full flex flex-col">
    <div class="flex justify-between items-center gap-2 px-4 pt-4">
      <!-- Task Filter -->
      <TaskFilter
        v-model:task-status-list="taskStatusFilter"
        :rollout="rollout"
      />
      <!-- View Toggle -->
      <NButtonGroup size="small">
        <NButton
          :type="!isTableView ? 'primary' : 'tertiary'"
          @click="isTableView = false"
        >
          <template #icon>
            <Columns3Icon :size="16" />
          </template>
        </NButton>
        <NButton
          :type="isTableView ? 'primary' : 'tertiary'"
          @click="isTableView = true"
        >
          <template #icon>
            <ListIcon :size="16" />
          </template>
        </NButton>
      </NButtonGroup>
    </div>

    <!-- Content area -->
    <div class="flex-1 overflow-hidden">
      <div v-if="!isTableView" class="w-full p-4">
        <div class="w-full overflow-scroll border rounded-lg p-6 bg-zinc-50">
          <StagesView
            :rollout="rollout"
            :merged-stages="mergedStages"
            :task-status-filter="taskStatusFilter"
            @run-tasks="handleRunTasks"
            @create-rollout-to-stage="handleCreateRolloutToStage"
          />
        </div>
      </div>
      <TaskTableView v-else :task-status-filter="taskStatusFilter" />
    </div>

    <!-- Task Rollout Action Panel -->
    <TaskRolloutActionPanel
      v-if="runTasksPanel.show && runTasksPanel.target"
      :show="runTasksPanel.show"
      action="RUN"
      :target="runTasksPanel.target"
      @close="handlePanelClose"
    />
  </div>
</template>

<script setup lang="ts">
import { create } from "@bufbuild/protobuf";
import { ListIcon, Columns3Icon } from "lucide-vue-next";
import { NButton, NButtonGroup } from "naive-ui";
import { ref } from "vue";
import { useI18n } from "vue-i18n";
import { rolloutServiceClientConnect } from "@/grpcweb";
import { pushNotification, useCurrentProjectV1 } from "@/store";
import { CreateRolloutRequestSchema } from "@/types/proto-es/v1/rollout_service_pb";
import type { Stage, Task } from "@/types/proto-es/v1/rollout_service_pb";
import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";
import { usePlanContextWithRollout } from "../../logic";
import StagesView from "./StagesView.vue";
import TaskFilter from "./TaskFilter.vue";
import TaskRolloutActionPanel from "./TaskRolloutActionPanel.vue";
import TaskTableView from "./TaskTableView.vue";
import { useRolloutViewContext } from "./context";

const { t } = useI18n();
const { project } = useCurrentProjectV1();
const { events } = usePlanContextWithRollout();
const { rollout, mergedStages } = useRolloutViewContext();

const isTableView = ref(false);
const taskStatusFilter = ref<Task_Status[]>([]);
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

const handlePanelClose = () => {
  runTasksPanel.value = {
    show: false,
  };
};
</script>
