<template>
  <div class="md:flex md:items-center md:justify-between">
    <div class="flex-1 min-w-0">
      <div class="flex flex-col">
        <div class="flex items-center">
          <div>
            <TaskStatusIcon
              v-if="!$props.new"
              class="mt-0.5"
              :taskStatus="task.status"
              :stageStatus="activeStage(task).status"
            />
          </div>
          <input
            required
            autocomplete="off"
            ref="nameTextField"
            id="name"
            name="name"
            type="text"
            placeholder="Task name"
            class="ml-2 my-0.5 w-full text-main focus:ring-control sm:text-lg font-bold rounded-md"
            :class="
              state.editing
                ? 'focus:border-control border-control-border'
                : 'border-white'
            "
            v-model="state.name"
            @click.prevent="state.editing = true"
            @blur="trySaveName"
          />
        </div>
        <div v-if="!$props.new">
          <p class="text-sm text-control-light">
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
    <div class="mt-6 flex space-x-3 md:mt-0 md:ml-4">
      <slot />
    </div>
  </div>
</template>

<script lang="ts">
import { reactive, ref, nextTick, watch, PropType } from "vue";
import TaskStatusIcon from "../components/TaskStatusIcon.vue";
import { activeStage } from "../utils";
import { Task } from "../types";
import { isEmpty } from "lodash";

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
  },
  components: { TaskStatusIcon },
  setup(props, { emit }) {
    const nameTextField = ref();

    const state = reactive<LocalState>({
      editing: false,
      name: props.task.name,
    });

    watch(
      () => props.task,
      (curTask, prevTask) => {
        state.name = curTask.name;
      }
    );

    const trySaveName = () => {
      if (isEmpty(state.name)) {
        nextTick(() => {
          nameTextField.value.focus();
        });
      } else if (state.name != props.task.name) {
        emit("update-name", state.name, (updatedTask: Task) => {
          state.editing = false;
        });
      } else {
        state.editing = false;
      }
    };

    return { nameTextField, state, activeStage, trySaveName };
  },
};
</script>
