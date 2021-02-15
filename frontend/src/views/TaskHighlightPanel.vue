<template>
  <div class="">
    <div
      class="px-4 py-6 md:flex md:items-center md:justify-between lg:border-t lg:border-block-border"
    >
      <div class="flex-1 min-w-0">
        <div class="flex items-center">
          <div>
            <div class="flex items-center">
              <TaskStatusIcon :task="task" />
              <!-- [TODO] overflow-ellipsis/clip doesn't seem to be working, so just use nowrap -->
              <p
                class="ml-1 text-xl font-bold leading-7 text-main whitespace-nowrap"
              >
                {{ task.attributes.name }}
              </p>
            </div>
            <div v-if="!state.new">
              <p class="mt-2 text-sm text-gray-500">
                #{{ task.id }} opened by
                <span href="#" class="font-medium text-control">{{
                  task.attributes.creator.name
                }}</span>
                at
                <span href="#" class="font-medium text-control">{{
                  moment(task.attributes.lastUpdatedTs).format("LLL")
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
  </div>
</template>

<script lang="ts">
import { PropType, watchEffect, reactive } from "vue";
import isEmpty from "lodash-es/isEmpty";
import TaskStatusIcon from "../components/TaskStatusIcon.vue";
import { Task } from "../types";

interface LocalState {
  new: boolean;
}

export default {
  name: "TaskHighlightPanel",
  props: {
    task: {
      required: true,
      type: Object as PropType<Task>,
    },
  },
  components: { TaskStatusIcon },
  setup(props, ctx) {
    const state = reactive<LocalState>({
      new: isEmpty(props.task.id),
    });

    const refreshState = () => {
      state.new = isEmpty(props.task.id);
    };

    watchEffect(refreshState);

    return { state };
  },
};
</script>
