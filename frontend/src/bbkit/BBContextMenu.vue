<!-- Usage
<button
    @click.prevent="$refs.menu.toggle($event, 'left click')"
    @contextmenu.capture.prevent="$refs.menu.toggle($event, 'right click')"
>
<BBContextMenu ref="menu" v-slot="slotProps">
    <div>
        Click using {{slotProps.context}}
    </div>
</BBContextMenu>
-->
<template>
  <div class="z-50 absolute right-0 rounded-md shadow-lg">
    <!--
      Profile dropdown panel, show/hide based on dropdown state.

      Entering: "transition ease-out duration-100"
        From: "transform opacity-0 scale-95"
        To: "transform opacity-100 scale-100"
      Leaving: "transition ease-in duration-75"
        From: "transform opacity-100 scale-100"
        To: "transform opacity-0 scale-95"
    -->
    <transition
      enter-active-class="transition ease-out duration-100"
      enter-class="transform opacity-0 scale-95"
      enter-to-class="transform opacity-100 scale-100"
      leave-active-class="transition ease-in duration-75"
      leave-class="transform opacity-100 scale-100"
      leave-to-class="transform opacity-0 scale-95"
    >
      <div
        v-show="state.isOpen"
        v-click-inside-outside="close"
        @contextmenu.capture.prevent
      >
        <div
          class="bg-white py-1 rounded-md shadow-xs"
          role="menu"
          aria-orientation="vertical"
          aria-labelledby="user-menu"
        >
          <slot :context="state.context" />
        </div>
      </div>
    </transition>
  </div>
</template>

<script lang="ts" setup>
import { reactive } from "vue";
import vClickInsideOutside from "./directives/click-inside-outside";

type Context = string | number | boolean | any;

const state = reactive({
  isOpen: false,
  context: {} as Context,
});

const open = (evt: any, context: Context) => {
  // This is a hack, upon initial click to bring up the context menu, it will
  // trigger both open and close. The latter is triggered because the click
  // is outside of the menu which hasn't brought up yet. So here, we use
  // setTimeout to schedule open after close.
  requestAnimationFrame(() => {
    state.isOpen = true;
    state.context = context;
  });
};
const close = () => {
  // Close the menu regardless of whether the click is inside the menu.
  state.isOpen = false;
  state.context = null;
};
const toggle = (evt: any, context: Context) => {
  if (!state.isOpen) {
    open(evt, context);
  } else {
    close();
  }
};

defineExpose({ open, close, toggle });
</script>
