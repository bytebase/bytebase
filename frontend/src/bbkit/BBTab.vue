<template>
  <div class="border-b border-block-border">
    <nav class="-mb-px flex" aria-label="Tabs">
      <a
        v-for="(item, index) in tabItemList"
        :id="item.id"
        :key="index"
        :href="`#${item.id}`"
        class="select-none cursor-pointer flex justify-between py-2 px-1 font-medium border-b-2 whitespace-nowrap"
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
          class="text-control hover:text-control-hover focus:outline-none focus-visible:ring-2 focus:ring-accent"
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
        <slot name="item" :item="item" :index="index">
          {{ item.title }}
        </slot>
        <button
          v-if="
            index != tabItemList.length - 1 &&
            (reorderModel == 'ALWAYS' ||
              (reorderModel == 'HOVER' && state.hoverIndex == index))
          "
          type="button"
          class="text-control hover:text-control-hover focus:outline-none focus-visible:ring-2 focus:ring-accent"
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
        class="flex justify-center py-2 text-control hover:text-control-hover focus:outline-none focus-visible:ring-2 focus:ring-accent"
        :class="addTabClass()"
        @click.prevent="$emit('create')"
      >
        <heroicons-solid:plus class="w-6 h-6" />
      </button>
    </nav>
  </div>
  <slot />
</template>

<script lang="ts" setup>
import { reactive, withDefaults } from "vue";
import { BBTabItem } from "./types";

export type ReorderModel = "NEVER" | "HOVER" | "ALWAYS";

const props = withDefaults(
  defineProps<{
    tabItemList: BBTabItem[];
    selectedIndex: number;
    allowCreate?: boolean;
    reorderModel?: ReorderModel;
  }>(),
  {
    allowCreate: false,
    reorderModel: "NEVER",
  }
);

const emit = defineEmits<{
  (event: "create"): void;
  (event: "reorder-index", from: number, to: number): void;
  (event: "select-index", index: number): void;
}>();

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
    " border-transparent text-control hover:text-control-hover hover:border-control-border"
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
</script>
