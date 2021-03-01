<template>
  <aside>
    <h2 class="sr-only">Details</h2>
    <div class="space-y-3">
      <div
        v-if="!state.new"
        class="flex flex-row space-x-2 lg:flex-col lg:space-x-0"
      >
        <h2 class="flex items-center textlabel w-1/4 lg:w-auto">Status</h2>
        <div class="lg:mt-3 w-3/4 lg:w-auto">
          <TaskStatusSelect
            :disabled="activeStageIsRunning(task)"
            :selectedStatus="task.attributes.status"
            @change-status="
              (value) => {
                $emit('update-task-status', value);
              }
            "
          />
        </div>
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
          <h2 class="flex items-center textlabel w-1/4 lg:w-auto">
            {{ field.name }}
            <span v-if="field.required" class="text-red-600">*</span>
          </h2>
          <template v-if="field.type == 'String'">
            <div class="lg:mt-3 w-3/4 lg:w-auto">
              <input
                type="text"
                class="textfield w-full"
                :name="field.id"
                :value="fieldValue(field)"
                :placeholder="field.placeholder"
                @input="
                  $emit('update-custom-field', field, $event.target.value)
                "
              />
            </div>
          </template>
          <template v-else-if="field.type == 'Environment'">
            <div class="lg:mt-3 w-3/4 lg:w-auto">
              <EnvironmentSelect
                :name="field.id"
                :selectedId="fieldValue(field)"
                :selectDefault="false"
                @select-environment-id="
                  (environmentId) => {
                    $emit('update-custom-field', field, environmentId);
                  }
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
import EnvironmentSelect from "../components/EnvironmentSelect.vue";
import TaskStatusSelect from "../components/TaskStatusSelect.vue";
import { TaskField } from "../plugins";
import { Task } from "../types";
import { activeStageIsRunning } from "../utils";

interface LocalState {
  new: boolean;
}

export default {
  name: "TaskSidebar",
  emits: ["update-task-status", "update-custom-field"],
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
  components: { EnvironmentSelect, TaskStatusSelect },
  setup(props, { emit }) {
    const state = reactive<LocalState>({
      new: isEmpty(props.task.id),
    });

    const refreshState = () => {
      state.new = isEmpty(props.task.id);
    };

    const fieldValue = (field: TaskField): string => {
      return field.preprocessor
        ? field.preprocessor(props.task.attributes.payload[field.id])
        : props.task.attributes.payload[field.id];
    };

    watchEffect(refreshState);

    return { state, activeStageIsRunning, fieldValue };
  },
};
</script>
