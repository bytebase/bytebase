<template>
  <div>
    <div :class="responsive ? 'sm:hidden' : 'hidden'">
      <label for="tabs" class="sr-only">Select a tab</label>
      <select
        id="tabs"
        name="tabs"
        class="control block w-full"
        @change="
          (e: any) => {
            $emit('select-index', parseInt(e.target.value, 10));
          }
        "
      >
        <option
          v-for="(item, index) in tabItemList"
          :key="index"
          :value="index"
          :selected="index == selectedIndex"
        >
          {{ item.title }}
        </option>
      </select>
    </div>
    <div :class="responsive ? 'hidden sm:block' : 'block'">
      <div
        class="flex py-1 space-x-4 w-full overflow-x-auto hide-scrollbar"
        aria-label="Tabs"
      >
        <button
          v-for="(item, index) in tabItemList"
          :key="index"
          class="tab px-3 py-1 flex items-center"
          :class="buttonClass(index == selectedIndex)"
          @click.prevent="$emit('select-index', index)"
        >
          {{ item.title }}
          <span
            v-if="item.alert"
            class="flex items-center justify-center rounded-full select-none ml-2 w-4 h-4 text-white"
            :class="index == selectedIndex ? 'bg-gray-600' : 'bg-red-600'"
          >
            <span
              class="h-2 w-2 rounded-full text-center pb-6 font-normal text-base"
              aria-hidden="true"
              >!</span
            >
          </span>
        </button>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { withDefaults } from "vue";
import { BBTabFilterItem } from "./types";

withDefaults(
  defineProps<{
    tabItemList: BBTabFilterItem[];
    selectedIndex: number;
    responsive?: boolean;
  }>(),
  {
    responsive: true,
  }
);

defineEmits<{
  (event: "select-index", index: number): void;
}>();

const buttonClass = (selected: boolean) => {
  if (selected) {
    return "bg-gray-200 text-gray-800 whitespace-nowrap";
  }
  return "text-gray-500 hover:text-gray-700 whitespace-nowrap";
};
</script>
