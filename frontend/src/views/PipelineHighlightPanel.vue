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
            <div>
              <p class="mt-2 text-sm text-gray-500">
                #{{ pipeline.id }} opened by
                <span href="#" class="font-medium text-normal-text">{{
                  pipeline.attributes.creator.name
                }}</span>
                at
                <span href="#" class="font-medium text-normal-text">{{
                  moment(pipeline.attributes.lastUpdatedTs).format("LLL")
                }}</span>
              </p>
            </div>
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
