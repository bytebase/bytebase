<template>
  <div class="">
    <div
      class="px-2 py-4 md:flex md:flex-col lg:border-t lg:border-b lg:border-block-border"
    >
      <h3 class="px-4 text-lg leading-6 font-medium text-main">Result</h3>

      <div v-if="template" class="my-2 mx-4 space-y-2">
        <template
          v-for="(field, index) in template.outputFieldList"
          :key="index"
        >
          <div class="flex">
            <span
              class="z-0 whitespace-nowrap inline-flex items-center px-3 rounded-l-md border border-l border-r-0 border-control-border bg-gray-50 text-control-light sm:text-sm"
            >
              {{ field.name }}
            </span>
            <input
              type="text"
              :name="field.id"
              class="z-10 flex-1 min-w-0 block w-full px-3 py-2 border border-r border-control-border focus:ring-control focus:border-control sm:text-sm"
            />
            <button
              class="z-0 -ml-px px-2 py-2 border border-gray-300 text-sm font-medium text-control-light bg-gray-50 hover:bg-gray-100 focus:ring-control focus:outline-none focus-visible:ring-2 focus:ring-offset-1"
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
              class="z-0 -ml-px px-2 py-2 border border-gray-300 text-sm font-medium rounded-r-md text-control-light bg-gray-50 hover:bg-gray-100 focus:ring-control focus:outline-none focus-visible:ring-2 focus:ring-offset-1"
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
    </div>
  </div>
</template>

<script lang="ts">
import { PropType, reactive } from "vue";
import { taskTemplateList } from "../plugins";
import { Task } from "../types";

interface LocalState {}

export default {
  name: "TaskOutputPanel",
  emits: ["update-field"],
  props: {
    task: {
      required: true,
      type: Object as PropType<Task>,
    },
  },
  components: {},
  setup(props, ctx) {
    const state = reactive<LocalState>({});

    const template = taskTemplateList.find(
      (template) => template.type == props.task.attributes.type
    );

    return { state, template };
  },
};
</script>
