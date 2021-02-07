<template>
  <aside>
    <h2 class="sr-only">Details</h2>
    <div class="space-y-4">
      <div>
        <h2 class="text-sm font-medium text-gray-500">Assignee</h2>
        <ul class="mt-3 space-y-3">
          <li class="flex justify-start items-center space-x-2">
            <div class="flex-shrink-0">
              <BBAvatar
                :size="'small'"
                :username="pipeline.attributes.assignee.name"
              />
            </div>
            <div class="text-sm font-medium text-gray-900">
              {{ pipeline.attributes.assignee.name }}
            </div>
          </li>
        </ul>
      </div>
      <div>
        <h2 class="text-sm font-medium text-gray-500">Requested DB Name</h2>
        <div class="mt-1">
          <input
            type="text"
            name="db"
            id="db"
            class="shadow-sm focus:ring-indigo-500 focus:border-indigo-500 block w-64 text-sm border-gray-300 rounded-md"
            placeholder="Your DB Name"
          />
        </div>
      </div>
    </div>
    <div class="mt-6 border-t border-gray-200 py-6 space-y-4">
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
          class="h-5 w-5 text-gray-400"
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
        <span class="text-gray-900 text-sm font-medium">4 comments</span>
      </div>
      <div>
        <h2 class="text-sm font-medium text-gray-500">Update Time</h2>
        <span class="text-gray-900 text-sm font-medium">
          {{ moment(pipeline.attributes.lastUpdatedTs).format("LLL") }}</span
        >
      </div>
      <div>
        <h2 class="text-sm font-medium text-gray-500">Creation Time</h2>
        <span class="text-gray-900 text-sm font-medium">
          {{ moment(pipeline.attributes.createdTs).format("LLL") }}</span
        >
      </div>
    </div>
  </aside>
</template>

<script lang="ts">
import { PropType } from "vue";
import { Pipeline } from "../types";

export default {
  name: "PipelineSidebar",
  props: {
    pipeline: {
      required: true,
      type: Object as PropType<Pipeline>,
    },
  },
  components: {},
  setup(props, ctx) {},
};
</script>
