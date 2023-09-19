<template>
  <div class="bb-label-select" :class="{ disabled }">
    <VBinder>
      <VTarget>
        <div class="select-wrapper" @click="open">
          <template v-if="!empty">
            <slot name="value" :value="state.value">
              <template v-if="Array.isArray(state.value)">
                <ResponsiveTags :tags="state.value as string[]" />
              </template>
              <span v-else :class="{ capitalize }">
                {{ modifier(state.value) }}
              </span>
            </slot>
          </template>
          <template v-else>
            <slot name="placeholder" :placeholder="placeholder">
              <div class="placeholder">
                {{ placeholder }}
              </div>
            </slot>
          </template>
          <slot name="arrow" :open="state.open">
            <span
              v-if="!disabled"
              class="absolute inset-y-0 right-0 flex items-center pr-2 pointer-events-none"
            >
              <heroicons-solid:selector class="h-5 w-5 text-control-light" />
            </span>
          </slot>
        </div>
      </VTarget>
      <VFollower :show="state.open" placement="bottom-start">
        <transition appear name="fade-fast">
          <div
            v-if="state.open"
            ref="popup"
            class="rounded-md bg-white shadow-lg"
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
              <slot
                v-for="(item, index) in options"
                :key="index"
                name="item"
                :item="item"
                :index="index"
                :selected="isSelected(item)"
                :toggle-selection="() => toggleSelection(item)"
              >
                <li
                  role="option"
                  class="flex items-center text-main hover:text-main-text hover:bg-main-hover cursor-default select-none relative py-2 px-3"
                  @click="toggleSelection(item)"
                >
                  <span class="flex-1" :class="{ capitalize }">
                    {{ modifier(item) }}
                  </span>
                  <span
                    class="ml-1"
                    :class="[isSelected(item) ? 'visible' : 'invisible']"
                  >
                    <heroicons-solid:check class="h-5 w-5" />
                  </span>
                </li>
              </slot>
            </ul>
          </div>
        </transition>
      </VFollower>
    </VBinder>
  </div>
</template>

<script lang="ts">
import { onClickOutside } from "@vueuse/core";
import { reactive, PropType, watch, defineComponent, ref, computed } from "vue";
import { VBinder, VTarget, VFollower } from "vueuc";
import ResponsiveTags from "./ResponsiveTags.vue";

export type DataType = string | number;

interface LocalState {
  open: boolean;
  value: DataType | DataType[];
}

export default defineComponent({
  name: "LabelSelect",
  components: {
    VBinder,
    VTarget,
    VFollower,
    ResponsiveTags,
  },
  props: {
    value: {
      type: [String, Number, Array] as PropType<DataType | DataType[]>,
      required: true,
    },
    options: {
      type: Array as PropType<DataType[]>,
      default: () => [],
    },
    placeholder: {
      default: "",
      type: String,
    },
    disabled: {
      default: false,
      type: Boolean,
    },
    popupFitWidth: {
      type: Boolean,
      default: true,
    },
    multiple: {
      type: Boolean,
      default: false,
    },
    modifier: {
      type: Function as PropType<(data: DataType) => string>,
      default: (str: DataType) => str,
    },
    capitalize: {
      type: Boolean,
      default: false,
    },
  },
  emits: ["update:value"],
  setup(props, { emit }) {
    const popup = ref<HTMLElement>();

    const state = reactive<LocalState>({
      open: false,
      value: props.value,
    });

    watch(
      () => props.value,
      (cur) => {
        state.value = cur;
      }
    );

    const open = () => {
      if (props.disabled) {
        return;
      }

      state.open = !state.open;
    };

    const close = () => {
      state.open = false;
    };

    onClickOutside(popup, close);

    const empty = computed(() => {
      if (Array.isArray(state.value)) {
        return state.value.length === 0;
      } else {
        return !state.value;
      }
    });

    const isSelected = (item: DataType) => {
      if (Array.isArray(state.value)) {
        return state.value.includes(item);
      } else {
        return item === state.value;
      }
    };

    const toggleSelection = (item: DataType) => {
      if (Array.isArray(state.value)) {
        if (props.multiple) {
          const index = state.value.indexOf(item);
          if (index >= 0) {
            state.value.splice(index, 1);
          } else {
            state.value.push(item);
          }
        } else {
          state.value = [item];
        }
      } else {
        state.value = item;
        close();
      }
      emit("update:value", state.value);
    };

    return {
      popup,
      state,
      open,
      close,
      empty,
      isSelected,
      toggleSelection,
    };
  },
});
</script>

<style scoped lang="postcss">
.bb-label-select {
  @apply overflow-hidden;
}

.select-wrapper {
  @apply h-8 pl-3 pr-8 py-1.5 overflow-hidden max-w-full;
}
.disabled .select-wrapper {
  @apply pr-3;
}
.placeholder {
  @apply whitespace-nowrap text-control-light;
}
</style>
