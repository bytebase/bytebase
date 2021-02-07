<template>
  <div
    class="md:flex md:items-center md:justify-end md:space-x-4 lg:border-b lg:pb-6"
  >
    <div></div>
    <div class="mt-4 flex space-x-3 md:mt-0">
      <button
        type="button"
        class="inline-flex justify-center px-4 py-2 border border-gray-300 shadow-sm text-sm font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-gray-900"
      >
        <!-- Heroicon name: solid/pencil -->
        <svg
          class="-ml-1 mr-2 h-5 w-5 text-gray-400"
          xmlns="http://www.w3.org/2000/svg"
          viewBox="0 0 20 20"
          fill="currentColor"
          aria-hidden="true"
        >
          <path
            d="M13.586 3.586a2 2 0 112.828 2.828l-.793.793-2.828-2.828.793-.793zM11.379 5.793L3 14.172V17h2.828l8.38-8.379-2.83-2.828z"
          />
        </svg>
        <span>Edit</span>
      </button>
      <button
        type="button"
        class="inline-flex justify-center px-4 py-2 border border-gray-300 shadow-sm text-sm font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-gray-900"
      >
        <!-- Heroicon name: solid/bell -->
        <svg
          class="-ml-1 mr-2 h-5 w-5 text-gray-400"
          xmlns="http://www.w3.org/2000/svg"
          viewBox="0 0 20 20"
          fill="currentColor"
          aria-hidden="true"
        >
          <path
            d="M10 2a6 6 0 00-6 6v3.586l-.707.707A1 1 0 004 14h12a1 1 0 00.707-1.707L16 11.586V8a6 6 0 00-6-6zM10 18a3 3 0 01-3-3h6a3 3 0 01-3 3z"
          />
        </svg>
        <span>Subscribe</span>
      </button>
    </div>
  </div>
</template>

<script lang="ts">
import { PropType } from "vue";
import { Task } from "../types";

export default {
  name: "TaskToolbar",
  props: {
    task: {
      required: true,
      type: Object as PropType<Task>,
    },
  },
  components: {},
  setup(props, ctx) {},
};
</script>
