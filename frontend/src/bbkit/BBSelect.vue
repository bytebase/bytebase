<template>
  <VBinder>
    <VTarget>
      <button
        ref="button"
        type="button"
        aria-haspopup="listbox"
        aria-expanded="true"
        aria-labelledby="listbox-label"
        class="btn-select relative w-full pl-3 pr-10 py-2"
        :disabled="disabled"
        v-bind="$attrs"
        @click="toggle"
      >
        <div
          class="whitespace-nowrap hide-scrollbar overflow-x-auto"
          :class="
            error
              ? 'text-error'
              : isSelected
              ? 'text-control'
              : 'text-control-placeholder'
          "
        >
          <template v-if="isSelected">
            <slot
              name="menuItem"
              :item="state.selectedItem"
              :index="itemList.indexOf(state.selectedItem)"
            />
          </template>
          <template v-else>
            <slot name="placeholder" :placeholder="placeholder">
              {{ placeholder }}
            </slot>
          </template>
        </div>
        <span
          class="ml-3 absolute inset-y-0 right-0 flex items-center pr-2 pointer-events-none"
        >
          <heroicons-solid:selector class="h-5 w-5 text-control-light" />
        </span>
      </button>
    </VTarget>
    <VFollower :show="state.showMenu" placement="bottom-start">
      <transition :appear="true" name="fade-fast">
        <div
          v-if="state.showMenu"
          ref="popup"
          class="z-50 rounded-md bg-white shadow-lg mt-0.5"
          :style="popupStyle"
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
            <slot v-if="showPrefixItem" name="prefixItem">
              <li
                class="cursor-default select-none py-2 px-3 text-control-light"
              >
                {{ placeholder }}
              </li>
            </slot>
            <li
              v-for="(item, index) in itemList"
              :key="index"
              role="option"
              class="group text-main cursor-default select-none relative py-2 pl-3 pr-9 hover:bg-gray-200"
              @click="
                if (item !== state.selectedItem) {
                  $emit('select-item', item, () => {
                    state.selectedItem = item;
                  });
                }
                close();
              "
            >
              <div class="whitespace-nowrap hide-scrollbar overflow-x-auto">
                <slot name="menuItem" :item="item" :index="index" />
              </div>
              <span
                v-if="item === state.selectedItem"
                class="absolute inset-y-0 right-0 flex items-center pr-4"
              >
                <!-- Heroicon name: solid/check -->
                <heroicons-solid:check class="h-5 w-5" />
              </span>
            </li>
            <div @click="close()">
              <slot name="suffixItem"></slot>
            </div>
          </ul>
        </div>
      </transition>
    </VFollower>
  </VBinder>
</template>

<script lang="ts">
import {
  reactive,
  PropType,
  watch,
  defineComponent,
  ref,
  CSSProperties,
  computed,
} from "vue";
import { VBinder, VTarget, VFollower } from "vueuc";
import { onClickOutside, useElementBounding } from "@vueuse/core";

interface LocalState {
  showMenu: boolean;
  selectedItem: any;
}

type ItemType = number | string | any;

type FitWidthMode = "fit" | "min";

export default defineComponent({
  name: "BBSelect",
  components: {
    VBinder,
    VTarget,
    VFollower,
  },
  inheritAttrs: false,
  props: {
    selectedItem: {
      type: [Object, Number, String] as PropType<ItemType>,
      default: undefined,
    },
    itemList: {
      required: true,
      type: Array as PropType<ItemType[]>,
    },
    placeholder: {
      default: "",
      type: String,
    },
    disabled: {
      default: false,
      type: Boolean,
    },
    showPrefixItem: {
      default: false,
      type: Boolean,
    },
    fitWidth: {
      type: String as PropType<FitWidthMode>,
      default: "fit",
    },
    error: {
      type: Boolean,
      default: false,
    },
  },
  emits: ["select-item"],
  setup(props) {
    const state = reactive<LocalState>({
      showMenu: false,
      selectedItem: props.selectedItem,
    });

    const button = ref<HTMLButtonElement>();
    const popup = ref<HTMLElement>();

    const { width } = useElementBounding(button);

    const popupStyle = computed(() => {
      const style = {} as CSSProperties;

      const key = props.fitWidth === "fit" ? "width" : "min-width";
      style[key] = `${width.value}px`;

      return style;
    });

    const isSelected = computed(() => {
      return (
        state.selectedItem !== null &&
        state.selectedItem !== undefined &&
        state.selectedItem !== ""
      );
    });

    watch(
      () => props.selectedItem,
      (cur) => {
        state.selectedItem = cur;
      }
    );

    const toggle = () => {
      state.showMenu = !state.showMenu;
    };

    const close = () => {
      state.showMenu = false;
    };

    onClickOutside(popup, close);

    return {
      state,
      isSelected,
      toggle,
      close,
      button,
      popup,
      popupStyle,
    };
  },
});
</script>
