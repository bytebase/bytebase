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
      <div class="flex space-x-4" aria-label="Tabs">
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
            class="
              flex
              items-center
              justify-center
              rounded-full
              select-none
              ml-2
              w-4
              h-4
              text-white
            "
            :class="index == selectedIndex ? 'bg-gray-600' : 'bg-red-600'"
          >
            <span
              class="
                h-2
                w-2
                rounded-full
                text-center
                pb-6
                font-normal
                text-base
              "
              aria-hidden="true"
              >!</span
            >
          </span>
        </button>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import { PropType } from "vue";
import { BBTabFilterItem } from "./types";

export default {
  name: "BBTabFilter",
  props: {
    tabItemList: {
      required: true,
      type: Object as PropType<BBTabFilterItem[]>,
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
  emits: ["select-index"],
  setup() {
    const buttonClass = (selected: boolean) => {
      if (selected) {
        return "bg-gray-200 text-gray-800 whitespace-nowrap";
      }
      return "text-gray-500 hover:text-gray-700 whitespace-nowrap";
    };

    return { buttonClass };
  },
};
</script>
