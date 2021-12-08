<template>
  <div
    v-click-inside-outside="
      (_, inside) => {
        if (!inside) {
          close();
        }
      }
    "
    class="relative flex flex-shrink-0"
  >
    <button
      type="button"
      aria-haspopup="listbox"
      aria-expanded="true"
      aria-labelledby="listbox-label"
      class="btn-select relative w-full pl-3 pr-10 py-1.5"
      :disabled="disabled"
      @click="toggle"
    >
      <template v-if="state.selectedItem">
        <slot name="menuItem" :item="state.selectedItem" />
      </template>
      <template v-else>
        <slot name="placeholder" :placeholder="placeholder">{{
          placeholder
        }}</slot>
      </template>
      <span
        class="
          ml-3
          absolute
          inset-y-0
          right-0
          flex
          items-center
          pr-2
          pointer-events-none
        "
      >
        <!-- Heroicon name: solid/selector -->
        <heroicons-solid:selector class="h-5 w-5 text-control-light" />
      </span>
    </button>

    <!--
      Select popover, show/hide based on select state.

      Entering: ""
        From: ""
        To: ""
      Leaving: "transition ease-in duration-100"
        From: "opacity-100"
        To: "opacity-0"
    -->
    <transition
      enter-active-class=""
      enter-class=""
      enter-to-class=""
      leave-active-class="transition ease-in duration-100"
      leave-class="opacity-100"
      leave-to-class="opacity-0"
    >
      <div
        v-show="state.showMenu"
        class="z-50 absolute w-full rounded-md bg-white shadow-lg"
      >
        <ul
          tabindex="-1"
          role="listbox"
          aria-labelledby="listbox-label"
          aria-activedescendant="listbox-item-3"
          class="
            max-h-56
            rounded-md
            py-1
            ring-1 ring-black ring-opacity-5
            overflow-auto
            focus:outline-none
            sm:text-sm
          "
        >
          <!--
          Select option, manage highlight styles based on mouseenter/mouseleave and keyboard navigation.

          Highlighted: "text-white bg-indigo-600", Not Highlighted: "text-gray-900"
        -->
          <li
            v-for="(item, index) in itemList"
            :key="index"
            role="option"
            class="
              text-main
              hover:text-main-text hover:bg-main-hover
              cursor-default
              select-none
              relative
              py-2
              pl-3
              pr-9
            "
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
  </div>
</template>

<script lang="ts">
import { reactive, PropType, watch } from "vue";
import clickInsideOutside from "./directives/click-inside-outside";

interface LocalState {
  showMenu: boolean;
  selectedItem: any;
}

export default {
  name: "BBSelect",
  directives: {
    "click-inside-outside": clickInsideOutside,
  },
  components: {},
  props: {
    selectedItem: {},
    itemList: {
      required: true,
      type: Object as PropType<any[]>,
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

    return {
      state,
      toggle,
      close,
    };
  },
};
</script>
