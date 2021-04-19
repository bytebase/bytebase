<template>
  <div class="md:flex md:items-center md:justify-between">
    <div class="flex-1 min-w-0">
      <div class="flex flex-col">
        <div class="flex items-center">
          <div>
            <TaskStatusIcon
              v-if="!$props.new"
              :taskStatus="task.status"
              :stageStatus="activeStage(task).status"
            />
          </div>
          <BBTextField
            class="ml-2 my-0.5 w-full text-lg font-bold"
            :disabled="!allowEdit"
            :required="true"
            :focusOnMount="$props.new"
            :bordered="false"
            :value="state.name"
            :placeholder="'Task name'"
            @end-editing="(text) => trySaveName(text)"
          />
        </div>
        <div v-if="!$props.new">
          <p class="text-sm text-control-light">
            #{{ task.id }} opened by
            <router-link
              :to="`/u/${task.creator.id}`"
              class="font-medium text-control hover:underline"
              >{{ task.creator.name }}</router-link
            >
            at
            <span href="#" class="font-medium text-control">{{
              moment(task.updatedTs).format("LLL")
            }}</span>
          </p>
        </div>
      </div>
    </div>
    <div class="mt-6 flex space-x-3 md:mt-0 md:ml-4">
      <slot />
    </div>
  </div>
</template>

<script lang="ts">
import { reactive, watch, PropType } from "vue";
import TaskStatusIcon from "../components/TaskStatusIcon.vue";
import { activeStage } from "../utils";
import { Task } from "../types";

interface LocalState {
  editing: boolean;
  name: string;
}

export default {
  name: "TaskHighlightPanel",
  emits: ["update-name"],
  props: {
    task: {
      required: true,
      type: Object as PropType<Task>,
    },
    new: {
      required: true,
      type: Boolean,
    },
    allowEdit: {
      required: true,
      type: Boolean,
    },
  },
  components: { TaskStatusIcon },
  setup(props, { emit }) {
    const state = reactive<LocalState>({
      editing: false,
      name: props.task.name,
    });

    watch(
      () => props.task,
      (curTask, _) => {
        state.name = curTask.name;
      }
    );

    const trySaveName = (text: string) => {
      state.name = text;
      if (text != props.task.name) {
        emit("update-name", state.name);
      }
    };

    return { state, activeStage, trySaveName };
  },
};
</script>
