<template>
  <div class="md:flex md:items-center md:justify-between">
    <div class="flex-1 min-w-0">
      <div class="flex items-center">
        <div>
          <div class="flex items-center">
            <TaskStatusIcon
              v-if="!$props.new"
              :taskStatus="task.status"
              :stageStatus="activeStage(task).status"
            />
            <p
              class="ml-2 text-xl font-bold leading-7 text-main whitespace-nowrap md:w-96 lg:w-160 truncate"
            >
              {{ task.name }}
            </p>
          </div>
          <div v-if="!$props.new">
            <p class="mt-2 text-sm text-gray-500">
              #{{ task.id }} opened by
              <span href="#" class="font-medium text-control">{{
                task.creator.name
              }}</span>
              at
              <span href="#" class="font-medium text-control">{{
                moment(task.lastUpdatedTs).format("LLL")
              }}</span>
            </p>
          </div>
        </div>
      </div>
    </div>
    <div class="mt-6 flex space-x-3 md:mt-0 md:ml-4">
      <slot />
    </div>
  </div>
</template>

<script lang="ts">
import { PropType } from "vue";
import TaskStatusIcon from "../components/TaskStatusIcon.vue";
import { activeStage } from "../utils";
import { Task } from "../types";

export default {
  name: "TaskHighlightPanel",
  props: {
    task: {
      required: true,
      type: Object as PropType<Task>,
    },
    new: {
      required: true,
      type: Boolean,
    },
  },
  components: { TaskStatusIcon },
  setup(props, ctx) {
    return { activeStage };
  },
};
</script>
