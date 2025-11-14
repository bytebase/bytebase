<template>
  <div class="flex items-center gap-x-3">
    <RolloutActionButton
      v-for="action in taskRolloutActionList"
      :key="action"
      :action="action"
      :stage-rollout-action-list="stageRolloutActionList"
      @perform-action="$emit('perform-action', $event)"
    />
  </div>
</template>

<script lang="ts" setup>
import type {
  StageRolloutAction,
  TaskRolloutAction,
} from "@/components/IssueV1/logic";
import type { RolloutAction } from "./common";
import RolloutActionButton from "./RolloutActionButton.vue";

defineProps<{
  taskRolloutActionList: TaskRolloutAction[];
  stageRolloutActionList: StageRolloutAction[];
}>();

defineEmits<{
  (event: "perform-action", action: RolloutAction): void;
}>();
</script>
