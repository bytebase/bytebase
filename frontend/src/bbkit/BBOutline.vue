<template>
  <div
    @click.prevent="toggleCollapse"
    class="outline-title flex py-1"
    :class="allowCollapse ? 'collapsible' : ''"
    @mouseenter="state.hoverTitle = true"
    @mouseleave="state.hoverTitle = false"
  >
    <span :class="titleClass()">{{ title }}</span>
    <template v-if="allowCollapse && state.hoverTitle">
      <svg
        v-if="collapseState"
        class="mr-2 h-4 w-4 transform group-hover:text-gray-400 group-focus:text-gray-400 transition-colors ease-in-out duration-150"
        viewBox="0 0 20 20"
      >
        <path d="M6 6L14 10L6 14V6Z" fill="currentColor" />
      </svg>
      <svg
        v-else
        class="mr-2 h-4 w-4 transform rotate-90 group-hover:text-gray-400 group-focus:text-gray-400 transition-colors ease-in-out duration-150"
        viewBox="0 0 20 20"
      >
        <path d="M6 6L14 10L6 14V6Z" fill="currentColor" />
      </svg>
    </template>
  </div>
  <div v-if="!allowCollapse || !collapseState">
    <div
      role="group"
      v-for="(item, index) in itemList"
      :key="index"
      @mouseenter="state.hoverIndex = index"
      @mouseleave="state.hoverIndex = -1"
    >
      <BBOutline
        v-if="item.childList"
        :id="[id, item.id].join('.')"
        :title="item.name"
        :itemList="item.childList"
        :allowCollapse="item.childCollapse"
        :level="level + 1"
      />
      <router-link
        v-else-if="item.link"
        :to="item.link"
        class="outline-item flex justify-between pr-1 py-1"
        :class="'pl-' + (4 + level * 3)"
      >
        <span class="truncate">{{ item.name }}</span>
        <button
          v-if="allowDelete && index == state.hoverIndex"
          class="focus:outline-none"
          @click.prevent="$emit('delete-index', index)"
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
      </router-link>
      <span v-else class="pl-1 py-1 truncate">{{ item.name }}</span>
    </div>
  </div>
</template>

<script lang="ts">
interface LocalState {
  hoverIndex: number;
  hoverTitle: boolean;
  collapseState: boolean;
}

import { computed, reactive, PropType } from "vue";
import { useStore } from "vuex";
import { BBOutlineItem } from "./types";

export default {
  name: "BBOutline",
  emits: ["delete-index"],
  props: {
    // Used for storing the collapse state.
    // Empty id means not to store collapse state.
    id: {
      default: "",
      type: String,
    },
    title: {
      required: true,
      type: String,
    },
    itemList: {
      required: true,
      type: Object as PropType<BBOutlineItem[]>,
    },
    allowDelete: {
      default: false,
      type: Boolean,
    },
    allowCollapse: {
      default: false,
      type: Boolean,
    },
    level: {
      default: 0,
      type: Number,
    },
  },
  setup(props, ctx) {
    const store = useStore();

    const state = reactive<LocalState>({
      hoverIndex: -1,
      hoverTitle: false,
      collapseState: true,
    });

    const collapseState = computed(() => {
      if (props.id) {
        return store.getters["uistate/collapseStateByKey"](props.id);
      }
      return state.collapseState;
    });

    const toggleCollapse = () => {
      if (props.allowCollapse) {
        if (props.id) {
          const newState = !collapseState.value;
          store
            .dispatch("uistate/savecollapseStateByKey", {
              key: props.id,
              collapse: newState,
            })
            .catch((error) => {
              console.log(error);
            });
        } else {
          state.collapseState = !state.collapseState;
        }
      }
    };

    const titleClass = () => {
      return (
        (props.level > 0 ? "text-main" : "") + " pl-" + 2 * (props.level + 1)
      );
    };

    return {
      state,
      collapseState,
      titleClass,
      toggleCollapse,
    };
  },
};
</script>
