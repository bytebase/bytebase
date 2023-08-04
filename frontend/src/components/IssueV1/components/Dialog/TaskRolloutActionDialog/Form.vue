<template>
  <div class="mt-2 flex flex-col gap-y-4">
    <div class="text-sm">
      <label v-if="taskList.length > 1" class="textlabel">
        {{ $t("common.tasks") }}
      </label>
      <ul class="mt-1 max-h-[6rem] overflow-y-auto">
        <li
          v-for="item in distinctTaskList"
          :key="item.task.uid"
          class="text-sm textinfolabel"
        >
          <span class="textinfolabel">
            {{ item.task.title }}
          </span>
          <span v-if="item.similar.length > 0" class="ml-2 text-gray-400">
            {{
              $t("task.n-similar-tasks", {
                count: item.similar.length + 1,
              })
            }}
          </span>
        </li>
      </ul>
    </div>
    <div class="flex flex-col gap-y-1">
      <p class="textlabel">
        {{ $t("common.comment") }}
      </p>
      <NInput
        v-model:value="comment"
        type="textarea"
        :placeholder="$t('issue.leave-a-comment')"
        :autosize="{
          minRows: 3,
          maxRows: 10,
        }"
      />
    </div>
    <div class="py-1 flex justify-end gap-x-3">
      <NButton @click="$emit('cancel')">
        {{ $t("common.cancel") }}
      </NButton>
      <NButton
        v-bind="taskRolloutActionButtonProps(action)"
        @click="$emit('confirm', action, comment)"
      >
        {{ taskRolloutActionDialogButtonName(action, taskList) }}
      </NButton>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, ref } from "vue";
import {
  TaskRolloutAction,
  taskRolloutActionButtonProps,
  taskRolloutActionDialogButtonName,
} from "@/components/IssueV1/logic";
import { Task } from "@/types/proto/v1/rollout_service";
import { groupBy } from "lodash-es";

const props = defineProps<{
  action: TaskRolloutAction;
  taskList: Task[];
}>();

defineEmits<{
  (event: "cancel"): void;
  (event: "confirm", action: TaskRolloutAction, comment?: string): void;
}>();

const comment = ref("");

const distinctTaskList = computed(() => {
  type DistinctTaskList = { task: Task; similar: Task[] };
  const groups = groupBy(props.taskList, (task) => task.title);

  return Object.keys(groups).map<DistinctTaskList>((taskName) => {
    const [task, ...similar] = groups[taskName];
    return { task, similar };
  });
});
</script>
