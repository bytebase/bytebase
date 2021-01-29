<template>
  <div class="sm:hidden">
    <label for="tabs" class="sr-only">Select a tab</label>
    <select
      id="tabs"
      name="tabs"
      class="block w-full focus:ring-accent focus:border-accent border-gray-300 rounded-md"
      @change="
        (e) => {
          $emit('select-index', parseInt(e.target.value));
        }
      "
    >
      <option
        v-for="(item, index) in itemList"
        :key="index"
        :value="index"
        :selected="index == selectedIndex"
      >
        {{ item.title }}
      </option>
    </select>
  </div>
  <div class="hidden sm:block">
    <div class="flex space-x-4" aria-label="Tabs">
      <button
        v-for="(item, index) in itemList"
        :key="index"
        class="focus:outline-none px-3 py-1 font-medium text-sm rounded-md"
        :class="buttonClass(index == selectedIndex)"
        @click.prevent="$emit('select-index', index)"
      >
        {{ item.title }}
      </button>
    </div>
  </div>
</template>

<script lang="ts">
import { computed, reactive } from "vue";

export default {
  name: "BBTableTabFilter",
  emits: ["select-index"],
  props: {
    itemList: {
      required: true,
      type: Array,
    },
    selectedIndex: {
      required: true,
      type: Number,
    },
  },
  setup(props, ctx) {
    const buttonClass = (selected: boolean) => {
      if (selected) {
        return "bg-gray-200 text-gray-800";
      }
      return "text-gray-500 hover:text-gray-700";
    };

    return { buttonClass };
  },
};
</script>
