<template>
  <VBinder>
    <VTarget>
      <div
        ref="button"
        class="btn-select relative pl-3 pr-10"
        :class="[
          disabled && 'bg-control-bg opacity-50 !cursor-not-allowed',
          state.open && 'ring-1 ring-control border-control',
        ]"
        v-bind="$attrs"
        @click="toggle"
      >
        <div class="relative h-9 py-1.5">
          <template v-if="state.value && !state.open">
            <slot name="menuItem" :item="state.value" />
          </template>
          <template v-if="!state.value && !state.query">
            <slot name="placeholder" :placeholder="placeholder">
              {{ placeholder }}
            </slot>
          </template>
          <input
            v-if="!disabled"
            v-model="state.query"
            type="text"
            class="absolute w-full h-full inset-0 p-0 border-none focus:border-transparent focus:ring-0"
            :class="{
              'opacity-0': !state.open,
            }"
            @compositionstart="state.compositing = true"
            @compositionend="state.compositing = false"
            @keydown="handleKeyEvent"
          />
        </div>
        <span
          class="ml-3 absolute inset-y-0 right-0 flex items-center pr-2 pointer-events-none"
        >
          <heroicons-solid:selector class="h-5 w-5 text-control-light" />
        </span>
      </div>
    </VTarget>
    <VFollower :show="state.open" placement="bottom-start">
      <transition :appear="true" name="fade-fast">
        <div
          v-if="state.open"
          ref="popup"
          class="z-50 rounded-md bg-white shadow-lg mt-0.5"
          v-bind="dataLabelAttrs"
          :style="{ width: `${width}px` }"
        >
          <ul
            tabindex="-1"
            role="listbox"
            aria-labelledby="listbox-label"
            aria-activedescendant="listbox-item-3"
            class="max-h-56 rounded-md py-1 ring-1 ring-black ring-opacity-5 overflow-auto focus:outline-none sm:text-sm"
          >
            <!--
              Select option, manage highlight styles based on mouseenter/mouseleave and keyboard navigation.
              Highlighted: "text-white bg-indigo-600", Not Highlighted: "text-gray-900"
            -->
            <li
              v-for="(item, index) in state.filteredOptions"
              :key="index"
              :ref="(el: any) => (itemRefs[index] = el)"
              role="option"
              class="text-main cursor-default select-none relative py-2 pl-3 pr-9"
              :class="
                index === state.activeIndex && 'text-main-text bg-main-hover'
              "
              @click="doSelect(item)"
              @mouseover="state.activeIndex = index"
            >
              <slot name="menuItem" :item="item" />
              <span
                v-if="item === state.value"
                class="absolute inset-y-0 right-0 flex items-center pr-4"
              >
                <heroicons-solid:check class="h-5 w-5" />
              </span>
            </li>
          </ul>
        </div>
      </transition>
    </VFollower>
  </VBinder>
</template>

<script lang="ts" setup>
import { reactive, watch, ref, withDefaults } from "vue";
import { VBinder, VTarget, VFollower } from "vueuc";
import { onClickOutside, useElementBounding } from "@vueuse/core";
import useDataLabelAttrs from "@/composables/useDataLabelAttrs";
import { isAncestorOf, scrollIntoViewIfNeeded } from "./BBUtil";

export type ItemType = number | string | object | any;

type LocalState = {
  open: boolean;
  value: ItemType | undefined;
  query: string;
  compositing: boolean;
  filteredOptions: ItemType[];
  activeIndex: number;
  activeItem: ItemType | undefined;
};

const props = withDefaults(
  defineProps<{
    value?: ItemType | undefined;
    options: ItemType[];
    placeholder?: string;
    disabled?: boolean;
    filter?: (
      options: ItemType[],
      query: string
    ) => ItemType[] | Promise<ItemType[]>;
  }>(),
  {
    value: undefined,
    placeholder: "",
    disabled: false,
    filter: async (options) => options,
  }
);

const emit = defineEmits<{
  (event: "update:value", value: ItemType): void;
}>();

const state = reactive<LocalState>({
  open: false,
  value: props.value,
  query: "",
  compositing: false,
  filteredOptions: props.options,
  activeIndex: props.options.indexOf(props.value),
  activeItem: props.value,
});

const dataLabelAttrs = useDataLabelAttrs("", "-item");

const button = ref<HTMLButtonElement | null>(null);
const popup = ref<HTMLElement | null>(null);
const itemRefs = ref<HTMLElement[]>([]);

// Sync the popup's width to the button's
const { width } = useElementBounding(button);

watch(
  () => props.value,
  (cur) => {
    state.value = cur;
    state.activeIndex = props.options.indexOf(cur);
    state.activeItem = cur;
  }
);

const close = () => {
  state.open = false;
  state.compositing = false;
};

const open = () => {
  state.open = true;
  state.query = "";
  state.compositing = false;
};

const toggle = () => {
  if (props.disabled) {
    return;
  }

  if (!state.open) open();
  else close();
};

onClickOutside(popup, (e) => {
  if (button.value) {
    if (isAncestorOf(button.value, e.target as HTMLElement)) {
      // Do not close the popup when clicking on the button
      return;
    }
  }
  // Otherwise close the popup
  close();
});

watch(
  () => [state.query, props.options, state.compositing],
  (args) => {
    const [query, options, compositing] = args as [string, ItemType[], boolean];
    if (compositing) {
      // Will do nothing if the IME is compositing
      return;
    }
    if (!query) {
      // Won't call filter function if query is empty
      state.filteredOptions = options;
    } else {
      const result = props.filter(options, query);
      if (Array.isArray(result)) {
        // If filter function returns an Array, set value directly
        state.filteredOptions = result;
      } else {
        // Otherwise it's a Promise<Array>, wait.
        result.then((filteredOptions) => {
          if (query === state.query) {
            // Only to set value when query matches
            // In case that the async filter function does not response as quickly as query changes.
            state.filteredOptions = filteredOptions;
          }
        });
      }
    }
  }
);

watch(
  () => state.filteredOptions,
  (newList) => {
    // Try to keep activeIndex correct when filteredOptions changed
    // Find the new index of the selected item
    if (state.activeItem) {
      const newIndex = newList.indexOf(state.activeItem);
      state.activeIndex = newIndex; // -1 or some index
      state.activeItem = newList[newIndex]; // undefined or some item
    }
  }
);

const doSelect = (item: ItemType) => {
  if (item !== state.value) {
    emit("update:value", item);
  }
  close();
};

const moveIndex = (diff: number) => {
  const max = state.filteredOptions.length;
  const index = state.activeIndex + diff;
  if (index < 0 || index >= max) return;

  // Change activeIndex if it doesn't exceed the boundary
  state.activeIndex = index;
  state.activeItem = state.filteredOptions[index];

  const elem = itemRefs.value[index];
  if (elem) {
    scrollIntoViewIfNeeded(elem);
  }
};

const handleKeyEvent = (e: KeyboardEvent) => {
  if (e.key === "ArrowUp") {
    e.preventDefault();
    moveIndex(-1);
  }
  if (e.key === "ArrowDown") {
    e.preventDefault();
    moveIndex(1);
  }
  if (e.key === "Enter") {
    e.preventDefault();
    const index = state.activeIndex;
    const list = state.filteredOptions;
    if (index >= 0 && index < list.length) {
      doSelect(list[index]);
    }
  }
};

watch(
  () => props.disabled,
  (disabled) => {
    if (!disabled) close();
  }
);
</script>

<script lang="ts">
export default {
  inheritAttrs: false,
};
</script>
