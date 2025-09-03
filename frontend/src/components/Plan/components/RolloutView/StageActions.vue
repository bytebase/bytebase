<template>
  <template v-if="isCreated && runableTasks.length > 0">
    <NButton
      size="small"
      :disabled="!canRunTasks"
      @click.stop="$emit('run-tasks')"
    >
      <template #icon>
        <PlayIcon />
      </template>
      {{ $t("common.run") }}
    </NButton>
  </template>
  <template v-else-if="!isCreated && canCreateRollout">
    <NPopconfirm
      :negative-text="null"
      :positive-text="$t('common.confirm')"
      @positive-click="$emit('create-rollout')"
    >
      <template #trigger>
        <NTooltip>
          <template #trigger>
            <NButton :size="'small'">
              <template #icon>
                <CircleFadingPlusIcon class="w-5 h-5" />
              </template>
              {{ $t("common.start") }}
            </NButton>
          </template>
          {{ $t("rollout.stage.start-stage") }}
        </NTooltip>
      </template>
      {{ $t("common.confirm-and-add") }}
    </NPopconfirm>
  </template>
</template>

<script setup lang="ts">
import { CircleFadingPlusIcon, PlayIcon } from "lucide-vue-next";
import { NButton, NPopconfirm, NTooltip } from "naive-ui";
import type { Task } from "@/types/proto-es/v1/rollout_service_pb";

defineProps<{
  isCreated: boolean;
  runableTasks: Task[];
  canRunTasks: boolean;
  canCreateRollout: boolean;
}>();

defineEmits<{
  (event: "run-tasks"): void;
  (event: "create-rollout"): void;
}>();
</script>
