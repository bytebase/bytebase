<template>
  <h2 class="px-4 text-lg font-medium">Result</h2>

  <div class="my-2 mx-4 space-y-2">
    <template v-for="(field, index) in fieldList" :key="index">
      <div class="flex">
        <span
          class="whitespace-nowrap inline-flex items-center px-3 rounded-l-md border border-l border-r-0 border-control-border bg-gray-50 text-control-light sm:text-sm"
        >
          {{ field.name }}
          <span v-if="field.required" class="text-red-600">*</span>
        </span>
        <input
          type="text"
          class="flex-1 min-w-0 block w-full px-3 py-2 border border-r border-control-border focus:mr-0.5 focus:ring-control focus:border-control sm:text-sm"
          :name="field.id"
          :value="
            field.preprocessor
              ? field.preprocessor(task.attributes.payload[field.id])
              : task.attributes.payload[field.id]
          "
          @input="$emit('update-custom-field', field, $event.target.value)"
        />
        <!-- Disallow tabbing since the focus ring is partially covered by the text field due to overlaying -->
        <button
          tabindex="-1"
          class="-ml-px px-2 py-2 border border-gray-300 text-sm font-medium text-control-light bg-gray-50 hover:bg-gray-100 focus:ring-control focus:outline-none focus-visible:ring-2 focus:ring-offset-1"
        >
          <svg
            class="w-6 h-6"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
            xmlns="http://www.w3.org/2000/svg"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2"
            ></path>
          </svg>
        </button>
        <button
          tabindex="-1"
          class="-ml-px px-2 py-2 border border-gray-300 text-sm font-medium rounded-r-md text-control-light bg-gray-50 hover:bg-gray-100 focus:ring-control focus:outline-none focus-visible:ring-2 focus:ring-offset-1"
        >
          <svg
            class="w-6 h-6"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
            xmlns="http://www.w3.org/2000/svg"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"
            ></path>
          </svg>
        </button>
      </div>
    </template>
  </div>
</template>

<script lang="ts">
import { PropType, reactive } from "vue";
import { TaskField } from "../plugins";
import { Task } from "../types";

interface LocalState {}

export default {
  name: "TaskOutputPanel",
  emits: ["update-custom-field"],
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
  components: {},
  setup(props, ctx) {
    const state = reactive<LocalState>({});

    return { state };
  },
};
</script>
