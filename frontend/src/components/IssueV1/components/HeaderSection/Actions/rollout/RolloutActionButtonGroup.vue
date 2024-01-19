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
import {
  StageRolloutAction,
  TaskRolloutAction,
} from "@/components/IssueV1/logic";
import RolloutActionButton from "./RolloutActionButton.vue";
import { RolloutAction } from "./common";

defineProps<{
  taskRolloutActionList: TaskRolloutAction[];
  stageRolloutActionList: StageRolloutAction[];
}>();

defineEmits<{
  (event: "perform-action", action: RolloutAction): void;
}>();
</script>
