<template>
  <div class="">
    <div
      class="px-4 py-6 md:flex md:items-center md:justify-between lg:border-t lg:border-block-border"
    >
      <div class="flex-1 min-w-0">
        <!-- Profile -->
        <div class="flex items-center">
          <div>
            <div class="flex items-center">
              <!-- [TODO] overflow-ellipsis/clip doesn't seem to be working, so just use nowrap -->
              <p
                class="text-2xl font-bold leading-7 text-gray-900 sm:leading-9 whitespace-nowrap"
              >
                {{ pipeline.attributes.name }}
              </p>
            </div>
            <dl class="mt-6 flex flex-col sm:mt-1 sm:flex-row sm:flex-wrap">
              <dt class="sr-only">Account status</dt>
              <dd
                class="mt-3 flex items-center text-sm text-gray-500 font-medium sm:mr-6 sm:mt-0 capitalize"
              >
                <!-- Heroicon name: solid/check-circle -->
                <svg
                  class="flex-shrink-0 mr-1.5 h-5 w-5 text-green-400"
                  xmlns="http://www.w3.org/2000/svg"
                  viewBox="0 0 20 20"
                  fill="currentColor"
                  aria-hidden="true"
                >
                  <path
                    fill-rule="evenodd"
                    d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z"
                    clip-rule="evenodd"
                  />
                </svg>
                Verified account
              </dd>
            </dl>
          </div>
        </div>
      </div>
      <div class="mt-6 flex space-x-3 md:mt-0 md:ml-4">
        <button
          type="button"
          class="inline-flex items-center px-4 py-2 border border-gray-300 shadow-sm text-sm font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-cyan-500"
        >
          Add money
        </button>
        <button
          type="button"
          class="inline-flex items-center px-4 py-2 border border-transparent shadow-sm text-sm font-medium rounded-md text-accent-text bg-accent hover:bg-accent-hover focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-cyan-500"
        >
          Send money
        </button>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import { PropType } from "vue";
import { Pipeline } from "../types";

export default {
  name: "PipelineHighlightPanel",
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
