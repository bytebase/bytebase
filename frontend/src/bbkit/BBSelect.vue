<template>
  <VBinder>
    <VTarget>
      <button
        ref="button"
        type="button"
        aria-haspopup="listbox"
        aria-expanded="true"
        aria-labelledby="listbox-label"
        class="btn-select relative w-full pl-3 pr-10 py-1.5"
        :disabled="disabled"
        v-bind="$attrs"
        @click="toggle"
      >
        <template v-if="state.selectedItem">
          <slot name="menuItem" :item="state.selectedItem" />
        </template>
        <template v-else>
          <slot name="placeholder" :placeholder="placeholder">
            {{ placeholder }}
          </slot>
        </template>
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
          class="z-50 rounded-md bg-white shadow-lg"
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
              v-for="(item, index) in itemList"
              :key="index"
              role="option"
              class="text-main hover:text-main-text hover:bg-main-hover cursor-default select-none relative py-2 pl-3 pr-9"
              @click="
                if (item !== state.selectedItem) {
                  $emit('select-item', item, () => {
                    state.selectedItem = item;
                  });
                }
                close();
              "
            >
              <slot name="menuItem" :item="item" />
              <span
                v-if="item === state.selectedItem"
                class="absolute inset-y-0 right-0 flex items-center pr-4"
              >
                <!-- Heroicon name: solid/check -->
                <heroicons-solid:check class="h-5 w-5" />
              </span>
            </li>
          </ul>
        </div>
      </transition>
    </VFollower>
  </VBinder>
</template>

<script lang="ts">
import { reactive, PropType, watch, defineComponent, ref } from "vue";
import { VBinder, VTarget, VFollower } from "vueuc";
import { onClickOutside, useElementBounding } from "@vueuse/core";

interface LocalState {
  showMenu: boolean;
  selectedItem: any;
}

type ItemType = number | string | any;

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
  },
  emits: ["select-item"],
  setup(props) {
    const state = reactive<LocalState>({
      showMenu: false,
      selectedItem: props.selectedItem,
    });

    const button = ref<HTMLButtonElement | null>(null);
    const popup = ref<HTMLElement | null>(null);

    const { width } = useElementBounding(button);

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
      toggle,
      close,
      button,
      popup,
      width,
    };
  },
});
</script>
