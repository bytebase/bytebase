<template>
  <div
    @click.prevent="toggleExpand"
    class="outline-title flex text-xs px-2 py-2"
    :class="allowCollapse ? 'collapsible' : ''"
    @mouseenter="state.hoverTitle = true"
    @mouseleave="state.hoverTitle = false"
  >
    {{ title }}
    <template v-if="allowCollapse && state.hoverTitle">
      <svg
        v-if="expandState"
        class="mr-2 h-4 w-4 transform rotate-90 group-hover:text-gray-400 group-focus:text-gray-400 transition-colors ease-in-out duration-150"
        viewBox="0 0 20 20"
      >
        <path d="M6 6L14 10L6 14V6Z" fill="currentColor" />
      </svg>
      <svg
        v-else
        class="mr-2 h-4 w-4 transform group-hover:text-gray-400 group-focus:text-gray-400 transition-colors ease-in-out duration-150"
        viewBox="0 0 20 20"
      >
        <path d="M6 6L14 10L6 14V6Z" fill="currentColor" />
      </svg>
    </template>
  </div>
  <div v-if="expandState" class="space-y-1">
    <div
      role="group"
      v-for="(item, index) in itemList"
      :key="item.id"
      @mouseenter="state.hoverIndex = index"
      @mouseleave="state.hoverIndex = -1"
    >
      <div
        class="outline-item flex justify-between px-3 py-1"
        @click.prevent="$emit('click-index', index)"
      >
        <span class="truncate">{{ item }}</span>
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
      </div>
    </div>
  </div>
</template>

<script lang="ts">
interface LocalState {
  hoverIndex: number;
  hoverTitle: boolean;
  expandState: boolean;
}

import { computed, reactive, PropType } from "vue";
import { useStore } from "vuex";

export default {
  name: "BBOutline",
  emits: ["click-index", "delete-index"],
  props: {
    // Used for storing the expand state.
    // Empty id means not to store expand state.
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
      type: Object as PropType<any[]>,
    },
    allowDelete: {
      default: false,
      type: Boolean,
    },
    allowCollapse: {
      default: false,
      type: Boolean,
    },
  },
  setup(props, ctx) {
    const store = useStore();

    const state = reactive<LocalState>({
      hoverIndex: -1,
      hoverTitle: false,
      expandState: true,
    });

    const expandState = computed(() => {
      if (props.id) {
        return store.getters["uistate/expandStateByKey"](props.id);
      }
      return state.expandState;
    });

    const toggleExpand = () => {
      if (props.allowCollapse) {
        if (props.id) {
          const newState = !expandState.value;
          store
            .dispatch("uistate/saveExpandStateByKey", {
              key: props.id,
              expand: newState,
            })
            .catch((error) => {
              console.log(error);
              return;
            });
        } else {
          state.expandState = !state.expandState;
        }
      }
    };

    return {
      state,
      expandState,
      toggleExpand,
    };
  },
};
</script>
