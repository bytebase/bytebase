<template>
  <!-- Secondary navigation -->
  <h3
    class="px-3 text-xs leading-4 font-semibold text-control-light uppercase tracking-wider"
  >
    {{ title }}
  </h3>
  <div
    class="mt-2 space-y-1"
    role="group"
    v-for="(item, index) in itemList"
    :key="item.id"
    @mouseenter="state.hoverIndex = index"
    @mouseleave="state.hoverIndex = -1"
  >
    <div
      class="outline-item px-3 py-1"
      @click.prevent="$emit('click-item', item)"
    >
      <span class="truncate">{{ item.name }}</span>
      <button
        v-if="allowDelete && index == state.hoverIndex"
        class="focus:outline-none"
        @click.prevent="$emit('delete-item', item)"
      >
        <svg
          class="w-4 h-4 hover:text-control-hover"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
          xmlns="http://www.w3.org/2000/svg"
        >
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M6 18L18 6M6 6l12 12"
          ></path>
        </svg>
      </button>
    </div>
  </div>
</template>

<script lang="ts">
interface LocalState {
  hoverIndex: number;
}

import { reactive, PropType } from "vue";

export default {
  name: "BBOutline",
  emits: ["click-item", "delete-item"],
  props: {
    title: {
      required: true,
      type: String,
    },
    itemList: {
      required: true,
      type: Object as PropType<any[]>,
    },
    allowDelete: {
      default: false,
      type: Boolean,
    },
  },
  setup(props, ctx) {
    const state = reactive<LocalState>({
      hoverIndex: -1,
    });

    return {
      state,
    };
  },
};
</script>
