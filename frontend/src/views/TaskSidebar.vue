<template>
  <aside>
    <h2 class="sr-only">Details</h2>
    <div class="space-y-4">
      <div>
        <h2 class="textlabel">Assignee</h2>
        <ul class="mt-3 space-y-3">
          <li class="flex justify-start items-center space-x-2">
            <div class="flex-shrink-0">
              <BBAvatar
                :size="'small'"
                :username="task.attributes.assignee.name"
              />
            </div>
            <div class="text-sm font-medium text-gray-900">
              {{ task.attributes.assignee.name }}
            </div>
          </li>
        </ul>
      </div>
      <template v-if="template">
        <template v-for="(field, index) in template.fieldList" :key="index">
          <template v-if="field.type == 'String'">
            <div>
              <h2 class="textlabel">
                {{ field.name }}
                <span v-if="field.required" class="text-red-600">*</span>
              </h2>
              <div class="mt-3">
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
                  @input="$emit('update-field', field, $event.target.value)"
                />
              </div>
            </div>
          </template>
        </template>
      </template>
    </div>
    <div
      v-if="!state.new"
      class="mt-6 border-t border-block-border py-6 space-y-4"
    >
      <div class="flex items-center space-x-2">
        <!-- Heroicon name: solid/lock-open -->
        <svg
          class="h-5 w-5 text-green-500"
          xmlns="http://www.w3.org/2000/svg"
          viewBox="0 0 20 20"
          fill="currentColor"
          aria-hidden="true"
        >
          <path
            d="M10 2a5 5 0 00-5 5v2a2 2 0 00-2 2v5a2 2 0 002 2h10a2 2 0 002-2v-5a2 2 0 00-2-2H7V7a3 3 0 015.905-.75 1 1 0 001.937-.5A5.002 5.002 0 0010 2z"
          />
        </svg>
        <span class="text-green-700 text-sm font-medium">Open Issue</span>
      </div>
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
import { PropType, reactive } from "vue";
import isEmpty from "lodash-es/isEmpty";
import { taskTemplateList } from "../plugins";
import { Task } from "../types";

interface LocalState {
  new: boolean;
}

export default {
  name: "TaskSidebar",
  emits: ["update-field"],
  props: {
    task: {
      required: true,
      type: Object as PropType<Task>,
    },
  },
  components: {},
  setup(props, { emit }) {
    const state = reactive<LocalState>({
      new: isEmpty(props.task.id),
    });

    const template = taskTemplateList.find(
      (template) => template.type == props.task.attributes.type
    );

    return { state, template };
  },
};
</script>
