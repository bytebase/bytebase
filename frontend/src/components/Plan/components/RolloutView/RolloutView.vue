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
      <StagesView
        v-if="!isTableView"
        :merged-stages="mergedStages"
        :task-status-filter="taskStatusFilter"
      />
      <TaskTable
        v-else
        :rollout="rollout"
        :task-status-filter="taskStatusFilter"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ListIcon, Columns3Icon } from "lucide-vue-next";
import { NButton, NButtonGroup } from "naive-ui";
import { ref } from "vue";
import { Task_Status } from "@/types/proto/v1/rollout_service";
import StagesView from "./StagesView.vue";
import TaskFilter from "./TaskFilter.vue";
import TaskTable from "./TaskTable.vue";
import { provideRolloutViewContext } from "./context";

// Provide the context and get its values directly
const { rollout, mergedStages } = provideRolloutViewContext();

const isTableView = ref(false);
const taskStatusFilter = ref<Task_Status[]>([]);
</script>
