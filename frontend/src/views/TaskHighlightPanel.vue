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
            v-if="state.editing"
            required
            ref="nameTextField"
            id="name"
            name="name"
            type="text"
            class="textfield ml-2 my-0.5 w-full"
            v-model="state.name"
            @blur="trySaveName"
          />
          <!-- Extra padding is to prevent flickering when entering the edit mode -->
          <p
            v-else
            class="ml-2 mt-2 mb-1.5 w-full text-xl font-bold leading-7 text-main whitespace-nowrap truncate"
          >
            <span @click.prevent="clickName">{{ state.name }}</span>
          </p>
        </div>
        <div v-if="!$props.new">
          <p class="mt-2 text-sm text-control-light">
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

    const clickName = () => {
      state.editing = true;
      nextTick(() => {
        nameTextField.value.focus();
      });
    };

    const trySaveName = () => {
      if (state.name != props.task.name) {
        emit("update-name", state.name, (updatedTask: Task) => {
          state.editing = false;
        });
      } else {
        state.editing = false;
      }
    };

    return { nameTextField, state, activeStage, clickName, trySaveName };
  },
};
</script>
