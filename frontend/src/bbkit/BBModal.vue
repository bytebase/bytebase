<!--
  rgba(209, 213, 219, 0.8) is bg-gray-300
-->
<template>
  <div
    class="fixed inset-0 w-full h-screen flex items-center justify-center z-40"
    style="background-color: rgba(209, 213, 219, 0.8)"
  >
    <div
      class="
        relative
        max-h-screen
        w-full
        max-w-max
        bg-white
        shadow-lg
        rounded-lg
        p-8
        flex
        space-y-6
        divide-y divide-block-border
      "
    >
      <div>
        <div class="absolute left-0 top-0 my-4 mx-8 text-xl text-main">
          {{ title }}
          <div
            v-if="subtitle"
            class="pr-1 bg-white text-sm text-control whitespace-nowrap"
          >
            {{ subtitle }}
          </div>
        </div>
        <button
          v-if="showClose"
          class="absolute right-0 top-0 my-4 mx-4 text-control-light"
          aria-label="close"
          @click.prevent="close"
        >
          <span class="sr-only">Close</span>
          <!-- Heroicons name: x -->
          <heroicons-solid:x class="w-6 h-6" />
        </button>
      </div>
      <div class="pt-4 px-0.5 max-h-screen overflow-auto w-full">
        <slot />
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import { onMounted, onUnmounted } from "vue";

export default {
  name: "BBModal",
  props: {
    title: {
      required: true,
      type: String,
    },
    subtitle: {
      default: "",
      type: String,
    },
    showClose: {
      type: Boolean,
      default: true,
    },
  },
  emits: ["close"],
  setup(props, { emit }) {
    const close = () => {
      emit("close");
    };

    const escHandler = (e: KeyboardEvent) => {
      if (e.code == "Escape") {
        close();
      }
    };

    onMounted(() => {
      document.addEventListener("keydown", escHandler);
    });

    onUnmounted(() => {
      document.removeEventListener("keydown", escHandler);
    });

    return {
      close,
    };
  },
};
</script>
