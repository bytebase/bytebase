<template>
  <aside>
    <h2 class="sr-only">Details</h2>
    <div class="space-y-3">
      <div
        v-if="!state.new"
        class="flex flex-row space-x-2 lg:flex-col lg:space-x-0"
      >
        <h2 class="flex items-center textlabel w-1/4 lg:w-auto">Status</h2>
        <TaskStatusSelect
          class="lg:mt-3 w-3/4 lg:w-auto"
          :selectedStatus="task.attributes.status"
          @change-status="
            (value) => {
              $emit('update-builtin-field', 'status', value);
            }
          "
        />
      </div>
      <div class="flex flex-row space-x-2 lg:flex-col lg:space-x-0">
        <h2 class="flex items-center textlabel w-1/4 lg:w-auto">Assignee</h2>
        <ul class="lg:mt-3 w-3/4 lg:w-auto">
          <li class="flex justify-start items-center space-x-2">
            <div v-if="task.attributes.assignee" class="flex-shrink-0">
              <BBAvatar
                :size="'small'"
                :username="task.attributes.assignee.name"
              />
            </div>
            <div class="text-sm font-medium text-main">
              {{
                task.attributes.assignee
                  ? task.attributes.assignee.name
                  : "Unassigned"
              }}
            </div>
          </li>
        </ul>
      </div>
      <div class="flex flex-row space-x-2 lg:flex-col lg:space-x-0">
        <h2 class="flex items-center textlabel w-1/4 lg:w-auto">Reporter</h2>
        <ul class="lg:mt-3 w-3/4 lg:w-auto">
          <li class="flex justify-start items-center space-x-2">
            <div class="flex-shrink-0">
              <BBAvatar
                :size="'small'"
                :username="task.attributes.creator.name"
              />
            </div>
            <div class="text-sm font-medium text-main">
              {{ task.attributes.creator.name }}
            </div>
          </li>
        </ul>
      </div>
      <template v-for="(field, index) in fieldList" :key="index">
        <div class="flex flex-row space-x-2 lg:flex-col lg:space-x-0">
          <template v-if="field.type == 'String'">
            <h2 class="flex items-center textlabel w-1/4 lg:w-auto">
              {{ field.name }}
              <span v-if="field.required" class="text-red-600">*</span>
            </h2>
            <div class="lg:mt-3 w-3/4 lg:w-auto">
              <input
                type="text"
                class="textfield w-full"
                :name="field.id"
                :value="
                  field.preprocessor
                    ? field.preprocessor(task.attributes.payload[field.id])
                    : task.attributes.payload[field.id]
                "
                :placeholder="field.placeholder"
                @input="
                  $emit('update-custom-field', field, $event.target.value)
                "
              />
            </div>
          </template>
        </div>
      </template>
    </div>
    <div
      v-if="!state.new"
      class="mt-6 border-t border-block-border py-6 space-y-4"
    >
      <div class="flex items-center space-x-2">
        <!-- Heroicon name: solid/chat-alt -->
        <svg
          class="h-5 w-5 text-control-light"
          xmlns="http://www.w3.org/2000/svg"
          viewBox="0 0 20 20"
          fill="currentColor"
          aria-hidden="true"
        >
          <path
            fill-rule="evenodd"
            d="M18 5v8a2 2 0 01-2 2h-5l-5 4v-4H4a2 2 0 01-2-2V5a2 2 0 012-2h12a2 2 0 012 2zM7 8H5v2h2V8zm2 0h2v2H9V8zm6 0h-2v2h2V8z"
            clip-rule="evenodd"
          />
        </svg>
        <span class="textfield">4 comments</span>
      </div>
      <div>
        <h2 class="textlabel">Update Time</h2>
        <span class="textfield">
          {{ moment(task.attributes.lastUpdatedTs).format("LLL") }}</span
        >
      </div>
      <div>
        <h2 class="textlabel">Creation Time</h2>
        <span class="textfield">
          {{ moment(task.attributes.createdTs).format("LLL") }}</span
        >
      </div>
    </div>
  </aside>
</template>

<script lang="ts">
import { PropType, watchEffect, reactive } from "vue";
import isEmpty from "lodash-es/isEmpty";
import TaskStatusSelect from "../components/TaskStatusSelect.vue";
import { TaskField } from "../plugins";
import { Task } from "../types";

interface LocalState {
  new: boolean;
}

export default {
  name: "TaskSidebar",
  emits: ["update-builtin-field", "update-custom-field"],
  props: {
    task: {
      required: true,
      type: Object as PropType<Task>,
    },
    fieldList: {
      required: true,
      type: Object as PropType<TaskField[]>,
    },
  },
  components: { TaskStatusSelect },
  setup(props, { emit }) {
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
