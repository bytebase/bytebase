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
            <div v-if="!state.new">
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
        <template v-if="state.new">
          <button type="button" class="btn-primary px-4 py-2">Create</button>
        </template>
        <template v-else>
          <button type="button" class="btn-normal px-4 py-2">Close</button>
          <button type="button" class="btn-primary px-4 py-2">Resolve</button>
        </template>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import { PropType, reactive } from "vue";
import isEmpty from "lodash-es/isEmpty";
import { Task } from "../types";

interface LocalState {
  new: boolean;
}

export default {
  name: "TaskHighlightPanel",
  props: {
    task: {
      required: true,
      type: Object as PropType<Task>,
    },
  },
  components: {},
  setup(props, ctx) {
    const state = reactive<LocalState>({
      new: isEmpty(props.task.id),
    });

    return { state };
  },
};
</script>
