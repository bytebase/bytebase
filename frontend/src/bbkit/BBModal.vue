<!--
  rgba(209, 213, 219, 0.8) is bg-gray-300
-->
<template>
  <div
    class="fixed inset-0 w-full h-screen flex items-center justify-center z-50"
    @click.self="closeIfShown"
    style="background-color: rgba(209, 213, 219, 0.8)"
  >
    <div
      class="relative max-h-screen w-full max-w-max bg-white shadow-lg rounded-lg p-8 flex space-y-6 divide-y divide-block-border"
    >
      <div>
        <div class="absolute left-0 top-0 my-4 mx-8 text-xl text-main">
          {{ $props.title }}
        </div>
        <button
          v-if="showClose"
          class="absolute right-0 top-0 my-4 mx-4 text-control-light"
          aria-label="close"
          @click.prevent="close"
        >
          <span class="sr-only">Close</span>
          <!-- Heroicon name: x -->
          <svg
            class="h-6 w-6"
            xmlns="http://www.w3.org/2000/svg"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
            aria-hidden="true"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M6 18L18 6M6 6l12 12"
            />
          </svg>
        </button>
      </div>
      <div class="pt-2 max-h-screen w-full">
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
    showClose: {
      type: Boolean,
      default: true,
    },
    backgroundClose: {
      type: Boolean,
      default: true,
    },
  },
  setup(props, { emit }) {
    const close = () => {
      emit("close");
    };

    const closeIfShown = () => {
      if (props.showClose && props.backgroundClose) {
        close();
      }
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
      closeIfShown,
    };
  },
};
</script>
