<template>
  <div class="border-b border-block-border">
    <nav class="-mb-px flex" aria-label="Tabs">
      <div
        v-for="(title, index) in tabTitleList"
        :key="index"
        class="cursor-pointer flex justify-between py-2 px-1 font-medium border-b-2 border-transparent"
        v-bind:class="tabClass(index == selectedIndex)"
        @click.self="$emit('select-index', index)"
        @mouseenter="state.hoverIndex = index"
        @mouseleave="state.hoverIndex = -1"
      >
        <button
          v-if="
            index != 0 &&
            (reorderModel == 'ALWAYS' ||
              (reorderModel == 'HOVER' && state.hoverIndex == index))
          "
          @click.prevent="
            () => {
              $emit('select-index', index);
              $emit('reorder-index', index, index - 1);
            }
          "
          type="button"
          class="text-gray-500 hover:text-gray-900 focus:outline-none"
        >
          <svg
            class="w-6 h-6"
            fill="currentColor"
            viewBox="0 0 20 20"
            xmlns="http://www.w3.org/2000/svg"
          >
            <path
              fill-rule="evenodd"
              d="M10 18a8 8 0 100-16 8 8 0 000 16zm.707-10.293a1 1 0 00-1.414-1.414l-3 3a1 1 0 000 1.414l3 3a1 1 0 001.414-1.414L9.414 11H13a1 1 0 100-2H9.414l1.293-1.293z"
              clip-rule="evenodd"
            ></path>
          </svg>
        </button>
        <div v-else class="pl-6"></div>
        {{ title }}
        <button
          v-if="
            index != tabTitleList.length - 1 &&
            (reorderModel == 'ALWAYS' ||
              (reorderModel == 'HOVER' && state.hoverIndex == index))
          "
          @click.prevent="
            () => {
              $emit('select-index', index);
              $emit('reorder-index', index, index + 1);
            }
          "
          type="button"
          class="text-gray-500 hover:text-gray-900 focus:outline-none"
        >
          <svg
            class="w-6 h-6"
            fill="currentColor"
            viewBox="0 0 20 20"
            xmlns="http://www.w3.org/2000/svg"
          >
            <path
              fill-rule="evenodd"
              d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-8.707l-3-3a1 1 0 00-1.414 1.414L10.586 9H7a1 1 0 100 2h3.586l-1.293 1.293a1 1 0 101.414 1.414l3-3a1 1 0 000-1.414z"
              clip-rule="evenodd"
            ></path>
          </svg>
        </button>
        <div v-else class="pr-6"></div>
      </div>
      <button
        v-if="allowCreate"
        @click.prevent="$emit('create')"
        type="button"
        class="flex justify-center py-2 text-gray-500 hover:text-gray-900 focus:outline-none focus-visible:ring-2 focus:ring-accent"
        v-bind:class="addTabClass()"
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
            d="M12 6v6m0 0v6m0-6h6m-6 0H6"
          ></path>
        </svg>
      </button>
    </nav>
  </div>
  <slot />
</template>

<script lang="ts">
import { reactive, PropType } from "vue";

export default {
  name: "BBTab",
  emits: ["create", "reorder-index", "select-index"],
  props: {
    tabTitleList: {
      required: true,
      type: Object as PropType<String[]>,
    },
    selectedIndex: {
      required: true,
      type: Number,
    },
    allowCreate: {
      default: false,
      type: Boolean,
    },
    reorderModel: {
      default: "NEVER",
      type: String as PropType<"NEVER" | "HOVER" | "ALWAYS">,
    },
  },
  data: function () {
    return {};
  },
  setup(props, ctx) {
    const state = reactive({
      hoverIndex: -1,
    });

    const tabClass = (selected: boolean) => {
      const width =
        "w-1/" + (props.tabTitleList.length + (props.allowCreate ? 1 : 0));
      if (selected) {
        return width + " text-gray-900 border-gray-500";
      }
      return (
        width + " text-gray-500 hover:text-gray-900 hover:border-control-border"
      );
    };

    const addTabClass = () => {
      if (props.tabTitleList.length == 0) {
        return "w-1/6 ";
      }
      return "w-1/12";
    };

    return {
      tabClass,
      addTabClass,
      state,
    };
  },
};
</script>
