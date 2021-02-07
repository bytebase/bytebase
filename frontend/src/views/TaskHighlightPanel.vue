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
                {{ task.attributes.name }}
              </p>
            </div>
            <div>
              <p class="mt-2 text-sm text-gray-500">
                #{{ task.id }} opened by
                <span href="#" class="font-medium text-control">{{
                  task.attributes.creator.name
                }}</span>
                at
                <span href="#" class="font-medium text-control">{{
                  moment(task.attributes.lastUpdatedTs).format("LLL")
                }}</span>
              </p>
            </div>
          </div>
        </div>
      </div>
      <div class="mt-6 flex space-x-3 md:mt-0 md:ml-4">
        <button type="button" class="btn-normal px-4 py-2">Add money</button>
        <button type="button" class="btn-primary px-4 py-2">Send money</button>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import { PropType } from "vue";
import { Task } from "../types";

export default {
  name: "TaskHighlightPanel",
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
