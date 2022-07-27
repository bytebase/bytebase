<template>
  <Transition name="slide">
    <div
      v-if="show"
      class="absolute top-0 right-0 h-screen bg-white shadow-xl outline-none"
      :style="`width: ${width}px`"
      tabindex="0"
      @keyup.esc="handleClose"
    >
      <header
        class="flex flex-row justify-between items-center w-full h-16 p-4 border-b"
      >
        <h2 class="font-semibold">{{ title }}</h2>

        <i
          class="w-6 h-6 hover:bg-slate-200 hover:cursor-pointer rounded-md"
          @click="handleClose"
          ><heroicons-outline:x class="h-6 w-6"
        /></i>
      </header>
      <section class="p-4">
        <slot name="body"></slot>
      </section>
    </div>
  </Transition>
</template>

<script lang="ts" setup>
import { defineProps, defineEmits } from "vue";

const props = defineProps({
  show: {
    type: Boolean,
    default: false,
  },
  width: {
    type: Number,
    default: 320,
  },
  title: {
    type: String,
    default: "",
  },
  onClose: {
    type: Function,
    required: false,
    default: () => {
      /* do nth by default */
    },
  },
});

const emit = defineEmits(["close-drawer"]);

const handleClose = () => {
  const { onClose } = props;
  emit("close-drawer");
  onClose();
};
</script>

<style scope>
.slide-enter-active,
.slide-leave-active {
  transition: all 0.3s ease;
}

.slide-enter-from,
.slide-leave-to {
  transform: translateX(100%);
}
</style>
