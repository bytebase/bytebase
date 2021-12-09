<template>
  <div class="border-b border-block-border">
    <nav class="-mb-px flex" aria-label="Tabs">
      <a
        v-for="(item, index) in tabItemList"
        :id="item.id"
        :key="index"
        :href="`#${item.id}`"
        class="
          select-none
          cursor-pointer
          flex
          justify-between
          py-2
          px-1
          font-medium
          border-b-2 border-transparent
          whitespace-nowrap
        "
        :class="tabClass(index == selectedIndex)"
        @click.self="selectTabIndex(index)"
        @mouseenter="state.hoverIndex = index"
        @mouseleave="state.hoverIndex = -1"
      >
        <button
          v-if="
            index != 0 &&
            (reorderModel == 'ALWAYS' ||
              (reorderModel == 'HOVER' && state.hoverIndex == index))
          "
          type="button"
          class="
            text-control
            hover:text-control-hover
            focus:outline-none
            focus-visible:ring-2
            focus:ring-accent
          "
          @click.prevent="
            () => {
              selectTabIndex(index);
              $emit('reorder-index', index, index - 1);
            }
          "
        >
          <heroicons-solid:arrow-circle-left class="w-6 h-6" />
        </button>
        <div v-else class="pl-6"></div>
        {{ item.title }}
        <button
          v-if="
            index != tabItemList.length - 1 &&
            (reorderModel == 'ALWAYS' ||
              (reorderModel == 'HOVER' && state.hoverIndex == index))
          "
          type="button"
          class="
            text-control
            hover:text-control-hover
            focus:outline-none
            focus-visible:ring-2
            focus:ring-accent
          "
          @click.prevent="
            () => {
              selectTabIndex(index);
              $emit('reorder-index', index, index + 1);
            }
          "
        >
          <heroicons-solid:arrow-circle-right class="w-6 h-6" />
        </button>
        <div v-else class="pr-6"></div>
      </a>
      <button
        v-if="allowCreate"
        type="button"
        class="
          flex
          justify-center
          py-2
          text-control
          hover:text-control-hover
          focus:outline-none
          focus-visible:ring-2
          focus:ring-accent
        "
        :class="addTabClass()"
        @click.prevent="$emit('create')"
      >
        <heroicons-solid:plus class="w-6 h-6" />
      </button>
    </nav>
  </div>
  <slot />
</template>

<script lang="ts">
import { reactive, PropType } from "vue";
import { BBTabItem } from "./types";

export default {
  name: "BBTab",
  props: {
    tabItemList: {
      required: true,
      type: Object as PropType<BBTabItem[]>,
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
  emits: ["create", "reorder-index", "select-index"],
  setup(props, { emit }) {
    const state = reactive({
      hoverIndex: -1,
    });

    const tabClass = (selected: boolean) => {
      const width =
        "w-1/" + (props.tabItemList.length + (props.allowCreate ? 1 : 0));
      if (selected) {
        return width + " text-control-hover border-accent";
      }
      return (
        width +
        " text-control hover:text-control-hover hover:border-control-border"
      );
    };

    const addTabClass = () => {
      if (props.tabItemList.length == 0) {
        return "w-1/6 ";
      }
      return "w-1/12";
    };

    const selectTabIndex = (index: number) => {
      emit("select-index", index);
    };

    return {
      tabClass,
      addTabClass,
      selectTabIndex,
      state,
    };
  },
  data: function () {
    return {};
  },
};
</script>
