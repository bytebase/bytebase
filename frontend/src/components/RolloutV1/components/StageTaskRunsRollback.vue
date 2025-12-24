<template>
  <div v-if="rollbackableTaskRuns.length > 0">
    <NButton size="small" text @click="showRollbackDrawer = true">
      <template #icon>
        <DatabaseBackupIcon :size="16" />
      </template>
      {{ $t("task-run.rollback.available", rollbackableTaskRuns.length) }}
    </NButton>

    <TaskRunRollbackDrawer
      v-model:show="showRollbackDrawer"
      :rollout="rollout"
      :rollbackable-task-runs="rollbackableTaskRuns"
      @close="showRollbackDrawer = false"
    />
  </div>
</template>

<script lang="ts" setup>
import { DatabaseBackupIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { ref, toRef } from "vue";
import { usePlanContextWithRollout } from "@/components/Plan/logic";
import type { Stage } from "@/types/proto-es/v1/rollout_service_pb";
import { useRollbackableTasks } from "./composables/useRollbackableTasks";
import TaskRunRollbackDrawer from "./TaskRunRollbackDrawer.vue";

const props = defineProps<{
  stage: Stage | null | undefined;
}>();

const { rollout, taskRuns } = usePlanContextWithRollout();
const stageRef = toRef(props, "stage");
const { rollbackableTaskRuns } = useRollbackableTasks(stageRef, taskRuns);

const showRollbackDrawer = ref(false);
</script>
