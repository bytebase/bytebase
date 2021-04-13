<template>
  <div>
    <div :class="responsive ? 'sm:hidden' : 'hidden'">
      <label for="tabs" class="sr-only">Select a tab</label>
      <select
        id="tabs"
        name="tabs"
        class="control block w-full"
        @change="
          (e) => {
            $emit('select-index', parseInt(e.target.value));
          }
        "
      >
        <option
          v-for="(item, index) in tabList"
          :key="index"
          :value="index"
          :selected="index == selectedIndex"
        >
          {{ item }}
        </option>
      </select>
    </div>
    <div :class="responsive ? 'hidden sm:block' : 'block'">
      <div class="flex space-x-4" aria-label="Tabs">
        <button
          v-for="(item, index) in tabList"
          :key="index"
          class="tab px-3 py-1"
          :class="buttonClass(index == selectedIndex)"
          @click.prevent="$emit('select-index', index)"
        >
          {{ item }}
        </button>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import { PropType } from "vue";

export default {
  name: "BBTableTabFilter",
  emits: ["select-index"],
  props: {
    tabList: {
      required: true,
      type: Object as PropType<String[]>,
    },
    selectedIndex: {
      required: true,
      type: Number,
    },
    responsive: {
      default: true,
      type: Boolean,
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
